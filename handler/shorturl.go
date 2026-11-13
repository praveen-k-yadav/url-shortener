package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"url-shortener/db"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ShortURLHandler handles URL creation, idempotency, hash collision retries,
// and pre-populates the Redis cache upon successful insert.
func ShortURLHandler(collection *mongo.Collection, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var data struct {
			URL string `json:"url"`
		}
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil || data.URL == "" {
			http.Error(w, "Invalid Request Body", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// 1. Check if the URL has already been shortened (MongoDB check for idempotency)
		existing, err := db.GetURLByOriginal(ctx, collection, data.URL)
		if err == nil {
			// Ensure it's cached in Redis as well
			rdb.Set(ctx, "url:"+existing.ShortURL, existing.OriginalURL, 30*24*time.Hour)
			fmt.Println("Data coming from Redis...")
			RespondJSON(w, existing.ShortURL)
			return
		}

		// 2. Retry loop to handle rare short_url hash collisions and
		// race conditions from concurrent duplicate submissions
		var shortURL string
		maxRetries := 5
		salt := ""

		for i := 0; i < maxRetries; i++ {
			shortURL = GenerateShortURL(data.URL, salt)

			err = db.InsertURL(ctx, collection, data.URL, shortURL)
			if err == nil {
				break
			}

			// Handle duplicate key conflicts (code 11000)
			if mongo.IsDuplicateKeyError(err) {
				existingRecord, fetchErr := db.GetURLByOriginal(ctx, collection, data.URL)
				if fetchErr == nil {
					RespondJSON(w, existingRecord.ShortURL)
					return
				}

				salt = fmt.Sprintf("_%d", time.Now().UnixNano())
				if i == maxRetries-1 {
					http.Error(w, "Failed to generate a unique short URL", http.StatusInternalServerError)
					return
				}
				continue
			}

			http.Error(w, "Failed to save URL to database", http.StatusInternalServerError)
			return
		}

		// 3. Pre-populate Redis cache with a 30-day TTL upon successful creation
		cacheKey := "url:" + shortURL
		if err := rdb.Set(ctx, cacheKey, data.URL, 30*24*time.Hour).Err(); err != nil {
			log.Printf("Warning: Failed to set Redis cache: %v", err)
		}

		RespondJSON(w, shortURL)
	}
}
