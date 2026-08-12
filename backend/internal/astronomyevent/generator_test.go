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
