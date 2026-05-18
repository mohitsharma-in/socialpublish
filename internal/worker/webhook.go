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

	"github.com/mohitsharma-in/socialpublish/internal/store"
	socialpublish "github.com/mohitsharma-in/socialpublish/sdk"
)

// WebhookPayload is serialized into webhook delivery jobs.
type WebhookPayload struct {
	WorkspaceID string `json:"workspace_id"`
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

		deliveries, err := h.webhooks.ListPendingDeliveries(ctx, ep.ID, 50)
		if err != nil {
			slog.Error("failed to list pending deliveries", "endpoint_id", ep.ID, "err", err)
			continue
		}
		if len(deliveries) == 0 {
			continue
		}

		secret, err := h.tokens.Decrypt(ctx, string(ep.SecretEnc))
		if err != nil {
			slog.Error("failed to decrypt webhook secret", "endpoint_id", ep.ID, "err", err)
			continue
		}

		for _, delivery := range deliveries {
			event := socialpublish.WebhookEvent{
				ID:        delivery.ID,
				Type:      delivery.EventType,
				CreatedAt: time.Now(),
			}
			dataBytes, _ := json.Marshal(delivery.Payload)
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
			req.Header.Set("X-SocialPublish-Event", delivery.EventType)

			resp, err := h.client.Do(req)
			if err != nil {
				lastErr = err
				continue
			}
			
			status := resp.StatusCode
			resp.Body.Close()

			if status < 200 || status >= 300 {
				lastErr = fmt.Errorf("endpoint %s returned status %d", ep.URL, status)
			}

			if err := h.webhooks.MarkDelivered(ctx, delivery.ID, status); err != nil {
				slog.Error("failed to mark webhook delivered", "delivery_id", delivery.ID, "err", err)
			}
		}
	}

	return lastErr
}
