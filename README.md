# URL Shortener

A lightweight, high-performance URL shortener service built with Go. This project provides fast URL shortening, seamless redirection, persistent storage, built-in rate limiting, and Docker support for easy deployment.

## Features

* **Fast URL Shortening:** Generate compact, shareable short links from long URLs.
* **Instant Redirection:** Low-latency HTTP redirects from short codes to original destination URLs.
* **Rate Limiting Middleware:** Protects endpoints from abuse and spam via custom request rate limiting (`middleware/ratelimitter.go`).
* **Clean Architecture:** Modular separation between HTTP handlers, database repositories, and middleware.
* **Containerized:** Ready-to-deploy Dockerfile included for consistent production builds.

## Tech Stack

* **Language:** Go (Golang)
* **Database & Caching:** MongoDB & Redis
* **Containerization:** Docker

## Project Structure

```text
url-shortener/
├── db/
│   ├── database.go      # Database connection and initialization
│   └── repository.go    # Data access methods for storing and retrieving URLs
├── handler/
│   ├── helper.go        # Utility functions for handlers and short-code generation
│   ├── redirecturl.go   # Handles redirection from short URL to original URL
│   ├── rootpageurl.go   # Serves the root/landing endpoint
│   └── shorturl.go      # Handles incoming requests to create shortened URLs
├── middleware/
│   └── ratelimitter.go  # HTTP middleware for request rate limiting
├── .dockerignore        # Files excluded from Docker builds
├── Dockerfile           # Multi-stage or production Docker build instructions
├── go.mod               # Go module dependencies
├── go.sum               # Go module checksums
└── main.go              # Application entry point, server setup, and routing
```

## Getting Started

### Prerequisites

* **Go** (Version 1.21+)
* **MongoDB** (Installed and running natively or via Docker)
* **Redis** (For caching and rate-limiting layers)
* **Docker** (Optional, for containerized execution)

### Local Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/url-shortener.git
   cd url-shortener
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Run the application:
   ```bash
   go run main.go
   ```

### Running with Docker

1. Build the Docker image:
   ```bash
   docker build -t url-shortener .
   ```

2. Run the container:
   ```bash
   docker run -p 8080:8080 --env-file .env url-shortener
   ```

## API Usage

### 1. Create a Short URL

* **Endpoint:** `POST /shorten`
* **Content-Type:** `application/json`

**Request Body:**
```json
{
  "url": "https://github.com/"
}
```

**Response:**
```json
{
  "shortCode": "abc1234"
}
```

### 2. Redirect to Original URL

* **Endpoint:** `GET /{shortCode}`
* **Description:** Automatically redirects the browser or HTTP client to the original long URL destination.

```bash
curl -I http://localhost:8080/abc1234
```