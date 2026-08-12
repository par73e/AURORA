package observatory

import (
	"testing"
	"time"
)

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
		Geometry: map[string]any{"objects": []any{"venus", "moon"}, "separationDegrees": 0.5},
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
		Geometry: map[string]any{"object": "neptune"},
	}, 31.2304, 121.4737, "Asia/Shanghai")

	valid := map[string]bool{"observable": true, "limited": true, "not_visible": true}
	if !valid[vis.Status] {
		t.Errorf("planetary opposition status=%s, want observable/limited/not_visible", vis.Status)
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
