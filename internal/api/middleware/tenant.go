package middleware

import (
	"net/http"

	"github.com/mohitsharma-in/socialpublish/internal/store"
	"github.com/mohitsharma-in/socialpublish/internal/tenant"
)

// InjectTenant enriches context with workspace details.
func InjectTenant(workspaces store.WorkspaceStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ws, ok := tenant.FromContext(r.Context())
			if !ok {
				http.Error(w, `{"code":"unauthorized","message":"workspace missing"}`, http.StatusUnauthorized)
				return
			}
			record, err := workspaces.Get(r.Context(), ws.ID)
			if err == nil {
				ws.Plan = record.Plan
			}
			next.ServeHTTP(w, r.WithContext(tenant.WithWorkspace(r.Context(), ws)))
		})
	}
}
