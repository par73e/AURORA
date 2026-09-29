package astronomyevent

import (
	"testing"
	"time"
)

// coreEvents 是 EphemerisSyncer 唯一 computed 写入路径中负责核心月球/季节事件的部分，
// 这里直接覆盖 coreEvents 的输出，不再经过已删除的独立 Generator 路径。
func TestCoreEventsBuildsEighteenMonthsOfComputedEvents(t *testing.T) {
	from := time.Date(2026, time.August, 11, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 18, 0)
	verifiedAt := time.Date(2026, time.August, 11, 12, 0, 0, 0, time.UTC)

	events := coreEvents(from, to, verifiedAt)
	if len(events) < 100 {
		t.Fatalf("events = %d, want a full 18-month calendar", len(events))
	}
	for _, event := range events {
		if event.Origin != "computed" || event.SourceURL != auroraModelSourceURL {
			t.Errorf("unexpected generated event metadata: %#v", event)
			break
		}
	}
}

func TestAuroraModelSourceLinksToItsImplementation(t *testing.T) {
	const oldURL = "https://aurora.local/astronomy-model"
	const codeURL = "https://github.com/par73e/AURORA/blob/main/backend/internal/observatory/calendar.go"
	if auroraModelSourceURL != codeURL {
		t.Fatalf("new model source URL = %q, want %q", auroraModelSourceURL, codeURL)
	}
	if got := ReadableSourceURL("aurora_astronomy_model", oldURL); got != codeURL {
		t.Fatalf("stored model source URL = %q, want %q", got, codeURL)
	}
	if got := ReadableSourceURL("other_source", oldURL); got != oldURL {
		t.Fatalf("unrelated source URL changed to %q", got)
	}
}
