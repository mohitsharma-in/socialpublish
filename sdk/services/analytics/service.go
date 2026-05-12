package analytics

import (
	"context"
	"net/http"
	"net/url"
)

// Doer sends SDK HTTP requests.
type Doer interface {
	DoJSON(ctx context.Context, method, path string, query url.Values, body any, out any) error
}

// Service manages analytics retrieval.
type Service interface {
	Post(ctx context.Context, postID string) (*PostMetrics, error)
	Account(ctx context.Context, accountID string) (*AccountMetrics, error)
	Summary(ctx context.Context, params *SummaryParams) (*Summary, error)
}

type service struct {
	doer Doer
}

// New creates an analytics Service.
func New(doer Doer) Service {
	return &service{doer: doer}
}

// Post fetches post analytics.
func (s *service) Post(ctx context.Context, postID string) (*PostMetrics, error) {
	var out PostMetrics
	if err := s.doer.DoJSON(ctx, http.MethodGet, "/v1/analytics/posts/"+url.PathEscape(postID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Account fetches account analytics.
func (s *service) Account(ctx context.Context, accountID string) (*AccountMetrics, error) {
	var out AccountMetrics
	if err := s.doer.DoJSON(ctx, http.MethodGet, "/v1/analytics/accounts/"+url.PathEscape(accountID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Summary fetches workspace analytics totals.
func (s *service) Summary(ctx context.Context, params *SummaryParams) (*Summary, error) {
	query := url.Values{}
	if params != nil {
		if params.From != nil {
			query.Set("from", params.From.Format("2006-01-02T15:04:05Z07:00"))
		}
		if params.To != nil {
			query.Set("to", params.To.Format("2006-01-02T15:04:05Z07:00"))
		}
	}
	var out Summary
	if err := s.doer.DoJSON(ctx, http.MethodGet, "/v1/analytics/summary", query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
