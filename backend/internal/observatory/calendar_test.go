package observatory

import (
	"testing"
	"time"
)

func TestCoreCalendarEventsIncludesExpectedFamilies(t *testing.T) {
	from := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	events := CoreCalendarEvents(from, to)
	if len(events) == 0 {
		t.Fatal("CoreCalendarEvents() returned no events")
	}
	want := map[string]bool{"new_moon": false, "full_moon": false, "lunar_perigee": false, "lunar_apogee": false, "ascending_node": false, "descending_node": false, "september_equinox": false}
	for _, event := range events {
		if _, ok := want[event.Kind]; ok {
			want[event.Kind] = true
		}
		if event.At.Before(from) || !event.At.Before(to) {
			t.Errorf("event %s outside requested range: %s", event.Kind, event.At)
		}
	}
	for kind, found := range want {
		if !found {
			t.Errorf("missing %s", kind)
		}
	}
}
