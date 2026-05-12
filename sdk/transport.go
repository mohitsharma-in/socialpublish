package socialpublish

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	headerAuthorization = "Authorization"
	headerRequestID     = "X-Request-ID"
	headerRetryAfter    = "Retry-After"
	headerUserAgent     = "User-Agent"
	fallbackRetryMin    = 100 * time.Millisecond
	fallbackRetryMax    = 5 * time.Second
)

type transport struct {
	cfg    config
	client *http.Client
}

func newTransport(cfg config) *transport {
	hc := cfg.httpClient
	if hc == nil {
		hc = &http.Client{Timeout: cfg.timeout}
	}
	if hc.Timeout == 0 {
		copied := *hc
		copied.Timeout = cfg.timeout
		hc = &copied
	}
	return &transport{cfg: cfg, client: hc}
}

// DoJSON sends a JSON request and decodes a JSON response.
func (t *transport) DoJSON(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
	}

	var lastErr error
	attempts := t.cfg.maxRetries + 1
	for attempt := range attempts {
		req, err := t.newRequest(ctx, method, path, query, payload)
		if err != nil {
			return err
		}

		resp, err := t.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("send request: %w", err)
			if attempt < attempts-1 {
				t.sleep(ctx, attempt, nil)
				continue
			}
			return lastErr
		}

		err = t.handleResponse(resp, out)
		if err == nil {
			return nil
		}
		lastErr = err

		var apiErr *Error
		if !errors.As(err, &apiErr) || !apiErr.IsRetryable() || attempt == attempts-1 {
			return err
		}
		t.sleep(ctx, attempt, apiErr.RetryAfter)
	}
	return lastErr
}

func (t *transport) newRequest(ctx context.Context, method, path string, query url.Values, payload []byte) (*http.Request, error) {
	u, err := url.Parse(t.cfg.baseURL + "/" + strings.TrimLeft(path, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse request URL: %w", err)
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set(headerAuthorization, "Bearer "+t.cfg.apiKey)
	req.Header.Set(headerUserAgent, "socialpublish-go/"+sdkVersion)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (t *transport) handleResponse(resp *http.Response, out any) error {
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		if out == nil || resp.StatusCode == http.StatusNoContent {
			_, _ = io.Copy(io.Discard, resp.Body)
			return nil
		}
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		return nil
	}

	apiErr := &Error{
		Code:       codeForStatus(resp.StatusCode),
		Message:    http.StatusText(resp.StatusCode),
		HTTPStatus: resp.StatusCode,
		RequestID:  resp.Header.Get(headerRequestID),
	}
	if retryAfter := parseRetryAfter(resp.Header.Get(headerRetryAfter)); retryAfter != nil {
		apiErr.RetryAfter = retryAfter
	}

	var body struct {
		Code     Code           `json:"code"`
		Message  string         `json:"message"`
		Platform string         `json:"platform"`
		Detail   map[string]any `json:"detail"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err == nil {
		if body.Code != "" {
			apiErr.Code = body.Code
		}
		if body.Message != "" {
			apiErr.Message = body.Message
		}
		apiErr.Platform = body.Platform
		apiErr.Detail = body.Detail
	}
	return apiErr
}

func (t *transport) sleep(ctx context.Context, attempt int, retryAfter *time.Duration) {
	wait := t.backoff(attempt)
	if retryAfter != nil {
		wait = *retryAfter
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func (t *transport) backoff(attempt int) time.Duration {
	minWait := t.cfg.retryWaitMin
	if minWait <= 0 {
		minWait = fallbackRetryMin
	}
	maxWait := t.cfg.retryWaitMax
	if maxWait <= 0 {
		maxWait = fallbackRetryMax
	}
	multiplier := math.Pow(2, float64(attempt))
	wait := time.Duration(float64(minWait) * multiplier)
	if wait > maxWait {
		return maxWait
	}
	return wait
}

func parseRetryAfter(value string) *time.Duration {
	if value == "" {
		return nil
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		d := time.Duration(seconds) * time.Second
		return &d
	}
	if when, err := http.ParseTime(value); err == nil {
		d := time.Until(when)
		if d < 0 {
			d = 0
		}
		return &d
	}
	return nil
}

func codeForStatus(status int) Code {
	switch status {
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusTooManyRequests:
		return CodeRateLimit
	case http.StatusUnprocessableEntity:
		return CodeValidation
	default:
		if status >= http.StatusInternalServerError {
			return CodeInternal
		}
		return CodeValidation
	}
}
