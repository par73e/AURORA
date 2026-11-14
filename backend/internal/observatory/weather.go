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
	Elevation   float64      `json:"elevation"` // Open-Meteo 返回的观测点海拔（米）
	Current     Current      `json:"current"`
	Hourly      []Hourly     `json:"hourly"`
	AirQuality  []AirQuality `json:"airQuality"`
	// Degraded 仅在无法刷新上游预报、改为提供进程内旧快照时设置；
	// 它不进入缓存，新鲜命中不会携带。
	Degraded *DegradedNotice `json:"degraded,omitempty"`
	// Selected 仅在请求携带 time 参数时返回：该时刻最近的逐小时预报快照。
	Selected *SelectedObservation `json:"selected,omitempty"`
	// Scores 仅在请求携带 scores=1 时返回：逐小时观测评分（供动态推荐）。
	Scores []HourScore `json:"scores,omitempty"`
}

// DegradedNotice 说明本次响应是上游不可用时的旧预报降级结果。
type DegradedNotice struct {
	Reason string `json:"reason"`
	Since  string `json:"since"` // 本次刷新失败的时刻（UTC RFC3339）
}

// SelectedObservation 是 conditions 接口的可选 time 参数产物。
type SelectedObservation struct {
	Hour                 Hourly `json:"hour"`
	WithinForecastWindow bool   `json:"withinForecastWindow"`
}

type Current struct {
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
	until    time.Time
	storedAt time.Time // 成功获取上游预报的时刻；降级宽限以它为界
	failedAt time.Time // 最近一次刷新失败的时刻；冷却期内不再重复等待上游超时
	value    Conditions
}

// GridCell 把观测坐标归一到 0.01° 网格。进程内天气缓存与响应缓存网关共用同一网格，
// 使设备定位的坐标抖动不会让两层缓存各自 miss。
func GridCell(latitude, longitude float64) string {
	return strconv.FormatFloat(GridCoordinate(latitude), 'f', 2, 64) + "," + strconv.FormatFloat(GridCoordinate(longitude), 'f', 2, 64)
}

// GridCoordinate 把单个坐标轴归一到 0.01° 网格值。
func GridCoordinate(value float64) float64 {
	return math.Round(value*100) / 100
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

// staleGrace 是上游不可用时允许继续提供旧预报的宽限期。预报按小时更新、前端以
// 90 分钟为过期阈值；宽限只在“刷新失败”时启用，响应携带 degraded 标记与 no-store，
// 响应缓存与浏览器都不会把降级快照当作新鲜数据继续传播。
const staleGrace = 6 * time.Hour

// failureCooldown 是上游失败后的冷却期：冷却内的请求直接返回降级旧预报，
// 不让每个访客都重新等待一遍上游超时。
const failureCooldown = 30 * time.Second

func (client *Client) Conditions(ctx context.Context, latitude, longitude float64) (Conditions, error) {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 || math.IsNaN(latitude) || math.IsNaN(longitude) {
		return Conditions{}, errors.New("observer coordinates are invalid")
	}
	key := GridCell(latitude, longitude)
	now := client.now()
	client.mu.Lock()
	cached, hasCached := client.cache[key]
	if hasCached && now.Before(cached.until) {
		client.mu.Unlock()
		return cached.value, nil
	}
	// 冷却期内上游大概率仍不可用：有宽限内的旧快照就直接降级，避免重复阻塞。
	if hasCached && !cached.storedAt.IsZero() && !cached.failedAt.IsZero() &&
		now.Sub(cached.failedAt) < failureCooldown && now.Sub(cached.storedAt) <= staleGrace {
		client.mu.Unlock()
		return client.degraded(cached.value, now), nil
	}
	client.mu.Unlock()

	weather, air, weatherErr := client.fetchUpstreams(ctx, latitude, longitude)
	if weatherErr != nil {
		client.mu.Lock()
		entry := cached
		if !hasCached {
			entry = cacheEntry{}
		}
		entry.failedAt = now
		entry.until = time.Time{}
		client.cache[key] = entry
		client.mu.Unlock()
		// stale-while-error：上游不可用时，若该网格在宽限期内仍有成功快照，
		// 提供旧预报并诚实标记降级；没有可用旧快照时保持失败，不伪造数据。
		if hasCached && !cached.storedAt.IsZero() && now.Sub(cached.storedAt) <= staleGrace {
			return client.degraded(cached.value, now), nil
		}
		return Conditions{}, weatherErr
	}
	// Air quality improves the observing explanation but must not make a weather
	// forecast unavailable if that independent public endpoint is slow or down.
	if air != nil {
		weather.AirQuality = air
	}
	now = client.now()
	weather.RetrievedAt = now.UTC().Format(time.RFC3339)
	weather.Source = "Open-Meteo forecast"

	client.mu.Lock()
	// 天气数据源按小时更新；缓存节奏对齐为 1 小时，避免把旧预报当新预报提供。
	client.cache[key] = cacheEntry{until: now.Add(time.Hour), storedAt: now, value: weather}
	client.mu.Unlock()
	return weather, nil
}

// degraded 复制旧快照并附加降级标记；storedAt/retrievedAt 保持原义，
// 调用方与前端仍能据 retrievedAt 判断资料时效。
func (client *Client) degraded(stale Conditions, now time.Time) Conditions {
	stale.Degraded = &DegradedNotice{Reason: "upstream_unavailable", Since: now.UTC().Format(time.RFC3339)}
	return stale
}

// fetchUpstreams 并发请求天气与空气质量：空气质量失败不影响天气预报，
// 并发也使最坏耗时不再叠加为两路超时之和。
func (client *Client) fetchUpstreams(ctx context.Context, latitude, longitude float64) (Conditions, []AirQuality, error) {
	type weatherResult struct {
		value Conditions
		err   error
	}
	type airResult struct {
		value []AirQuality
		err   error
	}
	weatherOut := make(chan weatherResult, 1)
	airOut := make(chan airResult, 1)
	go func() {
		value, err := client.weather(ctx, latitude, longitude)
		weatherOut <- weatherResult{value: value, err: err}
	}()
	go func() {
		value, err := client.airQuality(ctx, latitude, longitude)
		airOut <- airResult{value: value, err: err}
	}()
	weather := <-weatherOut
	air := <-airOut
	return weather.value, air.value, weather.err
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
	// 按观测点当地自然日读取今天与明天。前端从当地当前整点展示到明日 24:00，
	// 同时保留当前天气、今夜跨午夜评分和完整的明日夜间评分数据。
	query.Set("forecast_days", "2")
	query.Set("current", "temperature_2m,dew_point_2m,cloud_cover,cloud_cover_low,cloud_cover_mid,cloud_cover_high,visibility,relative_humidity_2m,precipitation,wind_speed_10m,wind_gusts_10m,weather_code")
	query.Set("hourly", "temperature_2m,dew_point_2m,cloud_cover,cloud_cover_low,cloud_cover_mid,cloud_cover_high,visibility,relative_humidity_2m,precipitation,wind_speed_10m,wind_direction_10m,wind_gusts_10m,pressure_msl,weather_code")
	endpoint.RawQuery = query.Encode()

	var payload struct {
		Timezone  string  `json:"timezone"`
		Elevation float64 `json:"elevation"`
		Current   struct {
			Time           string  `json:"time"`
			Temperature    float64 `json:"temperature_2m"`
			DewPoint       float64 `json:"dew_point_2m"`
			CloudCover     float64 `json:"cloud_cover"`
			CloudCoverLow  float64 `json:"cloud_cover_low"`
			CloudCoverMid  float64 `json:"cloud_cover_mid"`
			CloudCoverHigh float64 `json:"cloud_cover_high"`
			Visibility     float64 `json:"visibility"`
			Humidity       float64 `json:"relative_humidity_2m"`
			Precipitation  float64 `json:"precipitation"`
			WindSpeed      float64 `json:"wind_speed_10m"`
			WindGusts      float64 `json:"wind_gusts_10m"`
			WeatherCode    int     `json:"weather_code"`
		} `json:"current"`
		Hourly struct {
			Time           []string  `json:"time"`
			Temperature    []float64 `json:"temperature_2m"`
			DewPoint       []float64 `json:"dew_point_2m"`
			CloudCover     []float64 `json:"cloud_cover"`
			CloudCoverLow  []float64 `json:"cloud_cover_low"`
			CloudCoverMid  []float64 `json:"cloud_cover_mid"`
			CloudCoverHigh []float64 `json:"cloud_cover_high"`
			Visibility     []float64 `json:"visibility"`
			Humidity       []float64 `json:"relative_humidity_2m"`
			Precipitation  []float64 `json:"precipitation"`
			WindSpeed      []float64 `json:"wind_speed_10m"`
			WindDirection  []float64 `json:"wind_direction_10m"`
			WindGusts      []float64 `json:"wind_gusts_10m"`
			Pressure       []float64 `json:"pressure_msl"`
			WeatherCode    []int     `json:"weather_code"`
		} `json:"hourly"`
	}
	var raw json.RawMessage
	if err := client.getJSON(ctx, endpoint.String(), &raw); err != nil {
		return Conditions{}, err
	}
	if err := validateWeatherPayload(raw); err != nil {
		return Conditions{}, err
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Conditions{}, fmt.Errorf("decode observing forecast: %w", err)
	}
	result := Conditions{Timezone: payload.Timezone, Elevation: payload.Elevation, Current: Current{
		Time: payload.Current.Time, Temperature: payload.Current.Temperature, DewPoint: payload.Current.DewPoint, CloudCover: payload.Current.CloudCover,
		CloudCoverLow: payload.Current.CloudCoverLow, CloudCoverMid: payload.Current.CloudCoverMid, CloudCoverHigh: payload.Current.CloudCoverHigh, VisibilityMeters: payload.Current.Visibility,
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

// A missing or null weather factor must never become Go's numeric zero and
// produce an apparently excellent observing score.
func validateWeatherPayload(raw json.RawMessage) error {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}
	var current, hourly map[string]json.RawMessage
	if err := json.Unmarshal(root["current"], &current); err != nil {
		return fmt.Errorf("observing forecast current data is missing: %w", err)
	}
	if err := json.Unmarshal(root["hourly"], &hourly); err != nil {
		return fmt.Errorf("observing forecast hourly data is missing: %w", err)
	}
	var currentTime string
	if err := json.Unmarshal(current["time"], &currentTime); err != nil || currentTime == "" {
		return errors.New("observing forecast current time is missing")
	}
	if _, err := time.Parse("2006-01-02T15:04", currentTime); err != nil {
		return errors.New("observing forecast current time is invalid")
	}
	var times []string
	if err := json.Unmarshal(hourly["time"], &times); err != nil || len(times) == 0 {
		return errors.New("observing forecast hourly times are missing")
	}
	for _, at := range times {
		if _, err := time.Parse("2006-01-02T15:04", at); err != nil {
			return errors.New("observing forecast hourly time is invalid")
		}
	}
	fields := []string{"temperature_2m", "dew_point_2m", "cloud_cover", "cloud_cover_low", "cloud_cover_mid", "cloud_cover_high", "visibility", "relative_humidity_2m", "precipitation", "wind_speed_10m", "wind_gusts_10m", "weather_code"}
	for _, name := range fields {
		if len(current[name]) == 0 || string(current[name]) == "null" {
			return fmt.Errorf("observing forecast current %s is missing", name)
		}
		var number float64
		if err := json.Unmarshal(current[name], &number); err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return fmt.Errorf("observing forecast current %s is invalid", name)
		}
		var values []json.RawMessage
		if err := json.Unmarshal(hourly[name], &values); err != nil || len(values) != len(times) {
			return fmt.Errorf("observing forecast hourly %s is incomplete", name)
		}
		for _, value := range values {
			if len(value) == 0 || string(value) == "null" {
				return fmt.Errorf("observing forecast hourly %s contains missing values", name)
			}
			if err := json.Unmarshal(value, &number); err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
				return fmt.Errorf("observing forecast hourly %s contains invalid values", name)
			}
		}
	}
	return nil
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
	query.Set("forecast_days", "2")
	query.Set("hourly", "pm2_5,pm10,aerosol_optical_depth")
	endpoint.RawQuery = query.Encode()
	var payload struct {
		Hourly struct {
			Time []string  `json:"time"`
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
