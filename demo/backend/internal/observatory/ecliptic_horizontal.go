package observatory

import (
	"math"
	"time"
)

// EclipticCoordinates 是地心黄道坐标（度）。行星坐标来自后端缓存的 JPL Horizons 星历，
// 月球坐标则由本地月球模型提供；两者均可复用同一地平坐标转换。
type EclipticCoordinates struct {
	LongitudeDegrees float64
	LatitudeDegrees  float64
}

// EquatorialCoordinates 是赤道坐标（J2000 赤经/赤纬，度），用于亮星和星团等静态目录资料。
type EquatorialCoordinates struct {
	RightAscensionDegrees float64
	DeclinationDegrees    float64
}

// EclipticToHorizontal 把天体的地心黄道坐标转换为给定地点、时刻的地平坐标。
// 返回 (altitude, azimuth)，方位角以北为零、向东为正。
// 该函数仅做坐标变换，不做可见性判定。
// object 参数目前仅用于扩展接口；实际坐标由 sunLongitude/moonPosition 给出。
func EclipticToHorizontal(object string, latitude, longitude float64, at time.Time) (altitude, azimuth float64) {
	return eclipticToHorizontal(object, latitude, longitude, at)
}

// EclipticToHorizontalAltitude 只返回地平高度，供可见性求解器快速采样。
func EclipticToHorizontalAltitude(object string, latitude, longitude float64, at time.Time) float64 {
	alt, _ := eclipticToHorizontal(object, latitude, longitude, at)
	return alt
}

// EclipticToHorizontalAzimuth 只返回方位角。
func EclipticToHorizontalAzimuth(object string, latitude, longitude float64, at time.Time) float64 {
	_, az := eclipticToHorizontal(object, latitude, longitude, at)
	return az
}

// MoonEclipticCoordinates 返回月球在给定时刻的地心黄道坐标，供月合事件与本地可见性使用。
func MoonEclipticCoordinates(at time.Time) EclipticCoordinates {
	lon, lat, _ := moonPosition(julianCenturies(at))
	return EclipticCoordinates{LongitudeDegrees: lon, LatitudeDegrees: lat}
}

// EclipticCoordinatesToHorizontal 把任意已知地心黄道坐标转换为本地点地平坐标。
func EclipticCoordinatesToHorizontal(coordinates EclipticCoordinates, latitude, longitude float64, at time.Time) (altitude, azimuth float64) {
	T := julianCenturies(at)
	ra, dec := eclipticToEquatorial(coordinates.LongitudeDegrees, coordinates.LatitudeDegrees, obliquity(T))
	phi := deg2rad(latitude)
	hourAngle := deg2rad(greenwichSidereal(T, julianDay(at))+longitude) - ra
	sinAlt := math.Sin(phi)*math.Sin(dec) + math.Cos(phi)*math.Cos(dec)*math.Cos(hourAngle)
	altitude = rad2deg(math.Asin(clampUnit(sinAlt)))
	azimuth = norm360(rad2deg(math.Atan2(math.Sin(hourAngle), math.Cos(hourAngle)*math.Sin(phi)-math.Tan(dec)*math.Cos(phi))) + 180)
	return altitude, azimuth
}

// EquatorialCoordinatesToHorizontal 把赤道坐标转换为本地点地平坐标。
func EquatorialCoordinatesToHorizontal(coordinates EquatorialCoordinates, latitude, longitude float64, at time.Time) (altitude, azimuth float64) {
	T := julianCenturies(at)
	phi := deg2rad(latitude)
	ra := deg2rad(coordinates.RightAscensionDegrees)
	dec := deg2rad(coordinates.DeclinationDegrees)
	hourAngle := deg2rad(greenwichSidereal(T, julianDay(at))+longitude) - ra
	sinAlt := math.Sin(phi)*math.Sin(dec) + math.Cos(phi)*math.Cos(dec)*math.Cos(hourAngle)
	altitude = rad2deg(math.Asin(clampUnit(sinAlt)))
	azimuth = norm360(rad2deg(math.Atan2(math.Sin(hourAngle), math.Cos(hourAngle)*math.Sin(phi)-math.Tan(dec)*math.Cos(phi))) + 180)
	return altitude, azimuth
}

// EquatorialToEclipticCoordinates 将静态亮星/星团赤道坐标换到给定日期的黄道坐标，
// 仅用于寻找月球的接近候选；最终本地高度仍按赤道坐标计算。
func EquatorialToEclipticCoordinates(coordinates EquatorialCoordinates, at time.Time) EclipticCoordinates {
	ra, dec := deg2rad(coordinates.RightAscensionDegrees), deg2rad(coordinates.DeclinationDegrees)
	epsilon := deg2rad(obliquity(julianCenturies(at)))
	lon := math.Atan2(math.Sin(ra)*math.Cos(epsilon)+math.Tan(dec)*math.Sin(epsilon), math.Cos(ra))
	lat := math.Asin(math.Sin(dec)*math.Cos(epsilon) - math.Cos(dec)*math.Sin(epsilon)*math.Sin(ra))
	return EclipticCoordinates{LongitudeDegrees: norm360(rad2deg(lon)), LatitudeDegrees: rad2deg(lat)}
}

// eclipticToHorizontal 是内部实现：取已支持天体的黄道坐标，转赤道，再转地平。
// 仅太阳和月球有本地模型；行星必须通过 EclipticCoordinatesToHorizontal 传入 JPL 坐标，
// 绝不能回退到太阳坐标。
func eclipticToHorizontal(object string, latitude, longitude float64, at time.Time) (altitude, azimuth float64) {
	T := julianCenturies(at)
	var lon, lat float64
	switch object {
	case "sun":
		lon = sunLongitude(T)
		lat = 0
	case "moon":
		lon, lat, _ = moonPosition(T)
	default:
		return math.NaN(), math.NaN()
	}
	return EclipticCoordinatesToHorizontal(EclipticCoordinates{LongitudeDegrees: lon, LatitudeDegrees: lat}, latitude, longitude, at)
}
