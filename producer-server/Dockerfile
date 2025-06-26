# Use official Golang image as the builder
FROM golang:1.22 as builder

# Set working directory inside container
WORKDIR /app

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# Build the Go binary
RUN go build -o producer-server

# Start a new lightweight image
FROM debian:bullseye-slim

# Install SSL certs for HTTPS communication
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/producer-server .
COPY --from=builder /app/public ./public

# Expose the port the app runs on
EXPOSE 3000

# Run the binary
CMD ["./producer-server"]
