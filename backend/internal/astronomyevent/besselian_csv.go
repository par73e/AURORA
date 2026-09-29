package astronomyevent

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
)

// parseNASABesselianCSV keeps only near-future solar eclipses from NASA/GSFC's
// five-millennium Besselian table. The original CSV is retained as a snapshot.
func parseNASABesselianCSV(_ context.Context, body []byte, sourceURL string, year int) (parsedEvents, error) {
	reader := csv.NewReader(bytes.NewReader(body))
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return parsedEvents{payload: body}, fmt.Errorf("read Besselian header: %w", err)
	}
	columns := make(map[string]int, len(header))
	for index, name := range header {
		columns[strings.TrimSpace(name)] = index
	}
	required := []string{"year", "month", "day", "td_ge", "dt", "eclipse_type", "t0", "tmin", "tmax", "tan_f1", "tan_f2"}
	for _, prefix := range []string{"x", "y"} {
		for order := 0; order <= 3; order++ {
			required = append(required, fmt.Sprintf("%s%d", prefix, order))
		}
	}
	for _, prefix := range []string{"d", "mu", "l1", "l2"} {
		for order := 0; order <= 2; order++ {
			required = append(required, fmt.Sprintf("%s%d", prefix, order))
		}
	}
	for _, name := range required {
		if _, ok := columns[name]; !ok {
			return parsedEvents{payload: body}, fmt.Errorf("Besselian column %q missing", name)
		}
	}
	var events []Event
	for {
		row, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return parsedEvents{payload: body}, fmt.Errorf("read Besselian row: %w", readErr)
		}
		field := func(name string) string {
			index := columns[name]
			if index >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[index])
		}
		eventYear, err := strconv.Atoi(field("year"))
		if err != nil || eventYear < year-1 || eventYear > year+2 {
			continue
		}
		month, monthErr := strconv.Atoi(field("month"))
		day, dayErr := strconv.Atoi(field("day"))
		if monthErr != nil || dayErr != nil || month < 1 || month > 12 || day < 1 || day > 31 {
			return parsedEvents{payload: body}, fmt.Errorf("invalid Besselian eclipse date %d-%d-%d", eventYear, month, day)
		}
		clock, err := time.Parse("15:04:05", field("td_ge"))
		if err != nil {
			return parsedEvents{payload: body}, fmt.Errorf("invalid greatest eclipse time: %w", err)
		}
		at := time.Date(eventYear, time.Month(month), day, clock.Hour(), clock.Minute(), clock.Second(), 0, time.UTC)
		coefficients := make(map[string]float64, len(required)-6)
		for _, name := range required {
			switch name {
			case "year", "month", "day", "td_ge", "eclipse_type":
				continue
			}
			value, parseErr := strconv.ParseFloat(field(name), 64)
			if parseErr != nil || math.IsNaN(value) || math.IsInf(value, 0) {
				return parsedEvents{payload: body}, fmt.Errorf("invalid Besselian coefficient %s for %s", name, at.Format(time.DateOnly))
			}
			coefficients[name] = value
		}
		eclipseType := map[string]string{"T": "Total", "A": "Annular", "P": "Partial", "H": "Hybrid"}[field("eclipse_type")]
		if eclipseType == "" {
			return parsedEvents{payload: body}, fmt.Errorf("unknown Besselian eclipse type %q", field("eclipse_type"))
		}
		geometry, _ := json.Marshal(map[string]any{
			"precision": "nasa_gsfc_besselian", "eclipseType": eclipseType,
			"besselian": coefficients, "besselianDay": at.Format(time.DateOnly),
		})
		events = append(events, Event{
			ID:   fmt.Sprintf("nasa-gsfc-besselian-solar_eclipse-%s", at.Format("20060102")),
			Kind: "solar_eclipse", Title: eclipseType + "·日食", TitleEN: "SOLAR ECLIPSE " + eclipseType,
			StartsAt: at.Add(-time.Duration(math.Round(coefficients["dt"]*1000)) * time.Millisecond), DateLabel: at.Format("2006年1月2日"),
			Summary: "NASA/GSFC 贝塞尔根数预测的日食；当地食相由观测坐标计算。",
			Origin:  "external_forecast", SourceCode: "nasa_gsfc_eclipse", SourceURL: sourceURL,
			VerifiedAt: time.Now().UTC().Format(time.DateOnly), Geometry: geometry,
		})
	}
	if len(events) == 0 {
		return parsedEvents{payload: body}, fmt.Errorf("Besselian CSV contains no near-future eclipses")
	}
	coverageStart := time.Date(year-1, time.January, 1, 0, 0, 0, 0, time.UTC)
	coverageEnd := coverageStart.AddDate(4, 0, 0)
	return parsedEvents{events: events, payload: body, coverageStart: &coverageStart, coverageEnd: &coverageEnd}, nil
}
