package socialpublish

import (
	"net/http"
	"time"
)

// Option configures a Client.
type Option func(*config)

type config struct {
	apiKey       string
	baseURL      string
	timeout      time.Duration
	maxRetries   int
	retryWaitMin time.Duration
	retryWaitMax time.Duration
	httpClient   *http.Client
}

// WithAPIKey sets the API key.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithBaseURL overrides the API base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithTimeout sets the per-request HTTP timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// WithHTTPClient injects a custom *http.Client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *config) { c.httpClient = hc }
}

// WithMaxRetries sets the maximum number of retry attempts for 429 and 5xx responses.
func WithMaxRetries(n int) Option {
	return func(c *config) { c.maxRetries = n }
}
