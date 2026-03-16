#!/usr/bin/env bash
# Test: Chaos — randomly kill/restart nodes while writing continuously.
# Mini Jepsen-style soak test for replication robustness.
#
# Parameters (env vars):
#   CHAOS_DURATION   — total test duration in seconds (default: 60)
#   CHAOS_NODES      — cluster size (default: 5)
#   CHAOS_KILL_INTERVAL — seconds between random kills (default: 8)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/helpers.sh"

ensure_built
setup_tmpdir

CHAOS_DURATION=${CHAOS_DURATION:-60}
CHAOS_NODES=${CHAOS_NODES:-5}
CHAOS_KILL_INTERVAL=${CHAOS_KILL_INTERVAL:-8}

log_info "=== Chaos Test: ${CHAOS_NODES}-node cluster, ${CHAOS_DURATION}s duration ==="

start_cluster "$CHAOS_NODES"

log_info "Waiting for initial leader election..."
sleep 8

# ── Writer loop (background) ────────────────────────────────────────────────
WRITE_COUNT_FILE="${TEST_TMPDIR}/write_count"
echo 0 > "$WRITE_COUNT_FILE"
WRITER_DONE="${TEST_TMPDIR}/writer_done"

writer_loop() {
  local count=0
  local end_time=$((SECONDS + CHAOS_DURATION))

  while (( SECONDS < end_time )); do
    # Find current leader
    for nid in $(seq 1 "$CHAOS_NODES"); do
      if [[ -z "${NODE_PIDS[$nid]:-}" ]] || ! kill -0 "${NODE_PIDS[$nid]}" 2>/dev/null; then
        continue
      fi
      send_cmd "$nid" "status"
      sleep 0.2
      if tail -3 "${NODE_LOG_FILES[$nid]}" 2>/dev/null | grep -q "leader=1"; then
        count=$((count + 1))
        send_cmd "$nid" "put chaos_${count} val_${count}_$(date +%s%N)"
        sleep 0.1
        break
      fi
    done
    sleep 0.2
  done

  echo "$count" > "$WRITE_COUNT_FILE"
  touch "$WRITER_DONE"
}

writer_loop &
WRITER_PID=$!

# ── Chaos loop (main thread) ────────────────────────────────────────────────
chaos_end=$((SECONDS + CHAOS_DURATION - 15))  # Stop killing 15s before end
kill_count=0

while (( SECONDS < chaos_end )); do
  sleep "$CHAOS_KILL_INTERVAL"

  # Count alive nodes — don't kill if we'd lose majority
  local alive=0
  for nid in $(seq 1 "$CHAOS_NODES"); do
    if [[ -n "${NODE_PIDS[$nid]:-}" ]] && kill -0 "${NODE_PIDS[$nid]}" 2>/dev/null; then
      alive=$((alive + 1))
    fi
  done

  local majority=$(( (CHAOS_NODES / 2) + 1 ))
  if (( alive <= majority )); then
    log_info "Only $alive nodes alive (need $majority majority) — restarting killed nodes instead"
    for nid in $(seq 1 "$CHAOS_NODES"); do
      if [[ -z "${NODE_PIDS[$nid]:-}" ]] || ! kill -0 "${NODE_PIDS[$nid]}" 2>/dev/null; then
        log_info "Restarting node $nid..."
        restart_node "$nid" "$CHAOS_NODES"
        sleep 2
      fi
    done
    continue
  fi

  # Pick a random alive node to kill
  local candidates=()
  for nid in $(seq 1 "$CHAOS_NODES"); do
    if [[ -n "${NODE_PIDS[$nid]:-}" ]] && kill -0 "${NODE_PIDS[$nid]}" 2>/dev/null; then
      candidates+=("$nid")
    fi
  done

  if (( ${#candidates[@]} == 0 )); then
    continue
  fi

  local victim=${candidates[$((RANDOM % ${#candidates[@]}))]}
  kill_count=$((kill_count + 1))
  log_info "Chaos #$kill_count: killing node $victim"
  kill_node "$victim"

  # Wait a bit, then restart it
  sleep 3
  log_info "Restarting node $victim..."
  restart_node "$victim" "$CHAOS_NODES"
  sleep 2
done

# ── Wait for writer to finish ────────────────────────────────────────────────
log_info "Waiting for writer to finish..."
wait "$WRITER_PID" 2>/dev/null || true

local_write_count=$(cat "$WRITE_COUNT_FILE")
log_info "Writer completed: $local_write_count records written, $kill_count kills performed"

# ── Stabilization period ─────────────────────────────────────────────────────
log_info "Stabilization period (15s) — let all nodes catch up..."

# Ensure all nodes are alive
for nid in $(seq 1 "$CHAOS_NODES"); do
  if [[ -z "${NODE_PIDS[$nid]:-}" ]] || ! kill -0 "${NODE_PIDS[$nid]}" 2>/dev/null; then
    restart_node "$nid" "$CHAOS_NODES"
  fi
done
sleep 15

# ── Shutdown and verify ─────────────────────────────────────────────────────
for nid in $(seq 1 "$CHAOS_NODES"); do
  stop_node "$nid"
done
sleep 2

# Find the node with the most records (should be the last leader)
local max_records=0
local max_node=1
for nid in $(seq 1 "$CHAOS_NODES"); do
  local dump
  dump=$(dump_wal "$nid")
  local cnt
  cnt=$(wc -l < "$dump" | tr -d ' ')
  log_info "Node $nid: $cnt records"
  if (( cnt > max_records )); then
    max_records=$cnt
    max_node=$nid
  fi
done

log_info "Max records: $max_records (node $max_node)"
log_info "Writes attempted: $local_write_count"

# In chaos mode, not all writes may succeed (leader may be killed mid-write).
# But all surviving records must be consistent across nodes.
# Check that all nodes that have records are consistent with each other.
local ref_dump
ref_dump=$(dump_wal "$max_node")
local converged=true

for nid in $(seq 1 "$CHAOS_NODES"); do
  if [[ "$nid" == "$max_node" ]]; then continue; fi
  local other_dump
  other_dump=$(dump_wal "$nid")

  local ref_lines other_lines
  ref_lines=$(wc -l < "$ref_dump" | tr -d ' ')
  other_lines=$(wc -l < "$other_dump" | tr -d ' ')

  # The node with fewer records should be a prefix of the max node
  if (( other_lines <= ref_lines )); then
    if head -n "$other_lines" "$ref_dump" | diff -q - "$other_dump" > /dev/null 2>&1; then
      log_ok "Node $nid ($other_lines records) is a valid prefix of node $max_node"
    else
      log_fail "Node $nid has divergent data vs node $max_node"
      converged=false
    fi
  else
    # This node has more — check the reverse
    if head -n "$ref_lines" "$other_dump" | diff -q - "$ref_dump" > /dev/null 2>&1; then
      log_ok "Node $max_node is a valid prefix of node $nid"
    else
      log_fail "Node $nid and node $max_node have divergent data"
      converged=false
    fi
  fi
done

if $converged; then
  log_ok "Chaos test: all nodes are consistent (prefix-consistent)"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  log_fail "Chaos test: data divergence detected"
  TESTS_FAILED=$((TESTS_FAILED + 1))
fi

if (( max_records > 0 )); then
  log_ok "Chaos test: cluster accepted $max_records records despite $kill_count kills"
  TESTS_PASSED=$((TESTS_PASSED + 1))
else
  log_fail "Chaos test: no records survived"
  TESTS_FAILED=$((TESTS_FAILED + 1))
fi

print_summary
