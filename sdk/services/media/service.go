package media

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

// Service manages media upload and processing operations.
type Service interface {
	CreateUpload(ctx context.Context, req *UploadRequest) (*UploadResponse, error)
	Get(ctx context.Context, mediaID string) (*Asset, error)
	List(ctx context.Context, params *ListParams) (*Page, error)
	Delete(ctx context.Context, mediaID string) error
	SetThumbnail(ctx context.Context, mediaID string, key string) (*Asset, error)
}

type service struct {
	doer Doer
}

// New creates a media Service.
func New(doer Doer) Service {
	return &service{doer: doer}
}

// CreateUpload starts a direct upload.
func (s *service) CreateUpload(ctx context.Context, req *UploadRequest) (*UploadResponse, error) {
	var out UploadResponse
	if err := s.doer.DoJSON(ctx, http.MethodPost, "/v1/media/upload", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get fetches media by ID.
func (s *service) Get(ctx context.Context, mediaID string) (*Asset, error) {
	var out Asset
	if err := s.doer.DoJSON(ctx, http.MethodGet, "/v1/media/"+url.PathEscape(mediaID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List fetches media assets.
func (s *service) List(ctx context.Context, params *ListParams) (*Page, error) {
	query := url.Values{}
	if params != nil {
		if params.Limit > 0 {
			query.Set("limit", fmt.Sprintf("%d", params.Limit))
		}
		if params.Cursor != "" {
			query.Set("cursor", params.Cursor)
		}
		if params.Status != "" {
			query.Set("status", string(params.Status))
		}
		if params.Type != "" {
			query.Set("type", string(params.Type))
		}
	}
	var out Page
	if err := s.doer.DoJSON(ctx, http.MethodGet, "/v1/media", query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes media by ID.
func (s *service) Delete(ctx context.Context, mediaID string) error {
	return s.doer.DoJSON(ctx, http.MethodDelete, "/v1/media/"+url.PathEscape(mediaID), nil, nil, nil)
}

// SetThumbnail updates the media thumbnail.
func (s *service) SetThumbnail(ctx context.Context, mediaID string, key string) (*Asset, error) {
	var out Asset
	body := map[string]string{"thumbnail_key": key}
	if err := s.doer.DoJSON(ctx, http.MethodPost, "/v1/media/"+url.PathEscape(mediaID)+"/thumbnail", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
