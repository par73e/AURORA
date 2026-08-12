package observatory

import (
	"math"
	"sort"
	"time"
)

// CalendarEvent 是与地理位置无关的基础天文事件。这里的时刻使用现有 Meeus 级
// 太阳/月球模型计算；日月食、流星雨和小天体仍由各自的校验/资料流程补充。
type CalendarEvent struct {
	Kind      string
	At        time.Time
	Value     float64
	Direction string
}

// CoreCalendarEvents 计算可由现有模型直接得到的月球周期、月球轨道与季节节点。
// 边界为 [from, to)，调用方可安全地按固定窗口每日重建这些 computed 记录。
func CoreCalendarEvents(from, to time.Time) []CalendarEvent {
	if !to.After(from) {
		return nil
	}
	from, to = from.UTC(), to.UTC()
	events := append([]CalendarEvent{}, lunarPhaseEvents(from, to)...)
	events = append(events, lunarOrbitEvents(from, to)...)
	events = append(events, seasonalEvents(from, to)...)
	sort.Slice(events, func(i, j int) bool { return events[i].At.Before(events[j].At) })
	return events
}

func lunarPhaseEvents(from, to time.Time) []CalendarEvent {
	targets := []struct {
		kind  string
		phase float64
	}{
		{"new_moon", 0},
		{"first_quarter", 90},
		{"full_moon", 180},
		{"last_quarter", 270},
	}
	var events []CalendarEvent
	for _, target := range targets {
		for _, at := range angularCrossings(from, to, 6*time.Hour, target.phase, func(t time.Time) float64 {
			return moonPhaseAt(t).Phase
		}) {
			events = append(events, CalendarEvent{Kind: target.kind, At: at, Value: target.phase})
		}
	}
	return events
}

func seasonalEvents(from, to time.Time) []CalendarEvent {
	targets := []struct {
		kind      string
		longitude float64
	}{
		{"march_equinox", 0},
		{"june_solstice", 90},
		{"september_equinox", 180},
		{"december_solstice", 270},
	}
	var events []CalendarEvent
	for _, target := range targets {
		for _, at := range angularCrossings(from, to, 24*time.Hour, target.longitude, func(t time.Time) float64 {
			return sunLongitude(julianCenturies(t))
		}) {
			events = append(events, CalendarEvent{Kind: target.kind, At: at, Value: target.longitude})
		}
	}
	return events
}

func lunarOrbitEvents(from, to time.Time) []CalendarEvent {
	const step = 6 * time.Hour
	var events []CalendarEvent
	previousAt := from
	_, previousLatitude, previousDistance := moonPosition(julianCenturies(previousAt))
	for at := from.Add(step); at.Before(to.Add(step)); at = at.Add(step) {
		currentAt := at
		if currentAt.After(to) {
			currentAt = to
		}
		_, currentLatitude, currentDistance := moonPosition(julianCenturies(currentAt))
		if previousLatitude < 0 && currentLatitude >= 0 {
			root := refineCalendarRoot(previousAt, currentAt, func(t time.Time) float64 {
				_, latitude, _ := moonPosition(julianCenturies(t))
				return latitude
			})
			events = append(events, CalendarEvent{Kind: "ascending_node", At: root, Direction: "ascending"})
		}
		if previousLatitude > 0 && currentLatitude <= 0 {
			root := refineCalendarRoot(previousAt, currentAt, func(t time.Time) float64 {
				_, latitude, _ := moonPosition(julianCenturies(t))
				return latitude
			})
			events = append(events, CalendarEvent{Kind: "descending_node", At: root, Direction: "descending"})
		}

		nextAt := currentAt.Add(step)
		if nextAt.After(to) {
			nextAt = to
		}
		if nextAt.After(currentAt) {
			_, _, nextDistance := moonPosition(julianCenturies(nextAt))
			if previousDistance > currentDistance && currentDistance <= nextDistance {
				at, distance := refineDistanceExtremum(previousAt, nextAt, true)
				events = append(events, CalendarEvent{Kind: "lunar_perigee", At: at, Value: distance})
			}
			if previousDistance < currentDistance && currentDistance >= nextDistance {
				at, distance := refineDistanceExtremum(previousAt, nextAt, false)
				events = append(events, CalendarEvent{Kind: "lunar_apogee", At: at, Value: distance})
			}
		}
		previousAt, previousLatitude, previousDistance = currentAt, currentLatitude, currentDistance
	}
	return events
}

// angularCrossings 利用角度的连续向前演化定位某个黄经/相位目标；步长远小于对应周期，
// 因而不会跨过两个同类事件。
func angularCrossings(from, to time.Time, step time.Duration, target float64, value func(time.Time) float64) []time.Time {
	previousAt := from
	previous := value(previousAt)
	unwrapped := previous
	var events []time.Time
	for at := from.Add(step); at.Before(to.Add(step)); at = at.Add(step) {
		currentAt := at
		if currentAt.After(to) {
			currentAt = to
		}
		current := value(currentAt)
		delta := signedAngle(current - previous)
		nextUnwrapped := unwrapped + delta
		firstCycle := math.Ceil((unwrapped - target) / 360)
		for cycle := firstCycle; target+360*cycle <= nextUnwrapped+1e-9; cycle++ {
			goal := target + 360*cycle
			if goal+1e-9 < unwrapped {
				continue
			}
			events = append(events, refineAngularRoot(previousAt, currentAt, target, value))
		}
		previousAt, previous, unwrapped = currentAt, current, nextUnwrapped
	}
	return events
}

func refineAngularRoot(left, right time.Time, target float64, value func(time.Time) float64) time.Time {
	for range 32 {
		middle := left.Add(right.Sub(left) / 2)
		if signedAngle(value(middle)-target) >= 0 {
			right = middle
		} else {
			left = middle
		}
	}
	return left.Add(right.Sub(left) / 2).UTC()
}

func refineCalendarRoot(left, right time.Time, value func(time.Time) float64) time.Time {
	leftValue := value(left)
	for range 32 {
		middle := left.Add(right.Sub(left) / 2)
		if (value(middle) >= 0) == (leftValue >= 0) {
			left = middle
			leftValue = value(left)
		} else {
			right = middle
		}
	}
	return left.Add(right.Sub(left) / 2).UTC()
}

func refineDistanceExtremum(left, right time.Time, minimum bool) (time.Time, float64) {
	for range 36 {
		span := right.Sub(left)
		first := left.Add(span / 3)
		second := right.Add(-span / 3)
		_, _, firstDistance := moonPosition(julianCenturies(first))
		_, _, secondDistance := moonPosition(julianCenturies(second))
		if (firstDistance < secondDistance) == minimum {
			right = second
		} else {
			left = first
		}
	}
	at := left.Add(right.Sub(left) / 2).UTC()
	_, _, distance := moonPosition(julianCenturies(at))
	return at, distance
}

func signedAngle(value float64) float64 {
	value = math.Mod(value+180, 360)
	if value < 0 {
		value += 360
	}
	return value - 180
}
