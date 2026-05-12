package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/mohitsharma-in/socialpublish/internal/platform"
	"github.com/mohitsharma-in/socialpublish/internal/store"
)

// PublishPayload is serialized into the job queue.
type PublishPayload struct {
	PostID   string `json:"post_id"`
	TargetID string `json:"target_id"`
}

type publishHandler struct {
	posts    store.PostStore
	accounts store.AccountStore
	tokens   store.TokenStore
	adapters platform.Registry
	webhooks store.WebhookStore
}

// NewPublishHandler creates a publish task handler.
func NewPublishHandler(posts store.PostStore, accounts store.AccountStore, tokens store.TokenStore, adapters platform.Registry, webhooks store.WebhookStore) asynq.Handler {
	return &publishHandler{posts: posts, accounts: accounts, tokens: tokens, adapters: adapters, webhooks: webhooks}
}

func (h *publishHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload PublishPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: unmarshal publish payload: %v", asynq.SkipRetry, err)
	}

	target, err := h.posts.GetTarget(ctx, payload.TargetID)
	if err != nil {
		return fmt.Errorf("get target %s: %w", payload.TargetID, err)
	}
	account, err := h.accounts.Get(ctx, target.AccountID)
	if err != nil {
		return fmt.Errorf("get account %s: %w", target.AccountID, err)
	}
	accessToken, err := h.tokens.Decrypt(ctx, account.TokenID)
	if err != nil {
		return fmt.Errorf("decrypt token for account %s: %w", account.ID, err)
	}
	adapter, ok := h.adapters.Get(target.Platform)
	if !ok {
		return fmt.Errorf("%w: no adapter for platform %s", asynq.SkipRetry, target.Platform)
	}
	result, err := adapter.Publish(ctx, &platform.PublishRequest{
		AccountID:   account.ID,
		AccessToken: accessToken,
		Target:      target,
	})
	if err != nil {
		if markErr := h.posts.SetTargetFailed(ctx, payload.TargetID, err.Error()); markErr != nil {
			slog.Error("failed to record publish failure", "target_id", payload.TargetID, "err", markErr)
		}
		return fmt.Errorf("adapter publish: %w", err)
	}
	if err := h.posts.SetTargetPublished(ctx, payload.TargetID, result.PlatformPostID, result.Permalink); err != nil {
		return fmt.Errorf("%w: persist publish result: %v", asynq.SkipRetry, err)
	}
	if err := h.webhooks.EnqueueDelivery(ctx, store.WebhookDeliveryParams{
		WorkspaceID: account.WorkspaceID,
		EventType:   "post.published",
		Payload: map[string]any{
			"post_id":          payload.PostID,
			"target_id":        payload.TargetID,
			"platform":         target.Platform,
			"platform_post_id": result.PlatformPostID,
			"permalink":        result.Permalink,
		},
	}); err != nil {
		slog.Error("failed to enqueue publish webhook", "target_id", payload.TargetID, "err", err)
	}
	return nil
}
