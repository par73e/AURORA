package observatory

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
		fmt.Fprint(writer, `{"timezone":"Asia/Shanghai","hourly":{"time":[]}}`)
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
