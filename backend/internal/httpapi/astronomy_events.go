package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"aurora/backend/internal/astronomyevent"
	"aurora/backend/internal/observatory"
)

const astronomyEventsMaximumRange = 548 * 24 * time.Hour

type astronomyEventResponse struct {
	astronomyevent.Event
	Global map[string]any        `json:"global"`
	Local  *eventLocalVisibility `json:"local,omitempty"`
	Source astronomyEventSource  `json:"source"`
}

// astronomyEventSource 保留文档中的稳定 source 对象；扁平字段继续保留以兼容当前前端。
type astronomyEventSource struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	VerifiedAt string `json:"verifiedAt"`
}

type eventLocalVisibility struct {
	Status          string                       `json:"status"`
	BestAt          *string                      `json:"bestAt,omitempty"`
	WindowStart     *string                      `json:"windowStart,omitempty"`
	WindowEnd       *string                      `json:"windowEnd,omitempty"`
	AzimuthDegrees  *float64                     `json:"azimuthDegrees,omitempty"`
	AltitudeDegrees *float64                     `json:"altitudeDegrees,omitempty"`
	Reason          string                       `json:"reason"`
	EclipseContacts *observatory.EclipseContacts `json:"eclipseContacts,omitempty"`
}

func astronomyEventsHandler(store astronomyevent.Store, solver *observatory.VisibilitySolver, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "天象事件库尚未配置"})
			return
		}
		from, to, err := astronomyEventRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"), now())
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "from 和 to 必须是 YYYY-MM-DD，查询范围不能超过 18 个月"})
			return
		}
		latitude, longitude, timezone, hasLocation, err := astronomyEventLocation(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "latitude、longitude 必须同时提供且范围有效；timezone 必须为 IANA 时区"})
			return
		}
		elevation := 0.0
		if raw := r.URL.Query().Get("elevation"); raw != "" {
			elevation, err = strconv.ParseFloat(raw, 64)
			if err != nil || math.IsNaN(elevation) || math.IsInf(elevation, 0) || elevation < -500 || elevation > 10000 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "elevation 必须是 -500 至 10000 米之间的数字"})
				return
			}
		}
		events, err := store.List(r.Context(), astronomyevent.ListQuery{From: from, To: to})
		if err != nil {
			slog.Error("load astronomy events", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "暂时无法读取天象事件"})
			return
		}
		events = deduplicateAstronomyEvents(events)
		sourceStatuses := []astronomyevent.SourceStatus{}
		if statusStore, ok := store.(astronomyevent.SourceStatusStore); ok {
			statuses, statusErr := statusStore.ListSourceStatuses(r.Context())
			if statusErr != nil {
				slog.Warn("load astronomy source statuses", "error", statusErr)
			} else {
				sourceStatuses = statuses
			}
		}
		response := make([]astronomyEventResponse, 0, len(events))
		for _, event := range events {
			event.SourceURL = astronomyevent.ReadableSourceURL(event.SourceCode, event.SourceURL)
			global := decodeEventGeometry(event.Geometry)
			global["description"] = event.Summary
			item := astronomyEventResponse{
				Event:  event,
				Global: global,
				Source: astronomyEventSource{Kind: event.Origin, Name: event.SourceName, URL: event.SourceURL, VerifiedAt: event.VerifiedAt},
			}
			if hasLocation && solver != nil {
				item.Local = resolveEventLocalVisibility(event, latitude, longitude, timezone, solver, elevation)
			}
			response = append(response, item)
		}
		if hasLocation {
			sort.SliceStable(response, func(i, j int) bool {
				return astronomyVisibilityRank(response[i].Local) < astronomyVisibilityRank(response[j].Local)
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"events": response,
			"range": map[string]string{
				"from": from.Format(time.DateOnly),
				"to":   to.Add(-time.Nanosecond).Format(time.DateOnly),
			},
			"locationVisibility": map[bool]string{true: "partial", false: "location_required"}[hasLocation],
			"sources":            sourceStatuses,
		})
	}
}

// deduplicateAstronomyEvents resolves the migration overlap between the old
// curated 2026 list and current computed/external feeds. The canonical record
// keeps the richer geometry needed by local visibility, while curated display
// copy is retained when the canonical source has no presentation document.
func deduplicateAstronomyEvents(events []astronomyevent.Event) []astronomyevent.Event {
	result := make([]astronomyevent.Event, 0, len(events))
	for _, event := range events {
		key := astronomyEventSemanticKey(event)
		if key == "" {
			result = append(result, event)
			continue
		}
		index := -1
		for candidateIndex := len(result) - 1; candidateIndex >= 0; candidateIndex-- {
			candidate := result[candidateIndex]
			if astronomyEventSemanticKey(candidate) != key {
				continue
			}
			difference := event.StartsAt.Sub(candidate.StartsAt)
			if difference < 0 {
				difference = -difference
			}
			if difference <= astronomyEventDuplicateTolerance(event.Kind) {
				index = candidateIndex
				break
			}
		}
		if index < 0 {
			result = append(result, event)
			continue
		}
		current := result[index]
		if astronomyEventQuality(event) > astronomyEventQuality(current) {
			event.Presentation = preferredEventPresentation(event.Presentation, current.Presentation)
			result[index] = event
		} else {
			current.Presentation = preferredEventPresentation(current.Presentation, event.Presentation)
			result[index] = current
		}
	}
	return result
}

func astronomyEventSemanticKey(event astronomyevent.Event) string {
	switch event.Kind {
	case "solar_eclipse", "lunar_eclipse", "new_moon", "first_quarter", "full_moon", "last_quarter",
		"march_equinox", "june_solstice", "september_equinox", "december_solstice":
		return event.Kind
	case "meteor_shower", "planetary_elongation", "planetary_opposition", "planetary_conjunction", "moon_conjunction":
		subject := astronomyEventSubject(event)
		if subject != "" {
			return event.Kind + "|" + subject
		}
	}
	return ""
}

func astronomyEventDuplicateTolerance(kind string) time.Duration {
	switch kind {
	case "planetary_elongation", "planetary_opposition", "planetary_conjunction", "moon_conjunction":
		return 48 * time.Hour
	case "meteor_shower", "solar_eclipse", "lunar_eclipse":
		return 36 * time.Hour
	default:
		return 12 * time.Hour
	}
}

func astronomyEventSubject(event astronomyevent.Event) string {
	geometry := decodeEventGeometry(event.Geometry)
	if slug, ok := geometry["slug"].(string); ok && slug != "" {
		return strings.ToLower(slug)
	}
	if object, ok := geometry["object"].(string); ok && object != "" {
		return strings.ToLower(object)
	}
	if objects, ok := geometry["objects"].([]any); ok {
		names := make([]string, 0, len(objects))
		for _, value := range objects {
			if name, ok := value.(string); ok {
				names = append(names, strings.ToLower(name))
			}
		}
		sort.Strings(names)
		if len(names) > 0 {
			return strings.Join(names, "+")
		}
	}
	text := strings.ToLower(event.ID + " " + event.Title + " " + event.TitleEN)
	aliases := []struct {
		canonical string
		values    []string
	}{
		{"perseids", []string{"perseids", "英仙座"}},
		{"kappa-cygnids", []string{"kappa-cygnids", "kappa cygnids", "天鹅座 κ", "天鹅座κ"}},
		{"aurigids", []string{"aurigids", "御夫座"}},
		{"september-epsilon-perseids", []string{"september-epsilon-perseids", "september epsilon perseids", "九月 ε 英仙座", "九月ε英仙座"}},
		{"mercury", []string{"mercury", "水星"}},
		{"venus", []string{"venus", "金星"}},
		{"mars", []string{"mars", "火星"}},
		{"jupiter", []string{"jupiter", "木星"}},
		{"saturn", []string{"saturn", "土星"}},
		{"uranus", []string{"uranus", "天王星"}},
		{"neptune", []string{"neptune", "海王星"}},
	}
	for _, alias := range aliases {
		for _, value := range alias.values {
			if strings.Contains(text, value) {
				return alias.canonical
			}
		}
	}
	return ""
}

func astronomyEventQuality(event astronomyevent.Event) int {
	score := 0
	if event.Origin != "curated" {
		score += 10
	}
	geometry := decodeEventGeometry(event.Geometry)
	if event.Kind == "solar_eclipse" {
		if _, ok := geometry["besselian"]; ok {
			score += 30
		}
	}
	if len(geometry) > 0 {
		score += 5
	}
	if _, ok := geometry["positions"]; ok {
		score += 20
	}
	if event.EndsAt != nil {
		score++
	}
	return score
}

func preferredEventPresentation(primary, fallback json.RawMessage) json.RawMessage {
	if len(decodeEventGeometry(primary)) > 0 {
		return primary
	}
	return fallback
}

func astronomyVisibilityRank(local *eventLocalVisibility) int {
	if local == nil {
		return 5
	}
	switch local.Status {
	case "observable":
		return 0
	case "limited":
		return 1
	case "not_calculated":
		return 2
	case "non_visual":
		return 3
	case "not_visible":
		return 4
	default:
		return 5
	}
}

func astronomyEventLocation(request *http.Request) (latitude, longitude float64, timezone string, hasLocation bool, err error) {
	query := request.URL.Query()
	rawLatitude, rawLongitude := query.Get("latitude"), query.Get("longitude")
	if rawLatitude == "" && rawLongitude == "" {
		return 0, 0, "", false, nil
	}
	if rawLatitude == "" || rawLongitude == "" {
		return 0, 0, "", false, errors.New("incomplete coordinates")
	}
	latitude, err = coordinate(rawLatitude, -90, 90)
	if err != nil {
		return 0, 0, "", false, err
	}
	longitude, err = coordinate(rawLongitude, -180, 180)
	if err != nil {
		return 0, 0, "", false, err
	}
	timezone = query.Get("timezone")
	if timezone == "" {
		timezone = "UTC"
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return 0, 0, "", false, err
	}
	return latitude, longitude, timezone, true, nil
}

// resolveEventLocalVisibility 把全球事件换算为本地可见性。
// 尚未接入专用求解器的类别返回 not_calculated，绝不把全球事件误标为本地可见。
func resolveEventLocalVisibility(event astronomyevent.Event, latitude, longitude float64, timezone string, solver *observatory.VisibilitySolver, elevation ...float64) *eventLocalVisibility {
	geometry := decodeEventGeometry(event.Geometry)
	input := observatory.EventInput{
		ID:       event.ID,
		Kind:     event.Kind,
		StartsAt: event.StartsAt,
		EndsAt:   event.EndsAt,
		Geometry: geometry,
	}
	if len(elevation) > 0 {
		input.Elevation = elevation[0]
	}
	vis := solver.Solve(input, latitude, longitude, timezone)
	return &eventLocalVisibility{
		Status:          vis.Status,
		BestAt:          vis.BestAt,
		WindowStart:     vis.WindowStart,
		WindowEnd:       vis.WindowEnd,
		AzimuthDegrees:  vis.AzimuthDegrees,
		AltitudeDegrees: vis.AltitudeDegrees,
		Reason:          vis.Reason,
		EclipseContacts: vis.EclipseContacts,
	}
}

// decodeEventGeometry 把事件 geometry JSON 解为 map，供求解器读取 object/objects。
func decodeEventGeometry(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var geometry map[string]any
	if err := json.Unmarshal(raw, &geometry); err != nil {
		return map[string]any{}
	}
	return geometry
}

func astronomyEventRange(rawFrom, rawTo string, current time.Time) (time.Time, time.Time, error) {
	defaultFrom := current.UTC()
	from, err := parseEventDate(rawFrom, defaultFrom)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to, err := parseEventDate(rawTo, from.Add(30*24*time.Hour))
	if err != nil || !to.After(from) || to.Sub(from) > astronomyEventsMaximumRange {
		return time.Time{}, time.Time{}, errors.New("invalid astronomy event range")
	}
	return from, to, nil
}

func parseEventDate(raw string, fallback time.Time) (time.Time, error) {
	if raw == "" {
		return fallback, nil
	}
	return time.Parse(time.DateOnly, raw)
}
