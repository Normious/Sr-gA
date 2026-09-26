package handlers

import (
	"net/http"
	"strconv"

	"github.com/Normious/Sr-gA/internal/auth"
)

func (h *SitemapHandler) History(w http.ResponseWriter, r *http.Request) {
	project, ok := auth.ProjectFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Project not found in context")
		return
	}

	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	entries, total, err := h.DB.ListHistory(r.Context(), project.ID,
		q.Get("operation"), q.Get("status"), q.Get("search"), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch history")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"entries": entries,
		"pagination": map[string]any{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (h *SitemapHandler) Stats(w http.ResponseWriter, r *http.Request) {
	project, ok := auth.ProjectFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Project not found in context")
		return
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 365 {
		days = 30
	}

	stats, err := h.DB.GetStats(r.Context(), project.ID, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch stats")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":      true,
		"days":         stats.Days,
		"daily":        stats.Daily,
		"totals":       stats.Totals,
		"by_operation": stats.ByOperation,
		"project":      project.Name,
	})
}
