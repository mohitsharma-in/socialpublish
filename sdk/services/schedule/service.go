package schedule

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

// Service manages scheduling operations.
type Service interface {
	Calendar(ctx context.Context, params *CalendarParams) (*CalendarResponse, error)
	Queue(ctx context.Context, params *QueueParams) (*QueueResponse, error)
	NextAvailable(ctx context.Context) (*Window, error)
}

type service struct {
	doer Doer
}

// New creates a schedule Service.
func New(doer Doer) Service {
	return &service{doer: doer}
}

// Calendar returns scheduled posts in a time range.
func (s *service) Calendar(ctx context.Context, params *CalendarParams) (*CalendarResponse, error) {
	query := url.Values{}
	if params != nil {
		if params.From != nil {
			query.Set("from", params.From.Format("2006-01-02T15:04:05Z07:00"))
		}
		if params.To != nil {
			query.Set("to", params.To.Format("2006-01-02T15:04:05Z07:00"))
		}
	}
	var out CalendarResponse
	if err := s.doer.DoJSON(ctx, http.MethodGet, "/v1/schedule", query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Queue returns pending scheduled work.
func (s *service) Queue(ctx context.Context, params *QueueParams) (*QueueResponse, error) {
	query := url.Values{}
	if params != nil {
		if params.Limit > 0 {
			query.Set("limit", fmt.Sprintf("%d", params.Limit))
		}
		if params.Cursor != "" {
			query.Set("cursor", params.Cursor)
		}
	}
	var out QueueResponse
	if err := s.doer.DoJSON(ctx, http.MethodGet, "/v1/schedule/queue", query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// NextAvailable returns the next available publishing window.
func (s *service) NextAvailable(ctx context.Context) (*Window, error) {
	var out Window
	if err := s.doer.DoJSON(ctx, http.MethodGet, "/v1/schedule/next-available", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
