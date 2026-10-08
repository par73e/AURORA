package httpapi

import (
	"net/http"
	"time"

	"aurora/backend/internal/dailyimage"
)

func imageWallHandler(provider dailyimage.WallProvider, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if provider == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "宇宙图像窗服务尚未配置"})
			return
		}
		wall, err := provider.Wall(r.Context(), now())
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "宇宙图像窗暂不可用，请稍后重试"})
			return
		}
		writeJSON(w, http.StatusOK, wall)
	}
}
