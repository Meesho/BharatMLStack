#!/usr/bin/env bash
# e2e-dataloader.sh — Drive the full OnyxDB data-plane pipeline through the
# control plane API and verify the read server flips via the data-loader sidecar.
#
#   control plane REST  →  etcd  →  data-loader watcher  →  IPC  →  read server flip
#
# Prereqs: docker compose -f docker-compose.dataloader.yml up -d
set -euo pipefail

CP="http://localhost:8080/api/v1"
RS_HTTP="http://localhost:9100"
TENANT="online_fs_push_sst_test"
STORE="ds_catalog_user_geohash_1_3"
VERSION="20260615_002"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

hr() { printf '─%.0s' {1..70}; echo; }
step() { echo; hr; echo "  $1"; hr; }

# ── 0. Pre-flight ─────────────────────────────────────────────────────────────
step "0. Pre-flight: services healthy?"
curl -sf "$CP/health" >/dev/null && echo "✓ control plane up" || { echo "✗ control plane down"; exit 1; }
curl -sf "$RS_HTTP/healthz" >/dev/null && echo "✓ read server up" || { echo "✗ read server down"; exit 1; }

echo -n "Read server warm state (expect 'not warm' — nothing loaded yet): "
curl -s "$RS_HTTP/healthz?check=warm"; echo

# ── 1. Create the store ───────────────────────────────────────────────────────
step "1. POST /stores  — create store $TENANT/$STORE (shardCount=2)"
curl -s -X POST "$CP/tenants/$TENANT/stores" \
  -H 'Content-Type: application/json' \
  -d '{"name":"'"$STORE"'","entityKey":"catalog_id|geohash","shardCount":2}' | python3 -m json.tool

# ── 2. Publish a version — THIS is the API update that triggers the flip ──────
step "2. POST /versions/$VERSION/publish  — writes status=READY to etcd"
echo "  → the data-loader watcher will see this and run: download → IPC load → IPC activate"
curl -s -X POST "$CP/tenants/$TENANT/stores/$STORE/versions/$VERSION/publish" \
  -H 'Content-Type: application/json' \
  -d '{"date":"20260526","run":"001"}' | python3 -m json.tool

# ── 3. Wait for the data loader to drive the read server warm ────────────────
step "3. Waiting for data-loader → IPC → read server atomic flip"
for i in $(seq 1 20); do
  state=$(curl -s "$RS_HTTP/healthz?check=warm" || true)
  if [ "$state" = "warm" ]; then
    echo "✓ read server is WARM after ${i}s — the atomic flip happened"
    break
  fi
  printf '  ...%ss (state=%s)\n' "$i" "$state"
  sleep 1
done
[ "$(curl -s "$RS_HTTP/healthz?check=warm")" = "warm" ] || { echo "✗ read server never warmed — check: docker compose -f docker-compose.dataloader.yml logs onyxdb-dataloader"; exit 1; }

# ── 4. Verify the data loader registered itself as a warm pod in etcd ────────
step "4. Pod registration in etcd (written by the data loader)"
docker exec onyxdb-etcd etcdctl --endpoints=http://localhost:2379 \
  get --prefix "/config/mnemo-cluster-manager/$TENANT/$STORE/" | sed 's/^/  /'

# ── 5. Promote via control plane — coverage check must pass (pod is warm) ────
step "5. POST /rollback-able promote  — control plane CAS flips activeVersion"
echo "  → control plane checks pod coverage, then the data-loader sees the"
echo "    activeVersion change and sends an (idempotent) IPC activate."
curl -s -X POST "$CP/tenants/$TENANT/stores/$STORE/versions/$VERSION/promote" | python3 -m json.tool

step "6. GET /topology  — active version + shard→pod assignment"
curl -s "$CP/tenants/$TENANT/stores/$STORE/topology" | python3 -m json.tool

# ── 7. Real TCP query against the flipped read server ────────────────────────
step "7. TCP string-key lookups against the now-serving read server"
python3 "$SCRIPT_DIR/tcp-test-string.py"

step "DONE — full pipeline verified: API → etcd → watcher → IPC → flip → serving"
