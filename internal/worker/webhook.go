package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"

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
}

// NewWebhookDeliverHandler creates a webhook delivery handler.
func NewWebhookDeliverHandler(webhooks store.WebhookStore) asynq.Handler {
	return &webhookDeliverHandler{webhooks: webhooks}
}

func (h *webhookDeliverHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload WebhookPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: unmarshal webhook payload: %v", asynq.SkipRetry, err)
	}
	if err := h.webhooks.EnqueueDelivery(ctx, store.WebhookDeliveryParams{
		WorkspaceID: payload.WorkspaceID,
		EventType:   payload.EventType,
		Payload:     payload.Payload,
	}); err != nil {
		return fmt.Errorf("enqueue webhook delivery: %w", err)
	}
	return nil
}
