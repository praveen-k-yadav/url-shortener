package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter limit requests per IP address using Redis
func RateLimiter(rdb *redis.Client, limit int, window time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := context.Background()

			// Get client IP address
			clientIP := r.RemoteAddr

			// Construct a unique Redis key for this IP
			key := "rate_limit:" + clientIP

			// Increment the counter for this IP in Redis atomically
			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				// Fail open: if Redis has an issue, allow the request so the app doesn't crash
				next.ServeHTTP(w, r)
				return
			}

			// If this is the first request in the current window, set an expiration timer
			if count == 1 {
				rdb.Expire(ctx, key, window)
			}

			// Check if the user has exceeded their request limit
			if count > int64(limit) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests) // HTTP 429 Too Many Requests
				w.Write([]byte(`{"error": "Rate limit exceeded. Please try again later."}`))
				return
			}

			// If under the limit, pass the request along to the actual endpoint handler
			next.ServeHTTP(w, r)
		}
	}
}
