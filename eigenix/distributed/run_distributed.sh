#!/usr/bin/env bash
set -euo pipefail

# Usage: ./run_distributed.sh <build_dir> [workers.txt] [n] [k] [dim]
# Example: ./run_distributed.sh ../build workers.txt.example 1000000 256 128

BUILD="${1:?Usage: $0 <build_dir> [workers.txt] [n] [k] [dim]}"
WORKERS_FILE="${2:-workers.txt.example}"
N="${3:-1000000}"
K="${4:-256}"
DIM="${5:-128}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
if [[ "$WORKERS_FILE" == /* ]] || [[ -f "$WORKERS_FILE" ]]; then
    WORKERS_PATH="$WORKERS_FILE"
else
    WORKERS_PATH="${SCRIPT_DIR}/${WORKERS_FILE}"
fi

if [[ ! -f "$WORKERS_PATH" ]]; then
    echo "Workers file not found: $WORKERS_PATH"
    exit 1
fi

# Count non-comment, non-empty lines.
N_WORKERS=$(grep -cv '^\s*#\|^\s*$' "$WORKERS_PATH" || true)
echo "Starting ${N_WORKERS} workers locally..."

PIDS=()

# Start workers (assumes all workers run on localhost for this script).
while IFS= read -r line; do
    # Skip comments and empty lines.
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ -z "${line// /}" ]] && continue

    PORT="${line##*:}"
    echo "  Starting worker on port ${PORT}"
    "${BUILD}/distributed/dist_worker" --port "${PORT}" --verbose &
    PIDS+=($!)
done < "$WORKERS_PATH"

# Give workers time to bind their ports.
sleep 1

echo "Starting coordinator..."
"${BUILD}/distributed/dist_coordinator" \
    --workers "$WORKERS_PATH" \
    --n "$N" \
    --k "$K" \
    --dim "$DIM" \
    --max-iter 100 \
    --tol 0.01 \
    --seed 42 \
    --verbose

echo "Waiting for workers to exit..."
for pid in "${PIDS[@]}"; do
    wait "$pid" 2>/dev/null || true
done

echo "Done."
