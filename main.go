package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"url-shortener/db"
	"url-shortener/handler"
	"url-shortener/middleware"

	"github.com/redis/go-redis/v9"
)

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
	http.HandleFunc("/", handler.RootPageUrlhandler)

	limiter := middleware.RateLimiter(rdb, 10, 1*time.Minute)
	http.HandleFunc("/shorten", limiter(handler.ShortURLHandler(collection, rdb)))
	http.HandleFunc("/redirect/", handler.RedirectURLHandler(collection, rdb))

	fmt.Println("Starting the server at port 3000...")
	err = http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatalf("Error starting the server: %v", err)
	}
}
