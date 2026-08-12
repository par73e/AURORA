package astronomyevent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const horizonsEventsEndpoint = "https://ssd.jpl.nasa.gov/api/horizons.api"

var horizonPlanetCommands = map[string]string{
	"sun": "10", "mercury": "199", "venus": "299", "mars": "499",
	"jupiter": "599", "saturn": "699", "uranus": "799", "neptune": "899",
}

// FetchGeocentricEclipticSamples 取得地心黄道位置矢量；以 AU 为单位，日采样。
// 每颗行星使用一个顺序请求，符合 Horizons 的单请求使用约定。
func FetchGeocentricEclipticSamples(ctx context.Context, client *http.Client, body string, from, to time.Time) ([]EphemerisSample, error) {
	baseBody := strings.TrimSuffix(body, "_heliocentric")
	command, ok := horizonPlanetCommands[baseBody]
	if !ok {
		return nil, fmt.Errorf("unsupported Horizons body %q", body)
	}
	query := url.Values{}
	query.Set("format", "text")
	query.Set("COMMAND", command)
	query.Set("EPHEM_TYPE", "VECTORS")
	center := "500@399"
	if strings.HasSuffix(body, "_heliocentric") {
		center = "500@10"
	}
	query.Set("CENTER", center)
	query.Set("REF_PLANE", "ECLIPTIC")
	query.Set("OUT_UNITS", "AU-D")
	query.Set("VEC_TABLE", "1")
	query.Set("CSV_FORMAT", "YES")
	query.Set("MAKE_EPHEM", "YES")
	query.Set("START_TIME", from.UTC().Format("2006-01-02T15:04:05"))
	query.Set("STOP_TIME", to.UTC().Format("2006-01-02T15:04:05"))
	query.Set("STEP_SIZE", "1d")
	endpoint := horizonsEventsEndpoint + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "AURORA astronomy event cache/1.0")
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request Horizons %s: %w", body, err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read Horizons %s: %w", body, err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Horizons %s status %d", body, response.StatusCode)
	}
	return parseHorizonsVectorSamples(body, endpoint, string(raw))
}

func parseHorizonsVectorSamples(body, sourceURL, text string) ([]EphemerisSample, error) {
	start, end := strings.Index(text, "$$SOE"), strings.Index(text, "$$EOE")
	if start < 0 || end < 0 || end <= start {
		return nil, fmt.Errorf("Horizons %s response contains no SOE section", body)
	}
	var samples []EphemerisSample
	for _, line := range strings.Split(text[start+5:end], "\n") {
		fields := strings.Split(strings.TrimSpace(line), ",")
		if len(fields) < 5 {
			continue
		}
		epoch, err := time.Parse("A.D. 2006-Jan-02 15:04:05.0000", strings.TrimSpace(fields[1]))
		if err != nil {
			return nil, fmt.Errorf("parse Horizons %s epoch: %w", body, err)
		}
		var x, y, z float64
		if _, err := fmt.Sscanf(strings.TrimSpace(fields[2]), "%f", &x); err != nil {
			return nil, err
		}
		if _, err := fmt.Sscanf(strings.TrimSpace(fields[3]), "%f", &y); err != nil {
			return nil, err
		}
		if _, err := fmt.Sscanf(strings.TrimSpace(fields[4]), "%f", &z); err != nil {
			return nil, err
		}
		samples = append(samples, EphemerisSample{Body: body, Epoch: epoch.UTC(), XAU: x, YAU: y, ZAU: z, SourceURL: sourceURL})
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("Horizons %s returned no vector rows", body)
	}
	return samples, nil
}

type EphemerisSyncer struct {
	store  EphemerisStore
	client *http.Client
	now    func() time.Time
}

func NewEphemerisSyncer(store EphemerisStore) *EphemerisSyncer {
	return &EphemerisSyncer{store: store, client: &http.Client{Timeout: 30 * time.Second}, now: time.Now}
}

func (s *EphemerisSyncer) SyncPlanetaryPositions(ctx context.Context) (syncErr error) {
	if s == nil || s.store == nil {
		return fmt.Errorf("planetary ephemeris store 未配置")
	}
	runID, err := s.store.StartSourceSync(ctx, "jpl_horizons_events")
	if err != nil {
		return err
	}
	records := 0
	defer func() { _ = s.store.FinishSourceSync(context.Background(), runID, records, syncErr) }()
	from := time.Date(s.now().UTC().Year(), s.now().UTC().Month(), s.now().UTC().Day(), 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 18, 0)
	byBody := make(map[string][]EphemerisSample)
	for _, body := range []string{"sun", "mercury", "venus", "mars", "jupiter", "saturn", "uranus", "neptune"} {
		samples, err := FetchGeocentricEclipticSamples(ctx, s.client, body, from, to)
		if err != nil {
			return err
		}
		if err := s.store.ReplaceEphemerisSamples(ctx, body, from, to, samples); err != nil {
			return err
		}
		byBody[body] = samples
		records += len(samples)
	}
	for _, body := range []string{"mercury", "venus", "mars", "jupiter", "saturn", "uranus", "neptune"} {
		heliocentricBody := body + "_heliocentric"
		samples, err := FetchGeocentricEclipticSamples(ctx, s.client, heliocentricBody, from, to)
		if err != nil {
			return err
		}
		if err := s.store.ReplaceEphemerisSamples(ctx, heliocentricBody, from, to, samples); err != nil {
			return err
		}
		byBody[heliocentricBody] = samples
		records += len(samples)
	}
	events := append(coreEvents(from, to, s.now()), planetaryEvents(byBody, s.now())...)
	events = append(events, moonConjunctionEvents(byBody, s.now())...)
	events = append(events, orbitalDistanceEvents(byBody, s.now())...)
	events = append(events, multiPlanetAlignmentEvents(byBody, s.now())...)
	if err := s.store.ReplaceComputed(ctx, ListQuery{From: from, To: to}, events); err != nil {
		return err
	}
	records += len(events)
	return nil
}
