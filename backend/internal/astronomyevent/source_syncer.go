package astronomyevent

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const sourceResponseMaximumBytes = 5 << 20

type sourceFeed struct {
	SourceCode string
	URL        string
	Year       int
}

// parsedEvents 是一次外部资料解析的产物。
// events 为解析得到的事件（可为空）；payload 是用于写快照的原始字节；
// coverage 描述这批资料覆盖的时间范围。
type parsedEvents struct {
	events        []Event
	payload       []byte
	payloadJSON   json.RawMessage
	coverageStart *time.Time
	coverageEnd   *time.Time
}

// SourceSyncer 把权威外部资料低频镜像为可审计的快照，并把可解析的资料写成事件。
// 解析失败时不清空旧事件，只记录失败来源。
type SourceSyncer struct {
	store         ExternalSourceStore
	client        *http.Client
	now           func() time.Time
	parseUSNO     func(ctx context.Context, body []byte, sourceURL string, year int) (parsedEvents, error)
	parseNASAGSFC func(ctx context.Context, body []byte, sourceURL string) (parsedEvents, error)
	parseIMO      func(ctx context.Context, body []byte, sourceURL string, year int) (parsedEvents, error)
}

// ExternalSourceStore 是外部资料同步所需的最小存储接口。
type ExternalSourceStore interface {
	SourceSnapshotStore
	ReplaceExternalForecast(ctx context.Context, sourceCode, sourceURL string, events []Event) (int, error)
}

func NewSourceSyncer(store ExternalSourceStore) *SourceSyncer {
	return &SourceSyncer{
		store:         store,
		client:        &http.Client{Timeout: 20 * time.Second},
		now:           time.Now,
		parseUSNO:     parseUSNOMoonPhases,
		parseNASAGSFC: parseNASAGSFCEclipsePage,
		parseIMO:      parseIMOMeteorCalendar,
	}
}

// SyncOfficialSources 每日检查 USNO、NASA/GSFC 和 IMO 的公开资料。
// 解析成功的事件以 external_forecast 写入；失败时保留旧事件，只记录失败来源。
func (s *SourceSyncer) SyncOfficialSources(ctx context.Context) error {
	if s == nil || s.store == nil {
		return errors.New("astronomy source snapshot store 未配置")
	}
	year := s.now().UTC().Year()
	eclipseDecade := (year-1)/10*10 + 1
	feeds := []sourceFeed{
		{SourceCode: "usno_astronomy", URL: fmt.Sprintf("https://aa.usno.navy.mil/api/moon/phases/year?year=%d", year), Year: year},
		{SourceCode: "usno_astronomy", URL: fmt.Sprintf("https://aa.usno.navy.mil/api/seasons?year=%d", year), Year: year},
		{SourceCode: "nasa_gsfc_eclipse", URL: fmt.Sprintf("https://eclipse.gsfc.nasa.gov/SEdecade/SEdecade%d.html", eclipseDecade)},
		{SourceCode: "nasa_gsfc_eclipse", URL: fmt.Sprintf("https://eclipse.gsfc.nasa.gov/SEdecade/SEdecade%d.html", eclipseDecade+10)},
		{SourceCode: "nasa_gsfc_eclipse", URL: "https://eclipse.gsfc.nasa.gov/LEdecade/LEdecade2021.html"},
		{SourceCode: "imo_meteor_calendar", URL: fmt.Sprintf("https://www.imo.net/files/meteor-shower/cal%d.pdf", year), Year: year},
		{SourceCode: "imo_meteor_calendar", URL: fmt.Sprintf("https://www.imo.net/files/meteor-shower/cal%d.pdf", year+1), Year: year + 1},
		{SourceCode: "imo_meteor_calendar", URL: "https://www.imo.net/feed/", Year: year},
		{SourceCode: "iau_mdc", URL: fmt.Sprintf("https://www.ta3.sk/IAUC22DB/MDC2022/Etc/streamestablisheddata%d.txt", year), Year: year},
		{SourceCode: "jpl_small_bodies", URL: fmt.Sprintf("https://ssd-api.jpl.nasa.gov/cad.api?date-min=%s&date-max=%s&dist-max=0.05&sort=dist", startOfUTCDay(s.now()).Format(time.DateOnly), startOfUTCDay(s.now()).AddDate(0, 18, 0).Format(time.DateOnly)), Year: year},
	}
	var failures []error
	for _, feed := range feeds {
		if err := s.syncFeed(ctx, feed, year); err != nil {
			failures = append(failures, err)
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("official astronomy source sync: %w", errors.Join(failures...))
	}
	return nil
}

func (s *SourceSyncer) syncFeed(ctx context.Context, feed sourceFeed, year int) (syncErr error) {
	if feed.Year != 0 {
		year = feed.Year
	}
	runID, err := s.store.StartSourceSync(ctx, feed.SourceCode)
	if err != nil {
		return err
	}
	records := 0
	defer func() { _ = s.store.FinishSourceSync(context.Background(), runID, records, syncErr) }()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feed.URL, nil)
	if err != nil {
		return fmt.Errorf("build %s request: %w", feed.SourceCode, err)
	}
	req.Header.Set("User-Agent", "AURORA astronomy event cache/1.0")
	response, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", feed.SourceCode, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("fetch %s: unexpected HTTP status %d", feed.SourceCode, response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, sourceResponseMaximumBytes+1))
	if err != nil {
		return fmt.Errorf("read %s: %w", feed.SourceCode, err)
	}
	if len(body) > sourceResponseMaximumBytes {
		return fmt.Errorf("read %s: response exceeds %d bytes", feed.SourceCode, sourceResponseMaximumBytes)
	}

	// 解析得到的事件（可为空）。解析失败不清空旧事件。
	parsed, parseErr := s.parseFeed(ctx, feed, body, year)
	if parseErr != nil {
		// 仍写快照以便审计，但不替换事件。
		s.saveSnapshot(ctx, feed, parsed, body, year)
		return fmt.Errorf("parse %s: %w", feed.SourceCode, parseErr)
	}
	if (feed.SourceCode == "nasa_gsfc_eclipse" || strings.HasSuffix(strings.ToLower(feed.URL), ".pdf")) && len(parsed.events) == 0 {
		s.saveSnapshot(ctx, feed, parsed, body, year)
		return fmt.Errorf("parse %s: source contained no usable events", feed.SourceCode)
	}
	// USNO 是 AURORA 本地月相/季节模型的交叉校验，不是独立的日历主数据。
	// 因而只保存可审计快照；绝不将其另写为会与 computed 事件重复的 external_forecast。
	if feed.SourceCode == "usno_astronomy" {
		parsed.events = nil
	}

	// 先写快照（去重），再按需替换 external_forecast 事件。
	inserted, err := s.saveSnapshot(ctx, feed, parsed, body, year)
	if err != nil {
		return err
	}
	if inserted {
		records = 1
	}
	if len(parsed.events) > 0 {
		n, replaceErr := s.store.ReplaceExternalForecast(ctx, feed.SourceCode, feed.URL, parsed.events)
		if replaceErr != nil {
			return fmt.Errorf("replace external forecast %s: %w", feed.SourceCode, replaceErr)
		}
		records += n
	}
	return nil
}

// parseFeed 根据来源分派解析器。
func (s *SourceSyncer) parseFeed(ctx context.Context, feed sourceFeed, body []byte, year int) (parsedEvents, error) {
	switch feed.SourceCode {
	case "usno_astronomy":
		if strings.Contains(feed.URL, "/seasons") {
			return parseUSNOSeasons(ctx, body, feed.URL, year)
		}
		return s.parseUSNO(ctx, body, feed.URL, year)
	case "nasa_gsfc_eclipse":
		return s.parseNASAGSFC(ctx, body, feed.URL)
	case "imo_meteor_calendar":
		if !strings.HasSuffix(strings.ToLower(feed.URL), ".pdf") {
			return parseIMORSS(ctx, body, feed.URL, year)
		}
		return s.parseIMO(ctx, body, feed.URL, year)
	case "iau_mdc":
		return parseIAUMDC(ctx, body, feed.URL, year)
	case "jpl_small_bodies":
		return parseJPLCloseApproaches(ctx, body, feed.URL, year)
	default:
		return parsedEvents{payload: body}, nil
	}
}

// saveSnapshot 写来源快照（内容哈希去重），返回是否为新插入。
func (s *SourceSyncer) saveSnapshot(ctx context.Context, feed sourceFeed, parsed parsedEvents, body []byte, year int) (bool, error) {
	payload := parsed.payloadJSON
	if payload == nil {
		payload = sourceSnapshotPayload(body, "application/octet-stream")
	}
	hash := sha256.Sum256(body)
	coverageStart := parsed.coverageStart
	coverageEnd := parsed.coverageEnd
	if coverageStart == nil {
		start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
		coverageStart = &start
	}
	if coverageEnd == nil {
		end := coverageStart.AddDate(1, 0, 0)
		coverageEnd = &end
	}
	return s.store.SaveSourceSnapshot(ctx, SourceSnapshot{
		SourceCode:     feed.SourceCode,
		CoverageStart:  coverageStart,
		CoverageEnd:    coverageEnd,
		SourceURL:      feed.URL,
		PayloadHash:    hex.EncodeToString(hash[:]),
		RawPayload:     payload,
		RecordsWritten: len(parsed.events),
	})
}

func sourceSnapshotPayload(body []byte, contentType string) json.RawMessage {
	if json.Valid(body) {
		return json.RawMessage(body)
	}
	payload, _ := json.Marshal(map[string]any{
		"contentType": contentType,
		"bytes":       len(body),
		"encoding":    "base64",
		"data":        base64.StdEncoding.EncodeToString(body),
	})
	return payload
}
