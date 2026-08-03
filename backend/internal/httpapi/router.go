package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"aurora/backend/internal/moon"
	"aurora/backend/internal/orbit"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Router(repository *orbit.Repository, moonRepository *moon.Repository) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	router.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	router.Get("/api/v1/orbit/overview", func(w http.ResponseWriter, r *http.Request) {
		overview, err := repository.Overview(r.Context())
		if err != nil {
			slog.Error("load ORBIT overview", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "暂时无法读取 ORBIT 数据"})
			return
		}
		writeJSON(w, http.StatusOK, overview)
	})
	router.Get("/api/v1/moon/spacecraft", func(w http.ResponseWriter, r *http.Request) {
		items, err := moonRepository.ListSpacecraft(r.Context())
		if err != nil {
			slog.Error("load moon spacecraft", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "暂时无法读取月球飞行器数据"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"spacecraft": items})
	})
	router.Get("/api/v1/moon/landing-sites", func(w http.ResponseWriter, r *http.Request) {
		items, err := moonRepository.ListLandingSites(r.Context())
		if err != nil {
			slog.Error("load moon landing sites", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "暂时无法读取月球着陆点数据"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"landingSites": items})
	})
	return router
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("write JSON response", "error", err)
	}
}
