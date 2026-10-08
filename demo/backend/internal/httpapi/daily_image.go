package httpapi

import (
	"errors"
	"net/http"
	"time"

	"aurora/backend/internal/dailyimage"
)

func dailyImageHandler(provider dailyimage.Provider, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if provider == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "每日一图服务尚未配置"})
			return
		}
		date, err := parseDailyImageDate(r.URL.Query().Get("date"), now())
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "date 必须是 YYYY-MM-DD，且不能晚于今天"})
			return
		}
		image, err := provider.Daily(r.Context(), date)
		if errors.Is(err, dailyimage.ErrNotConfigured) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "每日一图尚未配置 NASA API Key"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "NASA 每日一图暂不可用，请稍后重试"})
			return
		}
		writeJSON(w, http.StatusOK, image)
	}
}

func parseDailyImageDate(raw string, current time.Time) (time.Time, error) {
	current = current.UTC()
	if raw == "" {
		return current, nil
	}
	date, err := time.Parse(time.DateOnly, raw)
	if err != nil || date.After(time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, time.UTC)) {
		return time.Time{}, errors.New("invalid APOD date")
	}
	return date, nil
}
