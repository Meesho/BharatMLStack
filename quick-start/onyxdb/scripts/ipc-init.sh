#!/bin/sh
# ipc-init.sh — runs inside the Alpine init container.
# Waits for the read server, then sends IPC load + activate.
set -e

apk add --no-cache socat curl >/dev/null 2>&1

echo "=== waiting for readserver HTTP ==="
until curl -sf http://onyxdb-readserver:9100/healthz >/dev/null 2>&1; do
  printf '.'
  sleep 2
done
echo " up!"

echo "=== IPC load ==="
echo '{"cmd":"load","version":"20260526_001","shards":{"0":"/data/shard"}}' \
  | socat - UNIX-CONNECT:/ipc/onyxdb-readserver.sock

echo "=== IPC activate ==="
echo '{"cmd":"activate","version":"20260526_001"}' \
  | socat - UNIX-CONNECT:/ipc/onyxdb-readserver.sock

echo "=== warm check ==="
curl -sf 'http://onyxdb-readserver:9100/healthz?check=warm' && echo " warm!"

echo "=== ready — TCP queries on localhost:9091 ==="
