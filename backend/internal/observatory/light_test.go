package observatory

import (
	"testing"
)

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
