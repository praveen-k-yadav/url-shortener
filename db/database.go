package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ConnectDB initializes a MongoDB client with connection pool settings,
// pings the server, and returns the collection and cleanup function.
func ConnectDB() (*mongo.Collection, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Replace with your MongoDB URI or Atlas connection string
	uri := "mongodb://localhost:27017"

	// Configure connection pool & timeouts
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(100).                       // Max concurrent connections in pool
		SetMinPoolSize(10).                        // Keep 10 warm connections ready
		SetMaxConnIdleTime(30 * time.Second).      // Close idle connections after 30s
		SetConnectTimeout(5 * time.Second).        // Timeout for establishing new TCP connections
		SetServerSelectionTimeout(5 * time.Second) // Timeout waiting for a free connection from the pool

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create mongo client: %w", err)
	}

	// Verify the connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	fmt.Println("Successfully connected to MongoDB with connection pool configured!")

	// Select database and collection (preserving your original DB name)
	database := client.Database("url_shortener_db")
	collection := database.Collection("urls")

	// Define cleanup function to handle graceful shutdown
	cleanup := func() {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer disconnectCancel()

		if err := client.Disconnect(disconnectCtx); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		} else {
			fmt.Println("Disconnected from MongoDB successfully.")
		}
	}

	return collection, cleanup, nil
}
