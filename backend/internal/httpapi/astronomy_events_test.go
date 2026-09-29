package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aurora/backend/internal/astronomyevent"
	"aurora/backend/internal/observatory"
)

type astronomyEventStoreStub struct {
	query   astronomyevent.ListQuery
	events  []astronomyevent.Event
	sources []astronomyevent.SourceStatus
	err     error
}

func (s *astronomyEventStoreStub) List(_ context.Context, query astronomyevent.ListQuery) ([]astronomyevent.Event, error) {
	s.query = query
	return s.events, s.err
}

func (s *astronomyEventStoreStub) ListSourceStatuses(_ context.Context) ([]astronomyevent.SourceStatus, error) {
	return s.sources, s.err
}

func TestAstronomyEventsHandlerUsesRequestedDateRange(t *testing.T) {
	store := &astronomyEventStoreStub{events: []astronomyevent.Event{{ID: "perseids-2026"}}}
	handler := astronomyEventsHandler(store, nil, func() time.Time { return time.Date(2026, 8, 11, 9, 0, 0, 0, time.UTC) })
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/events?from=2026-08-12&to=2026-09-01", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if got, want := store.query.From.Format(time.DateOnly), "2026-08-12"; got != want {
		t.Errorf("from = %s, want %s", got, want)
	}
	if got, want := store.query.To.Format(time.DateOnly), "2026-09-01"; got != want {
		t.Errorf("to = %s, want %s", got, want)
	}
}

func TestAstronomyEventsHandlerDefaultsToThirtyDaysFromNow(t *testing.T) {
	now := time.Date(2026, 8, 11, 9, 37, 0, 0, time.UTC)
	store := &astronomyEventStoreStub{}
	handler := astronomyEventsHandler(store, nil, func() time.Time { return now })
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/events", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !store.query.From.Equal(now) {
		t.Errorf("from = %s, want current instant %s", store.query.From, now)
	}
	if want := now.Add(30 * 24 * time.Hour); !store.query.To.Equal(want) {
		t.Errorf("to = %s, want %s", store.query.To, want)
	}
}

func TestAstronomyEventsHandlerReturnsSourceStatuses(t *testing.T) {
	success := false
	store := &astronomyEventStoreStub{sources: []astronomyevent.SourceStatus{{Code: "imo_meteor_calendar", Name: "IMO", Success: &success, Error: "temporarily offline"}}}
	handler := astronomyEventsHandler(store, nil, time.Now)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/events", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"code":"imo_meteor_calendar"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAstronomyEventsHandlerRepairsStoredHorizonsLink(t *testing.T) {
	const apiURL = "https://ssd.jpl.nasa.gov/api/horizons.api"
	const pageURL = "https://ssd.jpl.nasa.gov/horizons/app.html"
	store := &astronomyEventStoreStub{events: []astronomyevent.Event{{
		ID: "saturn-opposition-20261004", SourceCode: "jpl_horizons_events",
		SourceName: "NASA/JPL Horizons · 天象星历", SourceURL: apiURL,
	}}}
	handler := astronomyEventsHandler(store, nil, time.Now)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/events", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Events []struct {
			SourceURL string `json:"sourceUrl"`
			Source    struct {
				URL string `json:"url"`
			} `json:"source"`
		} `json:"events"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Events) != 1 || response.Events[0].SourceURL != pageURL || response.Events[0].Source.URL != pageURL {
		t.Fatalf("legacy source link was not repaired: %+v", response.Events)
	}
}

func TestAstronomyEventsHandlerRepairsStoredAuroraModelLink(t *testing.T) {
	const codeURL = "https://github.com/par73e/AURORA/blob/main/backend/internal/observatory/calendar.go"
	store := &astronomyEventStoreStub{events: []astronomyevent.Event{{
		ID: "full-moon-20261004", SourceCode: "aurora_astronomy_model",
		SourceName: "AURORA 天文计算模型", SourceURL: "https://aurora.local/astronomy-model",
	}}}
	handler := astronomyEventsHandler(store, nil, time.Now)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/events", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Events []struct {
			SourceURL string `json:"sourceUrl"`
			Source    struct {
				URL string `json:"url"`
			} `json:"source"`
		} `json:"events"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Events) != 1 || response.Events[0].SourceURL != codeURL || response.Events[0].Source.URL != codeURL {
		t.Fatalf("legacy source link was not repaired: %+v", response.Events)
	}
}

func TestAstronomyEventsHandlerSortsLocalVisibilityAndReturnsSourceObject(t *testing.T) {
	store := &astronomyEventStoreStub{events: []astronomyevent.Event{
		{ID: "season", Kind: "march_equinox", StartsAt: time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC), SourceName: "AURORA", SourceURL: "https://aurora.local", Origin: "computed", VerifiedAt: "2026-08-11"},
		{ID: "unknown", Kind: "unknown", StartsAt: time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC), Summary: "全球事件说明", SourceName: "AURORA", SourceURL: "https://aurora.local", Origin: "computed", VerifiedAt: "2026-08-11"},
	}}
	handler := astronomyEventsHandler(store, observatory.NewVisibilitySolver(nil), time.Now)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/events?latitude=31.23&longitude=121.47&timezone=Asia/Shanghai", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Events []struct {
			ID     string         `json:"id"`
			Global map[string]any `json:"global"`
			Source struct {
				Kind string `json:"kind"`
				Name string `json:"name"`
			} `json:"source"`
		} `json:"events"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Events) != 2 || response.Events[0].ID != "unknown" {
		t.Fatalf("events sorted incorrectly: %+v", response.Events)
	}
	if got := response.Events[0].Source; got.Kind != "computed" || got.Name != "AURORA" {
		t.Errorf("source=%+v, want computed AURORA", got)
	}
	if got, want := response.Events[0].Global["description"], "全球事件说明"; got != want {
		t.Errorf("global description=%v, want %q", got, want)
	}
}

func TestAstronomyEventsHandlerRejectsInvalidRange(t *testing.T) {
	store := &astronomyEventStoreStub{}
	handler := astronomyEventsHandler(store, nil, time.Now)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/events?from=2026-09-01&to=2026-08-12", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAstronomyEventsHandlerHidesStoreErrors(t *testing.T) {
	store := &astronomyEventStoreStub{err: errors.New("database unavailable")}
	handler := astronomyEventsHandler(store, nil, time.Now)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/events", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAstronomyEventsHandlerValidatesLocationTogether(t *testing.T) {
	store := &astronomyEventStoreStub{}
	handler := astronomyEventsHandler(store, observatory.NewVisibilitySolver(nil), time.Now)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/astronomy/events?latitude=31.23", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestDeduplicateAstronomyEventsPrefersComputableRecordAndKeepsCuratedPresentation(t *testing.T) {
	at := time.Date(2026, time.August, 15, 5, 59, 0, 0, time.UTC)
	curatedPresentation := json.RawMessage(`{"advice":"等待日落后再观察"}`)
	events := []astronomyevent.Event{
		{ID: "venus-greatest-eastern-elongation-2026", Kind: "planetary_elongation", Title: "金星东大距", StartsAt: at, Origin: "curated", Presentation: curatedPresentation},
		{ID: "planetary_elongation-20260814-venus", Kind: "planetary_elongation", Title: "金星东大距", StartsAt: at.Add(-25 * time.Hour), Origin: "computed", Geometry: json.RawMessage(`{"object":"venus","positions":{"venus":{"longitudeDegrees":120,"latitudeDegrees":0}}}`)},
	}
	got := deduplicateAstronomyEvents(events)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %+v", len(got), got)
	}
	if got[0].Origin != "computed" || got[0].ID != "planetary_elongation-20260814-venus" {
		t.Fatalf("canonical event = %+v, want computed event", got[0])
	}
	if string(got[0].Presentation) != string(curatedPresentation) {
		t.Fatalf("presentation = %s, want curated copy %s", got[0].Presentation, curatedPresentation)
	}
}

func TestDeduplicateAstronomyEventsPrefersBesselianSolarRecord(t *testing.T) {
	at := time.Date(2027, time.August, 2, 10, 6, 34, 0, time.UTC)
	events := []astronomyevent.Event{
		{ID: "nasa-catalog", Kind: "solar_eclipse", StartsAt: at.Add(time.Minute), Origin: "external_forecast", Geometry: json.RawMessage(`{"precision":"nasa_gsfc_table"}`)},
		{ID: "nasa-besselian", Kind: "solar_eclipse", StartsAt: at, Origin: "external_forecast", Geometry: json.RawMessage(`{"precision":"nasa_gsfc_besselian","besselian":{"dt":76}}`)},
	}
	got := deduplicateAstronomyEvents(events)
	if len(got) != 1 || got[0].ID != "nasa-besselian" {
		t.Fatalf("canonical eclipse=%+v, want Besselian record", got)
	}
}

func TestDeduplicateAstronomyEventsKeepsLaterIndependentEvent(t *testing.T) {
	at := time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC)
	events := []astronomyevent.Event{
		{ID: "venus-elongation-august", Kind: "planetary_elongation", Title: "金星东大距", StartsAt: at, Origin: "computed", Geometry: json.RawMessage(`{"object":"venus"}`)},
		{ID: "venus-elongation-december", Kind: "planetary_elongation", Title: "金星西大距", StartsAt: at.AddDate(0, 4, 0), Origin: "computed", Geometry: json.RawMessage(`{"object":"venus"}`)},
	}
	if got := deduplicateAstronomyEvents(events); len(got) != 2 {
		t.Fatalf("len = %d, want two independent elongations", len(got))
	}
}

func TestDeduplicateAstronomyEventsKeepsDifferentMeteorShowers(t *testing.T) {
	at := time.Date(2026, time.August, 12, 16, 0, 0, 0, time.UTC)
	events := []astronomyevent.Event{
		{ID: "perseids-2026", Kind: "meteor_shower", Title: "英仙座流星雨极大", StartsAt: at, Origin: "curated"},
		{ID: "imo-meteor_shower-per-20260812", Kind: "meteor_shower", Title: "英仙座流星雨极大", StartsAt: at, Origin: "external_forecast", Geometry: json.RawMessage(`{"slug":"perseids"}`)},
		{ID: "aurigids-2026", Kind: "meteor_shower", Title: "御夫座流星雨极大", StartsAt: at, Origin: "curated"},
	}
	got := deduplicateAstronomyEvents(events)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %+v", len(got), got)
	}
	if got[0].Origin != "external_forecast" {
		t.Fatalf("perseids canonical origin = %s, want external_forecast", got[0].Origin)
	}
}
