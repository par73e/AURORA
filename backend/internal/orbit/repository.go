package orbit

import (
	"context"
	"encoding/json"
	"fmt"
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
	items := make([]Spacecraft, 0, 3)
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
		SELECT external_id, name, name_zh, status_name, status_name_zh, status_abbrev,
		       net, window_start, window_end,
		       COALESCE(pad_name,''), pad_name_zh, COALESCE(location_name,''), location_name_zh,
		       latitude, longitude,
		       COALESCE(mission_name,''), mission_name_zh,
		       COALESCE(mission_type,''), mission_type_zh,
		       COALESCE(mission_description,''), mission_description_zh,
		       COALESCE(provider_name,''), source_url, synced_at, has_original
		FROM launch_events
		WHERE net >= now() AND net <= now() + interval '30 days'
		ORDER BY net
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
			mission_description,mission_description_zh,provider_name,source_url,raw_payload,has_original)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
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
		provider_name=EXCLUDED.provider_name,source_url=EXCLUDED.source_url,raw_payload=EXCLUDED.raw_payload,
		has_original=EXCLUDED.has_original,synced_at=now()`,
		event.ExternalID, event.Name, event.NameZH,
		event.StatusName, event.StatusNameZH, event.StatusAbbrev,
		event.Net, event.WindowStart, event.WindowEnd,
		event.PadName, event.PadNameZH, event.LocationName, event.LocationNameZH,
		event.Latitude, event.Longitude,
		event.MissionName, event.MissionNameZH, event.MissionType, event.MissionTypeZH,
		event.MissionDescription, event.MissionDescriptionZH,
		event.ProviderName, event.SourceURL, raw, event.HasOriginal)
	return err
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
