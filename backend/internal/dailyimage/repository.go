package dailyimage

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

func (r *Repository) GetWallCache(ctx context.Context, date time.Time) (ImageWall, bool, error) {
	if r == nil {
		return ImageWall{}, false, nil
	}
	return r.loadWallCache(ctx, `
		SELECT recent, collection, generated_at
		FROM daily_image_wall_cache
		WHERE cache_date = $1`, wallCacheDate(date))
}

func (r *Repository) LatestWallCache(ctx context.Context) (ImageWall, bool, error) {
	if r == nil {
		return ImageWall{}, false, nil
	}
	return r.loadWallCache(ctx, `
		SELECT recent, collection, generated_at
		FROM daily_image_wall_cache
		ORDER BY cache_date DESC
		LIMIT 1`)
}

func (r *Repository) loadWallCache(ctx context.Context, query string, args ...any) (ImageWall, bool, error) {
	var recentRaw, collectionRaw []byte
	var generatedAt time.Time
	err := r.pool.QueryRow(ctx, query, args...).Scan(&recentRaw, &collectionRaw, &generatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ImageWall{}, false, nil
	}
	if err != nil {
		return ImageWall{}, false, fmt.Errorf("query daily image wall cache: %w", err)
	}
	var wall ImageWall
	if err := json.Unmarshal(recentRaw, &wall.Recent); err != nil {
		return ImageWall{}, false, fmt.Errorf("decode recent daily image wall cache: %w", err)
	}
	if err := json.Unmarshal(collectionRaw, &wall.Collection); err != nil {
		return ImageWall{}, false, fmt.Errorf("decode collection daily image wall cache: %w", err)
	}
	wall.GeneratedAt = generatedAt.UTC().Format(time.RFC3339)
	return wall, true, nil
}

func (r *Repository) SaveWallCache(ctx context.Context, date time.Time, wall ImageWall) error {
	if r == nil {
		return nil
	}
	recentRaw, err := json.Marshal(wall.Recent)
	if err != nil {
		return fmt.Errorf("encode recent daily image wall cache: %w", err)
	}
	collectionRaw, err := json.Marshal(wall.Collection)
	if err != nil {
		return fmt.Errorf("encode collection daily image wall cache: %w", err)
	}
	generatedAt, err := time.Parse(time.RFC3339, wall.GeneratedAt)
	if err != nil {
		generatedAt = time.Now().UTC()
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO daily_image_wall_cache(cache_date, recent, collection, generated_at, refreshed_at)
		VALUES($1, $2, $3, $4, now())
		ON CONFLICT(cache_date) DO UPDATE SET
			recent = EXCLUDED.recent,
			collection = EXCLUDED.collection,
			generated_at = EXCLUDED.generated_at,
			refreshed_at = now()`,
		wallCacheDate(date), recentRaw, collectionRaw, generatedAt.UTC())
	if err != nil {
		return fmt.Errorf("upsert daily image wall cache: %w", err)
	}
	return nil
}

var _ WallCacheStore = (*Repository)(nil)
