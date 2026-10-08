package observatory

import (
	"math"
	"time"
)

const eclipseHorizonAltitude = -0.3          // NASA local-circumstances convention
const eclipseEdgeUncertainty = 2.0 / 6378.14 // ~2 km lunar-limb uncertainty in Earth-radius units

// NASA/GSFC's published Besselian polynomials describe the Moon's shadow on
// Earth's fundamental plane. This implementation evaluates those coefficients
// directly and locates local contacts by bracketed root finding.
type besselianModel struct {
	day                            time.Time
	c                              map[string]float64
	latitude, longitude, elevation float64
}

type eclipsePoint struct {
	t, margin, centralMargin, distance, l1, l2, altitude, azimuth float64
}

func solveBesselianEclipse(event EventInput, latitude, longitude float64, loc *time.Location) EventVisibility {
	raw, ok := event.Geometry["besselian"].(map[string]any)
	if !ok || latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 || math.IsNaN(event.Elevation) {
		return EventVisibility{Status: "not_calculated", Reason: "NASA 贝塞尔根数或观测坐标无效。"}
	}
	keys := []string{"dt", "t0", "tmin", "tmax", "tan_f1", "tan_f2"}
	for _, prefix := range []string{"x", "y"} {
		for i := 0; i <= 3; i++ {
			keys = append(keys, prefix+string(rune('0'+i)))
		}
	}
	for _, prefix := range []string{"d", "mu", "l1", "l2"} {
		for i := 0; i <= 2; i++ {
			keys = append(keys, prefix+string(rune('0'+i)))
		}
	}
	c := make(map[string]float64, len(keys))
	for _, key := range keys {
		value, valid := raw[key].(float64)
		if !valid || math.IsNaN(value) || math.IsInf(value, 0) {
			return EventVisibility{Status: "not_calculated", Reason: "NASA 贝塞尔根数不完整。"}
		}
		c[key] = value
	}
	if c["tmax"] <= c["tmin"] || c["tmax"]-c["tmin"] > 12 || math.Abs(event.Elevation) > 10000 {
		return EventVisibility{Status: "not_calculated", Reason: "NASA 贝塞尔根数的计算范围无效。"}
	}
	day := time.Date(event.StartsAt.Year(), event.StartsAt.Month(), event.StartsAt.Day(), 0, 0, 0, 0, time.UTC)
	if text, ok := event.Geometry["besselianDay"].(string); ok {
		parsed, err := time.Parse(time.DateOnly, text)
		if err != nil {
			return EventVisibility{Status: "not_calculated", Reason: "NASA 贝塞尔根数的日期无效。"}
		}
		day = parsed
	}
	model := besselianModel{day: day, c: c, latitude: latitude, longitude: longitude, elevation: event.Elevation}
	const step = 1.0 / 60.0
	var samples []eclipsePoint
	bestIndex := -1
	for t := c["tmin"]; t < c["tmax"]; t = math.Min(t+step, c["tmax"]) {
		point := model.at(t)
		samples = append(samples, point)
		if bestIndex < 0 || point.margin > samples[bestIndex].margin {
			bestIndex = len(samples) - 1
		}
	}
	samples = append(samples, model.at(c["tmax"]))
	if samples[len(samples)-1].margin > samples[bestIndex].margin {
		bestIndex = len(samples) - 1
	}
	left := math.Max(c["tmin"], samples[bestIndex].t-step)
	right := math.Min(c["tmax"], samples[bestIndex].t+step)
	for i := 0; i < 45; i++ {
		a := left + (right-left)/3
		b := right - (right-left)/3
		if model.at(a).margin < model.at(b).margin {
			left = a
		} else {
			right = b
		}
	}
	peak := model.at((left + right) / 2)
	if math.Abs(peak.margin) <= eclipseEdgeUncertainty {
		return EventVisibility{Status: "not_calculated", Reason: "此地点接近日食半影边缘，约 2 公里内的月面轮廓误差不足以可靠判定。"}
	}
	if peak.margin < 0 {
		return EventVisibility{Status: "not_visible", Reason: "此次日食的月球半影未经过当前地点。"}
	}
	if samples[0].margin > 0 || samples[len(samples)-1].margin > 0 {
		return EventVisibility{Status: "not_calculated", Reason: "当地食相超出 NASA 贝塞尔根数的有效时段。"}
	}
	first, last := c["tmin"], c["tmax"]
	for i := 1; i < len(samples); i++ {
		if samples[i-1].margin <= 0 && samples[i].margin > 0 {
			first = model.root(samples[i-1].t, samples[i].t, false)
		}
		if samples[i-1].margin > 0 && samples[i].margin <= 0 {
			last = model.root(samples[i-1].t, samples[i].t, false)
		}
	}
	if first >= last {
		return EventVisibility{Status: "not_calculated", Reason: "未能求出本地日食接触时刻。"}
	}
	begin, end := model.at(first), model.at(last)
	contacts := &EclipseContacts{
		PartialBegin: model.utc(first).Format(time.RFC3339), Peak: model.utc(peak.t).Format(time.RFC3339), PartialEnd: model.utc(last).Format(time.RFC3339),
		PartialBeginVisible: begin.altitude > eclipseHorizonAltitude, PeakVisible: peak.altitude > eclipseHorizonAltitude, PartialEndVisible: end.altitude > eclipseHorizonAltitude,
		Kind: "partial",
	}
	if peak.centralMargin > 0 {
		if peak.l2 < 0 {
			contacts.Kind = "total"
		} else {
			contacts.Kind = "annular"
		}
		centralFirst, centralLast := peak.t, peak.t
		if model.at(first).centralMargin <= 0 && model.at(last).centralMargin <= 0 {
			centralFirst = model.root(first, peak.t, true)
			centralLast = model.root(peak.t, last, true)
		}
		if centralFirst < peak.t && centralLast > peak.t {
			startText, endText := model.utc(centralFirst).Format(time.RFC3339), model.utc(centralLast).Format(time.RFC3339)
			startVisible, endVisible := model.at(centralFirst).altitude > eclipseHorizonAltitude, model.at(centralLast).altitude > eclipseHorizonAltitude
			contacts.CentralBegin, contacts.CentralEnd = &startText, &endText
			contacts.CentralBeginVisible, contacts.CentralEndVisible = &startVisible, &endVisible
		}
	}
	denominator := peak.l1 + peak.l2
	if denominator <= 0 {
		return EventVisibility{Status: "not_calculated", Reason: "NASA 贝塞尔根数的视圆盘半径无效。"}
	}
	magnitude := math.Max(0, (peak.l1-peak.distance)/denominator)
	if peak.centralMargin > 0 {
		magnitude = (peak.l1 - peak.l2) / denominator
	}
	contacts.Magnitude = math.Round(magnitude*10000) / 10000
	contacts.ObscurationPercent = math.Round(eclipseObscuration(peak.distance, denominator/2, (peak.l1-peak.l2)/2)*10) / 10
	if contacts.Kind != "total" {
		contacts.ObscurationPercent = math.Min(99.9, contacts.ObscurationPercent)
	}
	// NASA's local-circumstances convention allows the solar centre to be
	// 0.3 degrees below the geometric horizon (standard near-horizon refraction).
	var visible []eclipsePoint
	for t := first; t < last; t = math.Min(t+step, last) {
		point := model.at(t)
		if point.altitude > eclipseHorizonAltitude {
			visible = append(visible, point)
		}
	}
	if end.altitude > eclipseHorizonAltitude {
		visible = append(visible, end)
	}
	if len(visible) == 0 {
		return EventVisibility{Status: "not_visible", Reason: "日食发生时，当前地点的太阳位于地平线下。", EclipseContacts: contacts}
	}
	best := visible[0]
	for _, point := range visible[1:] {
		if point.margin > best.margin {
			best = point
		}
	}
	windowStartT, windowEndT := visible[0].t, visible[len(visible)-1].t
	if previous := math.Max(first, windowStartT-step); previous < windowStartT && model.at(previous).altitude <= eclipseHorizonAltitude {
		windowStartT = model.horizonRoot(previous, windowStartT, eclipseHorizonAltitude)
	}
	if next := math.Min(last, windowEndT+step); next > windowEndT && model.at(next).altitude <= eclipseHorizonAltitude {
		windowEndT = model.horizonRoot(windowEndT, next, eclipseHorizonAltitude)
	}
	if peak.altitude > eclipseHorizonAltitude && peak.margin > best.margin {
		best = peak
	}
	if horizonPoint := model.at(windowEndT); horizonPoint.margin > best.margin {
		best = horizonPoint
	}
	if horizonPoint := model.at(windowStartT); horizonPoint.margin > best.margin {
		best = horizonPoint
	}
	bestAt := model.utc(best.t).In(loc).Format(time.RFC3339)
	windowStart := model.utc(windowStartT).In(loc).Format(time.RFC3339)
	windowEnd := model.utc(windowEndT).In(loc).Format(time.RFC3339)
	status, reason := "observable", "此次日食在当地可见；观测全程须使用合格的太阳滤镜。"
	if best.altitude < 10 {
		status, reason = "limited", "此次日食在当地可见，但太阳位置较低；观测全程须使用合格的太阳滤镜。"
	}
	return EventVisibility{Status: status, BestAt: &bestAt, WindowStart: &windowStart, WindowEnd: &windowEnd, AzimuthDegrees: &best.azimuth, AltitudeDegrees: &best.altitude, Reason: reason, EclipseContacts: contacts}
}

func (m besselianModel) utc(t float64) time.Time {
	hours := m.c["t0"] + t - m.c["dt"]/3600
	return m.day.Add(time.Duration(math.Round(hours*3600)) * time.Second)
}

func (m besselianModel) at(t float64) eclipsePoint {
	poly := func(prefix string, degree int) float64 {
		value := 0.0
		for i := degree; i >= 0; i-- {
			value = value*t + m.c[prefix+string(rune('0'+i))]
		}
		return value
	}
	x, y := poly("x", 3), poly("y", 3)
	d := deg2rad(poly("d", 2))
	h := deg2rad(poly("mu", 2) + m.longitude - m.c["dt"]*15.004/3600)
	phi := deg2rad(m.latitude)
	// WGS84 geodetic latitude and height to geocentric coordinates, expressed
	// in the equatorial Earth-radius units used by NASA's Besselian elements.
	const a, besselianRadius = 6378137.0, 6378140.0
	const flattening = 1.0 / 298.257223563
	eccentricitySquared := flattening * (2 - flattening)
	primeVertical := a / math.Sqrt(1-eccentricitySquared*math.Sin(phi)*math.Sin(phi))
	rhoSin := (primeVertical*(1-eccentricitySquared) + m.elevation) * math.Sin(phi) / besselianRadius
	rhoCos := (primeVertical + m.elevation) * math.Cos(phi) / besselianRadius
	xi := rhoCos * math.Sin(h)
	eta := rhoSin*math.Cos(d) - rhoCos*math.Cos(h)*math.Sin(d)
	zeta := rhoSin*math.Sin(d) + rhoCos*math.Cos(h)*math.Cos(d)
	distance := math.Hypot(x-xi, y-eta)
	l1 := poly("l1", 2) - zeta*m.c["tan_f1"]
	l2 := poly("l2", 2) - zeta*m.c["tan_f2"]
	altitude := rad2deg(math.Asin(clampUnit(math.Sin(phi)*math.Sin(d) + math.Cos(phi)*math.Cos(d)*math.Cos(h))))
	east := -math.Cos(d) * math.Sin(h)
	north := math.Sin(d)*math.Cos(phi) - math.Cos(d)*math.Sin(phi)*math.Cos(h)
	azimuth := math.Mod(rad2deg(math.Atan2(east, north))+360, 360)
	return eclipsePoint{t: t, margin: l1 - distance, centralMargin: math.Abs(l2) - distance, distance: distance, l1: l1, l2: l2, altitude: altitude, azimuth: azimuth}
}

func (m besselianModel) root(left, right float64, central bool) float64 {
	value := func(t float64) float64 {
		p := m.at(t)
		if central {
			return p.centralMargin
		}
		return p.margin
	}
	leftSign := value(left) > 0
	for i := 0; i < 45; i++ {
		mid := (left + right) / 2
		if (value(mid) > 0) == leftSign {
			left = mid
		} else {
			right = mid
		}
	}
	return (left + right) / 2
}

func (m besselianModel) horizonRoot(left, right, altitude float64) float64 {
	leftAbove := m.at(left).altitude > altitude
	for i := 0; i < 40; i++ {
		mid := (left + right) / 2
		if (m.at(mid).altitude > altitude) == leftAbove {
			left = mid
		} else {
			right = mid
		}
	}
	return (left + right) / 2
}

func eclipseObscuration(distance, sunRadius, moonRadius float64) float64 {
	if distance >= sunRadius+moonRadius {
		return 0
	}
	if distance <= math.Abs(sunRadius-moonRadius) {
		if moonRadius >= sunRadius {
			return 100
		}
		return 100 * moonRadius * moonRadius / (sunRadius * sunRadius)
	}
	a := math.Acos(clampUnit((distance*distance + sunRadius*sunRadius - moonRadius*moonRadius) / (2 * distance * sunRadius)))
	b := math.Acos(clampUnit((distance*distance + moonRadius*moonRadius - sunRadius*sunRadius) / (2 * distance * moonRadius)))
	area := sunRadius*sunRadius*a + moonRadius*moonRadius*b - 0.5*math.Sqrt(math.Max(0, (-distance+sunRadius+moonRadius)*(distance+sunRadius-moonRadius)*(distance-sunRadius+moonRadius)*(distance+sunRadius+moonRadius)))
	return 100 * area / (math.Pi * sunRadius * sunRadius)
}
