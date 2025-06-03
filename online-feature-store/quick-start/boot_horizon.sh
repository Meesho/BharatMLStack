#!/bin/bash

set -e

IMAGE=meeshotech/bharatmlstack-horizon:v0.1.19
CONTAINER_NAME=horizon
PORT=8082

docker rm -f "$CONTAINER_NAME"

echo "🚀 Running container..."
docker run -d  --network "onfs-network" \
  --name "$CONTAINER_NAME" \
  -p "$PORT:$PORT" \
  \
  -e APP_NAME=horizon \
  -e APP_ENVIRONMENT=PROD \
  -e APP_ENV=production \
  -e APP_PORT="$PORT" \
  -e APP_LOG_LEVEL=DEBUG \
  -e APP_METRIC_SAMPLING_RATE=1 \
  -e APP_GC_PERCENTAGE=1 \
  \
  -e MYSQL_MASTER_MAX_POOL_SIZE=5 \
  -e MYSQL_MASTER_MIN_POOL_SIZE=2 \
  -e MYSQL_MASTER_PASSWORD=root \
  -e MYSQL_MASTER_HOST=mysql \
  -e MYSQL_MASTER_PORT=3306 \
  -e MYSQL_DB_NAME=testdb \
  -e MYSQL_MASTER_USERNAME=root \
  \
  -e MYSQL_SLAVE_MAX_POOL_SIZE=5 \
  -e MYSQL_SLAVE_MIN_POOL_SIZE=2 \
  -e MYSQL_SLAVE_PASSWORD=root \
  -e MYSQL_SLAVE_HOST=mysql \
  -e MYSQL_SLAVE_PORT=3306 \
  -e MYSQL_SLAVE_USERNAME=root \
  -e MYSQL_ACTIVE_CONFIG_IDS=2 \
  \
  -e ETCD_WATCHER_ENABLED=true \
  -e ETCD_SERVER=etcd:2379 \
  -e ORION_APP_NAME=orion \
  \
  -e SCYLLA_1_CONTACT_POINTS=scylla \
  -e SCYLLA_1_KEYSPACE=orion \
  -e SCYLLA_1_NUM_CONNS=1 \
  -e SCYLLA_1_PORT=9042 \
  -e SCYLLA_1_TIMEOUT_IN_MS=300000 \
  -e SCYLLA_1_PASSWORD= \
  -e SCYLLA_1_USERNAME= \
  -e SCYLLA_ACTIVE_CONFIG_IDS=1 \
  \
  "$IMAGE"

echo "⏳ Waiting for service to be available..."

# Healthcheck loop
for i in {1..30}; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:$PORT/health || true)
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
