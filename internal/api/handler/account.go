package handler

import (
	"net/http"

	"github.com/yourorg/socialpublish/internal/store"
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
func (h *Account) List(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Connect connects an account.
func (h *Account) Connect(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Get fetches an account.
func (h *Account) Get(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Delete deletes an account.
func (h *Account) Delete(w http.ResponseWriter, r *http.Request) { notImplemented(w) }

// Status fetches account status.
func (h *Account) Status(w http.ResponseWriter, r *http.Request) { notImplemented(w) }
