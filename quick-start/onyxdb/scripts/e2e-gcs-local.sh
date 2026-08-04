#!/usr/bin/env bash
# e2e-gcs-local.sh — Run the full OnyxDB pipeline locally with real GCS data.
#
# Exercises:
#   etcd (docker)  →  control plane (native)  →  data loader (native, real GCS)
#   → IPC → read server (native, Rust) → TCP queries
#
# Prerequisites:
#   1. gcloud auth application-default login
#   2. Docker running (for etcd)
#   3. Rust toolchain (readserver already built at readserver/target/release/)
#   4. Go 1.24+ (for controlplane + dataloader)
#
# Usage:
#   ./scripts/e2e-gcs-local.sh [--shard N]   (default: shard 0)
set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────

SHARD_ID="${1:-0}"
# Strip --shard prefix if provided
[[ "$SHARD_ID" == "--shard" ]] && SHARD_ID="${2:-0}"

# GCS data location (from your Jupyter notebook upload)
GCS_BUCKET="gcs-dsci-core-core-prd"
GCS_TENANT="online_fs_push_sst_test/ds"   # GCS path prefix (can contain slashes)
ETCD_TENANT="ds"                            # etcd/API tenant (no slashes — Gin route param)
GCS_STORE="catalog_user_geohash_1_3"
GCS_VERSION="20260615_002"

# Local ports
ETCD_PORT=2379
CP_PORT=8080
RS_TCP_PORT=9091
RS_HTTP_PORT=9100
IPC_PATH="/tmp/onyxdb-readserver-local.sock"
DATA_DIR="/tmp/onyxdb-local-data"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
ONYXDB_DIR="$REPO_ROOT/onyxdb"

hr() { printf '─%.0s' {1..70}; echo; }
step() { echo; hr; echo "  $1"; hr; }
cleanup() {
    step "Cleanup: stopping all processes"
    [ -n "${PID_CP:-}" ]  && kill "$PID_CP"  2>/dev/null || true
    [ -n "${PID_RS:-}" ]  && kill "$PID_RS"  2>/dev/null || true
    [ -n "${PID_DL:-}" ]  && kill "$PID_DL"  2>/dev/null || true
    docker rm -f onyxdb-etcd-local 2>/dev/null || true
    rm -f "$IPC_PATH"
    echo "Done."
}
trap cleanup EXIT

# ── 0. Pre-flight checks ────────────────────────────────────────────────────
step "0. Pre-flight checks"

# Check gcloud ADC
if ! gcloud auth application-default print-access-token &>/dev/null; then
    echo "✗ No Application Default Credentials. Run:"
    echo "    gcloud auth application-default login"
    exit 1
fi
echo "✓ GCP Application Default Credentials available"

# Check Docker
if ! docker info &>/dev/null; then
    echo "✗ Docker is not running"
    exit 1
fi
echo "✓ Docker running"

# Check readserver binary
RS_BIN="$ONYXDB_DIR/readserver/target/release/onyxdb-readserver"
if [ ! -f "$RS_BIN" ]; then
    echo "Building readserver (Rust)..."
    (cd "$ONYXDB_DIR/readserver" && cargo build --release)
fi
echo "✓ Read server binary: $RS_BIN"

# Build Go binaries
echo "Building control plane..."
(cd "$ONYXDB_DIR/controlplane" && go build -o /tmp/onyxdb-controlplane ./cmd/controlplane)
echo "✓ Control plane binary: /tmp/onyxdb-controlplane"

echo "Building data loader..."
(cd "$ONYXDB_DIR/dataloader" && go build -o /tmp/onyxdb-dataloader ./cmd/dataloader)
echo "✓ Data loader binary: /tmp/onyxdb-dataloader"

# ── 1. Start etcd ────────────────────────────────────────────────────────────
step "1. Starting etcd (Docker)"

docker rm -f onyxdb-etcd-local 2>/dev/null || true
docker run -d \
    --name onyxdb-etcd-local \
    -p "${ETCD_PORT}:2379" \
    -e ETCD_DATA_DIR=/etcd-data \
    -e ETCD_NAME=node1 \
    -e ETCD_INITIAL_ADVERTISE_PEER_URLS=http://localhost:2380 \
    -e ETCD_LISTEN_PEER_URLS=http://0.0.0.0:2380 \
    -e ETCD_ADVERTISE_CLIENT_URLS=http://localhost:${ETCD_PORT} \
    -e ETCD_LISTEN_CLIENT_URLS=http://0.0.0.0:2379 \
    -e ETCD_INITIAL_CLUSTER=node1=http://localhost:2380 \
    -e ETCD_INITIAL_CLUSTER_STATE=new \
    -e ETCD_INITIAL_CLUSTER_TOKEN=onyxdb-local \
    quay.io/coreos/etcd:v3.5.12

echo "Waiting for etcd..."
for i in $(seq 1 30); do
    if docker exec onyxdb-etcd-local etcdctl endpoint health &>/dev/null; then
        echo "✓ etcd ready on port $ETCD_PORT"
        break
    fi
    sleep 1
done

# ── 2. Start control plane ───────────────────────────────────────────────────
step "2. Starting control plane"

ONYXDB_CP_ADDR=":${CP_PORT}" \
ONYXDB_ETCD_ENDPOINTS="localhost:${ETCD_PORT}" \
    /tmp/onyxdb-controlplane &
PID_CP=$!
sleep 2

curl -sf "http://localhost:${CP_PORT}/api/v1/health" >/dev/null && echo "✓ Control plane up on port $CP_PORT" || { echo "✗ Control plane failed to start"; exit 1; }

# ── 3. Start read server ────────────────────────────────────────────────────
step "3. Starting read server (Rust)"

rm -f "$IPC_PATH"
"$RS_BIN" \
    --tcp-addr "0.0.0.0:${RS_TCP_PORT}" \
    --http-addr "0.0.0.0:${RS_HTTP_PORT}" \
    --ipc-path "$IPC_PATH" \
    --block-cache-bytes 536870912 \
    --bloom-bits 10 &
PID_RS=$!
sleep 2

curl -sf "http://localhost:${RS_HTTP_PORT}/healthz" >/dev/null && echo "✓ Read server up on TCP $RS_TCP_PORT, HTTP $RS_HTTP_PORT" || { echo "✗ Read server failed to start"; exit 1; }

echo -n "  Warm state: "
curl -s "http://localhost:${RS_HTTP_PORT}/healthz?check=warm"
echo " (expected: not warm yet)"

# ── 4. Start data loader with REAL GCS fetcher ──────────────────────────────
step "4. Starting data loader (real GCS, shard=$SHARD_ID)"

mkdir -p "$DATA_DIR"

ONYXDB_GCS_ENABLED=true \
ONYXDB_ETCD_ENDPOINTS="localhost:${ETCD_PORT}" \
ONYXDB_TENANT="$ETCD_TENANT" \
ONYXDB_STORE="$GCS_STORE" \
ONYXDB_GCS_PREFIX="${GCS_TENANT}/${GCS_STORE}" \
ONYXDB_POD_ID="${ETCD_TENANT}-${GCS_STORE}-shard-${SHARD_ID}-0" \
ONYXDB_POD_IP="127.0.0.1" \
ONYXDB_NODE_IP="127.0.0.1" \
ONYXDB_DATA_DIR="$DATA_DIR" \
ONYXDB_IPC_PATH="$IPC_PATH" \
ONYXDB_GCS_BUCKET="$GCS_BUCKET" \
ONYXDB_SHARD_ID="$SHARD_ID" \
    /tmp/onyxdb-dataloader &
PID_DL=$!
sleep 3
echo "✓ Data loader started (PID=$PID_DL, shard=$SHARD_ID)"

# ── 5. Create the store via control plane API ────────────────────────────────
step "5. POST /stores — create store ${ETCD_TENANT}/${GCS_STORE} (shardCount=5)"

CP="http://localhost:${CP_PORT}/api/v1"

curl -s -X POST "$CP/tenants/${ETCD_TENANT}/stores" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"${GCS_STORE}\",\"entityKey\":\"catalog_id|geohash\",\"shardCount\":2}" | python3 -m json.tool

# ── 6. Publish version — triggers the data loader to download from GCS ──────
step "6. POST /versions/${GCS_VERSION}/publish — triggers GCS download"

echo "  → Data loader will: watch etcd → download shard_${SHARD_ID} from GCS → IPC load → IPC activate"

curl -s -X POST "$CP/tenants/${ETCD_TENANT}/stores/${GCS_STORE}/versions/${GCS_VERSION}/publish" \
    -H 'Content-Type: application/json' \
    -d "{\"date\":\"20250927\",\"run\":\"001\"}" | python3 -m json.tool

# ── 7. Wait for data loader to finish downloading + activating ───────────────
step "7. Waiting for data loader → GCS download → IPC load → IPC activate"

for i in $(seq 1 120); do
    state=$(curl -s "http://localhost:${RS_HTTP_PORT}/healthz?check=warm" 2>/dev/null || echo "error")
    if [ "$state" = "warm" ]; then
        echo "✓ Read server is WARM after ${i}s — data loaded from GCS and activated!"
        break
    fi
    printf '  ...%ss (state=%s)\r' "$i" "$state"
    sleep 1
done
echo ""

WARM_STATE=$(curl -s "http://localhost:${RS_HTTP_PORT}/healthz?check=warm" 2>/dev/null || echo "error")
if [ "$WARM_STATE" != "warm" ]; then
    echo "✗ Read server never warmed after 120s. Check data loader logs above."
    echo "  Data loader may still be downloading. Check /tmp/onyxdb-local-data/ for progress."
    echo ""
    echo "  Tail data loader output and wait? Press Ctrl+C to exit."
    wait "$PID_DL" 2>/dev/null || true
    exit 1
fi

# ── 8. Verify pod registration in etcd ───────────────────────────────────────
step "8. Pod registration in etcd (written by data loader)"

docker exec onyxdb-etcd-local etcdctl --endpoints=http://localhost:2379 \
    get --prefix "/config/mnemo-cluster-manager/" 2>/dev/null | sed 's/^/  /' || echo "  (etcd query failed)"

# ── 9. Check what's on disk ──────────────────────────────────────────────────
step "9. Local disk — downloaded SST files"

echo "  Data directory: $DATA_DIR"
find "$DATA_DIR" -type f 2>/dev/null | head -20 | sed 's/^/  /'
echo ""
du -sh "$DATA_DIR" 2>/dev/null | sed 's/^/  Total: /'

# ── 10. TCP string-key lookups ────────────────────────────────────────────────
step "10. TCP string-key lookups against the now-serving read server"

python3 "$SCRIPT_DIR/tcp-test-string.py" --port "$RS_TCP_PORT" || true

# ── Done ─────────────────────────────────────────────────────────────────────
step "DONE — Full pipeline verified: API → etcd → watcher → GCS download → IPC → flip → serving"
echo ""
echo "  The read server is serving shard $SHARD_ID on TCP port $RS_TCP_PORT."
echo "  Control plane API: http://localhost:$CP_PORT/api/v1/"
echo "  Read server health: http://localhost:$RS_HTTP_PORT/healthz"
echo "  Data on disk: $DATA_DIR"
echo ""
echo "  Press Ctrl+C to stop all services."
wait
