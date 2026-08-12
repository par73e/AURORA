package observatory

import (
	"fmt"
	"math"
	"time"
)

// EventInput 是给本地可见性求解器的最小事件输入。
// 它故意不依赖 astronomyevent 包，避免 observatory 反向依赖事件领域。
type EventInput struct {
	Kind     string
	StartsAt time.Time
	Geometry map[string]any // 行星事件的 object/objects、合的 separationDegrees 等
}

// EventVisibility 是天象事件在某地点的本地可见性结论。
// 对应设计文档中的 observable / limited / not_visible / non_visual 四态。
type EventVisibility struct {
	Status          string   // observable | limited | not_visible | non_visual | not_calculated
	BestAt          *string  // RFC3339 本地时刻；non_visual/not_calculated 可为空
	WindowStart     *string  // 最佳观测窗口起止（可观测/有限时填写）
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
		"march_equinox", "june_solstice", "september_equinox", "december_solstice":
		// 季节节点、月球轨道节点是非视觉几何节点。
		return EventVisibility{Status: "non_visual", Reason: "这是重要的天文几何节点，不对应一个只在该瞬间可见的独立目标。"}
	case "full_moon", "first_quarter", "last_quarter":
		return s.solveMoonPhase(event, latitude, longitude, loc)
	case "lunar_eclipse":
		return s.solveLunarEclipse(event, latitude, longitude, loc)
	case "meteor_shower":
		return s.solveMeteorShower(event, latitude, longitude, loc)
	case "planetary_conjunction", "planetary_opposition", "planetary_elongation":
		return s.solvePlanetary(event, latitude, longitude, loc)
	case "solar_eclipse":
		// 日食只在路径覆盖区可见；用 NASA/GSFC 食类型 + 当地太阳高度判定。
		// 路径数据（Besselian 元素）解析接入后可精确判定；当前用食类型 + 当地白天判定。
		return s.solveSolarEclipse(event, latitude, longitude, loc)
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
	status, reason := "observable", "月球在本地日期内有较好的中天高度。"
	if altitude < 10 {
		status, reason = "limited", "月球中天高度较低，地平线遮挡和大气消光可能影响观测。"
	}
	return EventVisibility{
		Status:          status,
		BestAt:          &bestAtText,
		AzimuthDegrees:  azimuth,
		AltitudeDegrees: &altitude,
		Reason:          reason,
	}
}

// solvePlanetary 把行星地心黄道坐标转为本地点的地平坐标，
// 在事件当日 ±12 小时窗口内采样目标高度与太阳高度，给出 observable/limited/not_visible。
// 合事件（planetary_conjunction）的 objects 数组可能含 moon；
// 月合用两个 object 中地平高度更高者做可见性判定，避免月球不可见误判行星合。
func (s *VisibilitySolver) solvePlanetary(event EventInput, latitude, longitude float64, loc *time.Location) EventVisibility {
	objects := collectConjunctionObjects(event.Geometry)
	if len(objects) == 0 {
		object, ok := event.Geometry["object"].(string)
		if !ok {
			return EventVisibility{Status: "not_calculated", Reason: "缺少行星对象信息，无法做本地可见性判断。"}
		}
		objects = []string{object}
	}

	start := event.StartsAt.UTC().Add(-12 * time.Hour)
	end := event.StartsAt.UTC().Add(12 * time.Hour)
	bestAlt := -math.MaxFloat64
	bestObject := ""
	var bestAt time.Time
	bestSunAlt := math.MaxFloat64
	found := false
	for at := start; !at.After(end); at = at.Add(30 * time.Minute) {
		sunAlt := EclipticToHorizontalAltitude("sun", latitude, longitude, at)
		// 要求太阳在暮光结束后（民用暮光 -6°）
		if sunAlt > -6 {
			continue
		}
		// 合事件：取两个 object 中地平高度更高者
		currentBestAlt := -math.MaxFloat64
		currentBestObject := objects[0]
		for _, object := range objects {
			alt := EclipticToHorizontalAltitude(object, latitude, longitude, at)
			if alt > currentBestAlt {
				currentBestAlt = alt
				currentBestObject = object
			}
		}
		found = true
		if currentBestAlt > bestAlt {
			bestAlt = currentBestAlt
			bestObject = currentBestObject
			bestAt = at
			bestSunAlt = sunAlt
		}
	}

	if !found {
		return EventVisibility{Status: "not_visible", Reason: "事件发生时本地为白昼或极昼，目标不可见。"}
	}

	azimuth := EclipticToHorizontalAzimuth(bestObject, latitude, longitude, bestAt)
	bestAtText := bestAt.In(loc).Format(time.RFC3339)
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
			Reason:          "目标高度较低或暮光较强，观测条件有限。",
		}
	}
	return EventVisibility{
		Status:          "observable",
		BestAt:          &bestAtText,
		AzimuthDegrees:  azPtr,
		AltitudeDegrees: altPtr,
		Reason:          "本地夜间目标高度足够，适合观测。",
	}
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
	for at := start; !at.After(end); at = at.Add(15 * time.Minute) {
		moonAlt := s.moons.Altitude(latitude, longitude, 0, at)
		sunAlt := EclipticToHorizontalAltitude("sun", latitude, longitude, at)
		if moonAlt <= 0 || sunAlt > -6 {
			continue
		}
		found = true
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
			Reason:          fmt.Sprintf("%s型月食：当地仍处于暮光阶段，观测条件有限。%s", eclipseType, magNote),
		}
	}
	if bestMoonAlt < 10 {
		return EventVisibility{
			Status:          "limited",
			BestAt:          &bestAtText,
			AltitudeDegrees: altPtr,
			Reason:          fmt.Sprintf("%s型月食：月球高度较低，地平线遮挡和大气消光可能影响观测。%s", eclipseType, magNote),
		}
	}
	return EventVisibility{
		Status:          "observable",
		BestAt:          &bestAtText,
		AltitudeDegrees: altPtr,
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
	at := event.StartsAt.UTC()

	// 扩展观测窗口（事件时刻 ±3 小时）内采样辐射点高度与太阳高度，
	// 找出"太阳在地平下、辐射点高度最高"的最佳时刻。
	start := at.Add(-3 * time.Hour)
	end := at.Add(3 * time.Hour)
	bestRadiantAlt := -math.MaxFloat64
	var bestAt time.Time
	bestSunAlt := math.MaxFloat64
	found := false
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
			Reason:          "流星雨极大时当地辐射点高度较低，观测条件有限。",
		}
	}
	if bestSunAlt > -12 {
		return EventVisibility{
			Status:          "limited",
			BestAt:          &bestAtText,
			AltitudeDegrees: altPtr,
			Reason:          "流星雨极大时当地仍处于暮光阶段，观测条件有限。",
		}
	}
	if moonlightStrong {
		return EventVisibility{
			Status:          "limited",
			BestAt:          &bestAtText,
			AltitudeDegrees: altPtr,
			Reason:          "流星雨极大时月光较强，观测条件有限。",
		}
	}
	return EventVisibility{
		Status:          "observable",
		BestAt:          &bestAtText,
		AltitudeDegrees: altPtr,
		Reason:          "流星雨极大时当地为夜间、辐射点高度足够且月光干扰较低，适合观测。",
	}
}

// meteorShowerRadiantAltitude 计算流星雨辐射点在给定地点、时刻的地平高度。
// 辐射点赤经/赤纬来自 IMO/IAU MDC 资料；这里内置主要流星雨的固定辐射点坐标。
// 辐射点坐标在 geometry 中（geometry.radiantRA/radiantDec）或按事件 ID 查表。
func meteorShowerRadiantAltitude(event EventInput, latitude, longitude float64, at time.Time) float64 {
	ra, dec := meteorShowerRadiant(event)
	return equatorialToHorizontalAltitude(ra, dec, latitude, longitude, at)
}

// meteorShowerRadiant 返回流星雨辐射点的赤经/赤纬（度）。
// 优先用 geometry.radiantRA/radiantDec；否则按事件 Kind 查内置表。
func meteorShowerRadiant(event EventInput) (ra, dec float64) {
	if v, ok := event.Geometry["radiantRA"].(float64); ok {
		ra = v
		if v2, ok := event.Geometry["radiantDec"].(float64); ok {
			dec = v2
			return
		}
	}
	// 内置主要流星雨辐射点（J2000 赤经/赤纬，度），按事件 Kind 索引
	radiants := map[string][2]float64{
		"perseids":                         {48, 58},   // 英仙座
		"kappa-cygnids":                    {305, 59},  // 天鹅座 κ
		"aurigids":                         {90, 40},   // 御夫座
		"september-epsilon-perseids":       {48, 40},   // 九月 ε 英仙座
		"orionids":                         {95, 16},   // 猎户座
		"leonids":                          {152, 22},  // 狮子座
		"geminids":                         {112, 33},  // 双子座
		"ursids":                           {217, 76},  // 小熊座
		"quadrantids":                      {230, 49},  // 象限仪座
		"taurs":                            {54, 22},   // 金牛座
		"lyrids":                           {271, 34},  // 天琴座
		"eta-aquariids":                    {338, -1},  // 宝瓶座 η
		"delta-aquariids":                  {339, -16}, // 宝瓶座 δ
		"capricornids":                     {307, -10}, // 摩羯座
	}
	if r, ok := radiants[event.Kind]; ok {
		return r[0], r[1]
	}
	// 默认：英仙座辐射点
	return 48, 58
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

// solveSolarEclipse 用 NASA/GSFC 食类型 + 当地太阳高度判定日食可见性。
// 日食只在路径覆盖区可见；当前用食类型 + 当地白天判定，
// 路径数据（Besselian 元素）解析接入后可精确判定路径覆盖。
func (s *VisibilitySolver) solveSolarEclipse(event EventInput, latitude, longitude float64, loc *time.Location) EventVisibility {
	at := event.StartsAt.UTC()
	sunAlt := EclipticToHorizontalAltitude("sun", latitude, longitude, at)
	bestAtText := at.In(loc).Format(time.RFC3339)

	// 食类型来自 NASA/GSFC 解析（geometry.eclipseType）
	eclipseType, _ := event.Geometry["eclipseType"].(string)
	if eclipseType == "" {
		eclipseType = "Partial"
	}
	pathURL, _ := event.Geometry["pathUrl"].(string)

	// 日食发生时太阳必须在地平线上（白天）
	if sunAlt <= 0 {
		return EventVisibility{
			Status: "not_visible",
			Reason: "日食发生时当地太阳在地平线下，无可见画面。",
		}
	}

	// 路径覆盖区判定：当前无 Besselian 元素，保守标记为 limited（需查路径图）
	// 食类型为 Total/Annular/Hybrid 时路径覆盖区可见；Partial 时部分地区可见
	if eclipseType == "Total" || eclipseType == "Annular" || eclipseType == "Hybrid" {
		return EventVisibility{
			Status:          "limited",
			BestAt:          &bestAtText,
			AltitudeDegrees: &sunAlt,
			Reason:          fmt.Sprintf("%s型日食：当地为白天，但只在路径覆盖区才可见；请查 NASA/GSFC 路径图确认。", eclipseType),
		}
	}
	// Partial 日食：部分地区可见，仍需查路径图
	if pathURL != "" {
		return EventVisibility{
			Status:          "limited",
			BestAt:          &bestAtText,
			AltitudeDegrees: &sunAlt,
			Reason:          "偏食：当地为白天，部分地区可见；请查 NASA/GSFC 路径图确认。",
		}
	}
	return EventVisibility{
		Status:          "limited",
		BestAt:          &bestAtText,
		AltitudeDegrees: &sunAlt,
		Reason:          "日食发生时当地为白天，需查 NASA/GSFC 路径图确认当地可见性。",
	}
}
