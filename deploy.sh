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

# 📁 2. Enter producer-server
cd producer-server

# 🛠️ 3. Auto-init go.mod if missing
if [ ! -f "go.mod" ]; then
  echo "📄 Initializing go.mod..."
  go mod init producer-server
  go get github.com/gorilla/mux
  go get github.com/redis/go-redis/v9
  go get github.com/segmentio/kafka-go
else
  echo "🔍 go.mod already exists. Skipping init."
fi

# 🔄 4. Replace bad Redis import paths in source files
echo "🔧 Fixing Redis import paths in Go source..."
find . -type f -name "*.go" -exec sed -i 's|github.com/go-redis/redis/v9|github.com/redis/go-redis/v9|g' {} +

# 🧹 5. Clean up bad go.mod line (optional safety)
sed -i '/github.com\/go-redis\/redis\/v9/d' go.mod
sed -i '/github.com\/go-redis\/redis\/v9/d' go.sum || true

# 📦 6. Add correct Redis module
go get github.com/redis/go-redis/v9

# 🔄 7. Finalize dependencies
echo "🔄 Running go mod tidy..."
go mod tidy

cd ..

# 🐳 8. Docker Compose deployment
echo "🐳 Rebuilding and starting Docker Compose..."
docker compose down --remove-orphans
docker compose build
docker compose up -d

echo "🚀 Deployment complete!"
echo "🌐 Visit: http://localhost:8080"
