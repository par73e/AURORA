package httpapi

import (
	"aurora/backend/internal/astronomyevent"
	"aurora/backend/internal/dailyimage"
	observerlocation "aurora/backend/internal/location"
	"aurora/backend/internal/observatory"
	"aurora/backend/internal/orbit"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// DemoRouter has no database or astronomy sync jobs. Location uses AMap; weather/air quality use Open-Meteo.
// Snapshots are immutable for a process lifetime; uploading data + restarting
// the service switches the complete data bundle together.
func DemoRouter(dir string) (http.Handler, func(), error) {
	var overview orbit.Overview
	var events []astronomyevent.Event
	var wall dailyimage.ImageWall
	var manifest map[string]any
	for name, target := range map[string]any{"overview": &overview, "events": &events, "image-wall": &wall, "manifest": &manifest} {
		if err := readDemoJSON(dir, name, target); err != nil {
			return nil, nil, err
		}
	}
	if len(overview.Spacecraft) == 0 || len(wall.Recent)+len(wall.Collection) == 0 {
		return nil, nil, errors.New("demo snapshots are incomplete")
	}
	snapshots := map[string]json.RawMessage{}
	for route, name := range map[string]string{"/api/v1/moon/spacecraft": "moon-spacecraft", "/api/v1/moon/landing-sites": "moon-sites", "/api/v1/mars/spacecraft": "mars-spacecraft", "/api/v1/mars/landing-sites": "mars-sites", "/api/v1/voyage/probes": "probes"} {
		var raw json.RawMessage
		if err := readDemoJSON(dir, name, &raw); err != nil {
			return nil, nil, err
		}
		snapshots[route] = raw
	}
	moons := observatory.NewMoonService()
	lights, err := observatory.OpenRasterLightProvider(filepath.Join(dir, "viirs-2025-cn.avnl"))
	if err != nil {
		return nil, nil, fmt.Errorf("local light raster: %w", err)
	}
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Aurora-Mode", "hybrid-demo")
			next.ServeHTTP(w, r)
		})
	})
	router.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"status": "ok", "mode": "hybrid-demo", "exportedAt": manifest["exportedAt"]})
	})
	manifest["mode"] = "hybrid-demo"
	manifest["weatherMode"] = "live"
	manifest["locationMode"] = "live"
	router.Get("/api/v1/demo/meta", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, manifest) })
	router.Get("/api/v1/orbit/overview", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, overview) })
	router.Get("/api/v1/orbit/spacecraft", demoCatalogHandler(overview.Spacecraft))
	for route, raw := range snapshots {
		router.Get(route, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, raw) })
	}
	weather := observatory.NewClientWithURLs("https://api.open-meteo.com/v1/forecast", "https://air-quality-api.open-meteo.com/v1/air-quality", &http.Client{Timeout: 6 * time.Second})
	router.Get("/api/v1/astronomy/conditions", observingConditionsHandler(weather, moons))
	router.Get("/api/v1/astronomy/score", observingScoreHandler(weather, moons, lights))
	router.Get("/api/v1/astronomy/moon", moonHandler(moons))
	router.Get("/api/v1/astronomy/light-pollution", lightPollutionHandler(lights))
	router.Get("/api/v1/astronomy/events", astronomyEventsHandler(demoEvents{events}, observatory.NewVisibilitySolver(moons), time.Now))
	router.Get("/api/v1/astronomy/image-wall", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, wall) })
	router.Get("/api/v1/astronomy/daily-image", func(w http.ResponseWriter, r *http.Request) {
		items := append(append([]dailyimage.ImageWindow{}, wall.Recent...), wall.Collection...)
		item := items[0]
		if date := r.URL.Query().Get("date"); date != "" {
			for _, candidate := range items {
				if strings.HasPrefix(candidate.PublishedAt, date) {
					item = candidate
					break
				}
			}
		}
		writeJSON(w, 200, dailyimage.Image{Date: item.PublishedAt, Title: item.Title, Explanation: item.Summary, MediaType: "image", URL: item.ImageURL, HDURL: item.HDURL, Copyright: item.Credit, SourceName: item.SourceName, SourceURL: item.SourceURL})
	})
	places := observerlocation.NewAMapClient(os.Getenv("AMAP_WEB_KEY"))
	router.Get("/api/v1/location/reverse", reverseLocationHandler(places))
	router.Get("/api/v1/location/search", searchLocationHandler(places))
	return router, func() { _ = lights.Close() }, nil
}
func readDemoJSON(dir, name string, target any) error {
	b, e := os.ReadFile(filepath.Join(dir, name+".json"))
	if e != nil {
		return fmt.Errorf("read %s: %w", name, e)
	}
	if e = json.Unmarshal(b, target); e != nil {
		return fmt.Errorf("decode %s: %w", name, e)
	}
	return nil
}

type demoEvents struct{ events []astronomyevent.Event }

func (s demoEvents) List(_ context.Context, q astronomyevent.ListQuery) ([]astronomyevent.Event, error) {
	computed := astronomyevent.CoreEvents(q.From, q.To)
	kinds := map[string]bool{}
	for _, e := range computed {
		kinds[e.Kind] = true
	}
	result := append([]astronomyevent.Event{}, computed...)
	for _, e := range s.events {
		if !e.StartsAt.Before(q.From) && e.StartsAt.Before(q.To) && !kinds[e.Kind] {
			result = append(result, e)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].StartsAt.Before(result[j].StartsAt) })
	return result, nil
}
func demoCatalogHandler(items []orbit.Spacecraft) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		page, e := positiveInt(q.Get("page"), 1)
		if e != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid page"})
			return
		}
		size, e := positiveInt(q.Get("pageSize"), 20)
		if e != nil || size > 200 {
			writeJSON(w, 400, map[string]string{"error": "pageSize must be 1–200"})
			return
		}
		if page > 1000000 {
			writeJSON(w, 400, map[string]string{"error": "page too large"})
			return
		}
		mode := q.Get("mode")
		if mode != "" && mode != "keyword" && mode != "regex" {
			writeJSON(w, 400, map[string]string{"error": "invalid mode"})
			return
		}
		query := q.Get("q")
		if len(query) > 256 {
			writeJSON(w, 400, map[string]string{"error": "query too long"})
			return
		}
		var rx *regexp.Regexp
		if mode == "regex" {
			rx, e = regexp.Compile("(?i)" + query)
			if e != nil {
				writeJSON(w, 400, map[string]string{"error": "invalid regex"})
				return
			}
		}
		result := []orbit.Spacecraft{}
		for _, item := range items {
			if op := q.Get("operator"); op != "" && op != "all" && item.OperatorName != op {
				continue
			}
			text := fmt.Sprintf("%s %s %s %d", item.NameZH, item.NameEN, item.OperatorName, item.NORADCatalogID)
			matches := strings.Contains(strings.ToLower(text), strings.ToLower(query))
			if rx != nil {
				matches = rx.MatchString(text)
			}
			if matches {
				result = append(result, item)
			}
		}
		sort.SliceStable(result, func(i, j int) bool {
			switch q.Get("sort") {
			case "norad":
				return result[i].NORADCatalogID < result[j].NORADCatalogID
			case "operator":
				return result[i].OperatorName < result[j].OperatorName
			default:
				return result[i].NameZH < result[j].NameZH
			}
		})
		total := len(result)
		start := (page - 1) * size
		if start > total {
			start = total
		}
		end := min(start+size, total)
		writeJSON(w, 200, orbit.SpacecraftPage{Items: result[start:end], Page: page, PageSize: size, Total: int64(total)})
	}
}
