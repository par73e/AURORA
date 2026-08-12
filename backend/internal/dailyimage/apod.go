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

type APODClient struct {
	key        string
	baseURL    string
	httpClient *http.Client
}

func NewAPODClient(key string) *APODClient {
	return &APODClient{key: strings.TrimSpace(key), baseURL: apodURL, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (client *APODClient) Daily(ctx context.Context, date time.Time) (Image, error) {
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
	var payload struct {
		Date         string `json:"date"`
		Title        string `json:"title"`
		Explanation  string `json:"explanation"`
		MediaType    string `json:"media_type"`
		URL          string `json:"url"`
		ThumbnailURL string `json:"thumbnail_url"`
		HDURL        string `json:"hdurl"`
		Copyright    string `json:"copyright"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return Image{}, errors.New("decode daily image response")
	}
	if payload.Date == "" || payload.Title == "" || payload.URL == "" || (payload.MediaType != "image" && payload.MediaType != "video") {
		return Image{}, errors.New("daily image response is incomplete")
	}
	return Image{Date: payload.Date, Title: payload.Title, Explanation: payload.Explanation, MediaType: payload.MediaType, URL: payload.URL, ThumbnailURL: payload.ThumbnailURL, HDURL: payload.HDURL, Copyright: payload.Copyright, SourceName: "NASA Astronomy Picture of the Day", SourceURL: "https://apod.nasa.gov/apod/astropix.html"}, nil
}
