package observatory

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// moonFixtureCase 与 frontend/tests/moon_reference.test.mjs 共享的基准用例。
// Go 端计算后写入 testdata/moon_reference.json，node 端用 astronomy-engine 复核。
type moonFixtureCase struct {
	Name         string  `json:"name"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	At           int64   `json:"at"`
	Timezone     string  `json:"timezone"`
	DayStart     int64   `json:"dayStart"` // at 所在本地日期零点（Unix 秒），供 node 端构造搜索起点
	Phase        float64 `json:"phase"`
	Illumination float64 `json:"illumination"`
	Age          float64 `json:"age"`
	Label        string  `json:"label"`
	Moonrise     *int64  `json:"moonrise"`
	Moonset      *int64  `json:"moonset"`
	Transit      *int64  `json:"transit"`
}

func TestMoonServiceInvariantsAndFixture(t *testing.T) {
	cases := []moonFixtureCase{
		{Name: "shanghai-20260809", Latitude: 31.23, Longitude: 121.47, At: unix("2026-08-09T12:00:00Z"), Timezone: "Asia/Shanghai"},
		{Name: "beijing-fullmoon-20260303", Latitude: 39.9042, Longitude: 116.4074, At: unix("2026-03-03T12:00:00Z"), Timezone: "Asia/Shanghai"},
		{Name: "newyork-20260118", Latitude: 40.7128, Longitude: -74.006, At: unix("2026-01-18T06:00:00Z"), Timezone: "America/New_York"},
		{Name: "sydney-20260621", Latitude: -33.8688, Longitude: 151.2093, At: unix("2026-06-21T04:00:00Z"), Timezone: "Australia/Sydney"},
		{Name: "tokyo-20261102", Latitude: 35.6762, Longitude: 139.6503, At: unix("2026-11-02T15:00:00Z"), Timezone: "Asia/Tokyo"},
	}

	service := NewMoonService()
	for index := range cases {
		c := &cases[index]
		at := time.Unix(c.At, 0)
		// 缓存 TTL 是"下一个本地零点"；注入以 at 为基准的时钟，避免真实时钟使过去日期的缓存立即过期。
		service.now = func() time.Time { return at }
		location, err := time.LoadLocation(c.Timezone)
		if err != nil {
			t.Fatalf("load timezone %q: %v", c.Timezone, err)
		}
		local := at.In(location)
		dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
		c.DayStart = dayStart.Unix()

		day, err := service.Day(c.Latitude, c.Longitude, 0, at, c.Timezone)
		if err != nil {
			t.Fatalf("%s: Day() error: %v", c.Name, err)
		}
		c.Phase, c.Illumination, c.Age, c.Label = day.Phase, day.Illumination, day.Age, day.Label
		c.Moonrise, c.Moonset, c.Transit = day.Moonrise, day.Moonset, day.Transit

		if day.Phase < 0 || day.Phase >= 360 {
			t.Errorf("%s: phase 越界 %v", c.Name, day.Phase)
		}
		if day.Illumination < 0 || day.Illumination > 1 {
			t.Errorf("%s: illumination 越界 %v", c.Name, day.Illumination)
		}
		if day.Age < 0 || day.Age >= meanSynodicMonthDays {
			t.Errorf("%s: age 越界 %v", c.Name, day.Age)
		}
		if day.Label == "" {
			t.Errorf("%s: label 为空", c.Name)
		}
		if day.Date != local.Format("2006-01-02") {
			t.Errorf("%s: 日期 %q != %q", c.Name, day.Date, local.Format("2006-01-02"))
		}
		if day.ComputedAt != c.At {
			t.Errorf("%s: ComputedAt %d != at %d", c.Name, day.ComputedAt, c.At)
		}
		for name, value := range map[string]*int64{"moonrise": day.Moonrise, "moonset": day.Moonset, "transit": day.Transit} {
			if value != nil && (*value < dayStart.Unix() || *value > dayStart.Unix()+86400) {
				t.Errorf("%s: %s %d 超出本地日窗口 [%d, %d]", c.Name, name, *value, dayStart.Unix(), dayStart.Unix()+86400)
			}
		}

		// 每日缓存：同一天不同时刻请求返回同一份数据（不重算）。
		again, err := service.Day(c.Latitude, c.Longitude, 0, at.Add(3*time.Hour), c.Timezone)
		if err != nil {
			t.Fatalf("%s: second Day() error: %v", c.Name, err)
		}
		if again.ComputedAt != day.ComputedAt {
			t.Errorf("%s: 同日缓存未命中（应每日只算一次）", c.Name)
		}
		// 次日应生成新的日期数据。
		next, err := service.Day(c.Latitude, c.Longitude, 0, at.Add(24*time.Hour), c.Timezone)
		if err != nil {
			t.Fatalf("%s: next-day Day() error: %v", c.Name, err)
		}
		if next.Date == day.Date {
			t.Errorf("%s: 次日日期未推进：%q", c.Name, next.Date)
		}
	}

	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}
	data, err := json.MarshalIndent(map[string]any{"cases": cases}, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	path := filepath.Join("testdata", "moon_reference.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func TestMoonServiceTimezoneValidation(t *testing.T) {
	service := NewMoonService()
	if _, err := service.Day(31.23, 121.47, 0, time.Now(), "Not/AZone"); err != ErrInvalidTimezone {
		t.Fatalf("非法时区应返回 ErrInvalidTimezone，得到 %v", err)
	}
	if _, err := service.Day(200, 0, 0, time.Now(), "UTC"); err == nil {
		t.Fatalf("非法纬度应报错")
	}
	if _, err := service.Day(31.23, 121.47, 99999, time.Now(), "UTC"); err == nil {
		t.Fatalf("非法海拔应报错")
	}
}

func TestMoonElevationShiftsRise(t *testing.T) {
	service := NewMoonService()
	at := unixTime("2026-08-09T12:00:00Z")
	seaLevel, err := service.Day(31.23, 121.47, 0, at, "Asia/Shanghai")
	if err != nil {
		t.Fatalf("sea-level Day error: %v", err)
	}
	high, err := service.Day(31.23, 121.47, 3000, at, "Asia/Shanghai")
	if err != nil {
		t.Fatalf("high-elevation Day error: %v", err)
	}
	// 海拔升高使观察者提前看到月出：若两地都有月出，高处月出不应晚于海平面。
	if seaLevel.Moonrise != nil && high.Moonrise != nil && *high.Moonrise > *seaLevel.Moonrise {
		t.Errorf("海拔 3000m 月出应不晚于海平面：高= %d，海平面= %d", *high.Moonrise, *seaLevel.Moonrise)
	}
	// 海拔变化应产生不同缓存条目。
	again, err := service.Day(31.23, 121.47, 3000, at, "Asia/Shanghai")
	if err != nil {
		t.Fatalf("repeat high-elevation Day error: %v", err)
	}
	if again.ComputedAt != high.ComputedAt {
		t.Errorf("同海拔同日应命中缓存")
	}
}

func TestMoonPhaseShape(t *testing.T) {
	// 2026-03-03 约为满月（相位接近 180），2026-01-18 约为朔月（相位接近 0）。
	full := moonPhaseAt(unixTime("2026-03-03T12:00:00Z"))
	if full.Label != "满月" {
		t.Errorf("2026-03-03 应为满月，得到 %s（相位 %.1f°）", full.Label, full.Phase)
	}
	if full.Illumination < 0.97 {
		t.Errorf("满月亮面占比应接近 1，得到 %.3f", full.Illumination)
	}
}

func TestMoonPositionMeeusExample(t *testing.T) {
	// Meeus《Astronomical Algorithms》第 47 章算例 47.a：
	// 1992-04-12 0h TT，月球地心黄经 λ=133.162655°，黄纬 β=-3.229126°，距离 Δ=368409.7 km。
	at := unixTime("1992-04-12T00:00:00Z") // UT 与 TT 相差 <1 分钟，对本精度可忽略
	longitude, latitude, distance := moonPosition(julianCenturies(at))
	if math.Abs(longitude-133.162655) > 0.1 {
		t.Errorf("黄经 = %.4f°，期望 ≈133.1627°", longitude)
	}
	if math.Abs(latitude-(-3.229126)) > 0.1 {
		t.Errorf("黄纬 = %.4f°，期望 ≈-3.2291°", latitude)
	}
	if math.Abs(distance-368409.7) > 500 {
		t.Errorf("距离 = %.1f km，期望 ≈368409.7 km", distance)
	}
}

func unix(value string) int64 { return unixTime(value).Unix() }

func unixTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
