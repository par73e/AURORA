// 光污染集成：观测评分的光污染因子与 /astronomy/light-pollution 端点。
//
// 数据诚实边界：本项目不为没有实测来源的光污染伪造 Bortle/SQM 值。本模块提供
// LightPollutionProvider 接口与可配置客户端（Light Pollution Map QueryRaster，
// 需 API Key，配置于 LIGHT_POLLUTION_KEY / LIGHT_POLLUTION_URL）：
//   - 未配置 Key 时返回 ErrNotConfigured，评分不含光污染因子，接口返回 503；
//   - 配置 Key 后按所设服务拉取夜间灯光辐射值并换算为 SQM/Bortle（换算公式
//     已单测覆盖；服务端响应契约以所用服务实际返回为准）。
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
	Bortle      float64 `json:"bortle"` // 1（最暗）– 9（城市中心）
	SQM         float64 `json:"sqm"`    // 天顶天空亮度 mag/arcsec²（约 17–22）
	Source      string  `json:"source"`
	RetrievedAt int64   `json:"retrievedAt"`
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
	// 夜间灯光辐射（nW/cm²/sr）→ 天顶亮度 SQM（mag/arcsec²）。
	// 采用文献常用的对数拟合（暗空 SQM≈22 对应极小辐射），属于估算而非实测。
	sqm := 22.3 - 2.5*math.Log10(radiance+0.05)
	if sqm > 22.3 {
		sqm = 22.3
	}
	return LightPollution{
		Bortle:      sqmToBortle(sqm),
		SQM:         math.Round(sqm*100) / 100,
		Source:      "Light Pollution Map (VIIRS 夜间灯光，估算)",
		RetrievedAt: client.now().UTC().Unix(),
	}, nil
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
