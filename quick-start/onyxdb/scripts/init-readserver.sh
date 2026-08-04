#!/usr/bin/env bash
# init-readserver.sh — send IPC load+activate to a running onyxdb-readserver container.
# Run this after `docker compose up` if the onyxdb-readserver-init container already ran.
set -euo pipefail

VERSION="${1:-20260526_001}"
SHARD_PATH="/data/shard"
SOCK="onyxdb-ipc:/ipc/onyxdb-readserver.sock"

echo "=== IPC load version $VERSION ==="
docker exec onyxdb-readserver-init \
  sh -c "printf '{\"cmd\":\"load\",\"version\":\"$VERSION\",\"shards\":{\"0\":\"$SHARD_PATH\"}}\n' | socat - UNIX-CONNECT:/ipc/onyxdb-readserver.sock" 2>/dev/null || \
docker run --rm --network onyxdb-quick-start_onyxdb-net \
  -v onyxdb-quick-start_onyxdb-ipc:/ipc \
  alpine sh -c "
    apk add --no-cache socat >/dev/null 2>&1
    printf '{\"cmd\":\"load\",\"version\":\"$VERSION\",\"shards\":{\"0\":\"$SHARD_PATH\"}}\n' | socat - UNIX-CONNECT:/ipc/onyxdb-readserver.sock
  "

echo "=== IPC activate version $VERSION ==="
docker run --rm --network onyxdb-quick-start_onyxdb-net \
  -v onyxdb-quick-start_onyxdb-ipc:/ipc \
  alpine sh -c "
    apk add --no-cache socat >/dev/null 2>&1
    printf '{\"cmd\":\"activate\",\"version\":\"$VERSION\"}\n' | socat - UNIX-CONNECT:/ipc/onyxdb-readserver.sock
  "

echo "=== warm check ==="
curl -sf "http://localhost:9100/healthz?check=warm" && echo " warm!"
