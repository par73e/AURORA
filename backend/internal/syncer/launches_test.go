package syncer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchUpcomingLaunchesReadsEveryPage(t *testing.T) {
	start := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	cutoff := start.Add(30 * 24 * time.Hour)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "100" || q.Get("ordering") != "net" || q.Get("net__gte") != start.Format(time.RFC3339) || q.Get("net__lte") != cutoff.Format(time.RFC3339) {
			t.Errorf("unexpected launch query: %s", r.URL.RawQuery)
		}
		first := q.Get("offset") == ""
		if first {
			results := make([]map[string]any, 20)
			for i := range results {
				results[i] = map[string]any{"id": fmt.Sprintf("launch-%d", i), "net": start.Add(time.Duration(i+1) * time.Hour).Format(time.RFC3339)}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"results": results, "next": server.URL + r.URL.Path + "?" + q.Encode() + "&offset=20"})
			return
		}
		results := make([]map[string]any, 5)
		for i := range results {
			results[i] = map[string]any{"id": fmt.Sprintf("launch-%d", i+20), "net": start.Add(time.Duration(i+21) * time.Hour).Format(time.RFC3339)}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"results": results, "next": nil})
	}))
	defer server.Close()
	s := &Syncer{client: server.Client()}
	items, err := s.fetchUpcomingLaunches(context.Background(), server.URL+"/launches/upcoming/", start, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 25 {
		t.Fatalf("got %d launches, want 25", len(items))
	}
}

func TestFetchUpcomingLaunchesRejectsIncompletePages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}, "next": "https://example.invalid/other"})
	}))
	defer server.Close()
	s := &Syncer{client: server.Client()}
	if _, err := s.fetchUpcomingLaunches(context.Background(), server.URL+"/launches/upcoming/", time.Now(), time.Now().Add(30*24*time.Hour)); err == nil {
		t.Fatal("cross-origin next page was accepted as a complete sync")
	}
}

func TestFetchUpcomingLaunchesFailsWhenSecondPageFails(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("offset") == "100" {
			http.Error(w, "unavailable", http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{map[string]any{"id": "first"}}, "next": server.URL + r.URL.Path + "?offset=100"})
	}))
	defer server.Close()
	s := &Syncer{client: server.Client()}
	items, err := s.fetchUpcomingLaunches(context.Background(), server.URL+"/launches/upcoming/", time.Now(), time.Now().Add(30*24*time.Hour))
	if err == nil || len(items) != 0 {
		t.Fatalf("partial pagination returned %d items with error %v", len(items), err)
	}
}
