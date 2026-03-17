#!/usr/bin/env bash
# Shared helpers for mwal replication integration tests.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
MWAL_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="${MWAL_ROOT}/build"
REPLICATED_NODE="${BUILD_DIR}/examples/replicated_node"
WAL_DUMP="${BUILD_DIR}/examples/wal_dump"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# Log to stderr so output is visible even when functions run inside $(...) command substitution
log_info()  { echo -e "${CYAN}[INFO]${NC}  $*" >&2; }
log_ok()    { echo -e "${GREEN}[PASS]${NC}  $*" >&2; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*" >&2; }
log_fail()  { echo -e "${RED}[FAIL]${NC}  $*" >&2; }

declare -a NODE_PIDS=()
declare -a NODE_WAL_DIRS=()
declare -a NODE_LOG_FILES=()
declare -a NODE_INPUT_PIPES=()
declare -a NODE_WRITER_PIDS=()

TEST_TMPDIR=""

setup_tmpdir() {
  TEST_TMPDIR="$(mktemp -d /tmp/mwal_integ_XXXXXX)"
  log_info "Test temp dir: $TEST_TMPDIR"
}

cleanup_all() {
  log_info "Cleaning up..."
  for pid in "${NODE_WRITER_PIDS[@]:-}"; do
    [[ -n "${pid:-}" ]] && kill "$pid" 2>/dev/null || true
  done
  for pid in "${NODE_PIDS[@]:-}"; do
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
    fi
  done
  for p in "${NODE_INPUT_PIPES[@]:-}"; do
    [[ -p "${p:-}" ]] && rm -f "$p"
  done
  if [[ -n "${TEST_TMPDIR:-}" && -d "${TEST_TMPDIR:-}" ]]; then
    rm -rf "$TEST_TMPDIR"
  fi
}

trap cleanup_all EXIT

ensure_built() {
  if [[ ! -x "$REPLICATED_NODE" ]]; then
    log_fail "replicated_node not found at $REPLICATED_NODE"
    log_info "Build with: cd $BUILD_DIR && cmake .. -DMWAL_BUILD_REPLICATION=ON -DMWAL_BUILD_EXAMPLES=ON && make -j"
    exit 1
  fi
  if [[ ! -x "$WAL_DUMP" ]]; then
    log_fail "wal_dump not found at $WAL_DUMP"
    exit 1
  fi
}

start_node() {
  local nid=$1
  local num_nodes=$2
  local grpc_base=${GRPC_BASE_PORT:-52050}
  local raft_base=${RAFT_BASE_PORT:-49152}
  local grpc_port=$((grpc_base + nid))
  local raft_port=$((raft_base + nid))
  local wal_dir="${TEST_TMPDIR}/wal_node_${nid}"
  local log_file="${TEST_TMPDIR}/node_${nid}.log"
  local input_pipe="${TEST_TMPDIR}/node_${nid}.pipe"

  mkdir -p "$wal_dir"
  [[ -p "$input_pipe" ]] || mkfifo "$input_pipe"

  NODE_WAL_DIRS[$nid]="$wal_dir"
  NODE_LOG_FILES[$nid]="$log_file"
  NODE_INPUT_PIPES[$nid]="$input_pipe"

  "$REPLICATED_NODE" "$nid" "$grpc_port" "$raft_port" "$num_nodes" "$wal_dir" \
    "$raft_base" "$grpc_base" < "$input_pipe" >> "$log_file" 2>&1 &
  NODE_PIDS[$nid]=$!

  ( while true; do sleep 3600; done ) > "$input_pipe" &
  NODE_WRITER_PIDS[$nid]=$!
  disown ${NODE_WRITER_PIDS[$nid]} 2>/dev/null || true

  log_info "Started node $nid (pid=${NODE_PIDS[$nid]}, gRPC=$grpc_port, Raft=$raft_port)"
}

start_cluster() {
  local num_nodes=$1
  export RAFT_BASE_PORT=${RAFT_BASE_PORT:-49152}
  export GRPC_BASE_PORT=${GRPC_BASE_PORT:-52050}
  for nid in $(seq 1 "$num_nodes"); do
    start_node "$nid" "$num_nodes"
    [[ $nid -lt $num_nodes ]] && sleep 0.5 || true
  done
}

send_cmd() {
  local nid=$1
  shift
  local cmd="$*"
  echo "$cmd" > "${NODE_INPUT_PIPES[$nid]}"
}

wait_for_nodes_started() {
  local num_nodes=$1
  local timeout=${2:-10}
  local deadline=$((SECONDS + timeout))

  while (( SECONDS < deadline )); do
    local all_ok=true
    for nid in $(seq 1 "$num_nodes"); do
      if ! kill -0 "${NODE_PIDS[$nid]:-0}" 2>/dev/null; then
        log_fail "Node $nid process has exited (check log below)"
        tail -50 "${NODE_LOG_FILES[$nid]:-}" 2>/dev/null || true
        return 1
      fi
      if ! grep -q " started " "${NODE_LOG_FILES[$nid]:-}" 2>/dev/null; then
        all_ok=false
        break
      fi
    done
    if $all_ok; then return 0; fi
    sleep 0.5
  done

  log_fail "Not all nodes printed 'started' within ${timeout}s. Logs:"
  for nid in $(seq 1 "$num_nodes"); do
    log_info "--- Node $nid (pid=${NODE_PIDS[$nid]:-?}) last 30 lines ---"
    tail -30 "${NODE_LOG_FILES[$nid]:-}" 2>/dev/null || true
  done
  return 1
}

wait_for_leader() {
  local num_nodes=$1
  local timeout=${2:-20}
  local deadline=$((SECONDS + timeout))
  log_info "Waiting for node startup..."
  sleep 4
  if ! wait_for_nodes_started "$num_nodes" 10; then
    return 1
  fi
  log_info "Nodes started, waiting for leader election..."
  while (( SECONDS < deadline )); do
    for nid in $(seq 1 "$num_nodes"); do
      if [[ -z "${NODE_PIDS[$nid]:-}" ]] || ! kill -0 "${NODE_PIDS[$nid]}" 2>/dev/null; then
        continue
      fi
      send_cmd "$nid" "status"
      sleep 0.3
      if tail -15 "${NODE_LOG_FILES[$nid]:-}" 2>/dev/null | grep -q "leader=1"; then
        echo "$nid"
        return 0
      fi
    done
    sleep 0.5
  done

  log_fail "No leader elected in ${timeout}s. Node logs:"
  for nid in $(seq 1 "$num_nodes"); do
    log_info "--- Node $nid log (last 25 lines) ---"
    tail -25 "${NODE_LOG_FILES[$nid]:-}" 2>/dev/null || true
  done
  return 1
}

write_records() {
  local nid=$1
  local count=$2
  local prefix=${3:-"key"}
  for i in $(seq 1 "$count"); do
    send_cmd "$nid" "put ${prefix}_${i} value_${i}"
    sleep 0.02
  done
}

stop_node() {
  local nid=$1
  # Only send "quit" if the node is still running; otherwise writing to the FIFO blocks (no reader)
  if [[ -n "${NODE_PIDS[$nid]:-}" ]] && kill -0 "${NODE_PIDS[$nid]}" 2>/dev/null; then
    send_cmd "$nid" "quit"
    # Give node time to read "quit", run mgr.Stop(), wal->Close(), and exit (releases WAL LOCK)
    sleep 2
    if kill -0 "${NODE_PIDS[$nid]}" 2>/dev/null; then
      kill "${NODE_PIDS[$nid]}" 2>/dev/null || true
      wait "${NODE_PIDS[$nid]}" 2>/dev/null || true
    fi
  fi
  # Always wait so the process is fully gone and WAL LOCK is released before wal_dump runs
  [[ -n "${NODE_PIDS[$nid]:-}" ]] && wait "${NODE_PIDS[$nid]}" 2>/dev/null || true
  if [[ -n "${NODE_WRITER_PIDS[$nid]:-}" ]]; then
    kill "${NODE_WRITER_PIDS[$nid]}" 2>/dev/null || true
  fi
}

kill_node() {
  local nid=$1
  if [[ -n "${NODE_PIDS[$nid]:-}" ]] && kill -0 "${NODE_PIDS[$nid]}" 2>/dev/null; then
    kill -9 "${NODE_PIDS[$nid]}"
    wait "${NODE_PIDS[$nid]}" 2>/dev/null || true
  fi
  if [[ -n "${NODE_WRITER_PIDS[$nid]:-}" ]]; then
    kill "${NODE_WRITER_PIDS[$nid]}" 2>/dev/null || true
  fi
}

# Pause node process (SIGSTOP) so it stops replicating and can be moved to OSR.
pause_node() {
  local nid=$1
  if [[ -n "${NODE_PIDS[$nid]:-}" ]] && kill -0 "${NODE_PIDS[$nid]}" 2>/dev/null; then
    kill -STOP "${NODE_PIDS[$nid]}"
  fi
}

# Resume node process (SIGCONT) after pause.
resume_node() {
  local nid=$1
  if [[ -n "${NODE_PIDS[$nid]:-}" ]] && kill -0 "${NODE_PIDS[$nid]}" 2>/dev/null; then
    kill -CONT "${NODE_PIDS[$nid]}"
  fi
}

restart_node() {
  local nid=$1
  local num_nodes=$2
  stop_node "$nid"
  sleep 1
  start_node "$nid" "$num_nodes"
}

verify_convergence() {
  local num_nodes=$1
  local ref_dump="${TEST_TMPDIR}/conv_ref.txt"
  "$WAL_DUMP" "${NODE_WAL_DIRS[1]}" > "$ref_dump" 2>/dev/null || true
  for nid in $(seq 2 "$num_nodes"); do
    local dump="${TEST_TMPDIR}/conv_${nid}.txt"
    "$WAL_DUMP" "${NODE_WAL_DIRS[$nid]}" > "$dump" 2>/dev/null || true
    if ! diff -q "$ref_dump" "$dump" >/dev/null 2>&1; then
      return 1
    fi
  done
  return 0
}

check_record_count() {
  local nid=$1
  local min_count=$2
  local wal_dir="${NODE_WAL_DIRS[$nid]}"
  local count
  count=$("$WAL_DUMP" "$wal_dir" 2>/dev/null | wc -l)
  [[ ${count:-0} -ge $min_count ]]
}

# Write node WAL to a temp file and echo its path (for chaos test verification).
dump_wal() {
  local nid=$1
  local out="${TEST_TMPDIR}/chaos_dump_${nid}.txt"
  "$WAL_DUMP" "${NODE_WAL_DIRS[$nid]}" > "$out" 2>/dev/null || true
  echo "$out"
}

compare_wal_dumps() {
  local nid_a=$1
  local nid_b=$2
  local dir_a="${NODE_WAL_DIRS[$nid_a]}"
  local dir_b="${NODE_WAL_DIRS[$nid_b]}"
  local out_a="${TEST_TMPDIR}/dump_${nid_a}.txt"
  local out_b="${TEST_TMPDIR}/dump_${nid_b}.txt"
  "$WAL_DUMP" "$dir_a" > "$out_a" 2>/dev/null || true
  "$WAL_DUMP" "$dir_b" > "$out_b" 2>/dev/null || true
  if diff -q "$out_a" "$out_b" >/dev/null 2>&1; then
    return 0
  fi
  local count_a count_b
  count_a=$(wc -l < "$out_a" 2>/dev/null || echo 0)
  count_b=$(wc -l < "$out_b" 2>/dev/null || echo 0)
  log_fail "WAL dumps differ: node $nid_a has $count_a lines, node $nid_b has $count_b lines"
  log_info "First 30 lines of diff (node $nid_a vs node $nid_b):"
  diff -u "$out_a" "$out_b" 2>/dev/null | head -30 >&2 || true
  return 1
}

assert_ok() {
  local msg=$1
  shift
  if "$@"; then
    log_ok "$msg"
    return 0
  else
    log_fail "$msg"
    return 1
  fi
}

print_summary() {
  local failed="${TESTS_FAILED:-0}"
  if [[ "$failed" -gt 0 ]]; then
    log_fail "$failed test(s) failed"
    exit 1
  fi
  log_ok "All tests passed"
  exit 0
}
