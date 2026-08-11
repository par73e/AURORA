// 月相板块后端实现：自研（Meeus 级）太阳/月球位置，计算相位、亮面占比、月龄
// 与月出/月落/中天。
//
// 设计决策：
//   - 无外网、不引入第三方天文库，算法在仓库内自包含；
//   - 结果按 (经纬度, 时区, 本地日期) 每日缓存 —— 月相每天只需更新一次；
//   - 精度通过 frontend/tests/moon_reference.test.mjs 与前端 astronomy-engine
//     交叉校验（见 moon_test.go 生成的 testdata/moon_reference.json）。
package observatory

import (
	"errors"
	"math"
	"strconv"
	"sync"
	"time"
)

// ErrInvalidTimezone 表示 timezone 参数不是合法 IANA 时区名。
var ErrInvalidTimezone = errors.New("invalid IANA timezone")

const (
	meanSynodicMonthDays = 29.530588853
	moonRiseSetAltitude  = -0.5667 // 地平标准高度：34′ 平均大气折射（度）
	earthMeanRadiusKm    = 6378.137
	obliquityJ2000       = 23.4392911
)

// MoonPhaseResult 是月相在某一时刻的快照，供月相板块与观测评分共用。
type MoonPhaseResult struct {
	Phase        float64 `json:"phase"`        // 相位角 0–360（0 朔、180 望）
	Illumination float64 `json:"illumination"` // 亮面占比 0–1
	Age          float64 `json:"age"`          // 月龄（天）
	Label        string  `json:"label"`        // 中文月相名
}

// MoonDay 是月相板块每日数据：本地日期、相位快照与当天的月出/月落/中天。
// 时间字段为 Unix 秒（UTC 绝对时刻），由客户端在本地时区展示。
type MoonDay struct {
	Date     string `json:"date"`     // 数据所属的本地日期 YYYY-MM-DD（按 timezone 计算）
	Timezone string `json:"timezone"` // IANA 时区名
	MoonPhaseResult
	Moonrise   *int64 `json:"moonrise"` // 本地日内月出（Unix 秒）；极区当日可能无
	Moonset    *int64 `json:"moonset"`
	Transit    *int64 `json:"transit"` // 本地日内中天（最高点）
	ComputedAt int64  `json:"computedAt"`
}

// MoonProvider 是 httpapi 依赖的最小月相接口。
type MoonProvider interface {
	Day(latitude, longitude, elevation float64, at time.Time, timezone string) (MoonDay, error)
	Phase(at time.Time) MoonPhaseResult
	Altitude(latitude, longitude, elevation float64, at time.Time) float64
}

// MoonService 提供每日缓存的月相数据与任意时刻的相位快照。
type MoonService struct {
	now   func() time.Time
	mu    sync.Mutex
	cache map[string]moonCacheEntry
}

type moonCacheEntry struct {
	until time.Time
	value MoonDay
}

// NewMoonService 构造月相服务。now 可注入用于测试缓存过期。
func NewMoonService() *MoonService {
	return &MoonService{now: time.Now, cache: make(map[string]moonCacheEntry)}
}

// Day 返回 at 所在本地日期（按 timezone）的月相数据，按 (经纬度, 海拔, 时区, 日期) 每日缓存。
// elevation 为观测点海拔（米），参与 topocentric 视差修正；timezone 为空时按 UTC 解释。
func (s *MoonService) Day(latitude, longitude, elevation float64, at time.Time, timezone string) (MoonDay, error) {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 || math.IsNaN(latitude) || math.IsNaN(longitude) {
		return MoonDay{}, errors.New("observer coordinates are invalid")
	}
	if elevation < -500 || elevation > 9000 || math.IsNaN(elevation) {
		return MoonDay{}, errors.New("observer elevation is invalid")
	}
	if timezone == "" {
		timezone = "UTC"
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return MoonDay{}, ErrInvalidTimezone
	}
	local := at.In(location)
	date := local.Format("2006-01-02")
	key := moonCacheKey(latitude, longitude) + "|" + elevationCacheKey(elevation) + "|" + timezone + "|" + date

	s.mu.Lock()
	if cached, ok := s.cache[key]; ok && s.now().Before(cached.until) {
		s.mu.Unlock()
		return cached.value, nil
	}
	s.mu.Unlock()

	day := computeMoonDay(latitude, longitude, elevation, at, location)
	s.mu.Lock()
	s.cache[key] = moonCacheEntry{until: nextLocalMidnight(local, location), value: day}
	s.mu.Unlock()
	return day, nil
}

// Phase 返回任意时刻（UTC）的月相快照，不缓存，供观测评分使用。
func (s *MoonService) Phase(at time.Time) MoonPhaseResult {
	return moonPhaseAt(at)
}

// Altitude 返回指定地点与时刻的月球地平高度角（度），包含地心视差修正。
// 评分使用与月出月落计算相同的 moonRiseSetAltitude 阈值，避免月亮落下后仍扣月光分。
func (s *MoonService) Altitude(latitude, longitude, elevation float64, at time.Time) float64 {
	return moonAltitude(latitude, longitude, elevation, at)
}

// computeMoonDay 计算 at 所在本地日期（location）内的月相与升落中天。
func computeMoonDay(latitude, longitude, elevation float64, at time.Time, location *time.Location) MoonDay {
	local := at.In(location)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	end := start.Add(24 * time.Hour)
	phase := moonPhaseAt(at)
	moonrise, moonset := moonRiseSet(latitude, longitude, elevation, start, end)
	transit := moonTransit(latitude, longitude, elevation, start, end)
	return MoonDay{
		Date:            local.Format("2006-01-02"),
		Timezone:        location.String(),
		MoonPhaseResult: phase,
		Moonrise:        moonrise,
		Moonset:         moonset,
		Transit:         transit,
		ComputedAt:      at.Unix(),
	}
}

func moonPhaseAt(at time.Time) MoonPhaseResult {
	T := julianCenturies(at)
	moonLongitude, _, _ := moonPosition(T)
	phase := norm360(moonLongitude - sunLongitude(T))
	illumination := (1 - math.Cos(deg2rad(phase))) / 2
	return MoonPhaseResult{
		Phase:        phase,
		Illumination: illumination,
		Age:          phase / 360 * meanSynodicMonthDays,
		Label:        moonLabel(phase),
	}
}

func moonLabel(phase float64) string {
	switch {
	case phase < 18 || phase >= 342:
		return "朔月"
	case phase < 72:
		return "娥眉月"
	case phase < 108:
		return "上弦月"
	case phase < 162:
		return "盈凸月"
	case phase < 198:
		return "满月"
	case phase < 252:
		return "亏凸月"
	case phase < 288:
		return "下弦月"
	default:
		return "残月"
	}
}

// moonRiseSet 在 [start, end) 内搜索当日第一个月出与第一个月落。
func moonRiseSet(latitude, longitude, elevation float64, start, end time.Time) (*int64, *int64) {
	const step = 5 * time.Minute
	var rise, set *int64
	previousTime := start
	previous := moonAltitude(latitude, longitude, elevation, start) - moonRiseSetAltitude
	for t := start.Add(step); !t.After(end); t = t.Add(step) {
		current := moonAltitude(latitude, longitude, elevation, t) - moonRiseSetAltitude
		if previous < 0 && current >= 0 && rise == nil {
			cross := refineCrossing(latitude, longitude, elevation, previousTime, t)
			rise = &cross
		}
		if previous > 0 && current <= 0 && set == nil {
			cross := refineCrossing(latitude, longitude, elevation, previousTime, t)
			set = &cross
		}
		previousTime = t
		previous = current
	}
	return rise, set
}

// refineCrossing 用二分法精化高度角穿越地平标准高度的时刻。
func refineCrossing(latitude, longitude, elevation float64, a, b time.Time) int64 {
	for i := 0; i < 10; i++ {
		mid := a.Add(b.Sub(a) / 2)
		aboveMid := moonAltitude(latitude, longitude, elevation, mid) >= moonRiseSetAltitude
		aboveStart := moonAltitude(latitude, longitude, elevation, a) >= moonRiseSetAltitude
		if aboveMid == aboveStart {
			a = mid
		} else {
			b = mid
		}
	}
	return a.Add(b.Sub(a) / 2).Unix()
}

// moonTransit 返回 [start, end) 内月球最高点时刻（5 分钟采样，与前端 transitFor 一致）。
func moonTransit(latitude, longitude, elevation float64, start, end time.Time) *int64 {
	best := start
	bestAltitude := math.Inf(-1)
	for t := start; t.Before(end); t = t.Add(5 * time.Minute) {
		altitude := moonAltitude(latitude, longitude, elevation, t)
		if altitude > bestAltitude {
			bestAltitude = altitude
			best = t
		}
	}
	if math.IsInf(bestAltitude, 1) {
		return nil
	}
	epoch := best.Unix()
	return &epoch
}

// moonAltitude 返回月球在 (latitude, longitude, elevation) 处 at 时刻的地平高度（度）。
// 使用地心视位置 + 周日视差修正（Meeus 第 40 章 topocentric 公式，含海拔 ρ 因子），不含大气折射。
func moonAltitude(latitude, longitude, elevation float64, at time.Time) float64 {
	T := julianCenturies(at)
	moonLongitude, moonLatitude, distanceKm := moonPosition(T)
	ra, dec := eclipticToEquatorial(moonLongitude, moonLatitude, obliquity(T))
	parallax := math.Asin(earthMeanRadiusKm / distanceKm) // 周日视差（弧度）

	phi := deg2rad(latitude)
	// 观察者地心纬度与地心距离因子（Meeus 40.10-40.12）：海拔 h 使观察者远离地心。
	u := math.Atan(0.99664719 * math.Tan(phi))
	hKm := elevation / 1000
	rhoSinPhiPrime := 0.99664719*math.Sin(u) + hKm/6378.137*math.Sin(phi)
	rhoCosPhiPrime := math.Cos(u) + hKm/6378.137*math.Cos(phi)
	hourAngle := deg2rad(greenwichSidereal(T, julianDay(at))+longitude) - ra

	sinPi := math.Sin(parallax)
	deltaRA := math.Atan2(-rhoCosPhiPrime*sinPi*math.Sin(hourAngle),
		math.Cos(dec)-rhoSinPhiPrime*sinPi*math.Cos(hourAngle))
	decTopo := math.Atan2((math.Sin(dec)-rhoSinPhiPrime*sinPi)*math.Cos(deltaRA),
		math.Cos(dec)-rhoSinPhiPrime*sinPi*math.Cos(hourAngle))

	sinAltitude := math.Sin(phi)*math.Sin(decTopo) + math.Cos(phi)*math.Cos(decTopo)*math.Cos(hourAngle-deltaRA)
	return rad2deg(math.Asin(clampUnit(sinAltitude)))
}

// ---------- 基础天算 ----------

func deg2rad(degrees float64) float64 { return degrees * math.Pi / 180 }
func rad2deg(radians float64) float64 { return radians * 180 / math.Pi }

func norm360(degrees float64) float64 {
	degrees = math.Mod(degrees, 360)
	if degrees < 0 {
		degrees += 360
	}
	return degrees
}

func clampUnit(value float64) float64 {
	if value > 1 {
		return 1
	}
	if value < -1 {
		return -1
	}
	return value
}

// julianDay 返回 UT 儒略日。用 Unix 秒计算，避免 time.Location 非 UTC 时
// 本地访问器（Date/Hour）把本地时刻误当 UTC 造成约时区小时的恒星时偏移。
func julianDay(t time.Time) float64 {
	return 2440587.5 + float64(t.Unix())/86400.0
}

// julianCenturies 返回自 J2000.0 起的儒略世纪数（T）。
func julianCenturies(t time.Time) float64 {
	return (julianDay(t) - 2451545.0) / 36525.0
}

// sunLongitude 返回太阳视黄经（度）。Meeus 第 25 章低精度公式（约 0.01°）。
func sunLongitude(T float64) float64 {
	l0 := norm360(280.46646 + 36000.76983*T + 0.0003032*T*T)
	m := norm360(357.52911 + 35999.05029*T - 0.0001537*T*T)
	equation := (1.914602-0.004817*T-0.000014*T*T)*math.Sin(deg2rad(m)) +
		(0.019993-0.000101*T)*math.Sin(deg2rad(2*m)) +
		0.000289*math.Sin(deg2rad(3*m))
	omega := norm360(125.04 - 1934.136*T)
	return norm360(l0 + equation - 0.00569 - 0.00478*math.Sin(deg2rad(omega)))
}

func obliquity(T float64) float64 {
	return obliquityJ2000 - 0.0130042*T
}

// eclipticToEquatorial 黄道→赤道坐标，返回 RA/Dec（弧度）。
func eclipticToEquatorial(longitude, latitude, epsilon float64) (ra, dec float64) {
	lon, lat, eps := deg2rad(longitude), deg2rad(latitude), deg2rad(epsilon)
	ra = math.Atan2(math.Sin(lon)*math.Cos(eps)-math.Tan(lat)*math.Sin(eps), math.Cos(lon))
	dec = math.Asin(math.Sin(lat)*math.Cos(eps) + math.Cos(lat)*math.Sin(eps)*math.Sin(lon))
	return ra, dec
}

// greenwichSidereal 返回格林尼治平恒星时（度）。
func greenwichSidereal(T, jd float64) float64 {
	return norm360(280.46061837 + 360.98564736629*(jd-2451545.0) + 0.000387933*T*T - T*T*T/38710000)
}

// ---------- 月球位置（Meeus 第 47 章截断级数） ----------

// periodicTerm 表示一个周期项：系数 × sin(D·d + M·m + M'·mp + F·f)，E 为章动因子幂次。
type periodicTerm struct {
	d, m, mp, f int
	E           int
	coeff       float64
}

// 月球经度周期项（10⁻⁶ 度），取主导 25 项，残余 < 0.03°。
var moonLongitudeTerms = []periodicTerm{
	{0, 0, 1, 0, 0, 6288774}, {2, 0, -1, 0, 0, 1274027}, {2, 0, 0, 0, 0, 658314}, {0, 0, 2, 0, 0, 213618},
	{0, 1, 0, 0, 1, -185116}, {0, 0, 0, 2, 0, -114332}, {2, 0, -2, 0, 0, 58793}, {2, -1, -1, 0, 1, 57066},
	{2, 0, 1, 0, 0, 53322}, {2, -1, 0, 0, 1, 45758}, {0, 1, -1, 0, 1, -40923}, {1, 0, 0, 0, 0, -34720},
	{0, 1, 1, 0, 1, -30383}, {2, 0, 0, -2, 0, 15327}, {0, 0, 1, 2, 0, -12528}, {0, 0, 1, -2, 0, 10980},
	{4, 0, -1, 0, 0, 10675}, {0, 0, 3, 0, 0, 10034}, {4, 0, -2, 0, 0, 8548}, {2, 1, -1, 0, 1, -7888},
	{2, -1, 1, 0, 1, -6766}, {1, 0, -2, 0, 0, -5163}, {1, -1, 0, 0, 1, 4987}, {1, 1, 0, 0, 1, 4036},
	{2, 0, -1, 2, 0, 3994},
}

// 月球纬度周期项（10⁻⁶ 度），取主导 15 项，残余 < 0.02°。
var moonLatitudeTerms = []periodicTerm{
	{0, 0, 0, 1, 0, 5128122}, {0, 0, 1, 1, 0, 280602}, {0, 0, 1, -1, 0, 277693}, {2, 0, 0, -1, 0, 173237},
	{2, 0, -1, 1, 0, 55413}, {2, 0, -1, -1, 0, 46271}, {2, 0, 0, 1, 0, 32573}, {0, 0, 2, 1, 0, 17198},
	{2, 0, 1, -1, 0, 9266}, {0, 0, 2, -1, 0, 8822}, {2, -1, 0, -1, 1, 8216}, {2, 0, -2, -1, 0, 4324},
	{2, 0, 1, 1, 0, 4200}, {2, 1, 0, -1, 1, -3359}, {2, -1, -1, 1, 1, 2463},
}

// 月球距离周期项（米），取主导 18 项（用于视差，精度需求低于经度）。
var moonDistanceTerms = []periodicTerm{
	{0, 0, 1, 0, 0, -20905355}, {2, 0, -1, 0, 0, -3699111}, {2, 0, 0, 0, 0, -2955968}, {0, 0, 2, 0, 0, -569925},
	{0, 1, 0, 0, 1, 48888}, {0, 0, 0, 2, 0, -3149}, {2, 0, -2, 0, 0, 246158}, {2, -1, -1, 0, 1, -152138},
	{2, 0, 1, 0, 0, -170733}, {2, -1, 0, 0, 1, -204586}, {0, 1, -1, 0, 1, -129620}, {1, 0, 0, 0, 0, 108743},
	{0, 1, 1, 0, 1, 104755}, {2, 0, 0, -2, 0, 10321}, {0, 0, 1, -2, 0, 79661}, {4, 0, -1, 0, 0, -34782},
	{0, 0, 3, 0, 0, -23210}, {4, 0, -2, 0, 0, -21636},
}

// moonPosition 返回月球地心视黄经、黄纬（度）与地心距离（km）。
func moonPosition(T float64) (longitude, latitude, distanceKm float64) {
	lprime := norm360(218.3164477 + 481267.88123421*T - 0.0015786*T*T + T*T*T/538841 - T*T*T*T/65194000)
	d := norm360(297.8501921 + 445267.1114034*T - 0.0018819*T*T + T*T*T/545868 - T*T*T*T/113065000)
	m := norm360(357.5291092 + 35999.0502909*T - 0.0001536*T*T + T*T*T/24490000)
	mp := norm360(134.9633964 + 477198.8675055*T + 0.0087414*T*T + T*T*T/69699 - T*T*T*T/14712000)
	f := norm360(93.2720950 + 483202.0175233*T - 0.0036539*T*T - T*T*T/3526000 + T*T*T*T/863310000)
	e := 1 - 0.002516*T - 0.0000074*T*T

	sum := func(terms []periodicTerm, unit float64, cosine bool) float64 {
		total := 0.0
		for _, term := range terms {
			argument := float64(term.d)*d + float64(term.m)*m + float64(term.mp)*mp + float64(term.f)*f
			amplitude := term.coeff
			if term.E > 0 {
				amplitude *= e
			}
			value := math.Sin(deg2rad(argument))
			if cosine {
				value = math.Cos(deg2rad(argument))
			}
			total += amplitude * value
		}
		return total * unit
	}
	sumL := sum(moonLongitudeTerms, 1e-6, false)
	sumB := sum(moonLatitudeTerms, 1e-6, false)
	sumR := sum(moonDistanceTerms, 1.0/1000, true) // 距离系数单位为米，且为余弦级数

	// 黄经章动主导项（角秒 → 度）
	omega := norm360(125.04452 - 1934.136261*T)
	sunMeanLongitude := norm360(280.46646 + 36000.76983*T)
	nutation := (-17.20*math.Sin(deg2rad(omega)) - 1.32*math.Sin(deg2rad(2*sunMeanLongitude)) -
		0.23*math.Sin(deg2rad(2*lprime)) + 0.21*math.Sin(deg2rad(2*omega))) / 3600

	return norm360(lprime + sumL + nutation), sumB, 385000.56 + sumR
}

func moonCacheKey(latitude, longitude float64) string {
	return strconv.FormatFloat(math.Round(latitude*100)/100, 'f', 2, 64) + "," +
		strconv.FormatFloat(math.Round(longitude*100)/100, 'f', 2, 64)
}

// elevationCacheKey 按 50m 归并海拔，避免缓存因微小海拔抖动而失效。
func elevationCacheKey(elevation float64) string {
	return strconv.FormatInt(int64(math.Round(elevation/50)*50), 10)
}

// nextLocalMidnight 返回 timezone 中 local 所在日期的下一个本地零点（缓存过期时刻）。
func nextLocalMidnight(local time.Time, location *time.Location) time.Time {
	return time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, location)
}
