// Package responsecache caches an explicit allowlist of public JSON responses.
// The existing API remains the source of truth and also handles every bypass.
package responsecache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

const maxBody = 1 << 20

type Store interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte, time.Duration) error
}

type entry struct {
	Body     json.RawMessage `json:"body"`
	StoredAt time.Time       `json:"storedAt"`
}

type result struct {
	body     []byte
	header   http.Header
	status   int
	storedAt time.Time
	hit      bool
}

type Cache struct {
	store        Store
	upstream     *url.URL
	proxy        *httputil.ReverseProxy
	client       *http.Client
	ctx          context.Context
	group        singleflight.Group
	blockedUntil atomic.Int64
	hits         atomic.Uint64
	misses       atomic.Uint64
	errors       atomic.Uint64
}

func New(ctx context.Context, upstream *url.URL, store Store) *Cache {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil // The upstream is always the local API, never an outbound proxy.
	transport.MaxConnsPerHost = 16
	transport.MaxIdleConnsPerHost = 8
	transport.ResponseHeaderTimeout = 15 * time.Second
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	proxy.Transport = transport
	return &Cache{ctx: ctx, upstream: upstream, store: store, proxy: proxy, client: &http.Client{
		Transport: transport, Timeout: 20 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func policy(r *http.Request) (string, time.Duration) {
	if r.Method != http.MethodGet || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("Range") != "" || strings.Contains(r.Header.Get("Cache-Control"), "no-cache") {
		return "", 0
	}
	var ttl time.Duration
	var allowed string
	switch r.URL.Path {
	case "/api/v1/orbit/overview":
		ttl = time.Minute
	case "/api/v1/moon/spacecraft", "/api/v1/mars/spacecraft":
		ttl = 2 * time.Minute
	case "/api/v1/moon/landing-sites", "/api/v1/mars/landing-sites":
		ttl = 6 * time.Hour
	case "/api/v1/voyage/probes", "/api/v1/astronomy/image-wall":
		ttl = 5 * time.Minute
	case "/api/v1/astronomy/events":
		ttl, allowed = 5*time.Minute, "|from|to|latitude|longitude|elevation|timezone|"
	case "/api/v1/astronomy/conditions":
		ttl, allowed = 2*time.Minute, "|latitude|longitude|scores|"
	default:
		return "", 0
	}
	if len(r.URL.RawQuery) > 1024 {
		return "", 0
	}
	params, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return "", 0
	}
	for key, values := range params {
		if len(values) != 1 || !strings.Contains(allowed, "|"+key+"|") {
			return "", 0
		}
	}
	// Date-sensitive default ranges and image selection must not cross UTC midnight.
	now := time.Now().UTC()
	if remaining := now.Truncate(24 * time.Hour).Add(24 * time.Hour).Sub(now); ttl > remaining {
		ttl = remaining
	}
	key := "aurora:prod:response:v1:" + strings.TrimPrefix(r.URL.Path, "/api/v1/")
	if len(params) > 0 {
		digest := sha256.Sum256([]byte(params.Encode()))
		key += ":" + hex.EncodeToString(digest[:])
	}
	return key, ttl
}

func (c *Cache) failed() {
	c.errors.Add(1)
	until := time.Now().Add(10 * time.Second).UnixNano()
	previous := c.blockedUntil.Swap(until)
	if previous < time.Now().UnixNano() {
		slog.Warn("Redis cache unavailable; using existing API")
	}
}

func (c *Cache) read(key string) (result, bool) {
	if c.store == nil || time.Now().UnixNano() < c.blockedUntil.Load() {
		return result{}, false
	}
	ctx, cancel := context.WithTimeout(c.ctx, 60*time.Millisecond)
	defer cancel()
	data, err := c.store.Get(ctx, key)
	if err != nil {
		c.failed()
		return result{}, false
	}
	if len(data) == 0 {
		return result{}, false
	}
	var value entry
	if len(data) > maxBody+1024 || json.Unmarshal(data, &value) != nil || !json.Valid(value.Body) {
		return result{}, false
	}
	return result{body: value.Body, status: http.StatusOK, storedAt: value.StoredAt, hit: true,
		header: http.Header{"Content-Type": {"application/json; charset=utf-8"}}}, true
}

func (c *Cache) load(r *http.Request, key string, ttl time.Duration) (result, error) {
	// Double-check after joining the flight: another request may have just filled it.
	if value, ok := c.read(key); ok {
		return value, nil
	}
	ctx, cancel := context.WithTimeout(c.ctx, 20*time.Second)
	defer cancel()
	u := *c.upstream
	u.Path, u.RawQuery = r.URL.Path, r.URL.RawQuery
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return result{}, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return result{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBody+1))
	if err != nil {
		return result{}, err
	}
	if len(body) > maxBody {
		return result{status: http.StatusTemporaryRedirect}, nil
	}
	value := result{body: body, header: response.Header.Clone(), status: response.StatusCode, storedAt: time.Now().UTC()}
	cacheable := response.StatusCode == http.StatusOK && json.Valid(body) &&
		strings.HasPrefix(response.Header.Get("Content-Type"), "application/json") &&
		response.Header.Get("Set-Cookie") == "" && response.Header.Get("Vary") == "" &&
		!strings.Contains(response.Header.Get("Cache-Control"), "no-store") &&
		!strings.Contains(response.Header.Get("Cache-Control"), "private")
	if cacheable && c.store != nil && time.Now().UnixNano() >= c.blockedUntil.Load() {
		data, err := json.Marshal(entry{Body: body, StoredAt: value.storedAt})
		if err == nil {
			ctx, cancel := context.WithTimeout(c.ctx, 60*time.Millisecond)
			defer cancel()
			if err := c.store.Set(ctx, key, data, ttl); err != nil {
				c.failed()
			}
		}
	}
	return value, nil
}

func (c *Cache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key, ttl := policy(r)
	if key == "" || c.store == nil || time.Now().UnixNano() < c.blockedUntil.Load() {
		w.Header().Set("X-Aurora-Cache", "BYPASS")
		c.proxy.ServeHTTP(w, r)
		return
	}
	value, hit := c.read(key)
	if !hit {
		ch := c.group.DoChan(key, func() (any, error) { return c.load(r, key, ttl) })
		select {
		case <-r.Context().Done():
			return
		case response := <-ch:
			if response.Err != nil {
				http.Error(w, "API temporarily unavailable", http.StatusBadGateway)
				return
			}
			value = response.Val.(result)
		}
	}
	if value.status == http.StatusTemporaryRedirect {
		w.Header().Set("X-Aurora-Cache", "BYPASS")
		c.proxy.ServeHTTP(w, r)
		return
	}
	// Preserve API headers, including failure retry hints; drop hop-by-hop headers.
	value.header = value.header.Clone() // Single-flight waiters share the result.
	for _, name := range strings.Split(value.header.Get("Connection"), ",") {
		value.header.Del(strings.TrimSpace(name))
	}
	for _, name := range []string{"Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "TE", "Trailer", "Transfer-Encoding", "Upgrade", "Content-Length"} {
		value.header.Del(name)
	}
	for name, values := range value.header {
		w.Header()[name] = append([]string(nil), values...)
	}
	if value.hit {
		c.hits.Add(1)
		w.Header().Set("X-Aurora-Cache", "HIT")
	} else {
		c.misses.Add(1)
		w.Header().Set("X-Aurora-Cache", "MISS")
	}
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(value.status)
	_, _ = w.Write(value.body)
}

// Stats are served only on the loopback listener, never under the public /api/ route.
func (c *Cache) Stats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"hits": c.hits.Load(), "misses": c.misses.Load(), "redisErrors": c.errors.Load(), "redisBypassed": time.Now().UnixNano() < c.blockedUntil.Load()})
}
