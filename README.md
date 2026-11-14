URL Shortener

A lightweight, high-performance URL shortener service built with Go. This project provides fast URL shortening, seamless redirection, persistent storage, built-in rate limiting, and Docker support for easy deployment.

Features

->Fast URL Shortening: Generate compact, shareable short links fThirom long URLs.

->Instant Redirection: Low-latency HTTP redirects from short codes to original destination URLs.

->Rate Limiting Middleware: Protects endpoints from abuse and spam via custom request rate limiting (middleware/ratelimitter.go).

->Clean Architecture: Modular separation between HTTP handlers, database repositories, and middleware.

->Containerized: Ready-to-deploy Dockerfile included for consistent production builds.

Tech Stack

->Language: Go (Golang)

->Database Layer: Custom repository pattern (db/database.go, db/repository.go)

->Containerization: Docker

Project Structure

url-shortener-main/
├── db/
│   ├── database.go    # Database connection and initialise
│   └── repository.go  # Data access methods for storing
retrieving URLs
├── handler/
│   ├── helper.go      # Utility functions for handlers and short-code generation
│   ├── redirecturl.go     # Handles redirection from short URL to original URL
│   ├── rootpageurl.go     # Serves the root/landing endpoint
│   └── shorturl.go        # Handles incoming requests to create shortened URLs
├── middleware/
│   └── ratelimitter.go    # HTTP middleware for request rate limiting
├── .dockerignore          # Files excluded from Docker builds
├── Dockerfile             # Multi-stage or production Docker build instructions
├── go.mod                 # Go module dependencies
├── go.sum                 # Go module checksums
└── main.go                # Application entry point, server setup, and routing

Getting Started

Prerequisites

->Go (Version 1.20+ / built with Go 1.27.3)

->Docker (To run the Redis caching and rate-limiting layer)

->MongoDB (Installed and running natively on your local machine)

Local Installation

Clone the repository:

->git clone https://github.com/your-username/url-shortener.git
->cd url-shortener



Install dependencies:

->go mod download


Run the application:

->go run main.go



Running with Docker

Build the Docker image:

->docker build -t url-shortener .


Run the container:

->docker run -p 8080:8080 --env-file .env url-shortener



API Usage

1. Create a Short URL

postman request 'http://localhost:3000/shorten' \
  --header 'Content-Type: application/json' \
  --body '{
    "url":"https://github.com/"
}'

Response:-



2. Redirect to Original URL

Endpoint: GET /{shortCode}
