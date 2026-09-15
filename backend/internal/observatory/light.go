// 光污染集成：提供地点长期环境基线与 /astronomy/light-pollution 端点。
//
// 数据诚实边界：VIIRS 的 radiance 是卫星实测的向上夜间辐射，SQM/Bortle 则是
// 由辐射值推导的观测参考，不是地面仪器实测值。接口同时返回原始辐射、数据年份、
// 分辨率和 estimated 标记，避免把估算值包装成实时实测。
package observatory

import (
	"context"
	"errors"
	"math"
	"time"
)

// ErrNotConfigured 表示本地光污染栅格尚未配置。
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

// estimatedLightPollution 保留卫星实测辐射，并给出便于理解夜空背景亮度的启发式 SQM/Bortle。
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
