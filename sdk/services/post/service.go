package post

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Doer sends SDK HTTP requests.
type Doer interface {
	DoJSON(ctx context.Context, method, path string, query url.Values, body any, out any) error
}

// Service manages post lifecycle operations.
type Service interface {
	Create(ctx context.Context, req *CreateRequest) (*Post, error)
	Get(ctx context.Context, postID string) (*Post, error)
	List(ctx context.Context, params *ListParams) (*Page, error)
	Update(ctx context.Context, postID string, req *UpdateRequest) (*Post, error)
	Delete(ctx context.Context, postID string) error
	Publish(ctx context.Context, postID string) (*Post, error)
	Cancel(ctx context.Context, postID string) (*Post, error)
}

type service struct {
	doer Doer
}

// New creates a post Service.
func New(doer Doer) Service {
	return &service{doer: doer}
}

// Create creates a post.
func (s *service) Create(ctx context.Context, req *CreateRequest) (*Post, error) {
	var out Post
	if err := s.doer.DoJSON(ctx, http.MethodPost, "/v1/posts", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get fetches a post by ID.
func (s *service) Get(ctx context.Context, postID string) (*Post, error) {
	var out Post
	if err := s.doer.DoJSON(ctx, http.MethodGet, "/v1/posts/"+url.PathEscape(postID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List fetches posts.
func (s *service) List(ctx context.Context, params *ListParams) (*Page, error) {
	query := listQuery(params)
	var out Page
	if err := s.doer.DoJSON(ctx, http.MethodGet, "/v1/posts", query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update updates a post.
func (s *service) Update(ctx context.Context, postID string, req *UpdateRequest) (*Post, error) {
	var out Post
	if err := s.doer.DoJSON(ctx, http.MethodPatch, "/v1/posts/"+url.PathEscape(postID), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes a post.
func (s *service) Delete(ctx context.Context, postID string) error {
	return s.doer.DoJSON(ctx, http.MethodDelete, "/v1/posts/"+url.PathEscape(postID), nil, nil, nil)
}

// Publish publishes a post immediately.
func (s *service) Publish(ctx context.Context, postID string) (*Post, error) {
	var out Post
	if err := s.doer.DoJSON(ctx, http.MethodPost, "/v1/posts/"+url.PathEscape(postID)+"/publish", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Cancel cancels a scheduled post.
func (s *service) Cancel(ctx context.Context, postID string) (*Post, error) {
	var out Post
	if err := s.doer.DoJSON(ctx, http.MethodPost, "/v1/posts/"+url.PathEscape(postID)+"/cancel", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func listQuery(params *ListParams) url.Values {
	query := url.Values{}
	if params == nil {
		return query
	}
	if params.Limit > 0 {
		query.Set("limit", strconvItoa(params.Limit))
	}
	if params.Cursor != "" {
		query.Set("cursor", params.Cursor)
	}
	if params.Status != "" {
		query.Set("status", string(params.Status))
	}
	if params.Platform != "" {
		query.Set("platform", string(params.Platform))
	}
	if params.From != nil {
		query.Set("from", params.From.Format(timeRFC3339))
	}
	if params.To != nil {
		query.Set("to", params.To.Format(timeRFC3339))
	}
	return query
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func strconvItoa(v int) string {
	return fmt.Sprintf("%d", v)
}
