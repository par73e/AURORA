package mars

import "math"

// 火星引力常数（km³/s²）
const muMars = 42828.37

// Elements 轨道根数（火心，黄道参考系）
type Elements struct {
	A               float64 // 半长轴 km
	E               float64 // 离心率
	InclinationDeg  float64 // 轨道倾角
	RaanDeg         float64 // 升交点黄经
	ArgPeriapsisDeg float64 // 近心点幅角
	MeanAnomalyDeg  float64 // 历元平近点角
	PeriodSeconds   float64 // 周期 s
}

func clamp(v float64) float64 {
	if v > 1 {
		return 1
	}
	if v < -1 {
		return -1
	}
	return v
}

// RVToElements 状态矢量 → 瞬时（osculating）轨道根数（与 moon.RVToElements 同一变换，μ 为火星）
func RVToElements(rx, ry, rz, vx, vy, vz float64) Elements {
	r := math.Sqrt(rx*rx + ry*ry + rz*rz)
	v2 := vx*vx + vy*vy + vz*vz

	hx := ry*vz - rz*vy
	hy := rz*vx - rx*vz
	hz := rx*vy - ry*vx
	h := math.Sqrt(hx*hx + hy*hy + hz*hz)

	nx := -hy
	ny := hx
	n := math.Sqrt(nx*nx + ny*ny)

	ex := (vy*hz-vz*hy)/muMars - rx/r
	ey := (vz*hx-vx*hz)/muMars - ry/r
	ez := (vx*hy-vy*hx)/muMars - rz/r
	e := math.Sqrt(ex*ex + ey*ey + ez*ez)

	a := 1.0 / (2.0/r - v2/muMars)

	inc := math.Acos(clamp(hz / h))

	raan := 0.0
	if n > 1e-12 {
		raan = math.Atan2(ny, nx)
		if raan < 0 {
			raan += 2 * math.Pi
		}
	}

	argp := 0.0
	if n > 1e-12 && e > 1e-12 {
		cosw := (nx*ex + ny*ey) / (n * e)
		argp = math.Acos(clamp(cosw))
		if ez < 0 {
			argp = 2*math.Pi - argp
		}
	}

	nu := 0.0
	if e > 1e-12 {
		cosnu := (ex*rx + ey*ry + ez*rz) / (e * r)
		nu = math.Acos(clamp(cosnu))
		if rx*vx+ry*vy+rz*vz < 0 {
			nu = 2*math.Pi - nu
		}
	}

	E := 2 * math.Atan2(math.Sqrt(1-e)*math.Sin(nu/2), math.Sqrt(1+e)*math.Cos(nu/2))
	M := E - e*math.Sin(E)
	if M < 0 {
		M += 2 * math.Pi
	}

	return Elements{
		A:               a,
		E:               e,
		InclinationDeg:  inc * 180 / math.Pi,
		RaanDeg:         raan * 180 / math.Pi,
		ArgPeriapsisDeg: argp * 180 / math.Pi,
		MeanAnomalyDeg:  M * 180 / math.Pi,
		PeriodSeconds:   2 * math.Pi * math.Sqrt(a*a*a/muMars),
	}
}
