#!/usr/bin/env bash
# Test: Basic replication — 3-node and 5-node clusters.
# Verifies that writes to the leader are replicated to all followers.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/helpers.sh"

ensure_built
setup_tmpdir

run_3_node_test() {
  log_info "=== Test: 3-node basic replication ==="

  start_cluster 3

  log_info "Waiting for leader election..."
  local leader
  leader=$(wait_for_leader 3 20) || {
    for nid in 1 2 3; do
      log_info "--- Node $nid log ---"
      tail -25 "${NODE_LOG_FILES[$nid]:-}" 2>/dev/null || true
    done
    exit 1
  }
  log_info "Leader is node $leader"

  log_info "Writing 50 records to leader (node $leader)..."
  write_records "$leader" 50 "basic3"

  sleep 3

  for nid in 1 2 3; do
    stop_node "$nid"
  done
  sleep 1

  assert_ok "3-node: all nodes converge" verify_convergence 3
  assert_ok "3-node: leader has >= 50 records" check_record_count "$leader" 50
}

run_5_node_test() {
  log_info "=== Test: 5-node basic replication ==="

  NODE_PIDS=()
  NODE_INPUT_PIPES=()
  NODE_WRITER_PIDS=()
  NODE_WAL_DIRS=()
  NODE_LOG_FILES=()

  TEST_TMPDIR="$(mktemp -d /tmp/mwal_integ_5n_XXXXXX)"
  log_info "Test temp dir: $TEST_TMPDIR"

  start_cluster 5

  log_info "Waiting for leader election..."
  local leader
  leader=$(wait_for_leader 5 25) || {
    for nid in 1 2 3 4 5; do
      log_info "--- Node $nid log ---"
      tail -25 "${NODE_LOG_FILES[$nid]:-}" 2>/dev/null || true
    done
    exit 1
  }
  log_info "Leader is node $leader"

  log_info "Writing 100 records to leader (node $leader)..."
  write_records "$leader" 100 "basic5"

  sleep 5

  for nid in 1 2 3 4 5; do
    stop_node "$nid"
  done
  sleep 1

  assert_ok "5-node: all nodes converge" verify_convergence 5
  assert_ok "5-node: leader has >= 100 records" check_record_count "$leader" 100
}

TESTS_FAILED=0
run_3_node_test || TESTS_FAILED=$((TESTS_FAILED + 1))
run_5_node_test || TESTS_FAILED=$((TESTS_FAILED + 1))

if [[ $TESTS_FAILED -gt 0 ]]; then
  log_fail "$TESTS_FAILED test(s) failed"
  exit 1
fi
log_ok "All basic replication tests passed"
exit 0
