package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"aurora/backend/internal/mars"
	"aurora/backend/internal/moon"
	"aurora/backend/internal/orbit"
	"aurora/backend/internal/voyage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Router(repository *orbit.Repository, moonRepository *moon.Repository, marsRepository *mars.Repository, voyageRepository *voyage.Repository) http.Handler {
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
	router.Get("/api/v1/orbit/spacecraft", func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		page, err := positiveInt(params.Get("page"), 1)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "page 必须是正整数"})
			return
		}
		pageSize, err := positiveInt(params.Get("pageSize"), 20)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pageSize 必须是正整数"})
			return
		}
		mode := params.Get("mode")
		if mode != "" && mode != "keyword" && mode != "regex" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mode 仅支持 keyword 或 regex"})
			return
		}
		result, err := repository.SearchSpacecraft(r.Context(), orbit.SpacecraftQuery{
			Query: params.Get("q"), Operator: params.Get("operator"), Sort: params.Get("sort"),
			Regex: mode == "regex", Page: page, PageSize: pageSize,
		})
		if err != nil {
			slog.Warn("search ORBIT spacecraft", "error", err)
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "查询条件无效或暂时无法读取航天器目录"})
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	router.Get("/api/v1/moon/spacecraft", func(w http.ResponseWriter, r *http.Request) {
		items, err := moonRepository.ListSpacecraft(r.Context())
		if err != nil {
			slog.Error("load moon spacecraft", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "暂时无法读取月球飞行器数据"})
			return
		}
		syncedAt, syncErr := moonRepository.LastSyncTime(r.Context())
		if syncErr != nil {
			slog.Error("load moon sync time", "error", syncErr)
		}
		writeJSON(w, http.StatusOK, map[string]any{"spacecraft": items, "syncedAt": syncedAt})
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
	router.Get("/api/v1/mars/spacecraft", func(w http.ResponseWriter, r *http.Request) {
		items, err := marsRepository.ListSpacecraft(r.Context())
		if err != nil {
			slog.Error("load mars spacecraft", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "暂时无法读取火星飞行器数据"})
			return
		}
		syncedAt, syncErr := marsRepository.LastSyncTime(r.Context())
		if syncErr != nil {
			slog.Error("load mars sync time", "error", syncErr)
		}
		writeJSON(w, http.StatusOK, map[string]any{"spacecraft": items, "syncedAt": syncedAt})
	})
	router.Get("/api/v1/mars/landing-sites", func(w http.ResponseWriter, r *http.Request) {
		items, err := marsRepository.ListLandingSites(r.Context())
		if err != nil {
			slog.Error("load mars landing sites", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "暂时无法读取火星着陆点数据"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"landingSites": items})
	})
	router.Get("/api/v1/voyage/probes", func(w http.ResponseWriter, r *http.Request) {
		items, err := voyageRepository.ListProbes(r.Context())
		if err != nil {
			slog.Error("load deep space probes", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "暂时无法读取深空探测器数据"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"probes": items})
	})
	return router
}

func positiveInt(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if value < 1 {
		return 0, strconv.ErrSyntax
	}
	return value, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("write JSON response", "error", err)
	}
}
