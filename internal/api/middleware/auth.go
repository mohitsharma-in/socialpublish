package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/yourorg/socialpublish/internal/store"
	"github.com/yourorg/socialpublish/internal/tenant"
)

// Authenticate validates bearer API keys.
func Authenticate(keys store.APIKeyStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if raw == "" {
				http.Error(w, `{"code":"unauthorized","message":"API key required"}`, http.StatusUnauthorized)
				return
			}
			sum := sha256.Sum256([]byte(raw))
			key, err := keys.FindByHash(r.Context(), hex.EncodeToString(sum[:]))
			if err != nil {
				http.Error(w, `{"code":"unauthorized","message":"invalid API key"}`, http.StatusUnauthorized)
				return
			}
			_ = keys.TouchLastUsed(r.Context(), key.ID)
			ctx := tenant.WithWorkspace(r.Context(), tenant.Workspace{ID: key.WorkspaceID, Plan: "free"})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
