package observatory

import (
	"context"
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
	result MoonPhaseResult
}

func (s stubMoon) Phase(_ time.Time) MoonPhaseResult { return s.result }
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
	// 期望：50 + min(24, 20×2.4=48) − 10×0.52 − 0.25×22 − 0 − 0 = 63.3 → 63
	if *score.Score != 63 {
		t.Errorf("score = %d，期望 63", *score.Score)
	}
	if score.Verdict != "条件一般，优先安排亮目标" {
		t.Errorf("verdict = %q", score.Verdict)
	}
	if score.Weather == nil || score.Weather.Time != "2026-08-09T10:00" {
		t.Errorf("weather 应命中 10:00，得到 %+v", score.Weather)
	}
	if score.Factors.VisibilityBonus != 24 || score.Factors.CloudPenalty != 5.2 || score.Factors.MoonPenalty != 5.5 {
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
	// 期望：50 + 24 − 5.2 − 5.5 − 18 = 45.3 → 45
	if *score.Score != 45 {
		t.Errorf("score = %d，期望 45（含降水惩罚 18）", *score.Score)
	}
	if score.Factors.PrecipitationPenalty != 18 {
		t.Errorf("降水惩罚 = %v，期望 18", score.Factors.PrecipitationPenalty)
	}
	if score.Weather == nil || score.Weather.Time != "2026-08-09T08:00" {
		t.Errorf("weather 应命中 08:00，得到 %+v", score.Weather)
	}
}

func TestScoreObservingLightPollutionPenalty(t *testing.T) {
	conditions := stubConditions{report: Conditions{Timezone: "Asia/Shanghai", Hourly: shanghaiHourly()}}
	moon := stubMoon{result: MoonPhaseResult{Illumination: 0.25}}
	// 光污染：SQM 19.0（Bortle ~5，城市边缘）→ 惩罚 (22-19)*6 = 18
	light := stubLight{result: LightPollution{Bortle: 5, SQM: 19.0, Source: "test"}}
	at := unixTime("2026-08-09T02:00:00Z")

	score, err := ScoreObserving(context.Background(), conditions, moon, light, 31.23, 121.47, at)
	if err != nil {
		t.Fatalf("ScoreObserving error: %v", err)
	}
	// 期望：50 + 24 − 5.2 − 5.5 − 0 − 18 = 45.3 → 45
	if *score.Score != 45 {
		t.Errorf("score = %d，期望 45（含光污染惩罚 18）", *score.Score)
	}
	if score.Factors.LightPollutionPenalty != 18 {
		t.Errorf("光污染惩罚 = %v，期望 18", score.Factors.LightPollutionPenalty)
	}
	if score.LightPollution == nil || score.LightPollution.Bortle != 5 {
		t.Errorf("评分应携带光污染数据：%+v", score.LightPollution)
	}
}

func TestScoreSeries(t *testing.T) {
	hourly := shanghaiHourly()
	conditions := stubConditions{report: Conditions{Timezone: "Asia/Shanghai", Hourly: hourly}}
	moon := stubMoon{result: MoonPhaseResult{Illumination: 0.25}}

	scores := ScoreSeries(conditions.report.Hourly, conditions.report.Timezone, moon, nil, 31.23, 121.47)
	if len(scores) != len(hourly) {
		t.Fatalf("scores 长度 = %d，期望 %d", len(scores), len(hourly))
	}
	// 每小时同天气同月光 → 每小时同分 63
	for index, score := range scores {
		if score.Score != 63 {
			t.Errorf("scores[%d] = %d，期望 63", index, score.Score)
		}
		if score.Time != hourly[index].Time {
			t.Errorf("scores[%d].time = %q，期望 %q", index, score.Time, hourly[index].Time)
		}
	}
}

type stubLight struct {
	result LightPollution
	err    error
}

func (s stubLight) Light(_ context.Context, _, _ float64) (LightPollution, error) {
	return s.result, s.err
}
