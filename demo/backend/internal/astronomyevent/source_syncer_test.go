package astronomyevent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	"aurora/backend/internal/observatory"
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

// Rows from NASA/GSFC's published Besselian CSV.
const besselianSample = `year,month,day,td_ge,dt,eclipse_type,t0,tmin,tmax,tan_f1,tan_f2,x0,x1,x2,x3,y0,y1,y2,y3,d0,d1,d2,mu0,mu1,mu2,l10,l11,l12,l20,l21,l22
2026,8,12,17:47:06,75.40000000,T,18.00000000,-3.00000000,3.00000000,.00461410,.00459110,.47551400,.51892490,-.00007730,-.00000804,.77118300,-.23016800,-.00012460,.00000377,14.79667000,-.01206500,-.00000300,88.74779000,15.00309000,.00000000,.53795500,.00009390,-.00001210,-.00814200,.00009350,-.00001210
2027,2,6,16:00:48,75.70000000,A,16.00000000,-3.00000000,3.00000000,.00474260,.00471900,.11167600,.46649520,-.00003370,-.00000527,-.27329300,.20318560,.00010250,-.00000246,-15.54794000,.01238300,.00000400,56.49307000,15.00051000,.00000000,.57192800,-.00006530,-.00001010,.02566200,-.00006500,-.00001000
2027,8,2,10:07:50,76.00000000,T,10.00000000,-3.00000000,3.00000000,.00460640,.00458340,-.01977200,.54471230,-.00004460,-.00000922,.16006100,-.21115820,-.00012170,.00000376,17.76247000,-.01018100,-.00000400,328.42255000,15.00210000,.00000000,.53059600,.00001380,-.00001280,-.01546400,.00001370,-.00001280
`

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
		if strings.HasSuffix(request.URL.Path, ".csv") {
			body = besselianSample
		}
		if request.URL.Host == "ssd-api.jpl.nasa.gov" {
			body = `{"fields":["des","cd","dist"],"data":[["2026 AB","2026-Aug-12 17:00","0.01"]]}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}

	if err := syncer.SyncOfficialSources(context.Background()); err != nil {
		t.Fatalf("SyncOfficialSources() error = %v", err)
	}
	if got, want := len(store.snapshots), 11; got != want {
		t.Fatalf("snapshots = %d, want %d", got, want)
	}
	if got := string(store.snapshots[0].RawPayload); !strings.Contains(got, "endpoint") {
		t.Errorf("first snapshot = %s, want original JSON", got)
	}
	if got, want := store.finished, 11; got != want {
		t.Errorf("finished = %d, want %d", got, want)
	}
}

func TestBesselianCSVProducesLocalNASAContactTimes(t *testing.T) {
	expectTime := func(label, got, want string) {
		t.Helper()
		actual, err := time.Parse(time.RFC3339, got)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := time.Parse(time.RFC3339, want)
		if err != nil {
			t.Fatal(err)
		}
		if delta := actual.Sub(expected); delta < -2*time.Second || delta > 2*time.Second {
			t.Errorf("%s=%s, NASA calculator=%s", label, got, want)
		}
	}
	parsed, err := parseNASABesselianCSV(context.Background(), []byte(besselianSample), "https://eclipse.gsfc.nasa.gov/eclipse_besselian_from_mysqldump2.csv", 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.events) != 3 {
		t.Fatalf("events=%d, want 3", len(parsed.events))
	}
	event := parsed.events[2]
	if got := event.StartsAt.Format(time.RFC3339); got != "2027-08-02T10:06:34Z" {
		t.Fatalf("greatest UT=%s", got)
	}
	var geometry map[string]any
	if err := json.Unmarshal(event.Geometry, &geometry); err != nil {
		t.Fatal(err)
	}
	vis := observatory.NewVisibilitySolver(nil).Solve(observatory.EventInput{Kind: event.Kind, StartsAt: event.StartsAt, Geometry: geometry}, 25.50236, 33.19617, "UTC")
	if vis.Status != "observable" || vis.EclipseContacts == nil || vis.EclipseContacts.Kind != "total" || vis.EclipseContacts.CentralBegin == nil || vis.EclipseContacts.CentralEnd == nil {
		t.Fatalf("local result=%+v", vis)
	}
	contacts := vis.EclipseContacts
	peak, err := time.Parse(time.RFC3339, contacts.Peak)
	if err != nil {
		t.Fatal(err)
	}
	if delta := peak.Sub(event.StartsAt); delta < -2*time.Second || delta > 2*time.Second {
		t.Errorf("local greatest eclipse differs from NASA's global point by %s", delta)
	}
	if vis.AltitudeDegrees == nil || math.Abs(*vis.AltitudeDegrees-81.7) > .2 {
		t.Errorf("sun altitude=%v, NASA gives 81.7 degrees", vis.AltitudeDegrees)
	}
	if vis.AzimuthDegrees == nil || math.Abs(*vis.AzimuthDegrees-202) > .2 {
		t.Errorf("sun azimuth=%v, NASA gives 202.0 degrees", vis.AzimuthDegrees)
	}
	if math.Abs(contacts.Magnitude-1.079) > .002 || contacts.ObscurationPercent != 100 {
		t.Errorf("magnitude/obscuration=%f/%f", contacts.Magnitude, contacts.ObscurationPercent)
	}
	begin, err := time.Parse(time.RFC3339, *contacts.CentralBegin)
	if err != nil {
		t.Fatal(err)
	}
	end, err := time.Parse(time.RFC3339, *contacts.CentralEnd)
	if err != nil {
		t.Fatal(err)
	}
	if delta := end.Sub(begin) - (6*time.Minute + 23*time.Second); delta < -3*time.Second || delta > 3*time.Second {
		t.Errorf("central duration=%s, NASA gives 6m23s", end.Sub(begin))
	}
	// NASA's 10:06 UT path table places the northern umbral edge near
	// 26°34.9′N, 33°44.1′E. Check points well inside/outside that edge.
	for _, example := range []struct {
		latitude, longitude float64
		kind                string
	}{
		{26.3, 33.735, "total"},
		{26.9, 33.735, "partial"},
	} {
		boundary := observatory.NewVisibilitySolver(nil).Solve(observatory.EventInput{Kind: event.Kind, StartsAt: event.StartsAt, Geometry: geometry}, example.latitude, example.longitude, "UTC")
		if boundary.EclipseContacts == nil || boundary.EclipseContacts.Kind != example.kind {
			t.Errorf("path side at %.3f,%.3f: %+v", example.latitude, example.longitude, boundary)
		}
	}
	annular := parsed.events[1]
	if err := json.Unmarshal(annular.Geometry, &geometry); err != nil {
		t.Fatal(err)
	}
	annularVisibility := observatory.NewVisibilitySolver(nil).Solve(observatory.EventInput{Kind: annular.Kind, StartsAt: annular.StartsAt, Geometry: geometry}, -31.30245, -48.45416, "UTC")
	if annularVisibility.EclipseContacts == nil || annularVisibility.EclipseContacts.Kind != "annular" {
		t.Errorf("annular eclipse result=%+v", annularVisibility)
	} else {
		annularContacts := annularVisibility.EclipseContacts
		if annularContacts.CentralBegin == nil || annularContacts.CentralEnd == nil {
			t.Fatal("annular central contacts missing")
		}
		expectTime("annular C1", annularContacts.PartialBegin, "2027-02-06T14:11:29Z")
		expectTime("annular peak", annularContacts.Peak, "2027-02-06T15:59:32Z")
		expectTime("annular C2", *annularContacts.CentralBegin, "2027-02-06T15:55:36Z")
		expectTime("annular C3", *annularContacts.CentralEnd, "2027-02-06T16:03:27Z")
	}
	if invisible := observatory.NewVisibilitySolver(nil).Solve(observatory.EventInput{Kind: event.Kind, StartsAt: event.StartsAt, Geometry: decodeGeometryForTest(t, event.Geometry)}, 31.2304, 121.4737, "Asia/Shanghai"); invisible.Status != "not_visible" {
		t.Errorf("Shanghai result=%s, want not_visible", invisible.Status)
	}
	firstEvent := parsed.events[0]
	firstGeometry := decodeGeometryForTest(t, firstEvent.Geometry)
	for _, example := range []struct {
		name                string
		latitude, longitude float64
		want                string
	}{
		{"Reykjavik", 64.15, -21.94, "observable"},
		{"Madrid", 40.42, -3.7, "limited"},
		{"Shanghai", 31.23, 121.47, "not_visible"},
	} {
		t.Run(example.name, func(t *testing.T) {
			result := observatory.NewVisibilitySolver(nil).Solve(observatory.EventInput{Kind: firstEvent.Kind, StartsAt: firstEvent.StartsAt, Geometry: firstGeometry}, example.latitude, example.longitude, "UTC")
			if result.Status != example.want {
				t.Errorf("status=%s, want %s", result.Status, example.want)
			}
			if example.name == "Reykjavik" {
				if result.EclipseContacts == nil || result.EclipseContacts.CentralBegin == nil || result.EclipseContacts.CentralEnd == nil {
					t.Fatal("one-minute totality contacts missing")
				}
				expectTime("Reykjavik C1", result.EclipseContacts.PartialBegin, "2026-08-12T16:47:07Z")
				expectTime("Reykjavik peak", result.EclipseContacts.Peak, "2026-08-12T17:48:41Z")
				expectTime("Reykjavik C2", *result.EclipseContacts.CentralBegin, "2026-08-12T17:48:11Z")
				expectTime("Reykjavik C3", *result.EclipseContacts.CentralEnd, "2026-08-12T17:49:12Z")
				expectTime("Reykjavik C4", result.EclipseContacts.PartialEnd, "2026-08-12T18:47:33Z")
			}
			if example.name == "Madrid" {
				if result.EclipseContacts == nil {
					t.Fatal("Madrid contacts missing")
				}
				expectTime("Madrid C1", result.EclipseContacts.PartialBegin, "2026-08-12T17:36:40Z")
				expectTime("Madrid peak", result.EclipseContacts.Peak, "2026-08-12T18:32:18Z")
			}
		})
	}
	sunset := observatory.NewVisibilitySolver(nil).Solve(observatory.EventInput{Kind: firstEvent.Kind, StartsAt: firstEvent.StartsAt, Geometry: firstGeometry}, 0, -20, "UTC")
	if sunset.EclipseContacts == nil || sunset.EclipseContacts.PeakVisible || sunset.WindowEnd == nil || *sunset.WindowEnd >= sunset.EclipseContacts.Peak {
		t.Errorf("sunset result=%+v", sunset)
	}
	if sunset.WindowEnd != nil {
		expectTime("sunset visible window end", *sunset.WindowEnd, "2026-08-12T19:26:14Z")
	}
}

func decodeGeometryForTest(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var geometry map[string]any
	if err := json.Unmarshal(raw, &geometry); err != nil {
		t.Fatal(err)
	}
	return geometry
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

func TestParseNASAGSFCSolarDecadeLinksAndTime(t *testing.T) {
	body := []byte(`<table><tr>
<td><a href="../SEplot/SEplot2001/SE2026Aug12T.GIF"> 2026 Aug 12 </a></td>
<td><a href="../SEanimate/SEanimate2001/SE2026Aug12T.GIF"> 17:47:05 </a></td>
<td><a href="../SEgoogle/SEgoogle2001/SE2026Aug12Tgoogle.html"> Total </a></td>
<td><a href="../SEsaros/SEsaros126.html"> 126 </a></td>
<td> 1.039</td>
<td><a href="../SEpath/SEpath2001/SE2026Aug12Tpath.html"> 02m18s </a></td>
</tr></table>`)
	parsed, err := parseNASAGSFCEclipsePage(context.Background(), body, "https://eclipse.gsfc.nasa.gov/SEdecade/SEdecade2021.html")
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.events) != 1 {
		t.Fatalf("events=%d, want 1", len(parsed.events))
	}
	event := parsed.events[0]
	if event.Kind != "solar_eclipse" || event.StartsAt.Format(time.RFC3339) != "2026-08-12T17:47:05Z" {
		t.Fatalf("event=%+v, want solar eclipse at catalog maximum", event)
	}
	var geometry map[string]any
	if err := json.Unmarshal(event.Geometry, &geometry); err != nil {
		t.Fatal(err)
	}
	if geometry["eclipseType"] != "Total" || geometry["pathUrl"] != "https://eclipse.gsfc.nasa.gov/SEpath/SEpath2001/SE2026Aug12Tpath.html" {
		t.Fatalf("geometry=%v", geometry)
	}
}

func TestParseNASAGSFCSkipsRowsWithoutMaximumTime(t *testing.T) {
	body := []byte(`<table><tr><td>2027 Aug 02</td><td>Total</td><td>1.079</td></tr></table>`)
	parsed, err := parseNASAGSFCEclipsePage(context.Background(), body, "https://eclipse.gsfc.nasa.gov/SEdecade/SEdecade2021.html")
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.events) != 0 {
		t.Fatalf("events=%d, a missing NASA maximum time must not become 12:00 UTC", len(parsed.events))
	}
}
