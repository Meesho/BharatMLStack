#!/usr/bin/env bash
# Master runner for all mwal replication integration tests.
#
# Usage:
#   ./run_all.sh              # Run all tests
#   ./run_all.sh basic        # Run only basic replication tests
#   ./run_all.sh failover     # Run only leader failover tests
#   ./run_all.sh isr          # Run only ISR/OSR tests
#   ./run_all.sh stress       # Run only stress tests
#   ./run_all.sh chaos        # Run only chaos tests
#
# Environment:
#   CHAOS_DURATION=120        # Override chaos test duration (default: 60s)
#   CHAOS_NODES=5             # Override chaos cluster size (default: 5)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MWAL_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="${MWAL_ROOT}/build"

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[0;33m'
NC='\033[0m'

# ── Build check ──────────────────────────────────────────────────────────────
if [[ ! -x "${BUILD_DIR}/examples/replicated_node" ]] || [[ ! -x "${BUILD_DIR}/examples/wal_dump" ]]; then
  echo -e "${YELLOW}[WARN]${NC} Binaries not found. Building with replication enabled..."
  mkdir -p "$BUILD_DIR"
  cd "$BUILD_DIR"
  cmake .. -DMWAL_BUILD_REPLICATION=ON 2>&1 | tail -5
  make -j"$(sysctl -n hw.ncpu 2>/dev/null || nproc 2>/dev/null || echo 4)" 2>&1 | tail -10
  cd "$SCRIPT_DIR"
  echo -e "${GREEN}[OK]${NC} Build complete."
  echo ""
fi

# ── Test registry ────────────────────────────────────────────────────────────
declare -A TESTS=(
  [basic]="test_basic_replication.sh"
  [failover]="test_leader_failover.sh"
  [isr]="test_isr_osr.sh"
  [stress]="test_stress.sh"
  [chaos]="test_chaos.sh"
)

TEST_ORDER=(basic failover isr stress chaos)

# ── Determine which tests to run ─────────────────────────────────────────────
if [[ $# -gt 0 ]]; then
  selected=("$@")
else
  selected=("${TEST_ORDER[@]}")
fi

# ── Run tests ────────────────────────────────────────────────────────────────
total_passed=0
total_failed=0
declare -A suite_results=()

echo ""
echo "╔══════════════════════════════════════════════════════╗"
echo "║     mwal Replication Integration Test Suite          ║"
echo "╚══════════════════════════════════════════════════════╝"
echo ""

for name in "${selected[@]}"; do
  script="${TESTS[$name]:-}"
  if [[ -z "$script" ]]; then
    echo -e "${RED}Unknown test suite: $name${NC}"
    echo "Available: ${!TESTS[*]}"
    exit 1
  fi

  script_path="${SCRIPT_DIR}/${script}"
  if [[ ! -x "$script_path" ]]; then
    chmod +x "$script_path"
  fi

  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${CYAN}  Running: $name ($script)${NC}"
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo ""

  start_time=$SECONDS
  if bash "$script_path"; then
    elapsed=$((SECONDS - start_time))
    suite_results[$name]="PASS (${elapsed}s)"
    echo -e "\n${GREEN}  ✓ $name passed (${elapsed}s)${NC}\n"
  else
    elapsed=$((SECONDS - start_time))
    suite_results[$name]="FAIL (${elapsed}s)"
    total_failed=$((total_failed + 1))
    echo -e "\n${RED}  ✗ $name failed (${elapsed}s)${NC}\n"
  fi
done

# ── Summary ──────────────────────────────────────────────────────────────────
echo ""
echo "╔══════════════════════════════════════════════════════╗"
echo "║                  FINAL SUMMARY                       ║"
echo "╠══════════════════════════════════════════════════════╣"

for name in "${selected[@]}"; do
  result="${suite_results[$name]:-SKIPPED}"
  if [[ "$result" == PASS* ]]; then
    printf "║  ${GREEN}%-12s %s${NC}%*s║\n" "$name" "$result" $((36 - ${#name} - ${#result})) ""
  else
    printf "║  ${RED}%-12s %s${NC}%*s║\n" "$name" "$result" $((36 - ${#name} - ${#result})) ""
  fi
done

echo "╚══════════════════════════════════════════════════════╝"
echo ""

if (( total_failed > 0 )); then
  echo -e "${RED}$total_failed suite(s) failed.${NC}"
  exit 1
else
  echo -e "${GREEN}All suites passed!${NC}"
  exit 0
fi
