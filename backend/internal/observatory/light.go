// 光污染集成：观测评分的光污染因子与 /astronomy/light-pollution 端点。
//
// 数据诚实边界：VIIRS 的 radiance 是卫星实测的向上夜间辐射，SQM/Bortle 则是
// 由辐射值推导的观测参考，不是地面仪器实测值。接口同时返回原始辐射、数据年份、
// 分辨率和 estimated 标记，避免把估算值包装成实时实测。
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
	"strings"
	"time"
)

// ErrNotConfigured 表示光污染源尚未配置（缺 Key 或未启用）。
var ErrNotConfigured = errors.New("light pollution source is not configured")

// LightPollution 是光污染数据：Bortle 等级与天顶亮度。
type LightPollution struct {
	Bortle           float64 `json:"bortle"`       // 1（最暗）– 9（城市中心），估算
	SQM              float64 `json:"sqm"`          // mag/arcsec²，估算
	Radiance         float64 `json:"radiance"`     // VIIRS 年度合成辐射值
	RadianceUnit     string  `json:"radianceUnit"` // nW/cm²/sr
	DataYear         int     `json:"dataYear,omitempty"`
	ResolutionMeters int     `json:"resolutionMeters,omitempty"`
	Estimated        bool    `json:"estimated"`
	Model            string  `json:"model,omitempty"`
	Source           string  `json:"source"`
	RetrievedAt      int64   `json:"retrievedAt"`
}

// LightPollutionProvider 提供指定地点的光污染数据。
type LightPollutionProvider interface {
	Light(context.Context, float64, float64) (LightPollution, error)
}

// LightClient 是 Light Pollution Map QueryRaster 的客户端。
type LightClient struct {
	key        string
	baseURL    string
	httpClient *http.Client
	now        func() time.Time
}

// NewLightPollutionClient 构造光污染客户端。key 为空时所有调用返回 ErrNotConfigured。
func NewLightPollutionClient(key string) *LightClient {
	return NewLightPollutionClientWithURLs(key, "https://www.lightpollutionmap.info", &http.Client{Timeout: 8 * time.Second})
}

func NewLightPollutionClientWithURLs(key, baseURL string, httpClient *http.Client) *LightClient {
	return &LightClient{key: strings.TrimSpace(key), baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpClient, now: time.Now}
}

// Light 查询指定地点的夜间灯光辐射值并换算为 SQM/Bortle。
func (client *LightClient) Light(ctx context.Context, latitude, longitude float64) (LightPollution, error) {
	if client.key == "" {
		return LightPollution{}, ErrNotConfigured
	}
	radiance, err := client.queryRadiance(ctx, latitude, longitude)
	if err != nil {
		return LightPollution{}, err
	}
	if !isFinite(radiance) || radiance < 0 {
		return LightPollution{}, errors.New("light pollution source returned invalid radiance")
	}
	return estimatedLightPollution(radiance, "Light Pollution Map QueryRaster", 0, 0, client.now()), nil
}

// estimatedLightPollution 保留卫星实测辐射，并给出兼容现有评分的启发式 SQM/Bortle。
// 该换算没有模拟地形、大气和周边光源传播，因此必须始终标记为 estimated。
func estimatedLightPollution(radiance float64, source string, dataYear, resolutionMeters int, retrievedAt time.Time) LightPollution {
	sqm := 22.3 - 2.5*math.Log10(radiance+0.05)
	if sqm > 22.3 {
		sqm = 22.3
	}
	return LightPollution{
		Bortle:           sqmToBortle(sqm),
		SQM:              math.Round(sqm*100) / 100,
		Radiance:         math.Round(radiance*100) / 100,
		RadianceUnit:     "nW/cm²/sr",
		DataYear:         dataYear,
		ResolutionMeters: resolutionMeters,
		Estimated:        true,
		Model:            "AURORA VIIRS radiance heuristic v1",
		Source:           source,
		RetrievedAt:      retrievedAt.UTC().Unix(),
	}
}

func (client *LightClient) queryRadiance(ctx context.Context, latitude, longitude float64) (float64, error) {
	endpoint, err := url.Parse(client.baseURL + "/QueryRaster/")
	if err != nil {
		return 0, err
	}
	query := endpoint.Query()
	query.Set("q", fmt.Sprintf("%.4f,%.4f", latitude, longitude))
	query.Set("key", client.key)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return 0, err
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, errors.New("request light pollution source failed")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("light pollution source returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return 0, err
	}
	// 兼容常见响应形态：裸数字文本 / JSON 数字 / JSON 数组首元素。
	if value, ok := parseNumericBody(body); ok {
		return value, nil
	}
	return 0, errors.New("light pollution source returned an unrecognized response")
}

func parseNumericBody(body []byte) (float64, bool) {
	var number float64
	if err := json.Unmarshal(body, &number); err == nil {
		return number, true
	}
	var list []float64
	if err := json.Unmarshal(body, &list); err == nil && len(list) > 0 {
		return list[0], true
	}
	var text string
	if err := json.Unmarshal(body, &text); err == nil {
		if value, perr := strconv.ParseFloat(strings.TrimSpace(text), 64); perr == nil {
			return value, true
		}
	}
	raw := strings.TrimSpace(string(body))
	if value, err := strconv.ParseFloat(raw, 64); err == nil {
		return value, true
	}
	return 0, false
}

// sqmToBortle 按公开的 SQM↔Bortle 分档近似映射。
func sqmToBortle(sqm float64) float64 {
	switch {
	case sqm >= 21.7:
		return 1
	case sqm >= 21.5:
		return 2
	case sqm >= 21.3:
		return 3
	case sqm >= 20.4:
		return 4
	case sqm >= 19.1:
		return 5
	case sqm >= 18.0:
		return 6
	case sqm >= 17.3:
		return 7
	case sqm >= 16.6:
		return 8
	default:
		return 9
	}
}
