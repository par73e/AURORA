package dailyimage

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type wallAPODStub struct {
	image Image
	err   error
}

func (stub wallAPODStub) Daily(context.Context, time.Time) (Image, error) {
	return stub.image, stub.err
}

type wallLibraryStub struct {
	window ImageWindow
	err    error
}

func (stub wallLibraryStub) Pick(context.Context, time.Time) (ImageWindow, error) {
	return stub.window, stub.err
}

func TestWallServiceKeepsSourceFailuresInsideTheirWindow(t *testing.T) {
	service := NewWallService(
		wallAPODStub{image: Image{Date: "2026-08-12", Title: "APOD", MediaType: "image", URL: "https://example.test/apod.jpg", SourceName: "NASA APOD", SourceURL: "https://example.test/apod"}},
		wallLibraryStub{err: errors.New("upstream unavailable")},
	)
	wall, err := service.Wall(context.Background(), time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(wall.Recent) != 5 || len(wall.Collection) != 13 {
		t.Fatalf("recent=%d collection=%d, want 5 and 13", len(wall.Recent), len(wall.Collection))
	}
	if wall.Recent[0].SourceID != "apod" || wall.Recent[0].Status != "ready" {
		t.Fatalf("APOD window = %#v", wall.Recent[0])
	}
	if got := NewWallService(wallAPODStub{err: errors.New("APOD offline")}, wallLibraryStub{}).apodWindow(context.Background(), time.Now()).SourceURL; got != "https://apod.nasa.gov/apod/astropix.html" {
		t.Fatalf("APOD fallback source URL = %q", got)
	}
	if wall.Collection[0].SourceID != "nasa-library" || wall.Collection[0].Status != "error" {
		t.Fatalf("library window = %#v", wall.Collection[0])
	}
	for _, window := range wall.Collection[5:] {
		if window.Status != "ready" || window.Credit == "" || window.SourceURL == "" {
			t.Fatalf("curated window is incomplete: %#v", window)
		}
	}
}

func TestNASAImageLibraryClientUsesPublicSearchAndCachesResult(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("media_type") != "image" || r.URL.Query().Get("q") == "" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"collection":{"items":[{"data":[{"nasa_id":"ARC-001","title":"A NASA archive image","description":"An archive description","date_created":"2025-01-02T00:00:00Z","center":"JPL"}],"links":[{"href":"https://example.test/thumb.jpg"}]}]}}`))
	}))
	defer server.Close()
	client := NewNASAImageLibraryClient()
	client.baseURL = server.URL
	client.httpClient = server.Client()
	at := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	first, err := client.Pick(context.Background(), at)
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.Pick(context.Background(), at)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || first.SourceURL != "https://images.nasa.gov/details/ARC-001" || second.Title != first.Title {
		t.Fatalf("calls=%d first=%#v second=%#v", calls, first, second)
	}
}
