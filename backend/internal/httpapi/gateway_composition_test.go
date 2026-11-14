package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"aurora/backend/internal/observatory"
	"aurora/backend/internal/responsecache"
)

type gatewayStore struct {
	mu   sync.Mutex
	data map[string][]byte
}

// 与 observatory 包测试同构的最小合法预报载荷（跨包测试不共享 fixture）。
const gatewayValidWeatherJSON = `{"timezone":"Asia/Shanghai","current":{"time":"2026-10-11T20:00","temperature_2m":18,"dew_point_2m":12,"cloud_cover":0,"cloud_cover_low":0,"cloud_cover_mid":0,"cloud_cover_high":0,"visibility":20000,"relative_humidity_2m":60,"precipitation":0,"wind_speed_10m":2,"wind_gusts_10m":4,"weather_code":0},"hourly":{"time":["2026-10-11T20:00"],"temperature_2m":[18],"dew_point_2m":[12],"cloud_cover":[0],"cloud_cover_low":[0],"cloud_cover_mid":[0],"cloud_cover_high":[0],"visibility":[20000],"relative_humidity_2m":[60],"precipitation":[0],"wind_speed_10m":[2],"wind_gusts_10m":[4],"weather_code":[0]}}`

func (s *gatewayStore) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data[key], nil
}

func (s *gatewayStore) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

func (s *gatewayStore) len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data)
}

// 组合验证：真实 conditions handler 产出的降级响应经过真实缓存网关后，
// 仍保留 no-store、不进入响应缓存，恢复前的每次请求都重新回源。
func TestDegradedConditionsPassThroughGatewayUncached(t *testing.T) {
	var calls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		observingConditionsHandler(degradedProviderStub{}, nil)(w, r)
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := &gatewayStore{data: make(map[string][]byte)}
	gateway := responsecache.New(ctx, target, store)

	path := "/api/v1/astronomy/conditions?latitude=31.230400&longitude=121.473700"
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		gateway.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: status=%d", i, w.Code)
		}
		if got := w.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("request %d: Cache-Control=%q, want no-store", i, got)
		}
		if got := w.Header().Get("X-Aurora-Cache"); got == "HIT" {
			t.Fatalf("request %d: degraded response served from cache", i)
		}
	}
	if calls != 2 || store.len() != 0 {
		t.Fatalf("degraded response cached: upstream calls=%d store keys=%d", calls, store.len())
	}
}

// 组合验证：真实天气客户端经真实 handler 与网关的完整链路。
// 上游持续失败且无旧快照时，502 原样透传且不进入响应缓存。
func TestUpstreamFailureWithoutSnapshotPassesThroughGateway(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/astronomy/conditions" {
			observingConditionsHandler(observatory.NewClientWithURLs("http://127.0.0.1:1/weather", "http://127.0.0.1:1/air", &http.Client{Timeout: 200 * time.Millisecond}), nil)(w, r)
			return
		}
		http.Error(w, "upstream down", http.StatusBadGateway)
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := &gatewayStore{data: make(map[string][]byte)}
	gateway := responsecache.New(ctx, target, store)

	path := "/api/v1/astronomy/conditions?latitude=31.23&longitude=121.47"
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		gateway.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusBadGateway {
			t.Fatalf("request %d: status=%d, want 502", i, w.Code)
		}
	}
	if store.len() != 0 {
		t.Fatalf("failure response cached: keys=%d", store.len())
	}
}

// 端到端隔离验证：真实 TCP 监听 + 真实 HTTP 客户端，覆盖上游健康→失败→恢复。
// 各网格相互独立：已缓存网格在上游失败期间仍由缓存服务；全新网格无旧快照时诚实 502；
// 失败响应不进入响应缓存，上游恢复后同一网格重新回源成功。
func TestDegradationEndToEndOverRealSockets(t *testing.T) {
	var fail atomic.Bool
	fakeUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			http.Error(w, "upstream down", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/air" {
			fmt.Fprint(w, `{"hourly":{"time":[]}}`)
			return
		}
		fmt.Fprint(w, gatewayValidWeatherJSON)
	}))
	defer fakeUpstream.Close()
	client := observatory.NewClientWithURLs(fakeUpstream.URL+"/weather", fakeUpstream.URL+"/air", fakeUpstream.Client())
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observingConditionsHandler(client, nil)(w, r)
	}))
	defer api.Close()
	target, err := url.Parse(api.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := &gatewayStore{data: make(map[string][]byte)}
	gateway := httptest.NewServer(responsecache.New(ctx, target, store))
	defer gateway.Close()

	httpClient := &http.Client{Timeout: 5 * time.Second}
	request := func(label, latitude string) (*http.Response, map[string]any) {
		t.Helper()
		response, err := httpClient.Get(gateway.URL + "/api/v1/astronomy/conditions?latitude=" + latitude + "&longitude=121.473700")
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		defer response.Body.Close()
		var body map[string]any
		if response.StatusCode == http.StatusOK {
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("%s: decode: %v", label, err)
			}
		}
		t.Logf("%-14s lat=%s status=%d cache=%s cache-control=%s degraded=%v", label, latitude, response.StatusCode,
			response.Header.Get("X-Aurora-Cache"), response.Header.Get("Cache-Control"), body != nil && body["degraded"] != nil)
		return response, body
	}

	// 网格 A：健康上游，首次 MISS 写缓存，随后 HIT。
	first, body := request("fresh", "31.230400")
	if first.StatusCode != 200 || first.Header.Get("X-Aurora-Cache") != "MISS" || body["degraded"] != nil {
		t.Fatalf("fresh miss wrong: %d %s", first.StatusCode, first.Header.Get("X-Aurora-Cache"))
	}
	second, _ := request("fresh-again", "31.230499")
	if second.StatusCode != 200 || second.Header.Get("X-Aurora-Cache") != "HIT" {
		t.Fatalf("jittered fresh request missed grid key: %d %s", second.StatusCode, second.Header.Get("X-Aurora-Cache"))
	}

	// 上游失败：已缓存网格 A 继续由缓存服务；全新网格 B 无旧快照，诚实 502。
	fail.Store(true)
	cachedDuringOutage, _ := request("cached-during", "31.230400")
	if cachedDuringOutage.StatusCode != 200 || cachedDuringOutage.Header.Get("X-Aurora-Cache") != "HIT" {
		t.Fatalf("cached grid must survive outage: %d %s", cachedDuringOutage.StatusCode, cachedDuringOutage.Header.Get("X-Aurora-Cache"))
	}
	cold, coldBody := request("cold-no-snapshot", "32.000000")
	if cold.StatusCode != http.StatusBadGateway || coldBody != nil {
		t.Fatalf("cold grid without snapshot must fail honestly: %d", cold.StatusCode)
	}

	// 失败响应不得进入响应缓存：恢复后网格 B 重新回源并成功。
	fail.Store(false)
	recovered, _ := request("recovered", "32.000000")
	if recovered.StatusCode != 200 || recovered.Header.Get("X-Aurora-Cache") != "MISS" {
		t.Fatalf("failure response was cached: %d %s", recovered.StatusCode, recovered.Header.Get("X-Aurora-Cache"))
	}
}

func TestRealClientFreshConditionsAreCachedByGateway(t *testing.T) {
	fakeUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/air" {
			fmt.Fprint(w, `{"hourly":{"time":[]}}`)
			return
		}
		fmt.Fprint(w, gatewayValidWeatherJSON)
	}))
	defer fakeUpstream.Close()
	client := observatory.NewClientWithURLs(fakeUpstream.URL+"/weather", fakeUpstream.URL+"/air", fakeUpstream.Client())
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observingConditionsHandler(client, nil)(w, r)
	}))
	defer api.Close()
	target, err := url.Parse(api.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := &gatewayStore{data: make(map[string][]byte)}
	gateway := responsecache.New(ctx, target, store)

	first := httptest.NewRecorder()
	gateway.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/conditions?latitude=31.230401&longitude=121.473701", nil))
	second := httptest.NewRecorder()
	gateway.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/conditions?latitude=31.230499&longitude=121.473799", nil))
	if first.Code != 200 || second.Code != 200 {
		t.Fatalf("status %d/%d", first.Code, second.Code)
	}
	if got := second.Header().Get("X-Aurora-Cache"); got != "HIT" {
		t.Fatalf("grid key miss on jittered coordinates: %s", got)
	}
	if first.Header().Get("Cache-Control") == "no-store" {
		t.Fatal("fresh response was marked no-store")
	}
}

