package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mohitsharma-in/socialpublish/internal/queue"
	"github.com/mohitsharma-in/socialpublish/internal/store"
	"github.com/mohitsharma-in/socialpublish/internal/tenant"
)

// Post handles post routes.
type Post struct {
	posts store.PostStore
	media store.MediaStore
	q     queue.Queue
}

// NewPost creates a Post handler.
func NewPost(posts store.PostStore, media store.MediaStore, q queue.Queue) *Post {
	return &Post{posts: posts, media: media, q: q}
}

type createPostRequest struct {
	MediaIDs    []string       `json:"media_ids"`
	ScheduledAt *time.Time     `json:"scheduled_at,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Targets     []targetReq    `json:"targets"`
}
type targetReq struct {
	AccountID string         `json:"account_id"`
	Platform  string         `json:"platform"`
	Format    string         `json:"format"`
	Config    map[string]any `json:"config,omitempty"`
}

// Create creates a post.
func (h *Post) Create(w http.ResponseWriter, r *http.Request) {
	ws := tenant.MustFromContext(r.Context())
	var req createPostRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}
	status := "draft"
	if req.ScheduledAt != nil {
		status = "scheduled"
	}
	p := &store.Post{WorkspaceID: ws.ID, Status: status, MediaIDs: req.MediaIDs, ScheduledAt: req.ScheduledAt, Metadata: req.Metadata}
	if err := h.posts.Create(r.Context(), p); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("create post: %v", err))
		return
	}
	for _, t := range req.Targets {
		target := &store.PostTarget{PostID: p.ID, AccountID: t.AccountID, Platform: t.Platform, Format: t.Format, Config: t.Config}
		if err := h.posts.CreateTarget(r.Context(), target); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("create target: %v", err))
			return
		}
	}
	writeJSON(w, http.StatusCreated, p)
}

// List lists posts.
func (h *Post) List(w http.ResponseWriter, r *http.Request) {
	ws := tenant.MustFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	cursor := r.URL.Query().Get("cursor")
	posts, next, err := h.posts.List(r.Context(), ws.ID, limit, cursor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": posts, "next_cursor": next})
}

// Get fetches a post.
func (h *Post) Get(w http.ResponseWriter, r *http.Request) {
	p, err := h.posts.Get(r.Context(), chi.URLParam(r, "postID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "post not found")
		return
	}
	targets, _ := h.posts.ListTargets(r.Context(), p.ID)
	writeJSON(w, http.StatusOK, map[string]any{"post": p, "targets": targets})
}

// Update updates a post.
func (h *Post) Update(w http.ResponseWriter, r *http.Request) {
	var fields map[string]any
	if err := decodeBody(r, &fields); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid body")
		return
	}
	if err := h.posts.Update(r.Context(), chi.URLParam(r, "postID"), fields); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	p, _ := h.posts.Get(r.Context(), chi.URLParam(r, "postID"))
	writeJSON(w, http.StatusOK, p)
}

// Delete deletes a post.
func (h *Post) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.posts.Delete(r.Context(), chi.URLParam(r, "postID")); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "post not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Publish publishes a post.
func (h *Post) Publish(w http.ResponseWriter, r *http.Request) {
	postID := chi.URLParam(r, "postID")
	if err := h.posts.SetStatus(r.Context(), postID, "publishing"); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	targets, err := h.posts.ListTargets(r.Context(), postID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	for _, t := range targets {
		payload, _ := json.Marshal(map[string]string{"target_id": t.ID, "post_id": postID})
		_ = h.q.Enqueue(r.Context(), "post:publish", payload)
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "publishing", "post_id": postID})
}

// Cancel cancels a post.
func (h *Post) Cancel(w http.ResponseWriter, r *http.Request) {
	if err := h.posts.SetStatus(r.Context(), chi.URLParam(r, "postID"), "cancelled"); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// Duplicate duplicates a post.
func (h *Post) Duplicate(w http.ResponseWriter, r *http.Request) {
	ws := tenant.MustFromContext(r.Context())
	orig, err := h.posts.Get(r.Context(), chi.URLParam(r, "postID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "post not found")
		return
	}
	dup := &store.Post{WorkspaceID: ws.ID, Status: "draft", MediaIDs: orig.MediaIDs, Metadata: orig.Metadata}
	if err := h.posts.Create(r.Context(), dup); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, dup)
}
