package httpapi

import (
	"aurora/backend/internal/astronomyevent"
	"aurora/backend/internal/orbit"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDemoCalendarWorksWithoutAnExternalSnapshot(t *testing.T) {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	events, e := (demoEvents{}).List(context.Background(), astronomyevent.ListQuery{From: from, To: to})
	if e != nil || len(events) < 4 {
		t.Fatal("offline month must include local calendar events")
	}
	for _, event := range events {
		if event.StartsAt.Before(from) || !event.StartsAt.Before(to) {
			t.Fatal("event out of query range")
		}
	}
}
func TestDemoCatalogFiltersAndPages(t *testing.T) {
	handler := demoCatalogHandler([]orbit.Spacecraft{{ID: "1", NameZH: "国际空间站", OperatorName: "NASA"}, {ID: "2", NameZH: "天宫", OperatorName: "CNSA"}})
	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest("GET", "/api/v1/orbit/spacecraft?q=天宫&pageSize=1", nil))
	var result orbit.SpacecraftPage
	if e := json.Unmarshal(recorder.Body.Bytes(), &result); e != nil {
		t.Fatal(e)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != "2" {
		t.Fatal(result)
	}
	recorder = httptest.NewRecorder()
	handler(recorder, httptest.NewRequest("GET", "/api/v1/orbit/spacecraft?mode=regex&q=%5B", nil))
	if recorder.Code != 400 {
		t.Fatal("invalid regex must return 400")
	}
}
