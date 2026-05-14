package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mohitsharma-in/socialpublish/internal/store"
	"github.com/mohitsharma-in/socialpublish/internal/tenant"
)

// Analytics handles analytics routes.
type Analytics struct {
	analytics store.AnalyticsStore
}

// NewAnalytics creates an Analytics handler.
func NewAnalytics(analytics store.AnalyticsStore) *Analytics { return &Analytics{analytics: analytics} }

// Post returns post analytics.
func (h *Analytics) Post(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.analytics.GetPostMetrics(r.Context(), chi.URLParam(r, "postID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "analytics not found")
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

// Account returns account analytics.
func (h *Analytics) Account(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.analytics.GetAccountMetrics(r.Context(), chi.URLParam(r, "accountID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "analytics not found")
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

// Summary returns summary analytics.
func (h *Analytics) Summary(w http.ResponseWriter, r *http.Request) {
	ws := tenant.MustFromContext(r.Context())
	
	var from, to *time.Time
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = &t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = &t
		}
	}

	metrics, err := h.analytics.GetSummary(r.Context(), ws.ID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}
