package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/mohitsharma-in/socialpublish/internal/store"
	"github.com/mohitsharma-in/socialpublish/internal/tenant"
)

// Account handles account routes.
type Account struct {
	accounts store.AccountStore
	tokens   store.TokenStore
}

// NewAccount creates an Account handler.
func NewAccount(accounts store.AccountStore, tokens store.TokenStore) *Account {
	return &Account{accounts: accounts, tokens: tokens}
}

// List lists accounts.
func (h *Account) List(w http.ResponseWriter, r *http.Request) {
	ws := tenant.MustFromContext(r.Context())
	accounts, err := h.accounts.List(r.Context(), ws.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": accounts})
}

type connectRequest struct {
	Platform       string `json:"platform"`
	PlatformUserID string `json:"platform_user_id"`
	DisplayName    string `json:"display_name"`
	AvatarURL      string `json:"avatar_url"`
	AccessToken    string `json:"access_token"`
}

// Connect connects an account.
func (h *Account) Connect(w http.ResponseWriter, r *http.Request) {
	ws := tenant.MustFromContext(r.Context())
	var req connectRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}
	if req.Platform == "" || req.AccessToken == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "platform and access_token required")
		return
	}
	tokenID, err := h.tokens.Save(r.Context(), req.AccessToken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	account := &store.Account{
		Platform: req.Platform, PlatformUserID: req.PlatformUserID,
		DisplayName: req.DisplayName, AvatarURL: req.AvatarURL, TokenID: tokenID,
	}
	if err := h.accounts.Create(r.Context(), ws.ID, account); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, account)
}

// Get fetches an account.
func (h *Account) Get(w http.ResponseWriter, r *http.Request) {
	account, err := h.accounts.Get(r.Context(), chi.URLParam(r, "accountID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "account not found")
		return
	}
	writeJSON(w, http.StatusOK, account)
}

// Delete deletes an account.
func (h *Account) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.accounts.Delete(r.Context(), chi.URLParam(r, "accountID")); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "account not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Status fetches account token health status.
func (h *Account) Status(w http.ResponseWriter, r *http.Request) {
	account, err := h.accounts.Get(r.Context(), chi.URLParam(r, "accountID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "account not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"account_id":    account.ID,
		"token_healthy": account.TokenHealthy,
		"expires_at":    account.TokenExpiresAt,
	})
}
