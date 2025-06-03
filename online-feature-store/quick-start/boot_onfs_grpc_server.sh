#!/bin/bash

set -e

IMAGE=meeshotech/bharatmlstack-onfs-grpc-api-server:v0.1.16
CONTAINER_NAME=onfs-grpc-api-server
PORT=8089

docker rm -f "$CONTAINER_NAME"

echo "🚀 Running container..."
docker run -d --network "onfs-network" \
  --name "$CONTAINER_NAME" \
  -p "$PORT:$PORT" \
  \
  -e APP_ENV=local \
  -e APP_LOG_LEVEL=DEBUG \
  -e APP_METRIC_SAMPLING_RATE=1 \
  -e APP_NAME=onfs \
  -e APP_PORT="$PORT" \
  -e AUTH_TOKEN=test \
  -e POD_IP="127.0.0.1" \
  -e NODE_IP="127.0.0.1" \
  \
  -e STORAGE_REDIS_STANDALONE_2_ADDR=redis:6379 \
  -e STORAGE_REDIS_STANDALONE_2_DB=0 \
  -e STORAGE_REDIS_STANDALONE_2_DISABLE_IDENTITY=true \
  -e STORAGE_REDIS_STANDALONE_2_MAX_IDLE_CONN=32 \
  -e STORAGE_REDIS_STANDALONE_2_MIN_IDLE_CONN=20 \
  -e STORAGE_REDIS_STANDALONE_2_MAX_ACTIVE_CONN=32 \
  -e STORAGE_REDIS_STANDALONE_2_MAX_RETRY=-1 \
  -e STORAGE_REDIS_STANDALONE_2_POOL_FIFO=false \
  -e STORAGE_REDIS_STANDALONE_2_READ_TIMEOUT_IN_MS=300 \
  -e STORAGE_REDIS_STANDALONE_2_WRITE_TIMEOUT_IN_MS=300 \
  -e STORAGE_REDIS_STANDALONE_2_POOL_TIMEOUT_IN_MS=300 \
  -e STORAGE_REDIS_STANDALONE_2_POOL_SIZE=32 \
  -e STORAGE_REDIS_STANDALONE_2_CONN_MAX_IDLE_TIMEOUT_IN_MINUTES=15 \
  -e STORAGE_REDIS_STANDALONE_2_CONN_MAX_AGE_IN_MINUTES=30 \
  -e STORAGE_REDIS_STANDALONE_ACTIVE_CONFIG_IDS=2 \
  -e DISTRIBUTED_CACHE_CONF_IDS=2 \
  \
  -e ETCD_SERVER=http://etcd:2379 \
  -e ETCD_WATCHER_ENABLED=true \
  \
  -e IN_MEM_CACHE_3_ENABLED=true \
  -e IN_MEM_CACHE_3_NAME=onfs \
  -e IN_MEM_CACHE_3_SIZE_IN_BYTES=100000 \
  -e IN_MEM_CACHE_ACTIVE_CONFIG_IDS=3 \
  \
  -e STORAGE_SCYLLA_1_CONTACT_POINTS=scylla \
  -e STORAGE_SCYLLA_1_KEYSPACE=onfs \
  -e STORAGE_SCYLLA_1_NUM_CONNS=1 \
  -e STORAGE_SCYLLA_1_PORT=9042 \
  -e STORAGE_SCYLLA_1_TIMEOUT_IN_MS=300000 \
  -e STORAGE_SCYLLA_1_USERNAME= \
  -e STORAGE_SCYLLA_1_PASSWORD= \
  -e STORAGE_SCYLLA_ACTIVE_CONFIG_IDS=1 \
  \
  -e P2P_CACHE_5_ENABLED=true \
  -e P2P_CACHE_5_NAME=p2p-onfs \
  -e P2P_CACHE_5_OWN_PARTITION_SIZE_IN_BYTES=100000 \
  -e P2P_CACHE_5_GLOBAL_SIZE_IN_BYTES=1000 \
  -e P2P_CACHE_ACTIVE_CONFIG_IDS=5 \
  \
  "$IMAGE"

echo "⏳ Waiting for /health/self to be available..."

# Wait until /health/self responds successfully (max 30 seconds)
for i in {1..30}; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:$PORT/health/self || true)
  if [ "$STATUS" = "200" ]; then
    echo "✅ /health/self check passed"
    echo "📦 Container '$CONTAINER_NAME' is running on port $PORT"
    echo "👉 Visit http://localhost:$PORT or check logs with: docker logs -f $CONTAINER_NAME"
    exit 0
  fi
  sleep 1
done

echo "❌ /health/self check failed after timeout"
docker logs $CONTAINER_NAME
docker stop $CONTAINER_NAME > /dev/null
exit 1
