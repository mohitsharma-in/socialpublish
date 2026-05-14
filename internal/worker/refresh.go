package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"

	"github.com/mohitsharma-in/socialpublish/internal/platform"
	"github.com/mohitsharma-in/socialpublish/internal/store"
)

// TokenRefreshPayload is serialized into token refresh jobs.
type TokenRefreshPayload struct {
	AccountID    string `json:"account_id"`
	Platform     string `json:"platform"`
	RefreshToken string `json:"refresh_token"`
}

type tokenRefreshHandler struct {
	accounts store.AccountStore
	tokens   store.TokenStore
	adapters platform.Registry
}

// NewTokenRefreshHandler creates a token refresh handler.
func NewTokenRefreshHandler(accounts store.AccountStore, tokens store.TokenStore, adapters platform.Registry) asynq.Handler {
	return &tokenRefreshHandler{accounts: accounts, tokens: tokens, adapters: adapters}
}

func (h *tokenRefreshHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload TokenRefreshPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: unmarshal token refresh payload: %v", asynq.SkipRetry, err)
	}
	adapter, ok := h.adapters.Get(payload.Platform)
	if !ok {
		return fmt.Errorf("%w: no adapter for platform %s", asynq.SkipRetry, payload.Platform)
	}
	token, err := adapter.RefreshToken(ctx, payload.RefreshToken)
	if err != nil {
		return fmt.Errorf("refresh token for account %s: %w", payload.AccountID, err)
	}
	tokenID, err := h.tokens.Save(ctx, token.AccessToken)
	if err != nil {
		return fmt.Errorf("save refreshed token for account %s: %w", payload.AccountID, err)
	}
	if err := h.accounts.UpdateToken(ctx, payload.AccountID, tokenID, token.ExpiresAt); err != nil {
		return fmt.Errorf("update account token %s: %w", payload.AccountID, err)
	}
	return nil
}
