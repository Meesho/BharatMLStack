#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Stopping OnyxDB local dev environment..."
docker compose -f "$SCRIPT_DIR/docker-compose.yml" down
echo "Done."
