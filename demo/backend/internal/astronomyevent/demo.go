package astronomyevent

import "time"

// CoreEvents exposes the offline calendar for the standalone demo.
func CoreEvents(from, to time.Time) []Event { return coreEvents(from, to, time.Now()) }
