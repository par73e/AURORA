package astronomyevent

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
)

func TestParseHorizonsVectorSamples(t *testing.T) {
	text := `header
$$SOE
2460000.500000000, A.D. 2026-Aug-11 00:00:00.0000,  1.250000000000E+00, -2.500000000000E-01,  3.000000000000E-02,
$$EOE`
	samples, err := parseHorizonsVectorSamples("venus", "https://example.test", text)
	if err != nil {
		t.Fatalf("parseHorizonsVectorSamples() error = %v", err)
	}
	if got, want := len(samples), 1; got != want {
		t.Fatalf("samples = %d, want %d", got, want)
	}
	if sample := samples[0]; sample.Body != "venus" || sample.XAU != 1.25 || sample.YAU != -0.25 || sample.ZAU != 0.03 {
		t.Errorf("sample = %#v", sample)
	}
}

func TestPlanetaryEventsFindsConjunctionCandidate(t *testing.T) {
	base := time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC)
	day := func(body string, degrees []float64) []EphemerisSample {
		items := make([]EphemerisSample, 0, len(degrees))
		for i, degree := range degrees {
			radians := degree * math.Pi / 180
			items = append(items, EphemerisSample{Body: body, Epoch: base.AddDate(0, 0, i), XAU: math.Cos(radians), YAU: math.Sin(radians)})
		}
		return items
	}
	items := planetaryEvents(map[string][]EphemerisSample{
		"mercury": day("mercury", []float64{10, 11, 12}),
		"venus":   day("venus", []float64{14, 11.1, 15}),
		"mars":    day("mars", []float64{80, 81, 82}), "jupiter": day("jupiter", []float64{120, 121, 122}), "saturn": day("saturn", []float64{160, 161, 162}), "uranus": day("uranus", []float64{200, 201, 202}), "neptune": day("neptune", []float64{240, 241, 242}), "sun": day("sun", []float64{0, 1, 2}),
	}, base)
	for _, item := range items {
		if item.Kind != "planetary_conjunction" {
			continue
		}
		var geometry map[string]any
		_ = json.Unmarshal(item.Geometry, &geometry)
		objects, _ := geometry["objects"].([]any)
		if len(objects) == 2 && objects[0] == "mercury" && objects[1] == "venus" {
			return
		}
	}
	t.Fatalf("planetaryEvents() did not find Mercury-Venus conjunction: %#v", items)
}

func TestParseHorizonsVectorSamplesRejectsMissingSection(t *testing.T) {
	_, err := parseHorizonsVectorSamples("venus", "", "not ephemeris")
	if err == nil || !strings.Contains(err.Error(), "SOE") {
		t.Fatalf("error = %v, want SOE failure", err)
	}
}
