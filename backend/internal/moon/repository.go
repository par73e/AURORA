package moon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

/** 最近一次数据同步时间（jpl_horizons——月球飞行器/快照数据源；无记录返回 nil） */
func (r *Repository) LastSyncTime(ctx context.Context) (*time.Time, error) {
	var t time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT finished_at FROM sync_runs
		WHERE source_code = 'jpl_horizons' AND finished_at IS NOT NULL
		ORDER BY started_at DESC LIMIT 1`).Scan(&t)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("last sync time: %w", err)
	}
	return &t, nil
}

/** 月球飞行器列表（按 sort_order 排序；联表取每个飞行器最新轨道快照） */
func (r *Repository) ListSpacecraft(ctx context.Context) ([]Spacecraft, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ms.id, ms.name_zh, ms.name_en, ms.type, ms.operator_name, ms.description,
		       ms.launch_date, ms.launch_site, ms.launch_vehicle, ms.source_name,
		       ms.display_inclination, ms.display_eccentricity, ms.display_period,
		       ms.kind, ms.orbit_a, ms.orbit_e, ms.inclination_deg, ms.raan_deg, ms.arg_periapsis_deg, ms.period_seconds,
		       ms.stationary_offset_x, ms.stationary_offset_y, ms.stationary_offset_z,
		       ms.sort_order,
		       sn.epoch, sn.a_km, sn.eccentricity, sn.inclination_deg, sn.raan_deg,
		       sn.arg_periapsis_deg, sn.mean_anomaly_deg, sn.period_seconds
		FROM moon_spacecraft ms
		LEFT JOIN LATERAL (
			SELECT * FROM moon_orbit_snapshots sn
			WHERE sn.spacecraft_id = ms.id
			ORDER BY sn.epoch DESC LIMIT 1
		) sn ON true
		ORDER BY ms.sort_order, ms.id`)
	if err != nil {
		return nil, fmt.Errorf("query moon spacecraft: %w", err)
	}
	defer rows.Close()

	items := []Spacecraft{}
	for rows.Next() {
		var item Spacecraft
		var ox, oy, oz float64
		var snapshotEpoch *time.Time
		var aKm, ecc, inc, raan, argp, ma, period *float64
		if err := rows.Scan(
			&item.ID, &item.NameZH, &item.NameEN, &item.Type, &item.OperatorName, &item.Description,
			&item.LaunchDate, &item.LaunchSite, &item.LaunchVehicle, &item.SourceName,
			&item.DisplayInclination, &item.DisplayEccentricity, &item.DisplayPeriod,
			&item.Kind, &item.OrbitA, &item.OrbitE, &item.InclinationDeg, &item.RaanDeg, &item.ArgPeriapsisDeg, &item.PeriodSeconds,
			&ox, &oy, &oz,
			&item.SortOrder,
			&snapshotEpoch, &aKm, &ecc, &inc, &raan, &argp, &ma, &period,
		); err != nil {
			return nil, fmt.Errorf("scan moon spacecraft: %w", err)
		}
		item.StationaryOffset = [3]float64{ox, oy, oz}
		if snapshotEpoch != nil && aKm != nil {
			item.Snapshot = &OrbitSnapshot{
				Epoch:           snapshotEpoch.UTC().Format(time.RFC3339),
				AKm:             *aKm,
				Eccentricity:    *ecc,
				InclinationDeg:  *inc,
				RaanDeg:         *raan,
				ArgPeriapsisDeg: *argp,
				MeanAnomalyDeg:  *ma,
				PeriodSeconds:   *period,
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate moon spacecraft: %w", err)
	}
	return items, nil
}

/** 月球着陆点列表（真实经纬度，按任务时间排序） */
func (r *Repository) ListLandingSites(ctx context.Context) ([]LandingSite, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name_zh, name_en, program, operator_name,
		       landing_date, latitude, longitude, region, description, sort_order,
		       site_name, official_name, mission_name, hardware, side, category, icon, track
		FROM moon_landing_sites
		ORDER BY sort_order`)
	if err != nil {
		return nil, fmt.Errorf("query moon landing sites: %w", err)
	}
	defer rows.Close()

	items := []LandingSite{}
	for rows.Next() {
		var item LandingSite
		var hardwareRaw []byte
		var trackRaw []byte
		if err := rows.Scan(
			&item.ID, &item.NameZH, &item.NameEN, &item.Program, &item.OperatorName,
			&item.LandingDate, &item.Latitude, &item.Longitude, &item.Region, &item.Description,
			&item.SortOrder,
			&item.SiteName, &item.OfficialName, &item.MissionName, &hardwareRaw, &item.Side, &item.Category, &item.Icon, &trackRaw,
		); err != nil {
			return nil, fmt.Errorf("scan moon landing site: %w", err)
		}
		if len(hardwareRaw) > 0 {
			_ = json.Unmarshal(hardwareRaw, &item.Hardware)
		}
		if len(trackRaw) > 0 {
			_ = json.Unmarshal(trackRaw, &item.Track)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate moon landing sites: %w", err)
	}
	return items, nil
}

/** 保存月球飞行器轨道快照（镜像 orbit.SaveOrbitSnapshot） */
func (r *Repository) SaveMoonSnapshot(ctx context.Context, spacecraftID string, epoch string, el Elements, raw string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO moon_orbit_snapshots
			(spacecraft_id, epoch, a_km, eccentricity, inclination_deg, raan_deg,
			 arg_periapsis_deg, mean_anomaly_deg, period_seconds, raw)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		spacecraftID, epoch, el.A, el.E, el.InclinationDeg, el.RaanDeg,
		el.ArgPeriapsisDeg, el.MeanAnomalyDeg, el.PeriodSeconds, raw)
	if err != nil {
		return fmt.Errorf("save moon orbit snapshot: %w", err)
	}
	return nil
}
