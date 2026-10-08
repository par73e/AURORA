// Package dailyimage retrieves the official NASA Astronomy Picture of the Day.
package dailyimage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const apodURL = "https://api.nasa.gov/planetary/apod"

var ErrNotConfigured = errors.New("daily image service is not configured")

// Image contains only the public APOD fields AURORA renders. The API key never
// crosses this boundary into an HTTP response or browser bundle.
type Image struct {
	Date         string `json:"date"`
	Title        string `json:"title"`
	Explanation  string `json:"explanation"`
	MediaType    string `json:"mediaType"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
	HDURL        string `json:"hdUrl,omitempty"`
	Copyright    string `json:"copyright,omitempty"`
	SourceName   string `json:"sourceName"`
	SourceURL    string `json:"sourceUrl"`
}

type Provider interface {
	Daily(context.Context, time.Time) (Image, error)
}

// RecentProvider is implemented by sources that can return a short image-only
// release history in one request. APOD may publish video entries, but AURORA's
// image wall deliberately skips them instead of rendering video as a picture.
type RecentProvider interface {
	Recent(context.Context, time.Time, int) ([]Image, error)
}

type APODClient struct {
	key        string
	baseURL    string
	httpClient *http.Client
	pageURL    string
}

func NewAPODClient(key string) *APODClient {
	return &APODClient{key: strings.TrimSpace(key), baseURL: apodURL, pageURL: "https://science.nasa.gov/apod/", httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (client *APODClient) Daily(ctx context.Context, date time.Time) (Image, error) {
	image, err := client.dailyAPI(ctx, date)
	if err == nil || errors.Is(err, ErrNotConfigured) {
		return image, err
	}
	images, pageErr := client.pageImages(ctx, date, 1, true)
	if pageErr != nil {
		return Image{}, fmt.Errorf("APOD API unavailable (%v); official page: %w", err, pageErr)
	}
	return images[0], nil
}

func (client *APODClient) dailyAPI(ctx context.Context, date time.Time) (Image, error) {
	if client.key == "" {
		return Image{}, ErrNotConfigured
	}
	endpoint, err := url.Parse(client.baseURL)
	if err != nil {
		return Image{}, errors.New("prepare daily image request")
	}
	query := endpoint.Query()
	query.Set("api_key", client.key)
	query.Set("date", date.UTC().Format(time.DateOnly))
	query.Set("thumbs", "true")
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return Image{}, errors.New("create daily image request")
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return Image{}, fmt.Errorf("request daily image: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Image{}, fmt.Errorf("daily image service returned %d", response.StatusCode)
	}
	var payload apodPayload
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return Image{}, errors.New("decode daily image response")
	}
	return payload.image()
}

func (client *APODClient) Recent(ctx context.Context, end time.Time, count int) ([]Image, error) {
	images, err := client.recentAPI(ctx, end, count)
	if err == nil || errors.Is(err, ErrNotConfigured) {
		return images, err
	}
	images, pageErr := client.pageImages(ctx, end, count, false)
	if pageErr != nil {
		return nil, fmt.Errorf("APOD API unavailable (%v); official page: %w", err, pageErr)
	}
	return images, nil
}

func (client *APODClient) recentAPI(ctx context.Context, end time.Time, count int) ([]Image, error) {
	if client.key == "" {
		return nil, ErrNotConfigured
	}
	if count < 1 {
		return []Image{}, nil
	}
	endpoint, err := url.Parse(client.baseURL)
	if err != nil {
		return nil, errors.New("prepare recent APOD request")
	}
	query := endpoint.Query()
	query.Set("api_key", client.key)
	query.Set("start_date", end.UTC().AddDate(0, 0, -9).Format(time.DateOnly))
	query.Set("end_date", end.UTC().Format(time.DateOnly))
	query.Set("thumbs", "true")
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, errors.New("create recent APOD request")
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request recent APOD: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("recent APOD service returned %d", response.StatusCode)
	}
	var payloads []apodPayload
	if err := json.NewDecoder(response.Body).Decode(&payloads); err != nil {
		return nil, errors.New("decode recent APOD response")
	}
	// Reject a corrupted batch in full rather than mixing site chrome with APOD.
	for _, payload := range payloads {
		if invalidAPODContent(payload.Title, payload.URL) {
			return nil, errors.New("recent APOD response contains site branding")
		}
	}
	images := make([]Image, 0, count)
	for index := len(payloads) - 1; index >= 0 && len(images) < count; index-- {
		image, err := payloads[index].image()
		if err == nil && image.MediaType == "image" {
			images = append(images, image)
		}
	}
	if len(images) == 0 {
		return nil, errors.New("recent APOD response is incomplete")
	}
	return images, nil
}

type apodPayload struct {
	Date         string `json:"date"`
	Title        string `json:"title"`
	Explanation  string `json:"explanation"`
	MediaType    string `json:"media_type"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
	HDURL        string `json:"hdurl"`
	Copyright    string `json:"copyright"`
}

func (payload apodPayload) image() (Image, error) {
	if invalidAPODContent(payload.Title, payload.URL) {
		return Image{}, errors.New("APOD response contains site branding")
	}
	if payload.Date == "" || payload.Title == "" || payload.URL == "" || (payload.MediaType != "image" && payload.MediaType != "video") {
		return Image{}, errors.New("daily image response is incomplete")
	}
	publishedAt, err := time.Parse(time.DateOnly, payload.Date)
	if err != nil {
		return Image{}, errors.New("daily image date is invalid")
	}
	sourceURL := "https://apod.nasa.gov/apod/ap" + publishedAt.Format("060102") + ".html"
	return Image{Date: payload.Date, Title: payload.Title, Explanation: payload.Explanation, MediaType: payload.MediaType, URL: payload.URL, ThumbnailURL: payload.ThumbnailURL, HDURL: payload.HDURL, Copyright: payload.Copyright, SourceName: "NASA Astronomy Picture of the Day", SourceURL: sourceURL}, nil
}
