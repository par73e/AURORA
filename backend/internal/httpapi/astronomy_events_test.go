package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aurora/backend/internal/astronomyevent"
	"aurora/backend/internal/observatory"
)

type astronomyEventStoreStub struct {
	query  astronomyevent.ListQuery
	events []astronomyevent.Event
	err    error
}

func (s *astronomyEventStoreStub) List(_ context.Context, query astronomyevent.ListQuery) ([]astronomyevent.Event, error) {
	s.query = query
	return s.events, s.err
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
