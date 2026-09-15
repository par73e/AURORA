package dailyimage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
)

const nasaImageLibraryURL = "https://images-api.nasa.gov/search"

type cachedLibraryWindow struct {
	window    ImageWindow
	expiresAt time.Time
}

// NASAImageLibraryClient uses NASA's public image-search API. It requires no
// key and selects a deterministic image from a small astronomy topic rotation.
type NASAImageLibraryClient struct {
	baseURL    string
	httpClient *http.Client

	mu          sync.Mutex
	cache       map[string]cachedLibraryWindow
	lastSuccess *ImageWindow
}

func NewNASAImageLibraryClient() *NASAImageLibraryClient {
	return &NASAImageLibraryClient{baseURL: nasaImageLibraryURL, httpClient: &http.Client{Timeout: 10 * time.Second}, cache: make(map[string]cachedLibraryWindow)}
}

func (client *NASAImageLibraryClient) Pick(ctx context.Context, at time.Time) (ImageWindow, error) {
	key := at.UTC().Format(time.DateOnly)
	client.mu.Lock()
	if cached, ok := client.cache[key]; ok && time.Now().Before(cached.expiresAt) {
		client.mu.Unlock()
		return cached.window, nil
	}
	client.mu.Unlock()

	window, err := client.fetch(ctx, at)
	if err != nil {
		client.mu.Lock()
		defer client.mu.Unlock()
		if client.lastSuccess != nil {
			fallback := *client.lastSuccess
			fallback.IsFallback = true
			return fallback, nil
		}
		return ImageWindow{}, err
	}
	client.mu.Lock()
	client.cache[key] = cachedLibraryWindow{window: window, expiresAt: time.Now().Add(30 * time.Minute)}
	client.lastSuccess = &window
	client.mu.Unlock()
	return window, nil
}

func (client *NASAImageLibraryClient) fetch(ctx context.Context, at time.Time) (ImageWindow, error) {
	endpoint, err := url.Parse(client.baseURL)
	if err != nil {
		return ImageWindow{}, errors.New("prepare NASA image library request")
	}
	topics := []string{"nebula", "galaxy", "Jupiter", "Mars", "Moon", "supernova", "telescope"}
	topic := topics[at.UTC().YearDay()%len(topics)]
	query := endpoint.Query()
	query.Set("q", topic)
	query.Set("media_type", "image")
	query.Set("page", "1")
	query.Set("page_size", "50")
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return ImageWindow{}, errors.New("create NASA image library request")
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return ImageWindow{}, fmt.Errorf("request NASA image library: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ImageWindow{}, fmt.Errorf("NASA image library returned %d", response.StatusCode)
	}
	var payload struct {
		Collection struct {
			Items []struct {
				Data []struct {
					NASAID      string `json:"nasa_id"`
					Title       string `json:"title"`
					Description string `json:"description"`
					DateCreated string `json:"date_created"`
					Center      string `json:"center"`
				} `json:"data"`
				Links []struct {
					Href string `json:"href"`
				} `json:"links"`
			} `json:"items"`
		} `json:"collection"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return ImageWindow{}, errors.New("decode NASA image library response")
	}
	type candidate struct {
		nasaID string
		window ImageWindow
	}
	items := make([]candidate, 0, len(payload.Collection.Items))
	for _, item := range payload.Collection.Items {
		if len(item.Data) == 0 || len(item.Links) == 0 {
			continue
		}
		data, thumbnail := item.Data[0], item.Links[0].Href
		if data.NASAID == "" || data.Title == "" || thumbnail == "" {
			continue
		}
		credit := "NASA"
		if strings.TrimSpace(data.Center) != "" {
			credit = "NASA / " + strings.TrimSpace(data.Center)
		}
		items = append(items, candidate{nasaID: data.NASAID, window: ImageWindow{ID: "nasa-library", SourceID: "nasa-library", SourceName: "NASA Image and Video Library", Title: data.Title, PublishedAt: dateOnly(data.DateCreated), ImageURL: thumbnail, ThumbnailURL: thumbnail, MediaType: "image", Credit: credit, LicenseNote: "版权与使用条件见原始链接", SourceURL: "https://images.nasa.gov/details/" + url.PathEscape(data.NASAID), SelectionMode: "rotating", Summary: data.Description, Status: "ready"}})
	}
	if len(items) == 0 {
		return ImageWindow{}, errors.New("NASA image library returned no usable image")
	}
	selected := items[at.UTC().YearDay()%len(items)]
	selected.window.HDURL = selected.window.ImageURL
	if originalURL, err := client.fetchOriginalAsset(ctx, selected.nasaID); err == nil {
		selected.window.HDURL = originalURL
	}
	return selected.window, nil
}

func (client *NASAImageLibraryClient) fetchOriginalAsset(ctx context.Context, nasaID string) (string, error) {
	endpoint, err := url.Parse(client.baseURL)
	if err != nil {
		return "", errors.New("prepare NASA image asset request")
	}
	endpoint.RawQuery = ""
	endpoint.Path = path.Join(path.Dir(endpoint.Path), "asset", url.PathEscape(nasaID))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", errors.New("create NASA image asset request")
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("request NASA image asset: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("NASA image asset returned %d", response.StatusCode)
	}
	var payload struct {
		Collection struct {
			Items []struct {
				Href string `json:"href"`
			} `json:"items"`
		} `json:"collection"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", errors.New("decode NASA image asset response")
	}
	for _, item := range payload.Collection.Items {
		href := strings.Replace(item.Href, "http://images-assets.nasa.gov/", "https://images-assets.nasa.gov/", 1)
		if strings.Contains(strings.ToLower(href), "~orig.") && imageAssetURL(href) {
			return href, nil
		}
	}
	return "", errors.New("NASA image asset response has no original image")
}

func imageAssetURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	switch strings.ToLower(path.Ext(parsed.Path)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".tif", ".tiff":
		return true
	default:
		return false
	}
}

func dateOnly(raw string) string {
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.UTC().Format(time.DateOnly)
	}
	if len(raw) >= len(time.DateOnly) {
		return raw[:len(time.DateOnly)]
	}
	return raw
}
