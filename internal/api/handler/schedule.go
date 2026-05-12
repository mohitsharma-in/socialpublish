package handler

import (
	"net/http"

	"github.com/mohitsharma-in/socialpublish/internal/store"
)

// Schedule handles schedule routes.
type Schedule struct {
	posts store.PostStore
}

// NewSchedule creates a Schedule handler.
func NewSchedule(posts store.PostStore) *Schedule { return &Schedule{posts: posts} }

// Calendar returns scheduled posts.
func (h *Schedule) Calendar(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Queue returns scheduled work.
func (h *Schedule) Queue(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// NextAvailable returns the next available slot.
func (h *Schedule) NextAvailable(w http.ResponseWriter, r *http.Request) { notImplemented(w) }
