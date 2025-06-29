# Stage 1: Build the Go binary
FROM golang:1.21-alpine AS builder

# Set working directory inside container
WORKDIR /app

# Install necessary packages
RUN apk update && apk add --no-cache git

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
RUN go build -o consumer-server

# Stage 2: Create a small runtime image
FROM alpine:latest

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/consumer-server .

# Set environment variables (can also be done via docker-compose)
ENV PORT=5001 \
    REDIS_HOST=redis:6379 \
    KAFKA_BROKER=kafka:9092

# Expose the service port
EXPOSE 5001

# Run the binary
CMD ["./consumer-server"]
