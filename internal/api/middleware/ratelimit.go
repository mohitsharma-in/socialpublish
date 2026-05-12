package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/mohitsharma-in/socialpublish/internal/tenant"
)

// RateLimiter enforces per-workspace request budgets.
type RateLimiter interface {
	Allow(ctx context.Context, workspaceID string, limit int) (remaining int, reset time.Time, ok bool)
}

// PlanLimits maps plans to requests per minute.
var PlanLimits = map[string]int{
	"free":       60,
	"starter":    300,
	"pro":        1000,
	"enterprise": 5000,
}

// RateLimit returns per-workspace rate limiting middleware.
func RateLimit(limiter RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ws := tenant.MustFromContext(r.Context())
			limit := PlanLimits[ws.Plan]
			if limit == 0 {
				limit = PlanLimits["free"]
			}
			remaining, reset, ok := limiter.Allow(r.Context(), ws.ID, limit)
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
			if !ok {
				w.Header().Set("Retry-After", strconv.FormatInt(time.Until(reset).Milliseconds()/1000+1, 10))
				http.Error(w, `{"code":"rate_limit","message":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
