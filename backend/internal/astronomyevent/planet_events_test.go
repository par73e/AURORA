package astronomyevent

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
)

// TestRefineQuadraticRecoversExtremum 验证等距三点二次插值能正确恢复抛物线极值位置。
// 构造 y = (t-0.3)² 的三个等距采样点（h=1，t=-1,0,1），
// 极值在 t=0.3，期望 refinedAt 接近 t0+0.3 天。
func TestRefineQuadraticRecoversExtremum(t *testing.T) {
	t0 := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	// 三个等距点，步长 1 天
	left := []EphemerisSample{
		{Epoch: t0.Add(-24 * time.Hour), XAU: 1.69}, // (t-0.3)² at t=-1 → 1.69
		{Epoch: t0, XAU: 0.09},                      // (t-0.3)² at t=0  → 0.09
		{Epoch: t0.Add(24 * time.Hour), XAU: 0.49},  // (t-0.3)² at t=1  → 0.49
	}
	right := []EphemerisSample{
		{Epoch: t0.Add(-24 * time.Hour)},
		{Epoch: t0},
		{Epoch: t0.Add(24 * time.Hour)},
	}
	// value 函数直接返回 left.XAU（测试用）
	refinedAt, refinedValue, ok := refineQuadratic(left, right, 1, func(l, r EphemerisSample) float64 {
		return l.XAU
	})
	if !ok {
		t.Fatalf("refineQuadratic() ok=false, want true")
	}
	// 期望极值在 t0 + 0.3 天 = 7.2 小时
	expectedOffset := time.Duration(0.3 * 24 * float64(time.Hour))
	actualOffset := refinedAt.Sub(t0)
	if math.Abs(float64(actualOffset-expectedOffset)) > float64(30*time.Minute) {
		t.Errorf("refinedAt offset = %v, want ~%v (0.3 day)", actualOffset, expectedOffset)
	}
	// 期望极值接近 0（抛物线 (t-0.3)² 在 t=0.3 处为 0）
	if math.Abs(refinedValue) > 0.01 {
		t.Errorf("refinedValue = %v, want ~0", refinedValue)
	}
}

// TestPlanetaryEventsRefinesConjunctionTiming 验证行星合事件经精化后
// geometry.precision 为 "refined" 且包含 refinedAt 字段。
func TestPlanetaryEventsRefinesConjunctionTiming(t *testing.T) {
	base := time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC)
	// 构造 mercury 和 venus 在第 1 天附近角距最小
	day := func(body string, degrees []float64) []EphemerisSample {
		items := make([]EphemerisSample, 0, len(degrees))
		for i, degree := range degrees {
			radians := degree * math.Pi / 180
			items = append(items, EphemerisSample{Body: body, Epoch: base.AddDate(0, 0, i), XAU: math.Cos(radians), YAU: math.Sin(radians)})
		}
		return items
	}
	events := planetaryEvents(map[string][]EphemerisSample{
		"mercury": day("mercury", []float64{10, 11, 12}),
		"venus":   day("venus", []float64{14, 11.1, 15}),
		"mars":    day("mars", []float64{80, 81, 82}),
		"jupiter": day("jupiter", []float64{120, 121, 122}),
		"saturn":  day("saturn", []float64{160, 161, 162}),
		"uranus":  day("uranus", []float64{200, 201, 202}),
		"neptune": day("neptune", []float64{240, 241, 242}),
		"sun":     day("sun", []float64{0, 1, 2}),
	}, base)

	foundRefined := false
	for _, event := range events {
		if event.Kind != "planetary_conjunction" {
			continue
		}
		var geometry map[string]any
		if err := json.Unmarshal(event.Geometry, &geometry); err != nil {
			t.Fatalf("unmarshal geometry: %v", err)
		}
		precision, _ := geometry["precision"].(string)
		if precision == "refined" {
			foundRefined = true
			if _, ok := geometry["refinedAt"]; !ok {
				t.Errorf("refined conjunction missing refinedAt field")
			}
			if _, ok := geometry["separationDegrees"]; !ok {
				t.Errorf("refined conjunction missing separationDegrees")
			}
		}
	}
	if !foundRefined {
		t.Errorf("no refined conjunction event found; precision should upgrade to refined")
	}
}

// TestRefineQuadraticHandlesBoundary 验证边界条件（index=0 或末尾）返回 ok=false。
func TestRefineQuadraticHandlesBoundary(t *testing.T) {
	left := []EphemerisSample{{Epoch: time.Now()}, {Epoch: time.Now()}, {Epoch: time.Now()}}
	right := left
	// index=0 → 边界
	if _, _, ok := refineQuadratic(left, right, 0, func(l, r EphemerisSample) float64 { return 0 }); ok {
		t.Errorf("refineQuadratic(index=0) ok=true, want false")
	}
}

func TestRefineQuadraticRejectsDegenerateSamples(t *testing.T) {
	t0 := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	right := []EphemerisSample{{Epoch: t0.Add(-24 * time.Hour)}, {Epoch: t0}, {Epoch: t0.Add(24 * time.Hour)}}
	tests := []struct {
		name string
		left []EphemerisSample
	}{
		{
			name: "zero sampling interval",
			left: []EphemerisSample{{Epoch: t0}, {Epoch: t0, XAU: 1}, {Epoch: t0.Add(24 * time.Hour), XAU: 2}},
		},
		{
			name: "linear values have no quadratic extremum",
			left: []EphemerisSample{{Epoch: t0.Add(-24 * time.Hour)}, {Epoch: t0, XAU: 1}, {Epoch: t0.Add(24 * time.Hour), XAU: 2}},
		},
		{
			name: "extremum outside central half step",
			left: []EphemerisSample{{Epoch: t0.Add(-24 * time.Hour), XAU: 0}, {Epoch: t0, XAU: 1}, {Epoch: t0.Add(24 * time.Hour), XAU: 3}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, ok := refineQuadratic(test.left, right, 1, func(l, _ EphemerisSample) float64 { return l.XAU }); ok {
				t.Fatal("refineQuadratic() ok=true, want false")
			}
		})
	}
}

func TestPlanetaryEventsIncludeCoordinatesForLocalVisibility(t *testing.T) {
	base := time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC)
	day := func(body string, degrees []float64) []EphemerisSample {
		items := make([]EphemerisSample, 0, len(degrees))
		for i, degree := range degrees {
			radians := degree * math.Pi / 180
			items = append(items, EphemerisSample{Body: body, Epoch: base.AddDate(0, 0, i), XAU: math.Cos(radians), YAU: math.Sin(radians)})
		}
		return items
	}
	samples := map[string][]EphemerisSample{
		"mercury": day("mercury", []float64{10, 11, 12}), "venus": day("venus", []float64{14, 11.1, 15}),
		"mars": day("mars", []float64{80, 81, 82}), "jupiter": day("jupiter", []float64{120, 121, 122}),
		"saturn": day("saturn", []float64{160, 161, 162}), "uranus": day("uranus", []float64{200, 201, 202}),
		"neptune": day("neptune", []float64{240, 241, 242}), "sun": day("sun", []float64{0, 1, 2}),
	}
	for _, event := range planetaryEvents(samples, base) {
		if event.Kind != "planetary_conjunction" {
			continue
		}
		var geometry map[string]any
		if err := json.Unmarshal(event.Geometry, &geometry); err != nil {
			t.Fatal(err)
		}
		if _, ok := geometry["positions"].(map[string]any); !ok {
			t.Fatalf("planetary event %s lacks positions for local visibility: %s", event.ID, event.Geometry)
		}
		return
	}
	t.Fatal("no conjunction generated")
}

func TestMoonBrightObjectConjunctionsCarryEquatorialCoordinates(t *testing.T) {
	base := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	reference := make([]EphemerisSample, 400)
	for index := range reference {
		reference[index] = EphemerisSample{Body: "sun", Epoch: base.AddDate(0, 0, index)}
	}
	events := moonBrightObjectConjunctionEvents(reference, base)
	if len(events) == 0 {
		t.Fatal("no bright-star or cluster conjunctions generated")
	}
	var geometry map[string]any
	if err := json.Unmarshal(events[0].Geometry, &geometry); err != nil {
		t.Fatal(err)
	}
	positions, ok := geometry["positions"].(map[string]any)
	if !ok || len(positions) != 1 {
		t.Fatalf("positions=%v, want one static target", geometry["positions"])
	}
	for _, value := range positions {
		coordinates, ok := value.(map[string]any)
		if !ok || coordinates["rightAscensionDegrees"] == nil || coordinates["declinationDegrees"] == nil {
			t.Fatalf("static target coordinates=%v, want RA/Dec", value)
		}
	}
}

func TestPlanetaryEventsIncludeSolarConjunction(t *testing.T) {
	base := time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC)
	day := func(body string, degrees []float64) []EphemerisSample {
		items := make([]EphemerisSample, 0, len(degrees))
		for i, degree := range degrees {
			radians := degree * math.Pi / 180
			items = append(items, EphemerisSample{Body: body, Epoch: base.AddDate(0, 0, i), XAU: math.Cos(radians), YAU: math.Sin(radians)})
		}
		return items
	}
	samples := map[string][]EphemerisSample{
		"mercury": day("mercury", []float64{2, 0.1, 3}), "venus": day("venus", []float64{40, 41, 42}),
		"mars": day("mars", []float64{80, 81, 82}), "jupiter": day("jupiter", []float64{120, 121, 122}),
		"saturn": day("saturn", []float64{160, 161, 162}), "uranus": day("uranus", []float64{200, 201, 202}),
		"neptune": day("neptune", []float64{240, 241, 242}), "sun": day("sun", []float64{0, 0, 0}),
	}
	for _, event := range planetaryEvents(samples, base) {
		if event.Kind == "planetary_solar_conjunction" && strings.Contains(event.ID, "mercury") {
			return
		}
	}
	t.Fatal("no Mercury solar conjunction generated")
}
