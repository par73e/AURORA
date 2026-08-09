package location

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAMapClientReverseConvertsGPSAndReturnsDistrict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v3/assistant/coordinate/convert":
			if got := r.URL.Query().Get("coordsys"); got != "gps" {
				t.Errorf("coordsys = %q, want gps", got)
			}
			if got := r.URL.Query().Get("locations"); got != "121.473700,31.230400" {
				t.Errorf("locations = %q", got)
			}
			_, _ = io.WriteString(w, `{"status":"1","info":"OK","infocode":"10000","locations":"121.478223,31.228442"}`)
		case "/v3/geocode/regeo":
			if got := r.URL.Query().Get("location"); got != "121.478223,31.228442" {
				t.Errorf("location = %q", got)
			}
			_, _ = io.WriteString(w, `{"status":"1","info":"OK","infocode":"10000","regeocode":{"formatted_address":"上海市黄浦区测试路","addressComponent":{"province":"上海市","city":[],"district":"黄浦区","township":"南京东路街道","adcode":"310101"}}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewAMapClient("test-key")
	client.baseURL = server.URL
	client.httpClient = server.Client()
	place, err := client.Reverse(context.Background(), 31.2304, 121.4737)
	if err != nil {
		t.Fatalf("Reverse() error = %v", err)
	}
	if place.Label != "上海市 · 黄浦区" {
		t.Fatalf("Label = %q", place.Label)
	}
	if place.City != "上海市" || place.District != "黄浦区" || place.Adcode != "310101" {
		t.Fatalf("unexpected place: %#v", place)
	}
}

func TestAMapClientReverseRequiresKey(t *testing.T) {
	_, err := NewAMapClient("  ").Reverse(context.Background(), 31.2, 121.4)
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("error = %v, want ErrNotConfigured", err)
	}
}

func TestAMapClientRequestErrorDoesNotExposeKey(t *testing.T) {
	client := NewAMapClient("private-test-key")
	client.httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return nil, errors.New(request.URL.String())
	})}
	_, err := client.Reverse(context.Background(), 31.2, 121.4)
	if err == nil || strings.Contains(err.Error(), "private-test-key") {
		t.Fatalf("error exposed key: %v", err)
	}
}

func TestPlaceLabelAvoidsDuplicateArea(t *testing.T) {
	if got := placeLabel("北京市", "北京市", "北京市"); got != "北京市" {
		t.Fatalf("placeLabel() = %q", got)
	}
}

func TestAMapClientLive(t *testing.T) {
	if os.Getenv("AMAP_LIVE_TEST") != "1" {
		t.Skip("set AMAP_LIVE_TEST=1 to call the real AMap service")
	}
	key := os.Getenv("AMAP_WEB_KEY")
	if key == "" {
		t.Fatal("AMAP_WEB_KEY is empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	place, err := NewAMapClient(key).Reverse(ctx, 31.2304, 121.4737)
	if err != nil {
		t.Fatalf("live reverse geocoding failed: %v", err)
	}
	if place.City == "" || place.District == "" {
		t.Fatalf("live response has no city or district: %#v", place)
	}
	t.Logf("resolved test coordinate to %s", place.Label)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
