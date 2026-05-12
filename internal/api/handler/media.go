package handler

import (
	"net/http"

	"github.com/yourorg/socialpublish/internal/queue"
	"github.com/yourorg/socialpublish/internal/storage"
	"github.com/yourorg/socialpublish/internal/store"
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

// Upload starts a media upload.
func (h *Media) Upload(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// List lists media.
func (h *Media) List(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Get fetches media.
func (h *Media) Get(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Delete deletes media.
func (h *Media) Delete(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// SetThumbnail sets a thumbnail.
func (h *Media) SetThumbnail(w http.ResponseWriter, r *http.Request) { notImplemented(w) }
