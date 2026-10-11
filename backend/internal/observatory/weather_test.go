package observatory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const validWeatherJSON = `{"timezone":"Asia/Shanghai","current":{"time":"2026-10-11T20:00","temperature_2m":18,"dew_point_2m":12,"cloud_cover":0,"cloud_cover_low":0,"cloud_cover_mid":0,"cloud_cover_high":0,"visibility":20000,"relative_humidity_2m":60,"precipitation":0,"wind_speed_10m":2,"wind_gusts_10m":4,"weather_code":0},"hourly":{"time":["2026-10-11T20:00"],"temperature_2m":[18],"dew_point_2m":[12],"cloud_cover":[0],"cloud_cover_low":[0],"cloud_cover_mid":[0],"cloud_cover_high":[0],"visibility":[20000],"relative_humidity_2m":[60],"precipitation":[0],"wind_speed_10m":[2],"wind_gusts_10m":[4],"weather_code":[0]}}`

func TestForecastRequestsUseTwoCalendarDays(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		if got := query.Get("forecast_days"); got != "2" {
			t.Errorf("forecast_days = %q，期望 2", got)
		}
		if got := query.Get("forecast_hours"); got != "" {
			t.Errorf("不应继续请求滚动 forecast_hours，得到 %q", got)
		}
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/air" {
			fmt.Fprint(writer, `{"hourly":{"time":[]}}`)
			return
		}
		fmt.Fprint(writer, validWeatherJSON)
	}))
	defer server.Close()

	client := NewClientWithURLs(server.URL+"/weather", server.URL+"/air", server.Client())
	if _, err := client.weather(context.Background(), 31.23, 121.47); err != nil {
		t.Fatalf("weather error: %v", err)
	}
	if _, err := client.airQuality(context.Background(), 31.23, 121.47); err != nil {
		t.Fatalf("air quality error: %v", err)
	}
}

func TestWeatherMissingFactorsNeverBecomeZero(t *testing.T) {
	for _, tc := range []struct{ name, field, value string }{
		{"null cloud", "cloud_cover", "null"},
		{"null precipitation", "precipitation", "null"},
		{"null weather code", "weather_code", "null"},
		{"null cloud layer", "cloud_cover_low", "null"},
		{"short array", "cloud_cover", "[]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var payload map[string]json.RawMessage
			if err := json.Unmarshal([]byte(validWeatherJSON), &payload); err != nil {
				t.Fatal(err)
			}
			var hourly map[string]json.RawMessage
			if err := json.Unmarshal(payload["hourly"], &hourly); err != nil {
				t.Fatal(err)
			}
			if tc.value == "[]" {
				hourly[tc.field] = json.RawMessage(tc.value)
			} else {
				hourly[tc.field] = json.RawMessage("[" + tc.value + "]")
			}
			payload["hourly"], _ = json.Marshal(hourly)
			raw, _ := json.Marshal(payload)
			if err := validateWeatherPayload(raw); err == nil {
				t.Fatal("incomplete weather was accepted")
			}
		})
	}
	if err := validateWeatherPayload([]byte(validWeatherJSON)); err != nil {
		t.Fatalf("real zero was rejected: %v", err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(validWeatherJSON), &payload); err != nil {
		t.Fatal(err)
	}
	var current map[string]json.RawMessage
	if err := json.Unmarshal(payload["current"], &current); err != nil {
		t.Fatal(err)
	}
	current["cloud_cover"] = json.RawMessage("null")
	payload["current"], _ = json.Marshal(current)
	raw, _ := json.Marshal(payload)
	if err := validateWeatherPayload(raw); err == nil {
		t.Fatal("null current cloud cover was accepted")
	}
}

func TestConditionsRejectsIncompleteForecast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"timezone":"Asia/Shanghai","current":{"time":"2026-10-11T20:00","cloud_cover":null},"hourly":{"time":["2026-10-11T20:00"],"cloud_cover":[null],"precipitation":[null],"weather_code":[null]}}`))
	}))
	defer server.Close()
	client := NewClientWithURLs(server.URL, server.URL, server.Client())
	if _, err := client.Conditions(context.Background(), 31.23, 121.47); err == nil {
		t.Fatal("incomplete forecast was exposed as valid conditions")
	}
}
