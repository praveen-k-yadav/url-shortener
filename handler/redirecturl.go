package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
	"url-shortener/db"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RedirectURLHandler implements the Cache-Aside pattern (Redis -> MongoDB fallback -> Backfill)
func RedirectURLHandler(collection *mongo.Collection, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/redirect/"):]
		if id == "" {
			http.Error(w, "Invalid Request", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		cacheKey := "url:" + id

		// Step A: Check Redis Cache first (O(1) lookup)
		cachedURL, err := rdb.Get(ctx, cacheKey).Result()
		if err == nil {
			// Cache Hit! Redirect immediately without touching MongoDB
			fmt.Println("Data coming from Redis-Redirect Url")
			http.Redirect(w, r, cachedURL, http.StatusFound)
			return
		} else if err != redis.Nil {
			// Log unexpected Redis errors (other than standard cache miss)
			log.Printf("Redis error: %v", err)
		}

		// Step B: Cache Miss - Query MongoDB source of truth
		urlRecord, err := db.GetURLByShortCode(ctx, collection, id)
		if err != nil {
			http.Error(w, "URL not found", http.StatusNotFound)
			return
		}

		// Step C: Backfill Redis Cache for subsequent requests (with 30-day TTL)
		if err := rdb.Set(ctx, cacheKey, urlRecord.OriginalURL, 30*24*time.Hour).Err(); err != nil {
			log.Printf("Warning: Failed to backfill Redis cache: %v", err)
		}

		// Step D: Redirect user
		http.Redirect(w, r, urlRecord.OriginalURL, http.StatusFound)
	}
}
