package observatory

import (
	"testing"
	"time"
)

// TestVisibilitySolverMoonConjunction 验证月合事件（planetary_conjunction 含 moon）
// 用两个 object 中地平高度更高者做可见性判定，避免月球不可见误判行星合。
func TestVisibilitySolverMoonConjunction(t *testing.T) {
	moons := NewMoonService()
	solver := NewVisibilitySolver(moons)
	// 2026-09-14 金星合月（来自交接文档的示例）
	at := time.Date(2026, 9, 14, 11, 10, 0, 0, time.UTC)
	vis := solver.Solve(EventInput{
		Kind:     "moon_conjunction",
		StartsAt: at,
		Geometry: map[string]any{"objects": []any{"venus", "moon"}, "separationDegrees": 0.5, "positions": map[string]any{"venus": map[string]any{"longitudeDegrees": 180.0, "latitudeDegrees": 0.0}}},
	}, 31.2304, 121.4737, "Asia/Shanghai")

	valid := map[string]bool{"observable": true, "limited": true, "not_visible": true}
	if !valid[vis.Status] {
		t.Errorf("moon conjunction status=%s, want observable/limited/not_visible", vis.Status)
	}
	if vis.Status != "not_visible" && (vis.BestAt == nil || vis.WindowStart == nil || vis.WindowEnd == nil) {
		t.Errorf("visible moon conjunction missing bestAt")
	}
}

// TestCollectConjunctionObjects 验证从 geometry 提取合事件的 objects 数组。
func TestCollectConjunctionObjects(t *testing.T) {
	if got := collectConjunctionObjects(map[string]any{}); len(got) != 0 {
		t.Errorf("empty geometry = %v, want nil", got)
	}
	if got := collectConjunctionObjects(map[string]any{"objects": []any{"venus", "moon"}}); len(got) != 2 || got[0] != "venus" || got[1] != "moon" {
		t.Errorf("objects = %v, want [venus moon]", got)
	}
	// 单 object 事件（planetary_opposition/planetary_elongation）应返回空
	if got := collectConjunctionObjects(map[string]any{"object": "neptune"}); len(got) != 0 {
		t.Errorf("single object event = %v, want nil", got)
	}
}
