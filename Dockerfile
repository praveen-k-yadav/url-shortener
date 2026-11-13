# --- Stage 1: Build the binary ---
FROM golang:1.27.3-alpine AS builder

WORKDIR /app

# Copy dependency files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build a statically linked Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# --- Stage 2: Create the minimal production image ---
FROM alpine:latest

# Install CA certificates for outbound HTTPS/TLS calls if needed
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy compiled binary from the builder stage
COPY --from=builder /app/main .

# Expose port 3000 to match your server
EXPOSE 3000

# Command to run the application
CMD ["./main"]