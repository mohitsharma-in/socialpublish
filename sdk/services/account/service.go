package account

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

// Service manages connected social accounts.
type Service interface {
	Connect(ctx context.Context, req *ConnectRequest) (*ConnectResponse, error)
	Get(ctx context.Context, accountID string) (*Account, error)
	List(ctx context.Context, params *ListParams) (*Page, error)
	Delete(ctx context.Context, accountID string) error
	Status(ctx context.Context, accountID string) (*Account, error)
}

type service struct {
	doer Doer
}

// New creates an account Service.
func New(doer Doer) Service {
	return &service{doer: doer}
}

// Connect connects a social account.
func (s *service) Connect(ctx context.Context, req *ConnectRequest) (*ConnectResponse, error) {
	var out ConnectResponse
	if err := s.doer.DoJSON(ctx, http.MethodPost, "/v1/accounts/connect", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get fetches an account by ID.
func (s *service) Get(ctx context.Context, accountID string) (*Account, error) {
	var out Account
	if err := s.doer.DoJSON(ctx, http.MethodGet, "/v1/accounts/"+url.PathEscape(accountID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List fetches connected accounts.
func (s *service) List(ctx context.Context, params *ListParams) (*Page, error) {
	query := url.Values{}
	if params != nil {
		if params.Limit > 0 {
			query.Set("limit", fmt.Sprintf("%d", params.Limit))
		}
		if params.Cursor != "" {
			query.Set("cursor", params.Cursor)
		}
		if params.Platform != "" {
			query.Set("platform", string(params.Platform))
		}
	}
	var out Page
	if err := s.doer.DoJSON(ctx, http.MethodGet, "/v1/accounts", query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes an account by ID.
func (s *service) Delete(ctx context.Context, accountID string) error {
	return s.doer.DoJSON(ctx, http.MethodDelete, "/v1/accounts/"+url.PathEscape(accountID), nil, nil, nil)
}

// Status fetches account health status.
func (s *service) Status(ctx context.Context, accountID string) (*Account, error) {
	var out Account
	if err := s.doer.DoJSON(ctx, http.MethodGet, "/v1/accounts/"+url.PathEscape(accountID)+"/status", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
