package dailyimage

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestAPODClientDailyMapsImageAndKeepsKeyPrivate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("api_key"); got != "private-key" {
			t.Fatalf("api key = %q", got)
		}
		if got := r.URL.Query().Get("date"); got != "2026-08-12" {
			t.Fatalf("date = %q", got)
		}
		_, _ = w.Write([]byte(`{"date":"2026-08-12","title":"Perseids","explanation":"A verified explanation.","media_type":"image","url":"https://example.test/image.jpg","hdurl":"https://example.test/image-hd.jpg","copyright":"A Photographer"}`))
	}))
	defer server.Close()
	client := NewAPODClient("private-key")
	client.baseURL = server.URL
	client.httpClient = server.Client()
	image, err := client.Daily(context.Background(), time.Date(2026, 8, 12, 15, 0, 0, 0, time.FixedZone("CST", 8*3600)))
	if err != nil {
		t.Fatal(err)
	}
	if image.Title != "Perseids" || image.MediaType != "image" || image.SourceName == "" {
		t.Fatalf("image = %#v", image)
	}
	if image.SourceURL != "https://apod.nasa.gov/apod/ap260812.html" {
		t.Fatalf("source URL = %q", image.SourceURL)
	}
}

func TestAPODClientDailyRequiresKey(t *testing.T) {
	_, err := NewAPODClient(" ").Daily(context.Background(), time.Now())
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("error = %v", err)
	}
}

func TestAPODClientRecentMapsNewestPublishedImages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("start_date"); got != "2026-08-03" {
			t.Fatalf("start_date = %q", got)
		}
		_, _ = w.Write([]byte(`[{"date":"2026-08-10","title":"Older","explanation":"x","media_type":"image","url":"https://example.test/older.jpg"},{"date":"2026-08-11","title":"Video","explanation":"x","media_type":"video","url":"https://example.test/video.mp4"},{"date":"2026-08-12","title":"Newest","explanation":"x","media_type":"image","url":"https://example.test/newest.jpg"}]`))
	}))
	defer server.Close()
	client := NewAPODClient("private-key")
	client.baseURL = server.URL
	client.httpClient = server.Client()
	images, err := client.Recent(context.Background(), time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC), 2)
	if err != nil || len(images) != 2 || images[0].Title != "Newest" || images[1].Title != "Older" {
		t.Fatalf("images=%#v err=%v", images, err)
	}
}

func TestAPODClientLive(t *testing.T) {
	if os.Getenv("NASA_LIVE_TEST") != "1" {
		t.Skip("set NASA_LIVE_TEST=1 with NASA_API_KEY to call NASA APOD")
	}
	key := os.Getenv("NASA_API_KEY")
	if key == "" {
		t.Fatal("NASA_API_KEY is empty")
	}
	image, err := NewAPODClient(key).Daily(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("APOD request failed: %v", err)
	}
	if image.Title == "" || image.URL == "" {
		t.Fatalf("incomplete APOD image: %#v", image)
	}
}
