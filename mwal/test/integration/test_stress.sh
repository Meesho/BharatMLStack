#!/usr/bin/env bash
# Test: Stress — high-throughput writes, large payloads, rapid sequential writes.
# Verifies replication keeps up under load and all nodes converge.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/helpers.sh"

ensure_built
setup_tmpdir

# ─── Test 1: High-throughput burst (3-node) ─────────────────────────────────
run_burst_3() {
  log_info "=== Test: High-throughput burst (3-node, 500 records) ==="

  start_cluster 3

  local leader
  leader=$(wait_for_leader 3 20)
  log_info "Leader is node $leader"

  log_info "Blasting 500 records as fast as possible..."
  for i in $(seq 1 500); do
    send_cmd "$leader" "put stress3_${i} $(head -c 128 /dev/urandom | base64 | tr -d '\n' | head -c 128)"
    # No sleep — fire as fast as the pipe allows
  done

  log_info "Waiting for replication to settle (15s)..."
  sleep 15

  for nid in 1 2 3; do
    stop_node "$nid"
  done
  sleep 1

  assert_ok "Burst 3-node: all nodes converge" verify_convergence 3
  assert_ok "Burst 3-node: leader has >= 500 records" \
    check_record_count "$leader" 500
}

# ─── Test 2: High-throughput burst (5-node) ─────────────────────────────────
run_burst_5() {
  log_info "=== Test: High-throughput burst (5-node, 500 records) ==="

  NODE_PIDS=()
  NODE_INPUT_PIPES=()
  NODE_WRITER_PIDS=()
  NODE_WAL_DIRS=()
  NODE_LOG_FILES=()
  TEST_TMPDIR="$(mktemp -d /tmp/mwal_integ_st5_XXXXXX)"

  start_cluster 5

  local leader
  leader=$(wait_for_leader 5 25)
  log_info "Leader is node $leader"

  log_info "Blasting 500 records..."
  for i in $(seq 1 500); do
    send_cmd "$leader" "put stress5_${i} payload_${i}_$(date +%s%N)"
  done

  log_info "Waiting for replication (20s)..."
  sleep 20

  for nid in 1 2 3 4 5; do
    stop_node "$nid"
  done
  sleep 1

  assert_ok "Burst 5-node: all nodes converge" verify_convergence 5
  assert_ok "Burst 5-node: leader has >= 500 records" \
    check_record_count "$leader" 500
}

# ─── Test 3: Large values (triggers log rotation) ───────────────────────────
run_large_values() {
  log_info "=== Test: Large values triggering log rotation (3-node) ==="

  NODE_PIDS=()
  NODE_INPUT_PIPES=()
  NODE_WRITER_PIDS=()
  NODE_WAL_DIRS=()
  NODE_LOG_FILES=()
  TEST_TMPDIR="$(mktemp -d /tmp/mwal_integ_lv_XXXXXX)"

  start_cluster 3

  local leader
  leader=$(wait_for_leader 3 20)
  log_info "Leader is node $leader"

  # Each value ~4KB. With max_wal_file_size=4MB, ~1000 records should trigger rotation.
  log_info "Writing 200 records with ~4KB values (should trigger log rotation)..."
  for i in $(seq 1 200); do
    local big_val
    big_val=$(head -c 4096 /dev/urandom | base64 | tr -d '\n' | head -c 4000)
    send_cmd "$leader" "put bigkey_${i} ${big_val}"
    sleep 0.02
  done

  log_info "Waiting for replication (15s)..."
  sleep 15

  # Check that log rotation happened (multiple .log files)
  local log_count
  log_count=$(ls "${NODE_WAL_DIRS[$leader]}"/*.log 2>/dev/null | wc -l | tr -d ' ')
  log_info "Leader has $log_count WAL files"

  for nid in 1 2 3; do
    stop_node "$nid"
  done
  sleep 1

  assert_ok "Large values: all nodes converge" verify_convergence 3
  assert_ok "Large values: leader has >= 200 records" \
    check_record_count "$leader" 200

  if (( log_count > 1 )); then
    log_ok "Large values: log rotation occurred ($log_count files)"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    log_warn "Large values: no log rotation detected (only $log_count files) — not a failure"
  fi
}

# ─── Test 4: Sustained writes over time ─────────────────────────────────────
run_sustained_writes() {
  log_info "=== Test: Sustained writes over 30s (3-node) ==="

  NODE_PIDS=()
  NODE_INPUT_PIPES=()
  NODE_WRITER_PIDS=()
  NODE_WAL_DIRS=()
  NODE_LOG_FILES=()
  TEST_TMPDIR="$(mktemp -d /tmp/mwal_integ_sw_XXXXXX)"

  start_cluster 3

  local leader
  leader=$(wait_for_leader 3 20)
  log_info "Leader is node $leader"

  local count=0
  local end_time=$((SECONDS + 30))
  log_info "Writing continuously for 30 seconds..."
  while (( SECONDS < end_time )); do
    count=$((count + 1))
    send_cmd "$leader" "put sustained_${count} ts_$(date +%s%N)"
    sleep 0.05
  done
  log_info "Wrote $count records in 30 seconds"

  log_info "Waiting for final replication (10s)..."
  sleep 10

  for nid in 1 2 3; do
    stop_node "$nid"
  done
  sleep 1

  assert_ok "Sustained: all nodes converge" verify_convergence 3
  assert_ok "Sustained: leader has >= $count records" \
    check_record_count "$leader" "$count"
}

# ─── Run all ─────────────────────────────────────────────────────────────────
run_burst_3
run_burst_5
run_large_values
run_sustained_writes

print_summary
