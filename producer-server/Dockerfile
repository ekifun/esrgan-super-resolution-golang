# Use debian:bullseye base with Go installed manually to match runtime glibc
FROM debian:bullseye as builder

# Install Go 1.22.3 manually
ENV GO_VERSION=1.22.3
RUN apt-get update && apt-get install -y wget git ca-certificates build-essential \
    && wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz \
    && rm go${GO_VERSION}.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:$PATH"

# Set working directory
WORKDIR /app

# Copy Go files and build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o producer-server

# Final image: debian:bullseye-slim (same glibc as builder)
FROM debian:bullseye-slim

# Install SSL certs
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/producer-server .
COPY --from=builder /app/public ./public

EXPOSE 3000
CMD ["./producer-server"]
