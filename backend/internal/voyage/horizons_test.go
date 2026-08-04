package voyage

import (
	"testing"
	"time"
)

// TestParseEphemerisEnd 星历截止日期解析（Horizons 错误正文），
// 正文取自 STEREO-A 实测返回（星历止于 2026-10-23）。
func TestParseEphemerisEnd(t *testing.T) {
	body := "*******************************************************************************\n" +
		"No ephemeris for target \"STEREO-A (spacecraft)\" after A.D. 2026-OCT-23 00:01:09.1824 TDB\n" +
		"*******************************************************************************\n"
	got := parseEphemerisEnd(body)
	want := time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("parseEphemerisEnd = %v, want %v", got, want)
	}

	// 无截止标记（普通错误）→ 零值
	if !parseEphemerisEnd("No ephemeris for target (unknown)").IsZero() {
		t.Error("无 'after A.D. <date>' 时应返回零值")
	}
	// 日期格式非法 → 零值
	if !parseEphemerisEnd("No ephemeris for target after A.D. 20XX-XXX-99").IsZero() {
		t.Error("日期格式非法时应返回零值")
	}
	// 空正文 → 零值
	if !parseEphemerisEnd("").IsZero() {
		t.Error("空正文应返回零值")
	}
}

// TestParseSOESamples 采样解析冒烟：CSV 表头 + 单行数据
func TestParseSOESamples(t *testing.T) {
	text := "$$SOE\n" +
		"2459183.500000000, A.D. 2026-May-06 00:00:00.0000,  1.234567e+08, -5.678901e+07,  1.000000e+06\n" +
		"2459184.500000000, A.D. 2026-May-07 00:00:00.0000,  1.240000e+08, -5.700000e+07,  1.010000e+06\n" +
		"$$EOE\n"
	samples, err := parseSOESamples(text)
	if err != nil {
		t.Fatalf("parseSOESamples: %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("采样数 = %d, want 2", len(samples))
	}
	if samples[0].X != 1.234567e8 || samples[1].Y != -5.7e7 {
		t.Errorf("坐标解析错误: %+v", samples)
	}
	if samples[0].Epoch.Format("2006-01-02") != "2026-05-06" {
		t.Errorf("历元解析错误: %v", samples[0].Epoch)
	}
}
