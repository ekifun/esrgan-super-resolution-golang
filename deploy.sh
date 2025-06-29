#!/bin/bash
set -e

GO_VERSION="1.22.3"
GO_TAR="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="https://go.dev/dl/${GO_TAR}"
GO_INSTALL_DIR="/usr/local/go"
PROFILE_FILE="$HOME/.bashrc"
COMPOSE_VERSION="v2.27.1"
COMPOSE_BIN="$HOME/.docker/cli-plugins/docker-compose"

# 1. Install Go if missing
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

# 2. Install Docker if missing
if ! command -v docker &> /dev/null; then
  echo "🐳 Docker not found. Installing Docker..."
  sudo dnf install -y docker
  sudo systemctl start docker
  sudo systemctl enable docker
  sudo usermod -aG docker $USER
  echo "✅ Docker installed. Please run 'newgrp docker' or restart your shell."
else
  echo "✅ Docker is already installed: $(docker --version)"
fi

# 3. Install Docker Compose v2 as plugin if missing
if ! docker compose version &> /dev/null; then
  echo "🔧 Docker Compose v2 not found. Installing ${COMPOSE_VERSION}..."
  mkdir -p ~/.docker/cli-plugins
  curl -SL https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-linux-x86_64 -o "$COMPOSE_BIN"
  chmod +x "$COMPOSE_BIN"
  echo "✅ Docker Compose v2 installed: $($COMPOSE_BIN version)"
else
  echo "✅ Docker Compose is already installed: $(docker compose version)"
fi

# 4. Fix Redis import paths and setup Go modules for both services
fix_redis_imports() {
  local service_dir=$1
  echo "🔧 Setting up Go modules in $service_dir..."
  cd "$service_dir"

  if [ ! -f "go.mod" ]; then
    echo "📄 Initializing go.mod for $service_dir..."
    go mod init "$service_dir"
  fi

  go get github.com/gorilla/mux || true
  go get github.com/segmentio/kafka-go || true
  go get github.com/redis/go-redis/v9 || true

  # Replace old Redis path and clean up go.mod/go.sum
  find . -type f -name "*.go" -exec sed -i 's|github.com/go-redis/redis/v9|github.com/redis/go-redis/v9|g' {} +
  sed -i '/github.com\/go-redis\/redis\/v9/d' go.mod || true
  sed -i '/github.com\/go-redis\/redis\/v9/d' go.sum || true

  go mod tidy
  cd ..
}

fix_redis_imports "producer-server"
fix_redis_imports "consumer-server"

# 5. Build and deploy all services
echo "🚢 Building and deploying Docker Compose services..."
docker compose down --remove-orphans
docker compose build
docker compose up -d

echo "🚀 Deployment complete!"
echo "🌐 Access the app at http://localhost:8080"
