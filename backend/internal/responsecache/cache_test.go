package responsecache

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type memoryStore struct {
	mu   sync.Mutex
	data map[string][]byte
	fail bool
}

func (s *memoryStore) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail {
		return nil, errors.New("offline")
	}
	return s.data[key], nil
}
func (s *memoryStore) Set(_ context.Context, key string, data []byte, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail {
		return errors.New("offline")
	}
	s.data[key] = data
	return nil
}

func setup(t *testing.T, handler http.HandlerFunc) (*Cache, *memoryStore) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	u, _ := url.Parse(server.URL)
	store := &memoryStore{data: make(map[string][]byte)}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return New(ctx, u, store), store
}
func get(c *Cache, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	return w
}

func TestCacheHitPreservesPayloadAndDoesNotReadUpstream(t *testing.T) {
	var calls atomic.Int32
	c, store := setup(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"syncedAt":"original-time","items":[1,2]}`)
	})
	first := get(c, "/api/v1/orbit/overview")
	second := get(c, "/api/v1/orbit/overview")
	if first.Code != 200 || second.Code != 200 || first.Body.String() != second.Body.String() || calls.Load() != 1 || second.Header().Get("X-Aurora-Cache") != "HIT" {
		t.Fatalf("cache failed: calls=%d first=%s second=%s", calls.Load(), first.Body, second.Body)
	}
	for _, data := range store.data {
		if !strings.Contains(string(data), `"body":{"syncedAt"`) {
			t.Fatalf("payload is not inspectable JSON: %s", data)
		}
	}
}

func TestSingleFlightAndConcurrentHit(t *testing.T) {
	var calls atomic.Int32
	c, _ := setup(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(30 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	})
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := get(c, "/api/v1/moon/spacecraft")
			if w.Code != 200 {
				t.Errorf("status=%d", w.Code)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("concurrent cold cache generated %d upstream calls", calls.Load())
	}
}

func TestRedisFailureFallsBackAndOpensCircuit(t *testing.T) {
	var calls atomic.Int32
	c, s := setup(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	})
	s.fail = true
	for i := 0; i < 2; i++ {
		if w := get(c, "/api/v1/orbit/overview"); w.Code != 200 {
			t.Fatalf("cache outage broke API: %d", w.Code)
		}
	}
	if calls.Load() != 2 || c.errors.Load() != 1 {
		t.Fatalf("fallback/circuit: upstream=%d errors=%d", calls.Load(), c.errors.Load())
	}
}

func TestErrorsAndPartialAuthResponsesAreNotStored(t *testing.T) {
	for _, kind := range []string{"error", "cookie", "private", "vary", "invalid-json"} {
		t.Run(kind, func(t *testing.T) {
			var calls atomic.Int32
			c, s := setup(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				switch kind {
				case "error":
					w.WriteHeader(502)
				case "cookie":
					w.Header().Set("Set-Cookie", "session=example")
				case "private":
					w.Header().Set("Cache-Control", "private")
				case "vary":
					w.Header().Set("Vary", "Accept-Language")
				case "invalid-json":
					fmt.Fprint(w, "not json")
					return
				}
				fmt.Fprint(w, `{"ok":false}`)
			})
			get(c, "/api/v1/orbit/overview")
			get(c, "/api/v1/orbit/overview")
			if calls.Load() != 2 || len(s.data) != 0 {
				t.Fatalf("unsafe response cached: calls=%d keys=%d", calls.Load(), len(s.data))
			}
		})
	}
}

func TestPolicyExcludesPersonalAndTimeSensitiveRequests(t *testing.T) {
	for _, path := range []string{"/api/health", "/api/v1/astronomy/score?at=123", "/api/v1/astronomy/moon?at=123", "/api/v1/astronomy/conditions?latitude=1&longitude=2&time=123", "/api/v1/orbit/overview?unexpected=1", "/api/v1/astronomy/events?latitude=1&latitude=2"} {
		if key, _ := policy(httptest.NewRequest("GET", path, nil)); key != "" {
			t.Fatalf("unexpected cache policy for %s", path)
		}
	}
	for _, header := range []string{"Cookie", "Authorization", "Range"} {
		r := httptest.NewRequest("GET", "/api/v1/orbit/overview", nil)
		r.Header.Set(header, "value")
		if key, _ := policy(r); key != "" {
			t.Fatal("personal request cached", header)
		}
	}
}

func TestLocationIsolationAndCanonicalQuery(t *testing.T) {
	key := func(path string) string { k, _ := policy(httptest.NewRequest("GET", path, nil)); return k }
	a := key("/api/v1/astronomy/events?latitude=31.123456&longitude=121.123456")
	b := key("/api/v1/astronomy/events?longitude=121.123456&latitude=31.123456")
	c := key("/api/v1/astronomy/events?latitude=31.123457&longitude=121.123456")
	if a == "" || a != b || a == c || strings.Contains(a, "31.123456") {
		t.Fatalf("incorrect coordinate/query key handling")
	}
}

func TestOversizedResponsesPassThroughIntact(t *testing.T) {
	body := `{"text":"` + strings.Repeat("a", maxBody) + `"}`
	c, s := setup(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	})
	w := get(c, "/api/v1/orbit/overview")
	if w.Body.String() != body || len(s.data) != 0 || w.Header().Get("X-Aurora-Cache") != "BYPASS" {
		t.Fatal("oversized response truncated or cached")
	}
}

func TestCancelledWaiterDoesNotCancelSharedLoad(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	c, _ := setup(t, func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	})
	ctx, cancel := context.WithCancel(context.Background())
	r := httptest.NewRequest("GET", "/api/v1/orbit/overview", nil).WithContext(ctx)
	done := make(chan struct{})
	go func() { c.ServeHTTP(httptest.NewRecorder(), r); close(done) }()
	<-started
	cancel()
	<-done
	close(release)
	w := get(c, "/api/v1/orbit/overview")
	if w.Code != 200 {
		t.Fatalf("shared load cancelled: %d", w.Code)
	}
}
