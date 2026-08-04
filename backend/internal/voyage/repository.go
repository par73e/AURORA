package voyage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListProbes 深空探测器目录 + 每个目标的完整采样窗口（按时间升序）。
// 同步任务每次整窗替换采样，因此当前窗口即最新一次同步的结果。
func (r *Repository) ListProbes(ctx context.Context) ([]Probe, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name_zh, name_en, operator_name, launch_date, launch_site, launch_vehicle,
		       mission_type, target, description, precision_grade, color, orbit_kind, sort_order
		FROM deep_space_probes
		ORDER BY sort_order, id`)
	if err != nil {
		return nil, fmt.Errorf("query deep space probes: %w", err)
	}
	defer rows.Close()

	items := []Probe{}
	for rows.Next() {
		var item Probe
		if err := rows.Scan(
			&item.ID, &item.NameZH, &item.NameEN, &item.OperatorName, &item.LaunchDate,
			&item.LaunchSite, &item.LaunchVehicle, &item.MissionType, &item.Target,
			&item.Description, &item.PrecisionGrade, &item.Color, &item.OrbitKind, &item.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("scan deep space probe: %w", err)
		}
		item.Positions = []PositionPoint{}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deep space probes: %w", err)
	}

	for i := range items {
		if err := r.loadSamples(ctx, &items[i]); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (r *Repository) loadSamples(ctx context.Context, probe *Probe) error {
	rows, err := r.pool.Query(ctx, `
		SELECT epoch, x_km, y_km, z_km, synced_at
		FROM probe_position_samples
		WHERE probe_id = $1
		ORDER BY epoch ASC`, probe.ID)
	if err != nil {
		return fmt.Errorf("query probe samples: %w", err)
	}
	defer rows.Close()

	probe.Positions = []PositionPoint{}
	var latestSynced time.Time
	for rows.Next() {
		var epoch time.Time
		var p PositionPoint
		var syncedAt time.Time
		if err := rows.Scan(&epoch, &p.X, &p.Y, &p.Z, &syncedAt); err != nil {
			return fmt.Errorf("scan probe sample: %w", err)
		}
		p.Epoch = epoch.UTC().Format(time.RFC3339)
		probe.Positions = append(probe.Positions, p)
		if syncedAt.After(latestSynced) {
			latestSynced = syncedAt
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate probe samples: %w", err)
	}
	if len(probe.Positions) > 0 {
		probe.SyncedAt = latestSynced.UTC().Format(time.RFC3339)
	}
	return nil
}

// ListCatalog 同步任务用目录（仅 id + NAIF ID）
func (r *Repository) ListCatalog(ctx context.Context) ([]CatalogItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, naif_id FROM deep_space_probes ORDER BY sort_order, id`)
	if err != nil {
		return nil, fmt.Errorf("query probe catalog: %w", err)
	}
	defer rows.Close()

	items := []CatalogItem{}
	for rows.Next() {
		var item CatalogItem
		if err := rows.Scan(&item.ID, &item.NaifID); err != nil {
			return nil, fmt.Errorf("scan probe catalog: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate probe catalog: %w", err)
	}
	return items, nil
}

// ReplaceProbeSamples 整窗替换一个探测器的位置采样（删除旧窗口 + 批量插入新窗口）。
// 每次日同步产生约 181 行，表保持小而精确。
func (r *Repository) ReplaceProbeSamples(ctx context.Context, probeID string, samples []PositionSample) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replace probe samples: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM probe_position_samples WHERE probe_id = $1`, probeID); err != nil {
		return fmt.Errorf("delete probe samples: %w", err)
	}
	for _, s := range samples {
		if _, err := tx.Exec(ctx, `
			INSERT INTO probe_position_samples (probe_id, epoch, x_km, y_km, z_km)
			VALUES ($1, $2, $3, $4, $5)`,
			probeID, s.Epoch, s.X, s.Y, s.Z); err != nil {
			return fmt.Errorf("insert probe sample: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit replace probe samples: %w", err)
	}
	return nil
}
