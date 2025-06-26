#!/bin/bash

set -e

GO_VERSION="1.22.3"
GO_TAR="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="https://go.dev/dl/${GO_TAR}"
GO_INSTALL_DIR="/usr/local/go"
PROFILE_FILE="$HOME/.bashrc"

# Check if Go is installed
if ! command -v go &> /dev/null; then
  echo "📦 Go not found. Installing Go ${GO_VERSION}..."
  
  wget -q $GO_URL -O /tmp/$GO_TAR
  sudo rm -rf $GO_INSTALL_DIR
  sudo tar -C /usr/local -xzf /tmp/$GO_TAR

  # Ensure PATH includes Go
  if ! grep -q "/usr/local/go/bin" "$PROFILE_FILE"; then
    echo 'export PATH=$PATH:/usr/local/go/bin' >> "$PROFILE_FILE"
    source "$PROFILE_FILE"
  fi

  export PATH=$PATH:/usr/local/go/bin
  echo "✅ Go ${GO_VERSION} installed."
else
  echo "✅ Go is already installed: $(go version)"
fi

echo "🔄 Syncing Go dependencies..."
cd producer-server
go mod tidy
cd ..

echo "🐳 Rebuilding and starting Docker Compose services..."
docker compose down --remove-orphans
docker compose build
docker compose up -d

echo "🚀 Deployment complete."
echo "🌐 Access the dashboard at: http://localhost:8080"
