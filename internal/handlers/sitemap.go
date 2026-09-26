package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Normious/Sr-gA/internal/auth"
	"github.com/Normious/Sr-gA/internal/cache"
	"github.com/Normious/Sr-gA/internal/config"
	"github.com/Normious/Sr-gA/internal/db"
	"github.com/Normious/Sr-gA/internal/sitemap"
)

type SitemapHandler struct {
	DB     *db.DB
	Config *config.Config
	Cache  *cache.MemoryCache
}

func (h *SitemapHandler) Generate(w http.ResponseWriter, r *http.Request) {
	project, ok := auth.ProjectFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Project not found in context")
		return
	}

	var req sitemap.SitemapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if len(req.URLs) == 0 {
		writeError(w, http.StatusBadRequest, "At least one URL is required")
		return
	}
	if len(req.URLs) > h.Config.MaxURLsPerRequest {
		writeError(w, http.StatusBadRequest, "Too many URLs")
		return
	}

	contentHash := db.HashJSON(req)
	cacheKey := "sitemap:" + contentHash

	// Layer 1: memory
	if xml, ok := h.Cache.Get(cacheKey); ok {
		h.respond(w, xml, len(req.URLs), "memory", 0, req.Compress)
		_ = h.DB.LogHistory(r.Context(), project.ID, "sitemap",
			len(req.URLs), len(req.URLs), 0, len(xml), boolToInt(req.Compress),
			0, "memory", "success", "", clientIP(r), r.UserAgent())
		return
	}

	// Layer 2: sqlite
	if cachedXML, compressed, err := h.DB.GetCache(r.Context(), contentHash, "sitemap"); err == nil {
		h.Cache.Put(cacheKey, []byte(cachedXML))
		h.respond(w, []byte(cachedXML), len(req.URLs), "sqlite", 0, compressed == 1)
		_ = h.DB.LogHistory(r.Context(), project.ID, "sitemap",
			len(req.URLs), len(req.URLs), 0, len(cachedXML), compressed,
			0, "sqlite", "success", "", clientIP(r), r.UserAgent())
		return
	}

	start := time.Now()
	xmlBytes, err := sitemap.GenerateURLSet(&req, project.DefaultChangeFreq, project.DefaultPriority)
	if err != nil {
		h.logFailure(r, project.ID, "sitemap", err.Error())
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	wasCompressed := false
	if req.Compress {
		xmlBytes, err = sitemap.Gzip(xmlBytes)
		if err != nil {
			h.logFailure(r, project.ID, "sitemap", "Gzip failed: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Compression failed")
			return
		}
		wasCompressed = true
	}

	durationMs := int(time.Since(start).Milliseconds())
	h.Cache.Put(cacheKey, xmlBytes)
	_ = h.DB.PutCache(r.Context(), contentHash, "sitemap", string(xmlBytes),
		boolToInt(wasCompressed), len(req.URLs), h.Config.SQLiteCacheTTLSeconds)

	_ = h.DB.LogHistory(r.Context(), project.ID, "sitemap",
		len(req.URLs), len(req.URLs), 0,
		len(xmlBytes), boolToInt(req.Compress),
		durationMs, "miss", "success", "", clientIP(r), r.UserAgent())

	h.respond(w, xmlBytes, len(req.URLs), "miss", durationMs, req.Compress)
}

func (h *SitemapHandler) respond(w http.ResponseWriter, xml []byte, urlCount int, cacheHit string, durationMs int, compressed bool) {
	contentType := "application/xml; charset=utf-8"
	filename := "sitemap.xml"
	if compressed {
		contentType = "application/gzip"
		filename = "sitemap.xml.gz"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "inline; filename=\""+filename+"\"")
	w.Header().Set("X-Sr-gA-URL-Count", strconv.Itoa(urlCount))
	w.Header().Set("X-Sr-gA-Cache", cacheHit)
	w.Header().Set("X-Sr-gA-Duration-Ms", strconv.Itoa(durationMs))
	w.WriteHeader(http.StatusOK)
	w.Write(xml)
}

func (h *SitemapHandler) logFailure(r *http.Request, projectID int64, operation, errMsg string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = h.DB.LogHistory(ctx, projectID, operation, 0, 0, 0, 0, 0, 0, "miss", "failed", errMsg, clientIP(r), r.UserAgent())
}

// ─── Helpers ──────────────────────────────────────────────────

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error":   msg,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
