package moon

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

/** 月球飞行器列表（按 sort_order 排序） */
func (r *Repository) ListSpacecraft(ctx context.Context) ([]Spacecraft, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name_zh, name_en, type, operator_name, description,
		       launch_date, launch_site, launch_vehicle, source_name,
		       display_inclination, display_eccentricity, display_period,
		       kind, orbit_a, orbit_e, inclination_deg, raan_deg, arg_periapsis_deg, period_seconds,
		       stationary_offset_x, stationary_offset_y, stationary_offset_z,
		       sort_order
		FROM moon_spacecraft
		ORDER BY sort_order, id`)
	if err != nil {
		return nil, fmt.Errorf("query moon spacecraft: %w", err)
	}
	defer rows.Close()

	items := []Spacecraft{}
	for rows.Next() {
		var item Spacecraft
		var ox, oy, oz float64
		if err := rows.Scan(
			&item.ID, &item.NameZH, &item.NameEN, &item.Type, &item.OperatorName, &item.Description,
			&item.LaunchDate, &item.LaunchSite, &item.LaunchVehicle, &item.SourceName,
			&item.DisplayInclination, &item.DisplayEccentricity, &item.DisplayPeriod,
			&item.Kind, &item.OrbitA, &item.OrbitE, &item.InclinationDeg, &item.RaanDeg, &item.ArgPeriapsisDeg, &item.PeriodSeconds,
			&ox, &oy, &oz,
			&item.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("scan moon spacecraft: %w", err)
		}
		item.StationaryOffset = [3]float64{ox, oy, oz}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate moon spacecraft: %w", err)
	}
	return items, nil
}
