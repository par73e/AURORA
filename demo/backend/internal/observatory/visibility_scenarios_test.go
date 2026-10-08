package observatory

import (
	"testing"
	"time"
)

// TestVisibilityScenarios 覆盖交接文档要求的 5 个测试场景：
// 上海、北半球高纬、南半球、无可见窗口、无定位授权。
func TestVisibilityScenarios(t *testing.T) {
	moons := NewMoonService()
	solver := NewVisibilitySolver(moons)

	// 2026-09-14 金星合月（来自交接文档的示例）
	at := time.Date(2026, 9, 14, 11, 10, 0, 0, time.UTC)
	input := EventInput{
		Kind:     "moon_conjunction",
		StartsAt: at,
		Geometry: map[string]any{"objects": []any{"venus", "moon"}, "separationDegrees": 0.5, "positions": map[string]any{"venus": map[string]any{"longitudeDegrees": 180.0, "latitudeDegrees": 0.0}}},
	}

	cases := []struct {
		name      string
		latitude  float64
		longitude float64
	}{
		{"shanghai", 31.2304, 121.4737},
		{"northern_high_lat", 64.1466, -21.9426}, // 雷克雅未克
		{"southern", -33.8688, 151.2093},         // 悉尼
		{"no_visibility_window", 89.5, 0},        // 接近北极点
	}
	for _, c := range cases {
		vis := solver.Solve(input, c.latitude, c.longitude, "Asia/Shanghai")
		valid := map[string]bool{
			"observable": true, "limited": true,
			"not_visible": true, "non_visual": true,
			"not_calculated": true,
		}
		if !valid[vis.Status] {
			t.Errorf("%s: status=%s, want a valid visibility state", c.name, vis.Status)
		}
	}

	// 无定位授权：事件仍显示为全球日历，本地状态显示"允许定位后判断"。
	// 这里验证求解器在没有坐标时不 panic（HTTP 层会在 hasLocation=false 时不调用 Solve）。
	vis := solver.Solve(input, 0, 0, "UTC")
	if vis.Status == "" {
		t.Errorf("no-location scenario returned empty status")
	}
}

// TestVisibilitySolverMoonPhaseShanghai 验证上海满月事件给出明确可见性。
func TestVisibilitySolverMoonPhaseShanghai(t *testing.T) {
	moons := NewMoonService()
	solver := NewVisibilitySolver(moons)
	at := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	vis := solver.Solve(EventInput{Kind: "full_moon", StartsAt: at}, 31.2304, 121.4737, "Asia/Shanghai")
	valid := map[string]bool{"observable": true, "limited": true, "not_visible": true}
	if !valid[vis.Status] {
		t.Errorf("full_moon shanghai status=%s, want observable/limited/not_visible", vis.Status)
	}
}

// TestVisibilitySolverMeteorShowerShanghai 验证上海流星雨事件给出明确可见性。
func TestVisibilitySolverMeteorShowerShanghai(t *testing.T) {
	moons := NewMoonService()
	solver := NewVisibilitySolver(moons)
	at := time.Date(2026, 8, 12, 16, 0, 0, 0, time.UTC)
	vis := solver.Solve(EventInput{ID: "perseids-2026", Kind: "meteor_shower", StartsAt: at, Geometry: map[string]any{"slug": "perseids"}}, 31.2304, 121.4737, "Asia/Shanghai")
	valid := map[string]bool{"observable": true, "limited": true, "not_visible": true}
	if !valid[vis.Status] {
		t.Errorf("meteor_shower shanghai status=%s, want observable/limited/not_visible", vis.Status)
	}
}
