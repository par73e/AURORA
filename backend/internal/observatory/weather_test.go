package observatory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
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

func TestConditionsDegradesToStaleSnapshotWhenUpstreamFails(t *testing.T) {
	var fail atomic.Bool
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if fail.Load() {
			http.Error(w, "upstream down", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/air" {
			fmt.Fprint(w, `{"hourly":{"time":[]}}`)
			return
		}
		fmt.Fprint(w, validWeatherJSON)
	}))
	defer server.Close()
	base := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	now := base
	client := NewClientWithURLs(server.URL+"/weather", server.URL+"/air", server.Client())
	client.now = func() time.Time { return now }

	fresh, err := client.Conditions(context.Background(), 31.2304, 121.4737)
	if err != nil || fresh.Degraded != nil {
		t.Fatalf("fresh fetch: err=%v degraded=%+v", err, fresh.Degraded)
	}

	// 新鲜窗口（1h）已过、宽限（6h）内：上游失败应返回旧预报并标记降级。
	now = base.Add(2 * time.Hour)
	fail.Store(true)
	stale, err := client.Conditions(context.Background(), 31.23049, 121.4737)
	if err != nil {
		t.Fatalf("degraded fetch returned error: %v", err)
	}
	if stale.Degraded == nil || stale.Degraded.Reason != "upstream_unavailable" {
		t.Fatalf("missing degraded marker: %+v", stale.Degraded)
	}
	if stale.Timezone != fresh.Timezone || stale.RetrievedAt != fresh.RetrievedAt || stale.Elevation != fresh.Elevation {
		t.Fatal("degraded snapshot must preserve the last successful forecast")
	}

	// 冷却期内不再打扰上游：直接降级返回。
	now = base.Add(2*time.Hour + 10*time.Second)
	before := hits.Load()
	cooldown, err := client.Conditions(context.Background(), 31.23049, 121.4737)
	if err != nil || cooldown.Degraded == nil {
		t.Fatalf("cooldown fetch: err=%v degraded=%+v", err, cooldown.Degraded)
	}
	if hits.Load() != before {
		t.Fatalf("cooldown window still contacted upstream: %d new hits", hits.Load()-before)
	}

	// 宽限过期后不再提供旧预报。
	now = base.Add(8 * time.Hour)
	if _, err := client.Conditions(context.Background(), 31.23049, 121.4737); err == nil {
		t.Fatal("forecast older than the stale grace was served as valid")
	}
}

func TestConditionsDegradesWhenUpstreamTimesOut(t *testing.T) {
	var fail atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			// 模拟上游挂起：超过客户端超时预算才答复。
			time.Sleep(2 * time.Second)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/air" {
			fmt.Fprint(w, `{"hourly":{"time":[]}}`)
			return
		}
		fmt.Fprint(w, validWeatherJSON)
	}))
	defer server.Close()
	base := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	now := base
	client := NewClientWithURLs(server.URL+"/weather", server.URL+"/air", &http.Client{Timeout: 200 * time.Millisecond})
	client.now = func() time.Time { return now }
	if _, err := client.Conditions(context.Background(), 40.1, 116.1); err != nil {
		t.Fatalf("seed fetch: %v", err)
	}
	fail.Store(true)
	now = base.Add(2 * time.Hour)
	start := time.Now()
	stale, err := client.Conditions(context.Background(), 40.1, 116.1)
	if err != nil || stale.Degraded == nil {
		t.Fatalf("upstream timeout did not degrade: err=%v degraded=%+v", err, stale.Degraded)
	}
	if elapsed := time.Since(start); elapsed > 1500*time.Millisecond {
		t.Fatalf("timeout budget not respected: %v", elapsed)
	}
}

func TestConditionsWithoutStaleSnapshotStillFails(t *testing.T) {	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream down", http.StatusBadGateway)
	}))
	defer server.Close()
	client := NewClientWithURLs(server.URL, server.URL, server.Client())
	if _, err := client.Conditions(context.Background(), 31.23, 121.47); err == nil {
		t.Fatal("failure without any cached snapshot was hidden")
	}
}

func TestConditionsSurvivesAirQualityFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/air" {
			http.Error(w, "air quality down", http.StatusBadGateway)
			return
		}
		fmt.Fprint(w, validWeatherJSON)
	}))
	defer server.Close()
	client := NewClientWithURLs(server.URL+"/weather", server.URL+"/air", server.Client())
	conditions, err := client.Conditions(context.Background(), 31.23, 121.47)
	if err != nil {
		t.Fatalf("air quality failure broke weather: %v", err)
	}
	if len(conditions.AirQuality) != 0 {
		t.Fatalf("unexpected air quality payload: %v", conditions.AirQuality)
	}
}

func TestConditionsFetchesUpstreamsConcurrently(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(400 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/air" {
			fmt.Fprint(w, `{"hourly":{"time":[]}}`)
			return
		}
		fmt.Fprint(w, validWeatherJSON)
	}))
	defer server.Close()
	client := NewClientWithURLs(server.URL+"/weather", server.URL+"/air", server.Client())
	start := time.Now()
	if _, err := client.Conditions(context.Background(), 31.23, 121.47); err != nil {
		t.Fatalf("conditions error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 750*time.Millisecond {
		t.Fatalf("upstream requests look serial: %v", elapsed)
	}
}
