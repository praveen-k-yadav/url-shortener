package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"

	"url-shortener/db"
	"url-shortener/middleware"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const base62Chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// generateShortURL creates an MD5 hash of the URL (with optional salt),
// converts it into a big integer, and Base62 encodes it (max 8 characters).
func generateShortURL(originalURL string, salt string) string {
	hasher := md5.New()
	hasher.Write([]byte(originalURL + salt))
	hashBytes := hasher.Sum(nil)

	num := new(big.Int).SetBytes(hashBytes)
	base := big.NewInt(62)
	zero := big.NewInt(0)
	mod := &big.Int{}
	var encoded []byte

	for num.Cmp(zero) > 0 {
		num.DivMod(num, base, mod)
		encoded = append([]byte{base62Chars[mod.Int64()]}, encoded...)
	}

	if len(encoded) == 0 {
		return string(base62Chars[0])
	}
	if len(encoded) > 8 {
		return string(encoded[:8])
	}
	return string(encoded)
}

func RootPageUrlhandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}

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
			respondJSON(w, existing.ShortURL)
			return
		}

		// 2. Retry loop to handle rare short_url hash collisions and
		// race conditions from concurrent duplicate submissions
		var shortURL string
		maxRetries := 5
		salt := ""

		for i := 0; i < maxRetries; i++ {
			shortURL = generateShortURL(data.URL, salt)

			err = db.InsertURL(ctx, collection, data.URL, shortURL)
			if err == nil {
				break
			}

			// Handle duplicate key conflicts (code 11000)
			if mongo.IsDuplicateKeyError(err) {
				existingRecord, fetchErr := db.GetURLByOriginal(ctx, collection, data.URL)
				if fetchErr == nil {
					respondJSON(w, existingRecord.ShortURL)
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

		respondJSON(w, shortURL)
	}
}

// respondJSON writes the short_url JSON payload cleanly
func respondJSON(w http.ResponseWriter, shortURL string) {
	response := struct {
		ShortURL string `json:"short_url"`
	}{ShortURL: shortURL}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

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

func main() {
	fmt.Println("Starting URL Shortener Project...")

	collection, cleanup, err := db.ConnectDB()
	if err != nil {
		log.Fatalf("Failed to initialise DB: %v", err)
	}
	defer cleanup()

	// Initialize Redis Client
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // Set password if your Redis instance requires one
		DB:       0,  // Default DB
	})

	// Ping Redis to verify connection on startup
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis successfully")

	// Ensure unique indexes exist on startup
	indexCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.EnsureIndexes(indexCtx, collection); err != nil {
		log.Fatalf("Failed to create database indexes: %v", err)
	}

	log.Printf("Successfully got collection handle: %s", collection.Name())

	// Register HTTP routes, passing both MongoDB collection and Redis client
	http.HandleFunc("/", RootPageUrlhandler)

	limiter := middleware.RateLimiter(rdb, 10, 1*time.Minute)
	http.HandleFunc("/shorten", limiter(ShortURLHandler(collection, rdb)))
	http.HandleFunc("/redirect/", RedirectURLHandler(collection, rdb))

	fmt.Println("Starting the server at port 3000...")
	err = http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatalf("Error starting the server: %v", err)
	}
}
