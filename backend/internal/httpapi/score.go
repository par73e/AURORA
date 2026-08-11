package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"aurora/backend/internal/observatory"
)

// observingScoreHandler 提供观测评分：GET /api/v1/astronomy/score?latitude&longitude&at
//   - latitude / longitude：必填；
//   - at：可选，Unix 秒；缺省为当前时刻。
//
// 评分由动态天气（分层云、能见度、降水、气溶胶）+ 月光组成；光污染作为地点长期环境数据单独返回。
func observingScoreHandler(conditions observatory.ConditionsProvider, moons observatory.MoonProvider, lights observatory.LightPollutionProvider) http.HandlerFunc {
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
		at := time.Now()
		if raw := r.URL.Query().Get("at"); raw != "" {
			seconds, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || seconds <= 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "at 必须是正 Unix 秒"})
				return
			}
			at = time.Unix(seconds, 0)
		}

		score, err := observatory.ScoreObserving(r.Context(), conditions, moons, lights, latitude, longitude, at)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "暂时无法读取观测评分"})
			return
		}
		writeJSON(w, http.StatusOK, score)
	}
}
