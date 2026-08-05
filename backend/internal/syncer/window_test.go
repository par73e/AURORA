package syncer

import "testing"

func TestProbeSampleWindowDays(t *testing.T) {
	cases := map[string]int{
		"parker":  90,
		"jwst":    90,
		"spitzer": 200, // 真实周期约 377 天：±200 覆盖 400 天（完整一圈 + 余量），保证椭圆拟合可靠
		"unknown": 90,
	}
	for id, want := range cases {
		if got := probeSampleWindowDays(id); got != want {
			t.Errorf("probeSampleWindowDays(%q) = %d, want %d", id, got, want)
		}
	}
}
