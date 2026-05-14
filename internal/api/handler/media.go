package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mohitsharma-in/socialpublish/internal/queue"
	"github.com/mohitsharma-in/socialpublish/internal/storage"
	"github.com/mohitsharma-in/socialpublish/internal/store"
	"github.com/mohitsharma-in/socialpublish/internal/tenant"
)

// Media handles media routes.
type Media struct {
	media store.MediaStore
	obj   storage.ObjectStorage
	q     queue.Queue
}

// NewMedia creates a Media handler.
func NewMedia(media store.MediaStore, obj storage.ObjectStorage, q queue.Queue) *Media {
	return &Media{media: media, obj: obj, q: q}
}

type uploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	MediaType   string `json:"media_type"`
}

// Upload starts a media upload by returning a presigned URL.
func (h *Media) Upload(w http.ResponseWriter, r *http.Request) {
	ws := tenant.MustFromContext(r.Context())
	var req uploadRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}
	if req.Filename == "" || req.ContentType == "" || req.MediaType == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "filename, content_type, and media_type required")
		return
	}
	key := fmt.Sprintf("uploads/%s/%s/%s", ws.ID, uuid.New().String(), req.Filename)
	m := &store.Media{WorkspaceID: ws.ID, Status: "uploading", MediaType: req.MediaType, OriginalKey: key, MimeType: req.ContentType}
	if err := h.media.Create(r.Context(), m); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	uploadURL, headers, err := h.obj.PresignUpload(r.Context(), key, req.ContentType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"media_id": m.ID, "upload_url": uploadURL, "headers": headers})
}

// List lists media.
func (h *Media) List(w http.ResponseWriter, r *http.Request) {
	ws := tenant.MustFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, next, err := h.media.List(r.Context(), ws.ID, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items, "next_cursor": next})
}

// Get fetches media.
func (h *Media) Get(w http.ResponseWriter, r *http.Request) {
	m, err := h.media.Get(r.Context(), chi.URLParam(r, "mediaID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "media not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// Delete deletes media.
func (h *Media) Delete(w http.ResponseWriter, r *http.Request) {
	mediaID := chi.URLParam(r, "mediaID")
	m, err := h.media.Get(r.Context(), mediaID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "media not found")
		return
	}
	_ = h.obj.Delete(r.Context(), m.OriginalKey)
	if err := h.media.Delete(r.Context(), mediaID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SetThumbnail sets a thumbnail.
func (h *Media) SetThumbnail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ThumbnailKey string `json:"thumbnail_key"`
	}
	if err := decodeBody(r, &req); err != nil || req.ThumbnailKey == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "thumbnail_key required")
		return
	}
	if err := h.media.SetThumbnail(r.Context(), chi.URLParam(r, "mediaID"), req.ThumbnailKey); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
