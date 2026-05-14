package handler

import (
	"net/http"
	"time"

	"github.com/mohitsharma-in/socialpublish/internal/store"
	"github.com/mohitsharma-in/socialpublish/internal/tenant"
)

// Schedule handles schedule routes.
type Schedule struct {
	posts store.PostStore
}

// NewSchedule creates a Schedule handler.
func NewSchedule(posts store.PostStore) *Schedule { return &Schedule{posts: posts} }

// Calendar returns scheduled posts in a time range.
func (h *Schedule) Calendar(w http.ResponseWriter, r *http.Request) {
	ws := tenant.MustFromContext(r.Context())
	from := time.Now()
	to := from.Add(30 * 24 * time.Hour)
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil { from = t }
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil { to = t }
	}
	posts, err := h.posts.ListScheduled(r.Context(), ws.ID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": posts, "from": from, "to": to})
}

// Queue returns posts in publishing state.
func (h *Schedule) Queue(w http.ResponseWriter, r *http.Request) {
	ws := tenant.MustFromContext(r.Context())
	posts, _, err := h.posts.List(r.Context(), ws.ID, 50, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	var queued []store.Post
	for _, p := range posts {
		if p.Status == "publishing" || p.Status == "scheduled" {
			queued = append(queued, p)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": queued})
}

// NextAvailable returns the next available slot.
func (h *Schedule) NextAvailable(w http.ResponseWriter, r *http.Request) {
	next := time.Now().Add(time.Hour).Truncate(time.Hour)
	writeJSON(w, http.StatusOK, map[string]any{"next_available": next})
}
