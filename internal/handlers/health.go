package handlers

import (
	"net/http"
	"time"
)

func (h *SitemapHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"service":   "Sr-gA — Sitemap/robots.txt Service",
		"version":   "1.0.0",
		"language":  "Go",
		"status":    "ok",
		"cache":     h.Cache.Stats(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
