package dailyimage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
)

func pageFixture(date, title, media, previous string) string {
	return fmt.Sprintf(`<html><head><title>NASA Science</title></head><body><header><img src="https://science.nasa.gov/themes/nasa-logo.png"></header><nav><img src="https://example.test/news.jpg"></nav><section class="hds-media-detail-hero"><div class="media-detail-hero__media">%s</div><h2>%s</h2><p class="media-detail-hero__description"><strong>Explanation:</strong> Real <a>astronomy</a> description.</p><table><tr class="media-detail-hero__meta-row"><th>Date</th><td>%s</td></tr><tr class="media-detail-hero__meta-row"><th>Credit &amp; Copyright</th><td>Jane Photographer</td></tr></table></section><a class="smd-next-prev__prev" href="%s">Yesterday's Image</a><footer><img src="https://example.test/logo.png"></footer></body></html>`, media, title, date, previous)
}

func TestAPODPageUsesBodyAndPublicationMetadata(t *testing.T) {
	body := pageFixture("October 4, 2026", "Rainbow", `<a href="/image-article/apod-2026-october-4-rainbow/"><img src="/images/rainbow.jpg?w=1200&amp;h=800"></a>`, "/image-article/apod-2026-october-3-mars/")
	doc, _ := html.Parse(strings.NewReader(body))
	base, _ := url.Parse("https://science.nasa.gov/apod/")
	image, prev, err := parseAPODPage(doc, base)
	if err != nil || image.Title != "Rainbow" || image.Date != "2026-10-04" || image.URL != "https://science.nasa.gov/images/rainbow.jpg?w=1200&h=800" || image.Copyright != "Jane Photographer" || image.Explanation != "Real astronomy description." || prev != "https://science.nasa.gov/image-article/apod-2026-october-3-mars/" || image.SourceURL != "https://science.nasa.gov/image-article/apod-2026-october-4-rainbow/" {
		t.Fatalf("image=%#v previous=%q err=%v", image, prev, err)
	}
}

func TestAPODPageRejectsLogoAndMissingDate(t *testing.T) {
	for _, tc := range []struct{ date, media string }{
		{"October 4, 2026", `<img src="/themes/nasa-logo.png">`},
		{"", `<img src="/images/rainbow.jpg">`},
	} {
		doc, _ := html.Parse(strings.NewReader(pageFixture(tc.date, "Rainbow", tc.media, "")))
		base, _ := url.Parse("https://science.nasa.gov/apod/")
		image, _, err := parseAPODPage(doc, base)
		if tc.date == "" && err == nil {
			t.Fatal("accepted missing date")
		}
		if image.MediaType == "image" {
			t.Fatal("accepted branding as image")
		}
	}
}

func TestAPODFallsBackFromBrandingAndWalksActualHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api":
			fmt.Fprint(w, `[{"date":"2026-10-04","title":"NASA Science","media_type":"image","url":"https://science.nasa.gov/themes/nasa-logo.png"}]`)
		case "/apod/":
			fmt.Fprint(w, pageFixture("October 4, 2026", "Rainbow", `<img src="/images/rainbow.jpg">`, "/image-article/apod-video/"))
		case "/image-article/apod-video/":
			fmt.Fprint(w, pageFixture("October 3, 2026", "Video", `<iframe src="https://example.test/movie"></iframe>`, "/image-article/apod-mars/"))
		case "/image-article/apod-mars/":
			fmt.Fprint(w, strings.ReplaceAll(strings.ReplaceAll(pageFixture("October 2, 2026", "APOD: 2026 October 2 – Mars", `<img src="/images/mars.jpg">`, ""), "h2>", "h1>"), "<th>Date</th>", "<th>Date:</th>"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := NewAPODClient("private-key")
	client.baseURL = server.URL + "/api"
	client.pageURL = server.URL + "/apod/"
	client.httpClient = server.Client()
	images, err := client.Recent(context.Background(), time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC), 2)
	if err != nil || len(images) != 2 || images[0].Date != "2026-10-04" || images[1].Date != "2026-10-02" || images[1].Title != "Mars" {
		t.Fatalf("images=%#v err=%v", images, err)
	}
	// Daily's malformed API response also falls back; it must not relabel today as yesterday.
	_, err = client.Daily(context.Background(), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("relabeled latest picture as requested date")
	}
}

func TestWallCacheRejectsAPODBranding(t *testing.T) {
	for _, tc := range []struct{ title, url string }{{"NASA Science", "https://example.test/photo.jpg"}, {"Rainbow", "https://science.nasa.gov/themes/nasa-logo.png"}} {
		wall := ImageWall{Recent: []ImageWindow{{Status: "ready", MediaType: "image", Title: tc.title, ImageURL: tc.url}}}
		if wallCacheable(wall) {
			t.Fatal("branding cache accepted")
		}
	}
}

// Opt-in smoke test against the current official DOM; deterministic fixtures above
// remain the regression checks used in ordinary test runs.
func TestAPODPageLive(t *testing.T) {
	if os.Getenv("NASA_PAGE_LIVE_TEST") != "1" {
		t.Skip("set NASA_PAGE_LIVE_TEST=1 to verify the official APOD body")
	}
	client := NewAPODClient("")
	images, err := client.pageImages(context.Background(), time.Now(), 5, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 5 {
		t.Fatalf("got %d publications, want 5", len(images))
	}
	dates := map[string]bool{}
	for _, image := range images {
		if dates[image.Date] || image.Copyright == "" || !imageURLLooksLikeImage(image.URL) || invalidAPODContent(image.Title, image.URL) {
			t.Fatalf("invalid publication: %#v", image)
		}
		dates[image.Date] = true
		t.Logf("%s | %s | %s", image.Date, image.Title, image.URL)
	}
}
