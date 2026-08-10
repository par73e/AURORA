package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"aurora/backend/internal/observatory"
)

// moonHandler 提供月相每日数据：GET /api/v1/astronomy/moon?latitude&longitude&elevation&at&timezone
//   - latitude / longitude：必填，经纬度范围校验；
//   - elevation：可选，观测点海拔（米），参与 topocentric 视差修正；缺省 0；
//   - at：可选，Unix 秒；缺省为当前时刻；
//   - timezone：可选，IANA 时区名；缺省为 UTC。
func moonHandler(moons observatory.MoonProvider) http.HandlerFunc {
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
		elevation := 0.0
		if raw := r.URL.Query().Get("elevation"); raw != "" {
			elevation, err = coordinate(raw, -500, 9000)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "elevation 必须是 -500 到 9000 之间的数字（米）"})
				return
			}
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

		day, err := moons.Day(latitude, longitude, elevation, at, r.URL.Query().Get("timezone"))
		if err != nil {
			if errors.Is(err, observatory.ErrInvalidTimezone) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "timezone 必须是有效 IANA 时区名"})
				return
			}
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "暂时无法计算月相"})
			return
		}
		writeJSON(w, http.StatusOK, day)
	}
}
