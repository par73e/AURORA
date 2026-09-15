package observatory

import (
	"context"
	"math"
	"testing"
	"time"
)

type stubConditions struct {
	report Conditions
}

func (s stubConditions) Conditions(_ context.Context, _, _ float64) (Conditions, error) {
	return s.report, nil
}

type stubMoon struct {
	result   MoonPhaseResult
	altitude float64
}

func (s stubMoon) Phase(_ time.Time) MoonPhaseResult { return s.result }
func (s stubMoon) Altitude(_, _, _ float64, _ time.Time) float64 {
	if s.altitude == 0 {
		return 30
	}
	return s.altitude
}
func (s stubMoon) Day(_, _, _ float64, _ time.Time, _ string) (MoonDay, error) {
	return MoonDay{}, nil
}

// shanghaiHourly 构造 08:00–15:00 共 8 个小时的固定预报。
func shanghaiHourly() []Hourly {
	times := []string{
		"2026-08-09T08:00", "2026-08-09T09:00", "2026-08-09T10:00", "2026-08-09T11:00",
		"2026-08-09T12:00", "2026-08-09T13:00", "2026-08-09T14:00", "2026-08-09T15:00",
	}
	hourly := make([]Hourly, len(times))
	for index, at := range times {
		hourly[index] = Hourly{Time: at, VisibilityMeters: 20000, CloudCover: 10, Precipitation: 0}
	}
	return hourly
}

func TestScoreObservingWithinWindow(t *testing.T) {
	conditions := stubConditions{report: Conditions{
		Timezone: "Asia/Shanghai",
		Hourly:   shanghaiHourly(),
	}}
	moon := stubMoon{result: MoonPhaseResult{Phase: 90, Illumination: 0.25, Age: 7.4, Label: "上弦月"}}
	// 2026-08-09T02:00:00Z = 上海 10:00，命中 10:00 那行。
	at := unixTime("2026-08-09T02:00:00Z")

	score, err := ScoreObserving(context.Background(), conditions, moon, nil, 31.23, 121.47, at)
	if err != nil {
		t.Fatalf("ScoreObserving error: %v", err)
	}
	if !score.WithinForecastWindow {
		t.Fatal("窗口内时刻应 withinForecastWindow=true")
	}
	if score.Score == nil {
		t.Fatal("窗口内应给出分数")
	}
	// 月球高度 30°，月光扣分 = 0.25×12×sqrt(sin(30°)) ≈ 2.12；
	// 未提供分层云量时，云量回退为 10×0.7=7。期望：100 − 7 − 2.12 = 90.88 → 91。
	if *score.Score != 91 {
		t.Errorf("score = %d，期望 91", *score.Score)
	}
	if score.Verdict != "条件出色，适合安排观测" {
		t.Errorf("verdict = %q", score.Verdict)
	}
	if score.Weather == nil || score.Weather.Time != "2026-08-09T10:00" {
		t.Errorf("weather 应命中 10:00，得到 %+v", score.Weather)
	}
	if score.Factors.VisibilityPenalty != 0 || score.Factors.CloudPenalty != 7 || math.Abs(score.Factors.MoonPenalty-2.1213203435596424) > 1e-9 {
		t.Errorf("因子分解不符：%+v", score.Factors)
	}
	if score.Moon.Label != "上弦月" {
		t.Errorf("moon 快照不符：%+v", score.Moon)
	}
	if score.LightPollution != nil {
		t.Errorf("未配置光污染时应为 nil，得到 %+v", score.LightPollution)
	}
}

func TestScoreObservingOutsideWindow(t *testing.T) {
	conditions := stubConditions{report: Conditions{Timezone: "Asia/Shanghai", Hourly: shanghaiHourly()}}
	moon := stubMoon{result: MoonPhaseResult{Illumination: 0.5}}
	// 次日 10:00 本地（2026-08-10T02:00:00Z）超出预报窗口。
	at := unixTime("2026-08-10T02:00:00Z")

	score, err := ScoreObserving(context.Background(), conditions, moon, nil, 31.23, 121.47, at)
	if err != nil {
		t.Fatalf("ScoreObserving error: %v", err)
	}
	if score.WithinForecastWindow {
		t.Fatal("窗口外时刻应 withinForecastWindow=false")
	}
	if score.Score != nil {
		t.Fatalf("窗口外不应给分数，得到 %d", *score.Score)
	}
	if score.Weather != nil {
		t.Fatalf("窗口外 weather 应为 nil")
	}
	if score.Verdict == "" {
		t.Fatal("窗口外应有明确结论文案")
	}
}

func TestScoreObservingPrecipitationPenalty(t *testing.T) {
	hourly := shanghaiHourly()
	hourly[0].Precipitation = 3.2 // 08:00 有降水
	conditions := stubConditions{report: Conditions{Timezone: "Asia/Shanghai", Hourly: hourly}}
	moon := stubMoon{result: MoonPhaseResult{Illumination: 0.25}}
	// 2026-08-09T00:30:00Z = 上海 08:30，命中 08:00 行。
	at := unixTime("2026-08-09T00:30:00Z")

	score, err := ScoreObserving(context.Background(), conditions, moon, nil, 31.23, 121.47, at)
	if err != nil {
		t.Fatalf("ScoreObserving error: %v", err)
	}
	// 任意可测降水都会把结果压到“不建议观测”的区间。
	if *score.Score != 16 {
		t.Errorf("score = %d，期望 16（含降水否决性惩罚）", *score.Score)
	}
	if score.Factors.PrecipitationPenalty != 75 {
		t.Errorf("降水惩罚 = %v，期望封顶 75", score.Factors.PrecipitationPenalty)
	}
	if score.Weather == nil || score.Weather.Time != "2026-08-09T08:00" {
		t.Errorf("weather 应命中 08:00，得到 %+v", score.Weather)
	}
}

func TestScoreObservingLightPollutionIsSeparateFromDynamicScore(t *testing.T) {
	conditions := stubConditions{report: Conditions{Timezone: "Asia/Shanghai", Hourly: shanghaiHourly()}}
	moon := stubMoon{result: MoonPhaseResult{Illumination: 0.25}}
	// 光污染：SQM 19.0（Bortle ~5，城市边缘）只作为长期环境数据返回。
	light := stubLight{result: LightPollution{Bortle: 5, SQM: 19.0, Source: "test"}}
	at := unixTime("2026-08-09T02:00:00Z")

	score, err := ScoreObserving(context.Background(), conditions, moon, light, 31.23, 121.47, at)
	if err != nil {
		t.Fatalf("ScoreObserving error: %v", err)
	}
	// 与相同天气和月光但没有光污染数据时一致：91 分。
	if *score.Score != 91 {
		t.Errorf("score = %d，期望 91（光污染不参与动态评分）", *score.Score)
	}
	if score.LightPollution == nil || score.LightPollution.Bortle != 5 {
		t.Errorf("评分应携带光污染数据：%+v", score.LightPollution)
	}
}

func TestScoreSeries(t *testing.T) {
	hourly := shanghaiHourly()
	conditions := stubConditions{report: Conditions{Timezone: "Asia/Shanghai", Hourly: hourly}}
	moon := stubMoon{result: MoonPhaseResult{Illumination: 0.25}}

	scores := ScoreSeries(conditions.report, moon, 31.23, 121.47)
	if len(scores) != len(hourly) {
		t.Fatalf("scores 长度 = %d，期望 %d", len(scores), len(hourly))
	}
	// 每小时同天气同月光 → 每小时同分 91
	for index, score := range scores {
		if score.Score != 91 {
			t.Errorf("scores[%d] = %d，期望 91", index, score.Score)
		}
		if score.Time != hourly[index].Time {
			t.Errorf("scores[%d].time = %q，期望 %q", index, score.Time, hourly[index].Time)
		}
	}
}

func TestScoreObservingUsesCurrentSnapshotNearNow(t *testing.T) {
	hourly := shanghaiHourly()
	hourly[2].Precipitation = 2.4 // 整点预报有雨，但 10:15 current 已无降水。
	conditions := stubConditions{report: Conditions{
		Timezone: "Asia/Shanghai",
		Current: Current{
			Time: "2026-08-09T10:15", VisibilityMeters: 20000, CloudCover: 10, Precipitation: 0,
		},
		Hourly: hourly,
	}}
	moon := stubMoon{result: MoonPhaseResult{Illumination: 0.25}}
	score, err := ScoreObserving(context.Background(), conditions, moon, nil, 31.23, 121.47, unixTime("2026-08-09T02:15:00Z"))
	if err != nil {
		t.Fatalf("ScoreObserving error: %v", err)
	}
	if score.Weather == nil || score.Weather.Time != "2026-08-09T10:15" {
		t.Fatalf("评分应使用 current 快照，得到 %+v", score.Weather)
	}
	if score.Factors.PrecipitationPenalty != 0 {
		t.Fatalf("current 无降水时不应沿用整点降水扣分，得到 %+v", score.Factors)
	}
}

func TestScoreObservingMoonBelowHorizonHasNoMoonPenalty(t *testing.T) {
	conditions := stubConditions{report: Conditions{Timezone: "Asia/Shanghai", Hourly: shanghaiHourly()}}
	moon := stubMoon{result: MoonPhaseResult{Illumination: 1}, altitude: -10}
	score, err := ScoreObserving(context.Background(), conditions, moon, nil, 31.23, 121.47, unixTime("2026-08-09T02:00:00Z"))
	if err != nil {
		t.Fatalf("ScoreObserving error: %v", err)
	}
	if score.MoonAboveHorizon || score.Factors.MoonPenalty != 0 {
		t.Fatalf("月亮在地平线下不应扣月光分，得到 altitude=%v factors=%+v", score.MoonAltitude, score.Factors)
	}
	if score.Score == nil || *score.Score != 93 {
		t.Fatalf("无月光扣分时 score=%v，期望 93", score.Score)
	}
}

func TestMoonPenaltyScalesWithAltitude(t *testing.T) {
	if got := moonPenaltyFrom(1, 0); got != 0 {
		t.Errorf("地平线上的满月扣分 = %v，期望 0", got)
	}
	if got := moonPenaltyFrom(1, 30); math.Abs(got-8.48528137423857) > 1e-9 {
		t.Errorf("30° 高满月扣分 = %v，期望约 8.49", got)
	}
	if got := moonPenaltyFrom(1, 90); math.Abs(got-12) > 1e-9 {
		t.Errorf("天顶满月扣分 = %v，期望 12", got)
	}
	if got := moonPenaltyFrom(0.5, 90); math.Abs(got-6) > 1e-9 {
		t.Errorf("天顶半月扣分 = %v，期望 6", got)
	}
}

func TestScoreDistributionMatchesObservingConditions(t *testing.T) {
	tests := []struct {
		name    string
		weather Hourly
		want    int
	}{
		{name: "clear moonless sky stays near the top of the scale", weather: Hourly{Time: "2026-08-09T10:00", VisibilityMeters: 20000}, want: 100},
		{name: "full high cloud is not a good general observing night", weather: Hourly{Time: "2026-08-09T10:00", VisibilityMeters: 20000, CloudCover: 100, CloudCoverHigh: 100}, want: 45},
		{name: "mixed cloud cover falls into the middle of the scale", weather: Hourly{Time: "2026-08-09T10:00", VisibilityMeters: 10000, CloudCover: 60, CloudCoverLow: 20, CloudCoverMid: 35, CloudCoverHigh: 50}, want: 63},
		{name: "overcast drizzle is effectively not an observing night", weather: Hourly{Time: "2026-08-09T10:00", VisibilityMeters: 8000, CloudCover: 96, CloudCoverLow: 22, CloudCoverMid: 83, CloudCoverHigh: 91, Precipitation: 0.1}, want: 0},
	}
	moon := stubMoon{result: MoonPhaseResult{Illumination: 0}, altitude: -10}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := Conditions{Timezone: "Asia/Shanghai", Hourly: []Hourly{test.weather}}
			score := scoreFromReport(report, moon, nil, 31.23, 121.47, unixTime("2026-08-09T02:00:00Z"))
			if score.Score == nil || *score.Score != test.want {
				t.Fatalf("score = %v，期望 %d，factors=%+v", score.Score, test.want, score.Factors)
			}
		})
	}
}

func TestAerosolPenaltyIsSmallTransparencyCorrection(t *testing.T) {
	readings := []AirQuality{{Time: "2026-08-09T10:00", AerosolOpticalDepth: 0.55}}
	if got := aerosolPenaltyAt(readings, "Asia/Shanghai", "2026-08-09T10:00"); math.Abs(got-6) > 1e-9 {
		t.Errorf("AOD 0.55 penalty = %v，期望 6", got)
	}
}

func TestOperationalWeatherRisksAffectScore(t *testing.T) {
	if got := weatherPenaltyFrom(95, 0); got != 75 {
		t.Fatalf("thunderstorm penalty = %v, want 75", got)
	}
	if got := windPenaltyFrom(40, 60); got != 15 {
		t.Fatalf("strong wind penalty = %v, want 15", got)
	}
	if got := dewPenaltyFrom(8, 7, 92); got != 8 {
		t.Fatalf("dew penalty = %v, want 8", got)
	}
	if got := dewPenaltyFrom(0, 0, 0); got != 0 {
		t.Fatalf("missing humidity must not create a dew penalty: %v", got)
	}
}

type stubLight struct {
	result LightPollution
	err    error
}

func (s stubLight) Light(_ context.Context, _, _ float64) (LightPollution, error) {
	return s.result, s.err
}
