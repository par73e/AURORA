package astronomyevent

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"aurora/backend/internal/observatory"
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
				geom["positions"] = eclipticPositions(samples, []string{left, right}, index, at)
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
			geom["positions"] = eclipticPositions(samples, []string{planet}, index, at)
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
				geom["positions"] = eclipticPositions(samples, []string{planet}, index, at)
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
				geom["positions"] = eclipticPositions(samples, []string{planet}, index, at)
				events = append(events, planetaryEvent("planetary_elongation", at, verifiedAt, geom,
					planetChinese(planet)+chinese, strings.ToUpper(planet)+" GREATEST "+strings.ToUpper(direction)+" ELONGATION"))
			}
		}
	}
	for _, planet := range []string{"mercury", "venus", "mars", "jupiter", "saturn", "uranus", "neptune"} {
		for _, index := range localMinima(samples[planet], samples["sun"], 5) {
			at, separation, refined := refineQuadratic(samples[planet], samples["sun"], index, angularDistance)
			precision := "daily_sample"
			if !refined {
				at, separation = samples[planet][index].Epoch, angularDistance(samples[planet][index], samples["sun"][index])
			} else {
				precision = "refined"
			}
			relation, suffix := "conjunction", "合日"
			if planet == "mercury" || planet == "venus" {
				if vectorDistance(samples[planet][index]) < vectorDistance(samples["sun"][index]) {
					relation, suffix = "inferior", "下合"
				} else {
					relation, suffix = "superior", "上合"
				}
			}
			geometry := map[string]any{
				"object": planet, "relation": relation, "separationDegrees": separation, "precision": precision,
				"positions": eclipticPositions(samples, []string{planet}, index, at),
			}
			if refined {
				geometry["refinedAt"] = at.UTC().Format(time.RFC3339)
			}
			events = append(events, planetaryEvent("planetary_solar_conjunction", at, verifiedAt, geometry,
				planetChinese(planet)+suffix, strings.ToUpper(planet)+" "+strings.ToUpper(relation)+" CONJUNCTION"))
		}
	}
	return events
}

// moonConjunctionEvents 使用本地月球模型与缓存的 JPL 行星星历寻找月合。
// 月球位置不写入数据库事实表；读取时会在当地可见性求解器中按同一时刻重新计算。
func moonConjunctionEvents(samples map[string][]EphemerisSample, verifiedAt time.Time) []Event {
	var events []Event
	for _, planet := range []string{"mercury", "venus", "mars", "jupiter", "saturn"} {
		planetSamples := samples[planet]
		if len(planetSamples) < 3 {
			continue
		}
		moonSamples := make([]EphemerisSample, 0, len(planetSamples))
		for _, sample := range planetSamples {
			coordinates := observatory.MoonEclipticCoordinates(sample.Epoch)
			lon, lat := coordinates.LongitudeDegrees*math.Pi/180, coordinates.LatitudeDegrees*math.Pi/180
			moonSamples = append(moonSamples, EphemerisSample{
				Body: "moon", Epoch: sample.Epoch,
				XAU: math.Cos(lat) * math.Cos(lon), YAU: math.Cos(lat) * math.Sin(lon), ZAU: math.Sin(lat),
			})
		}
		for _, index := range localMinima(moonSamples, planetSamples, 5) {
			at, separation, ok := refineQuadratic(moonSamples, planetSamples, index, angularDistance)
			precision := "daily_sample"
			if !ok {
				at, separation = planetSamples[index].Epoch, angularDistance(moonSamples[index], planetSamples[index])
			} else {
				precision = "refined"
			}
			geometry := map[string]any{
				"objects":           []string{"moon", planet},
				"separationDegrees": separation,
				"precision":         precision,
				"positions":         eclipticPositions(samples, []string{planet}, index, at),
			}
			if precision == "refined" {
				geometry["refinedAt"] = at.UTC().Format(time.RFC3339)
			}
			events = append(events, planetaryEvent("moon_conjunction", at, verifiedAt, geometry,
				fmt.Sprintf("%s合月", planetChinese(planet)), strings.ToUpper(planet)+"-MOON CONJUNCTION"))
		}
	}
	events = append(events, moonBrightObjectConjunctionEvents(samples["sun"], verifiedAt)...)
	return events
}

// moonBrightObjectConjunctionEvents 使用稳定的 J2000 赤道坐标补齐月合亮星与星团。
// 这些目录目标不依赖实时外网；事件寻找在黄道坐标中完成，而地点可见性在读取时
// 仍按赤道坐标重新计算，避免把星表资料误当成行星 JPL 星历。
func moonBrightObjectConjunctionEvents(reference []EphemerisSample, verifiedAt time.Time) []Event {
	targets := []struct {
		id, title, titleEN string
		ra, dec            float64
	}{
		{"sirius", "天狼星", "SIRIUS", 101.287, -16.716},
		{"arcturus", "大角星", "ARCTURUS", 213.915, 19.182},
		{"vega", "织女星", "VEGA", 279.235, 38.784},
		{"pleiades", "昴星团", "PLEIADES", 56.75, 24.116},
		{"hyades", "毕星团", "HYADES", 66.75, 15.87},
	}
	if len(reference) < 3 {
		return nil
	}
	var events []Event
	for _, target := range targets {
		star := observatory.EquatorialCoordinates{RightAscensionDegrees: target.ra, DeclinationDegrees: target.dec}
		moonSamples, targetSamples := make([]EphemerisSample, 0, len(reference)), make([]EphemerisSample, 0, len(reference))
		for _, sample := range reference {
			moon := observatory.MoonEclipticCoordinates(sample.Epoch)
			moonLon, moonLat := degRadians(moon.LongitudeDegrees), degRadians(moon.LatitudeDegrees)
			object := observatory.EquatorialToEclipticCoordinates(star, sample.Epoch)
			objectLon, objectLat := degRadians(object.LongitudeDegrees), degRadians(object.LatitudeDegrees)
			moonSamples = append(moonSamples, EphemerisSample{Body: "moon", Epoch: sample.Epoch, XAU: math.Cos(moonLat) * math.Cos(moonLon), YAU: math.Cos(moonLat) * math.Sin(moonLon), ZAU: math.Sin(moonLat)})
			targetSamples = append(targetSamples, EphemerisSample{Body: target.id, Epoch: sample.Epoch, XAU: math.Cos(objectLat) * math.Cos(objectLon), YAU: math.Cos(objectLat) * math.Sin(objectLon), ZAU: math.Sin(objectLat)})
		}
		for _, index := range localMinima(moonSamples, targetSamples, 5) {
			at, separation, refined := refineQuadratic(moonSamples, targetSamples, index, angularDistance)
			precision := "daily_sample"
			if !refined {
				at, separation = reference[index].Epoch, angularDistance(moonSamples[index], targetSamples[index])
			} else {
				precision = "refined"
			}
			geometry := map[string]any{
				"objects":           []string{"moon", target.id},
				"separationDegrees": separation,
				"precision":         precision,
				"positions": map[string]map[string]float64{target.id: {
					"rightAscensionDegrees": target.ra,
					"declinationDegrees":    target.dec,
				}},
			}
			if refined {
				geometry["refinedAt"] = at.UTC().Format(time.RFC3339)
			}
			events = append(events, planetaryEvent("moon_conjunction", at, verifiedAt, geometry,
				fmt.Sprintf("%s合月", target.title), target.titleEN+"-MOON CONJUNCTION"))
		}
	}
	return events
}

func degRadians(value float64) float64 { return value * math.Pi / 180 }

// orbitalDistanceEvents 从以太阳为中心的 JPL 向量中寻找各行星近日点与远日点。
// 这些是重要轨道几何节点，并不承诺某地点在该瞬间可见，因此本地层标为 non_visual。
func orbitalDistanceEvents(samples map[string][]EphemerisSample, verifiedAt time.Time) []Event {
	var events []Event
	for _, planet := range []string{"mercury", "venus", "mars", "jupiter", "saturn", "uranus", "neptune"} {
		series := samples[planet+"_heliocentric"]
		for index := 1; index+1 < len(series); index++ {
			before, current, after := vectorDistance(series[index-1]), vectorDistance(series[index]), vectorDistance(series[index+1])
			kind, titleSuffix, titleENSuffix := "", "", ""
			if current <= before && current < after {
				kind, titleSuffix, titleENSuffix = "planetary_perihelion", "近日点", "PERIHELION"
			} else if current >= before && current > after {
				kind, titleSuffix, titleENSuffix = "planetary_aphelion", "远日点", "APHELION"
			} else {
				continue
			}
			at, distance, refined := refineQuadratic(series, series, index, func(left, _ EphemerisSample) float64 { return vectorDistance(left) })
			precision := "daily_sample"
			if !refined {
				at, distance = series[index].Epoch, current
			} else {
				precision = "refined"
			}
			geometry := map[string]any{"object": planet, "distanceAU": distance, "precision": precision}
			if refined {
				geometry["refinedAt"] = at.UTC().Format(time.RFC3339)
			}
			events = append(events, planetaryEvent(kind, at, verifiedAt, geometry,
				planetChinese(planet)+titleSuffix, strings.ToUpper(planet)+" "+titleENSuffix))
		}
	}
	return events
}

func vectorDistance(sample EphemerisSample) float64 {
	return math.Sqrt(sample.XAU*sample.XAU + sample.YAU*sample.YAU + sample.ZAU*sample.ZAU)
}

// multiPlanetAlignmentEvents 只记录具有明确定义的几何事件：至少三颗行星在地心黄道上
// 落入 35° 以内，并以该紧凑角距的局部最小日作为事件时刻。它不使用“几星连珠”营销名称。
func multiPlanetAlignmentEvents(samples map[string][]EphemerisSample, verifiedAt time.Time) []Event {
	planets := []string{"mercury", "venus", "mars", "jupiter", "saturn", "uranus", "neptune"}
	var events []Event
	for left := 0; left < len(planets); left++ {
		for middle := left + 1; middle < len(planets); middle++ {
			for right := middle + 1; right < len(planets); right++ {
				bodies := []string{planets[left], planets[middle], planets[right]}
				series := samples[bodies[0]]
				for index := 1; index+1 < len(series); index++ {
					before := planetarySpan(samples, bodies, index-1)
					current := planetarySpan(samples, bodies, index)
					after := planetarySpan(samples, bodies, index+1)
					if current > 35 || current > before || current >= after {
						continue
					}
					at := series[index].Epoch
					geometry := map[string]any{
						"objects": bodies, "spanDegrees": current, "precision": "daily_sample",
						"positions": eclipticPositions(samples, bodies, index, at),
					}
					events = append(events, planetaryEvent("multi_planet_alignment", at, verifiedAt, geometry,
						"多颗行星同场几何接近", "MULTI-PLANET ALIGNMENT"))
				}
			}
		}
	}
	return events
}

func planetarySpan(samples map[string][]EphemerisSample, bodies []string, index int) float64 {
	longitudes := make([]float64, 0, len(bodies))
	for _, body := range bodies {
		series := samples[body]
		if index < 0 || index >= len(series) {
			return math.Inf(1)
		}
		longitudes = append(longitudes, normDegrees(math.Atan2(series[index].YAU, series[index].XAU)*180/math.Pi))
	}
	sort.Float64s(longitudes)
	largestGap := 0.0
	for i := range longitudes {
		next := longitudes[(i+1)%len(longitudes)]
		if i == len(longitudes)-1 {
			next += 360
		}
		largestGap = math.Max(largestGap, next-longitudes[i])
	}
	return 360 - largestGap
}

func normDegrees(value float64) float64 {
	value = math.Mod(value, 360)
	if value < 0 {
		value += 360
	}
	return value
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
		return time.Time{}, 0, false
	}
	yPrev := value(left[index-1], right[index-1])
	y0 := value(left[index], right[index])
	yNext := value(left[index+1], right[index+1])
	if math.IsNaN(yPrev) || math.IsNaN(y0) || math.IsNaN(yNext) ||
		math.IsInf(yPrev, 0) || math.IsInf(y0, 0) || math.IsInf(yNext, 0) {
		return time.Time{}, 0, false
	}
	denom := yPrev - 2*y0 + yNext
	scale := math.Max(1, math.Max(math.Abs(yPrev), math.Max(math.Abs(y0), math.Abs(yNext))))
	if math.Abs(denom) <= 1e-12*scale {
		return time.Time{}, 0, false
	}
	// 极值偏移（天）：t* - t0 = h * (yPrev - yNext) / (2 * denom)
	offset := h * (yPrev - yNext) / (2 * denom)
	// 限制偏移在 ±0.5 步长内，避免插值发散
	if math.IsNaN(offset) || math.IsInf(offset, 0) || math.Abs(offset) > math.Abs(h)*0.5 {
		return time.Time{}, 0, false
	}
	refinedAt := t0.Add(time.Duration(offset * 24 * float64(time.Hour)))
	// 用插值公式重新计算极值：y* = y0 + (yNext-yPrev)/(2h) * offset + denom/(2h²) * offset²
	refinedValue := y0 + (yNext-yPrev)/(2*h)*offset + denom/(2*h*h)*offset*offset
	if math.IsNaN(refinedValue) || math.IsInf(refinedValue, 0) {
		return time.Time{}, 0, false
	}
	return refinedAt, refinedValue, true
}

func planetaryEvent(kind string, at, verifiedAt time.Time, geometry map[string]any, title, titleEN string) Event {
	raw, _ := json.Marshal(geometry)
	identifier := kind + "-" + at.UTC().Format("20060102")
	if object, ok := geometry["object"].(string); ok {
		identifier += "-" + object
	}
	if objects, ok := geometry["objects"].([]string); ok {
		identifier += "-" + strings.Join(objects, "-")
	}
	summary := "基于 JPL Horizons 地心黄道坐标的日采样计算结果。"
	if geometry["precision"] == "refined" {
		summary = "基于 JPL Horizons 日采样并以三点二次插值精化的计算结果；精度说明见来源。"
	}
	return Event{ID: identifier, Kind: kind, Title: title, TitleEN: titleEN, StartsAt: at.UTC(), DateLabel: at.UTC().Format("2006年1月2日"), Summary: summary, Origin: "computed", SourceCode: "jpl_horizons_events", SourceURL: horizonsEventsPageURL, VerifiedAt: verifiedAt.UTC().Format(time.DateOnly), Geometry: raw, Presentation: json.RawMessage(`{}`)}
}

// eclipticPositions 把事件附近的 JPL 样本序列化为供本地可见性计算使用的黄道坐标。
func eclipticPositions(samples map[string][]EphemerisSample, bodies []string, index int, at time.Time) map[string]map[string]float64 {
	positions := make(map[string]map[string]float64, len(bodies))
	for _, body := range bodies {
		series := samples[body]
		if index < 0 || index >= len(series) {
			continue
		}
		sample := interpolateSample(series, index, at)
		lon := math.Atan2(sample.YAU, sample.XAU) * 180 / math.Pi
		lat := math.Atan2(sample.ZAU, math.Hypot(sample.XAU, sample.YAU)) * 180 / math.Pi
		positions[body] = map[string]float64{"longitudeDegrees": signedAngleDegrees(lon), "latitudeDegrees": lat}
	}
	return positions
}

func interpolateSample(series []EphemerisSample, index int, at time.Time) EphemerisSample {
	base := series[index]
	if index+1 >= len(series) || !at.After(base.Epoch) {
		return base
	}
	next := series[index+1]
	span := next.Epoch.Sub(base.Epoch)
	if span <= 0 {
		return base
	}
	ratio := at.Sub(base.Epoch).Seconds() / span.Seconds()
	if ratio < 0 || ratio > 1 {
		return base
	}
	base.XAU += (next.XAU - base.XAU) * ratio
	base.YAU += (next.YAU - base.YAU) * ratio
	base.ZAU += (next.ZAU - base.ZAU) * ratio
	base.Epoch = at
	return base
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
