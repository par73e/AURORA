package observatory

import (
	"context"
	"errors"
	"testing"
)

func TestLightClientNotConfigured(t *testing.T) {
	client := NewLightPollutionClient("")
	if _, err := client.Light(context.Background(), 31.23, 121.47); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("未配置 Key 应返回 ErrNotConfigured，得到 %v", err)
	}
}

func TestParseNumericBody(t *testing.T) {
	for input, want := range map[string]float64{
		`12.34`:           12.34,
		`"12.34"`:         12.34,
		`[12.34, 5.0]`:    12.34,
		`{"value": 3.14}`: 0, // 不支持的对象形态：返回 0 + false
	} {
		got, ok := parseNumericBody([]byte(input))
		if want == 0 {
			if ok {
				t.Errorf("parseNumericBody(%s) 应失败，得到 %v", input, got)
			}
			continue
		}
		if !ok || got != want {
			t.Errorf("parseNumericBody(%s) = %v/%v，期望 %v", input, got, ok, want)
		}
	}
}

func TestSQMToBortle(t *testing.T) {
	cases := map[float64]float64{
		22.0: 1, 21.6: 2, 21.4: 3, 20.8: 4, 19.5: 5, 18.5: 6, 17.6: 7, 17.0: 8, 16.0: 9,
	}
	for sqm, want := range cases {
		if got := sqmToBortle(sqm); got != want {
			t.Errorf("sqmToBortle(%.1f) = %.0f，期望 %.0f", sqm, got, want)
		}
	}
}
