package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mohitsharma-in/socialpublish/internal/store"
	"github.com/mohitsharma-in/socialpublish/internal/tenant"
)

// Webhook handles webhook routes.
type Webhook struct {
	webhooks store.WebhookStore
	tokens   store.TokenStore
}

// NewWebhook creates a Webhook handler.
func NewWebhook(webhooks store.WebhookStore, tokens store.TokenStore) *Webhook {
	return &Webhook{webhooks: webhooks, tokens: tokens}
}

type webhookCreateRequest struct {
	URL     string   `json:"url"`
	Events  []string `json:"events"`
	Enabled bool     `json:"enabled"`
}

// Create creates a webhook.
func (h *Webhook) Create(w http.ResponseWriter, r *http.Request) {
	ws := tenant.MustFromContext(r.Context())

	var req webhookCreateRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "url is required")
		return
	}

	// Generate a 32-byte secret
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to generate secret")
		return
	}
	secret := hex.EncodeToString(secretBytes)

	// Hash it for verification
	hash := sha256.Sum256([]byte(secret))
	secretHash := hex.EncodeToString(hash[:])

	// Encrypt the secret using TokenStore
	tokenID, err := h.tokens.Save(r.Context(), secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to save secret")
		return
	}

	ep := &store.WebhookEndpoint{
		WorkspaceID: ws.ID,
		URL:         req.URL,
		SecretHash:  secretHash,
		SecretEnc:   []byte(tokenID),
		Events:      req.Events,
		Enabled:     req.Enabled,
	}

	if err := h.webhooks.Create(r.Context(), ep); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         ep.ID,
		"url":        ep.URL,
		"events":     ep.Events,
		"enabled":    ep.Enabled,
		"secret":     secret, // Return it once!
		"created_at": ep.CreatedAt,
	})
}

// List lists webhooks.
func (h *Webhook) List(w http.ResponseWriter, r *http.Request) {
	ws := tenant.MustFromContext(r.Context())
	endpoints, err := h.webhooks.List(r.Context(), ws.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	var out []map[string]any
	for _, ep := range endpoints {
		out = append(out, map[string]any{
			"id":         ep.ID,
			"url":        ep.URL,
			"events":     ep.Events,
			"enabled":    ep.Enabled,
			"created_at": ep.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

// Delete deletes a webhook.
func (h *Webhook) Delete(w http.ResponseWriter, r *http.Request) {
	endpointID := chi.URLParam(r, "webhookID")
	if err := h.webhooks.Delete(r.Context(), endpointID); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "webhook not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Test sends a webhook test delivery.
func (h *Webhook) Test(w http.ResponseWriter, r *http.Request) {
	ws := tenant.MustFromContext(r.Context())
	endpointID := chi.URLParam(r, "webhookID")

	// Just a simple validation to ensure endpoint belongs to this workspace
	_, err := h.webhooks.Get(r.Context(), endpointID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "webhook not found")
		return
	}

	params := store.WebhookDeliveryParams{
		WorkspaceID: ws.ID,
		EventType:   "webhook.test",
		Payload: map[string]any{
			"message":   "test webhook delivery",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	if err := h.webhooks.EnqueueDelivery(r.Context(), params); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
