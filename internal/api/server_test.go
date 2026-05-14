package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mohitsharma-in/socialpublish/internal/api/middleware"
	"github.com/mohitsharma-in/socialpublish/internal/store"
)

func TestHealthRoute(t *testing.T) {
	t.Parallel()
	server := New(Config{}, store.Stores{}, nil, nil, middleware.InMemoryRateLimiter{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	server.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
}

func TestReadyzWithoutCheck(t *testing.T) {
	t.Parallel()
	s := New(Config{}, store.Stores{}, nil, nil, middleware.InMemoryRateLimiter{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/readyz", nil)
	s.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"status":"ready"}`, rec.Body.String())
}

func TestReadyzCheckPasses(t *testing.T) {
	t.Parallel()
	s := New(Config{
		ReadinessCheck: func(context.Context) error { return nil },
	}, store.Stores{}, nil, nil, middleware.InMemoryRateLimiter{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/readyz", nil)
	s.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"status":"ready"}`, rec.Body.String())
}

func TestReadyzCheckFails(t *testing.T) {
	t.Parallel()
	s := New(Config{
		ReadinessCheck: func(context.Context) error { return errors.New("db down") },
	}, store.Stores{}, nil, nil, middleware.InMemoryRateLimiter{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/readyz", nil)
	s.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.JSONEq(t, `{"status":"not_ready"}`, rec.Body.String())
}
