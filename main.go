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
// and concurrent duplicate submission safety.
func ShortURLHandler(collection *mongo.Collection) http.HandlerFunc {
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

		// 1. Check if the URL has already been shortened
		existing, err := db.GetURLByOriginal(ctx, collection, data.URL)
		if err == nil {
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
				// Case A: Another concurrent request just inserted the exact same original_url
				existingRecord, fetchErr := db.GetURLByOriginal(ctx, collection, data.URL)
				if fetchErr == nil {
					respondJSON(w, existingRecord.ShortURL)
					return
				}

				// Case B: Different original_url produced the same short_url (hash collision).
				// Append a nanosecond salt and retry.
				salt = fmt.Sprintf("_%d", time.Now().UnixNano())
				if i == maxRetries-1 {
					http.Error(w, "Failed to generate a unique short URL", http.StatusInternalServerError)
					return
				}
				continue
			}

			// Any other database error (e.g., network failure, timeout)
			http.Error(w, "Failed to save URL to database", http.StatusInternalServerError)
			return
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

// RedirectURLHandler fetches the original URL from MongoDB and redirects the user
func RedirectURLHandler(collection *mongo.Collection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/redirect/"):]
		if id == "" {
			http.Error(w, "Invalid Request", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		urlRecord, err := db.GetURLByShortCode(ctx, collection, id)
		if err != nil {
			http.Error(w, "URL not found", http.StatusNotFound)
			return
		}

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

	// Ensure unique indexes exist on startup
	indexCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.EnsureIndexes(indexCtx, collection); err != nil {
		log.Fatalf("Failed to create database indexes: %v", err)
	}

	log.Printf("Successfully got collection handle: %s", collection.Name())

	// Register HTTP routes
	http.HandleFunc("/", RootPageUrlhandler)
	http.HandleFunc("/shorten", ShortURLHandler(collection))
	http.HandleFunc("/redirect/", RedirectURLHandler(collection))

	fmt.Println("Starting the server at port 3000...")
	err = http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatalf("Error starting the server: %v", err)
	}
}
