#!/bin/bash

set -e

IMAGE=meeshotech/bharatmlstack-trufflebox:v0.1.6
CONTAINER_NAME=trufflebox
PORT=3000

docker rm -f "$CONTAINER_NAME"
echo "🚀 Running Trufflebox container..."
docker run -d --network "onfs-network" \
  --name "$CONTAINER_NAME" \
  -p "$PORT":80 \
  -e REACT_APP_HORIZON_BASE_URL="http://horizon:8082" \
  "$IMAGE"

echo "⏳ Waiting for service to be available..."

# Healthcheck loop
for i in {1..30}; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:$PORT || true)
  if [ "$STATUS" = "200" ]; then
    echo "✅ Health check passed"
    echo "📦 Container '$CONTAINER_NAME' is running on port $PORT"
    echo "👉 Visit http://localhost:$PORT"
    echo "👉 To see logs: docker logs -f $CONTAINER_NAME"
    exit 0
  fi
  echo "⏳ Attempt $i: Service not ready yet, waiting..."
  sleep 1
done

echo "❌ Health check failed after timeout"
docker logs "$CONTAINER_NAME"
exit 1
