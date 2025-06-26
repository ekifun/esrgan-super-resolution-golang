#!/bin/bash

set -e

# Deployment script for ESRGAN Super Resolution Golang system
# Includes building Go modules, Docker images, and starting services

echo "🔄 Syncing Go dependencies..."
cd producer-server
go mod tidy

echo "✅ Go modules synced."

echo "🐳 Building Docker images and starting containers..."
cd ..
docker compose down --remove-orphans
docker compose build
docker compose up -d

echo "🚀 Deployment complete."
echo "🌐 Access the dashboard at: http://localhost:8080"
