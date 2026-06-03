#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Starting mNemo local dev environment..."
docker compose -f "$SCRIPT_DIR/docker-compose.yml" up -d

echo ""
echo "Services:"
echo "  etcd  → localhost:2379"
echo ""
echo "Run './stop.sh' or 'make dev-stop' to tear down."
