package observatory

import (
	"math"
	"testing"
	"time"
)

func TestEquatorialHorizontalAzimuthRemainsFiniteNearPoles(t *testing.T) {
	at := time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC)
	for _, latitude := range []float64{-90, -89.999999, 89.999999, 90} {
		for _, declination := range []float64{-90, -89.999999, 0, 89.999999, 90} {
			altitude, azimuth := EquatorialCoordinatesToHorizontal(
				EquatorialCoordinates{RightAscensionDegrees: 123.456, DeclinationDegrees: declination},
				latitude, 45, at,
			)
			if math.IsNaN(altitude) || math.IsInf(altitude, 0) {
				t.Fatalf("latitude=%v declination=%v altitude=%v", latitude, declination, altitude)
			}
			if math.IsNaN(azimuth) || math.IsInf(azimuth, 0) || azimuth < 0 || azimuth >= 360 {
				t.Fatalf("latitude=%v declination=%v azimuth=%v", latitude, declination, azimuth)
			}
		}
	}
}
