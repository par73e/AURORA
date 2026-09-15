package location

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

type ForwardGeocoder interface {
	Search(context.Context, string) ([]Candidate, error)
}

// Candidate 是地点名称查询的最小结果。经纬度统一转换为 WGS84，
// 可直接交给天气、星历与光污染计算，不把高德 GCJ-02 坐标泄漏到计算层。
type Candidate struct {
	Label     string  `json:"label"`
	Province  string  `json:"province"`
	City      string  `json:"city"`
	District  string  `json:"district"`
	Adcode    string  `json:"adcode"`
	Level     string  `json:"level"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
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

// Search 使用高德正向地理编码解析乡、镇、区县与城市名称。高德返回 GCJ-02，
// 这里在服务端转换回 WGS84，确保后续天文计算与 Open-Meteo 使用同一坐标系。
func (client *AMapClient) Search(ctx context.Context, queryText string) ([]Candidate, error) {
	if client.key == "" {
		return nil, ErrNotConfigured
	}
	queryText = strings.TrimSpace(queryText)
	if queryText == "" {
		return nil, errors.New("location query is empty")
	}
	query := url.Values{
		"key":     {client.key},
		"address": {queryText},
		"output":  {"JSON"},
	}
	var response struct {
		Status   string `json:"status"`
		Info     string `json:"info"`
		InfoCode string `json:"infocode"`
		Geocodes []struct {
			FormattedAddress amapText `json:"formatted_address"`
			Province         amapText `json:"province"`
			City             amapText `json:"city"`
			District         amapText `json:"district"`
			Adcode           amapText `json:"adcode"`
			Location         amapText `json:"location"`
			Level            amapText `json:"level"`
		} `json:"geocodes"`
	}
	if err := client.getJSON(ctx, "/v3/geocode/geo", query, &response); err != nil {
		return nil, err
	}
	if response.Status != "1" {
		return nil, fmt.Errorf("amap geocoding failed: %s (%s)", response.Info, response.InfoCode)
	}
	candidates := make([]Candidate, 0, len(response.Geocodes))
	for _, item := range response.Geocodes {
		parts := strings.Split(string(item.Location), ",")
		if len(parts) != 2 {
			continue
		}
		gcjLongitude, longitudeErr := strconv.ParseFloat(parts[0], 64)
		gcjLatitude, latitudeErr := strconv.ParseFloat(parts[1], 64)
		if longitudeErr != nil || latitudeErr != nil {
			continue
		}
		longitude, latitude := gcj02ToWGS84(gcjLongitude, gcjLatitude)
		province, city, district := string(item.Province), string(item.City), string(item.District)
		if city == "" {
			city = province
		}
		label := strings.TrimSpace(string(item.FormattedAddress))
		if label == "" {
			label = placeLabel(city, district, province)
		}
		if label == "" {
			continue
		}
		candidates = append(candidates, Candidate{
			Label: label, Province: province, City: city, District: district,
			Adcode: string(item.Adcode), Level: string(item.Level),
			Latitude: latitude, Longitude: longitude,
		})
		if len(candidates) == 8 {
			break
		}
	}
	return candidates, nil
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

const (
	chinaSemiMajorAxis = 6378245.0
	chinaEccentricity  = 0.00669342162296594323
)

// gcj02ToWGS84 用数次正向迭代求逆。城市检索结果通常在中国境内；境外坐标原样返回。
func gcj02ToWGS84(longitude, latitude float64) (float64, float64) {
	if outsideMainlandChina(longitude, latitude) {
		return longitude, latitude
	}
	wgsLongitude, wgsLatitude := longitude, latitude
	for range 5 {
		convertedLongitude, convertedLatitude := wgs84ToGCJ02(wgsLongitude, wgsLatitude)
		wgsLongitude -= convertedLongitude - longitude
		wgsLatitude -= convertedLatitude - latitude
	}
	return wgsLongitude, wgsLatitude
}

func wgs84ToGCJ02(longitude, latitude float64) (float64, float64) {
	if outsideMainlandChina(longitude, latitude) {
		return longitude, latitude
	}
	deltaLongitude := transformLongitude(longitude-105, latitude-35)
	deltaLatitude := transformLatitude(longitude-105, latitude-35)
	radians := latitude / 180 * math.Pi
	magic := math.Sin(radians)
	magic = 1 - chinaEccentricity*magic*magic
	sqrtMagic := math.Sqrt(magic)
	deltaLatitude = deltaLatitude * 180 / ((chinaSemiMajorAxis * (1 - chinaEccentricity)) / (magic * sqrtMagic) * math.Pi)
	deltaLongitude = deltaLongitude * 180 / (chinaSemiMajorAxis / sqrtMagic * math.Cos(radians) * math.Pi)
	return longitude + deltaLongitude, latitude + deltaLatitude
}

func outsideMainlandChina(longitude, latitude float64) bool {
	return longitude < 72.004 || longitude > 137.8347 || latitude < 0.8293 || latitude > 55.8271
}

func transformLatitude(longitude, latitude float64) float64 {
	value := -100 + 2*longitude + 3*latitude + 0.2*latitude*latitude + 0.1*longitude*latitude + 0.2*math.Sqrt(math.Abs(longitude))
	value += (20*math.Sin(6*longitude*math.Pi) + 20*math.Sin(2*longitude*math.Pi)) * 2 / 3
	value += (20*math.Sin(latitude*math.Pi) + 40*math.Sin(latitude/3*math.Pi)) * 2 / 3
	return value + (160*math.Sin(latitude/12*math.Pi)+320*math.Sin(latitude*math.Pi/30))*2/3
}

func transformLongitude(longitude, latitude float64) float64 {
	value := 300 + longitude + 2*latitude + 0.1*longitude*longitude + 0.1*longitude*latitude + 0.1*math.Sqrt(math.Abs(longitude))
	value += (20*math.Sin(6*longitude*math.Pi) + 20*math.Sin(2*longitude*math.Pi)) * 2 / 3
	value += (20*math.Sin(longitude*math.Pi) + 40*math.Sin(longitude/3*math.Pi)) * 2 / 3
	return value + (150*math.Sin(longitude/12*math.Pi)+300*math.Sin(longitude/30*math.Pi))*2/3
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
