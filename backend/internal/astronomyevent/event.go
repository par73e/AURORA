package astronomyevent

import (
	"context"
	"encoding/json"
	"time"
)

// Event 是不依赖观察地点的全球天象事实。local visibility 由 HTTP 层在后续计算后附加，
// 不能把某一个地点的结论写回这里。
type Event struct {
	ID           string          `json:"id"`
	Kind         string          `json:"kind"`
	Title        string          `json:"title"`
	TitleEN      string          `json:"titleEn"`
	StartsAt     time.Time       `json:"startsAt"`
	EndsAt       *time.Time      `json:"endsAt,omitempty"`
	DateLabel    string          `json:"dateLabel"`
	Summary      string          `json:"summary"`
	Origin       string          `json:"origin"`
	SourceCode   string          `json:"-"`
	SourceName   string          `json:"sourceName"`
	SourceURL    string          `json:"sourceUrl"`
	VerifiedAt   string          `json:"verifiedAt"`
	Geometry     json.RawMessage `json:"geometry"`
	Presentation json.RawMessage `json:"presentation"`
}

// ReadableSourceURL replaces old non-browsable links in already stored events.
func ReadableSourceURL(sourceCode, sourceURL string) string {
	if sourceCode == "jpl_horizons_events" && sourceURL == horizonsEventsEndpoint {
		return horizonsEventsPageURL
	}
	if sourceCode == "aurora_astronomy_model" && sourceURL == "https://aurora.local/astronomy-model" {
		return auroraModelSourceURL
	}
	return sourceURL
}

type ListQuery struct {
	From time.Time
	To   time.Time
}

type SourceSnapshot struct {
	SourceCode     string
	CoverageStart  *time.Time
	CoverageEnd    *time.Time
	SourceURL      string
	PayloadHash    string
	RawPayload     json.RawMessage
	RecordsWritten int
}

// SourceStatus 汇总外部天象资料源最近一次尝试、最近一次成功和缓存覆盖范围。
// 它只暴露可用于界面判断新鲜度的信息，不包含原始响应。
type SourceStatus struct {
	Code           string     `json:"code"`
	Name           string     `json:"name"`
	URL            string     `json:"url"`
	LastAttemptAt  *time.Time `json:"lastAttemptAt,omitempty"`
	LastSuccessAt  *time.Time `json:"lastSuccessAt,omitempty"`
	Success        *bool      `json:"success,omitempty"`
	RecordsWritten int        `json:"recordsWritten"`
	Error          string     `json:"error,omitempty"`
	CoverageStart  *time.Time `json:"coverageStart,omitempty"`
	CoverageEnd    *time.Time `json:"coverageEnd,omitempty"`
}

// Store 让 HTTP 层不依赖 PostgreSQL，方便为日期边界和错误处理写单元测试。
type Store interface {
	List(context.Context, ListQuery) ([]Event, error)
}

type SourceStatusStore interface {
	ListSourceStatuses(context.Context) ([]SourceStatus, error)
}

// SourceSnapshotStore 是每日外部资料刷新所需的最小存储接口。
// 外部失败不会删除 event 表中的最后一份有效记录。
type SourceSnapshotStore interface {
	StartSourceSync(context.Context, string) (int64, error)
	FinishSourceSync(context.Context, int64, int, error) error
	SaveSourceSnapshot(context.Context, SourceSnapshot) (bool, error)
}

type ComputedEventStore interface {
	ReplaceComputed(context.Context, ListQuery, []Event) error
	StartSourceSync(context.Context, string) (int64, error)
	FinishSourceSync(context.Context, int64, int, error) error
}

type EphemerisSample struct {
	Body      string
	Epoch     time.Time
	XAU       float64
	YAU       float64
	ZAU       float64
	SourceURL string
}

type EphemerisStore interface {
	ComputedEventStore
	ReplaceEphemerisSamples(context.Context, string, time.Time, time.Time, []EphemerisSample) error
}
