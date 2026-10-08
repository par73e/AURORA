package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"

	observerlocation "aurora/backend/internal/location"
)

type locationSearchProvider interface {
	Search(context.Context, string) ([]observerlocation.Candidate, error)
}

func reverseLocationHandler(geocoder observerlocation.ReverseGeocoder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		latitude, err := coordinate(r.URL.Query().Get("latitude"), -90, 90)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "latitude 必须是 -90 到 90 之间的数字"})
			return
		}
		longitude, err := coordinate(r.URL.Query().Get("longitude"), -180, 180)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "longitude 必须是 -180 到 180 之间的数字"})
			return
		}
		if geocoder == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "定位服务尚未配置"})
			return
		}

		place, err := geocoder.Reverse(r.Context(), latitude, longitude)
		if err != nil {
			if errors.Is(err, observerlocation.ErrNotConfigured) {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "定位服务尚未配置"})
				return
			}
			slog.Warn("reverse observer location", "error", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "暂时无法确认城市区县"})
			return
		}
		writeJSON(w, http.StatusOK, place)
	}
}

func searchLocationHandler(geocoder observerlocation.ReverseGeocoder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if len([]rune(query)) < 2 || len([]rune(query)) > 80 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "地点名称需要 2 到 80 个字符"})
			return
		}
		provider, ok := geocoder.(locationSearchProvider)
		if !ok || provider == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "地点搜索尚未配置"})
			return
		}
		places, err := provider.Search(r.Context(), query)
		if err != nil {
			if errors.Is(err, observerlocation.ErrNotConfigured) {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "地点搜索尚未配置"})
				return
			}
			slog.Warn("search observer location", "error", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "暂时无法搜索地点"})
			return
		}
		if places == nil {
			places = []observerlocation.Candidate{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"places": places})
	}
}

func coordinate(raw string, minimum, maximum float64) (float64, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < minimum || value > maximum {
		return 0, strconv.ErrSyntax
	}
	return value, nil
}
