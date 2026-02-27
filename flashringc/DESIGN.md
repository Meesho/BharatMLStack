# flashringc — Design Document

## 1. Overview

**flashringc** is a high-performance, sharded key-value cache library with a C API. It uses a circular ring buffer on a raw block device or pre-allocated file as durable storage, with in-memory memtables and a per-shard key index. It is designed for low-latency get/put/delete and batch operations, with optional TTL and eviction when the ring is full.

### 1.1 Goals

- **Performance**: Minimize latency and maximize throughput via sharding, lock-free producer queues, and (on Linux) io_uring for async I/O.
- **Durability**: Data is written to a ring-backed device/file with O_DIRECT (Linux) or F_NOCACHE (macOS) to bypass kernel page cache.
- **Simplicity of API**: C API (`flashringc_open`, `flashringc_get`, `flashringc_put`, `flashringc_delete`, batch variants) for easy binding from other languages.
- **Portability**: Linux (io_uring optional) and macOS (pread/pwrite fallback).

### 1.2 Non-Goals (Current Scope)

- Distributed deployment; single-process, multi-threaded only.
- Strong consistency guarantees across multiple keys (per-key consistency only).
- Full database features (transactions, secondary indexes, range scans).

---

## 2. Architecture

### 2.1 Layering

```
┌─────────────────────────────────────────────────────────────────┐
│  C API (flashringc.h)                                            │
│  flashringc_open, flashringc_get/put/delete, flashringc_batch_*  │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│  Cache (cache.h/cache.cpp)                                       │
│  - Hash-based shard routing (Hash128 → shard)                     │
│  - SemaphorePool for blocking get/put/delete/batch                │
│  - Owns N ShardReactors + N threads                               │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│  ShardReactor (shard_reactor.h/cpp) — one per shard               │
│  - MPSCQueue<Request> inbox                                       │
│  - Event loop: process_inbox → handle_* → reap_completions        │
│  - io_uring (Linux) or pread/pwrite (macOS)                      │
│  - MemtableManager, KeyIndex, DeleteManager, RingDevice          │
└───────────────────────────────┬─────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
┌───────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ RingDevice    │     │ MemtableManager │     │ KeyIndex        │
│ (ring_device) │     │ (memtable_mgr)  │     │ (key_index)     │
│ - Circular    │     │ - Double-buffer │     │ - Hash → Entry  │
│   write/read  │     │ - Flush to ring │     │ - Eviction      │
│ - O_DIRECT    │     │ - try_read_     │     │ - TTL/freq      │
│ - TRIM/discard│     │   from_memory   │     │                 │
└───────────────┘     └─────────────────┘     └─────────────────┘
                                │
                                ▼
                      ┌─────────────────┐
                      │ DeleteManager   │
                      │ - Eviction      │
                      │ - TRIM/discard  │
                      │ - Index cleanup │
                      └─────────────────┘
```

### 2.2 Data Flow

- **Open**: `flashringc_open(path, capacity_bytes, num_shards)` → `Cache::open(config)` creates one `RingDevice` per shard (either regions of one block device or separate files), one `ShardReactor` per shard, and one thread per reactor. Reactors run `run()` (event loop).
- **Put**: Caller hashes key (xxHash 128-bit), acquires a semaphore slot, builds a `Request` (Put, hash, key, value, TTL), submits to `shard_for(hash)`, then waits on the semaphore. Reactor: `handle_put` → encode record → `MemtableManager::put` → optionally `swap_and_get_flush_buffer` and submit io_uring write; update `KeyIndex` with (hash → mem_id, offset, length). On completion, reactor posts semaphore.
- **Get**: Request (Get, hash, key) → reactor looks up `KeyIndex`; if in active/flushing memtable, read from memory; else submit io_uring read at `file_offset_for(mem_id) + offset`, then decode record and verify CRC; return value and post semaphore.
- **Delete**: Request (Delete, hash, key) → `KeyIndex::remove(hash)` (logical delete; no compaction of ring data).
- **Eviction**: When ring utilization exceeds `eviction_threshold`, `DeleteManager` runs: discard oldest blocks (TRIM), advance discard cursor, and amortized index cleanup (remove entries that point to discarded regions).

---

## 3. Core Components

### 3.1 RingDevice (`ring_device.h/cpp`)

- **Role**: Circular append-only buffer backed by a block device or file.
- **Details**:
  - Separate read and write FDs to avoid kernel-level contention on a single fd (e.g. independent NVMe submission queues).
  - Writes advance `write_offset_`; when offset reaches capacity, wrap and increment `wrap_count_`. Reads use `read(offset)`.
  - Block devices: `open_region(path, base, size)` for sharded regions; capacity can be auto-detected. Files: one file per shard (e.g. `path.0`, `path.1`).
  - `discard(offset, length)` sends BLKDISCARD/TRIM on block devices to inform SSD controller.
  - Block size: 4096 bytes; all I/O is block-aligned (see `AlignedBuffer`).

### 3.2 Record Format (`record.h`)

- **Layout**: `[key_len:4][val_len:4][key_data][val_data][crc32:4]`. CRC32C over first 8 + key_len + val_len bytes (Abseil CRC32C).
- **Encoding/decoding**: `encode_record`, `decode_record`, `verify_record_crc`. Used in memtable and when reading from ring.

### 3.3 MemtableManager (`memtable_manager.h/cpp`)

- **Role**: Double-buffered write buffer per shard. One “active” memtable receives new records; when it is full (or needs flush), it becomes “flushing” and the other becomes active.
- **Flow**: `put(data, len)` appends to active memtable (block-aligned). `needs_flush(next_record_len)` indicates full. `swap_and_get_flush_buffer()` returns the buffer to write via io_uring; after CQE, `complete_flush(mem_id, file_offset)` records offset and resets buffer. `try_read_from_memory` serves reads from active or flushing buffer; `file_offset_for(mem_id)` maps mem_id to ring offset for disk reads.

### 3.4 KeyIndex (`key_index.h/cpp`)

- **Role**: Single-threaded in-memory index: 128-bit hash → `Entry` (mem_id, offset, length, last_access, freq, ttl_seconds, flags). Used only by the owning ShardReactor.
- **Structure**: Fixed-capacity ring of `Entry` slots + `absl::flat_hash_map<uint64_t, uint32_t>` (hash_lo → ring index). Entries are 32-byte aligned; flags: Empty, Occupied, Deleted.
- **Operations**: `put`, `get`, `remove`, `evict_oldest` (for eviction/cleanup). TTL and frequency (e.g. Morris counter) can drive eviction policy.

### 3.5 DeleteManager (`delete_manager.h/cpp`)

- **Role**: When ring utilization exceeds `eviction_threshold`, discard oldest blocks (TRIM) and advance `discard_cursor`. Amortized cleanup of KeyIndex entries that point to discarded regions. Stops when utilization is at or below `clear_threshold`.
- **Config**: `eviction_threshold`, `clear_threshold`, `base_k`, `max_discards_per_put`, etc.

### 3.6 ShardReactor (`shard_reactor.h/cpp`)

- **Role**: Single-threaded event loop per shard. Owns one RingDevice, MemtableManager, KeyIndex, DeleteManager, MPSCQueue of Requests, and (on Linux) an io_uring instance.
- **Loop**:
  - If no in-flight I/O: block on `inbox_.pop_blocking(req)`, dispatch, optionally submit uring.
  - If in-flight I/O: drain inbox (up to `kMaxInboxBatch`), dispatch, then `reap_completions()` (complete Get reads, flush writes, etc.).
  - Periodically: `maybe_flush_memtable()`, `maybe_run_eviction()`.
- **Handlers**: `handle_get` (index lookup → memory or io_uring read), `handle_put` (encode, memtable put, index update, maybe flush), `handle_delete` (index remove).
- **Pending I/O**: `inflight_` and `pending_requests_` keyed by uring tag; completion moves result to `Result*` and posts semaphore (or decrements batch counter).

### 3.7 Cache (`cache.h/cache.cpp`)

- **Role**: Front-end that owns configuration, SemaphorePool, and N ShardReactors + N threads. Routes each key to a shard by `hash_key(key) % num_shards`.
- **Blocking API**: get/put/del acquire a semaphore slot, submit one Request (or many for batch), wait on the same slot, release slot, return Result.
- **Batch**: batch_get / batch_put submit one request per key to the appropriate shard(s), share one semaphore and an atomic `batch_remaining`; when the last request completes, it posts the semaphore once so the caller unblocks.

### 3.8 C API (`flashringc.h`, `flashringc_capi.cpp`)

- **Opaque handle**: `FlashRingC*` wraps `Cache`.
- **Functions**: `flashringc_open`, `flashringc_close`, `flashringc_get`, `flashringc_put`, `flashringc_delete`, `flashringc_batch_get`, `flashringc_batch_put`.
- **Return codes**: `FLASHRINGC_OK`, `FLASHRINGC_NOT_FOUND`, `FLASHRINGC_ERROR`. Batch APIs fill per-key status arrays and return count of successes (get) or successes (put).
- **Thread safety**: Safe to call from multiple threads; sharding and per-shard reactors provide concurrency.

### 3.9 Concurrency Primitives

- **MPSCQueue** (`mpsc_queue.h`): Bounded lock-free multi-producer single-consumer queue (sequence-based slots). Producers use `try_push`; consumer uses `try_pop` or `pop_blocking`. Used for Request inbox per shard.
- **SemaphorePool** (`semaphore_pool.h`): Pool of semaphores (or mutex+condvar on macOS where POSIX semaphores are not used). Caller acquires a slot index, passes it in the Request; reactor posts that slot when the request completes so the caller can wait and then release the slot.

---

## 4. Configuration

- **CacheConfig** (C++): `device_path`, `ring_capacity` (0 = auto for block device), `memtable_size`, `index_capacity`, `num_shards` (0 = hardware_concurrency()), `queue_capacity`, `uring_queue_depth`, `sem_pool_capacity`, `eviction_threshold`, `clear_threshold`.
- **C API**: `path`, `capacity_bytes`, `num_shards` (0 = auto). Other parameters use library defaults.

---

## 5. Dependencies

- **C++17**, CMake 3.14+.
- **Abseil**: `absl::flat_hash_map`, `absl::crc32c` (record CRC).
- **xxHash**: XXH3_128bits for key hashing (header-only, XXH_INLINE_ALL).
- **liburing** (optional, Linux): async I/O; if not found, pread/pwrite fallback.

---

## 6. Build and Tests

- **Build**: `cmake -B build && cmake --build build` in `flashringc/`. Library target: `flashring`.
- **Tests**: `test_capi`, `test_cache`, `test_shard_reactor`, `test_eviction`, `test_delete_manager`, `test_semaphore_pool`, `test_mpsc_queue`, `ring_test`, `index_test`, `index_perf_test`, `perf_test`; benchmarks: `bench_cache`, `bench_heavy`, `bench_long_run`.

---

## 7. File Layout Summary

| Path | Purpose |
|------|--------|
| `include/flashringc/flashringc.h` | C API declarations |
| `include/flashringc/cache.h` | Cache class and config |
| `include/flashringc/request.h` | Request, Result, KVPair, PendingOp |
| `include/flashringc/common.h` | Status, Hash128, AlignedBuffer, kBlockSize |
| `include/flashringc/record.h` | Record layout, encode/decode, CRC |
| `include/flashringc/ring_device.h` | RingDevice |
| `include/flashringc/memtable_manager.h` | MemtableManager, Memtable, FlushBuffer |
| `include/flashringc/key_index.h` | KeyIndex, Entry, LookupResult |
| `include/flashringc/delete_manager.h` | DeleteManager |
| `include/flashringc/shard_reactor.h` | ShardReactor |
| `include/flashringc/semaphore_pool.h` | SemaphorePool |
| `include/flashringc/mpsc_queue.h` | MPSCQueue |
| `src/flashringc_capi.cpp` | C API implementation |
| `src/cache.cpp` | Cache open, get/put/del, batch, flush, stats |
| `src/shard_reactor.cpp` | Reactor loop, handle_*, io_uring/reap |
| `src/ring_device.cpp` | Ring open, read, write, discard |
| `src/memtable_manager.cpp` | Memtable append, flush, read from memory |
| `src/key_index.cpp` | Index put/get/remove/evict, hash_key |
| `src/delete_manager.cpp` | Eviction, discard, index cleanup |
| `src/semaphore_pool.cpp` | Acquire, release, post, wait |

---

## 8. Security and Robustness Notes

- **Input**: Key/value lengths and buffers are trusted from the C API; buffer capacities are passed to avoid overflows.
- **Path**: Device/file path is provided by the caller; no path traversal from untrusted input in the library itself.
- **Concurrency**: No user-controlled URLs or network; single-process only. SSRF rules do not apply; secure coding for C++ (e.g. no raw deserialization of untrusted data, record CRC verification) is assumed.

---

## 9. Possible Future Extensions

- Optional read-through from a second-tier store.
- Configurable hash function or key comparison for custom types.
- Metrics/hooks for observability (latency, queue depth, eviction rate).
- Optional persistence of index (e.g. on close/open) to speed up cold start.
