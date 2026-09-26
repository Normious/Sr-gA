package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Normious/Sr-gA/internal/auth"
	"github.com/Normious/Sr-gA/internal/db"
	"github.com/Normious/Sr-gA/internal/sitemap"
)

func (h *SitemapHandler) GenerateIndex(w http.ResponseWriter, r *http.Request) {
	project, ok := auth.ProjectFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Project not found in context")
		return
	}

	var req sitemap.SitemapIndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	contentHash := db.HashJSON(req)
	cacheKey := "sitemap_index:" + contentHash

	if xml, ok := h.Cache.Get(cacheKey); ok {
		h.respond(w, xml, len(req.Sitemaps), "memory", 0, req.Compress)
		return
	}
	if cachedXML, compressed, err := h.DB.GetCache(r.Context(), contentHash, "sitemap_index"); err == nil {
		h.Cache.Put(cacheKey, []byte(cachedXML))
		h.respond(w, []byte(cachedXML), len(req.Sitemaps), "sqlite", 0, compressed == 1)
		return
	}

	start := time.Now()
	xmlBytes, err := sitemap.GenerateSitemapIndex(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	wasCompressed := false
	if req.Compress {
		xmlBytes, err = sitemap.Gzip(xmlBytes)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Compression failed")
			return
		}
		wasCompressed = true
	}

	durationMs := int(time.Since(start).Milliseconds())
	h.Cache.Put(cacheKey, xmlBytes)
	_ = h.DB.PutCache(r.Context(), contentHash, "sitemap_index", string(xmlBytes),
		boolToInt(wasCompressed), len(req.Sitemaps), h.Config.SQLiteCacheTTLSeconds)

	_ = h.DB.LogHistory(r.Context(), project.ID, "sitemap_index",
		len(req.Sitemaps), len(req.Sitemaps), 0,
		len(xmlBytes), boolToInt(req.Compress),
		durationMs, "miss", "success", "", clientIP(r), r.UserAgent())

	h.respond(w, xmlBytes, len(req.Sitemaps), "miss", durationMs, req.Compress)
}
