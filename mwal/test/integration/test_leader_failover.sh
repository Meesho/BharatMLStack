#!/usr/bin/env bash
# Test: Leader failover — kill leader, wait for re-election, restart old leader,
# verify convergence.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/helpers.sh"

ensure_built
setup_tmpdir

# ─── Test 1: 3-node failover ────────────────────────────────────────────────
run_3_node_failover() {
  log_info "=== Test: 3-node leader failover ==="

  start_cluster 3

  local leader
  leader=$(wait_for_leader 3 20)
  log_info "Initial leader is node $leader"

  log_info "Writing 30 records to leader..."
  write_records "$leader" 30 "pre_failover"

  sleep 2

  log_info "Killing leader (node $leader) with SIGKILL..."
  kill_node "$leader"

  log_info "Waiting for new leader election..."
  sleep 5

  local new_leader=""
  for nid in 1 2 3; do
    if [[ "$nid" == "$leader" ]]; then continue; fi
    if [[ -z "${NODE_PIDS[$nid]:-}" ]] || ! kill -0 "${NODE_PIDS[$nid]}" 2>/dev/null; then
      continue
    fi
    send_cmd "$nid" "status"
    sleep 0.5
    if grep -q "leader=1" "${NODE_LOG_FILES[$nid]}" 2>/dev/null; then
      new_leader="$nid"
      break
    fi
  done

  if [[ -z "$new_leader" ]]; then
    # Try harder — poll a few more times
    for attempt in $(seq 1 10); do
      for nid in 1 2 3; do
        if [[ "$nid" == "$leader" ]]; then continue; fi
        if [[ -z "${NODE_PIDS[$nid]:-}" ]] || ! kill -0 "${NODE_PIDS[$nid]}" 2>/dev/null; then
          continue
        fi
        send_cmd "$nid" "status"
        sleep 0.5
        if tail -5 "${NODE_LOG_FILES[$nid]}" 2>/dev/null | grep -q "leader=1"; then
          new_leader="$nid"
          break 2
        fi
      done
      sleep 1
    done
  fi

  if [[ -z "$new_leader" ]]; then
    log_fail "No new leader elected after failover"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return
  fi
  log_info "New leader is node $new_leader"

  log_info "Writing 20 more records to new leader..."
  write_records "$new_leader" 20 "post_failover"

  sleep 2

  log_info "Restarting old leader (node $leader)..."
  restart_node "$leader" 3

  log_info "Waiting for catch-up (10s)..."
  sleep 10

  # Stop all nodes
  for nid in 1 2 3; do
    stop_node "$nid"
  done
  sleep 1

  # Survivors should converge
  local survivors=()
  for nid in 1 2 3; do
    if [[ "$nid" != "$leader" ]]; then
      survivors+=("$nid")
    fi
  done

  assert_ok "3-node failover: survivors converge" \
    compare_wal_dumps "${survivors[0]}" "${survivors[1]}"

  # The restarted node should also catch up
  assert_ok "3-node failover: restarted node caught up" \
    compare_wal_dumps "${survivors[0]}" "$leader"

  assert_ok "3-node failover: have >= 30 records (pre-failover)" \
    check_record_count "${survivors[0]}" 30
}

# ─── Test 2: 5-node failover ────────────────────────────────────────────────
run_5_node_failover() {
  log_info "=== Test: 5-node leader failover ==="

  NODE_PIDS=()
  NODE_INPUT_PIPES=()
  NODE_WRITER_PIDS=()
  NODE_WAL_DIRS=()
  NODE_LOG_FILES=()
  TEST_TMPDIR="$(mktemp -d /tmp/mwal_integ_fo5_XXXXXX)"

  start_cluster 5

  local leader
  leader=$(wait_for_leader 5 25)
  log_info "Initial leader is node $leader"

  log_info "Writing 50 records..."
  write_records "$leader" 50 "fo5_pre"

  sleep 3

  log_info "Killing leader (node $leader)..."
  kill_node "$leader"
  sleep 8

  local new_leader=""
  for attempt in $(seq 1 15); do
    for nid in 1 2 3 4 5; do
      if [[ "$nid" == "$leader" ]]; then continue; fi
      if [[ -z "${NODE_PIDS[$nid]:-}" ]] || ! kill -0 "${NODE_PIDS[$nid]}" 2>/dev/null; then
        continue
      fi
      send_cmd "$nid" "status"
      sleep 0.3
      if tail -5 "${NODE_LOG_FILES[$nid]}" 2>/dev/null | grep -q "leader=1"; then
        new_leader="$nid"
        break 2
      fi
    done
    sleep 1
  done

  if [[ -z "$new_leader" ]]; then
    log_fail "No new leader elected in 5-node cluster"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return
  fi
  log_info "New leader is node $new_leader"

  log_info "Writing 30 more records to new leader..."
  write_records "$new_leader" 30 "fo5_post"

  sleep 3

  log_info "Restarting old leader (node $leader)..."
  restart_node "$leader" 5
  sleep 12

  for nid in 1 2 3 4 5; do
    stop_node "$nid"
  done
  sleep 1

  # Pick two survivors to compare
  local ref=""
  for nid in 1 2 3 4 5; do
    if [[ "$nid" != "$leader" ]]; then
      ref="$nid"
      break
    fi
  done

  assert_ok "5-node failover: restarted node caught up" \
    compare_wal_dumps "$ref" "$leader"
  assert_ok "5-node failover: have >= 50 records" \
    check_record_count "$ref" 50
}

# ─── Test 3: Double failover ────────────────────────────────────────────────
run_double_failover() {
  log_info "=== Test: Double failover (3-node) ==="

  NODE_PIDS=()
  NODE_INPUT_PIPES=()
  NODE_WRITER_PIDS=()
  NODE_WAL_DIRS=()
  NODE_LOG_FILES=()
  TEST_TMPDIR="$(mktemp -d /tmp/mwal_integ_dfo_XXXXXX)"

  start_cluster 3

  local leader
  leader=$(wait_for_leader 3 20)
  log_info "First leader is node $leader"

  write_records "$leader" 20 "round1"
  sleep 2

  log_info "Killing first leader (node $leader)..."
  kill_node "$leader"
  sleep 5

  local second_leader=""
  for attempt in $(seq 1 15); do
    for nid in 1 2 3; do
      if [[ "$nid" == "$leader" ]]; then continue; fi
      if [[ -z "${NODE_PIDS[$nid]:-}" ]] || ! kill -0 "${NODE_PIDS[$nid]}" 2>/dev/null; then
        continue
      fi
      send_cmd "$nid" "status"
      sleep 0.3
      if tail -5 "${NODE_LOG_FILES[$nid]}" 2>/dev/null | grep -q "leader=1"; then
        second_leader="$nid"
        break 2
      fi
    done
    sleep 1
  done

  if [[ -z "$second_leader" ]]; then
    log_fail "No second leader elected"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return
  fi
  log_info "Second leader is node $second_leader"

  write_records "$second_leader" 20 "round2"
  sleep 2

  log_info "Killing second leader (node $second_leader)..."
  kill_node "$second_leader"
  sleep 5

  # Only one node left — it can't form majority, so just restart both
  log_info "Restarting both killed nodes..."
  restart_node "$leader" 3
  restart_node "$second_leader" 3
  sleep 12

  local third_leader
  third_leader=$(wait_for_leader 3 20) || true

  if [[ -n "$third_leader" ]]; then
    write_records "$third_leader" 10 "round3"
    sleep 3
  fi

  for nid in 1 2 3; do
    stop_node "$nid"
  done
  sleep 1

  assert_ok "Double failover: all nodes converge" verify_convergence 3
}

# ─── Run all ─────────────────────────────────────────────────────────────────
run_3_node_failover
run_5_node_failover
run_double_failover

print_summary
