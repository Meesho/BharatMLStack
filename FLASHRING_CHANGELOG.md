# FlashRing: Engineering Changelog (Jan 6 - Apr 3, 2026)

## Executive Summary

Over the past two months, FlashRing underwent a deep performance optimization journey focused on reducing I/O latency, eliminating lock contention, and modernizing the storage engine internals. The work spanned **89 commits across 23 active days**, touching **56 files** with a net **+10,945 lines** of code. The effort progressed through five major phases: metrics infrastructure, I/O path optimization, io_uring integration, lockless data structures, and a final architecture cleanup.

---

## Change Profile

| Key Subsystems Touched | fs, iouring, index, memtable, shard_cache, metrics, cache |
| Branches | `flashring-externalize`, `flashring-externalize-lockless`, `flashring-externalize-cleanup` |

### Commit Activity Heatmap

```
Jan 09  ███            (3)
Jan 12  ██             (2)
Jan 13  ████           (4)
Jan 15  █              (1)
Jan 19  ██             (2)
Jan 21  █              (1)
Jan 22  ███            (3)
        --- gap (Jan 23 - Feb 9) ---
Feb 10  ██████         (6)
Feb 11  ████████       (8)
Feb 12  ███████        (7)
Feb 13  ████           (4)
Feb 17  █              (1)
Feb 20  █              (1)
Feb 22  ███            (3)
Feb 23  ███            (3)
Feb 24  ███            (3)
Feb 25  ████████       (8)
Feb 26  ███            (3)
Feb 27  ███            (3)
Mar 02  █████████      (9)
Mar 03  █████████      (9)
Mar 04  ████           (4)
Mar 05  █              (1)
```

Peak activity was during the io_uring integration (Feb 10-13) and the io_uring tuning/SQPOLL phase (Mar 2-4).

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      WrapCache (pkg/cache)                      │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐      ┌──────────┐  │
│   │ Shard 0  │  │ Shard 1  │  │ Shard 2  │ ...  │ Shard N  │  │
│   └────┬─────┘  └────┬─────┘  └────┬─────┘      └────┬─────┘  │
│        │              │              │                 │        │
│   ┌────▼─────┐  ┌────▼─────┐  ┌────▼─────┐     ┌────▼─────┐  │
│   │ Memtable │  │ Memtable │  │ Memtable │     │ Memtable │  │
│   │(Lock-free│  │(Lock-free│  │(Lock-free│     │(Lock-free│  │
│   │ bump     │  │ bump     │  │ bump     │     │ bump     │  │
│   │ alloc)   │  │ alloc)   │  │ alloc)   │     │ alloc)   │  │
│   └────┬─────┘  └────┬─────┘  └────┬─────┘     └────┬─────┘  │
│        │              │              │                 │        │
│   ┌────▼─────────────────────────────▼─────────────────▼─────┐ │
│   │               Index (internal/index)                      │ │
│   │    map[uint64]int  +  RingBuffer  +  DeleteManager        │ │
│   └────────────────────────┬──────────────────────────────────┘ │
│                            │                                    │
│   ┌────────────────────────▼──────────────────────────────────┐ │
│   │           WrapAppendFile (internal/fs)                     │ │
│   │  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐  │ │
│   │  │ WriteFd     │  │ ReadFd       │  │ TrimHead        │  │ │
│   │  │ (pwrite/    │  │ (pread/      │  │ (fallocate      │  │ │
│   │  │  io_uring)  │  │  io_uring)   │  │  PUNCH_HOLE)    │  │ │
│   │  └──────┬──────┘  └──────┬───────┘  └─────────────────┘  │ │
│   └─────────┼────────────────┼────────────────────────────────┘ │
│             │                │                                  │
│   ┌─────────▼────────────────▼──────────────────────────────┐   │
│   │           io_uring (internal/iouring)                     │   │
│   │  ┌────────────────┐    ┌────────────────────────────┐    │   │
│   │  │ IoUringWriter  │    │ BatchIoUringReader          │    │   │
│   │  │ (sync writes)  │    │ (async batched reads)       │    │   │
│   │  │                │    │  submitLoop + completeLoop  │    │   │
│   │  └────────────────┘    └────────────────────────────┘    │   │
│   │         1,086 lines  |  Raw syscall-based, no CGO       │   │
│   └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                               │
                        ┌──────▼──────┐
                        │  NVMe/SSD   │
                        │  (ext4/xfs) │
                        └─────────────┘
```

---

## Phase 1: Metrics Infrastructure & Observability (Jan 9 - Jan 22)

### Problem Statement

FlashRing had no structured observability. Performance profiling relied on ad-hoc logging, making it impossible to identify bottlenecks in production vs synthetic load.

### Timeline

| Date | Commit | Change | Rationale |
|------|--------|--------|-----------|
| Jan 9 | `b8ffbaf` | Guard lockless functions behind config flag | Lockless code paths were starting even when disabled |
| Jan 9 | `bfd1337` | Disable rewrite logic | `shouldReWrite` was triggering unnecessary write amplification |
| Jan 9 | `90cae0f` | Migrate shard cache metric maps to `sync.Map` | Regular maps panicked under concurrent metric updates |
| Jan 12 | `19887a9` | Build full metrics package (+1,056 lines) | Console, CSV, and StatsD loggers; metric tags; averaging |
| Jan 12 | `0d014c4` | Move metrics from `internal/` to `pkg/` | Needed by both internal components and test harnesses |
| Jan 13 | `8a56a25` | Set full sampling rate | 10% sampling was hiding tail latencies |
| Jan 13-15 | Multiple | Fix metrics, add fine-grained stats | StatsD tag formatting bugs; missing shard-level metrics |
| Jan 19 | `54ddd21` | Fix stats; adjust filesize multiplier in test plans | Prepopulate metrics were inflating averages |
| Jan 19 | `f5a1d6c` | Remove sync.Pool changes | Sync.Pool was causing GC churn with mmap-backed pages |
| Jan 21 | `1788d64` | Grid search fixes | Estimator/predictor tuning for hit-rate prediction |
| Jan 22 | `97da61e` | Clear files at mountpoint on start | Stale files from previous runs caused offset misalignment |
| Jan 22 | `56a5080` | Fix delete manager; add file stats | Delete manager was not properly tracking punch-hole boundaries |
| Jan 22 | `659d8e2` | Fix filewrite error after first punch hole | After first `TrimHead`, write offset reset was incorrect |

### Key Insight

The metrics infrastructure revealed that **prepopulate phase metrics were mixed with runtime metrics**, inflating average latencies and hiding the true production performance profile.

### Data Flow: Metrics Architecture

```
  ┌──────────┐     ┌──────────┐     ┌──────────┐
  │ cache.go │     │ shard_   │     │ wrap_    │
  │ Get/Put  │────▶│ cache.go │────▶│ file.go  │
  └────┬─────┘     └────┬─────┘     └────┬─────┘
       │                │                 │
       ▼                ▼                 ▼
  ┌─────────────────────────────────────────────┐
  │          pkg/metrics/statsd_logger          │
  │  ┌─────────┐ ┌──────────┐ ┌──────────────┐ │
  │  │ Timing  │ │  Gauge   │ │   Incr       │ │
  │  │ (p50/   │ │ (data    │ │ (hit/miss/   │ │
  │  │  p99)   │ │  length) │ │  punch_hole) │ │
  │  └────┬────┘ └────┬─────┘ └──────┬───────┘ │
  │       └───────────┼──────────────┘          │
  │                   ▼                         │
  │            DataDog StatsD                   │
  └─────────────────────────────────────────────┘
```

**Metrics tracked after this phase:**
- `KEY_GET_LATENCY`, `KEY_PUT_LATENCY` (per shard)
- `KEY_PREAD_LATENCY`, `KEY_PWRITE_LATENCY` (filesystem-level)
- `KEY_PUNCH_HOLE_COUNT`, `KEY_TRIM_HEAD_LATENCY`
- `KEY_DATA_LENGTH` (value sizes)
- Hit rate, miss rate, expired count

---

## Phase 2: I/O Path Optimization (Feb 10 - Feb 11)

### Problem Statement

Production pread latency was **~10ms** at just 1% traffic, while synthetic loads showed **<1ms**. CPU and disk were not saturated. Investigation revealed the root cause: **O_DIRECT on cloud persistent disks**.

### Root Cause Analysis

```
         Synthetic Load                    Production
        ┌─────────────┐               ┌─────────────┐
        │  Get(key)    │               │  Get(key)    │
        └──────┬──────┘                └──────┬──────┘
               │                              │
        ┌──────▼──────┐               ┌──────▼──────┐
        │  Memtable   │               │  Memtable   │
        │  (HIT ~95%) │               │  (MISS ~80%)│
        └──────┬──────┘                └──────┬──────┘
               │ (rare)                       │ (frequent)
        ┌──────▼──────┐               ┌──────▼──────┐
        │  Disk Read  │               │  Disk Read  │
        │  O_DIRECT   │               │  O_DIRECT   │
        │  <1ms (SSD) │               │  2-10ms     │
        └─────────────┘               │  (Cloud PD) │
                                      └─────────────┘
```

**Why O_DIRECT hurt on cloud persistent disks:**
- O_DIRECT bypasses the kernel page cache
- Cloud persistent disks have **2-10ms base latency** (network-attached storage)
- Without page cache, every read hits the network-attached disk
- Synthetic loads had high memtable hit rates, masking the disk latency

### Timeline

| Date | Commit | Change | Rationale | Result |
|------|--------|--------|-----------|--------|
| Feb 10 | `e92786d` | Add direct StatsD metrics for read/write latency | Needed per-operation latency visibility | Confirmed pread was the bottleneck |
| Feb 10 | `903165d` | Try lockless paths | Hypothesis: lock contention caused latency | Lockless alone did not fix it |
| Feb 10 | `b0a8e47` | Return error on trim needed | TrimHead during reads caused stalls | Prevented blocking reads |
| Feb 11 | `f16d4a6` | Add pread/pwrite latency instrumentation | Isolate syscall latency from application overhead | Confirmed syscall-level latency was high |
| Feb 11 | `dcac7f9` | Remove DSYNC from pwrite | Data sync on every write was unnecessary overhead | Reduced write latency |
| Feb 11 | `0b27e24` | **Remove O_DIRECT from write path** | O_DIRECT + cloud PD = high latency | Write latency improved |
| Feb 11 | `cbc6d3d` | Add memtable chunking on flush | Large single-write flushes blocked the write FD | Reduced flush-induced stalls |
| Feb 11 | `e3abf42` | Reduce chunk size | Initial chunk size too large | Better write distribution |
| Feb 11 | `710c80e` | Track time wasted in lock | Quantify lock contention | Confirmed lock wait was significant |

### Key Decisions

1. **Removed DSYNC from pwrite** - Data integrity was handled at a higher level (fsync on checkpoint), so per-write DSYNC was redundant overhead.

2. **Removed O_DIRECT from write path** - Reads still used O_DIRECT for alignment guarantees, but writes benefited from page cache buffering.

3. **Memtable chunking** - Instead of flushing the entire memtable in one large write, it was split into 4KB-aligned chunks. This prevented a single large pwrite from monopolizing the write file descriptor.

---

## Phase 3: io_uring Integration (Feb 12 - Feb 17)

### Problem Statement

Even after removing O_DIRECT from writes, read latency remained high. The read path was serialized - each `pread` syscall blocked until the kernel returned data. For cloud persistent disks with 2-10ms latency, this meant each read was a full round-trip.

### Approach: Batched Asynchronous I/O via io_uring

```
  BEFORE (synchronous pread)              AFTER (batched io_uring)
  
  Get(k1) ─── pread ──── 5ms ───▶        Get(k1) ──┐
  Get(k2) ─── pread ──── 5ms ───▶        Get(k2) ──┤ collect
  Get(k3) ─── pread ──── 5ms ───▶        Get(k3) ──┤ ~500μs
  Get(k4) ─── pread ──── 5ms ───▶        Get(k4) ──┘
                                                    │
  Total: 20ms (serial)                    ┌─────────▼─────────┐
                                          │  io_uring_enter   │
                                          │  (submit 4 SQEs)  │
                                          └─────────┬─────────┘
                                                    │ ~5ms
                                          ┌─────────▼─────────┐
                                          │  4 CQEs complete  │
                                          │  dispatch results  │
                                          └───────────────────┘
                                          
                                          Total: ~5.5ms (parallel)
```

### Timeline

| Date | Commit | Change | Rationale | Result |
|------|--------|--------|-----------|--------|
| Feb 12 | `f1d3b26` | **Implement io_uring** (+755 lines) | Async I/O to batch disk reads | Working prototype, raw syscall based |
| Feb 12 | `98362c2` | Move RLock position into index | Reduce lock hold time on reads | Decreased lock contention |
| Feb 12 | `0afc603` | Memtable chunk size = 16*4KB | Tuning chunk size for write throughput | Better write batching |
| Feb 12 | `fbaa622` | Fix mutex used for RLock | Wrong mutex was being locked | Fixed deadlock potential |
| Feb 12 | `5036e0b` | **Implement io_uring batching** (+339 lines) | Batch reader: collect requests, submit batch, dispatch results | Functional batched reads |
| Feb 13 | `8c510a3` | Wait 500μs for batch collection | Trade latency for batch size | ~20 reads per batch at 20k RPS |
| Feb 13 | `f2126ff` | io_uring write batching | Apply same batching to writes | Write latency improved |
| Feb 13 | `a9cefea` | Fix io_uring write path | Off-by-one in buffer slicing | Corrected data corruption |
| Feb 13 | `737e651` | Track chunked pread/pwrite latency | Measure io_uring vs syscall latency | io_uring was faster |
| Feb 17 | `399d797` | io_uring no-wait fixes | Avoid blocking on SQ full | Reduced tail latency |

### io_uring Implementation Details

The implementation was built from raw syscalls (no CGO, no external library):

```
┌─────────────────────────────────────────────────────┐
│                  IoUring (1,086 lines)               │
│                                                      │
│  ┌──────────────────────────────────────────────┐   │
│  │  Ring Setup (raw syscalls)                    │   │
│  │  - io_uring_setup (SYS_425)                   │   │
│  │  - mmap SQ/CQ rings                          │   │
│  │  - Optional SQPOLL mode                      │   │
│  └──────────────────────────────────────────────┘   │
│                                                      │
│  ┌──────────────────────────────────────────────┐   │
│  │  BatchIoUringReader                           │   │
│  │                                               │   │
│  │  reqCh ──▶ submitLoop ──▶ io_uring_enter      │   │
│  │                              │                │   │
│  │           completeLoop ◀────┘                 │   │
│  │              │                                │   │
│  │           done chan ──▶ caller                 │   │
│  └──────────────────────────────────────────────┘   │
│                                                      │
│  ┌──────────────────────────────────────────────┐   │
│  │  IoUringWriter                                │   │
│  │  - Synchronous batch writes                   │   │
│  │  - Sub-batch to fit ring depth                │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

**Key design decisions:**
- **Pure Go + raw syscalls** - No CGO dependency, no `go:linkname` hacks. Compatible with Go 1.24+.
- **Separate submit and complete goroutines** - The submit path is never blocked by CQE draining. New requests can be collected while previous results are being dispatched.
- **Request pooling** - `sync.Pool` for `batchReadRequest` objects to reduce GC pressure.
- **Sentinel NOP** - On shutdown, a NOP with `userData = ^uint64(0)` is submitted to unblock the completion goroutine cleanly.

---

## Phase 4: Lock Contention & Lockless Experiments (Feb 20 - Mar 3)

### Problem Statement

Metrics showed that **lock wait time was a significant portion of Get/Put latency**. The shard-level `sync.RWMutex` serialized all reads within a shard, and writes (flush, put, delete) held exclusive locks that blocked concurrent reads.

### Approach Timeline

This was the most iterative phase, with multiple approaches tried and reverted:

```
Timeline of Lock/Map Experiments:

Feb 20  ┤ Remove custom metrics, use StatsD only (-1,569 lines)
        │  Why: Metrics package was too heavy; StatsD was enough
        │
Feb 22  ┤ Parallelize io_urings (multiple rings)
        │  Why: Single ring was a serialization point
        │
Feb 23  ┤ Disable metrics by default; tune wait times
        │  Why: Metric collection itself added latency
        │
Feb 24  ┤ 4 io_urings → 2 io_urings
        │  Why: Diminishing returns beyond 2 rings
        │
Feb 25  ┤ Cleanup: remove lockless code (first attempt failed)
        │  Why: Lockless without proper data structures caused crashes
        │  Result: Concurrent map read/write panics
        │
Feb 26  ┤ Try xsync.Map instead of regular sync.Map
        │  Why: sync.Map has poor write performance; xsync.Map is sharded
        │  Result: Mixed - less contention but higher memory
        │
Feb 27  ┤ Lock-free memtable (atomic bump allocator)
        │  Why: Memtable lock was the biggest contention point
        │  ┤ v1 lockless implementation
        │  Result: CPU cycling issues (Gosched spin loop)
        │
Mar 2   ┤ Tune io_uring ring count (1→2→1→2→1)
        │  Why: Finding optimal ring count for workload
        │
Mar 2   ┤ Revert xsync.Map → Go native map
        │  Why: xsync.Map overhead not worth it with proper locking
        │
Mar 3   ┤ Write-lock on index only (not entire shard)
        │  Why: Minimize lock scope
        │
Mar 3   ┤ Implement SQPOLL mode for io_uring
        │  Why: Eliminate io_uring_enter syscall overhead
        │  ┤ Single SQPOLL ring → Two SQPOLL rings
        │
Mar 3   ┤ Separate io_uring submit and collect goroutines
        │  Why: Submit was blocking on CQE drain
        │  Result: Significant latency improvement
```

### Concurrent Map Problem (Feb 25 - Feb 27)

A critical issue was discovered: the `Index` struct used a plain Go `map[uint64]int` for key-to-slot mapping. When locks were removed, this caused panics:

```
PROBLEM:
                              map[uint64]int (NOT thread-safe)
                             ┌────────────────┐
  GetLL() goroutine 1  ────▶ │ i.rm[hlo]      │ ◀──── PutLL() goroutine
  GetLL() goroutine 2  ────▶ │ (concurrent    │ ◀──── DeleteManager
  GetLL() goroutine 3  ────▶ │  read + write) │
                             └────────────────┘
                                    │
                              PANIC: concurrent
                              map read and write

SOLUTIONS TRIED:
  1. sync.Map      → Poor write perf, high memory
  2. xsync.Map     → Better, but still overhead
  3. Native map    → Requires external lock ← CHOSEN
     + fine-grained locking (lock index only, not shard)
```

### Lockless Memtable (Feb 27)

The memtable was redesigned with an atomic bump allocator:

```
BEFORE (locked allocation):                AFTER (lock-free):

  Put(key, val)                             Put(key, val)
       │                                         │
  ┌────▼────┐                              ┌────▼────────────┐
  │  Lock() │ ◀── contention               │ atomic.AddInt64  │
  │         │     point                     │ (currentOffset,  │
  │ offset  │                               │  size)           │
  │  += sz  │                               │                  │
  │         │                               │ CAS retry on     │
  │Unlock() │                               │ conflict         │
  └─────────┘                               └─────────────────┘

  Problem: N goroutines                     Benefit: No lock
  serialize on Lock()                       contention.
                                            Trade-off: Need
                                            inflight WaitGroup
                                            for flush coord.
```

**Result:** The atomic bump allocator worked but introduced a new problem - when the memtable was full, the Put path fell into a `runtime.Gosched()` spin loop waiting for the flush to complete. This caused CPU cycling (high → ~50% → high).

### SQPOLL Mode (Mar 3)

SQPOLL eliminates the `io_uring_enter` syscall on the submission path by having a kernel thread poll the SQ ring:

```
NORMAL io_uring:                    SQPOLL io_uring:

  User           Kernel              User           Kernel
  ┌────┐        ┌────┐              ┌────┐        ┌────┐
  │Prep│        │    │              │Prep│        │Poll│
  │SQE │───────▶│    │              │SQE │        │thrd│
  │    │ enter  │exec│              │    │  tail++ │    │
  │    │◀───────│    │              │    │        │exec│
  └────┘ return └────┘              └────┘        └────┘
  
  Overhead:                          Overhead:
  1 syscall per                      0 syscalls
  submit                             (kernel polls)
  
  Trade-off:                         Trade-off:
  None                               1 kernel thread
                                     per ring (CPU cost)
```

---

## Phase 5: Architecture Cleanup (Mar 4 - Mar 5)

### Problem Statement

After weeks of experimentation, the codebase had accumulated:
- Three index implementations (`indices/`, `indicesV2/`, `indicesV3/`)
- Dead code from abandoned experiments (YCSB benchmarks, SIMD maps, byte slice allocators)
- Duplicated test plans with slightly different configs
- 5,243 lines of code that could be removed

### Changes

| Date | Commit | Change | Impact |
|------|--------|--------|--------|
| Mar 4 | `6efb971` | Fix ring depth based on fio tests | Aligned ring config with disk benchmarks |
| Mar 4 | `c9194e2` | 2 rings capped at 16 max in-flight | Optimal config per fio testing |
| Mar 4 | `2bacba1` | Single ring, 16 depth | Simplified; one ring was enough |
| Mar 4 | `6482591` | 24 queue depth | Final tuning based on production load |
| Mar 5 | `2b1f022` | **Code cleanup** (-5,243 / +1,142 lines) | Massive cleanup |

### Cleanup Details (Mar 5)

```
REMOVED (5,243 lines):
  ├── internal/indices/         (old v1 index)
  │     flat_bitmap, key_index, round_map, rb, encoder...
  ├── internal/indicesV2/       (old v2 index)
  │     index, rb, encoder, system, tests...
  ├── internal/indicesV3/       (superseded by index/)
  │     delete_manager, index_test, rb_bench_test, system...
  ├── internal/allocators/byte_slice_allocator*  (abandoned)
  ├── pkg/ycsb/                 (YCSB benchmarks + SIMD map)
  │     simdmap/, bazel_workspace/, ycsb_bench_test...
  └── internal/fs/iouring_wrapper.go  (moved to iouring/)

ADDED / RESTRUCTURED (1,142 lines):
  ├── internal/index/           (consolidated single package)
  │     constant, delete_manager, encoder, index, ringbuffer, system
  ├── internal/iouring/         (standalone io_uring package)
  │     iouring, iouring_reader, iouring_writer, iouring_test
  ├── cmd/flashringtest/main.go (unified test runner)
  └── Simplified test plans (DRY)
```

### Memory Profile Insights (Mar 5)

A heap profile taken during the cleanup phase revealed:

```
Heap Allocation Breakdown:
┌────────────────────────────────────────────────────────┐
│                                                        │
│  ████████████████████████████████████████  78.36%       │
│  NewRingBuffer                                         │
│  ~1.86 GB (50M slots across shards)                    │
│                                                        │
│  ████████  17.99%                                      │
│  Index.Put (map growth)                                │
│  ~0.43 GB                                              │
│                                                        │
│  ██  3.65%                                             │
│  Other                                                 │
│                                                        │
└────────────────────────────────────────────────────────┘

Root cause: KeysPerShard was set to 67M but only ~1M keys
were stored per shard. ~98.5% of RingBuffer memory was wasted.
Fix: Tune KeysPerShard to match actual usage.
```

---

## Decision Log

### Decisions That Stuck

| Decision | Date | Rationale | Outcome |
|----------|------|-----------|---------|
| StatsD-only metrics | Feb 20 | Custom metrics package was too heavy (-1,569 lines) | Simpler, lower overhead |
| io_uring for reads | Feb 12 | Batch async reads to amortize cloud PD latency | 4x improvement on batched reads |
| Separate submit/complete goroutines | Mar 3 | Decoupled submission from CQE draining | Eliminated head-of-batch delay |
| Native map + fine-grained locking | Mar 3 | Best perf/correctness trade-off | Stable, no panics |
| Consolidated `index/` package | Mar 5 | Three copies of similar code | -4,000 lines, single source of truth |

### Decisions Reverted

| Decision | Date Tried | Date Reverted | Why Reverted |
|----------|-----------|---------------|--------------|
| O_DIRECT on writes | Pre-Jan | Feb 11 | Cloud PD latency; page cache is beneficial for writes |
| DSYNC on pwrite | Pre-Jan | Feb 11 | Redundant with higher-level fsync |
| sync.Pool for pages | Pre-Jan | Jan 19 | GC churn with mmap-backed pages; LeakyPool is correct |
| xsync.Map for index | Feb 26 | Mar 2 | Memory overhead; native map with locking was simpler and sufficient |
| Lockless Put with Gosched spin | Feb 27 | Mar 3 | CPU cycling when memtable full; flush bottleneck |
| 4 io_uring rings | Feb 24 | Feb 24 | Diminishing returns; 1-2 rings were enough |
| SQPOLL with 0 wait time | Mar 3 | Mar 4 | Too aggressive; needed small window for batching |

---

## io_uring Ring Configuration Evolution

The number of io_uring rings and their queue depth was tuned extensively based on fio benchmarks and production metrics:

```
Date         Rings   Depth   Wait       Result
─────────────────────────────────────────────────────
Feb 12       1       64      500μs      Baseline: working
Feb 13       1       64      500μs      + write batching
Feb 22       N       64      -          Parallel rings
Feb 24       4       64      -          Too many; context switching
Feb 24       2       64      -          Better
Mar 2        1       64      4ms        Too long wait
Mar 2        1       64      500μs      Reverted
Mar 2        2       64      500μs      Tried again
Mar 2        1       64      -          Simplified
Mar 3        1       64      100μs      Too short
Mar 3        1       64      50μs       Even shorter
Mar 3        1+SQ    64      0          SQPOLL mode
Mar 3        2+SQ    64      0          Two SQPOLL rings
Mar 3        -       -       -          Separate submit/collect
Mar 4        2       16      -          Based on fio tests
Mar 4        1       16      -          Single ring sufficient
Mar 4        1       24      -          FINAL: optimal depth
```

### fio-Derived Optimal Configuration

```
fio randread benchmark results (NVMe/cloud PD):

Queue Depth    IOPS       Avg Latency    p99 Latency
    1           ~5K         ~0.2ms          0.5ms
    4          ~18K         ~0.2ms          0.6ms
    8          ~32K         ~0.25ms         0.8ms
   16          ~50K         ~0.3ms          1.2ms
   24          ~55K         ~0.4ms          1.8ms    ← chosen
   32          ~56K         ~0.5ms          2.5ms
   64          ~56K         ~1.0ms          5.0ms

Observation: IOPS plateau at depth ~24. Beyond that,
latency increases without throughput gain.
Chosen: 1 ring, 24 depth — maximum throughput with
acceptable tail latency.
```

---

## Metrics Bug Case Studies

### Bug: GetShardTag Slice Bounds Panic (Feb 22)

```go
// BUGGY: shardTags layout is [shard0_tag0, shard0_tag1, shard1_tag0, ...]
// Using shardIdx:shardIdx+2 reads wrong tags and panics at boundaries
func GetShardTag(shardIdx int) []string {
    return shardTags[shardIdx : shardIdx+2]  // WRONG
}

// FIX: multiply by stride
func GetShardTag(shardIdx int) []string {
    idx := shardIdx * 2
    return shardTags[idx : idx+2]
}
```

### Bug: Timing vs Gauge for Data Length (Feb 12)

```go
// BUGGY: Timing treats value as duration (nanoseconds)
metrics.Timing("KEY_DATA_LENGTH", int64(len(data)), tags)
// Grafana showed 0.004 instead of 4096

// FIX: Use Gauge for non-duration values
metrics.Gauge("KEY_DATA_LENGTH", float64(len(data)), tags)
```

### Bug: Tag Append Overwrites (Feb 18)

```go
// BUGGY: GetShardTag returns sub-slice with shared backing array
// append(tags, serviceName) overwrites the next shard's tag
func Count(name string, value int64, tags []string) {
    allTags := append(tags, serviceName)  // overwrites backing array!
    statsd.Count(name, value, allTags, 1)
}

// FIX: Service tag already set via WithTags; remove redundant append
func Count(name string, value int64, tags []string) {
    statsd.Count(name, value, tags, 1)
}
```

---

## shouldReWrite Data Safety Issue (Mar 3)

A subtle data race was identified in the `Get` path:

```go
func (wc *WrapCache) Get(key string) ([]byte, bool, bool) {
    // ...
    wc.shardLocks[shardIdx].RLock()
    keyFound, val, ttl, expired, shouldReWrite = wc.shards[shardIdx].Get(key)
    wc.shardLocks[shardIdx].RUnlock()  // ← val may point to memtable buffer

    if shouldReWrite {
        go wc.Put(key, val, ttl)  // ← RACE: val's backing memory may be
    }                              //   mutated by concurrent writes after
                                   //   RUnlock
}
```

**Fix:** Copy `val` before passing to the goroutine when `shouldReWrite` is true.

---

## Summary of File-Level Changes

```
Files with highest churn (by commit count):

  pkg/cache/cache.go              ████████████████████  ~40 commits
  pkg/metrics/statsd_logger.go    ████████████         ~12 commits
  internal/fs/wrap_file.go        ██████████           ~10 commits
  internal/shard/shard_cache.go   █████████            ~9 commits
  internal/indicesV3/index.go     ████████             ~8 commits
  internal/memtables/memtable.go  ███████              ~7 commits
  internal/fs/batch_iouring.go    █████                ~5 commits
  internal/fs/iouring.go          █████                ~5 commits
  internal/fs/fs.go               ████                 ~4 commits
  internal/indicesV3/delete_mgr   ████                 ~4 commits
```

---

## Lessons Learned

1. **Cloud persistent disks change all assumptions.** O_DIRECT, which is optimal for local NVMe, is harmful on network-attached storage because it bypasses the page cache that hides network latency.

2. **Metrics overhead matters.** The initial custom metrics package with console/CSV/StatsD loggers added measurable latency. Stripping it down to StatsD-only removed 1,569 lines and reduced per-operation overhead.

3. **Go maps are not thread-safe.** This seems obvious, but in a system with multiple locking strategies (locked, lockless, batch), it's easy to introduce a code path that accesses a map without the expected lock. The `concurrent map read and map write` panic was hit multiple times.

4. **io_uring tuning is workload-specific.** The optimal ring count (1-2), queue depth (24), and batch wait time depend on the disk, the request size, and the RPS. fio benchmarks provided the baseline, but production tuning was still needed.

5. **Separate submit and complete goroutines.** The initial io_uring reader had a single loop that submitted SQEs and then drained CQEs. This meant new requests had to wait for the entire previous batch to complete. Splitting into two goroutines eliminated this head-of-batch delay.

6. **Lock scope minimization > lock elimination.** Full lockless was attempted but introduced complexity (Gosched spin loops, data races). Moving the write lock to cover only the index update (not the full shard) achieved most of the benefit with much less complexity.

---

## Current State (Mar 6, 2026)

The codebase on `flashring-externalize-cleanup` represents the production-ready state:

- **1 io_uring ring**, 24 queue depth, with separate submit/complete goroutines for reads
- **io_uring writer** for batched pwrite operations
- **Native Go map** with fine-grained locking (write-lock on index only)
- **Lock-free memtable** with atomic bump allocator
- **Consolidated `index/` package** replacing three previous versions
- **StatsD-only metrics** with shard-level tagging
- **Clean codebase**: -5,243 lines of dead code removed, 56 files across the project

---

## Bug Fix: File Ring Wrap Capacity Loss (Mar 23, 2026)

### Problem Statement

After the first file wrap, `Pwrite`/`PwriteBatch` set `PhysicalWriteOffset = PhysicalStartOffset` instead of `0`. Because `TrimHead` advances `PhysicalStartOffset` by `FilePunchHoleSize` on each punch, the writer skipped the region `[0, PhysicalStartOffset)` on every subsequent cycle. This caused three cascading problems:

1. **Shrinking effective file capacity.** Each wrap-cycle wrote fewer bytes (`MaxFileSize - k*FilePunchHoleSize` on cycle k), wasting a growing dead zone at the start of the file until `PhysicalStartOffset` itself wrapped to 0.

2. **Valid data became unreadable.** The old read validation (`Pread`, `ValidateReadOffset`) only allowed reads from `[max(S, W), MaxFileSize)` in the wrapped case, rejecting the entire `[0, W)` region — including freshly written data. Index entries pointing there received `ErrFileOffsetOutOfRange`, appearing as cache misses.

3. **Accelerated churn.** Less usable space per cycle meant memtables filled the file faster, triggering more frequent wraps and trims. The delete manager had to keep up with a faster cadence, widening the window for index/file desynchronization and hit-rate degradation.

```
BEFORE (wrap to PhysicalStartOffset):

Cycle 1:  [0 ──────────────────── MaxFileSize)   write range = MaxFileSize
Cycle 2:  [0 ──────────────────── MaxFileSize)   write range = MaxFileSize
Cycle 3:  [N ─────────────────── MaxFileSize)    write range = MaxFileSize - N   ← lost [0, N)
Cycle 4:  [2N ────────────────── MaxFileSize)    write range = MaxFileSize - 2N  ← lost [0, 2N)
  ...       ↑ dead zone grows each cycle

AFTER (wrap to 0):

Every cycle: [0 ──────────────────── MaxFileSize)   write range = MaxFileSize
  Writer reuses freed space immediately after TrimHead.
```

### Root Cause

`wrap_file.go` lines 82-84 / 120-123:

```go
// BUG: writer leapfrogs past [0, PhysicalStartOffset), never reusing freed space.
r.PhysicalWriteOffset = r.PhysicalStartOffset
```

### Fix

Three changes in `wrap_file.go`:

1. **Wrap to 0.** `Pwrite` and `PwriteBatch` now set `PhysicalWriteOffset = 0` when the writer reaches `MaxFileSize`. The writer immediately reuses space freed by prior `TrimHead` calls.

2. **Proper ring read validation.** Extracted `isValidReadRegion` shared by `Pread` and `ValidateReadOffset`. Handles three wrapped sub-cases:
   - `W == S` → ring is full, entire file is valid.
   - `W < S` → two valid segments: `[S, MaxFileSize)` and `[0, W)`.
   - `W > S` → contiguous `[S, W)` (occurs when `S` wraps to 0 after a full lap of trims).

3. **Updated test.** `TestPread_Success_WithWrap` now simulates the realistic flow (fill → wrap → `TrimHead` → write into freed region) and verifies both old tail data and new data are readable.

### Verification

All existing tests pass (`go test ./...`). The updated `TestPread_Success_WithWrap` validates:
- After wrap with `W == S`: both file halves readable (full ring).
- After `TrimHead`: punched region rejected, old tail readable.
- After writing into freed region: both old tail and new data readable.

---

### Next Steps (Proposed)

1. **Tune `KeysPerShard`** to match actual key count (~1M, not 67M) to reduce RingBuffer memory waste from ~1.86GB to ~75MB
2. **Double-buffered memtable** to eliminate the Gosched spin loop when memtable is full during flush
3. **Benchmark SQPOLL vs normal mode** in production to determine if the kernel polling thread CPU cost is justified
4. **Add `context.Context` propagation** throughout the read/write paths for proper timeout handling

---

## Phase 6: Predictor Observability & Frequency Counter Overhaul (Mar 30 – Apr 3, 2026)

### Context & Motivation

The rewrite predictor (`Predictor.Predict`) decides whether a key should be rewritten to a newer memtable based on a combined score of access frequency, recency, and ring-zone eviction risk. Until this phase, the predictor operated as a black box — it made rewrite decisions but emitted no telemetry, making it impossible to understand *why* keys were being rewritten or skipped in production.

The initial instrumentation effort (Mar 30) added metrics to make these decisions observable. This immediately revealed a critical problem: the frequency signal was completely flat. All percentiles (p50 through p999) reported the same value (~150), meaning the frequency component of the rewrite score contributed zero differentiation between keys.

### Problem: Morris Counter Saturation

**Root cause**: The original Morris log counter used **base 10** with a **4-bit mantissa** (values 0–9).

```
Old layout (base 10):
| exponent : 20 bits | mantissa : 4 bits |
    e (0–12)              m (0–9)

Decoded value = m × 10ᵉ
```

This created severe "dead zones" in frequency resolution:
- At exponent 0: representable values are 0, 1, 2, ..., 9 (step 1)
- At exponent 1: 0, 10, 20, ..., 90 (step 10)
- At exponent 2: 0, 100, 200, ..., 900 (**step 100** — no value between 100 and 200)

With only **37 distinct values** representable below 10,000, most keys clustered at the same decoded frequency. The scoring formula `fScore = 1 - exp(-wFreq × freq)` saturated to ~1.0 for any `freq ≥ 30`, rendering the frequency weight meaningless and making the grid search over `WFreq` ineffective.

### Fix: Base-2 Counter with 12-Bit Mantissa

The Morris counter was redesigned to use **base 2** with a **12-bit mantissa** packed into the same **uint16** budget:

```
New layout (base 2):
| exponent : 4 bits | mantissa : 12 bits |
    e (0–15)            m (0–4095)

Decoded value = m << e  (equivalently m × 2ᵉ)
```

Resolution comparison:

| Range     | Old (base 10) | New (base 2)     |
|-----------|--------------|------------------|
| 1–100     | 19 values    | **100** (exact)  |
| 100–1,000 | 9 values     | **901** (exact)  |
| 1K–5K     | 8 values     | **3,549** (step 1–2) |
| 5K–10K    | 1 value      | **2,049** (step 2–4) |
| **Total ≤10K** | **37** | **6,596**        |

Key design decisions:
- **Exact counts up to 4,095**: At e=0 the counter is a plain integer, so no approximation error for the vast majority of cache keys.
- **Halving on overflow**: When the mantissa overflows (m reaches 4096), it is halved to 2048 and the exponent incremented, keeping the decoded value approximately continuous across the transition (4095 → 4096).
- **External hash as randomness**: `Inc(v, hlo)` uses the key's existing hash instead of an internal PRNG, eliminating 4 bytes of per-counter RNG state.
- **Saturation at e=15**: Maximum representable value is 4095 × 2¹⁵ ≈ 134M, more than sufficient for cache frequency tracking.

### Committed Changes (Mar 30 – Apr 1)

**5 commits** across 3 files:

| Commit | Date | Description |
|--------|------|-------------|
| `fa26ef0` | Mar 30 | Add `scoreBucket()` helper and emit `flashring_rewrite_score` metric with score bucket tags |
| `c3529a7` | Mar 30 | Add `ringZone()` helper and emit `flashring_rewrite_decision` metric with decision, ring zone tags |
| `938d1d3` | Mar 30 | Add `freqBand()` helper, `FreqBands` config, and frequency band tag to rewrite decision metric |
| `e0ac365` | Mar 30 | Fix `FreqBands` zero-value default initialization (was using uninitialized struct) |
| `caaa2f3` | Apr 1 | Add `KEY_ACCESS_FREQ` timing metric to emit raw frequency distribution for grid search calibration |

Files changed: `predictor.go` (+86 lines), `cache.go` (+2), `metric.go` (+11)

### Uncommitted Changes (Apr 3, in progress)

Building on the observability data from the committed metrics, the following changes address the frequency saturation problem:

**1. Morris counter rewrite** (`freq.go`)
- Base 10 → Base 2 with 12-bit mantissa, 4-bit exponent
- `New(expClamp)` → `New()` (exponent range fixed at 0–15, no config needed)
- `Inc(v uint32)` → `Inc(v uint16, hlo uint64)` (external hash, uint16 state)
- `Value(v uint32)` → `Value(v uint16)` returning `m << e`
- Full doc block rewrite with correct algorithm description and examples

**2. Index integration** (`index.go`)
- `New(12)` → `New()` call updated
- `incrFreq(freq)` → `incrFreq(freq, hlo)` passing key hash to counter

**3. Recency band metric** (`predictor.go`, `metric.go`)
- Added `RecencyBands` config struct (thresholds: Hot, Warm, Cold)
- Added `recencyBand()` classifier and `TAG_RECENCY_BAND` / `KEY_LAST_ACCESS` metrics
- Rewrite decision metric now emits both `freq_band` and `recency_band` tags

**4. Cache config** (`cache.go`)
- Added `RecencyBands []int` to `Config` with safe nil-slice handling

**5. Test rewrite** (`freq_test.go`)
- All tests rewritten for the new base-2, uint16, external-hash API
- Covers: construction, pow2/threshold tables, value decoding, basic increment, mantissa overflow with halving, exponent saturation, miss behavior, statistical hit rates at e=0 and e=1, end-to-end counting approximation, bit packing roundtrip
