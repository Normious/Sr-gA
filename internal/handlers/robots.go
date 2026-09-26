package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Normious/Sr-gA/internal/auth"
	"github.com/Normious/Sr-gA/internal/db"
	"github.com/Normious/Sr-gA/internal/sitemap"
)

func (h *SitemapHandler) GenerateRobots(w http.ResponseWriter, r *http.Request) {
	project, ok := auth.ProjectFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Project not found in context")
		return
	}

	var req sitemap.RobotsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	contentHash := db.HashJSON(req)
	cacheKey := "robots:" + contentHash

	if cached, ok := h.Cache.Get(cacheKey); ok {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", `inline; filename="robots.txt"`)
		w.Header().Set("X-Sr-gA-Cache", "memory")
		w.WriteHeader(http.StatusOK)
		w.Write(cached)
		return
	}
	if cachedXML, _, err := h.DB.GetCache(r.Context(), contentHash, "robots"); err == nil {
		h.Cache.Put(cacheKey, []byte(cachedXML))
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", `inline; filename="robots.txt"`)
		w.Header().Set("X-Sr-gA-Cache", "sqlite")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cachedXML))
		return
	}

	start := time.Now()
	content, err := sitemap.GenerateRobots(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	durationMs := int(time.Since(start).Milliseconds())
	h.Cache.Put(cacheKey, content)
	_ = h.DB.PutCache(r.Context(), contentHash, "robots", string(content),
		0, len(req.Sitemaps), h.Config.SQLiteCacheTTLSeconds)

	_ = h.DB.LogHistory(r.Context(), project.ID, "robots",
		len(req.Sitemaps), 0, 0,
		len(content), 0, durationMs, "miss", "success", "", clientIP(r), r.UserAgent())

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `inline; filename="robots.txt"`)
	w.Header().Set("X-Sr-gA-Cache", "miss")
	w.Header().Set("X-Sr-gA-Duration-Ms", strconv.Itoa(durationMs))
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}
