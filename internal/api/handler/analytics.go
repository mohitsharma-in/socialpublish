package handler

import (
	"net/http"

	"github.com/yourorg/socialpublish/internal/store"
)

// Analytics handles analytics routes.
type Analytics struct {
	analytics store.AnalyticsStore
}

// NewAnalytics creates an Analytics handler.
func NewAnalytics(analytics store.AnalyticsStore) *Analytics { return &Analytics{analytics: analytics} }

// Post returns post analytics.
func (h *Analytics) Post(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Account returns account analytics.
func (h *Analytics) Account(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Summary returns summary analytics.
func (h *Analytics) Summary(w http.ResponseWriter, r *http.Request) { notImplemented(w) }
