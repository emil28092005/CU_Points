package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// SpendRateLimit returns middleware that limits requests to maxPerMinute per
// authenticated partner (identified by user_id in context). It must be placed
// after Auth + RequireRole("partner") so that UserIDFromContext is populated.
//
// Implementation: Redis INCR + EXPIRE sliding-window counter.
// Key: rate_limit:spend:<partner_id> — expires after 1 minute.
// On Redis failure the middleware fails open (lets the request through) so that
// a Redis outage does not take down point transactions.
func SpendRateLimit(rdb *redis.Client, maxPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			partnerID := UserIDFromContext(r.Context())
			key := "rate_limit:spend:" + partnerID

			// Use a short-lived context for Redis so a slow Redis doesn't stall the request.
			rCtx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
			defer cancel()

			count, err := rdb.Incr(rCtx, key).Result()
			if err != nil {
				// Fail open: Redis unavailable should not block transactions.
				next.ServeHTTP(w, r)
				return
			}
			// Set the expiry only on the first increment so the window resets each minute.
			if count == 1 {
				rdb.Expire(rCtx, key, time.Minute) //nolint:errcheck
			}
			if count > int64(maxPerMinute) {
				http.Error(w, `{"error":"rate limit exceeded, max 10 requests per minute"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
