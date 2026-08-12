package astronomyevent

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// planetaryEvents 从每天一条 JPL 矢量中找到合、冲与内行星大距的候选日，
// 再用二次插值在候选日附近精化极值时刻，把日采样候选升级为可展示的时分。
// 精化只依赖已有日采样，不额外发 JPL 请求，避免在同步路径里放大延迟和失败面。
// geometry.precision 为 "refined"（已精化）或 "daily_sample"（候选日，未精化）。
func planetaryEvents(samples map[string][]EphemerisSample, verifiedAt time.Time) []Event {
	var events []Event
	planets := []string{"mercury", "venus", "mars", "jupiter", "saturn", "uranus", "neptune"}
	for i, left := range planets {
		for _, right := range planets[i+1:] {
			for _, index := range localMinima(samples[left], samples[right], 5) {
				refinedAt, refinedSep, ok := refineQuadratic(
					samples[left], samples[right], index,
					func(l, r EphemerisSample) float64 { return angularDistance(l, r) },
				)
				var at time.Time
				precision := "daily_sample"
				geom := map[string]any{"objects": []string{left, right}, "precision": precision}
				if ok {
					at = refinedAt
					precision = "refined"
					geom["separationDegrees"] = refinedSep
					geom["precision"] = precision
					geom["refinedAt"] = refinedAt.UTC().Format("2006-01-02T15:04:05")
				} else {
					at = samples[left][index].Epoch
					geom["separationDegrees"] = angularDistance(samples[left][index], samples[right][index])
				}
				events = append(events, planetaryEvent("planetary_conjunction", at, verifiedAt, geom,
					fmt.Sprintf("%s合%s", planetChinese(left), planetChinese(right)),
					strings.ToUpper(left)+"-"+strings.ToUpper(right)+" CONJUNCTION"))
			}
		}
	}
	for _, planet := range []string{"mars", "jupiter", "saturn", "uranus", "neptune"} {
		for _, index := range oppositionCrossings(samples[planet], samples["sun"]) {
			refinedAt, _, ok := refineQuadratic(
				samples[planet], samples["sun"], index,
				func(p, s EphemerisSample) float64 {
					return math.Abs(signedAngleDegrees(signedLongitudeDifference(p, s) - 180))
				},
			)
			var at time.Time
			precision := "daily_sample"
			geom := map[string]any{"object": planet, "precision": precision}
			if ok {
				at = refinedAt
				precision = "refined"
				geom["precision"] = precision
				geom["refinedAt"] = refinedAt.UTC().Format("2006-01-02T15:04:05")
			} else {
				at = samples[planet][index].Epoch
			}
			events = append(events, planetaryEvent("planetary_opposition", at, verifiedAt, geom,
				planetChinese(planet)+"冲日", strings.ToUpper(planet)+" OPPOSITION"))
		}
	}
	for _, planet := range []string{"mercury", "venus"} {
		for _, index := range localMaxima(samples[planet], samples["sun"]) {
			refinedAt, refinedSep, ok := refineQuadratic(
				samples[planet], samples["sun"], index,
				func(p, s EphemerisSample) float64 { return math.Abs(signedLongitudeDifference(p, s)) },
			)
			var at time.Time
			precision := "daily_sample"
			geom := map[string]any{"object": planet, "precision": precision}
			if ok {
				at = refinedAt
				precision = "refined"
				separation := signedLongitudeDifference(samples[planet][index], samples["sun"][index])
				direction, chinese := "west", "西大距"
				if separation > 0 {
					direction, chinese = "east", "东大距"
				}
				geom["elongationDegrees"] = refinedSep
				geom["direction"] = direction
				geom["precision"] = precision
				geom["refinedAt"] = refinedAt.UTC().Format("2006-01-02T15:04:05")
				events = append(events, planetaryEvent("planetary_elongation", at, verifiedAt, geom,
					planetChinese(planet)+chinese, strings.ToUpper(planet)+" GREATEST "+strings.ToUpper(direction)+" ELONGATION"))
			} else {
				at = samples[planet][index].Epoch
				separation := signedLongitudeDifference(samples[planet][index], samples["sun"][index])
				direction, chinese := "west", "西大距"
				if separation > 0 {
					direction, chinese = "east", "东大距"
				}
				geom["elongationDegrees"] = math.Abs(separation)
				geom["direction"] = direction
				events = append(events, planetaryEvent("planetary_elongation", at, verifiedAt, geom,
					planetChinese(planet)+chinese, strings.ToUpper(planet)+" GREATEST "+strings.ToUpper(direction)+" ELONGATION"))
			}
		}
	}
	return events
}

// refineQuadratic 在候选日 index 附近用等距三点二次插值求极值时刻。
// 返回 (refinedAt, refinedValue, ok)。ok=false 表示无法精化（边界或数据不足）。
// 等距三点抛物线极值偏移公式：t* - t0 = h * (yPrev - yNext) / (2 * (yPrev - 2*y0 + yNext))
// 其中 h 是采样步长（日采样为 1 天）。
func refineQuadratic(left, right []EphemerisSample, index int, value func(l, r EphemerisSample) float64) (time.Time, float64, bool) {
	if index <= 0 || index+1 >= len(left) || index+1 >= len(right) {
		return time.Time{}, 0, false
	}
	t0 := left[index].Epoch
	h := t0.Sub(left[index-1].Epoch).Hours() / 24 // 采样步长（天）
	if math.Abs(h) < 1e-9 {
		return t0, value(left[index], right[index]), true
	}
	yPrev := value(left[index-1], right[index-1])
	y0 := value(left[index], right[index])
	yNext := value(left[index+1], right[index+1])
	denom := yPrev - 2*y0 + yNext
	if denom == 0 {
		return t0, y0, true
	}
	// 极值偏移（天）：t* - t0 = h * (yPrev - yNext) / (2 * denom)
	offset := h * (yPrev - yNext) / (2 * denom)
	// 限制偏移在 ±0.5 步长内，避免插值发散
	if math.Abs(offset) > math.Abs(h)*0.5 {
		offset = 0
	}
	refinedAt := t0.Add(time.Duration(offset * 24 * float64(time.Hour)))
	// 用插值公式重新计算极值：y* = y0 + (yNext-yPrev)/(2h) * offset + denom/(2h²) * offset²
	refinedValue := y0 + (yNext-yPrev)/(2*h)*offset + denom/(2*h*h)*offset*offset
	if math.IsNaN(refinedValue) || math.IsInf(refinedValue, 0) {
		refinedValue = y0
	}
	return refinedAt, refinedValue, true
}

func planetaryEvent(kind string, at, verifiedAt time.Time, geometry map[string]any, title, titleEN string) Event {
	raw, _ := json.Marshal(geometry)
	return Event{ID: fmt.Sprintf("%s-%s", kind, at.UTC().Format("20060102")), Kind: kind, Title: title, TitleEN: titleEN, StartsAt: at.UTC(), DateLabel: at.UTC().Format("2006年1月2日"), Summary: "基于 JPL Horizons 地心黄道坐标的日采样计算结果；候选日附近会继续精化。", Origin: "computed", SourceCode: "jpl_horizons_events", SourceURL: horizonsEventsEndpoint, VerifiedAt: verifiedAt.UTC().Format(time.DateOnly), Geometry: raw, Presentation: json.RawMessage(`{}`)}
}

func localMinima(left, right []EphemerisSample, threshold float64) []int {
	var found []int
	for i := 1; i+1 < len(left) && i+1 < len(right); i++ {
		before, current, after := angularDistance(left[i-1], right[i-1]), angularDistance(left[i], right[i]), angularDistance(left[i+1], right[i+1])
		if current <= before && current < after && current <= threshold {
			found = append(found, i)
		}
	}
	return found
}
func localMaxima(left, sun []EphemerisSample) []int {
	var found []int
	for i := 1; i+1 < len(left) && i+1 < len(sun); i++ {
		before, current, after := math.Abs(signedLongitudeDifference(left[i-1], sun[i-1])), math.Abs(signedLongitudeDifference(left[i], sun[i])), math.Abs(signedLongitudeDifference(left[i+1], sun[i+1]))
		if current >= before && current > after {
			found = append(found, i)
		}
	}
	return found
}
func oppositionCrossings(planet, sun []EphemerisSample) []int {
	var found []int
	for i := 1; i < len(planet) && i < len(sun); i++ {
		before, current := signedAngleDegrees(signedLongitudeDifference(planet[i-1], sun[i-1])-180), signedAngleDegrees(signedLongitudeDifference(planet[i], sun[i])-180)
		if (before < 0 && current >= 0) || (before > 0 && current <= 0) {
			found = append(found, i)
		}
	}
	return found
}
func angularDistance(a, b EphemerisSample) float64 {
	dot := a.XAU*b.XAU + a.YAU*b.YAU + a.ZAU*b.ZAU
	norm := math.Sqrt(a.XAU*a.XAU+a.YAU*a.YAU+a.ZAU*a.ZAU) * math.Sqrt(b.XAU*b.XAU+b.YAU*b.YAU+b.ZAU*b.ZAU)
	return math.Acos(math.Max(-1, math.Min(1, dot/norm))) * 180 / math.Pi
}
func signedLongitudeDifference(a, b EphemerisSample) float64 {
	return signedAngleDegrees(math.Atan2(a.YAU, a.XAU)*180/math.Pi - math.Atan2(b.YAU, b.XAU)*180/math.Pi)
}
func signedAngleDegrees(value float64) float64 {
	value = math.Mod(value+180, 360)
	if value < 0 {
		value += 360
	}
	return value - 180
}
func planetChinese(body string) string {
	return map[string]string{"mercury": "水星", "venus": "金星", "mars": "火星", "jupiter": "木星", "saturn": "土星", "uranus": "天王星", "neptune": "海王星"}[body]
}
