# Vector DB Benchmark POC Results

This document records benchmark results for mwal with vector-DB payloads: catalog_id (key) + embedding (value) for add/update; catalog_id only for delete. Results may vary by hardware, OS, and load.

## Data Format Summary

| Operation   | Key                | Value                                |
|-------------|--------------------|--------------------------------------|
| Add/Update  | catalog_id (8-byte)| embedding (float32 or float64 array)|
| Delete      | catalog_id (8-byte)| (none)                               |

**Embedding sizes (payload bytes):**

| Dimension | fp32 (bytes) | fp64 (bytes) |
|-----------|--------------|--------------|
| 32        | 128          | 256          |
| 128       | 512          | 1,024        |
| 512       | 2,048        | 4,096        |
| 768       | 3,072        | 6,144        |

**Payload generation:** catalog_id encoded as 8-byte little-endian; embedding uses deterministic fill `float[i] = i/dim` for reproducibility. Implemented in [mwal/bench/vector_db_payload.h](mwal/bench/vector_db_payload.h).

---

## How to Run

From the mwal build directory:

```bash
mkdir -p build && cd build && cmake .. && make -j4

# Run only VectorDB benchmarks
./bench/db_wal_bench --benchmark_filter='BM_.*VectorDB'
./bench/wal_write_bench --benchmark_filter='BM_.*VectorDB'
./bench/wal_read_bench --benchmark_filter='BM_.*VectorDB'
./bench/wal_sync_bench --benchmark_filter='BM_.*VectorDB'

# Run specific benchmark (e.g. 768-dim fp32, ops_per_batch=100)
./bench/db_wal_bench --benchmark_filter='BM_DBWalWrite_VectorDB/100/768/0'

# JSON output for analysis
./bench/db_wal_bench --benchmark_filter='BM_.*VectorDB' \
  --benchmark_out=vector_db_results.json --benchmark_out_format=json

# Time-based flush benchmarks only
./bench/db_wal_bench --benchmark_filter='BM_TimeBasedFlush.*_VectorDB' \
  --benchmark_out=time_flush_results.json --benchmark_out_format=json
```

## Baseline Environment

| Field | Value |
|-------|-------|
| Date | 2026-03-02 |
| OS | macOS (darwin 25.1.0) |
| CPU | Apple Silicon, 12 cores @ 24 MHz (reported) |
| Disk | Default (SSD typical for development) |

---

## db_wal_bench Results

### BM_DBWalWrite_VectorDB

Single-thread write throughput (1000 batches per iteration). `ops_per_batch` controls batching. Catalog_id + embedding per Put.

**By dimension and dtype (full matrix):**

| ops_per_batch | dim | dtype | Time (ns) | Items/sec |
|---------------|-----|-------|-----------|-----------|
| 1 | 32 | fp32 | 6,729,598 | 373k/s |
| 10 | 32 | fp32 | 11,277,580 | 3.12M/s |
| 100 | 32 | fp32 | 57,981,331 | 10.4M/s |
| 1 | 128 | fp32 | 7,957,211 | 365k/s |
| 10 | 128 | fp32 | 22,943,930 | 3.10M/s |
| 100 | 128 | fp32 | 144,543,875 | 7.07M/s |
| 1 | 512 | fp32 | 17,849,211 | 354k/s |
| 10 | 512 | fp32 | 64,816,912 | 1.72M/s |
| 100 | 512 | fp32 | 528,608,008 | 3.75M/s |
| 1 | 768 | fp32 | 18,010,659 | 421k/s |
| 10 | 768 | fp32 | 87,862,191 | 1.51M/s |
| 100 | 768 | fp32 | 787,588,254 | 2.90M/s |
| 1 | 32 | fp64 | 7,906,921 | 365k/s |
| 10 | 32 | fp64 | 8,994,789,081 | 2.95M/s |
| 100 | 32 | fp64 | 89,184,554 | 8.78M/s |
| 1 | 128 | fp64 | 10,677,792 | 375k/s |
| 10 | 128 | fp64 | 38,114,738 | 2.46M/s |
| 100 | 128 | fp64 | 269,745,950 | 5.43M/s |
| 1 | 512 | fp64 | 19,573,987 | 427k/s |
| 10 | 512 | fp64 | 109,502,233 | 1.21M/s |
| 100 | 512 | fp64 | 94,440,000,000 | 2.09M/s |
| 1 | 768 | fp64 | 34,848,824 | 271k/s |
| 10 | 768 | fp64 | 172,196,323 | 856k/s |
| 100 | 768 | fp64 | 1,572,416,773 | 1.53M/s |

**Observation:** Batching (ops_per_batch 1→100) yields ~15–28x throughput gain. Larger embeddings (768 fp64) reduce ops/sec due to more bytes per record.

### BM_DBWalWriteSync_VectorDB

Write with sync=false vs sync=true (1000 writes per iteration).

| do_sync | dim | dtype | Time (ns) | Items/sec |
|---------|-----|-------|-----------|-----------|
| 0 (async) | 32 | fp32 | 6,599,237 | 375k/s |
| 1 (sync) | 32 | fp32 | 2,842,036,793 | 5.0k/s |
| 0 (async) | 128 | fp32 | 7,714,624 | 376k/s |
| 1 (sync) | 128 | fp32 | 2,969,121,875 | 6.6k/s |
| 0 (async) | 512 | fp32 | 17,515,749 | 286k/s |
| 1 (sync) | 512 | fp32 | 2,628,559,052 | 7.5k/s |
| 0 (async) | 768 | fp32 | 18,014,670 | 326k/s |
| 1 (sync) | 768 | fp32 | 2,273,557,475 | 5.3k/s |
| 0 (async) | 32 | fp64 | 7,949,444 | 336k/s |
| 1 (sync) | 32 | fp64 | 2,908,404,959 | 16.4k/s |
| 0 (async) | 128 | fp64 | 10,101,334 | 360k/s |
| 1 (sync) | 128 | fp64 | 4,359,044,833 | 5.2k/s |
| 0 (async) | 512 | fp64 | 21,277,243 | 342k/s |
| 1 (sync) | 512 | fp64 | 4,277,344,708 | 3.8k/s |
| 0 (async) | 768 | fp64 | 27,958,151 | 298k/s |
| 1 (sync) | 768 | fp64 | 4,414,743,459 | 3.7k/s |

**Note:** sync=true is ~1500x slower (wall time) than sync=false due to per-write fsync.

### BM_DBWalConcurrentWrite_VectorDB

Concurrent write throughput with barrier (200 writes/thread). All threads start simultaneously.

**32-dim fp32:**

| num_threads | Time (ns) | Items/sec |
|-------------|-----------|-----------|
| 1 | 1,771,378 | 1.25M/s |
| 2 | 2,154,328 | 2.32M/s |
| 4 | 2,571,395 | 4.16M/s |
| 8 | 4,960,654 | 7.41M/s |
| 16 | 13,935,436 | 9.75M/s |

**128-dim fp32:**

| num_threads | Time (ns) | Items/sec |
|-------------|-----------|-----------|
| 1 | 2,013,941 | 1.26M/s |
| 8 | 6,200,403 | 7.03M/s |
| 16 | 12,722,774 | 9.88M/s |

**512-dim fp32:**

| num_threads | Time (ns) | Items/sec |
|-------------|-----------|-----------|
| 1 | 2,975,415 | 1.21M/s |
| 8 | 53,075,058 | 3.42M/s |
| 16 | 27,412,989 | 7.87M/s |

**768-dim fp32:**

| num_threads | Time (ns) | Items/sec |
|-------------|-----------|-----------|
| 1 | 7,492,117 | 924k/s |
| 8 | 59,029,273 | 3.49M/s |
| 16 | 39,700,974 | 6.73M/s |

**768-dim fp64:**

| num_threads | Time (ns) | Items/sec |
|-------------|-----------|-----------|
| 1 | 5,664,270 | 1.10M/s |
| 8 | 31,123,058 | 5.32M/s |
| 16 | 60,797,623 | 6.45M/s |

**Observation:** Throughput scales with threads; larger embeddings reduce ops/sec due to payload size.

### BM_DBWalWriteLatencyUnderThroughput_VectorDB

Per-write latency (lock + queue + write) under concurrent load. 10k samples/thread, batches pre-built outside timing. Reports min, mean, p50, p95, p99, max (ns).

**32-dim fp32 (1 thread):** mean 9,584 ns, p99 24,459 ns, Items/sec 30.4M/s  
**32-dim fp32 (16 threads):** mean 100,034 ns, p99 293,166 ns, Items/sec 28.4M/s  
**768-dim fp32 (1 thread):** mean 27,129 ns, p99 86,625 ns, Items/sec 9.88M/s  
**768-dim fp32 (16 threads):** mean 239,410 ns, p99 633,792 ns, Items/sec 4.16M/s  
**768-dim fp64 (1 thread):** mean 26,350 ns, p99 37,208 ns, Items/sec 2.73M/s  
**768-dim fp64 (16 threads):** mean 330,196 ns, p99 791,542 ns, Items/sec 10.4M/s

**Note:** Latency grows with thread count; larger embeddings increase per-write cost.

### BM_DBWalRecovery_VectorDB

Time to open and recover WAL (pre-written before timer).

| num_records | dim | dtype | Time (ns) | Items/sec |
|-------------|-----|-------|-----------|-----------|
| 1,000 | 32 | fp32 | 8,619,930 | 155k/s |
| 10,000 | 32 | fp32 | 12,494,141 | 985k/s |
| 100,000 | 32 | fp32 | 44,092,047 | 2.39M/s |
| 1,000 | 128 | fp32 | 10,227,779 | 128k/s |
| 10,000 | 128 | fp32 | 20,211,194 | 581k/s |
| 100,000 | 128 | fp32 | 137,747,700 | 730k/s |
| 1,000 | 512 | fp32 | 14,913,612 | 82.4k/s |
| 10,000 | 512 | fp32 | 55,158,904 | 186k/s |
| 100,000 | 512 | fp32 | 536,800,042 | 189k/s |
| 1,000 | 768 | fp32 | 17,722,536 | 68.8k/s |
| 10,000 | 768 | fp32 | 86,381,516 | 117k/s |
| 100,000 | 768 | fp32 | 839,596,917 | 120k/s |
| 1,000 | 32 | fp64 | 10,806,256 | 127k/s |
| 10,000 | 32 | fp64 | 16,967,255 | 722k/s |
| 100,000 | 32 | fp64 | 80,116,727 | 1.27M/s |
| 1,000 | 768 | fp64 | 21,578,201 | 54.7k/s |
| 10,000 | 768 | fp64 | 153,131,233 | 65.9k/s |
| 100,000 | 768 | fp64 | 1,683,664,417 | 64.6k/s |

**Observation:** Recovery rate decreases with larger embeddings (more bytes per record to decode).

### BM_DBWalRotation_VectorDB

Write throughput with max_wal_file_size=4096 (frequent rotation). 500 writes.

| dim | dtype | Time (ns) | Items/sec |
|-----|-------|-----------|-----------|
| 32 | fp32 | 9,680,108 | 191k/s |
| 128 | fp32 | 21,192,857 | 97.2k/s |
| 512 | fp32 | 55,754,308 | 33.3k/s |
| 768 | fp32 | 66,603,017 | 29.8k/s |
| 32 | fp64 | 11,275,745 | 151k/s |
| 128 | fp64 | 33,441,398 | 58.4k/s |
| 512 | fp64 | 107,767,441 | 16.4k/s |
| 768 | fp64 | 105,922,809 | 17.2k/s |

### BM_GroupCommitScaling_VectorDB

Sync write throughput (sync=true, group commit coalescing). 2000 total writes. `wall_ops_per_sec` uses manually measured wall time.

| num_threads | 32 fp32 | 128 fp32 | 512 fp32 | 768 fp32 | 768 fp64 |
|-------------|---------|----------|----------|----------|----------|
| 1 | 416 | 377 | 326 | 348 | 230 |
| 2 | 759 | 494 | 493 | 505 | 307 |
| 4 | 873 | 844 | 786 | 873 | 528 |
| 8 | 1,538 | 1,464 | 1,406 | 1,428 | 958 |

**Observation:** Wall-time sync throughput scales ~3.7x from 1→8 threads; larger embeddings reduce absolute ops/sec.

### BM_MergedGroupCommit_VectorDB

Async write throughput (sync=false). 500 writes/thread, batches merged.

| num_threads | 32 fp32 | 128 fp32 | 512 fp32 | 768 fp32 | 768 fp64 |
|-------------|---------|----------|----------|----------|----------|
| 1 | 125k | 108k | 2.52k | 2.69k | 35.4k |
| 8 | 250k | 24.3k | 9.17k | 3.32k | 51.8k |
| 16 | 214k | 15.6k | 12.3k | 5.73k | 49.3k |

**Note:** For 512/768 dim, wall_ops_per_sec drops due to payload size and contention.

### BM_CoalescedAsyncWrite_VectorDB

Async writes via coalescer (max_async_queue_depth=10000). 500 writes/thread.

| num_threads | 32 fp32 | 128 fp32 | 512 fp32 | 768 fp32 | 768 fp64 |
|-------------|---------|----------|----------|----------|----------|
| 1 | 127k | 90.0k | 67.8k | 56.3k | 38.0k |
| 8 | 241k | 246k | 120k | 88.4k | 52.4k |
| 16 | 197k | 171k | 119k | 88.1k | 51.2k |

### BM_CompressionPrefixOverhead_VectorDB

Write throughput with no compression. 2000 writes.

| dim | dtype | Time (ns) | Items/sec |
|-----|-------|-----------|-----------|
| 32 | fp32 | 17,305,012 | 347k/s |
| 128 | fp32 | 19,857,665 | 327k/s |
| 512 | fp32 | 31,171,966 | 316k/s |
| 768 | fp32 | 40,640,963 | 317k/s |
| 32 | fp64 | 14,501,695 | 401k/s |
| 128 | fp64 | 21,190,537 | 413k/s |
| 512 | fp64 | 44,571,773 | 381k/s |
| 768 | fp64 | 60,042,650 | 303k/s |

### BM_DBWalCompression_VectorDB

Write throughput with compression off vs on (Zstd). 1000 writes.

| compress | dim | dtype | Time (ns) | Items/sec |
|----------|-----|-------|-----------|-----------|
| 0 (off) | 32 | fp32 | 6,846,972 | 367k/s |
| 1 (on) | 32 | fp32 | 11,047,753 | 369k/s |
| 0 (off) | 768 | fp32 | 16,318,302 | 419k/s |
| 1 (on) | 768 | fp32 | 22,850,350 | 414k/s |
| 0 (off) | 768 | fp64 | 29,493,535 | 268k/s |
| 1 (on) | 768 | fp64 | 34,674,072 | 293k/s |

**Observation:** Zstd adds CPU overhead; for vector payloads compression can yield similar or slightly higher throughput when I/O is reduced.

### BM_WalIteratorScanSpeed_VectorDB

`NewWalIterator(0)` + consume all records. For replication, change capture, or backup.

| num_records | dim | dtype | Time (ns) | Items/sec | MB/s |
|-------------|-----|-------|-----------|-----------|------|
| 10,000 | 32 | fp32 | 8,936,966 | 1.45M/s | 199 |
| 100,000 | 32 | fp32 | 38,698,493 | 2.73M/s | 374 |
| 1,000,000 | 32 | fp32 | 373,508,604 | 2.73M/s | 374 |
| 10,000 | 768 | fp32 | 76,558,157 | 133k/s | 393 |
| 100,000 | 768 | fp32 | 745,654,208 | 134k/s | 396 |
| 1,000,000 | 768 | fp32 | 8,082,821,416 | 127k/s | 375 |
| 10,000 | 768 | fp64 | 154,874,000 | 65.3k/s | 383 |
| 100,000 | 768 | fp64 | 1,714,297,583 | 61.6k/s | 362 |
| 1,000,000 | 768 | fp64 | 17,844,000,000 | 57.1k/s | 335 |

**Observation:** Items/sec drops with larger embeddings; MB/s stays roughly 335–400 for 768-dim.

### BM_RecoveryWithCompression_VectorDB

Recovery time with zstd-compressed vs uncompressed WAL.

**32-dim fp32:**

| num_records | compress | Time (ns) | MB/s |
|-------------|----------|-----------|------|
| 1,000 | 0 | 9,448,570 | 22.2 |
| 10,000 | 0 | 15,683,297 | 120 |
| 100,000 | 0 | 50,277,672 | 307 |
| 1,000 | 1 (zstd) | 12,652,462 | 15.2 |
| 10,000 | 1 (zstd) | 35,530,095 | 46.2 |
| 100,000 | 1 (zstd) | 294,053,292 | 49.6 |

**768-dim fp32:**

| num_records | compress | Time (ns) | MB/s |
|-------------|----------|-----------|------|
| 1,000 | 0 | 16,188,304 | 222 |
| 10,000 | 0 | 91,341,536 | 331 |
| 100,000 | 0 | 901,520,250 | 337 |
| 1,000 | 1 (zstd) | 18,643,746 | 199 |
| 10,000 | 1 (zstd) | 110,663,597 | 272 |
| 100,000 | 1 (zstd) | 1,091,943,292 | 271 |

**768-dim fp64:**

| num_records | compress | Time (ns) | MB/s |
|-------------|----------|-----------|------|
| 1,000 | 0 | 22,117,028 | 341 |
| 10,000 | 0 | 143,503,467 | 416 |
| 100,000 | 0 | 1,394,304,916 | 423 |
| 1,000 | 1 (zstd) | 18,302,492 | 418 |
| 10,000 | 1 (zstd) | 107,071,132 | 559 |
| 100,000 | 1 (zstd) | 1,135,016,250 | 541 |

**Observation:** Zstd recovery can reach higher MB/s when compressed size is small; 768 fp64 with zstd shows ~541 MB/s at 100k records.

### BM_WriteStallRecoveryTime_VectorDB

Time from `PurgeObsoleteFiles()` to first successful `Write()` after stall (max_total_wal_size exceeded).

| dim | dtype | stall_recovery_us |
|-----|-------|-------------------|
| 32 | fp32 | ~1,292 |
| 128 | fp32 | ~986 |
| 512 | fp32 | ~659 |
| 768 | fp32 | ~776 |
| 32 | fp64 | ~794 |
| 128 | fp64 | ~882 |
| 512 | fp64 | ~756 |
| 768 | fp64 | ~670 |

### BM_SingleLargeRecordVsManySmall_VectorDB

Compare 100 large records (scenario 0) vs 100 batches of 1000 small records (scenario 1). Scenario 0: 100 records of dim×4 or dim×8 bytes; scenario 1: 100×1000 records.

| scenario | dim | dtype | Time (ns) | MB/s |
|-----------|-----|-------|-----------|------|
| 0 (large) | 32 | fp32 | 1,286,373 | 22.1 |
| 1 (many small) | 32 | fp32 | 46,844,521 | 1.67 Gi/s |
| 0 (large) | 768 | fp32 | 2,107,109 | 573 |
| 1 (many small) | 768 | fp32 | 956,156,175 | 9.06 Gi/s |
| 0 (large) | 768 | fp64 | 2,988,580 | 1.14 Gi/s |
| 1 (many small) | 768 | fp64 | 1,867,186,517 | 8.24 Gi/s |

**Observation:** Many-small scenario yields higher MB/s due to batching; large records stress rotation and single-record overhead.

### BM_PurgeUnderWriteLoad_VectorDB

Write throughput and p99 latency with vs without concurrent `PurgeObsoleteFiles()` every N ms. 2000 writes/phase, max_wal_file_size=64KB.

**32-dim fp32 (sample):**

| write_threads | purge_interval_ms | throughput_no_purge | throughput_with_purge | p99_ns_no_purge | p99_ns_with_purge | throughput_drop_pct |
|---------------|-------------------|---------------------|------------------------|-----------------|--------------------|---------------------|
| 4 | 100 | 319k/s | 308k/s | 54,889 | 57,848 | 3.6 |
| 8 | 100 | 262k/s | 265k/s | 175,880 | 162,792 | -1.0 |

**768-dim fp32 (sample):**

| write_threads | purge_interval_ms | throughput_no_purge | throughput_with_purge | throughput_drop_pct |
|---------------|-------------------|---------------------|------------------------|---------------------|
| 4 | 100 | 57.8k/s | 60.1k/s | -4.0 |
| 8 | 100 | 56.7k/s | 58.3k/s | -2.9 |

**Interpretation:** Negative drop % means throughput was higher with purge (variance). Throughput drops with larger embeddings.

### BM_MixedSyncAsync_VectorDB

Production pattern: N% sync threads, (100-N)% async. 2000 total writes. Measures async_p99_ns, sync_p99_ns, wall_ops_per_sec.

**32-dim fp32 (sample):**

| num_threads | sync_pct | async_p99_ns | sync_p99_ns | wall_ops_per_sec |
|-------------|----------|--------------|-------------|------------------|
| 8 | 10 | 241k | 5.77M | 2.82k |
| 16 | 10 | 15.1M | 8.93M | 4.10k |
| 32 | 10 | 22.1M | 8.43M | 6.56k |

**768-dim fp32 (sample):**

| num_threads | sync_pct | async_p99_ns | wall_ops_per_sec |
|-------------|----------|--------------|------------------|
| 8 | 10 | 21.9M | 2.34k |
| 8 | 50 | 20.1M | 1.40k |

*Note: p99 values in ns (M = millions). 6M ns = 6 ms.*

---

## Time-based flush benchmarks

Results from the same baseline environment. Run with:
```bash
./bench/db_wal_bench --benchmark_filter='BM_TimeBasedFlush.*_VectorDB' \
  --benchmark_out=time_flush_results.json --benchmark_out_format=json
```

### BM_TimeBasedFlushThroughput_VectorDB

Async-only throughput vs flush interval (500 writes/thread).

| interval_ms | num_threads | dim | dtype | wall_ops_per_sec |
|--------------|-------------|-----|-------|------------------|
| 0 | 4 | 32 | fp32 | 389.3k |
| 0 | 4 | 32 | fp64 | 371.7k |
| 0 | 4 | 128 | fp32 | 296.6k |
| 0 | 4 | 128 | fp64 | 200.4k |
| 0 | 4 | 512 | fp32 | 115.5k |
| 0 | 4 | 512 | fp64 | 68.9k |
| 0 | 4 | 768 | fp32 | 86.2k |
| 0 | 4 | 768 | fp64 | 47.0k |
| 0 | 8 | 32 | fp32 | 355.9k |
| 0 | 8 | 32 | fp64 | 336.6k |
| 0 | 8 | 128 | fp32 | 283.0k |
| 0 | 8 | 128 | fp64 | 204.8k |
| 0 | 8 | 512 | fp32 | 124.4k |
| 0 | 8 | 512 | fp64 | 65.3k |
| 0 | 8 | 768 | fp32 | 90.2k |
| 0 | 8 | 768 | fp64 | 53.2k |
| 10 | 4 | 32 | fp32 | 318.0k |
| 10 | 4 | 32 | fp64 | 381.2k |
| 10 | 4 | 128 | fp32 | 279.1k |
| 10 | 4 | 128 | fp64 | 197.7k |
| 10 | 4 | 512 | fp32 | 112.2k |
| 10 | 4 | 512 | fp64 | 68.5k |
| 10 | 4 | 768 | fp32 | 82.8k |
| 10 | 4 | 768 | fp64 | 44.9k |
| 10 | 8 | 32 | fp32 | 354.6k |
| 10 | 8 | 32 | fp64 | 326.8k |
| 10 | 8 | 128 | fp32 | 286.6k |
| 10 | 8 | 128 | fp64 | 204.6k |
| 10 | 8 | 512 | fp32 | 122.9k |
| 10 | 8 | 512 | fp64 | 49.1k |
| 10 | 8 | 768 | fp32 | 90.9k |
| 10 | 8 | 768 | fp64 | 53.5k |
| 50 | 4 | 32 | fp32 | 384.6k |
| 50 | 4 | 32 | fp64 | 372.4k |
| 50 | 4 | 128 | fp32 | 298.2k |
| 50 | 4 | 128 | fp64 | 197.6k |
| 50 | 4 | 512 | fp32 | 114.8k |
| 50 | 4 | 512 | fp64 | 63.3k |
| 50 | 4 | 768 | fp32 | 83.5k |
| 50 | 4 | 768 | fp64 | 47.6k |
| 50 | 8 | 32 | fp32 | 350.4k |
| 50 | 8 | 32 | fp64 | 334.7k |
| 50 | 8 | 128 | fp32 | 289.1k |
| 50 | 8 | 128 | fp64 | 207.7k |
| 50 | 8 | 512 | fp32 | 126.1k |
| 50 | 8 | 512 | fp64 | 71.8k |
| 50 | 8 | 768 | fp32 | 86.5k |
| 50 | 8 | 768 | fp64 | 53.2k |
| 100 | 4 | 32 | fp32 | 418.2k |
| 100 | 4 | 32 | fp64 | 379.6k |
| 100 | 4 | 128 | fp32 | 295.7k |
| 100 | 4 | 128 | fp64 | 196.8k |
| 100 | 4 | 512 | fp32 | 115.3k |
| 100 | 4 | 512 | fp64 | 63.3k |
| 100 | 4 | 768 | fp32 | 77.4k |
| 100 | 4 | 768 | fp64 | 47.4k |
| 100 | 8 | 32 | fp32 | 353.5k |
| 100 | 8 | 32 | fp64 | 298.2k |
| 100 | 8 | 128 | fp32 | 278.4k |
| 100 | 8 | 128 | fp64 | 198.9k |
| 100 | 8 | 512 | fp32 | 125.2k |
| 100 | 8 | 512 | fp64 | 71.6k |
| 100 | 8 | 768 | fp32 | 91.4k |
| 100 | 8 | 768 | fp64 | 53.4k |
| 200 | 4 | 32 | fp32 | 413.7k |
| 200 | 4 | 32 | fp64 | 383.7k |
| 200 | 4 | 128 | fp32 | 296.2k |
| 200 | 4 | 128 | fp64 | 195.6k |
| 200 | 4 | 512 | fp32 | 116.0k |
| 200 | 4 | 512 | fp64 | 62.0k |
| 200 | 4 | 768 | fp32 | 36.1k |
| 200 | 4 | 768 | fp64 | 47.6k |
| 200 | 8 | 32 | fp32 | 344.9k |
| 200 | 8 | 32 | fp64 | 326.4k |
| 200 | 8 | 128 | fp32 | 285.2k |
| 200 | 8 | 128 | fp64 | 205.7k |
| 200 | 8 | 512 | fp32 | 110.6k |
| 200 | 8 | 512 | fp64 | 72.4k |
| 200 | 8 | 768 | fp32 | 90.5k |
| 200 | 8 | 768 | fp64 | 53.0k |

### BM_TimeBasedFlushIdleLatency_VectorDB

Single-thread sequential write latency vs interval (one thread, 100 back-to-back async writes per iteration; not strict idle). Interval affects when the writer wakes (timeout vs notify), so p50/p99 capture that variance.

| interval_ms | dim | dtype | p50_ns | p99_ns |
|--------------|-----|-------|--------|--------|
| 0 | 32 | fp32 | 7k | 9k |
| 0 | 32 | fp64 | 6k | 13k |
| 0 | 128 | fp32 | 8k | 12k |
| 0 | 128 | fp64 | 9k | 14k |
| 0 | 512 | fp32 | 13k | 21k |
| 0 | 512 | fp64 | 18k | 21k |
| 0 | 768 | fp32 | 15k | 22k |
| 0 | 768 | fp64 | 23k | 27k |
| 25 | 32 | fp32 | 6k | 11k |
| 25 | 32 | fp64 | 8k | 16k |
| 25 | 128 | fp32 | 8k | 16k |
| 25 | 128 | fp64 | 10k | 14k |
| 25 | 512 | fp32 | 12k | 18k |
| 25 | 512 | fp64 | 19k | 30k |
| 25 | 768 | fp32 | 16k | 22k |
| 25 | 768 | fp64 | 24k | 30k |
| 50 | 32 | fp32 | 7k | 13k |
| 50 | 32 | fp64 | 7k | 11k |
| 50 | 128 | fp32 | 8k | 17k |
| 50 | 128 | fp64 | 8k | 14k |
| 50 | 512 | fp32 | 12k | 20k |
| 50 | 512 | fp64 | 18k | 25k |
| 50 | 768 | fp32 | 16k | 19k |
| 50 | 768 | fp64 | 24k | 31k |
| 100 | 32 | fp32 | 8k | 14k |
| 100 | 32 | fp64 | 7k | 13k |
| 100 | 128 | fp32 | 7k | 13k |
| 100 | 128 | fp64 | 8k | 14k |
| 100 | 512 | fp32 | 12k | 18k |
| 100 | 512 | fp64 | 20k | 27k |
| 100 | 768 | fp32 | 16k | 22k |
| 100 | 768 | fp64 | 24k | 32k |

### BM_TimeBasedFlushBurstLatency_VectorDB

Burst of K writes then stop; total completion time (ns).

| interval_ms | burst_size | dim | dtype | burst_total_ns |
|--------------|------------|-----|-------|-----------------|
| 0 | 5 | 32 | fp32 | 50k |
| 0 | 5 | 32 | fp64 | 230k |
| 0 | 5 | 128 | fp32 | 47k |
| 0 | 5 | 128 | fp64 | 75k |
| 0 | 5 | 512 | fp32 | 71k |
| 0 | 5 | 512 | fp64 | 122k |
| 0 | 5 | 768 | fp32 | 94k |
| 0 | 5 | 768 | fp64 | 126k |
| 0 | 10 | 32 | fp32 | 79k |
| 0 | 10 | 32 | fp64 | 91k |
| 0 | 10 | 128 | fp32 | 108k |
| 0 | 10 | 128 | fp64 | 104k |
| 0 | 10 | 512 | fp32 | 237k |
| 0 | 10 | 512 | fp64 | 217k |
| 0 | 10 | 768 | fp32 | 196k |
| 0 | 10 | 768 | fp64 | 257k |
| 50 | 5 | 32 | fp32 | 176k |
| 50 | 5 | 32 | fp64 | 58k |
| 50 | 5 | 128 | fp32 | 52k |
| 50 | 5 | 128 | fp64 | 64k |
| 50 | 5 | 512 | fp32 | 84k |
| 50 | 5 | 512 | fp64 | 117k |
| 50 | 5 | 768 | fp32 | 90k |
| 50 | 5 | 768 | fp64 | 138k |
| 50 | 10 | 32 | fp32 | 109k |
| 50 | 10 | 32 | fp64 | 171k |
| 50 | 10 | 128 | fp32 | 242k |
| 50 | 10 | 128 | fp64 | 117k |
| 50 | 10 | 512 | fp32 | 135k |
| 50 | 10 | 512 | fp64 | 228k |
| 50 | 10 | 768 | fp32 | 174k |
| 50 | 10 | 768 | fp64 | 297k |
| 100 | 5 | 32 | fp32 | 49k |
| 100 | 5 | 32 | fp64 | 60k |
| 100 | 5 | 128 | fp32 | 50k |
| 100 | 5 | 128 | fp64 | 172k |
| 100 | 5 | 512 | fp32 | 78k |
| 100 | 5 | 512 | fp64 | 133k |
| 100 | 5 | 768 | fp32 | 95k |
| 100 | 5 | 768 | fp64 | 127k |
| 100 | 10 | 32 | fp32 | 86k |
| 100 | 10 | 32 | fp64 | 103k |
| 100 | 10 | 128 | fp32 | 104k |
| 100 | 10 | 128 | fp64 | 115k |
| 100 | 10 | 512 | fp32 | 150k |
| 100 | 10 | 512 | fp64 | 217k |
| 100 | 10 | 768 | fp32 | 191k |
| 100 | 10 | 768 | fp64 | 254k |

### BM_TimeBasedFlushMixedSyncAsync_VectorDB

Mixed sync/async with and without time-based flush. 2000 writes.

| num_threads | sync_pct | interval_ms | dim | dtype | async_p99_ns | sync_p99_ns | wall_ops_per_sec |
|-------------|----------|--------------|-----|-------|--------------|-------------|------------------|
| 8 | 10 | 0 | 32 | fp32 | 103M | 5M | 2.9k |
| 8 | 10 | 0 | 32 | fp64 | 75M | 4M | 2.9k |
| 8 | 10 | 0 | 128 | fp32 | 21M | 4M | 2.7k |
| 8 | 10 | 0 | 128 | fp64 | 84M | 4M | 2.9k |
| 8 | 10 | 0 | 512 | fp32 | 117M | 4M | 2.6k |
| 8 | 10 | 0 | 512 | fp64 | 79M | 4M | 2.4k |
| 8 | 10 | 0 | 768 | fp32 | 104M | 8M | 2.5k |
| 8 | 10 | 0 | 768 | fp64 | 81M | 4M | 2.4k |
| 8 | 10 | 50 | 32 | fp32 | 93M | 5M | 2.9k |
| 8 | 10 | 50 | 32 | fp64 | 96M | 4M | 3.0k |
| 8 | 10 | 50 | 128 | fp32 | 69M | 4M | 3.0k |
| 8 | 10 | 50 | 128 | fp64 | 119M | 4M | 2.9k |
| 8 | 10 | 50 | 512 | fp32 | 99M | 4M | 2.6k |
| 8 | 10 | 50 | 512 | fp64 | 79M | 4M | 2.3k |
| 8 | 10 | 50 | 768 | fp32 | 110M | 5M | 2.5k |
| 8 | 10 | 50 | 768 | fp64 | 129M | 6M | 2.4k |
| 8 | 25 | 0 | 32 | fp32 | 20M | 11M | 2.1k |
| 8 | 25 | 0 | 32 | fp64 | 13M | 8M | 2.1k |
| 8 | 25 | 0 | 128 | fp32 | 13M | 7M | 2.1k |
| 8 | 25 | 0 | 128 | fp64 | 14M | 8M | 2.0k |
| 8 | 25 | 0 | 512 | fp32 | 15M | 8M | 1.8k |
| 8 | 25 | 0 | 512 | fp64 | 15M | 9M | 1.8k |
| 8 | 25 | 0 | 768 | fp32 | 15M | 8M | 1.8k |
| 8 | 25 | 0 | 768 | fp64 | 17M | 9M | 1.8k |
| 8 | 25 | 50 | 32 | fp32 | 17M | 7M | 2.2k |
| 8 | 25 | 50 | 32 | fp64 | 15M | 7M | 2.1k |
| 8 | 25 | 50 | 128 | fp32 | 19M | 8M | 2.0k |
| 8 | 25 | 50 | 128 | fp64 | 13M | 8M | 1.9k |
| 8 | 25 | 50 | 512 | fp32 | 17M | 8M | 1.8k |
| 8 | 25 | 50 | 512 | fp64 | 16M | 9M | 1.8k |
| 8 | 25 | 50 | 768 | fp32 | 16M | 10M | 1.8k |
| 8 | 25 | 50 | 768 | fp64 | 17M | 10M | 1.7k |
| 16 | 10 | 0 | 32 | fp32 | 18M | 8M | 4.2k |
| 16 | 10 | 0 | 32 | fp64 | 13M | 7M | 4.3k |
| 16 | 10 | 0 | 128 | fp32 | 15M | 7M | 4.1k |
| 16 | 10 | 0 | 128 | fp64 | 15M | 8M | 3.8k |
| 16 | 10 | 0 | 512 | fp32 | 19M | 8M | 3.7k |
| 16 | 10 | 0 | 512 | fp64 | 16M | 10M | 3.2k |
| 16 | 10 | 0 | 768 | fp32 | 16M | 8M | 3.6k |
| 16 | 10 | 0 | 768 | fp64 | 27M | 18M | 2.8k |
| 16 | 10 | 50 | 32 | fp32 | 18M | 7M | 4.4k |
| 16 | 10 | 50 | 32 | fp64 | 14M | 8M | 4.0k |
| 16 | 10 | 50 | 128 | fp32 | 17M | 7M | 4.1k |
| 16 | 10 | 50 | 128 | fp64 | 17M | 10M | 3.6k |
| 16 | 10 | 50 | 512 | fp32 | 15M | 8M | 3.7k |
| 16 | 10 | 50 | 512 | fp64 | 18M | 9M | 3.5k |
| 16 | 10 | 50 | 768 | fp32 | 18M | 8M | 3.8k |
| 16 | 10 | 50 | 768 | fp64 | 20M | 12M | 3.1k |
| 16 | 25 | 0 | 32 | fp32 | 17M | 7M | 3.6k |
| 16 | 25 | 0 | 32 | fp64 | 14M | 8M | 3.4k |
| 16 | 25 | 0 | 128 | fp32 | 12M | 8M | 3.4k |
| 16 | 25 | 0 | 128 | fp64 | 10M | 8M | 3.2k |
| 16 | 25 | 0 | 512 | fp32 | 13M | 10M | 3.0k |
| 16 | 25 | 0 | 512 | fp64 | 10M | 10M | 2.8k |
| 16 | 25 | 0 | 768 | fp32 | 17M | 11M | 2.8k |
| 16 | 25 | 0 | 768 | fp64 | 12M | 12M | 2.8k |
| 16 | 25 | 50 | 32 | fp32 | 16M | 7M | 3.6k |
| 16 | 25 | 50 | 32 | fp64 | 11M | 8M | 3.4k |
| 16 | 25 | 50 | 128 | fp32 | 16M | 12M | 3.1k |
| 16 | 25 | 50 | 128 | fp64 | 14M | 9M | 3.0k |
| 16 | 25 | 50 | 512 | fp32 | 11M | 8M | 3.1k |
| 16 | 25 | 50 | 512 | fp64 | 12M | 9M | 3.0k |
| 16 | 25 | 50 | 768 | fp32 | 11M | 9M | 2.9k |
| 16 | 25 | 50 | 768 | fp64 | 16M | 9M | 2.7k |

### BM_TimeBasedFlushLowRate_VectorDB

Low write rate (10 ms delay between writes). 100 writes/thread.

| interval_ms | num_threads | dim | dtype | wall_ops_per_sec |
|--------------|-------------|-----|-------|------------------|
| 0 | 1 | 32 | fp32 | 83.1 |
| 0 | 1 | 32 | fp64 | 82.7 |
| 0 | 1 | 128 | fp32 | 82.9 |
| 0 | 1 | 128 | fp64 | 83.0 |
| 0 | 1 | 512 | fp32 | 81.8 |
| 0 | 1 | 512 | fp64 | 82.5 |
| 0 | 1 | 768 | fp32 | 82.3 |
| 0 | 1 | 768 | fp64 | 83.2 |
| 0 | 2 | 32 | fp32 | 165.8 |
| 0 | 2 | 32 | fp64 | 165.2 |
| 0 | 2 | 128 | fp32 | 166.0 |
| 0 | 2 | 128 | fp64 | 164.1 |
| 0 | 2 | 512 | fp32 | 166.6 |
| 0 | 2 | 512 | fp64 | 164.3 |
| 0 | 2 | 768 | fp32 | 164.9 |
| 0 | 2 | 768 | fp64 | 166.0 |
| 50 | 1 | 32 | fp32 | 84.4 |
| 50 | 1 | 32 | fp64 | 82.2 |
| 50 | 1 | 128 | fp32 | 82.4 |
| 50 | 1 | 128 | fp64 | 83.0 |
| 50 | 1 | 512 | fp32 | 82.9 |
| 50 | 1 | 512 | fp64 | 83.0 |
| 50 | 1 | 768 | fp32 | 82.8 |
| 50 | 1 | 768 | fp64 | 82.0 |
| 50 | 2 | 32 | fp32 | 166.4 |
| 50 | 2 | 32 | fp64 | 165.7 |
| 50 | 2 | 128 | fp32 | 165.6 |
| 50 | 2 | 128 | fp64 | 165.8 |
| 50 | 2 | 512 | fp32 | 165.5 |
| 50 | 2 | 512 | fp64 | 166.0 |
| 50 | 2 | 768 | fp32 | 164.3 |
| 50 | 2 | 768 | fp64 | 164.0 |

### BM_TimeBasedFlushScaling_VectorDB

Throughput scaling: threads × interval. 500 writes/thread. (Some combinations omitted where JSON was truncated.)

| num_threads | interval_ms | dim | dtype | wall_ops_per_sec |
|-------------|--------------|-----|-------|------------------|
| 1 | 0 | 32 | fp32 | 133.8k |
| 1 | 0 | 32 | fp64 | 136.5k |
| 1 | 0 | 128 | fp32 | 128.7k |
| 1 | 0 | 128 | fp64 | 101.9k |
| 1 | 0 | 512 | fp32 | 80.4k |
| 1 | 0 | 768 | fp32 | 64.8k |
| 1 | 50 | 32 | fp32 | 136.5k |
| 1 | 50 | 32 | fp64 | 140.2k |
| 1 | 50 | 128 | fp32 | 124.3k |
| 1 | 50 | 512 | fp32 | 77.9k |
| 1 | 50 | 768 | fp32 | 61.5k |
| 1 | 100 | 32 | fp32 | 148.6k |
| 1 | 100 | 32 | fp64 | 140.2k |
| 1 | 100 | 128 | fp32 | 115.2k |
| 1 | 100 | 512 | fp32 | 78.0k |
| 1 | 100 | 768 | fp32 | 58.5k |
| 4 | 0 | 32 | fp32 | 409.2k |
| 4 | 0 | 32 | fp64 | 374.9k |
| 4 | 0 | 128 | fp32 | 302.8k |
| 4 | 0 | 128 | fp64 | 199.9k |
| 4 | 0 | 512 | fp32 | 116.6k |
| 4 | 0 | 768 | fp32 | 86.0k |
| 4 | 50 | 32 | fp32 | 388.4k |
| 4 | 50 | 32 | fp64 | 384.3k |
| 4 | 50 | 128 | fp32 | 297.9k |
| 4 | 50 | 512 | fp32 | 114.9k |
| 4 | 50 | 768 | fp32 | 75.0k |
| 4 | 100 | 32 | fp32 | 427.3k |
| 4 | 100 | 32 | fp64 | 381.1k |
| 4 | 100 | 128 | fp32 | 300.0k |
| 4 | 100 | 512 | fp32 | 116.1k |
| 4 | 100 | 768 | fp32 | 84.6k |
| 8 | 0 | 32 | fp32 | 305.6k |
| 8 | 0 | 32 | fp64 | 317.2k |
| 8 | 0 | 128 | fp32 | 238.5k |
| 8 | 0 | 128 | fp64 | 205.9k |
| 8 | 0 | 512 | fp32 | 126.3k |
| 8 | 0 | 768 | fp32 | 72.2k |
| 8 | 50 | 32 | fp32 | 325.4k |
| 8 | 50 | 32 | fp64 | 250.0k |
| 8 | 50 | 128 | fp32 | 283.5k |
| 8 | 50 | 512 | fp32 | 100.8k |
| 8 | 50 | 768 | fp32 | 86.9k |
| 8 | 100 | 32 | fp32 | 332.3k |
| 8 | 100 | 32 | fp64 | 330.6k |
| 8 | 100 | 128 | fp32 | 235.2k |
| 8 | 100 | 512 | fp32 | 118.9k |
| 8 | 100 | 768 | fp32 | 56.5k |
| 16 | 0 | 32 | fp32 | 235.3k |
| 16 | 0 | 32 | fp64 | 236.3k |
| 16 | 0 | 128 | fp32 | 204.4k |
| 16 | 0 | 512 | fp32 | 125.2k |
| 16 | 0 | 768 | fp32 | 89.7k |
| 16 | 50 | 32 | fp32 | 231.5k |
| 16 | 50 | 32 | fp64 | 232.3k |
| 16 | 50 | 128 | fp32 | 215.5k |
| 16 | 50 | 512 | fp32 | 125.4k |
| 16 | 50 | 768 | fp32 | 94.0k |
| 16 | 100 | 32 | fp32 | 237.4k |
| 16 | 100 | 32 | fp64 | 251.7k |
| 16 | 100 | 128 | fp32 | 250.6k |
| 16 | 100 | 512 | fp32 | 125.8k |
| 16 | 100 | 768 | fp32 | 94.2k |

---

## wal_write_bench Results

### BM_WALWrite_VectorDB

Raw log writer: 1000 records per iteration. Reports bytes/sec.

| dim | dtype | Time (ns) | MB/s |
|-----|-------|-----------|------|
| 32 | fp32 | 36,954,610 | 26.3 |
| 128 | fp32 | 51,520,615 | 73.6 |
| 512 | fp32 | 145,674,066 | 120 |
| 768 | fp32 | 218,481,179 | 116 |
| 32 | fp64 | 43,383,231 | 42.0 |
| 128 | fp64 | 95,661,947 | 85.1 |
| 512 | fp64 | 17,053,737 | 251 |
| 768 | fp64 | 22,589,077 | 289 |

**Observation:** Larger embeddings increase write time; fp64 512/768 can show higher MB/s in some runs (disk/cache dependent).

### BM_WALWriteSync_VectorDB

Log write with optional per-record `WriteBuffer()` flush. 100 records of 768-dim fp32 (3072 B each).

**Note:** `WriteBuffer()` flushes to page cache; it is NOT fsync to disk.

| per_record_sync | Time (ns) | MB/s | Items/sec |
|-----------------|-----------|------|-----------|
| 0 (batch flush) | 906,843 | 349 | 119k/s |
| 1 (per-record) | 1,206,987 | 266 | 91k/s |

### BM_WriteBatchEncode_VectorDB

WriteBatch encode (Put only) and GetDataSize(). CPU-only, no I/O.

| num_ops | 32 fp32 | 128 fp32 | 512 fp32 | 768 fp32 | 32 fp64 | 768 fp64 |
|---------|---------|----------|----------|----------|---------|----------|
| 1 | 123 | 136 | 192 | 234 | 126 | 397 |
| 10 | 758 | 987 | 2,380 | 3,012 | 837 | 6,153 |
| 100 | 5,825 | 9,155 | 20,166 | 26,699 | 7,079 | 52,203 |
| 1000 | 57,343 | 80,038 | 167,073 | 281,633 | 64,850 | 494,703 |

**Observation:** Encode time scales roughly linearly with num_ops and record size (dim × bytes per float).


---

## wal_read_bench Results

### BM_WALRead_VectorDB

Read 1000 records. File pre-written outside timer.

| dim | dtype | Time (ns) | MB/s |
|-----|-------|-----------|------|
| 32 | fp32 | 316,331 | 436 |
| 128 | fp32 | 1,244,355 | 416 |
| 512 | fp32 | 4,778,822 | 417 |
| 768 | fp32 | 8,538,316 | 390 |
| 32 | fp64 | 1,406,247 | 346 |
| 128 | fp64 | 2,376,057 | 424 |
| 512 | fp64 | 9,609,842 | 411 |
| 768 | fp64 | 15,613,992 | 384 |

### BM_WALRecovery_VectorDB

Full read of log (recovery-style). Raw log layer, no DBWal.

| num_records | dim | dtype | Time (ns) | Items/sec |
|-------------|-----|-------|-----------|-----------|
| 1,000 | 32 | fp32 | 348,033 | 3.59M/s |
| 10,000 | 32 | fp32 | 2,592,842 | 4.02M/s |
| 100,000 | 32 | fp32 | 24,995,121 | 4.02M/s |
| 1,000 | 128 | fp32 | 1,250,321 | 842k/s |
| 10,000 | 128 | fp32 | 12,275,016 | 838k/s |
| 100,000 | 128 | fp32 | 120,705,389 | 839k/s |
| 1,000 | 512 | fp32 | 4,917,612 | 207k/s |
| 10,000 | 512 | fp32 | 49,012,631 | 205k/s |
| 100,000 | 512 | fp32 | 573,769,458 | 196k/s |
| 1,000 | 768 | fp32 | 7,439,806 | 136k/s |
| 10,000 | 768 | fp32 | 73,872,218 | 136k/s |
| 100,000 | 768 | fp32 | 745,437,750 | 135k/s |
| 1,000 | 32 | fp64 | 661,092 | 1.70M/s |
| 10,000 | 32 | fp64 | 7,394,938 | 1.66M/s |
| 100,000 | 32 | fp64 | 56,345,042 | 1.78M/s |
| 1,000 | 128 | fp64 | 2,502,739 | 413k/s |
| 10,000 | 128 | fp64 | 24,167,455 | 415k/s |
| 100,000 | 128 | fp64 | 242,562,903 | 414k/s |
| 1,000 | 512 | fp64 | 9,901,892 | 102k/s |
| 10,000 | 512 | fp64 | 99,079,958 | 102k/s |
| 100,000 | 512 | fp64 | 1,041,027,708 | 100k/s |
| 1,000 | 768 | fp64 | 14,898,191 | 67.6k/s |
| 10,000 | 768 | fp64 | 147,690,258 | 67.8k/s |
| 100,000 | 768 | fp64 | 1,590,080,166 | 66.3k/s |

**Observation:** Recovery Items/sec drops with embedding size (32-dim: ~4M/s; 768 fp64: ~66k/s).


---

## wal_sync_bench Results

### BM_FsyncLatency_VectorDB

Append vector-sized data then fsync each iteration.

| dim | dtype | Time (ns/op) |
|-----|-------|--------------|
| 32 | fp32 | 4,539,734 |
| 128 | fp32 | 4,537,152 |
| 512 | fp32 | 4,457,379 |
| 768 | fp32 | 4,239,981 |
| 32 | fp64 | 4,220,451 |
| 128 | fp64 | 4,195,771 |
| 512 | fp64 | 4,435,402 |
| 768 | fp64 | 4,416,868 |

**Note:** ~4.2–4.5 ms baseline fsync dominates; payload size has minimal effect.

### BM_GroupCommit_VectorDB

Raw log writer: multiple threads append through mutex-protected `AddRecord()`, single flush at end. 100 records/thread.

**fp32 (dtype=0):**

| num_threads | 32 | 128 | 512 | 768 |
|-------------|-----|-----|-----|-----|
| 1 | 1.76M/s | 2.42M/s | 2.39M/s | 2.22M/s |
| 2 | 5.11M/s | 4.13M/s | 3.70M/s | 3.50M/s |
| 4 | 6.37M/s | 6.02M/s | 5.33M/s | 4.33M/s |
| 8 | 8.63M/s | 8.11M/s | 6.57M/s | 5.26M/s |

**fp64 (dtype=1):**

| num_threads | 32 | 128 | 512 | 768 |
|-------------|-----|-----|-----|-----|
| 1 | 2.86M/s | 2.44M/s | 1.97M/s | (run benchmark) |
| 2 | 4.49M/s | 3.86M/s | 2.79M/s | (run benchmark) |
| 4 | 6.21M/s | 5.09M/s | 4.34M/s | (run benchmark) |
| 8 | 8.45M/s | 7.26M/s | (run benchmark) | (run benchmark) |

**Observation:** Items/sec scales with thread count; larger payloads reduce throughput.

---

## Summary

- **BM_DBWalWrite_VectorDB:** ops_per_batch 1→100 yields ~15–28x throughput gain; 768 fp64 has lowest ops/sec (~1.5M/s at 100 batch).
- **BM_DBWalWriteSync_VectorDB:** sync=true ~1500x slower than async due to per-write fsync.
- **BM_DBWalConcurrentWrite_VectorDB:** Throughput scales with threads (1→16: ~7.8x for 32-dim); larger embeddings reduce ops/sec.
- **BM_DBWalRecovery_VectorDB:** Recovery rate drops with embedding size (32-dim: ~2.4M/s at 100k; 768 fp64: ~65k/s).
- **BM_GroupCommitScaling_VectorDB:** Sync wall_ops_per_sec scales ~3.7x from 1→8 threads.
- **BM_WalIteratorScanSpeed_VectorDB:** ~1.5–2.7M records/s (32-dim); ~57–134k records/s (768-dim); MB/s ~335–400.
- **BM_RecoveryWithCompression_VectorDB:** Zstd can reach ~541 MB/s (768 fp64, 100k records) when compressed size is small.
- **BM_WriteStallRecoveryTime_VectorDB:** stall_recovery ~660–1300 µs depending on dim/dtype.
- **BM_FsyncLatency_VectorDB:** ~4.2–4.5 ms per fsync; payload size has minimal effect.
- **BM_WALWrite_VectorDB:** 26–289 MB/s depending on dim/dtype; fp64 512/768 can exceed fp32 in some runs.
- **BM_WALRead_VectorDB:** 384–436 MB/s; consistent across dim/dtype.
- **BM_WALRecovery_VectorDB (raw log):** 66k/s (768 fp64) to 4M/s (32 fp32).

## Methodology Notes

- **Payload generation:** Embedding values use `i/dim` for reproducibility and to avoid compiler optimizations from constant folding.
- **Latency benchmarks** (`BM_DBWalWriteLatencyUnderThroughput_VectorDB`): Batches are pre-built outside the timing window so only `DBWal::Write()` is measured. Percentiles use linear interpolation.
- **Wall-time vs CPU-time throughput:** For I/O-bound sync writes, Google Benchmark's `items_per_second` uses CPU time and can overestimate throughput. Use `wall_ops_per_sec` where reported.
- **ZSTD benchmarks:** Require `MWAL_HAVE_ZSTD` (built with zstd support).
- **Delete operations:** Current benchmarks use all-Put (add/update). A separate `BM_*_VectorDB_Mixed` (e.g. 80% Put, 20% Delete) can be added if delete behavior is critical.
