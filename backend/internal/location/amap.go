package location

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const amapBaseURL = "https://restapi.amap.com"

var ErrNotConfigured = errors.New("location service is not configured")

// Place 是前端真正需要的行政区划信息，不暴露高德响应中的无关字段。
type Place struct {
	Label    string `json:"label"`
	Province string `json:"province"`
	City     string `json:"city"`
	District string `json:"district"`
	Adcode   string `json:"adcode"`
}

type ReverseGeocoder interface {
	Reverse(context.Context, float64, float64) (Place, error)
}

type AMapClient struct {
	key        string
	baseURL    string
	httpClient *http.Client
}

func NewAMapClient(key string) *AMapClient {
	return &AMapClient{
		key:        strings.TrimSpace(key),
		baseURL:    amapBaseURL,
		httpClient: &http.Client{Timeout: 8 * time.Second},
	}
}

func (client *AMapClient) Reverse(ctx context.Context, latitude, longitude float64) (Place, error) {
	if client.key == "" {
		return Place{}, ErrNotConfigured
	}

	convertedLongitude, convertedLatitude, err := client.convertGPS(ctx, longitude, latitude)
	if err != nil {
		return Place{}, err
	}
	return client.reverseAMapCoordinate(ctx, convertedLongitude, convertedLatitude)
}

func (client *AMapClient) convertGPS(ctx context.Context, longitude, latitude float64) (float64, float64, error) {
	query := url.Values{
		"key":       {client.key},
		"locations": {formatCoordinate(longitude) + "," + formatCoordinate(latitude)},
		"coordsys":  {"gps"},
		"output":    {"JSON"},
	}
	var response struct {
		Status    string `json:"status"`
		Info      string `json:"info"`
		InfoCode  string `json:"infocode"`
		Locations string `json:"locations"`
	}
	if err := client.getJSON(ctx, "/v3/assistant/coordinate/convert", query, &response); err != nil {
		return 0, 0, err
	}
	if response.Status != "1" {
		return 0, 0, fmt.Errorf("amap coordinate conversion failed: %s (%s)", response.Info, response.InfoCode)
	}

	parts := strings.Split(response.Locations, ",")
	if len(parts) != 2 {
		return 0, 0, errors.New("amap coordinate conversion returned an invalid location")
	}
	convertedLongitude, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse converted longitude: %w", err)
	}
	convertedLatitude, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse converted latitude: %w", err)
	}
	return convertedLongitude, convertedLatitude, nil
}

func (client *AMapClient) reverseAMapCoordinate(ctx context.Context, longitude, latitude float64) (Place, error) {
	query := url.Values{
		"key":        {client.key},
		"location":   {formatCoordinate(longitude) + "," + formatCoordinate(latitude)},
		"extensions": {"base"},
		"output":     {"JSON"},
	}
	var response struct {
		Status   string `json:"status"`
		Info     string `json:"info"`
		InfoCode string `json:"infocode"`
		Regeo    struct {
			AddressComponent struct {
				Province amapText `json:"province"`
				City     amapText `json:"city"`
				District amapText `json:"district"`
				Adcode   amapText `json:"adcode"`
			} `json:"addressComponent"`
		} `json:"regeocode"`
	}
	if err := client.getJSON(ctx, "/v3/geocode/regeo", query, &response); err != nil {
		return Place{}, err
	}
	if response.Status != "1" {
		return Place{}, fmt.Errorf("amap reverse geocoding failed: %s (%s)", response.Info, response.InfoCode)
	}

	province := string(response.Regeo.AddressComponent.Province)
	city := string(response.Regeo.AddressComponent.City)
	district := string(response.Regeo.AddressComponent.District)
	// 高德对直辖市可能返回空 city；此时省级名称就是用户熟悉的城市名。
	if city == "" {
		city = province
	}
	label := placeLabel(city, district, province)
	if label == "" {
		return Place{}, errors.New("amap reverse geocoding returned no administrative area")
	}

	return Place{
		Label:    label,
		Province: province,
		City:     city,
		District: district,
		Adcode:   string(response.Regeo.AddressComponent.Adcode),
	}, nil
}

func (client *AMapClient) getJSON(ctx context.Context, path string, query url.Values, target any) error {
	endpoint, err := url.Parse(client.baseURL)
	if err != nil {
		return fmt.Errorf("parse amap base URL: %w", err)
	}
	endpoint.Path = path
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("create amap request: %w", err)
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		// url.Error 会包含完整请求 URL；高德 Key 位于查询参数中，不能进入日志链路。
		if ctx.Err() != nil {
			return fmt.Errorf("request amap: %w", ctx.Err())
		}
		return errors.New("request amap failed")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("amap returned HTTP %d", response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(target); err != nil {
		return fmt.Errorf("decode amap response: %w", err)
	}
	return nil
}

func formatCoordinate(value float64) string {
	return strconv.FormatFloat(value, 'f', 6, 64)
}

func placeLabel(city, district, province string) string {
	parts := make([]string, 0, 2)
	for _, value := range []string{city, district} {
		value = strings.TrimSpace(value)
		if value != "" && (len(parts) == 0 || parts[len(parts)-1] != value) {
			parts = append(parts, value)
		}
	}
	if len(parts) == 0 && strings.TrimSpace(province) != "" {
		parts = append(parts, strings.TrimSpace(province))
	}
	return strings.Join(parts, " · ")
}

// 高德的 city 等字段在无值时可能返回 []，有值时则返回字符串。
type amapText string

func (value *amapText) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*value = amapText(text)
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err == nil {
		*value = amapText(strings.Join(list, ""))
		return nil
	}
	if string(data) == "null" {
		*value = ""
		return nil
	}
	return fmt.Errorf("unsupported amap text value: %s", string(data))
}
