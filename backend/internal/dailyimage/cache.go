package dailyimage

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type WallCacheStore interface {
	GetWallCache(context.Context, time.Time) (ImageWall, bool, error)
	LatestWallCache(context.Context) (ImageWall, bool, error)
	SaveWallCache(context.Context, time.Time, ImageWall) error
}

type CachedWallService struct {
	upstream WallProvider
	store    WallCacheStore
	mu       sync.Mutex
}

func NewCachedWallService(upstream WallProvider, store WallCacheStore) *CachedWallService {
	return &CachedWallService{upstream: upstream, store: store}
}

func (service *CachedWallService) Wall(ctx context.Context, at time.Time) (ImageWall, error) {
	if service == nil || service.upstream == nil {
		return ImageWall{}, errors.New("image wall service is not configured")
	}
	cacheDate := wallCacheDate(at)
	if service.store != nil {
		wall, ok, err := service.store.GetWallCache(ctx, cacheDate)
		if err != nil {
			return ImageWall{}, fmt.Errorf("load daily image wall cache: %w", err)
		}
		if ok && wallCacheable(wall) {
			return wall, nil
		}
	}

	service.mu.Lock()
	defer service.mu.Unlock()
	if service.store != nil {
		wall, ok, err := service.store.GetWallCache(ctx, cacheDate)
		if err != nil {
			return ImageWall{}, fmt.Errorf("load daily image wall cache: %w", err)
		}
		if ok && wallCacheable(wall) {
			return wall, nil
		}
	}

	wall, err := service.upstream.Wall(ctx, at)
	if err != nil {
		if fallback, ok := service.latestCachedWall(ctx); ok {
			return fallback, nil
		}
		return ImageWall{}, err
	}
	if service.store != nil && wallCacheable(wall) {
		if err := service.store.SaveWallCache(ctx, cacheDate, wall); err != nil {
			return ImageWall{}, fmt.Errorf("save daily image wall cache: %w", err)
		}
	}
	return wall, nil
}

// Refresh forces the daily cache to be regenerated. It is used by the scheduler;
// request handlers still call Wall so they can serve existing cached data.
func (service *CachedWallService) Refresh(ctx context.Context, at time.Time) error {
	if service == nil || service.upstream == nil {
		return errors.New("image wall service is not configured")
	}
	if service.store == nil {
		_, err := service.upstream.Wall(ctx, at)
		return err
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	wall, err := service.upstream.Wall(ctx, at)
	if err != nil {
		return err
	}
	if !wallCacheable(wall) {
		return errors.New("daily image wall response has no valid image-only APOD windows")
	}
	return service.store.SaveWallCache(ctx, wallCacheDate(at), wall)
}

func (service *CachedWallService) latestCachedWall(ctx context.Context) (ImageWall, bool) {
	if service == nil || service.store == nil {
		return ImageWall{}, false
	}
	wall, ok, err := service.store.LatestWallCache(ctx)
	if err != nil || !ok || !wallCacheable(wall) {
		return ImageWall{}, false
	}
	return wall, true
}

func wallCacheDate(at time.Time) time.Time {
	utc := at.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func wallCacheable(wall ImageWall) bool {
	if len(wall.Recent) == 0 {
		return false
	}
	for _, window := range wall.Recent {
		if window.Status != "ready" || window.MediaType != "image" {
			return false
		}
	}
	return true
}
