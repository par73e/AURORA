package observatory

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// EventInput 是给本地可见性求解器的最小事件输入。
// 它故意不依赖 astronomyevent 包，避免 observatory 反向依赖事件领域。
type EventInput struct {
	ID       string
	Kind     string
	StartsAt time.Time
	Geometry map[string]any // 行星事件的 object/objects、合的 separationDegrees 等
}

// EventVisibility 是天象事件在某地点的本地可见性结论。
// 对应设计文档中的 observable / limited / not_visible / non_visual 四态。
type EventVisibility struct {
	Status          string  // observable | limited | not_visible | non_visual | not_calculated
	BestAt          *string // RFC3339 本地时刻；non_visual/not_calculated 可为空
	WindowStart     *string // 最佳观测窗口起止（可观测/有限时填写）
	WindowEnd       *string
	AzimuthDegrees  *float64 // 最佳时刻方位角（以北为零、向东为正）
	AltitudeDegrees *float64 // 最佳时刻目标地平高度
	Reason          string   // 面向用户的一句话解释
}

// VisibilitySolver 把全球天文事件在读取时换算为某个地点的本地可见性。
// 它只依赖现有 observatory 内部坐标函数，不引入新外网依赖。
type VisibilitySolver struct {
	moons MoonProvider
}

// NewVisibilitySolver 构造求解器；moons 可为 nil（退化为 not_calculated）。
func NewVisibilitySolver(moons MoonProvider) *VisibilitySolver {
	return &VisibilitySolver{moons: moons}
}

// Solve 根据事件 Kind 选择合适的本地可见性求解器。
// 尚未接入专用求解器的类别返回 not_calculated，绝不把全球事件误标为本地可见。
func (s *VisibilitySolver) Solve(event EventInput, latitude, longitude float64, timezone string) EventVisibility {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	switch event.Kind {
	case "new_moon", "lunar_perigee", "lunar_apogee", "ascending_node", "descending_node",
		"march_equinox", "june_solstice", "september_equinox", "december_solstice",
		"planetary_perihelion", "planetary_aphelion", "planetary_solar_conjunction":
		// 季节节点、月球轨道节点是非视觉几何节点。
		return EventVisibility{Status: "non_visual", Reason: "这是重要的天文几何节点，不对应一个只在该瞬间可见的独立目标。"}
	case "full_moon", "first_quarter", "last_quarter":
		return s.solveMoonPhase(event, latitude, longitude, loc)
	case "lunar_eclipse":
		return s.solveLunarEclipse(event, latitude, longitude, loc)
	case "meteor_shower":
		return s.solveMeteorShower(event, latitude, longitude, loc)
	case "moon_conjunction", "planetary_conjunction", "planetary_opposition", "planetary_elongation", "multi_planet_alignment":
		return s.solvePlanetary(event, latitude, longitude, loc)
	case "solar_eclipse":
		// 日食只在路径覆盖区可见；用 NASA/GSFC 食类型 + 当地太阳高度判定。
		// 路径数据（Besselian 元素）解析接入后可精确判定；当前用食类型 + 当地白天判定。
		return s.solveSolarEclipse(event, latitude, longitude, loc)
	case "small_body_close_approach":
		return EventVisibility{Status: "not_calculated", Reason: "该近地小天体已同步最近掠过资料；需要对象星历后才能计算当地位置与亮度。"}
	default:
		return EventVisibility{Status: "not_calculated", Reason: "该事件的专用本地可见性求解器尚未接入；不会把全球事件误标为本地可见。"}
	}
}

// solveMoonPhase 用月球中天高度判断满月/上下弦的本地可见性。
func (s *VisibilitySolver) solveMoonPhase(event EventInput, latitude, longitude float64, loc *time.Location) EventVisibility {
	if s.moons == nil {
		return EventVisibility{Status: "not_calculated", Reason: "本地星历服务尚未配置。"}
	}
	day, err := s.moons.Day(latitude, longitude, 0, event.StartsAt, loc.String())
	if err != nil || day.Transit == nil {
		return EventVisibility{Status: "not_visible", Reason: "该本地日期内月球没有可用的中天窗口。"}
	}
	bestAt := time.Unix(*day.Transit, 0).UTC()
	altitude := s.moons.Altitude(latitude, longitude, 0, bestAt)
	var azimuth *float64
	if provider, ok := s.moons.(MoonAzimuthProvider); ok {
		value := provider.Azimuth(latitude, longitude, 0, bestAt)
		azimuth = &value
	}
	bestAtText := bestAt.In(loc).Format(time.RFC3339)
	windowStartText := bestAt.Add(-2 * time.Hour).In(loc).Format(time.RFC3339)
	windowEndText := bestAt.Add(2 * time.Hour).In(loc).Format(time.RFC3339)
	if altitude < 0 {
		return EventVisibility{Status: "not_visible", Reason: "该本地日期内月球最高点仍在地平线以下。"}
	}
	status, reason := "observable", "月球在本地日期内有较好的中天高度。"
	if altitude < 10 {
		status, reason = "limited", "月球中天高度较低，地平线遮挡和大气消光可能影响观测。"
	}
	return EventVisibility{
		Status:          status,
		BestAt:          &bestAtText,
		WindowStart:     &windowStartText,
		WindowEnd:       &windowEndText,
		AzimuthDegrees:  azimuth,
		AltitudeDegrees: &altitude,
		Reason:          reason,
	}
}

// solvePlanetary 把事件中缓存的 JPL 行星地心黄道坐标转为本地点的地平坐标，
// 在事件当日 ±12 小时窗口内采样目标高度与太阳高度，给出 observable/limited/not_visible。
// 合事件中的每一个目标都必须在地平线上，不能以其中较高的一个替代另一个。
func (s *VisibilitySolver) solvePlanetary(event EventInput, latitude, longitude float64, loc *time.Location) EventVisibility {
	objects := collectConjunctionObjects(event.Geometry)
	if len(objects) == 0 {
		object, ok := event.Geometry["object"].(string)
		if !ok {
			return EventVisibility{Status: "not_calculated", Reason: "缺少行星对象信息，无法做本地可见性判断。"}
		}
		objects = []string{object}
	}
	for _, object := range objects {
		if object == "moon" {
			if s.moons == nil {
				return EventVisibility{Status: "not_calculated", Reason: "本地月球星历服务尚未配置。"}
			}
			continue
		}
		if !eventObjectHasCoordinates(event.Geometry, object) {
			return EventVisibility{Status: "not_calculated", Reason: "事件缺少该目标的坐标，不能以其他天体位置代替。"}
		}
	}

	start := event.StartsAt.UTC().Add(-12 * time.Hour)
	end := event.StartsAt.UTC().Add(12 * time.Hour)
	bestAlt := -math.MaxFloat64 // 组合事件取全部目标中的最低高度
	bestObject := ""
	var bestAt time.Time
	bestSunAlt := math.MaxFloat64
	foundNight := false
	var windowStart, windowEnd *time.Time
	for at := start; !at.After(end); at = at.Add(30 * time.Minute) {
		sunAlt := EclipticToHorizontalAltitude("sun", latitude, longitude, at)
		// 要求太阳在暮光结束后（民用暮光 -6°）
		if sunAlt > -6 {
			continue
		}
		foundNight = true
		currentMinimumAlt := math.MaxFloat64
		currentLowestObject := objects[0]
		for _, object := range objects {
			alt := eventObjectAltitude(s, event.Geometry, object, latitude, longitude, at)
			if alt < currentMinimumAlt {
				currentMinimumAlt = alt
				currentLowestObject = object
			}
		}
		if currentMinimumAlt >= 0 {
			if windowStart == nil {
				value := at
				windowStart = &value
			}
			value := at
			windowEnd = &value
		}
		if currentMinimumAlt > bestAlt {
			bestAlt = currentMinimumAlt
			bestObject = currentLowestObject
			bestAt = at
			bestSunAlt = sunAlt
		}
	}

	if !foundNight {
		return EventVisibility{Status: "not_visible", Reason: "事件发生时本地为白昼或极昼，目标不可见。"}
	}
	if windowStart == nil {
		return EventVisibility{Status: "not_visible", Reason: "事件发生时至少有一个目标在当地地平线以下。"}
	}

	azimuth := eventObjectAzimuth(s, event.Geometry, bestObject, latitude, longitude, bestAt)
	bestAtText := bestAt.In(loc).Format(time.RFC3339)
	windowStartText, windowEndText := windowStart.In(loc).Format(time.RFC3339), windowEnd.In(loc).Format(time.RFC3339)
	altPtr := &bestAlt
	azPtr := &azimuth

	if bestAlt < 0 {
		return EventVisibility{Status: "not_visible", Reason: "事件发生时目标在本当地地平线以下。"}
	}
	if bestAlt < 5 || bestSunAlt > -6 {
		return EventVisibility{
			Status:          "limited",
			BestAt:          &bestAtText,
			AzimuthDegrees:  azPtr,
			AltitudeDegrees: altPtr,
			WindowStart:     &windowStartText,
			WindowEnd:       &windowEndText,
			Reason:          "目标高度较低或暮光较强，观测条件有限。",
		}
	}
	return EventVisibility{
		Status:          "observable",
		BestAt:          &bestAtText,
		AzimuthDegrees:  azPtr,
		AltitudeDegrees: altPtr,
		WindowStart:     &windowStartText,
		WindowEnd:       &windowEndText,
		Reason:          "本地夜间目标高度足够，适合观测。",
	}
}

func eventEclipticCoordinates(geometry map[string]any, object string) (EclipticCoordinates, bool) {
	positions, ok := geometry["positions"].(map[string]any)
	if !ok {
		return EclipticCoordinates{}, false
	}
	position, ok := positions[object].(map[string]any)
	if !ok {
		return EclipticCoordinates{}, false
	}
	lon, lonOK := position["longitudeDegrees"].(float64)
	lat, latOK := position["latitudeDegrees"].(float64)
	return EclipticCoordinates{LongitudeDegrees: lon, LatitudeDegrees: lat}, lonOK && latOK
}

func eventEquatorialCoordinates(geometry map[string]any, object string) (EquatorialCoordinates, bool) {
	positions, ok := geometry["positions"].(map[string]any)
	if !ok {
		return EquatorialCoordinates{}, false
	}
	position, ok := positions[object].(map[string]any)
	if !ok {
		return EquatorialCoordinates{}, false
	}
	ra, raOK := position["rightAscensionDegrees"].(float64)
	dec, decOK := position["declinationDegrees"].(float64)
	return EquatorialCoordinates{RightAscensionDegrees: ra, DeclinationDegrees: dec}, raOK && decOK
}

func eventObjectHasCoordinates(geometry map[string]any, object string) bool {
	if _, ok := eventEclipticCoordinates(geometry, object); ok {
		return true
	}
	_, ok := eventEquatorialCoordinates(geometry, object)
	return ok
}

func eventObjectAltitude(s *VisibilitySolver, geometry map[string]any, object string, latitude, longitude float64, at time.Time) float64 {
	if object == "moon" {
		return s.moons.Altitude(latitude, longitude, 0, at)
	}
	if coordinates, ok := eventEclipticCoordinates(geometry, object); ok {
		altitude, _ := EclipticCoordinatesToHorizontal(coordinates, latitude, longitude, at)
		return altitude
	}
	coordinates, _ := eventEquatorialCoordinates(geometry, object)
	altitude, _ := EquatorialCoordinatesToHorizontal(coordinates, latitude, longitude, at)
	return altitude
}

func eventObjectAzimuth(s *VisibilitySolver, geometry map[string]any, object string, latitude, longitude float64, at time.Time) float64 {
	if object == "moon" {
		if provider, ok := s.moons.(MoonAzimuthProvider); ok {
			return provider.Azimuth(latitude, longitude, 0, at)
		}
		_, azimuth := EclipticCoordinatesToHorizontal(MoonEclipticCoordinates(at), latitude, longitude, at)
		return azimuth
	}
	if coordinates, ok := eventEclipticCoordinates(geometry, object); ok {
		_, azimuth := EclipticCoordinatesToHorizontal(coordinates, latitude, longitude, at)
		return azimuth
	}
	coordinates, _ := eventEquatorialCoordinates(geometry, object)
	_, azimuth := EquatorialCoordinatesToHorizontal(coordinates, latitude, longitude, at)
	return azimuth
}

// collectConjunctionObjects 从 geometry 提取合事件的 objects 数组（planetary_conjunction）。
// 单 object 事件（planetary_opposition/planetary_elongation）返回空，仍走 object 字段。
func collectConjunctionObjects(geometry map[string]any) []string {
	if objects, ok := geometry["objects"].([]any); ok {
		out := make([]string, 0, len(objects))
		for _, object := range objects {
			if name, ok := object.(string); ok {
				out = append(out, name)
			}
		}
		return out
	}
	return nil
}

// solveLunarEclipse 按当地月球高度+NASA/GSFC 食类型/食分判定月食可见性。
// 月食可裸眼安全观看。食类型/食分来自 NASA/GSFC 解析（geometry.eclipseType/magnitude）；
// 扩展观测窗口（事件时刻 ±2 小时）内采样月球高度，找最佳时刻。
func (s *VisibilitySolver) solveLunarEclipse(event EventInput, latitude, longitude float64, loc *time.Location) EventVisibility {
	if s.moons == nil {
		return EventVisibility{Status: "not_calculated", Reason: "本地星历服务尚未配置。"}
	}
	eclipseType, _ := event.Geometry["eclipseType"].(string)
	if eclipseType == "" {
		eclipseType = "Partial"
	}
	magnitude, _ := event.Geometry["magnitude"].(string)

	// 扩展观测窗口（事件时刻 ±2 小时）内采样月球高度与太阳高度，
	// 找出"月球在地平上、太阳在地平下"的最佳时刻。
	start := event.StartsAt.UTC().Add(-2 * time.Hour)
	end := event.StartsAt.UTC().Add(2 * time.Hour)
	bestMoonAlt := -math.MaxFloat64
	var bestAt time.Time
	bestSunAlt := math.MaxFloat64
	found := false
	var windowStart, windowEnd *time.Time
	for at := start; !at.After(end); at = at.Add(15 * time.Minute) {
		moonAlt := s.moons.Altitude(latitude, longitude, 0, at)
		sunAlt := EclipticToHorizontalAltitude("sun", latitude, longitude, at)
		if moonAlt <= 0 || sunAlt > -6 {
			continue
		}
		found = true
		if windowStart == nil {
			value := at
			windowStart = &value
		}
		value := at
		windowEnd = &value
		if moonAlt > bestMoonAlt {
			bestMoonAlt = moonAlt
			bestAt = at
			bestSunAlt = sunAlt
		}
	}

	if !found {
		at := event.StartsAt.UTC()
		moonAlt := s.moons.Altitude(latitude, longitude, 0, at)
		if moonAlt <= 0 {
			return EventVisibility{Status: "not_visible", Reason: "月食发生时月球在当地地平线以下。"}
		}
		return EventVisibility{Status: "not_visible", Reason: "月食发生时当地为白昼，月球不可见。"}
	}

	bestAtText := bestAt.In(loc).Format(time.RFC3339)
	windowStartText, windowEndText := windowStart.In(loc).Format(time.RFC3339), windowEnd.In(loc).Format(time.RFC3339)
	altPtr := &bestMoonAlt
	magNote := ""
	if magnitude != "" {
		magNote = fmt.Sprintf("食分 %s。", magnitude)
	}

	if bestSunAlt > -6 {
		return EventVisibility{
			Status:          "limited",
			BestAt:          &bestAtText,
			AltitudeDegrees: altPtr,
			WindowStart:     &windowStartText,
			WindowEnd:       &windowEndText,
			Reason:          fmt.Sprintf("%s型月食：当地仍处于暮光阶段，观测条件有限。%s", eclipseType, magNote),
		}
	}
	if bestMoonAlt < 10 {
		return EventVisibility{
			Status:          "limited",
			BestAt:          &bestAtText,
			AltitudeDegrees: altPtr,
			WindowStart:     &windowStartText,
			WindowEnd:       &windowEndText,
			Reason:          fmt.Sprintf("%s型月食：月球高度较低，地平线遮挡和大气消光可能影响观测。%s", eclipseType, magNote),
		}
	}
	return EventVisibility{
		Status:          "observable",
		BestAt:          &bestAtText,
		AltitudeDegrees: altPtr,
		WindowStart:     &windowStartText,
		WindowEnd:       &windowEndText,
		Reason:          fmt.Sprintf("%s型月食：月球在当地地平线以上且为夜间，适合观测。%s", eclipseType, magNote),
	}
}

// solveMeteorShower 按当地夜间+辐射点高度+月光条件判定流星雨可见性。
// 流星雨的辐射点赤经/赤纬来自 IMO/IAU MDC 资料；这里用主要流星雨的固定辐射点坐标，
// 在事件时刻把辐射点赤道坐标转地平坐标，按当地辐射点高度判定可见性。
// 辐射点高度 < 0 → not_visible；0-15° → limited；≥15° 且夜间且月光弱 → observable。
func (s *VisibilitySolver) solveMeteorShower(event EventInput, latitude, longitude float64, loc *time.Location) EventVisibility {
	if s.moons == nil {
		return EventVisibility{Status: "not_calculated", Reason: "本地星历服务尚未配置。"}
	}
	if _, _, ok := meteorShowerRadiant(event); !ok {
		return EventVisibility{Status: "not_calculated", Reason: "流星雨资料缺少辐射点坐标，无法诚实地计算本地可见性。"}
	}
	at := event.StartsAt.UTC()

	// 扩展观测窗口（事件时刻 ±3 小时）内采样辐射点高度与太阳高度，
	// 找出"太阳在地平下、辐射点高度最高"的最佳时刻。
	start := at.Add(-3 * time.Hour)
	end := at.Add(3 * time.Hour)
	bestRadiantAlt := -math.MaxFloat64
	var bestAt time.Time
	bestSunAlt := math.MaxFloat64
	found := false
	var windowStart, windowEnd *time.Time
	for t := start; !t.After(end); t = t.Add(20 * time.Minute) {
		sunAlt := EclipticToHorizontalAltitude("sun", latitude, longitude, t)
		if sunAlt > -6 {
			continue
		}
		radiantAlt := meteorShowerRadiantAltitude(event, latitude, longitude, t)
		if radiantAlt <= 0 {
			continue
		}
		found = true
		if windowStart == nil {
			value := t
			windowStart = &value
		}
		value := t
		windowEnd = &value
		if radiantAlt > bestRadiantAlt {
			bestRadiantAlt = radiantAlt
			bestAt = t
			bestSunAlt = sunAlt
		}
	}

	if !found {
		return EventVisibility{
			Status: "not_visible",
			Reason: "流星雨极大时当地辐射点在地平线下或为白昼，目标不可见。",
		}
	}

	bestAtText := bestAt.In(loc).Format(time.RFC3339)
	windowStartText, windowEndText := windowStart.In(loc).Format(time.RFC3339), windowEnd.In(loc).Format(time.RFC3339)
	altPtr := &bestRadiantAlt

	// 月光条件：月球亮面占比高且月球在地平上 → limited
	phase := s.moons.Phase(bestAt)
	moonAlt := s.moons.Altitude(latitude, longitude, 0, bestAt)
	moonlightStrong := phase.Illumination > 0.5 && moonAlt > 0

	if bestRadiantAlt < 15 {
		return EventVisibility{
			Status:          "limited",
			BestAt:          &bestAtText,
			AltitudeDegrees: altPtr,
			WindowStart:     &windowStartText,
			WindowEnd:       &windowEndText,
			Reason:          "流星雨极大时当地辐射点高度较低，观测条件有限。",
		}
	}
	if bestSunAlt > -12 {
		return EventVisibility{
			Status:          "limited",
			BestAt:          &bestAtText,
			AltitudeDegrees: altPtr,
			WindowStart:     &windowStartText,
			WindowEnd:       &windowEndText,
			Reason:          "流星雨极大时当地仍处于暮光阶段，观测条件有限。",
		}
	}
	if moonlightStrong {
		return EventVisibility{
			Status:          "limited",
			BestAt:          &bestAtText,
			AltitudeDegrees: altPtr,
			WindowStart:     &windowStartText,
			WindowEnd:       &windowEndText,
			Reason:          "流星雨极大时月光较强，观测条件有限。",
		}
	}
	return EventVisibility{
		Status:          "observable",
		BestAt:          &bestAtText,
		AltitudeDegrees: altPtr,
		WindowStart:     &windowStartText,
		WindowEnd:       &windowEndText,
		Reason:          "流星雨极大时当地为夜间、辐射点高度足够且月光干扰较低，适合观测。",
	}
}

// meteorShowerRadiantAltitude 计算流星雨辐射点在给定地点、时刻的地平高度。
// 辐射点赤经/赤纬来自 IMO/IAU MDC 资料；这里内置主要流星雨的固定辐射点坐标。
// 辐射点坐标在 geometry 中（geometry.radiantRA/radiantDec）或按事件 ID 查表。
func meteorShowerRadiantAltitude(event EventInput, latitude, longitude float64, at time.Time) float64 {
	ra, dec, ok := meteorShowerRadiant(event)
	if !ok {
		return math.NaN()
	}
	return equatorialToHorizontalAltitude(ra, dec, latitude, longitude, at)
}

// meteorShowerRadiant 返回流星雨辐射点的赤经/赤纬（度）。
// 优先用 geometry.radiantRA/radiantDec；否则按事件 Kind 查内置表。
func meteorShowerRadiant(event EventInput) (ra, dec float64, ok bool) {
	if v, ok := event.Geometry["radiantRA"].(float64); ok {
		ra = v
		if v2, ok := event.Geometry["radiantDec"].(float64); ok {
			dec = v2
			return ra, dec, true
		}
	}
	// 内置主要流星雨辐射点（J2000 赤经/赤纬，度），按事件 Kind 索引
	radiants := map[string][2]float64{
		"perseids":                   {48, 58},   // 英仙座
		"kappa-cygnids":              {305, 59},  // 天鹅座 κ
		"aurigids":                   {90, 40},   // 御夫座
		"september-epsilon-perseids": {48, 40},   // 九月 ε 英仙座
		"orionids":                   {95, 16},   // 猎户座
		"leonids":                    {152, 22},  // 狮子座
		"geminids":                   {112, 33},  // 双子座
		"ursids":                     {217, 76},  // 小熊座
		"quadrantids":                {230, 49},  // 象限仪座
		"taurs":                      {54, 22},   // 金牛座
		"lyrids":                     {271, 34},  // 天琴座
		"eta-aquariids":              {338, -1},  // 宝瓶座 η
		"delta-aquariids":            {339, -16}, // 宝瓶座 δ
		"capricornids":               {307, -10}, // 摩羯座
	}
	key, _ := event.Geometry["slug"].(string)
	if key == "" {
		for slug := range radiants {
			if strings.Contains(event.ID, slug) {
				key = slug
				break
			}
		}
	}
	if radiant, found := radiants[key]; found {
		return radiant[0], radiant[1], true
	}
	return 0, 0, false
}

// equatorialToHorizontalAltitude 把赤道坐标（赤经/赤纬，度）转为本地点、时刻的地平高度。
func equatorialToHorizontalAltitude(ra, dec, latitude, longitude float64, at time.Time) float64 {
	T := julianCenturies(at)
	phi := deg2rad(latitude)
	raRad := deg2rad(ra)
	decRad := deg2rad(dec)
	hourAngle := deg2rad(greenwichSidereal(T, julianDay(at))+longitude) - raRad
	sinAlt := math.Sin(phi)*math.Sin(decRad) + math.Cos(phi)*math.Cos(decRad)*math.Cos(hourAngle)
	return rad2deg(math.Asin(clampUnit(sinAlt)))
}

// solveSolarEclipse 在没有已验证的 Besselian 本地接触时刻前保持诚实的未计算状态。
// 仅凭当地白昼不能推断路径覆盖，绝不能把全球发生的日食写成当地 limited/observable。
func (s *VisibilitySolver) solveSolarEclipse(event EventInput, latitude, longitude float64, loc *time.Location) EventVisibility {
	return EventVisibility{Status: "not_calculated", Reason: "尚未接入经验证的 NASA/GSFC Besselian 本地路径求解器；不会把全球日食误标为当地可见。"}
}
