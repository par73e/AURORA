package observatory

import (
	"testing"
	"time"
)

type visibilityMoon struct {
	altitude func(time.Time) float64
	phase    func(time.Time) MoonPhaseResult
	azimuth  float64
}

func (moon visibilityMoon) Day(_, _, _ float64, at time.Time, timezone string) (MoonDay, error) {
	return MoonDay{Date: at.Format(time.DateOnly), Timezone: timezone}, nil
}

func (moon visibilityMoon) Phase(at time.Time) MoonPhaseResult {
	if moon.phase != nil {
		return moon.phase(at)
	}
	return MoonPhaseResult{}
}

func (moon visibilityMoon) Altitude(_, _, _ float64, at time.Time) float64 {
	if moon.altitude != nil {
		return moon.altitude(at)
	}
	return 45
}

func (moon visibilityMoon) Azimuth(_, _, _ float64, _ time.Time) float64 {
	return moon.azimuth
}

// TestVisibilitySolverNonVisualNodes 验证天文几何节点返回 non_visual。
func TestVisibilitySolverNonVisualNodes(t *testing.T) {
	solver := NewVisibilitySolver(nil)
	kinds := []string{
		"new_moon", "lunar_perigee", "lunar_apogee", "ascending_node", "descending_node",
		"march_equinox", "june_solstice", "september_equinox", "december_solstice",
	}
	for _, kind := range kinds {
		vis := solver.Solve(EventInput{Kind: kind, StartsAt: time.Now()}, 31.23, 121.47, "Asia/Shanghai")
		if vis.Status != "non_visual" {
			t.Errorf("kind=%s status=%s, want non_visual", kind, vis.Status)
		}
	}
}

// TestVisibilitySolverPlanetaryConjunction 验证行星合事件给出 observable/limited/not_visible 之一，
// 而非 not_calculated。
func TestVisibilitySolverPlanetaryConjunction(t *testing.T) {
	moons := NewMoonService()
	solver := NewVisibilitySolver(moons)
	// 2026-09-14 金星合月（来自交接文档的示例）
	at := time.Date(2026, 9, 14, 11, 10, 0, 0, time.UTC)
	vis := solver.Solve(EventInput{
		Kind:     "planetary_conjunction",
		StartsAt: at,
		Geometry: map[string]any{"objects": []any{"venus", "moon"}, "separationDegrees": 0.5, "positions": map[string]any{"venus": map[string]any{"longitudeDegrees": 180.0, "latitudeDegrees": 0.0}}},
	}, 31.2304, 121.4737, "Asia/Shanghai")

	valid := map[string]bool{"observable": true, "limited": true, "not_visible": true}
	if !valid[vis.Status] {
		t.Errorf("planetary conjunction status=%s, want observable/limited/not_visible", vis.Status)
	}
	if vis.Status != "not_visible" && vis.BestAt == nil {
		t.Errorf("visible conjunction missing bestAt")
	}
}

// TestVisibilitySolverPlanetaryOpposition 验证行星冲日事件给出可见性结论。
func TestVisibilitySolverPlanetaryOpposition(t *testing.T) {
	moons := NewMoonService()
	solver := NewVisibilitySolver(moons)
	// 2026-09-26 海王星冲日
	at := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	vis := solver.Solve(EventInput{
		Kind:     "planetary_opposition",
		StartsAt: at,
		Geometry: map[string]any{"object": "neptune", "positions": map[string]any{"neptune": map[string]any{"longitudeDegrees": 180.0, "latitudeDegrees": 0.0}}},
	}, 31.2304, 121.4737, "Asia/Shanghai")

	valid := map[string]bool{"observable": true, "limited": true, "not_visible": true}
	if !valid[vis.Status] {
		t.Errorf("planetary opposition status=%s, want observable/limited/not_visible", vis.Status)
	}
}

func TestVisibilitySolverRejectsPlanetaryEventWithoutCoordinates(t *testing.T) {
	solver := NewVisibilitySolver(NewMoonService())
	vis := solver.Solve(EventInput{Kind: "planetary_opposition", StartsAt: time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC), Geometry: map[string]any{"object": "neptune"}}, 31.2304, 121.4737, "Asia/Shanghai")
	if vis.Status != "not_calculated" {
		t.Fatalf("status=%s, want not_calculated when JPL coordinates are absent", vis.Status)
	}
}

// TestVisibilitySolverUnknownKindReturnsNotCalculated 验证未接入求解器的事件类型
// 返回 not_calculated，绝不把全球事件误标为本地可见。
func TestVisibilitySolverUnknownKindReturnsNotCalculated(t *testing.T) {
	solver := NewVisibilitySolver(nil)
	vis := solver.Solve(EventInput{Kind: "unknown_event_kind", StartsAt: time.Now()}, 31.23, 121.47, "Asia/Shanghai")
	if vis.Status != "not_calculated" {
		t.Errorf("unknown kind status=%s, want not_calculated", vis.Status)
	}
}

func TestVisibilitySolverDoesNotGuessSolarEclipseOrMeteorRadiant(t *testing.T) {
	solver := NewVisibilitySolver(NewMoonService())
	for _, event := range []EventInput{
		{Kind: "solar_eclipse", StartsAt: time.Now()},
		{Kind: "meteor_shower", StartsAt: time.Now(), Geometry: map[string]any{}},
	} {
		if got := solver.Solve(event, 31.23, 121.47, "Asia/Shanghai").Status; got != "not_calculated" {
			t.Errorf("kind=%s status=%s, want not_calculated", event.Kind, got)
		}
	}
}

func TestVisibilitySolverUsesStaticEquatorialTargetCoordinates(t *testing.T) {
	solver := NewVisibilitySolver(NewMoonService())
	vis := solver.Solve(EventInput{
		Kind:     "moon_conjunction",
		StartsAt: time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC),
		Geometry: map[string]any{"objects": []any{"moon", "vega"}, "positions": map[string]any{
			"vega": map[string]any{"rightAscensionDegrees": 279.235, "declinationDegrees": 38.784},
		}},
	}, 31.23, 121.47, "Asia/Shanghai")
	if vis.Status == "not_calculated" {
		t.Fatalf("status=%s, static RA/Dec must be usable for local visibility", vis.Status)
	}
}

// TestVisibilitySolverMoonPhaseWithMoons 验证满月事件通过月球中天给出 observable/limited。
func TestVisibilitySolverMoonPhaseWithMoons(t *testing.T) {
	moons := NewMoonService()
	solver := NewVisibilitySolver(moons)
	// 取一个未来满月时刻
	at := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	vis := solver.Solve(EventInput{
		Kind:     "full_moon",
		StartsAt: at,
	}, 31.2304, 121.4737, "Asia/Shanghai")

	valid := map[string]bool{"observable": true, "limited": true, "not_visible": true}
	if !valid[vis.Status] {
		t.Errorf("full_moon status=%s, want observable/limited/not_visible", vis.Status)
	}
}

func TestVisibilitySolverMoonPhaseSearchesAdjacentNight(t *testing.T) {
	eventAt := time.Date(2026, time.September, 24, 12, 0, 0, 0, time.UTC)
	solver := NewVisibilitySolver(visibilityMoon{altitude: func(time.Time) float64 { return 45 }, azimuth: 123})
	vis := solver.Solve(EventInput{Kind: "full_moon", StartsAt: eventAt}, 0, 0, "UTC")
	if vis.BestAt == nil || vis.WindowStart == nil || vis.WindowEnd == nil {
		t.Fatalf("adjacent-night result lacks times: %+v", vis)
	}
	bestAt, err := time.Parse(time.RFC3339, *vis.BestAt)
	if err != nil {
		t.Fatal(err)
	}
	if bestAt.Equal(eventAt) {
		t.Fatal("daytime phase instant was incorrectly used as the viewing time")
	}
	if sunAltitude := EclipticToHorizontalAltitude("sun", 0, 0, bestAt); sunAltitude > -6 {
		t.Fatalf("bestAt sun altitude = %.2f, want <= -6", sunAltitude)
	}
	if vis.AzimuthDegrees == nil || *vis.AzimuthDegrees != 123 {
		t.Fatalf("azimuth = %v, want 123", vis.AzimuthDegrees)
	}
}

func TestVisibilitySolverLunarEclipseUsesEventEndAndAzimuth(t *testing.T) {
	startsAt := time.Date(2026, time.December, 20, 0, 0, 0, 0, time.UTC)
	endsAt := startsAt.Add(4 * time.Hour)
	solver := NewVisibilitySolver(visibilityMoon{altitude: func(time.Time) float64 { return 40 }, azimuth: 210})
	vis := solver.Solve(EventInput{Kind: "lunar_eclipse", StartsAt: startsAt, EndsAt: &endsAt, Geometry: map[string]any{"eclipseType": "Total"}}, 0, 0, "UTC")
	if vis.Status == "not_visible" || vis.Status == "not_calculated" {
		t.Fatalf("lunar eclipse status = %s, want visible result", vis.Status)
	}
	if vis.AzimuthDegrees == nil || *vis.AzimuthDegrees != 210 {
		t.Fatalf("azimuth = %v, want 210", vis.AzimuthDegrees)
	}
	windowEnd, err := time.Parse(time.RFC3339, *vis.WindowEnd)
	if err != nil {
		t.Fatal(err)
	}
	if !windowEnd.After(startsAt.Add(2 * time.Hour)) {
		t.Fatalf("window ended at %s; event end %s was not considered", windowEnd, endsAt)
	}
}

func TestVisibilitySolverMeteorChecksMoonlightAcrossWindow(t *testing.T) {
	event := EventInput{ID: "perseids-2026", Kind: "meteor_shower", StartsAt: time.Date(2026, time.August, 12, 16, 0, 0, 0, time.UTC), Geometry: map[string]any{"slug": "perseids"}}
	darkMoon := visibilityMoon{altitude: func(time.Time) float64 { return 60 }, phase: func(time.Time) MoonPhaseResult { return MoonPhaseResult{} }, azimuth: 90}
	baseline := NewVisibilitySolver(darkMoon).Solve(event, 31.2304, 121.4737, "Asia/Shanghai")
	if baseline.Status != "observable" || baseline.BestAt == nil {
		t.Fatalf("baseline = %+v, want observable", baseline)
	}
	bestAt, err := time.Parse(time.RFC3339, *baseline.BestAt)
	if err != nil {
		t.Fatal(err)
	}
	variableMoon := visibilityMoon{
		altitude: func(time.Time) float64 { return 60 },
		phase: func(at time.Time) MoonPhaseResult {
			if at.Equal(bestAt) {
				return MoonPhaseResult{Illumination: 0}
			}
			return MoonPhaseResult{Illumination: 1}
		},
		azimuth: 90,
	}
	vis := NewVisibilitySolver(variableMoon).Solve(event, 31.2304, 121.4737, "Asia/Shanghai")
	if vis.Status != "limited" || vis.Reason != "流星雨极大时月光较强，观测条件有限。" {
		t.Fatalf("window moonlight result = %+v, want moonlight-limited", vis)
	}
	if vis.AzimuthDegrees == nil {
		t.Fatal("meteor shower result lacks radiant azimuth")
	}
}

func TestContiguousVisibilityWindowDoesNotMergeGaps(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	times := []time.Time{start, start.Add(30 * time.Minute), start.Add(3 * time.Hour), start.Add(3*time.Hour + 30*time.Minute)}
	windowStart, windowEnd, first, last := contiguousVisibilityWindow(times, 2, 30*time.Minute)
	if first != 2 || last != 3 || !windowStart.Equal(times[2]) || !windowEnd.Equal(times[3]) {
		t.Fatalf("window = %s..%s (%d..%d), want second contiguous segment", windowStart, windowEnd, first, last)
	}
}
