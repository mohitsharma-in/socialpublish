package handler

import (
	"net/http"

	"github.com/mohitsharma-in/socialpublish/internal/queue"
	"github.com/mohitsharma-in/socialpublish/internal/store"
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

// Create creates a post.
func (h *Post) Create(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// List lists posts.
func (h *Post) List(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Get fetches a post.
func (h *Post) Get(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Update updates a post.
func (h *Post) Update(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Delete deletes a post.
func (h *Post) Delete(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Publish publishes a post.
func (h *Post) Publish(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Cancel cancels a post.
func (h *Post) Cancel(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Duplicate duplicates a post.
func (h *Post) Duplicate(w http.ResponseWriter, r *http.Request) { notImplemented(w) }
