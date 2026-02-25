#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

BUILD_DIR="build"
NPROC="$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)"

echo "=== Building eigenix_bench (Release) ==="
cmake -B "$BUILD_DIR" -DCMAKE_BUILD_TYPE=Release
cmake --build "$BUILD_DIR" -j"$NPROC"

mkdir -p results

echo ""
echo "=== Running benchmark suite ==="
export OMP_PROC_BIND=close
export OMP_PLACES=cores

"$BUILD_DIR/eigenix_bench" | tee results/bench_stdout.txt

echo ""
echo "=== Done. CSV at results/benchmark_results.csv ==="
