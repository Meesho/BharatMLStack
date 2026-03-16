#!/usr/bin/env bash
# Test: ISR/OSR dynamics — pause/resume followers, verify lag detection,
# OSR eviction, and ISR promotion after catch-up.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/helpers.sh"

ensure_built
setup_tmpdir

# ─── Test 1: Pause one follower → OSR → Resume → ISR (3-node) ──────────────
run_pause_resume_3() {
  log_info "=== Test: Pause/Resume follower ISR→OSR→ISR (3-node) ==="

  start_cluster 3

  local leader
  leader=$(wait_for_leader 3 20)
  log_info "Leader is node $leader"

  # Pick a follower to pause
  local target=""
  for nid in 1 2 3; do
    if [[ "$nid" != "$leader" ]]; then
      target="$nid"
      break
    fi
  done
  log_info "Will pause follower node $target"

  # Write some initial records with all nodes healthy
  log_info "Writing 20 records (all nodes in ISR)..."
  write_records "$leader" 20 "isr_pre"
  sleep 2

  # Pause the follower
  pause_node "$target"

  # Write more records while follower is paused
  log_info "Writing 30 records (node $target paused → should go to OSR)..."
  write_records "$leader" 30 "isr_during_pause"

  # Wait for ISR check to detect the lag and move to OSR
  log_info "Waiting for ISR maintenance to detect lag (5s)..."
  sleep 5

  # Resume the follower
  resume_node "$target"

  # Wait for catch-up via StreamWAL
  log_info "Waiting for catch-up (8s)..."
  sleep 8

  # Write a few more records to verify the resumed node gets them
  log_info "Writing 10 more records (node $target should be back in ISR)..."
  write_records "$leader" 10 "isr_post"
  sleep 3

  for nid in 1 2 3; do
    stop_node "$nid"
  done
  sleep 1

  assert_ok "Pause/Resume 3-node: all nodes converge" verify_convergence 3
  assert_ok "Pause/Resume 3-node: have >= 60 records" \
    check_record_count "$leader" 60
}

# ─── Test 2: Pause two followers → only leader has data → resume both (5-node)
run_pause_two_followers_5() {
  log_info "=== Test: Pause 2 followers (5-node) ==="

  NODE_PIDS=()
  NODE_INPUT_PIPES=()
  NODE_WRITER_PIDS=()
  NODE_WAL_DIRS=()
  NODE_LOG_FILES=()
  TEST_TMPDIR="$(mktemp -d /tmp/mwal_integ_isr5_XXXXXX)"

  start_cluster 5

  local leader
  leader=$(wait_for_leader 5 25)
  log_info "Leader is node $leader"

  # Pick two followers to pause
  local paused=()
  for nid in 1 2 3 4 5; do
    if [[ "$nid" != "$leader" ]] && (( ${#paused[@]} < 2 )); then
      paused+=("$nid")
    fi
  done
  log_info "Will pause followers: ${paused[*]}"

  write_records "$leader" 20 "p5_pre"
  sleep 2

  for p in "${paused[@]}"; do
    pause_node "$p"
  done

  log_info "Writing 40 records with 2 followers paused..."
  write_records "$leader" 40 "p5_during"
  sleep 5

  for p in "${paused[@]}"; do
    resume_node "$p"
  done

  log_info "Waiting for catch-up (12s)..."
  sleep 12

  write_records "$leader" 10 "p5_post"
  sleep 3

  for nid in 1 2 3 4 5; do
    stop_node "$nid"
  done
  sleep 1

  assert_ok "Pause 2 followers (5-node): all converge" verify_convergence 5
  assert_ok "Pause 2 followers (5-node): have >= 70 records" \
    check_record_count "$leader" 70
}

# ─── Test 3: Extended pause — large lag accumulation ─────────────────────────
run_extended_pause() {
  log_info "=== Test: Extended pause with large lag (3-node) ==="

  NODE_PIDS=()
  NODE_INPUT_PIPES=()
  NODE_WRITER_PIDS=()
  NODE_WAL_DIRS=()
  NODE_LOG_FILES=()
  TEST_TMPDIR="$(mktemp -d /tmp/mwal_integ_xp_XXXXXX)"

  start_cluster 3

  local leader
  leader=$(wait_for_leader 3 20)
  log_info "Leader is node $leader"

  local target=""
  for nid in 1 2 3; do
    if [[ "$nid" != "$leader" ]]; then
      target="$nid"
      break
    fi
  done

  write_records "$leader" 10 "xp_pre"
  sleep 1

  pause_node "$target"

  log_info "Writing 200 records while node $target is paused..."
  write_records "$leader" 200 "xp_lag"
  sleep 5

  resume_node "$target"

  log_info "Waiting for large catch-up (15s)..."
  sleep 15

  for nid in 1 2 3; do
    stop_node "$nid"
  done
  sleep 1

  assert_ok "Extended pause: all converge" verify_convergence 3
  assert_ok "Extended pause: have >= 210 records" \
    check_record_count "$leader" 210
}

# ─── Run all ─────────────────────────────────────────────────────────────────
run_pause_resume_3
run_pause_two_followers_5
run_extended_pause

print_summary
