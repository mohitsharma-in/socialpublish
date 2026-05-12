package socialpublish

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTransportInjectsAuthAndDecodesResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer sp_test_key", r.Header.Get(headerAuthorization))
		require.Equal(t, "socialpublish-go/"+sdkVersion, r.Header.Get(headerUserAgent))
		require.Equal(t, "/v1/posts", r.URL.Path)
		require.Equal(t, "20", r.URL.Query().Get("limit"))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"post_id":"post_123","status":"draft"}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	tp := newTransport(testConfig(server.URL))
	var out struct {
		ID     string `json:"post_id"`
		Status string `json:"status"`
	}

	err := tp.DoJSON(context.Background(), http.MethodGet, "/v1/posts", url.Values{"limit": {"20"}}, nil, &out)
	require.NoError(t, err)
	require.Equal(t, "post_123", out.ID)
	require.Equal(t, "draft", out.Status)
}

func TestTransportRetriesRetryableStatus(t *testing.T) {
	t.Parallel()

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte(`{"code":"internal_error","message":"temporary"}`))
			require.NoError(t, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"ok":true}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)
	cfg.maxRetries = 1
	tp := newTransport(cfg)
	var out struct {
		OK bool `json:"ok"`
	}

	err := tp.DoJSON(context.Background(), http.MethodGet, "/health", nil, nil, &out)
	require.NoError(t, err)
	require.True(t, out.OK)
	require.Equal(t, 2, attempts)
}

func TestTransportParsesAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headerRequestID, "req_123")
		w.Header().Set(headerRetryAfter, "1")
		w.WriteHeader(http.StatusTooManyRequests)
		_, err := w.Write([]byte(`{"code":"rate_limit","message":"slow down","detail":{"limit":60}}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)
	cfg.maxRetries = 0
	tp := newTransport(cfg)

	err := tp.DoJSON(context.Background(), http.MethodGet, "/v1/posts", nil, nil, nil)
	require.Error(t, err)

	var apiErr *Error
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, CodeRateLimit, apiErr.Code)
	require.Equal(t, http.StatusTooManyRequests, apiErr.HTTPStatus)
	require.Equal(t, "req_123", apiErr.RequestID)
	require.Equal(t, "slow down", apiErr.Message)
	require.NotNil(t, apiErr.RetryAfter)
	require.Equal(t, time.Second, *apiErr.RetryAfter)
	require.Equal(t, float64(60), apiErr.Detail["limit"])
}

func testConfig(baseURL string) config {
	return config{
		apiKey:       "sp_test_key",
		baseURL:      baseURL,
		timeout:      2 * time.Second,
		maxRetries:   0,
		retryWaitMin: time.Millisecond,
		retryWaitMax: time.Millisecond,
	}
}
