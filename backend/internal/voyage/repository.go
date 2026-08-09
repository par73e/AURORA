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
// 目录与采样在同一条 LEFT JOIN 中读出，避免目录扩大时退化为 1 + N 次数据库往返。
func (r *Repository) ListProbes(ctx context.Context) ([]Probe, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT d.id, d.name_zh, d.name_en, d.operator_name, d.launch_date, d.launch_site, d.launch_vehicle,
		       d.mission_type, d.target, d.description, d.precision_grade, d.color, d.orbit_kind, d.sort_order,
		       s.epoch, s.x_km, s.y_km, s.z_km, s.synced_at
		FROM deep_space_probes d
		LEFT JOIN probe_position_samples s ON s.probe_id = d.id
		ORDER BY d.sort_order, d.id, s.epoch ASC`)
	if err != nil {
		return nil, fmt.Errorf("query deep space probes: %w", err)
	}
	defer rows.Close()

	items := []Probe{}
	latestSync := []time.Time{}
	lastID := ""
	for rows.Next() {
		var item Probe
		var epoch *time.Time
		var x, y, z *float64
		var syncedAt *time.Time
		if err := rows.Scan(
			&item.ID, &item.NameZH, &item.NameEN, &item.OperatorName, &item.LaunchDate,
			&item.LaunchSite, &item.LaunchVehicle, &item.MissionType, &item.Target,
			&item.Description, &item.PrecisionGrade, &item.Color, &item.OrbitKind, &item.SortOrder,
			&epoch, &x, &y, &z, &syncedAt,
		); err != nil {
			return nil, fmt.Errorf("scan deep space probe sample: %w", err)
		}
		if item.ID != lastID {
			item.Positions = []PositionPoint{}
			items = append(items, item)
			latestSync = append(latestSync, time.Time{})
			lastID = item.ID
		}
		if epoch == nil {
			continue
		}
		index := len(items) - 1
		items[index].Positions = append(items[index].Positions, PositionPoint{
			Epoch: epoch.UTC().Format(time.RFC3339),
			X:     *x,
			Y:     *y,
			Z:     *z,
		})
		if syncedAt.After(latestSync[index]) {
			latestSync[index] = *syncedAt
			items[index].SyncedAt = syncedAt.UTC().Format(time.RFC3339)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deep space probe samples: %w", err)
	}
	return items, nil
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
