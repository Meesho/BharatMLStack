#!/bin/bash

set -e

GO_MIN_VERSION="1.22"
INSTALL_LINK="https://go.dev/doc/install"
WORKSPACE_DIR="workspace"

check_go_version() {
  if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed."
    echo "👉 Please install Go $GO_MIN_VERSION+ from: $INSTALL_LINK"
    exit 1
  fi

  GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
  if [ "$(printf '%s\n' "$GO_MIN_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$GO_MIN_VERSION" ]; then
    echo "❌ Go version $GO_VERSION is less than required $GO_MIN_VERSION"
    echo "👉 Please install Go $GO_MIN_VERSION+ from: $INSTALL_LINK"
    exit 1
  fi

  echo "✅ Go version $GO_VERSION detected"
}

setup_workspace() {
  echo "📁 Setting up workspace in ./$WORKSPACE_DIR"
  mkdir -p "$WORKSPACE_DIR"
  cp -n ./docker-compose.yml "$WORKSPACE_DIR"/ || true
  cp -n ./check_db_and_init.sh "$WORKSPACE_DIR"/ || true
  cp -n ./boot_onfs_grpc_server.sh "$WORKSPACE_DIR"/ || true
  cp -n ./boot_trufflebox.sh "$WORKSPACE_DIR"/ || true
  cp -n ./boot_horizon.sh "$WORKSPACE_DIR"/ || true
}

start_docker_services() {
  echo "🐳 Starting Docker services via docker-compose..."
  (cd "$WORKSPACE_DIR" && docker-compose up -d)
}

run_check_and_init() {
  echo "🧪 Running DB checks and init..."
  (cd "$WORKSPACE_DIR" && ./check_db_and_init.sh)
}

run_horizon() {
  echo "🚀 Booting Horizon..."
  (cd "$WORKSPACE_DIR" && ./boot_horizon.sh)
}

run_trufflebox() {
  echo "🚀 Booting Trufflebox..."
  (cd "$WORKSPACE_DIR" && ./boot_trufflebox.sh)
}

run_onfs_grpc_server() {
  echo "🚀 Booting Online Feature Store gRPC API server..."
  (cd "$WORKSPACE_DIR" && ./boot_onfs_grpc_server.sh)
}

echo "🚀 Starting setup..."

setup_workspace
start_docker_services
run_check_and_init
run_horizon
run_trufflebox
run_onfs_grpc_server

echo "🎉 All done in ./$WORKSPACE_DIR — you're ready to go!"
