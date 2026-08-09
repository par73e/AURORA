// Package observatory provides short-lived, location-specific observing data.
package observatory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	weatherBaseURL = "https://api.open-meteo.com/v1/forecast"
	airBaseURL     = "https://air-quality-api.open-meteo.com/v1/air-quality"
)

// Conditions is the deliberately small public contract needed by the SKY client.
// Values are forecasts, not sensor readings. They are kept out of the database because
// the source updates frequently and historical observations are not a v1 feature.
type Conditions struct {
	RetrievedAt string       `json:"retrievedAt"`
	Timezone    string       `json:"timezone"`
	Source      string       `json:"source"`
	Current     Current      `json:"current"`
	Hourly      []Hourly     `json:"hourly"`
	AirQuality  []AirQuality `json:"airQuality"`
}

type Current struct {
	Time             string  `json:"time"`
	Temperature      float64 `json:"temperature"`
	DewPoint         float64 `json:"dewPoint"`
	CloudCover       float64 `json:"cloudCover"`
	VisibilityMeters float64 `json:"visibilityMeters"`
	Humidity         float64 `json:"humidity"`
	Precipitation    float64 `json:"precipitation"`
	WindSpeed        float64 `json:"windSpeed"`
	WindGusts        float64 `json:"windGusts"`
	WeatherCode      int     `json:"weatherCode"`
}

type Hourly struct {
	Time             string  `json:"time"`
	Temperature      float64 `json:"temperature"`
	DewPoint         float64 `json:"dewPoint"`
	CloudCover       float64 `json:"cloudCover"`
	CloudCoverLow    float64 `json:"cloudCoverLow"`
	CloudCoverMid    float64 `json:"cloudCoverMid"`
	CloudCoverHigh   float64 `json:"cloudCoverHigh"`
	VisibilityMeters float64 `json:"visibilityMeters"`
	Humidity         float64 `json:"humidity"`
	Precipitation    float64 `json:"precipitation"`
	WindSpeed        float64 `json:"windSpeed"`
	WindDirection    float64 `json:"windDirection"`
	WindGusts        float64 `json:"windGusts"`
	Pressure         float64 `json:"pressure"`
	WeatherCode      int     `json:"weatherCode"`
}

type AirQuality struct {
	Time                string  `json:"time"`
	PM25                float64 `json:"pm25"`
	PM10                float64 `json:"pm10"`
	AerosolOpticalDepth float64 `json:"aerosolOpticalDepth"`
}

type ConditionsProvider interface {
	Conditions(context.Context, float64, float64) (Conditions, error)
}

type Client struct {
	weatherURL string
	airURL     string
	httpClient *http.Client
	now        func() time.Time

	mu    sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	until time.Time
	value Conditions
}

func NewClient() *Client {
	return NewClientWithURLs(weatherBaseURL, airBaseURL, &http.Client{Timeout: 8 * time.Second})
}

func NewClientWithURLs(weatherURL, airURL string, httpClient *http.Client) *Client {
	return &Client{
		weatherURL: weatherURL,
		airURL:     airURL,
		httpClient: httpClient,
		now:        time.Now,
		cache:      make(map[string]cacheEntry),
	}
}

func (client *Client) Conditions(ctx context.Context, latitude, longitude float64) (Conditions, error) {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 || math.IsNaN(latitude) || math.IsNaN(longitude) {
		return Conditions{}, errors.New("observer coordinates are invalid")
	}
	key := strconv.FormatFloat(math.Round(latitude*100)/100, 'f', 2, 64) + "," + strconv.FormatFloat(math.Round(longitude*100)/100, 'f', 2, 64)
	client.mu.Lock()
	if cached, ok := client.cache[key]; ok && client.now().Before(cached.until) {
		client.mu.Unlock()
		return cached.value, nil
	}
	client.mu.Unlock()

	weather, err := client.weather(ctx, latitude, longitude)
	if err != nil {
		return Conditions{}, err
	}
	air, err := client.airQuality(ctx, latitude, longitude)
	// Air quality improves the observing explanation but must not make a weather
	// forecast unavailable if that independent public endpoint is slow or down.
	if err == nil {
		weather.AirQuality = air
	}
	weather.RetrievedAt = client.now().UTC().Format(time.RFC3339)
	weather.Source = "Open-Meteo forecast"

	client.mu.Lock()
	client.cache[key] = cacheEntry{until: client.now().Add(15 * time.Minute), value: weather}
	client.mu.Unlock()
	return weather, nil
}

func (client *Client) weather(ctx context.Context, latitude, longitude float64) (Conditions, error) {
	endpoint, err := url.Parse(client.weatherURL)
	if err != nil {
		return Conditions{}, fmt.Errorf("parse weather endpoint: %w", err)
	}
	query := endpoint.Query()
	query.Set("latitude", strconv.FormatFloat(latitude, 'f', 6, 64))
	query.Set("longitude", strconv.FormatFloat(longitude, 'f', 6, 64))
	query.Set("timezone", "auto")
	query.Set("forecast_hours", "24")
	query.Set("current", "temperature_2m,dew_point_2m,cloud_cover,visibility,relative_humidity_2m,precipitation,wind_speed_10m,wind_gusts_10m,weather_code")
	query.Set("hourly", "temperature_2m,dew_point_2m,cloud_cover,cloud_cover_low,cloud_cover_mid,cloud_cover_high,visibility,relative_humidity_2m,precipitation,wind_speed_10m,wind_direction_10m,wind_gusts_10m,pressure_msl,weather_code")
	endpoint.RawQuery = query.Encode()

	var payload struct {
		Timezone string `json:"timezone"`
		Current  struct {
			Time               string  `json:"time"`
			Temperature        float64 `json:"temperature_2m"`
			DewPoint           float64 `json:"dew_point_2m"`
			CloudCover         float64 `json:"cloud_cover"`
			Visibility         float64 `json:"visibility"`
			Humidity           float64 `json:"relative_humidity_2m"`
			Precipitation      float64 `json:"precipitation"`
			WindSpeed          float64 `json:"wind_speed_10m"`
			WindGusts          float64 `json:"wind_gusts_10m"`
			WeatherCode        int     `json:"weather_code"`
		} `json:"current"`
		Hourly struct {
			Time               []string  `json:"time"`
			Temperature        []float64 `json:"temperature_2m"`
			DewPoint           []float64 `json:"dew_point_2m"`
			CloudCover         []float64 `json:"cloud_cover"`
			CloudCoverLow      []float64 `json:"cloud_cover_low"`
			CloudCoverMid      []float64 `json:"cloud_cover_mid"`
			CloudCoverHigh     []float64 `json:"cloud_cover_high"`
			Visibility         []float64 `json:"visibility"`
			Humidity           []float64 `json:"relative_humidity_2m"`
			Precipitation      []float64 `json:"precipitation"`
			WindSpeed          []float64 `json:"wind_speed_10m"`
			WindDirection      []float64 `json:"wind_direction_10m"`
			WindGusts          []float64 `json:"wind_gusts_10m"`
			Pressure           []float64 `json:"pressure_msl"`
			WeatherCode        []int     `json:"weather_code"`
		} `json:"hourly"`
	}
	if err := client.getJSON(ctx, endpoint.String(), &payload); err != nil {
		return Conditions{}, err
	}
	result := Conditions{Timezone: payload.Timezone, Current: Current{
		Time: payload.Current.Time, Temperature: payload.Current.Temperature, DewPoint: payload.Current.DewPoint, CloudCover: payload.Current.CloudCover, VisibilityMeters: payload.Current.Visibility,
		Humidity: payload.Current.Humidity, Precipitation: payload.Current.Precipitation, WindSpeed: payload.Current.WindSpeed,
		WindGusts: payload.Current.WindGusts, WeatherCode: payload.Current.WeatherCode,
	}}
	for index, at := range payload.Hourly.Time {
		result.Hourly = append(result.Hourly, Hourly{Time: at,
			Temperature: valueAt(payload.Hourly.Temperature, index), DewPoint: valueAt(payload.Hourly.DewPoint, index),
			CloudCover: valueAt(payload.Hourly.CloudCover, index), CloudCoverLow: valueAt(payload.Hourly.CloudCoverLow, index),
			CloudCoverMid: valueAt(payload.Hourly.CloudCoverMid, index), CloudCoverHigh: valueAt(payload.Hourly.CloudCoverHigh, index),
			VisibilityMeters: valueAt(payload.Hourly.Visibility, index), Humidity: valueAt(payload.Hourly.Humidity, index),
			Precipitation: valueAt(payload.Hourly.Precipitation, index), WindSpeed: valueAt(payload.Hourly.WindSpeed, index),
			WindDirection: valueAt(payload.Hourly.WindDirection, index), WindGusts: valueAt(payload.Hourly.WindGusts, index),
			Pressure: valueAt(payload.Hourly.Pressure, index), WeatherCode: intAt(payload.Hourly.WeatherCode, index),
		})
	}
	return result, nil
}

func (client *Client) airQuality(ctx context.Context, latitude, longitude float64) ([]AirQuality, error) {
	endpoint, err := url.Parse(client.airURL)
	if err != nil {
		return nil, err
	}
	query := endpoint.Query()
	query.Set("latitude", strconv.FormatFloat(latitude, 'f', 6, 64))
	query.Set("longitude", strconv.FormatFloat(longitude, 'f', 6, 64))
	query.Set("timezone", "auto")
	query.Set("forecast_hours", "24")
	query.Set("hourly", "pm2_5,pm10,aerosol_optical_depth")
	endpoint.RawQuery = query.Encode()
	var payload struct {
		Hourly struct {
			Time []string `json:"time"`
			PM25 []float64 `json:"pm2_5"`
			PM10 []float64 `json:"pm10"`
			AOD  []float64 `json:"aerosol_optical_depth"`
		} `json:"hourly"`
	}
	if err := client.getJSON(ctx, endpoint.String(), &payload); err != nil {
		return nil, err
	}
	result := make([]AirQuality, 0, len(payload.Hourly.Time))
	for index, at := range payload.Hourly.Time {
		result = append(result, AirQuality{Time: at, PM25: valueAt(payload.Hourly.PM25, index), PM10: valueAt(payload.Hourly.PM10, index), AerosolOpticalDepth: valueAt(payload.Hourly.AOD, index)})
	}
	return result, nil
}

func (client *Client) getJSON(ctx context.Context, endpoint string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("request observing forecast failed")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("observing forecast returned HTTP %d", response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(target); err != nil {
		return fmt.Errorf("decode observing forecast: %w", err)
	}
	return nil
}

func valueAt(values []float64, index int) float64 {
	if index >= len(values) {
		return 0
	}
	return values[index]
}

func intAt(values []int, index int) int {
	if index >= len(values) {
		return 0
	}
	return values[index]
}
