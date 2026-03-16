# mwal + NuRaft + ISR/OSR Replication — Design Document

This document describes the high-level design for running **mwal** (Write-Ahead Log) with **Raft consensus** for leader election and **ISR/OSR** (In-Sync / Out-of-Sync Replica) replication. It covers scope, architecture, APIs, data flows, client routing, failure handling, and configuration.

**Document map**

| Section | Content |
|---------|---------|
| **1. Scope & goals** | What is in scope (WAL + Raft + ISR/OSR); what is out of scope (state store) |
| **2. Architecture overview** | Layers, components, per-node layout, directory isolation |
| **3. NuRaft integration** | Leader election only; metadata state machine; no WAL data in Raft |
| **4. ISR/OSR replication** | Sync to ISR, async to OSR; min in-sync replicas; selection strategy; committed vs persisted |
| **5. mwal API additions** | AppendReplicated (locking, sequence semantics); WriteRecordCallback; TruncateAfter |
| **6. gRPC replication protocol** | Replicate (with prev_lsn), StreamWAL, ReportProgress; request/response shapes |
| **7. Data flows** | Leader write path; replica receive path; catch-up (StreamWAL) |
| **8. Client request routing** | Redirect vs forward; recommended: reject + redirect to leader |
| **9. Leader failover & log reconciliation** | Truncation on new leader, replica divergence, fencing |
| **10. Snapshot, catch-up & WAL retention** | When used; WAL retention for slow replicas; snapshot + WAL stream; replica recovery |
| **11. Configuration** | Raft, replication, ISR, WAL retention parameters |
| **12. Summary** | Component responsibility matrix; directory layout |

---

## 1. Scope & Goals

### 1.1 In scope

- **mwal**: WAL library — append-only log, recovery, iterator (existing behaviour plus new APIs below).
- **NuRaft**: Leader election, term management, cluster membership, heartbeats. Raft log carries **cluster metadata only** (e.g. ISR set changes), not WAL data.
- **ISR (In-Sync Replicas)**: A subset of followers that receive **synchronous** replication; a write is committed only when the leader and all ISR members have persisted it (or when enough ISR members ack to meet `min_insync_replicas`).
- **OSR (Out-of-Sync Replicas)**: Remaining followers receive **asynchronous** replication; no impact on write commit or availability.
- **Client routing**: When a write lands on a non-leader, the node rejects and returns the leader's address (redirect); clients send writes to the leader.
- **Leader failover**: Log truncation and divergence reconciliation when a new leader is elected.

### 1.2 Out of scope

- **State store**: The structure that holds current id→vector (or any key-value) state and applies committed WAL entries. Not part of this design; a separate layer consumes mwal for durability.
- **Vector index** (e.g. HNSW/IVF): Built and maintained outside this replication layer.

### 1.3 Goals

- Run WAL with Raft and ISR/OSR in a single repo (self-contained).
- Leader is the only node that accepts and processes client writes; replicas only receive replication traffic.
- Durability: writes are committed only when replicated to at least `min_insync_replicas` (ISR).
- Availability: if ISR shrinks below `min_insync_replicas`, writes are rejected until ISR is restored or config is relaxed.
- Correctness: on leader failover, divergent (uncommitted) WAL tails are truncated so all nodes converge to the new leader's log.

---

## 2. Architecture Overview

### 2.1 Layer diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Application / Vector DB (out of scope)               │
└─────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Replication layer (in this repo)                         │
│  • ReplicationManager (ISR/OSR, sync/async fan-out, committed_lsn_)        │
│  • NuRaft integration (leader election, term, cluster metadata)            │
│  • gRPC replication service (Replicate, StreamWAL, ReportProgress)         │
│  • ISR selection & maintenance (zone-aware, lag-based)                      │
└─────────────────────────────────────────────────────────────────────────────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    ▼                   ▼                   ▼
┌───────────────────────────┐ ┌───────────────────┐ ┌─────────────────────────┐
│  mwal (existing + APIs)   │ │  NuRaft (lib)     │ │  gRPC (transport)        │
│  • DBWal                  │ │  • raft_server    │ │  • ReplicationServiceImpl│
│  • Write / Recover /       │ │  • state_machine   │ │  • Stubs to other nodes │
│    NewWalIterator         │ │  • log_store       │ │                          │
│  • AppendReplicated (NEW) │ │  • raft_callback    │ │                          │
│  • WriteRecordCallback    │ │                    │ │                          │
│    (NEW – for leader)     │ │                    │ │                          │
│  • TruncateAfter (NEW)   │ │                    │ │                          │
└───────────────────────────┘ └───────────────────┘ └─────────────────────────┘
```

### 2.2 What runs per node

Each node runs one process that includes:

| Component | Role |
|-----------|------|
| **NuRaft** | Leader election, term, heartbeats, cluster config. Does not replicate WAL data. |
| **mwal (DBWal)** | Local WAL: leader appends via Write(); replicas append via AppendReplicated(); Recover, NewWalIterator for catch-up. Each node has its **own** `wal_dir` with its own `LOCK` file. |
| **ReplicationManager** | Uses NuRaft for "am I leader?" and cluster list; uses mwal for local log; uses gRPC to send/receive WAL records. Implements ISR (sync) and OSR (async). Tracks `committed_lsn_` separately from mwal's `last_sequence_`. |
| **gRPC server** | Exposes Replicate, StreamWAL, ReportProgress. Receives WAL from leader; calls mwal's new APIs on replicas. |
| **gRPC clients** | Leader uses these to send Replicate / StreamWAL to followers (ISR and OSR). |

### 2.3 Per-node WAL directory isolation

Each node opens mwal with its own `wal_dir` (e.g. `/data/node-A/wal/`, `/data/node-B/wal/`). mwal acquires an exclusive `flock` on `wal_dir/LOCK` at `Open()` — this prevents two processes from using the same directory, but has no effect across nodes since each has a separate directory. Leader and replicas never share a WAL directory.

### 2.4 Example: 5 nodes A, B, C, D, E

- Raft elects **A** as leader.
- **min_insync_replicas = 2**; ISR is chosen (e.g. zone-aware) as **{C, E}**; OSR = **{B, D}**.
- Client writes go to A (after optional redirect). A appends to local WAL, sync-replicates to C and E, async-replicates to B and D. Commit when C and E ack (and |ISR| ≥ 2).

---

## 3. NuRaft Integration

### 3.1 Role of NuRaft

- **Use for:** Leader election, term management, cluster membership, heartbeats.
- **Do not use for:** Replicating WAL data. WAL content stays in mwal; replication is done by the replication layer over gRPC with an ISR-based protocol.

### 3.2 State machine (metadata only)

- Raft log entries carry **cluster metadata only**: e.g. ISR set changes, node add/remove.
- A minimal `state_machine` implementation applies these metadata updates (e.g. update in-memory ISR set and notify ReplicationManager).
- Vector/WAL data is never written to the Raft log.

### 3.3 Leader election → ReplicationManager

- When NuRaft elects a leader: Raft callback invokes `ReplicationManager::on_raft_leadership(term, true)`. ReplicationManager sets "I am leader," gets cluster list from Raft, runs ISR selection, starts sync/async replication.
- When the node steps down: callback invokes `on_raft_leadership(term, false)`. ReplicationManager clears leader state and stops replicating.
- **Term** is stored in ReplicationManager and included in every Replicate RPC so replicas can reject stale leaders.

### 3.4 Term in the WAL

- **Recommended:** Include an optional `term` field in `AppendReplicated`. The term is stored as a prefix in the WAL record (see §5.1) for audit/debugging and to support truncation during leader failover (see §9).
- mwal itself does not enforce term semantics — the replication layer uses the term to decide whether to accept or reject an `AppendReplicated` call. The term stored in the record is informational.
- If `term` is 0 or omitted, mwal treats it as "no term" — backward-compatible for non-replicated usage.

---

## 4. ISR/OSR Replication

### 4.1 Definitions

- **ISR (In-Sync Replicas):** Subset of followers that must ack before a write is committed. Size must be ≥ `min_insync_replicas` for writes to succeed.
- **OSR (Out-of-Sync Replicas):** Other followers; they receive replication asynchronously. No ack required for commit.
- **min_insync_replicas:** Minimum number of in-sync replicas (including policy) required for the leader to accept writes. If |ISR| < min_insync_replicas, the leader rejects writes (e.g. InsufficientReplicas).

### 4.2 ISR selection strategy (auto, validated)

- **Priority 1:** Zone diversity — prefer replicas in different failure domains from the leader.
- **Priority 2:** Replication lag — prefer nodes with smallest lag (closest to leader's LSN).
- **Priority 3:** Network RTT — prefer lower latency.
- **Maintenance:** Periodically evict from ISR nodes that exceed `max_lag_entries` or `replica_timeout_ms`; promote from OSR when a node catches up and is healthy.

### 4.3 Committed vs. persisted (two high-water marks)

There are two distinct sequence positions tracked in the system:

| Concept | Where tracked | Meaning |
|---------|--------------|---------|
| **Persisted LSN** (`last_sequence_` in mwal) | mwal (per node) | The highest sequence number written to the local WAL on this node. On the leader this advances on every `Write()`; on replicas it advances on every `AppendReplicated()`. |
| **Committed LSN** (`committed_lsn_` in ReplicationManager) | ReplicationManager (leader only) | The highest sequence number that has been ack'd by enough ISR members. Only records at or below this LSN are safe to apply to the state store. |

- mwal remains a generic WAL and has **no concept of a commit index**. The committed LSN lives entirely in `ReplicationManager`.
- The application layer (out of scope) should only apply records with `seq <= committed_lsn_`.
- The leader advances `committed_lsn_` after receiving ISR acks. It broadcasts `committed_lsn_` to replicas via the `leader_commit` field in `ReplicateRequest` so they know what is safe to apply.

### 4.4 Commit rule

- A write is **committed** when the leader has persisted it and received ack from enough ISR members such that the effective in-sync set meets `min_insync_replicas` (e.g. all current ISR members ack, or a defined subset). Exact policy (all ISR vs N of ISR) is a configuration choice; typically "all ISR" for simplicity.

---

## 5. mwal API Additions

### 5.1 Replica: AppendReplicated

```cpp
Status AppendReplicated(SequenceNumber first_seq,
                        uint32_t count,
                        const Slice& payload,
                        uint64_t term = 0);
```

- **Purpose:** Append a WAL record with a **given** sequence range and optional term. Used when a replica receives a Replicate RPC: it persists the same bytes the leader wrote, without mwal assigning a new sequence.

- **Parameters:**
  - `first_seq`: The first sequence number in the batch (matches the 8-byte sequence in the WriteBatch header inside `payload`).
  - `count`: Number of operations in the batch (matches the 4-byte count in the WriteBatch header). Used to compute the end of the sequence range.
  - `payload`: Exact bytes of one mwal WAL record (1-byte compression prefix + WriteBatch bytes). Same format as written by `Write()` on the leader.
  - `term`: Raft term of the leader that produced this record. Stored as metadata for audit/debugging and used during truncation (§9). 0 means "no term."

- **Semantics:**
  - Acquires `writer_mu_` directly — **does not go through `WriteThread`** (no group commit needed; replicas have a single source of writes: the leader's replication stream).
  - Calls `log_writer_->AddRecord(payload)` to append the record to disk.
  - Updates `last_sequence_` to `first_seq + count - 1` (the last sequence in the batch), matching how `Recover()` tracks `max_seq`.
  - Handles log rotation: if `max_wal_file_size` is exceeded after the append, calls `RotateLogFile()`.

- **Replica configuration:** On replicas, the `WriteCoalescer` should be disabled (`max_async_queue_depth = 0`) since replicas do not accept client writes. Only `AppendReplicated` writes to the WAL.

### 5.2 Leader: WriteRecordCallback

**Decision: Use the callback approach** (Option A from the original design).

```cpp
// In WALOptions:
using WriteRecordCallback = std::function<void(SequenceNumber first_seq,
                                                uint32_t count,
                                                const Slice& record_payload)>;
WriteRecordCallback write_record_callback;
```

- **When invoked:** Inside `Write()` and `WriteCoalescedBatches()`, immediately after `log_writer_->AddRecord(record_data)` succeeds, while still holding `writer_mu_`. This guarantees:
  - The callback fires exactly once per WAL record (even when group commit merges multiple batches).
  - No TOCTOU race — the record bytes are available synchronously.
  - The replication layer can buffer `(first_seq, count, payload)` and dispatch to ISR/OSR.

- **Why not `GetLastWrittenRecord`:** With group commit, another leader thread could overwrite the "last record" before the replication layer reads it. The callback avoids this race entirely.

- **What about `GetLastWrittenRecord` option:** Dropped. The callback is strictly better when concurrent group commits are possible.

- **Payload format:** `record_payload` = the exact bytes passed to `AddRecord` (1-byte compression prefix + WriteBatch bytes). The replication layer sends these bytes as-is to replicas.

### 5.3 New: TruncateAfter(lsn)

```cpp
Status TruncateAfter(SequenceNumber lsn);
```

- **Purpose:** Remove all WAL records with sequence number > `lsn`. Used during leader failover (§9) when a replica discovers it has divergent (uncommitted) records that the new leader does not have.
- **Semantics:**
  - Acquires `writer_mu_`.
  - Iterates WAL files in reverse order. For the current active file: truncate it at the byte offset of the last record with sequence ≤ `lsn`. For older files that are entirely beyond `lsn`, delete them.
  - Sets `last_sequence_` to `lsn`.
  - Closes and recreates `log_writer_` on the (possibly truncated) current file.
- **Safety:** Only called on replicas during leader failover, never on a live leader. The replication layer must ensure no concurrent `AppendReplicated` calls while truncation is in progress (e.g. hold a replication-layer lock).

### 5.4 Existing mwal APIs used as-is

- **Write(options, batch)** — Leader appends; `WriteRecordCallback` provides (first_seq, count, payload) for replication.
- **NewWalIterator(start_seq)**, **GetLiveWalFiles()** — Leader streams WAL for catch-up (StreamWAL).
- **Recover(callback)** — Replay local WAL on restart (usage for "apply to state" is out of scope here).

---

## 6. gRPC Replication Protocol

### 6.1 Services and RPCs

- **Replicate(ReplicateRequest) → ReplicateResponse**
  Leader sends one or more WAL entries (lsn + payload). Replica appends each via `AppendReplicated(lsn, count, payload, term)` and returns success and `last_persisted_lsn`. Used for both ISR (sync, wait for response) and OSR (async, fire-and-forget or background).

- **StreamWAL(StreamWALRequest) → stream WALChunk**
  Leader streams WAL entries from `start_lsn` (and optionally up to `end_lsn`). Used for replica catch-up after a snapshot or when far behind.

- **ReportProgress(ProgressReport) → ProgressAck**
  **Required** (not optional): follower periodically reports its persisted and applied LSN to the leader. The leader uses this to compute replica lag and maintain ISR (evict/promote). Without this, the leader can only infer lag from `Replicate` ack latency, which is unreliable during periods of low write traffic.

### 6.2 Key message fields

```protobuf
message ReplicateEntry {
  uint64 first_seq = 1;     // First sequence number in this batch
  uint32 count = 2;         // Number of operations in the batch
  bytes  payload = 3;       // Exact mwal WAL record bytes
}

message ReplicateRequest {
  uint64 term = 1;                     // Leader's Raft term
  uint64 leader_commit = 2;            // Leader's committed LSN
  uint64 prev_lsn = 3;                // Last sequence number the leader believes the replica has
  repeated ReplicateEntry entries = 4;
}

message ReplicateResponse {
  bool   success = 1;
  string message = 2;
  uint64 last_persisted_lsn = 3;      // Replica's persisted LSN after appending
  uint64 term_seen = 4;               // Replica's current Raft term (for fencing)
}

message StreamWALRequest {
  uint64 start_lsn = 1;
  uint64 end_lsn = 2;                 // 0 = stream to current end
  uint64 term = 3;
}

message WALChunk {
  repeated ReplicateEntry entries = 1;
}

message ProgressReport {
  uint64 node_id = 1;
  uint64 persisted_lsn = 2;           // Highest LSN written to replica's WAL
  uint64 applied_lsn = 3;             // Highest LSN applied to replica's state (optional)
  uint64 term = 4;
}

message ProgressAck {
  uint64 committed_lsn = 1;           // Leader's current committed LSN (so replica can apply)
}
```

### 6.3 Gap detection via `prev_lsn`

- **`prev_lsn`** in `ReplicateRequest` is the leader's expectation of the replica's `last_persisted_lsn` before appending the new entries. Analogous to Raft's `prevLogIndex`.
- **Replica check:** Before appending, the replica verifies `prev_lsn == my last_persisted_lsn`. If not:
  - If `prev_lsn > my last_persisted_lsn`: the replica is behind (gap). Return failure with `last_persisted_lsn` so the leader knows where to start catch-up via `StreamWAL`.
  - If `prev_lsn < my last_persisted_lsn`: the replica has divergent records (e.g. from a previous leader). The replica must truncate to `prev_lsn` (via `TruncateAfter(prev_lsn)`) before appending the new entries. See §9 for the full failover flow.
- **First Replicate after leader election:** The new leader sets `prev_lsn` to its own `last_sequence_` at the time of election. Replicas that diverged will detect the mismatch and truncate.

### 6.4 Payload format

- **payload** = exact bytes of one mwal WAL record (1-byte compression prefix + WriteBatch bytes). Same format as written by mwal on the leader so replicas can append and recover identically.

---

## 7. Data Flows

### 7.1 Leader write path

1. Client sends write to **leader** (after redirect if needed).
2. ReplicationManager checks: am I leader? |ISR| ≥ min_insync_replicas? If not, reject.
3. ReplicationManager calls **mwal::Write(options, batch)**. mwal appends and assigns sequence; `WriteRecordCallback` fires with `(first_seq, count, record_payload)`.
4. The callback buffers `(first_seq, count, payload)` in ReplicationManager's pending queue. When the batch is full (count/size) or timeout:
   - Send **Replicate(term, leader_commit, prev_lsn, entries)** to all ISR members in parallel; wait for acks.
   - If enough ISR acks: advance `committed_lsn_` (commit index).
   - Send same Replicate to OSR asynchronously (no wait).
5. Reply to client with success (or failure if ISR acks insufficient).

### 7.2 Replica receive path

1. ReplicationServiceImpl receives **Replicate(term, leader_commit, prev_lsn, entries)**.
2. If `term` < my Raft term: return failure, `term_seen` = my term.
3. **Gap / divergence check:** If `prev_lsn != my last_persisted_lsn`:
   - If `prev_lsn > my last_persisted_lsn`: return failure with `last_persisted_lsn` (I'm behind; leader should StreamWAL to catch me up).
   - If `prev_lsn < my last_persisted_lsn`: call `TruncateAfter(prev_lsn)` to remove divergent tail, then proceed.
4. For each entry: call **mwal::AppendReplicated(first_seq, count, payload, term)**. On first failure, return failure and `last_persisted_lsn`.
5. Update local `committed_lsn` from `leader_commit` (for the application layer to know what is safe to apply).
6. Return success and `last_persisted_lsn` = last entry's `first_seq + count - 1`.

### 7.3 Catch-up (StreamWAL)

1. Follower (e.g. new or recovered) is behind. It requests **StreamWAL(start_lsn, end_lsn, term)** from the leader.
2. Leader uses **NewWalIterator(start_lsn)** (and current WAL files) to iterate records; streams them in chunks (e.g. WALChunk with multiple entries).
3. Follower for each chunk calls **AppendReplicated(first_seq, count, payload, term)** for each entry until stream ends.
4. Follower is then caught up and can join OSR; ISR maintenance can later promote it to ISR when healthy and within lag.

### 7.4 Log rotation on replicas

- Replicas rotate their WAL files **independently** based on `max_wal_file_size`, just as the leader does. `AppendReplicated` checks file size after each append and calls `RotateLogFile()` if exceeded.
- This means log file boundaries (which records are in which `.log` file) may differ between leader and replicas. This is fine for correctness: mwal's recovery and `WalIterator` work based on sequence numbers, not file boundaries.
- The leader does **not** signal rotation events to replicas.

---

## 8. Client Request Routing

### 8.1 Requirement

- **Only the leader** should process client writes. So either non-leaders **reject and redirect** or they **forward** to the leader.

### 8.2 Recommended: Reject + redirect

- If a **non-leader** receives a write request:
  - Respond with **NotLeader** and the current leader's identity and address (e.g. `leader_id`, `leader_address`).
- Client (or load balancer) **retries** the write to the returned leader address.
- Clients can **cache** the leader and send subsequent writes directly to the leader until they receive a new redirect (e.g. after leader change).

**Benefits:** No extra hop; lower latency; leader is the only node that "gets" write traffic; followers do not proxy.

### 8.3 Alternative: Forward (proxy)

- Non-leader that receives a write **forwards** it to the leader and returns the leader's response to the client.
- **Drawbacks:** Extra hop (client → follower → leader → follower → client); forwarding load on followers. Not recommended for production if "leader should only get all client calls" is a goal.

---

## 9. Leader Failover & Log Reconciliation

### 9.1 The problem

When a leader fails and a new leader is elected from the ISR, there may be **divergent** WAL records:

- The old leader may have written records to its local WAL that were not yet ack'd by ISR (i.e. persisted > committed on the old leader).
- Other replicas may have received some of those uncommitted records (or none, depending on timing).
- The new leader's log is the source of truth — its last committed record is the new "end of log" for the cluster.

### 9.2 Reconciliation protocol

When a new leader is elected:

1. **New leader broadcasts its identity and `last_persisted_lsn` (which equals its last committed LSN, since ISR members are up-to-date).**
2. **Replicas compare** their `last_persisted_lsn` with the new leader's:
   - **Match:** No action needed; replica is in sync.
   - **Replica is behind:** Replica requests `StreamWAL(my_last_persisted_lsn, 0, term)` to catch up.
   - **Replica is ahead (divergent tail):** Replica has records the new leader does not. This happens when the replica received uncommitted records from the old leader. The replica calls `TruncateAfter(new_leader_last_lsn)` to remove the divergent tail, then catches up via StreamWAL if needed.
3. **First `Replicate` RPC** from the new leader carries `prev_lsn` = new leader's last LSN. Any replica that still has a divergent tail will detect the mismatch in the gap check (§6.3) and truncate automatically.

### 9.3 Fencing the old leader

- **Term-based fencing:** Every `Replicate` and `StreamWAL` RPC carries the leader's `term`. Replicas reject RPCs with a term lower than their current Raft term and return `term_seen`. When the old leader (if still alive) sees a `term_seen` higher than its own, it steps down.
- **Raft heartbeats:** NuRaft's heartbeat mechanism also detects the old leader's demotion. The old leader stops replicating once `on_raft_leadership(term, false)` fires.
- **Client fencing:** Clients cache the leader address. When the old leader steps down, it starts returning `NotLeader` with the new leader's address, forcing clients to redirect.

### 9.4 What about the old leader's uncommitted tail?

- The old leader (if it comes back online) will also need to truncate its divergent tail. When it restarts:
  1. It joins the cluster as a follower.
  2. The new leader sends `Replicate(prev_lsn = committed_lsn)`.
  3. The old leader detects `prev_lsn < my last_persisted_lsn`, calls `TruncateAfter(prev_lsn)`.
  4. Normal replication resumes.
- Records lost in the truncation were **never committed** (never ack'd by ISR), so no committed data is lost.

### 9.5 TruncateAfter implementation notes

- `TruncateAfter(lsn)` (§5.3) must be atomic with respect to WAL writes. The replication layer holds a lock that prevents concurrent `AppendReplicated` calls during truncation.
- After truncation, recovery of the local state store must also roll back any applied-but-uncommitted records. This is the application layer's responsibility (out of scope for mwal, but the application must track `committed_lsn` and only apply committed records).

---

## 10. Snapshot, Catch-Up & WAL Retention

### 10.1 When catch-up is needed

- A **new node** or **recovered node** that is far behind (e.g. missing many LSNs).
- A node that was in OSR for a long time and has fallen behind the oldest available WAL on the leader.

### 10.2 WAL retention strategy

The leader must retain WAL files long enough for the slowest replica to catch up via `StreamWAL`. If WAL files are purged before a replica has consumed them, that replica cannot use `StreamWAL` and needs a full snapshot instead (which is expensive and out-of-scope for mwal).

**Mechanism:**

- ReplicationManager on the leader tracks each replica's `last_persisted_lsn` (from `Replicate` acks and `ReportProgress` RPCs).
- It computes **`min_replica_lsn`** = the minimum `last_persisted_lsn` across all replicas (ISR and OSR).
- It calls **`mwal::SetMinLogNumberToKeep(log_number_containing(min_replica_lsn))`** so that `PurgeObsoleteFiles()` does not delete WAL files still needed by any replica.
- This interacts with mwal's existing purge controls (`WAL_ttl_seconds`, `WAL_size_limit_MB`): `SetMinLogNumberToKeep` takes precedence — a file is never purged if its `log_number >= min_log_to_keep`, regardless of TTL or size limits.

**Configuration:**

| Parameter | Example | Meaning |
|-----------|---------|---------|
| `max_replica_lag_before_snapshot` | 50000 | If a replica's lag exceeds this many entries, stop retaining WAL for it and mark it for snapshot-based recovery instead. Prevents unbounded WAL growth due to one stuck replica. |

### 10.3 WAL-based catch-up (StreamWAL)

1. Follower (e.g. new or recovered) is behind. It requests **StreamWAL(start_lsn, end_lsn, term)** from the leader.
2. Leader uses **NewWalIterator(start_lsn)** (and current WAL files) to iterate records; streams them in chunks (e.g. WALChunk with multiple entries).
3. Follower for each chunk calls **AppendReplicated(first_seq, count, payload, term)** for each entry until stream ends.
4. Follower is then caught up and can join OSR; ISR maintenance can later promote it to ISR when healthy and within lag.

### 10.4 Snapshot-based recovery (when WAL is insufficient)

If the leader's oldest WAL file has a starting sequence **greater than** the replica's `last_persisted_lsn`, `StreamWAL` cannot help — the needed records have been purged.

In this case:

1. The application layer (out of scope) creates a **snapshot** of the current state and transfers it to the replica.
2. The snapshot includes the `committed_lsn` at the time of creation.
3. After loading the snapshot, the replica calls **StreamWAL(snapshot_committed_lsn, 0, term)** to catch up on records written after the snapshot.
4. The replica then joins OSR and eventually ISR.

The snapshot creation and transfer protocol is defined by the layer that owns "state" and is **out of scope** for this design. mwal's role is limited to retaining WAL files (via `SetMinLogNumberToKeep`) and providing `StreamWAL` for the WAL portion of catch-up.

---

## 11. Configuration

### 11.1 Raft (NuRaft)

| Parameter | Example | Meaning |
|-----------|---------|---------|
| election_timeout_lower_bound_ms | 200 | Lower bound for election timeout (randomized per node). |
| election_timeout_upper_bound_ms | 400 | Upper bound for election timeout. |
| heart_beat_interval_ms | 75 | Leader heartbeat interval. |

### 11.2 Replication (ReplicationManager)

| Parameter | Example | Meaning |
|-----------|---------|---------|
| min_insync_replicas | 2 | Minimum ISR size for writes to be accepted. |
| replication_timeout_ms | 150 | Max time to wait for ISR ack before evicting/shrinking ISR. |
| isr_check_interval_ms | 1000 | Interval for ISR maintenance (evict/promote). |
| max_lag_entries | 5000 | Replica evicted from ISR if lag exceeds this. |
| replica_timeout_ms | 3000 | No ack for this long → evict from ISR. |
| batch_max_entries | 100 | Max WAL entries per Replicate RPC. |
| batch_max_bytes | 1048576 | Max bytes per Replicate RPC (e.g. 1 MiB). |
| progress_report_interval_ms | 500 | How often replicas send ReportProgress to leader. |
| max_replica_lag_before_snapshot | 50000 | Stop retaining WAL for a replica if it falls this far behind; require snapshot recovery. |

### 11.3 WAL retention (leader-side)

| Parameter | Example | Meaning |
|-----------|---------|---------|
| min_log_to_keep | (dynamic) | Set by ReplicationManager based on slowest replica's LSN. |
| WAL_ttl_seconds | 86400 | Delete WAL files older than 24h — but never if `log_number >= min_log_to_keep`. |
| WAL_size_limit_MB | 10240 | Purge oldest WAL files when total exceeds 10 GB — but never if `log_number >= min_log_to_keep`. |

### 11.4 Term in mwal log

- Stored as a field in `AppendReplicated` calls and optionally in a record envelope. Default 0 = "no term."
- Replay can ignore or enforce term semantics as needed.

---

## 12. Summary

### 12.1 Component responsibility matrix

| Component | Responsibility |
|-----------|----------------|
| **NuRaft** | Leader election, term, heartbeats, cluster membership; metadata-only state machine and log. |
| **mwal** | WAL: Write, Recover, NewWalIterator, GetLiveWalFiles; **AppendReplicated** (with locking, sequence, term); **WriteRecordCallback** (leader payload notification); **TruncateAfter** (log reconciliation). |
| **ReplicationManager** | Leader/ISR/OSR logic; batching; sync replicate to ISR, async to OSR; ISR selection and maintenance; `committed_lsn_` tracking; WAL retention (`SetMinLogNumberToKeep` based on replica progress); leader failover coordination. |
| **gRPC Replication service** | Replicate (with `prev_lsn` gap detection), StreamWAL, ReportProgress; calls mwal on replica; leader uses clients to send to followers. |

### 12.2 New mwal API summary

| API | Used by | Purpose |
|-----|---------|---------|
| `AppendReplicated(first_seq, count, payload, term)` | Replica | Append a pre-sequenced WAL record; bypasses WriteThread; acquires writer_mu_ directly. |
| `WriteRecordCallback` (in WALOptions) | Leader | Invoked inside Write()/WriteCoalescedBatches() after AddRecord succeeds, under writer_mu_. Provides (first_seq, count, payload) for replication. |
| `TruncateAfter(lsn)` | Replica (failover) | Remove all records with seq > lsn. Used during leader failover to remove divergent tail. |

### 12.3 Suggested directory layout

```
repo/
├── mwal/                          # Existing WAL library + new APIs
│   ├── include/mwal/
│   │   ├── db_wal.h               # + AppendReplicated; + TruncateAfter
│   │   ├── options.h              # + WriteRecordCallback
│   │   └── ...
│   ├── src/wal/
│   │   ├── db_wal.cc              # + AppendReplicated impl; + TruncateAfter impl
│   │   └── ...
│   └── docs/
│       ├── WAL_DESIGN.md
│       └── REPLICATION_DESIGN.md  # This document
│
├── replication/                   # Raft + ISR/OSR
│   ├── include/replication/
│   │   ├── replication_manager.h  # committed_lsn_, WAL retention, failover
│   │   ├── options.h
│   │   └── raft_callback.h
│   └── src/
│       ├── replication_manager.cc
│       ├── isr_selector.cc
│       ├── raft_integration.cc
│       └── ...
│
├── grpc/
│   ├── proto/replication.proto    # Replicate (with prev_lsn), StreamWAL, ReportProgress
│   └── replication_service_impl.cc
│
├── third_party/                   # NuRaft, gRPC, protobuf
│   └── nuraft/
│
└── examples/
    └── replicated_wal_node.cc     # Single-node entry: NuRaft + mwal + ReplicationManager + gRPC
```

### 12.4 Design decisions recap

- **Raft for election only;** WAL data is replicated via gRPC with ISR/OSR, not via Raft log.
- **mwal** gets **AppendReplicated** (bypasses WriteThread, acquires writer_mu_ directly, takes first_seq + count + term), **WriteRecordCallback** (invoked under writer_mu_ after AddRecord), and **TruncateAfter** (for log reconciliation on failover).
- **Committed vs. persisted** tracked separately: `last_sequence_` in mwal (persisted), `committed_lsn_` in ReplicationManager (committed). Application only applies records ≤ committed_lsn_.
- **Gap detection:** `prev_lsn` in ReplicateRequest lets replicas detect gaps (need catch-up) and divergence (need truncation).
- **Leader failover:** New leader's log is source of truth. Divergent tails on replicas (and the old leader) are truncated via `TruncateAfter`. No committed data is lost.
- **WAL retention:** Leader retains WAL files based on slowest replica's LSN via `SetMinLogNumberToKeep`. If a replica falls too far behind (`max_replica_lag_before_snapshot`), it requires snapshot-based recovery (out of scope for mwal).
- **ReportProgress is required** (not optional) so the leader can reliably compute replica lag even during low-traffic periods.
- **Each node has its own `wal_dir`** with its own `LOCK` file; nodes never share a WAL directory.
- **Log rotation on replicas is independent** — file boundaries may differ from the leader, which is fine for correctness.
- **State store** and "apply to state" are **out of scope**; this design is limited to WAL + Raft + ISR/OSR replication.
- **Client routing:** Non-leaders **reject** writes and return leader address; clients send writes **only to the leader** after redirect.
