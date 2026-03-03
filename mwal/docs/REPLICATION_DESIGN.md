# mwal + NuRaft + ISR/OSR Replication — Design Document

This document describes the high-level design for running **mwal** (Write-Ahead Log) with **Raft consensus** for leader election and **ISR/OSR** (In-Sync / Out-of-Sync Replica) replication. It covers scope, architecture, APIs, data flows, client routing, and configuration.

**Document map**

| Section | Content |
|---------|---------|
| **1. Scope & goals** | What is in scope (WAL + Raft + ISR/OSR); what is out of scope (state store) |
| **2. Architecture overview** | Layers, components, repo layout |
| **3. NuRaft integration** | Leader election only; metadata state machine; no WAL data in Raft |
| **4. ISR/OSR replication** | Sync to ISR, async to OSR; min in-sync replicas; selection strategy |
| **5. mwal API additions** | AppendReplicated; leader payload callback / GetLastWrittenRecord |
| **6. gRPC replication protocol** | Replicate, StreamWAL, ReportProgress; request/response shapes |
| **7. Data flows** | Leader write path; replica receive path; catch-up (StreamWAL) |
| **8. Client request routing** | Redirect vs forward; recommended: reject + redirect to leader |
| **9. Snapshot & catch-up** | When used; snapshot + WAL stream; replica recovery |
| **10. Configuration** | Raft, replication, ISR parameters |
| **11. Summary** | Component responsibility matrix; directory layout |

---

## 1. Scope & Goals

### 1.1 In scope

- **mwal**: WAL library — append-only log, recovery, iterator (existing behaviour plus new APIs below).
- **NuRaft**: Leader election, term management, cluster membership, heartbeats. Raft log carries **cluster metadata only** (e.g. ISR set changes), not WAL data.
- **ISR (In-Sync Replicas)**: A subset of followers that receive **synchronous** replication; a write is committed only when the leader and all ISR members have persisted it (or when enough ISR members ack to meet `min_insync_replicas`).
- **OSR (Out-of-Sync Replicas)**: Remaining followers receive **asynchronous** replication; no impact on write commit or availability.
- **Client routing**: When a write lands on a non-leader, the node rejects and returns the leader's address (redirect); clients send writes to the leader.

### 1.2 Out of scope

- **State store**: The structure that holds current id→vector (or any key-value) state and applies committed WAL entries. Not part of this design; a separate layer consumes mwal for durability.
- **Vector index** (e.g. HNSW/IVF): Built and maintained outside this replication layer.

### 1.3 Goals

- Run WAL with Raft and ISR/OSR in a single repo (self-contained).
- Leader is the only node that accepts and processes client writes; replicas only receive replication traffic.
- Durability: writes are committed only when replicated to at least `min_insync_replicas` (ISR).
- Availability: if ISR shrinks below `min_insync_replicas`, writes are rejected until ISR is restored or config is relaxed.

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
│  • ReplicationManager (ISR/OSR, sync/async fan-out)                        │
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
│  • WriteRecordCallback /  │ │                    │ │                          │
│    GetLastWrittenRecord   │ │                    │ │                          │
│    (NEW – for leader)     │ │                    │ │                          │
└───────────────────────────┘ └───────────────────┘ └─────────────────────────┘
```

### 2.2 What runs per node

Each node runs one process that includes:

| Component | Role |
|-----------|------|
| **NuRaft** | Leader election, term, heartbeats, cluster config. Does not replicate WAL data. |
| **mwal (DBWal)** | Local WAL: leader appends via Write(); replicas append via AppendReplicated(); Recover, NewWalIterator for catch-up. |
| **ReplicationManager** | Uses NuRaft for "am I leader?" and cluster list; uses mwal for local log; uses gRPC to send/receive WAL records. Implements ISR (sync) and OSR (async). |
| **gRPC server** | Exposes Replicate, StreamWAL, ReportProgress. Receives WAL from leader; calls mwal's new APIs on replicas. |
| **gRPC clients** | Leader uses these to send Replicate / StreamWAL to followers (ISR and OSR). |

### 2.3 Example: 5 nodes A, B, C, D, E

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

### 3.4 Term in the WAL (optional)

- **Not required for correctness.** Replicas reject stale-term requests at the Raft/replication layer before calling mwal.
- **Optional in mwal:** Add an optional term field to the log for audit/debugging or stricter replay semantics. If added, keep it optional so mwal remains a generic WAL.

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

### 4.3 Commit rule

- A write is **committed** when the leader has persisted it and received ack from enough ISR members such that the effective in-sync set meets `min_insync_replicas` (e.g. all current ISR members ack, or a defined subset). Exact policy (all ISR vs N of ISR) is a configuration choice; typically "all ISR" for simplicity.

---

## 5. mwal API Additions

### 5.1 Replica: AppendReplicated(lsn, payload [, term])

- **Purpose:** Append a WAL record with a **given** sequence number (and optional term). Used when a replica receives a Replicate RPC: it persists the same bytes the leader wrote, without mwal assigning a new sequence.
- **Signature (conceptual):** `Status AppendReplicated(SequenceNumber lsn, const Slice& payload, std::optional<uint64_t> term = std::nullopt);`
- **Semantics:** One record appended; `last_sequence_` (or equivalent) is updated to at least `lsn`. Format of `payload` must match mwal's normal WAL record format (e.g. 1-byte compression prefix + WriteBatch bytes) so recovery and iterators work unchanged.

### 5.2 Leader: Exact bytes written

- **Purpose:** The replication layer must send the **exact** bytes that were written to the WAL to ISR/OSR. Two options:
  - **Option A — Callback:** A `WriteRecordCallback` (or similar) in WALOptions invoked after each record is written, with `(sequence_number, slice)` of the record bytes.
  - **Option B — GetLastWrittenRecord:** After `Write()` returns, a method that returns the last written record's (sequence, slice) so the replication layer can send it.
- **Recommendation:** Callback keeps mwal agnostic of replication; GetLastWrittenRecord is simpler if a single writer is guaranteed. Choose one and document it.

### 5.3 Existing mwal APIs used as-is

- **Write(options, batch)** — Leader appends; replication layer uses callback or GetLastWrittenRecord to get payload.
- **NewWalIterator(start_seq)**, **GetLiveWalFiles()** — Leader streams WAL for catch-up (StreamWAL).
- **Recover(callback)** — Replay local WAL on restart (usage for "apply to state" is out of scope here).

---

## 6. gRPC Replication Protocol

### 6.1 Services and RPCs

- **Replicate(ReplicateRequest) → ReplicateResponse**  
  Leader sends one or more WAL entries (lsn + payload). Replica appends each via `AppendReplicated(lsn, payload)` and returns success and `last_persisted_lsn`. Used for both ISR (sync, wait for response) and OSR (async, fire-and-forget or background).

- **StreamWAL(StreamWALRequest) → stream WALChunk**  
  Leader streams WAL entries from `start_lsn` (and optionally up to `end_lsn`). Used for replica catch-up after a snapshot or when far behind.

- **ReportProgress(ProgressReport) → ProgressAck**  
  Optional: follower reports its persisted/applied LSN for leader to compute lag and maintain ISR (evict/promote).

### 6.2 Key message fields

- **ReplicateRequest:** `term`, `leader_commit`, `repeated ReplicateEntry` (each: `lsn`, `payload`).
- **ReplicateResponse:** `success`, `message`, `last_persisted_lsn`, `term_seen`.
- Replicas reject requests when `term` is less than their current Raft term (return failure and `term_seen` so leader can step down).

### 6.3 Payload format

- **payload** = exact bytes of one mwal WAL record (e.g. 1-byte compression prefix + WriteBatch bytes). Same format as written by mwal on the leader so replicas can append and recover identically.

---

## 7. Data Flows

### 7.1 Leader write path

1. Client sends write to **leader** (after redirect if needed).
2. ReplicationManager checks: am I leader? |ISR| ≥ min_insync_replicas? If not, reject.
3. ReplicationManager calls **mwal::Write(options, batch)**. mwal appends and assigns sequence; callback or GetLastWrittenRecord provides (lsn, payload).
4. ReplicationManager adds (lsn, payload) to a pending batch. When batch is full (count/size) or timeout:
   - Send **Replicate(term, entries)** to all ISR members in parallel; wait for acks.
   - If enough ISR acks: treat as committed (advance commit index).
   - Send same Replicate to OSR asynchronously (no wait).
5. Reply to client with success (or failure if ISR acks insufficient).

### 7.2 Replica receive path

1. ReplicationServiceImpl receives **Replicate(term, entries)**.
2. If `term` < my Raft term: return failure, `term_seen` = my term.
3. For each entry: call **mwal::AppendReplicated(lsn, payload)**. On first failure, return failure and `last_persisted_lsn`.
4. Return success and `last_persisted_lsn` = last entry's lsn.

### 7.3 Catch-up (StreamWAL)

1. Follower (e.g. new or recovered) is behind. It requests **StreamWAL(start_lsn, end_lsn)** from the leader.
2. Leader uses **NewWalIterator(start_lsn)** (and current WAL files) to iterate records; streams them in chunks (e.g. WALChunk with multiple entries).
3. Follower for each chunk calls **AppendReplicated(lsn, payload)** for each entry until stream ends.
4. Follower is then caught up and can join OSR; ISR maintenance can later promote it to ISR when healthy and within lag.

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

## 9. Snapshot & Catch-Up

### 9.1 When used

- A **new node** or **recovered node** that is far behind (e.g. missing many LSNs). Sending only WAL from 0 would be slow; so use **full snapshot** of the replicated state (out of scope here: "state" is not in this design) plus **WAL stream** from snapshot LSN to current.

### 9.2 In-scope behaviour

- **StreamWAL(start_lsn, end_lsn):** Leader uses mwal's **NewWalIterator(start_lsn)** and **GetLiveWalFiles()** to stream WAL records to the follower. Follower calls **AppendReplicated(lsn, payload)** for each entry. This is the WAL catch-up path; snapshot creation/transfer (if any) is defined by the layer that owns "state" and is out of scope for this doc.

---

## 10. Configuration

### 10.1 Raft (NuRaft)

| Parameter | Example | Meaning |
|-----------|---------|---------|
| election_timeout_lower_bound_ms | 200 | Lower bound for election timeout (randomized per node). |
| election_timeout_upper_bound_ms | 400 | Upper bound for election timeout. |
| heart_beat_interval_ms | 75 | Leader heartbeat interval. |

### 10.2 Replication (ReplicationManager)

| Parameter | Example | Meaning |
|-----------|---------|---------|
| min_insync_replicas | 2 | Minimum ISR size for writes to be accepted. |
| replication_timeout_ms | 150 | Max time to wait for ISR ack before evicting/shrinking ISR. |
| isr_check_interval_ms | 1000 | Interval for ISR maintenance (evict/promote). |
| max_lag_entries | 5000 | Replica evicted from ISR if lag exceeds this. |
| replica_timeout_ms | 3000 | No ack for this long → evict from ISR. |
| batch_max_entries | 100 | Max WAL entries per Replicate RPC. |
| batch_max_bytes | 1048576 | Max bytes per Replicate RPC (e.g. 1 MiB). |

### 10.3 Optional: term in mwal log

- If supported: optional field in record header or batch prefix; 0 or "unset" means no term. Replay can ignore or enforce term semantics as needed.

---

## 11. Summary

### 11.1 Component responsibility matrix

| Component | Responsibility |
|-----------|----------------|
| **NuRaft** | Leader election, term, heartbeats, cluster membership; metadata-only state machine and log. |
| **mwal** | WAL: Write, Recover, NewWalIterator, GetLiveWalFiles; **AppendReplicated**; **WriteRecordCallback / GetLastWrittenRecord**. |
| **ReplicationManager** | Leader/ISR/OSR logic; batching; sync replicate to ISR, async to OSR; ISR selection and maintenance; commit index. |
| **gRPC Replication service** | Replicate, StreamWAL, ReportProgress; calls mwal on replica; leader uses clients to send to followers. |

### 11.2 Suggested directory layout

```
repo/
├── mwal/                          # Existing WAL library + new APIs
│   ├── include/mwal/
│   │   ├── db_wal.h               # + AppendReplicated; + callback or GetLastWrittenRecord
│   │   ├── options.h
│   │   └── ...
│   ├── src/wal/
│   │   ├── db_wal.cc
│   │   └── ...
│   └── docs/
│       ├── WAL_DESIGN.md
│       └── REPLICATION_DESIGN.md   # This document
│
├── replication/                   # Raft + ISR/OSR
│   ├── include/replication/
│   │   ├── replication_manager.h
│   │   ├── options.h
│   │   └── raft_callback.h
│   └── src/
│       ├── replication_manager.cc
│       ├── isr_selector.cc
│       ├── raft_integration.cc
│       └── ...
│
├── grpc/
│   ├── proto/replication.proto
│   └── replication_service_impl.cc
│
├── third_party/                   # NuRaft, gRPC, protobuf
│   └── nuraft/
│
└── examples/
    └── replicated_wal_node.cc     # Single-node entry: NuRaft + mwal + ReplicationManager + gRPC
```

### 11.3 Design decisions recap

- **Raft for election only;** WAL data is replicated via gRPC with ISR/OSR, not via Raft log.
- **mwal** gets **AppendReplicated** and a way for the leader to obtain the exact written record bytes; no term in log unless opted in.
- **State store** and "apply to state" are **out of scope**; this design is limited to WAL + Raft + ISR/OSR replication.
- **Client routing:** Non-leaders **reject** writes and return leader address; clients send writes **only to the leader** after redirect.
