package astronomyevent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type sourceSnapshotStoreStub struct {
	started   []string
	finished  int
	snapshots []SourceSnapshot
	replaced  [][]Event
}

func (s *sourceSnapshotStoreStub) StartSourceSync(_ context.Context, source string) (int64, error) {
	s.started = append(s.started, source)
	return int64(len(s.started)), nil
}

func (s *sourceSnapshotStoreStub) FinishSourceSync(_ context.Context, _ int64, _ int, _ error) error {
	s.finished++
	return nil
}

func (s *sourceSnapshotStoreStub) SaveSourceSnapshot(_ context.Context, snapshot SourceSnapshot) (bool, error) {
	s.snapshots = append(s.snapshots, snapshot)
	return true, nil
}

func (s *sourceSnapshotStoreStub) ReplaceExternalForecast(_ context.Context, _ string, _ string, events []Event) (int, error) {
	s.replaced = append(s.replaced, events)
	return len(events), nil
}

type roundTripper func(*http.Request) (*http.Response, error)

func (fn roundTripper) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func TestSourceSyncerRecordsOfficialSourceSnapshots(t *testing.T) {
	store := &sourceSnapshotStoreStub{}
	syncer := NewSourceSyncer(store)
	syncer.now = func() time.Time { return time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC) }
	syncer.parseIMO = func(_ context.Context, body []byte, _ string, _ int) (parsedEvents, error) {
		return parsedEvents{payload: body, events: []Event{{ID: "meteor"}}}, nil
	}
	syncer.parseNASAGSFC = func(_ context.Context, body []byte, _ string) (parsedEvents, error) {
		return parsedEvents{payload: body, events: []Event{{ID: "eclipse"}}}, nil
	}
	syncer.client = &http.Client{Transport: roundTripper(func(request *http.Request) (*http.Response, error) {
		body := `{"endpoint":"` + request.URL.Path + `"}`
		if request.URL.Host == "ssd-api.jpl.nasa.gov" {
			body = `{"fields":["des","cd","dist"],"data":[["2026 AB","2026-Aug-12 17:00","0.01"]]}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}

	if err := syncer.SyncOfficialSources(context.Background()); err != nil {
		t.Fatalf("SyncOfficialSources() error = %v", err)
	}
	if got, want := len(store.snapshots), 9; got != want {
		t.Fatalf("snapshots = %d, want %d", got, want)
	}
	if got := string(store.snapshots[0].RawPayload); !strings.Contains(got, "endpoint") {
		t.Errorf("first snapshot = %s, want original JSON", got)
	}
	if got, want := store.finished, 9; got != want {
		t.Errorf("finished = %d, want %d", got, want)
	}
}

func TestParseJPLCloseApproachesUsesSchemaFields(t *testing.T) {
	body := []byte(`{"fields":["des","cd","dist","v_rel","h"],"data":[["2026 AB","2026-Aug-12 17:00","0.01","12.3","25.1"]]}`)
	parsed, err := parseJPLCloseApproaches(context.Background(), body, "https://example.test/cad", 2026)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(parsed.events), 1; got != want {
		t.Fatalf("events=%d, want %d", got, want)
	}
	event := parsed.events[0]
	if event.Kind != "small_body_close_approach" || event.Origin != "external_forecast" {
		t.Errorf("event=%+v, want external small-body forecast", event)
	}
	if got, want := event.StartsAt.Format(time.RFC3339), "2026-08-12T17:00:00Z"; got != want {
		t.Errorf("startsAt=%s, want %s", got, want)
	}
}

func TestSourceParseFailureKeepsExistingExternalForecast(t *testing.T) {
	store := &sourceSnapshotStoreStub{}
	syncer := NewSourceSyncer(store)
	syncer.parseNASAGSFC = func(_ context.Context, body []byte, _ string) (parsedEvents, error) {
		return parsedEvents{payload: body}, errors.New("source schema changed")
	}
	syncer.client = &http.Client{Transport: roundTripper(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("<html>changed</html>"))}, nil
	})}
	err := syncer.syncFeed(context.Background(), sourceFeed{SourceCode: "nasa_gsfc_eclipse", URL: "https://example.test/nasa"}, 2026)
	if err == nil {
		t.Fatal("syncFeed error=nil, want parse failure")
	}
	if got := len(store.replaced); got != 0 {
		t.Fatalf("ReplaceExternalForecast calls=%d, old events must not be cleared on parse failure", got)
	}
	if got := len(store.snapshots); got != 1 {
		t.Fatalf("snapshots=%d, parse failure must remain auditable", got)
	}
}

func TestParseIMOCalendarTextProducesAnnualForecast(t *testing.T) {
	text := `Perseids(007PER)Active:Jul17–Aug24;Maximum:August13,02h30mUT;ZHR=100;Radiant:α=48◦,δ=+58◦;Radiantdrift:seeTable6;`
	events := parseIMOCalendarText(text, "https://imo.example/cal2026.pdf", 2026)
	if got, want := len(events), 1; got != want {
		t.Fatalf("events=%d, want %d", got, want)
	}
	event := events[0]
	if got, want := event.StartsAt.Format(time.RFC3339), "2026-08-13T02:30:00Z"; got != want {
		t.Errorf("startsAt=%s, want %s", got, want)
	}
	if event.Origin != "external_forecast" || event.SourceCode != "imo_meteor_calendar" {
		t.Errorf("source=%s/%s, want external_forecast/imo_meteor_calendar", event.Origin, event.SourceCode)
	}
	var geometry map[string]any
	if err := json.Unmarshal(event.Geometry, &geometry); err != nil {
		t.Fatal(err)
	}
	if geometry["zhr"] != "100" || geometry["radiantRA"] != float64(48) || geometry["radiantDec"] != float64(58) {
		t.Errorf("geometry=%v, want ZHR and radiant coordinates", geometry)
	}
}

func TestSourceSnapshotPayloadHashesBinaryContentOnly(t *testing.T) {
	payload := sourceSnapshotPayload([]byte("%PDF-1.7"), "application/pdf")
	if !strings.Contains(string(payload), `"encoding":"base64"`) || !strings.Contains(string(payload), `"JVBERi0xLjc="`) {
		t.Errorf("payload = %s, want encoded raw content", payload)
	}
}

func TestParseUSNOMoonPhasesUsesReturnedTime(t *testing.T) {
	parsed, err := parseUSNOMoonPhases(context.Background(), []byte(`{"year":2026,"phasedata":[{"year":2026,"month":8,"day":28,"phase":"Full Moon","time":"04:14"}]}`), "https://example.test/usno", 2026)
	if err != nil {
		t.Fatalf("parseUSNOMoonPhases() error = %v", err)
	}
	if got, want := len(parsed.events), 1; got != want {
		t.Fatalf("events = %d, want %d", got, want)
	}
	if got, want := parsed.events[0].StartsAt.Format(time.RFC3339), "2026-08-28T04:14:00Z"; got != want {
		t.Errorf("startsAt = %s, want %s", got, want)
	}
}

func TestParseUSNOSeasonsAcceptsActualSchema(t *testing.T) {
	parsed, err := parseUSNOSeasons(context.Background(), []byte(`{"year":2026,"data":[{"year":2026,"month":3,"day":20,"phenom":"Equinox","time":"14:46"}]}`), "https://example.test/usno-seasons", 2026)
	if err != nil {
		t.Fatalf("parseUSNOSeasons() error = %v", err)
	}
	if got, want := parsed.coverageStart.Format("2006-01-02"), "2026-01-01"; got != want {
		t.Errorf("coverageStart=%s, want %s", got, want)
	}
	if !json.Valid(parsed.payloadJSON) {
		t.Errorf("payloadJSON is not valid JSON: %s", parsed.payloadJSON)
	}
}

func TestHTMLAndRSSPayloadsAreStoredAsValidJSONSnapshots(t *testing.T) {
	parsers := []func(context.Context, []byte, string) (parsedEvents, error){
		func(ctx context.Context, body []byte, url string) (parsedEvents, error) {
			return parseNASAGSFCEclipsePage(ctx, body, url)
		},
		func(ctx context.Context, body []byte, url string) (parsedEvents, error) {
			return parseNASAEclipsePathPage(ctx, body, url)
		},
		func(ctx context.Context, body []byte, url string) (parsedEvents, error) {
			return parseIMORSS(ctx, body, url, 2026)
		},
		func(ctx context.Context, body []byte, url string) (parsedEvents, error) {
			return parseIAUMDC(ctx, body, url, 2026)
		},
	}
	for _, parse := range parsers {
		parsed, err := parse(context.Background(), []byte("<html><body>source</body></html>"), "https://example.test/source")
		if err != nil {
			t.Fatalf("parser error = %v", err)
		}
		if len(parsed.payloadJSON) != 0 && !json.Valid(parsed.payloadJSON) {
			t.Fatalf("HTML parser returned a non-JSON payload: %q", parsed.payloadJSON)
		}
		payload := sourceSnapshotPayload(parsed.payload, "text/html")
		if !json.Valid(payload) {
			t.Fatalf("snapshot payload is not valid JSON: %s", payload)
		}
	}
}

func TestParseIAUMDCStoresNormalizedCatalogInSnapshot(t *testing.T) {
	body := []byte(`"00008"|"00007"|"006"|"PER"|" 1"|"2009-mm-dd"|"Perseids"|" annual "|""|""|"140.0"|"48.0"|"58.0"`)
	parsed, err := parseIAUMDC(context.Background(), body, "https://example.test/iau", 2026)
	if err != nil {
		t.Fatalf("parseIAUMDC() error = %v", err)
	}
	var payload struct {
		Format  string `json:"format"`
		Records []struct {
			Code       string  `json:"code"`
			RadiantRA  float64 `json:"radiantRA"`
			RadiantDec float64 `json:"radiantDec"`
		} `json:"records"`
	}
	if err := json.Unmarshal(parsed.payloadJSON, &payload); err != nil {
		t.Fatalf("decode IAU snapshot: %v", err)
	}
	if payload.Format != "iau_mdc_established_showers" || len(payload.Records) != 1 {
		t.Fatalf("IAU snapshot = %+v, want one normalized record", payload)
	}
	if record := payload.Records[0]; record.Code != "PER" || record.RadiantRA != 48 || record.RadiantDec != 58 {
		t.Errorf("record = %+v, want Perseids radiant", record)
	}
}

func TestParseNASAGSFCEclipsePageReadsCatalogTime(t *testing.T) {
	body := []byte(`<table><tr><td><a href="../LEplot/LEplot2001/LE2026Aug28P.pdf"> 2026 Aug 28 </a></td><td> 04:14:04 </td><td>Partial</td><td><a href="../LEsaros/LEsaros138.html"> 138 </a></td><td>0.927</td></tr></table>`)
	parsed, err := parseNASAGSFCEclipsePage(context.Background(), body, "https://eclipse.gsfc.nasa.gov/LEdecade/LEdecade2021.html")
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.events) != 1 {
		t.Fatalf("events=%d, want 1", len(parsed.events))
	}
	if got, want := parsed.events[0].StartsAt.Format(time.RFC3339), "2026-08-28T04:14:04Z"; got != want {
		t.Errorf("startsAt=%s, want %s", got, want)
	}
}
