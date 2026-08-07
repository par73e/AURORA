package orbit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Overview(ctx context.Context) (Overview, error) {
	spacecraft, err := r.listSpacecraft(ctx)
	if err != nil {
		return Overview{}, err
	}
	sites, err := r.listLaunchSites(ctx)
	if err != nil {
		return Overview{}, err
	}
	events, err := r.listLaunchEvents(ctx)
	if err != nil {
		return Overview{}, err
	}
	freshness, err := r.listFreshness(ctx)
	if err != nil {
		return Overview{}, err
	}
	return Overview{GeneratedAt: time.Now().UTC(), Spacecraft: spacecraft, LaunchSites: sites, Events: events, Freshness: freshness}, nil
}

func (r *Repository) listSpacecraft(ctx context.Context) ([]Spacecraft, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.name_zh, s.name_en, s.norad_catalog_id, s.category,
		       s.operator_name, s.description,
		       COALESCE(s.launch_date,''), COALESCE(s.launch_site,''), COALESCE(s.launch_vehicle,''),
		       ds.name, ds.base_url,
		       o.epoch, o.synced_at, o.raw_omm
		FROM spacecraft s
		JOIN data_sources ds ON ds.code = s.source_code
		LEFT JOIN LATERAL (
			SELECT epoch, synced_at, raw_omm
			FROM orbit_snapshots
			WHERE spacecraft_id = s.id
			ORDER BY epoch DESC
			LIMIT 1
		) o ON true
		ORDER BY s.id`)
	if err != nil {
		return nil, fmt.Errorf("list spacecraft: %w", err)
	}
	defer rows.Close()
	return scanSpacecraftRows(rows)
}

const spacecraftColumns = `
	SELECT s.id, s.name_zh, s.name_en, s.norad_catalog_id, s.category,
	       s.operator_name, s.description,
	       COALESCE(s.launch_date,''), COALESCE(s.launch_site,''), COALESCE(s.launch_vehicle,''),
	       ds.name, ds.base_url,
	       o.epoch, o.synced_at, o.raw_omm
	FROM spacecraft s
	JOIN data_sources ds ON ds.code = s.source_code
	LEFT JOIN LATERAL (
		SELECT epoch, synced_at, raw_omm
		FROM orbit_snapshots
		WHERE spacecraft_id = s.id
		ORDER BY epoch DESC
		LIMIT 1
	) o ON true`

// SearchSpacecraft 为目录提供服务端筛选与分页。白名单排序字段避免 SQL 标识符拼接，
// 正则由 PostgreSQL 执行，前端只传去掉 /.../ 包装后的表达式。
func (r *Repository) SearchSpacecraft(ctx context.Context, query SpacecraftQuery) (SpacecraftPage, error) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	sortSQL := "s.name_zh COLLATE \"C\", s.id"
	switch query.Sort {
	case "norad":
		sortSQL = "s.norad_catalog_id NULLS LAST, s.id"
	case "operator":
		sortSQL = "s.operator_name COLLATE \"C\", s.name_zh COLLATE \"C\", s.id"
	}

	searchable := "concat_ws(' ', s.name_zh, s.name_en, s.norad_catalog_id::text, s.operator_name, s.category)"
	matcher := searchable + " ILIKE '%' || $1 || '%'"
	if query.Regex {
		matcher = searchable + " ~* $1"
	}
	operatorPrimary := "regexp_replace(split_part(s.operator_name, ' / ', 1), '[（(].*$', '')"
	where := " WHERE ($1 = '' OR " + matcher + ") AND ($2 = '' OR " + operatorPrimary + " = $2)"

	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM spacecraft s"+where, query.Query, query.Operator).Scan(&total); err != nil {
		return SpacecraftPage{}, fmt.Errorf("count spacecraft catalog: %w", err)
	}
	rows, err := r.pool.Query(ctx, spacecraftColumns+where+" ORDER BY "+sortSQL+" LIMIT $3 OFFSET $4", query.Query, query.Operator, query.PageSize, (query.Page-1)*query.PageSize)
	if err != nil {
		return SpacecraftPage{}, fmt.Errorf("search spacecraft catalog: %w", err)
	}
	defer rows.Close()
	items, err := scanSpacecraftRows(rows)
	if err != nil {
		return SpacecraftPage{}, err
	}
	return SpacecraftPage{Items: items, Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}

func scanSpacecraftRows(rows pgx.Rows) ([]Spacecraft, error) {
	items := make([]Spacecraft, 0, 20)
	for rows.Next() {
		var item Spacecraft
		var epoch, syncedAt *time.Time
		var raw []byte
		if err := rows.Scan(&item.ID, &item.NameZH, &item.NameEN, &item.NORADCatalogID, &item.Category, &item.OperatorName, &item.Description, &item.LaunchDate, &item.LaunchSite, &item.LaunchVehicle, &item.SourceName, &item.SourceURL, &epoch, &syncedAt, &raw); err != nil {
			return nil, fmt.Errorf("scan spacecraft: %w", err)
		}
		item.OrbitEpoch = epoch
		item.OrbitSyncedAt = syncedAt
		if len(raw) > 0 {
			item.OMM = json.RawMessage(raw)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) listLaunchSites(ctx context.Context) ([]LaunchSite, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name_zh, name_en, country_code, country_name_zh, latitude, longitude, description, source_url, tier FROM launch_sites ORDER BY tier, sort_order, id`)
	if err != nil {
		return nil, fmt.Errorf("list launch sites: %w", err)
	}
	defer rows.Close()
	items := make([]LaunchSite, 0, 16)
	for rows.Next() {
		var item LaunchSite
		if err := rows.Scan(&item.ID, &item.NameZH, &item.NameEN, &item.CountryCode, &item.CountryNameZH, &item.Latitude, &item.Longitude, &item.Description, &item.SourceURL, &item.Tier); err != nil {
			return nil, fmt.Errorf("scan launch site: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) listLaunchEvents(ctx context.Context) ([]LaunchEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT e.external_id, e.name, e.name_zh, e.status_name, e.status_name_zh, e.status_abbrev,
		       e.net, e.window_start, e.window_end,
		       COALESCE(e.pad_name,''), e.pad_name_zh, COALESCE(e.location_name,''), e.location_name_zh,
		       COALESCE(ls.latitude, e.latitude), COALESCE(ls.longitude, e.longitude),
		       COALESCE(e.mission_name,''), e.mission_name_zh,
		       COALESCE(e.mission_type,''), e.mission_type_zh,
		       COALESCE(e.mission_description,''), e.mission_description_zh,
		       COALESCE(e.provider_name,''), e.source_url, e.synced_at, e.has_original,
		       COALESCE(ls.id, '') AS launch_site_id
		FROM launch_events e
		LEFT JOIN launch_sites ls ON ls.id = e.launch_site_id
		WHERE e.net >= now() AND e.net <= now() + interval '30 days'
		ORDER BY e.net
		LIMIT 20`)
	if err != nil {
		return nil, fmt.Errorf("list launch events: %w", err)
	}
	defer rows.Close()
	items := make([]LaunchEvent, 0, 20)
	for rows.Next() {
		var item LaunchEvent
		if err := rows.Scan(
			&item.ExternalID, &item.Name, &item.NameZH,
			&item.StatusName, &item.StatusNameZH, &item.StatusAbbrev,
			&item.Net, &item.WindowStart, &item.WindowEnd,
			&item.PadName, &item.PadNameZH, &item.LocationName, &item.LocationNameZH,
			&item.Latitude, &item.Longitude,
			&item.MissionName, &item.MissionNameZH,
			&item.MissionType, &item.MissionTypeZH,
			&item.MissionDescription, &item.MissionDescriptionZH,
			&item.ProviderName, &item.SourceURL, &item.SyncedAt, &item.HasOriginal,
			&item.LaunchSiteID,
		); err != nil {
			return nil, fmt.Errorf("scan launch event: %w", err)
		}
		if item.NameZH == "" || item.StatusNameZH == "" {
			LocalizeLaunchEvent(&item)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) listFreshness(ctx context.Context) ([]DataFreshness, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ds.code, ds.name, sr.finished_at, sr.success, COALESCE(sr.error_message,'')
		FROM data_sources ds
		LEFT JOIN LATERAL (
			SELECT finished_at, success, error_message
			FROM sync_runs
			WHERE source_code = ds.code
			ORDER BY started_at DESC
			LIMIT 1
		) sr ON true
		ORDER BY ds.code`)
	if err != nil {
		return nil, fmt.Errorf("list data freshness: %w", err)
	}
	defer rows.Close()
	items := make([]DataFreshness, 0, 2)
	for rows.Next() {
		var item DataFreshness
		if err := rows.Scan(&item.SourceCode, &item.SourceName, &item.LastFinishedAt, &item.Success, &item.ErrorMessage); err != nil {
			return nil, fmt.Errorf("scan freshness: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SpacecraftCatalog(ctx context.Context) (map[string]int64, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, norad_catalog_id FROM spacecraft")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := map[string]int64{}
	for rows.Next() {
		var id string
		var catalogID int64
		if err := rows.Scan(&id, &catalogID); err != nil {
			return nil, err
		}
		items[id] = catalogID
	}
	return items, rows.Err()
}

func (r *Repository) SaveOrbitSnapshot(ctx context.Context, spacecraftID string, epoch time.Time, raw json.RawMessage) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO orbit_snapshots(spacecraft_id, epoch, raw_omm) VALUES($1,$2,$3) ON CONFLICT(spacecraft_id, epoch) DO UPDATE SET raw_omm=EXCLUDED.raw_omm, synced_at=now()`, spacecraftID, epoch, raw)
	return err
}

func (r *Repository) SaveLaunchEvent(ctx context.Context, event LaunchEvent, raw json.RawMessage) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO launch_events(
			external_id,name,name_zh,status_name,status_name_zh,status_abbrev,
			net,window_start,window_end,pad_name,pad_name_zh,location_name,location_name_zh,
			latitude,longitude,mission_name,mission_name_zh,mission_type,mission_type_zh,
			mission_description,mission_description_zh,provider_name,launch_site_id,source_url,raw_payload,has_original)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,NULLIF($23,''),$24,$25,$26)
		ON CONFLICT(external_id) DO UPDATE SET
		name=EXCLUDED.name,name_zh=EXCLUDED.name_zh,
		status_name=EXCLUDED.status_name,status_name_zh=EXCLUDED.status_name_zh,status_abbrev=EXCLUDED.status_abbrev,
		net=EXCLUDED.net,window_start=EXCLUDED.window_start,window_end=EXCLUDED.window_end,
		pad_name=EXCLUDED.pad_name,pad_name_zh=EXCLUDED.pad_name_zh,
		location_name=EXCLUDED.location_name,location_name_zh=EXCLUDED.location_name_zh,
		latitude=EXCLUDED.latitude,longitude=EXCLUDED.longitude,
		mission_name=EXCLUDED.mission_name,mission_name_zh=EXCLUDED.mission_name_zh,
		mission_type=EXCLUDED.mission_type,mission_type_zh=EXCLUDED.mission_type_zh,
		mission_description=EXCLUDED.mission_description,mission_description_zh=EXCLUDED.mission_description_zh,
		provider_name=EXCLUDED.provider_name,launch_site_id=NULLIF(EXCLUDED.launch_site_id,''),
		source_url=EXCLUDED.source_url,raw_payload=EXCLUDED.raw_payload,
		has_original=EXCLUDED.has_original,synced_at=now()`,
		event.ExternalID, event.Name, event.NameZH,
		event.StatusName, event.StatusNameZH, event.StatusAbbrev,
		event.Net, event.WindowStart, event.WindowEnd,
		event.PadName, event.PadNameZH, event.LocationName, event.LocationNameZH,
		event.Latitude, event.Longitude,
		event.MissionName, event.MissionNameZH, event.MissionType, event.MissionTypeZH,
		event.MissionDescription, event.MissionDescriptionZH,
		event.ProviderName, event.LaunchSiteID, event.SourceURL, raw, event.HasOriginal)
	return err
}

// normalizeSiteName 规范化地点名称用于匹配：小写、去变音符、去非字母数字字符、折叠空白。
func normalizeSiteName(s string) string {
	var b strings.Builder
	prevSpace := true
	for _, r := range strings.ToLower(s) {
		if r >= 'à' && r <= 'ÿ' {
			r = foldDiacritic(r)
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevSpace = false
		} else if !prevSpace {
			b.WriteByte(' ')
			prevSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

// foldDiacritic 将带变音符的拉丁字母折叠为 ASCII 基础字母（覆盖常见西欧字符）。
func foldDiacritic(r rune) rune {
	switch r {
	case 'à', 'á', 'â', 'ä', 'ã', 'å':
		return 'a'
	case 'ç':
		return 'c'
	case 'è', 'é', 'ê', 'ë':
		return 'e'
	case 'ì', 'í', 'î', 'ï':
		return 'i'
	case 'ñ':
		return 'n'
	case 'ò', 'ó', 'ô', 'ö', 'õ':
		return 'o'
	case 'ù', 'ú', 'û', 'ü':
		return 'u'
	case 'ý', 'ÿ':
		return 'y'
	default:
		return r
	}
}

// launchSiteAliases：Launch Library 常用简称 → 发射场 id（规范化键，词边界前缀匹配，取最长者）。
var launchSiteAliases = map[string]string{
	"vandenberg sfb":     "vandenberg",
	"cape canaveral sfs": "cape-canaveral",
	"cape canaveral":     "cape-canaveral",
	"sriharikota":        "satish-dhawan",
	"mahia":              "rocket-lab",
	"kourou":             "guiana",
	"woomera":            "woomera",
	"alcantara":          "alcantara",
	"naro":               "naro",
	"baikonur":           "baikonur",
}

// matchLaunchSiteAlias 在别名表中做最长词边界匹配，取最长命中。
// contained=false（位置名前缀语义）："cape canaveral sfs, fl usa" 以 "cape canaveral sfs" 开头；
// contained=true（工位名包含语义）："lc 39a kennedy space center" 包含 "kennedy space center"。
func matchLaunchSiteAlias(key string, contained bool) string {
	best, bestLen := "", 0
	for alias, id := range launchSiteAliases {
		hit := key == alias ||
			strings.HasPrefix(key, alias+" ") ||
			(contained && (strings.HasSuffix(key, " "+alias) || strings.Contains(key, " "+alias+" ")))
		if hit && len(alias) > bestLen {
			best, bestLen = id, len(alias)
		}
	}
	return best
}

// siteNamePrefixMatch：位置名 key 是否以站点全名开头（词边界）。如
// "wenchang space launch site people s republic of china" → "wenchang space launch site " 开头。
func siteNamePrefixMatch(key, siteNorm string) bool {
	return key == siteNorm || strings.HasPrefix(key, siteNorm+" ")
}

// siteNameContainedMatch：工位名 key 是否包含站点全名（词边界）。如
// "lc 39a kennedy space center" 包含 "kennedy space center"。
func siteNameContainedMatch(key, siteNorm string) bool {
	return key == siteNorm ||
		strings.HasPrefix(key, siteNorm+" ") ||
		strings.HasSuffix(key, " "+siteNorm) ||
		strings.Contains(key, " "+siteNorm+" ")
}

// MatchLaunchSite 将事件地点解析到 launch_sites 的 id（空串表示无匹配，回退事件自带坐标）。
// 工位名（padName）归属更精确（如 "LC-39A, Kennedy Space Center" → kennedy），先于区域名（locationName）尝试。
func (r *Repository) MatchLaunchSite(ctx context.Context, locationName, padName string) (string, error) {
	// 1) 工位名：包含语义（站点名通常在 pad 名中部/末尾）
	if id, err := r.matchLaunchSite(ctx, normalizeSiteName(padName), true); err != nil || id != "" {
		return id, err
	}
	// 2) 位置名：前缀语义（"Wenchang Space Launch Site, PRC" 以站点全名开头）
	return r.matchLaunchSite(ctx, normalizeSiteName(locationName), false)
}

func (r *Repository) matchLaunchSite(ctx context.Context, key string, contained bool) (string, error) {
	if key == "" {
		return "", nil
	}
	if id := matchLaunchSiteAlias(key, contained); id != "" {
		return id, nil
	}
	siteNorm := `lower(regexp_replace(s.name_en, '[^a-zA-Z0-9]', ' ', 'g'))`
	predicate := `($1 = ` + siteNorm +
		` OR left($1, length(` + siteNorm + `)+1) = ` + siteNorm + ` || ' ')`
	if contained {
		predicate = `($1 = ` + siteNorm +
			` OR $1 LIKE ` + siteNorm + ` || ' %' OR $1 LIKE '% ' || ` + siteNorm +
			` OR $1 LIKE '% ' || ` + siteNorm + ` || ' %')`
	}
	var id string
	err := r.pool.QueryRow(ctx, `
		SELECT s.id FROM launch_sites s
		WHERE length(`+siteNorm+`) >= 8 AND `+predicate+`
		LIMIT 1`, key).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("match launch site: %w", err)
	}
	return id, nil
}

func (r *Repository) StartSync(ctx context.Context, sourceCode string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, "INSERT INTO sync_runs(source_code) VALUES($1) RETURNING id", sourceCode).Scan(&id)
	return id, err
}

func (r *Repository) FinishSync(ctx context.Context, id int64, records int, syncErr error) error {
	if syncErr == nil {
		_, err := r.pool.Exec(ctx, "UPDATE sync_runs SET finished_at=now(), success=true, records_written=$2 WHERE id=$1", id, records)
		return err
	}
	_, err := r.pool.Exec(ctx, "UPDATE sync_runs SET finished_at=now(), success=false, records_written=$2, error_message=$3 WHERE id=$1", id, records, syncErr.Error())
	return err
}

var _ = pgx.ErrNoRows
