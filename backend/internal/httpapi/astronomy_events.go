package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"aurora/backend/internal/astronomyevent"
	"aurora/backend/internal/observatory"
)

const astronomyEventsMaximumRange = 548 * 24 * time.Hour

type astronomyEventResponse struct {
	astronomyevent.Event
	Local *eventLocalVisibility `json:"local,omitempty"`
}

type eventLocalVisibility struct {
	Status          string   `json:"status"`
	BestAt          *string  `json:"bestAt,omitempty"`
	WindowStart     *string  `json:"windowStart,omitempty"`
	WindowEnd       *string  `json:"windowEnd,omitempty"`
	AzimuthDegrees  *float64 `json:"azimuthDegrees,omitempty"`
	AltitudeDegrees *float64 `json:"altitudeDegrees,omitempty"`
	Reason          string   `json:"reason"`
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
		events, err := store.List(r.Context(), astronomyevent.ListQuery{From: from, To: to})
		if err != nil {
			slog.Error("load astronomy events", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "暂时无法读取天象事件"})
			return
		}
		response := make([]astronomyEventResponse, 0, len(events))
		for _, event := range events {
			item := astronomyEventResponse{Event: event}
			if hasLocation && solver != nil {
				item.Local = resolveEventLocalVisibility(event, latitude, longitude, timezone, solver)
			}
			response = append(response, item)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"events": response,
			"range": map[string]string{
				"from": from.Format(time.DateOnly),
				"to":   to.Add(-time.Nanosecond).Format(time.DateOnly),
			},
			"locationVisibility": map[bool]string{true: "partial", false: "location_required"}[hasLocation],
		})
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
func resolveEventLocalVisibility(event astronomyevent.Event, latitude, longitude float64, timezone string, solver *observatory.VisibilitySolver) *eventLocalVisibility {
	geometry := decodeEventGeometry(event.Geometry)
	input := observatory.EventInput{
		Kind:     event.Kind,
		StartsAt: event.StartsAt,
		Geometry: geometry,
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
	defaultFrom := time.Date(current.UTC().Year(), current.UTC().Month(), current.UTC().Day(), 0, 0, 0, 0, time.UTC)
	from, err := parseEventDate(rawFrom, defaultFrom)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to, err := parseEventDate(rawTo, from.AddDate(0, 1, 0))
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
