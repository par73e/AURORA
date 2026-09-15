package dailyimage

import (
	"context"
	"errors"
	"testing"
	"time"
)

type cacheWallUpstreamStub struct {
	calls int
	wall  ImageWall
	err   error
}

func (stub *cacheWallUpstreamStub) Wall(context.Context, time.Time) (ImageWall, error) {
	stub.calls++
	return stub.wall, stub.err
}

type memoryWallCacheStore struct {
	byDate map[string]ImageWall
}

func newMemoryWallCacheStore() *memoryWallCacheStore {
	return &memoryWallCacheStore{byDate: make(map[string]ImageWall)}
}

func (store *memoryWallCacheStore) GetWallCache(_ context.Context, date time.Time) (ImageWall, bool, error) {
	wall, ok := store.byDate[wallCacheDate(date).Format(time.DateOnly)]
	return wall, ok, nil
}

func (store *memoryWallCacheStore) LatestWallCache(context.Context) (ImageWall, bool, error) {
	var latestKey string
	for key := range store.byDate {
		if key > latestKey {
			latestKey = key
		}
	}
	if latestKey == "" {
		return ImageWall{}, false, nil
	}
	return store.byDate[latestKey], true, nil
}

func (store *memoryWallCacheStore) SaveWallCache(_ context.Context, date time.Time, wall ImageWall) error {
	store.byDate[wallCacheDate(date).Format(time.DateOnly)] = wall
	return nil
}

func TestCachedWallServiceUsesDatabaseCacheForSameDay(t *testing.T) {
	at := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	upstream := &cacheWallUpstreamStub{wall: ImageWall{
		Recent:      []ImageWindow{{ID: "apod-2026-08-12", Status: "ready", MediaType: "image"}},
		Collection:  []ImageWindow{{ID: "eso", Status: "ready"}},
		GeneratedAt: at.Format(time.RFC3339),
	}}
	service := NewCachedWallService(upstream, newMemoryWallCacheStore())
	first, err := service.Wall(context.Background(), at)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Wall(context.Background(), at.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if upstream.calls != 1 {
		t.Fatalf("upstream calls=%d, want 1", upstream.calls)
	}
	if first.Recent[0].ID != second.Recent[0].ID {
		t.Fatalf("cached wall changed: first=%#v second=%#v", first, second)
	}
}

func TestCachedWallServiceFallsBackToLatestCacheWhenRefreshFails(t *testing.T) {
	store := newMemoryWallCacheStore()
	cachedAt := time.Date(2026, 8, 11, 9, 0, 0, 0, time.UTC)
	if err := store.SaveWallCache(context.Background(), cachedAt, ImageWall{
		Recent:      []ImageWindow{{ID: "apod-2026-08-11", Status: "ready", MediaType: "image"}},
		GeneratedAt: cachedAt.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	upstream := &cacheWallUpstreamStub{err: errors.New("upstream unavailable")}
	service := NewCachedWallService(upstream, store)
	wall, err := service.Wall(context.Background(), time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if wall.Recent[0].ID != "apod-2026-08-11" {
		t.Fatalf("fallback wall=%#v", wall)
	}
}

func TestCachedWallServiceReplacesVideoCacheWithImageOnlyWall(t *testing.T) {
	at := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	store := newMemoryWallCacheStore()
	if err := store.SaveWallCache(context.Background(), at, ImageWall{
		Recent: []ImageWindow{{ID: "apod-video", Status: "ready", MediaType: "video"}},
	}); err != nil {
		t.Fatal(err)
	}
	upstream := &cacheWallUpstreamStub{wall: ImageWall{
		Recent: []ImageWindow{{ID: "apod-image", Status: "ready", MediaType: "image"}},
	}}
	service := NewCachedWallService(upstream, store)
	wall, err := service.Wall(context.Background(), at)
	if err != nil {
		t.Fatal(err)
	}
	if upstream.calls != 1 || wall.Recent[0].ID != "apod-image" {
		t.Fatalf("calls=%d wall=%#v", upstream.calls, wall)
	}
}
