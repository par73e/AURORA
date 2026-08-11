// 观测评分：把会随时刻变化的天气（云层/降水/通透度）与月光结合，为指定时刻生成观测评分。
// 评分从 100 分向下扣除限制因素；光污染作为地点长期环境数据随结果返回，但不参与动态评分。
// 但加入时间维度：时刻落在未来 24 小时预报窗口内使用真实天气，窗口外降级为"仅星历"并明示。
package observatory

import (
	"context"
	"math"
	"time"
)

// ScoreFactors 是评分的因子分解，前端可据此解释分数来源。
type ScoreFactors struct {
	VisibilityPenalty    float64 `json:"visibilityPenalty"`
	CloudPenalty         float64 `json:"cloudPenalty"`
	MoonPenalty          float64 `json:"moonPenalty"`
	PrecipitationPenalty float64 `json:"precipitationPenalty"`
	AerosolPenalty       float64 `json:"aerosolPenalty"`
}

// ObservingScore 是一次观测评分结果。
type ObservingScore struct {
	At                   int64           `json:"at"`                   // 评分时刻（Unix 秒）
	Timezone             string          `json:"timezone"`             // 天气预报时区
	Score                *int            `json:"score"`                // 0–100；预报窗口外为 null
	Verdict              string          `json:"verdict"`              // 结论文案
	WithinForecastWindow bool            `json:"withinForecastWindow"` // 时刻是否在预报窗口内
	Factors              ScoreFactors    `json:"factors"`
	Weather              *Hourly         `json:"weather"`                  // 实际参与评分的天气快照；窗口外为 null
	Moon                 MoonPhaseResult `json:"moon"`                     // 评分时刻的月相
	MoonAltitude         float64         `json:"moonAltitude"`             // 评分时刻的月球地平高度角（度）
	MoonAboveHorizon     bool            `json:"moonAboveHorizon"`         // 是否达到月出/月落的标准地平高度
	LightPollution       *LightPollution `json:"lightPollution,omitempty"` // 可用时返回光污染数据
}

// HourScore 是逐小时评分（供动态推荐单请求获取）。
type HourScore struct {
	Time    string       `json:"time"`    // 本地时间 "2006-01-02T15:04"
	Score   int          `json:"score"`   // 0–100
	Verdict string       `json:"verdict"` // 结论文案
	Factors ScoreFactors `json:"factors"`
	Weather Hourly       `json:"weather"` // 该小时实际参与评分的天气快照
}

// ScoreObserving 计算指定时刻的观测评分。weather 失败时直接返回错误；
// 光污染源未配置或失败时只省略长期环境数据，不改变动态评分。
func ScoreObserving(ctx context.Context, conditions ConditionsProvider, moon MoonProvider, light LightPollutionProvider, latitude, longitude float64, at time.Time) (ObservingScore, error) {
	report, err := conditions.Conditions(ctx, latitude, longitude)
	if err != nil {
		return ObservingScore{}, err
	}
	lp := resolveLight(ctx, light, latitude, longitude)
	return scoreFromReport(report, moon, lp, latitude, longitude, at), nil
}

// ScoreSeries 为逐小时预报生成每小时评分（单请求，供前端动态推荐）。
func ScoreSeries(report Conditions, moon MoonProvider, latitude, longitude float64) []HourScore {
	if moon == nil || len(report.Hourly) == 0 {
		return nil
	}
	location := time.UTC
	if parsed, err := time.LoadLocation(report.Timezone); err == nil {
		location = parsed
	}
	scores := make([]HourScore, 0, len(report.Hourly))
	for _, hour := range report.Hourly {
		parsed, err := time.ParseInLocation("2006-01-02T15:04", hour.Time, location)
		if err != nil {
			continue
		}
		pointReport := Conditions{Timezone: report.Timezone, Elevation: report.Elevation, Hourly: []Hourly{hour}, AirQuality: report.AirQuality}
		score := scoreFromReport(pointReport, moon, nil, latitude, longitude, parsed)
		if score.Score == nil {
			continue
		}
		scores = append(scores, HourScore{Time: hour.Time, Score: *score.Score, Verdict: score.Verdict, Factors: score.Factors, Weather: hour})
	}
	return scores
}

func resolveLight(ctx context.Context, light LightPollutionProvider, latitude, longitude float64) *LightPollution {
	if light == nil {
		return nil
	}
	value, err := light.Light(ctx, latitude, longitude)
	if err != nil {
		return nil // 未配置或失败：不含光污染因子，也不把估算当实测
	}
	return &value
}

func scoreFromReport(report Conditions, moon MoonProvider, lp *LightPollution, latitude, longitude float64, at time.Time) ObservingScore {
	moonAltitude := moon.Altitude(latitude, longitude, report.Elevation, at)
	result := ObservingScore{
		At:               at.Unix(),
		Timezone:         report.Timezone,
		Moon:             moon.Phase(at),
		MoonAltitude:     moonAltitude,
		MoonAboveHorizon: moonAltitude >= moonRiseSetAltitude,
		LightPollution:   lp,
	}

	selected, within := weatherSnapshotAt(report, at)
	result.WithinForecastWindow = within
	if !within {
		result.Verdict = "超出未来 24 小时预报窗口，暂无天气评分；本地星历仍可计算"
		return result
	}

	result.Weather = &selected
	factors := ScoreFactors{
		VisibilityPenalty: visibilityPenaltyFrom(selected.VisibilityMeters),
		CloudPenalty:      cloudPenaltyFrom(selected),
		AerosolPenalty:    aerosolPenaltyAt(report.AirQuality, report.Timezone, selected.Time),
	}
	if result.MoonAboveHorizon {
		factors.MoonPenalty = moonPenaltyFrom(result.Moon.Illumination, moonAltitude)
	}
	factors.PrecipitationPenalty = precipitationPenaltyFrom(selected.Precipitation)
	raw := 100 - factors.VisibilityPenalty - factors.CloudPenalty - factors.MoonPenalty - factors.PrecipitationPenalty - factors.AerosolPenalty
	score := int(math.Round(math.Max(0, math.Min(100, raw))))
	result.Score = &score
	result.Factors = factors
	result.Verdict = scoreVerdict(score)
	return result
}

// weatherSnapshotAt 让评分与前端环境卡片使用同一份天气快照：
// 请求时刻靠近 Open-Meteo current（15 分钟粒度）时优先 current；预览其他时刻时使用最近整点预报。
func weatherSnapshotAt(report Conditions, at time.Time) (Hourly, bool) {
	selected, _, within := NearestHour(report.Hourly, report.Timezone, at)
	if !within {
		return Hourly{}, false
	}
	location := time.UTC
	if parsed, err := time.LoadLocation(report.Timezone); err == nil {
		location = parsed
	}
	currentAt, err := time.ParseInLocation("2006-01-02T15:04", report.Current.Time, location)
	if err != nil || math.Abs(at.Sub(currentAt).Minutes()) > 30 {
		return selected, true
	}
	current := report.Current
	return Hourly{
		Time: current.Time, Temperature: current.Temperature, DewPoint: current.DewPoint,
		CloudCover: current.CloudCover, CloudCoverLow: current.CloudCoverLow, CloudCoverMid: current.CloudCoverMid, CloudCoverHigh: current.CloudCoverHigh, VisibilityMeters: current.VisibilityMeters,
		Humidity: current.Humidity, Precipitation: current.Precipitation,
		WindSpeed: current.WindSpeed, WindGusts: current.WindGusts, WeatherCode: current.WeatherCode,
	}, true
}

// cloudPenaltyFrom 区分云层高度：低云最遮挡，中云其次，高云主要限制深空透明度。
// 个别天气源未给出分层云量时，使用总云量的保守回退。
func cloudPenaltyFrom(weather Hourly) float64 {
	layersAvailable := weather.CloudCoverLow > 0 || weather.CloudCoverMid > 0 || weather.CloudCoverHigh > 0
	if !layersAvailable {
		return math.Min(80, math.Max(0, weather.CloudCover*0.7))
	}
	value := weather.CloudCoverLow*0.75 + weather.CloudCoverMid*0.5 + weather.CloudCoverHigh*0.25
	return math.Min(80, math.Max(0, value))
}

// visibilityPenaltyFrom 仅把能见度低于 12 km 的部分视为通透度限制；高于此值不再虚增分数。
func visibilityPenaltyFrom(visibilityMeters float64) float64 {
	visibilityKM := visibilityMeters / 1000
	return math.Min(15, math.Max(0, (12-visibilityKM)*1.5))
}

// precipitationPenaltyFrom 对轻微降水与较强降水进行连续区分；一旦出现降水，镜面和天空都会受影响。
func precipitationPenaltyFrom(precipitationMM float64) float64 {
	if precipitationMM <= 0 {
		return 0
	}
	return math.Min(30, 10+10*math.Sqrt(precipitationMM))
}

// aerosolPenaltyAt 以 AOD 作为透明度的小幅修正。PM2.5 是人体空气质量指标，因此不直接计入。
func aerosolPenaltyAt(readings []AirQuality, timezone, at string) float64 {
	if len(readings) == 0 {
		return 0
	}
	location := time.UTC
	if parsed, err := time.LoadLocation(timezone); err == nil {
		location = parsed
	}
	target, err := time.ParseInLocation("2006-01-02T15:04", at, location)
	if err != nil {
		return 0
	}
	nearest := readings[0]
	nearestDiff := time.Duration(1<<62 - 1)
	found := false
	for _, reading := range readings {
		readingAt, err := time.ParseInLocation("2006-01-02T15:04", reading.Time, location)
		if err != nil {
			continue
		}
		diff := readingAt.Sub(target)
		if diff < 0 {
			diff = -diff
		}
		if diff < nearestDiff {
			nearest, nearestDiff = reading, diff
			found = true
		}
	}
	if !found {
		return 0
	}
	return math.Min(8, math.Max(0, (nearest.AerosolOpticalDepth-0.15)*15))
}

// moonPenaltyFrom 让月光影响同时随亮面比例和高度角连续变化。
// 基础观星仍可观察亮星与行星，因此天顶满月最多扣 12 分；深空限制由月光卡片另行说明。
func moonPenaltyFrom(illumination, altitude float64) float64 {
	if illumination <= 0 || altitude < moonRiseSetAltitude {
		return 0
	}
	clampedAltitude := math.Max(0, math.Min(90, altitude))
	altitudeFactor := math.Sqrt(math.Sin(clampedAltitude * math.Pi / 180))
	return math.Max(0, math.Min(1, illumination)) * 12 * altitudeFactor
}

// scoreVerdict 与前端 scoreVerdict 的分档保持一致。
func scoreVerdict(score int) string {
	switch {
	case score >= 85:
		return "条件出色，适合安排观测"
	case score >= 65:
		return "条件良好，适合多种目标"
	case score >= 40:
		return "条件一般，优先安排亮目标"
	default:
		return "条件受限，建议短时观察亮目标"
	}
}

// NearestHour 在逐小时序列中定位最接近 at 的时刻（按预报时区解释）。
// within 表示 at 落在预报覆盖区间内：不早于首小时、不晚于末小时后 1 小时。
func NearestHour(hourly []Hourly, timezone string, at time.Time) (Hourly, int, bool) {
	if len(hourly) == 0 {
		return Hourly{}, -1, false
	}
	location := time.UTC
	if parsed, err := time.LoadLocation(timezone); err == nil {
		location = parsed
	}
	local := at.In(location)

	best := 0
	bestDiff := time.Duration(1<<62 - 1)
	for index, hour := range hourly {
		parsed, err := time.ParseInLocation("2006-01-02T15:04", hour.Time, location)
		if err != nil {
			continue
		}
		diff := parsed.Sub(local)
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff {
			bestDiff = diff
			best = index
		}
	}

	first, firstErr := time.ParseInLocation("2006-01-02T15:04", hourly[0].Time, location)
	last, lastErr := time.ParseInLocation("2006-01-02T15:04", hourly[len(hourly)-1].Time, location)
	within := firstErr == nil && lastErr == nil && !local.Before(first) && !local.After(last.Add(time.Hour))
	return hourly[best], best, within
}
