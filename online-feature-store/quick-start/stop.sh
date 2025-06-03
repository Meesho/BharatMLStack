#!/bin/bash

set -e

WORKSPACE_DIR="workspace"
NETWORK_NAME="orion-network"
PURGE=false

# Parse flags
if [[ "$1" == "--purge" ]]; then
  PURGE=true
fi

echo "🛑 Stopping all services..."

# Stop docker-compose services
if [ -f "$WORKSPACE_DIR/docker-compose.yml" ]; then
  echo "🐳 Stopping Docker Compose services..."
  (cd "$WORKSPACE_DIR" && docker-compose down)
else
  echo "⚠️ docker-compose.yml not found in $WORKSPACE_DIR. Skipping Docker Compose shutdown."
fi

# Stop and optionally delete containers
for container in orion-grpc-api-server horizon trufflebox; do
  if docker ps -q -f name="$container" > /dev/null; then
    echo "🛑 Stopping container: $container"
    docker stop "$container" > /dev/null || true
  fi

  if [ "$PURGE" = true ]; then
    if docker ps -a -q -f name="$container" > /dev/null; then
      echo "🗑️ Removing container: $container"
      docker rm "$container" > /dev/null || true
    fi
  fi
done

# Optionally remove volumes and network
if [ "$PURGE" = true ]; then
  echo "🧹 Removing dangling volumes..."
  docker volume prune -f

  if docker network ls | grep -q "$NETWORK_NAME"; then
    echo "🧯 Removing Docker network: $NETWORK_NAME"
    docker network rm "$NETWORK_NAME" || true
  fi

  if [ -d "$WORKSPACE_DIR" ]; then
    echo "🧽 Deleting workspace directory: $WORKSPACE_DIR"
    rm -rf "$WORKSPACE_DIR"
  fi
fi

echo "✅ All services stopped."
if [ "$PURGE" = true ]; then
  echo "🔥 All containers, volumes, network, and workspace purged."
fi
