#!/bin/bash
set -e

GO_VERSION="1.22.3"
GO_TAR="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="https://go.dev/dl/${GO_TAR}"
GO_INSTALL_DIR="/usr/local/go"
PROFILE_FILE="$HOME/.bashrc"

# 🧠 1. Install Go if missing
if ! command -v go &> /dev/null; then
  echo "📦 Go not found. Installing Go ${GO_VERSION}..."
  wget -q $GO_URL -O /tmp/$GO_TAR
  sudo rm -rf $GO_INSTALL_DIR
  sudo tar -C /usr/local -xzf /tmp/$GO_TAR
  echo 'export PATH=$PATH:/usr/local/go/bin' >> "$PROFILE_FILE"
  export PATH=$PATH:/usr/local/go/bin
  echo "✅ Go ${GO_VERSION} installed."
else
  echo "✅ Go is already installed: $(go version)"
fi

# 🐳 2. Install Docker if missing
if ! command -v docker &> /dev/null; then
  echo "📦 Docker not found. Installing Docker..."
  sudo dnf install -y docker
  sudo systemctl start docker
  sudo systemctl enable docker
  sudo usermod -aG docker $USER
  echo "✅ Docker installed. Please run 'newgrp docker' or restart your shell to apply permissions."
else
  echo "✅ Docker is already installed: $(docker --version)"
fi

# 📁 3. Enter producer-server and init Go module
cd producer-server

if [ ! -f "go.mod" ]; then
  echo "📄 Initializing go.mod..."
  go mod init producer-server
  go get github.com/gorilla/mux
  go get github.com/redis/go-redis/v9
  go get github.com/segmentio/kafka-go
else
  echo "🔍 go.mod already exists. Skipping init."
fi

# 🔄 4. Fix Redis import paths in Go code
echo "🔧 Fixing Redis import paths in Go source..."
find . -type f -name "*.go" -exec sed -i 's|github.com/go-redis/redis/v9|github.com/redis/go-redis/v9|g' {} +
sed -i '/github.com\/go-redis\/redis\/v9/d' go.mod || true
sed -i '/github.com\/go-redis\/redis\/v9/d' go.sum || true
go get github.com/redis/go-redis/v9

echo "🔄 Running go mod tidy..."
go mod tidy
cd ..

# 🐳 5. Build and deploy with Docker Compose
echo "🐳 Building and starting Docker Compose services..."
docker compose down --remove-orphans
docker compose build
docker compose up -d

echo "🚀 Deployment complete!"
echo "🌐 Visit: http://localhost:8080"
