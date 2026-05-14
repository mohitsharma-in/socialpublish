package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements sliding-window rate limiting backed by Redis.
type RedisRateLimiter struct {
	rdb *redis.Client
}

// NewRedisRateLimiter creates a Redis-backed rate limiter.
func NewRedisRateLimiter(rdb *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{rdb: rdb}
}

// Allow checks whether a workspace has remaining request budget.
// Uses a fixed-window counter in Redis with 1-minute expiry.
func (rl *RedisRateLimiter) Allow(ctx context.Context, workspaceID string, limit int) (int, time.Time, bool) {
	key := fmt.Sprintf("ratelimit:%s", workspaceID)
	reset := time.Now().Add(time.Minute).Truncate(time.Minute)

	count, err := rl.rdb.Incr(ctx, key).Result()
	if err != nil {
		// On Redis failure, allow the request (fail-open).
		return limit, reset, true
	}
	if count == 1 {
		// First request in this window — set expiry.
		rl.rdb.Expire(ctx, key, time.Minute)
	}

	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}
	return remaining, reset, int(count) <= limit
}

// InMemoryRateLimiter is a permissive limiter for tests and dev.
type InMemoryRateLimiter struct{}

// Allow always permits the request.
func (InMemoryRateLimiter) Allow(_ context.Context, _ string, limit int) (int, time.Time, bool) {
	return limit, time.Now().Add(time.Minute).Truncate(time.Minute), true
}
