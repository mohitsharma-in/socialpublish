package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/hibiken/asynq"

	socialpublish "github.com/mohitsharma-in/socialpublish/sdk"
	"github.com/mohitsharma-in/socialpublish/internal/store"
)

// WebhookPayload is serialized into webhook delivery jobs.
type WebhookPayload struct {
	WorkspaceID string         `json:"workspace_id"`
	EventType   string         `json:"event_type"`
	Payload     map[string]any `json:"payload"`
}

type webhookDeliverHandler struct {
	webhooks store.WebhookStore
	tokens   store.TokenStore
	client   *http.Client
}

// NewWebhookDeliverHandler creates a webhook delivery handler.
func NewWebhookDeliverHandler(webhooks store.WebhookStore, tokens store.TokenStore) asynq.Handler {
	return &webhookDeliverHandler{
		webhooks: webhooks,
		tokens:   tokens,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *webhookDeliverHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload WebhookPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: unmarshal webhook payload: %v", asynq.SkipRetry, err)
	}

	endpoints, err := h.webhooks.List(ctx, payload.WorkspaceID)
	if err != nil {
		return fmt.Errorf("list endpoints: %w", err)
	}

	var lastErr error
	for _, ep := range endpoints {
		if !ep.Enabled {
			continue
		}
		// check if endpoint subscribes to this event
		subscribed := false
		if len(ep.Events) == 0 { // Empty means all events? Or none? Usually none, but let's check
			// Assume if events is empty, maybe it doesn't subscribe. But wait, we'll check if event matches.
			// Actually let's assume empty means it doesn't match unless explicitly "all" or specific.
		}
		for _, ev := range ep.Events {
			if ev == payload.EventType || ev == "*" {
				subscribed = true
				break
			}
		}
		if !subscribed && len(ep.Events) > 0 {
			continue
		}

		secret, err := h.tokens.Decrypt(ctx, string(ep.SecretEnc))
		if err != nil {
			slog.Error("failed to decrypt webhook secret", "endpoint_id", ep.ID, "err", err)
			continue
		}

		event := socialpublish.WebhookEvent{
			ID:        task.ResultWriter().TaskID(),
			Type:      payload.EventType,
			CreatedAt: time.Now(),
		}
		dataBytes, _ := json.Marshal(payload.Payload)
		event.Data = dataBytes

		body, _ := json.Marshal(event)
		signature := socialpublish.SignWebhookPayload(secret, body, time.Now())

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.URL, bytes.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-SocialPublish-Signature", signature)
		req.Header.Set("X-SocialPublish-Event", payload.EventType)

		resp, err := h.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("endpoint %s returned status %d", ep.URL, resp.StatusCode)
		}
	}

	return lastErr
}
