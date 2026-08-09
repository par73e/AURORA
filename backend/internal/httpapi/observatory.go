package httpapi

import (
	"net/http"
	"strconv"

	"aurora/backend/internal/observatory"
)

func observingConditionsHandler(provider observatory.ConditionsProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		latitude, err := strconv.ParseFloat(r.URL.Query().Get("latitude"), 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "latitude 必须是有效纬度"})
			return
		}
		longitude, err := strconv.ParseFloat(r.URL.Query().Get("longitude"), 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "longitude 必须是有效经度"})
			return
		}
		conditions, err := provider.Conditions(r.Context(), latitude, longitude)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "暂时无法读取观测条件预报"})
			return
		}
		writeJSON(w, http.StatusOK, conditions)
	}
}
