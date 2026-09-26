package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Normious/Sr-gA/internal/auth"
	"github.com/Normious/Sr-gA/internal/sitemap"
)

func (h *SitemapHandler) Validate(w http.ResponseWriter, r *http.Request) {
	project, ok := auth.ProjectFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Project not found in context")
		return
	}

	var req sitemap.ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if len(req.URLs) == 0 {
		writeError(w, http.StatusBadRequest, "At least one URL required")
		return
	}
	if len(req.URLs) > h.Config.MaxURLsForValidation {
		writeError(w, http.StatusBadRequest, "Too many URLs to validate")
		return
	}

	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = h.Config.MaxBatchValidationConcurrency
	}

	timeoutSecs := req.TimeoutSecs
	if timeoutSecs <= 0 {
		timeoutSecs = h.Config.DefaultValidationTimeoutSecs
	}

	validator := sitemap.NewValidator(h.Config.ValidationUserAgent, h.Config.ValidationMaxRedirects, timeoutSecs)

	start := time.Now()
	results := validator.ValidateAll(r.Context(), req.URLs, concurrency)
	durationMs := int(time.Since(start).Milliseconds())

	validCount := 0
	for _, res := range results {
		if res.IsValid {
			validCount++
		}
	}
	invalidCount := len(results) - validCount

	_ = h.DB.LogHistory(r.Context(), project.ID, "validate",
		len(req.URLs), validCount, invalidCount,
		0, 0, durationMs, "miss", "success", "", clientIP(r), r.UserAgent())

	writeJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"total":         len(results),
		"valid_count":   validCount,
		"invalid_count": invalidCount,
		"duration_ms":   durationMs,
		"results":       results,
		"project":       project.Name,
	})
}

func (h *SitemapHandler) Discover(w http.ResponseWriter, r *http.Request) {
	project, ok := auth.ProjectFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Project not found in context")
		return
	}

	var req sitemap.DiscoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if req.SitemapURL == "" {
		writeError(w, http.StatusBadRequest, "sitemap_url is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	start := time.Now()
	urls, err := sitemap.Discover(ctx, req.SitemapURL, req.IncludeAlt, req.MaxDepth,
		h.Config.ValidationUserAgent, h.Config.DefaultValidationTimeoutSecs)
	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		h.logFailure(r, project.ID, "discover", err.Error())
		writeError(w, http.StatusBadGateway, "Discovery failed: "+err.Error())
		return
	}

	_ = h.DB.LogHistory(r.Context(), project.ID, "discover",
		len(urls), len(urls), 0, 0, 0, durationMs, "miss", "success", "", clientIP(r), r.UserAgent())

	writeJSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"sitemap_url": req.SitemapURL,
		"url_count":   len(urls),
		"duration_ms": durationMs,
		"urls":        urls,
		"project":     project.Name,
	})
}
