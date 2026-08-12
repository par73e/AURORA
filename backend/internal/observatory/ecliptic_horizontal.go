package observatory

import (
	"math"
	"time"
)

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

// eclipticToHorizontal 是内部实现：取天体黄道坐标，转赤道，再转地平。
// 目前支持 sun 和 moon；其他 object 退化为 sun（行星事件另有 solvePlanetary 路径）。
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
		// 行星暂用太阳黄经占位；行星本地可见性由 solvePlanetary 单独处理。
		lon = sunLongitude(T)
		lat = 0
	}
	ra, dec := eclipticToEquatorial(lon, lat, obliquity(T))
	phi := deg2rad(latitude)
	hourAngle := deg2rad(greenwichSidereal(T, julianDay(at))+longitude) - ra
	sinAlt := math.Sin(phi)*math.Sin(dec) + math.Cos(phi)*math.Cos(dec)*math.Cos(hourAngle)
	altitude = rad2deg(math.Asin(clampUnit(sinAlt)))
	azimuth = norm360(rad2deg(math.Atan2(math.Sin(hourAngle), math.Cos(hourAngle)*math.Sin(phi)-math.Tan(dec)*math.Cos(phi))) + 180)
	return altitude, azimuth
}
