package httpapi

import (
	"context"
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
