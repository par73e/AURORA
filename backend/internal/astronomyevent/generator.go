package astronomyevent

import (
	"encoding/json"
	"fmt"
	"time"

	"aurora/backend/internal/observatory"
)

const auroraModelSourceURL = "https://aurora.local/astronomy-model"

// coreEvents 将已有太阳/月球模型转化为可持久化的全球事件事实。
// 这是 EphemerisSyncer.SyncPlanetaryPositions 唯一的 computed 写入路径中
// 负责核心月球/季节事件的部分；行星候选事件由 planetaryEvents 另外追加。
func coreEvents(from, to, verifiedAt time.Time) []Event {
	computed := observatory.CoreCalendarEvents(from, to)
	events := make([]Event, 0, len(computed))
	for _, event := range computed {
		title, titleEN, summary := calendarCopy(event.Kind)
		geometry, _ := json.Marshal(calendarGeometry(event))
		events = append(events, Event{
			ID:           fmt.Sprintf("%s-%s", event.Kind, event.At.UTC().Format("20060102-1504")),
			Kind:         event.Kind,
			Title:        title,
			TitleEN:      titleEN,
			StartsAt:     event.At.UTC(),
			DateLabel:    event.At.UTC().Format("2006年1月2日"),
			Summary:      summary,
			Origin:       "computed",
			SourceName:   "AURORA 天文计算模型",
			SourceURL:    auroraModelSourceURL,
			VerifiedAt:   verifiedAt.UTC().Format(time.DateOnly),
			Geometry:     geometry,
			Presentation: json.RawMessage(`{}`),
		})
	}
	return events
}

func startOfUTCDay(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func calendarGeometry(event observatory.CalendarEvent) map[string]any {
	geometry := map[string]any{}
	switch event.Kind {
	case "new_moon", "first_quarter", "full_moon", "last_quarter":
		geometry["phaseDegrees"] = event.Value
	case "lunar_perigee", "lunar_apogee":
		geometry["distanceKm"] = event.Value
	case "march_equinox", "june_solstice", "september_equinox", "december_solstice":
		geometry["solarLongitudeDegrees"] = event.Value
	case "ascending_node", "descending_node":
		geometry["direction"] = event.Direction
	}
	return geometry
}

func calendarCopy(kind string) (string, string, string) {
	copy := map[string][3]string{
		"new_moon":          {"新月", "NEW MOON", "月球与太阳在黄经上接近同一方向的朔望月节点。"},
		"first_quarter":     {"上弦月", "FIRST QUARTER", "月球相位达到上弦，傍晚至午夜更易观察。"},
		"full_moon":         {"满月", "FULL MOON", "月球相位达到满月。"},
		"last_quarter":      {"下弦月", "LAST QUARTER", "月球相位达到下弦，后半夜至清晨更易观察。"},
		"lunar_perigee":     {"月球过近地点", "LUNAR PERIGEE", "月球到达本轮椭圆轨道中距地球较近的位置。"},
		"lunar_apogee":      {"月球过远地点", "LUNAR APOGEE", "月球到达本轮椭圆轨道中距地球较远的位置。"},
		"ascending_node":    {"月球过升交点", "MOON ASCENDING NODE", "月球由黄道南侧穿越至黄道北侧。"},
		"descending_node":   {"月球过降交点", "MOON DESCENDING NODE", "月球由黄道北侧穿越至黄道南侧。"},
		"march_equinox":     {"春分", "MARCH EQUINOX", "太阳视黄经到达 0° 的季节节点。"},
		"june_solstice":     {"夏至", "JUNE SOLSTICE", "太阳视黄经到达 90° 的季节节点。"},
		"september_equinox": {"秋分", "SEPTEMBER EQUINOX", "太阳视黄经到达 180° 的季节节点。"},
		"december_solstice": {"冬至", "DECEMBER SOLSTICE", "太阳视黄经到达 270° 的季节节点。"},
	}
	if values, ok := copy[kind]; ok {
		return values[0], values[1], values[2]
	}
	return kind, kind, "由 AURORA 天文计算模型生成。"
}
