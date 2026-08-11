package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"aurora/backend/internal/observatory"
)

func observingConditionsHandler(provider observatory.ConditionsProvider, moons observatory.MoonProvider, lights observatory.LightPollutionProvider) http.HandlerFunc {
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
		// 可选 time 参数：返回该时刻（Unix 秒）最近的逐小时预报快照，供按时间查询天气。
		if raw := r.URL.Query().Get("time"); raw != "" {
			seconds, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || seconds <= 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "time 必须是正 Unix 秒"})
				return
			}
			hour, _, within := observatory.NearestHour(conditions.Hourly, conditions.Timezone, time.Unix(seconds, 0))
			conditions.Selected = &observatory.SelectedObservation{Hour: hour, WithinForecastWindow: within}
		}
		// 可选 scores=1 参数：追加逐小时观测评分（单请求，供动态推荐）。
		if r.URL.Query().Get("scores") == "1" {
			conditions.Scores = observatory.ScoreSeries(conditions, moons, lights, latitude, longitude)
		}
		writeJSON(w, http.StatusOK, conditions)
	}
}
