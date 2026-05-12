package handler

import (
	"net/http"

	"github.com/yourorg/socialpublish/internal/store"
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

// Create creates a webhook.
func (h *Webhook) Create(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// List lists webhooks.
func (h *Webhook) List(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Delete deletes a webhook.
func (h *Webhook) Delete(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Test sends a webhook test delivery.
func (h *Webhook) Test(w http.ResponseWriter, r *http.Request) { notImplemented(w) }
