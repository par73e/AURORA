package httpapi

import (
	"errors"
	"net/http"

	"aurora/backend/internal/observatory"
)

// lightPollutionHandler 提供光污染数据：GET /api/v1/astronomy/light-pollution?latitude&longitude
// 未配置数据源（缺 Key）时返回 503，评分同步降级为不含光污染因子。
func lightPollutionHandler(lights observatory.LightPollutionProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if lights == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "光污染数据源尚未配置"})
			return
		}
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
		light, err := lights.Light(r.Context(), latitude, longitude)
		if err != nil {
			if errors.Is(err, observatory.ErrNotConfigured) {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "光污染数据源尚未配置"})
				return
			}
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "暂时无法读取光污染数据"})
			return
		}
		writeJSON(w, http.StatusOK, light)
	}
}
