package socialpublish

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	signatureHeader       = "X-SocialPublish-Signature"
	eventHeader           = "X-SocialPublish-Event"
	defaultClockTolerance = 5 * time.Minute
)

// WebhookEvent is the common envelope delivered to webhook handlers.
type WebhookEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	CreatedAt time.Time       `json:"created_at"`
	Data      json.RawMessage `json:"data"`
}

// WebhookHandler handles one verified webhook event.
type WebhookHandler func(ctx context.Context, event WebhookEvent) error

// WebhookRouter verifies and dispatches webhook deliveries.
type WebhookRouter struct {
	secret         string
	clockTolerance time.Duration
	handlers       map[string]WebhookHandler
}

// NewWebhookRouter creates a webhook router with the signing secret.
func NewWebhookRouter(secret string) *WebhookRouter {
	return &WebhookRouter{
		secret:         secret,
		clockTolerance: defaultClockTolerance,
		handlers:       make(map[string]WebhookHandler),
	}
}

// WithClockTolerance sets the accepted timestamp skew for webhook signatures.
func (r *WebhookRouter) WithClockTolerance(d time.Duration) *WebhookRouter {
	r.clockTolerance = d
	return r
}

// Handle registers a handler for an event type.
func (r *WebhookRouter) Handle(eventType string, handler WebhookHandler) {
	r.handlers[eventType] = handler
}

// ServeHTTP verifies the request and dispatches it to the registered handler.
func (r *WebhookRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	event, err := r.Parse(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	handler, ok := r.handlers[event.Type]
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := handler(req.Context(), event); err != nil {
		http.Error(w, "webhook handler failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Parse verifies and decodes a webhook request.
func (r *WebhookRouter) Parse(req *http.Request) (WebhookEvent, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return WebhookEvent{}, fmt.Errorf("read webhook body: %w", err)
	}
	if err := VerifyWebhookSignature(r.secret, req.Header.Get(signatureHeader), body, time.Now(), r.clockTolerance); err != nil {
		return WebhookEvent{}, err
	}

	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return WebhookEvent{}, fmt.Errorf("decode webhook event: %w", err)
	}
	if event.Type == "" {
		event.Type = req.Header.Get(eventHeader)
	}
	if event.Type == "" {
		return WebhookEvent{}, fmt.Errorf("webhook event type is required")
	}
	return event, nil
}

// SignWebhookPayload signs a payload using the SocialPublish webhook format.
func SignWebhookPayload(secret string, payload []byte, now time.Time) string {
	timestamp := now.Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	return fmt.Sprintf("t=%d,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
}

// VerifyWebhookSignature verifies a SocialPublish webhook signature.
func VerifyWebhookSignature(secret string, signature string, payload []byte, now time.Time, tolerance time.Duration) error {
	if secret == "" {
		return fmt.Errorf("webhook secret is required")
	}
	timestamp, got, err := parseSignature(signature)
	if err != nil {
		return err
	}
	signedAt := time.Unix(timestamp, 0)
	if tolerance > 0 && now.Sub(signedAt).Abs() > tolerance {
		return fmt.Errorf("webhook signature timestamp outside tolerance")
	}

	expected := SignWebhookPayload(secret, payload, signedAt)
	_, expectedSig, err := parseSignature(expected)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(got), []byte(expectedSig)) {
		return fmt.Errorf("webhook signature mismatch")
	}
	return nil
}

func parseSignature(signature string) (int64, string, error) {
	if signature == "" {
		return 0, "", fmt.Errorf("webhook signature is required")
	}
	parts := strings.Split(signature, ",")
	var timestamp int64
	var sig string
	for _, part := range parts {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return 0, "", fmt.Errorf("parse webhook signature timestamp: %w", err)
			}
			timestamp = parsed
		case "v1":
			sig = value
		}
	}
	if timestamp == 0 || sig == "" {
		return 0, "", fmt.Errorf("webhook signature must include t and v1")
	}
	return timestamp, sig, nil
}
