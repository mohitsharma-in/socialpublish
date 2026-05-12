package socialpublish

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Code classifies API errors.
type Code string

const (
	// CodeUnauthorized indicates missing or invalid authentication.
	CodeUnauthorized Code = "unauthorized"
	// CodeForbidden indicates valid authentication without required permission.
	CodeForbidden Code = "forbidden"
	// CodeNotFound indicates the requested resource does not exist.
	CodeNotFound Code = "not_found"
	// CodeRateLimit indicates the request exceeded the workspace rate limit.
	CodeRateLimit Code = "rate_limit"
	// CodeValidation indicates request validation failed.
	CodeValidation Code = "validation_error"
	// CodePlatformError indicates an upstream platform rejected the operation.
	CodePlatformError Code = "platform_error"
	// CodeMediaNotReady indicates the media has not finished processing.
	CodeMediaNotReady Code = "media_not_ready"
	// CodeTranscodeFail indicates media transcoding failed.
	CodeTranscodeFail Code = "transcode_failed"
	// CodePublishFailed indicates publishing failed.
	CodePublishFailed Code = "publish_failed"
	// CodeTokenExpired indicates a connected account token is expired.
	CodeTokenExpired Code = "token_expired"
	// CodeQuotaExceeded indicates a plan or platform quota was exceeded.
	CodeQuotaExceeded Code = "quota_exceeded"
	// CodeInternal indicates an internal service error.
	CodeInternal Code = "internal_error"
)

// Error is the structured error type returned by SDK methods.
type Error struct {
	Code       Code           `json:"code"`
	Message    string         `json:"message"`
	HTTPStatus int            `json:"http_status"`
	RequestID  string         `json:"request_id"`
	Platform   string         `json:"platform,omitempty"`
	RetryAfter *time.Duration `json:"-"`
	Detail     map[string]any `json:"detail,omitempty"`
}

// Error formats the API error for logs and diagnostics.
func (e *Error) Error() string {
	if e.Platform != "" {
		return fmt.Sprintf("[%s/%s] %s (request_id=%s)", e.Platform, e.Code, e.Message, e.RequestID)
	}
	return fmt.Sprintf("[%s] %s (request_id=%s)", e.Code, e.Message, e.RequestID)
}

// Is matches another *Error by Code, enabling errors.Is.
func (e *Error) Is(target error) bool {
	var t *Error
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

// IsRetryable reports whether the error is safe to retry.
func (e *Error) IsRetryable() bool {
	return e.HTTPStatus == http.StatusTooManyRequests || e.HTTPStatus >= http.StatusInternalServerError
}

var (
	// ErrUnauthorized matches unauthorized API errors.
	ErrUnauthorized = &Error{Code: CodeUnauthorized, HTTPStatus: http.StatusUnauthorized}
	// ErrForbidden matches forbidden API errors.
	ErrForbidden = &Error{Code: CodeForbidden, HTTPStatus: http.StatusForbidden}
	// ErrNotFound matches not-found API errors.
	ErrNotFound = &Error{Code: CodeNotFound, HTTPStatus: http.StatusNotFound}
	// ErrRateLimit matches rate-limit API errors.
	ErrRateLimit = &Error{Code: CodeRateLimit, HTTPStatus: http.StatusTooManyRequests}
	// ErrMediaNotReady matches media-not-ready API errors.
	ErrMediaNotReady = &Error{Code: CodeMediaNotReady}
	// ErrTokenExpired matches token-expired API errors.
	ErrTokenExpired = &Error{Code: CodeTokenExpired}
	// ErrPublishFailed matches publish-failed API errors.
	ErrPublishFailed = &Error{Code: CodePublishFailed}
)

// ValidationError wraps field-level validation failures.
type ValidationError struct {
	Fields []FieldError `json:"fields"`
}

// Error formats all field validation failures.
func (e *ValidationError) Error() string {
	msgs := make([]string, len(e.Fields))
	for i, field := range e.Fields {
		msgs[i] = field.Field + ": " + field.Message
	}
	return "validation failed: " + strings.Join(msgs, "; ")
}

// FieldError describes a single field validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
