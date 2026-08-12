package dailyimage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	items := make([]ImageWindow, 0, len(payload.Collection.Items))
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
		items = append(items, ImageWindow{ID: "nasa-library", SourceID: "nasa-library", SourceName: "NASA Image and Video Library", Title: data.Title, PublishedAt: dateOnly(data.DateCreated), ImageURL: thumbnail, ThumbnailURL: thumbnail, MediaType: "image", Credit: credit, LicenseNote: "请以 NASA 条目中的版权与使用说明为准", SourceURL: "https://images.nasa.gov/details/" + url.PathEscape(data.NASAID), SelectionMode: "rotating", Summary: data.Description, Status: "ready"})
	}
	if len(items) == 0 {
		return ImageWindow{}, errors.New("NASA image library returned no usable image")
	}
	return items[at.UTC().YearDay()%len(items)], nil
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
