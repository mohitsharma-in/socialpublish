package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"github.com/mohitsharma-in/socialpublish/internal/platform"
	"github.com/mohitsharma-in/socialpublish/internal/store"
)

// AnalyticsPayload is serialized into analytics polling jobs.
type AnalyticsPayload struct {
	WorkspaceID    string `json:"workspace_id"`
	AccountID      string `json:"account_id"`
	Platform       string `json:"platform"`
	TokenID        string `json:"token_id"`
	PostID         string `json:"post_id"`
	PlatformPostID string `json:"platform_post_id"`
}

type analyticsHandler struct {
	analytics store.AnalyticsStore
	accounts  store.AccountStore
	tokens    store.TokenStore
	adapters  platform.Registry
}

// NewAnalyticsHandler creates an analytics polling handler.
func NewAnalyticsHandler(analytics store.AnalyticsStore, accounts store.AccountStore, tokens store.TokenStore, adapters platform.Registry) asynq.Handler {
	return &analyticsHandler{analytics: analytics, accounts: accounts, tokens: tokens, adapters: adapters}
}

func (h *analyticsHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload AnalyticsPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: unmarshal analytics payload: %v", asynq.SkipRetry, err)
	}
	adapter, ok := h.adapters.Get(payload.Platform)
	if !ok {
		return fmt.Errorf("%w: no adapter for platform %s", asynq.SkipRetry, payload.Platform)
	}
	accessToken, err := h.tokens.Decrypt(ctx, payload.TokenID)
	if err != nil {
		return fmt.Errorf("decrypt analytics token: %w", err)
	}
	metrics, err := adapter.FetchMetrics(ctx, accessToken, payload.PlatformPostID)
	if err != nil {
		return fmt.Errorf("fetch metrics: %w", err)
	}
	if err := h.analytics.Record(ctx, store.AnalyticsSnapshot{
		WorkspaceID:    payload.WorkspaceID,
		AccountID:      payload.AccountID,
		PostID:         payload.PostID,
		PlatformPostID: payload.PlatformPostID,
		CollectedAt:    time.Now(),
		Metrics: map[string]any{
			"views":    metrics.Views,
			"likes":    metrics.Likes,
			"comments": metrics.Comments,
			"shares":   metrics.Shares,
			"reach":    metrics.Reach,
			"extra":    metrics.Extra,
		},
	}); err != nil {
		return fmt.Errorf("record metrics: %w", err)
	}
	return nil
}
