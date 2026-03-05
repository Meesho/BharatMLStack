# VM Vector DB Benchmark Results

Benchmark results run on a Linux VM: **BM_DBWalWrite_VectorDB** (single-thread write throughput) and **BM_MixedReadWrite_VectorDB** (mixed read/write with 20k writes/s, 5k records per read). `ops_per_batch` = operations per batch; `dim` = embedding dimension; **dtype**: 0 = fp32, 1 = fp64.

---

## VM / environment details

| Field | Value |
|-------|--------|
| **Date** | 2026-03-05 (Thu Mar 5; MixedReadWrite), 2026-03-03 (BM_DBWalWrite) |
| **Host** | vm-datascience-mlp-vss-ngt-poc-1-prd-ase1 (Linux VM) |
| **CPUs** | 16 × 3050 MHz |
| **L1 Data** | 32 KiB (×8) |
| **L1 Instruction** | 32 KiB (×8) |
| **L2 Unified** | 512 KiB (×8) |
| **L3 Unified** | 32 MiB (×2) |
| **Binary** | `./bench/db_wal_bench` |

*(Add OS version, RAM, and disk type here if you have them, e.g. from `uname -a`, `free -h`, or cloud VM spec.)*

---

## BM_MixedReadWrite_VectorDB

Mixed read/write workload on the same VM: writers throttled to **~20k inserts/s** (1 insert per batch), readers each do **5k-record** chunk reads. Fixed **dim 128, fp32**. Parameter **read_pct** (10–90) = percentage of threads that are readers (10 threads total). Phase duration **5 s**. Run: `./bench/db_wal_bench --benchmark_filter='BM_MixedReadWrite_VectorDB'`.

### Throughput

| read_pct | Wall time (ns) | write_ops_per_sec | read_5k_ops_per_sec | records_read_per_sec | read_MB_per_sec |
|----------|----------------|-------------------|----------------------|----------------------|------------------|
| 10       | 5,118,232,163  | 19.998k           | 40                  | 130.54k              | 67.88            |
| 20       | 5,124,793,714  | 19.9976k          | 80.2                 | 261.60k              | 136.03           |
| 30       | 5,073,852,006  | 19.998k           | 122.4                | 407.57k              | 211.94           |
| 40       | 5,127,166,834  | 19.9988k          | 165.6                | 572.58k              | 297.74           |
| 50       | 5,114,267,675  | 19.9998k          | 208.4                | 732.82k              | 381.06           |
| 60       | 5,106,215,074  | 20k               | 247.6                | 855.27k              | 444.74           |
| 70       | 5,142,984,042  | 19.998k           | 290.8                | 1.01351M             | 527.03           |
| 80       | 5,092,527,195  | 19.9966k          | 335.6                | 1.21112M             | 629.78           |
| 90       | 5,107,057,654  | 19.9982k          | 398.6                | 1.57388M             | 818.42           |

### Write latency (ns)

Per single-batch write. p50/p95/p99 in nanoseconds.

| read_pct | write_p50_ns | write_p95_ns | write_p99_ns |
|----------|--------------|--------------|--------------|
| 10       | 50.34k       | 94.36k       | 128.41k      |
| 20       | 44.03k       | 82.38k       | 112.61k      |
| 30       | 40.48k       | 75.37k       | 103.41k      |
| 40       | 38.28k       | 75.91k       | 101.58k      |
| 50       | 38.72k       | 73.42k       | 94.80k       |
| 60       | 36.29k       | 56.90k       | 71.99k       |
| 70       | 35.94k       | 46.19k       | 56.43k       |
| 80       | 31k          | 40.80k       | 46.84k       |
| 90       | 28.11k       | 34.75k       | 41.03k       |

### Read latency (ns)

Per 5k-record read op. p50/p95/p99 in nanoseconds (M = millions of ns, e.g. 18.29M ≈ 18.3 ms).

| read_pct | read_p50_ns | read_p95_ns | read_p99_ns |
|----------|-------------|-------------|-------------|
| 10       | 18.29M      | 82.47M      | 113.37M     |
| 20       | 17.54M      | 82.34M      | 119.40M     |
| 30       | 15.76M      | 87.32M      | 112.84M     |
| 40       | 13.93M      | 83.22M      | 114.91M     |
| 50       | 12.62M      | 84.61M      | 114.34M     |
| 60       | 14.14M      | 80.36M      | 112.41M     |
| 70       | 12.79M      | 82.17M      | 117.98M     |
| 80       | 11.30M      | 85.00M      | 115.02M     |
| 90       | 13.30M      | 80.86M      | 114.54M     |

### Records per read (mean)

Actual records consumed per 5k-read op (often &lt; 5k when readers catch up or near end of snapshot).

| read_pct | records_per_read_mean |
|----------|------------------------|
| 10       | 3.26k                 |
| 20       | 3.26k                 |
| 30       | 3.33k                 |
| 40       | 3.46k                 |
| 50       | 3.52k                 |
| 60       | 3.45k                 |
| 70       | 3.49k                 |
| 80       | 3.61k                 |
| 90       | 3.95k                 |

**Observations**

- Write throughput stays at **~20k/s** across all read_pct; throttle is effective.
- Write latency **improves** as read_pct increases (fewer writer threads, less contention): p99 from ~128 µs (10% read) down to ~41 µs (90% read).
- Read throughput scales with reader count: **read_5k_ops_per_sec** and **records_read_per_sec** increase with read_pct.
- Read latency (per 5k-read op) is in the **tens of milliseconds** (p50 ~11–18 ms); p99 ~80–119 ms. Dominated by iterator setup and sequential read of 5k records (dim 128 fp32).

---

## BM_DBWalWrite_VectorDB

| ops_per_batch | dim | dtype | Time (ns)   | Items/sec   |
|---------------|-----|-------|-------------|-------------|
| 1             | 32  | fp32  | 25,676,117  | 103.34k/s   |
| 10            | 32  | fp32  | 31,512,521  | 888.32k/s   |
| 100           | 32  | fp32  | 86,036,089  | 3.83M/s     |
| 1             | 128 | fp32  | 27,116,355  | 102.62k/s   |
| 10            | 128 | fp32  | 46,319,203  | 733.21k/s   |
| 100           | 128 | fp32  | 226,657,734 | 2.03M/s     |
| 1             | 512 | fp32  | 33,203,176  | 94.72k/s    |
| 10            | 512 | fp32  | 103,421,617 | 424.16k/s   |
| 100           | 512 | fp32  | 796,766,509 | 691.62k/s   |
| 1             | 768 | fp32  | 37,481,711  | 85.61k/s    |
| 10            | 768 | fp32  | 143,410,082 | 314.64k/s   |
| 100           | 768 | fp32  | 1,489,311,511 | 190.17k/s |
| 1             | 32  | fp64  | 26,306,725  | 104.04k/s   |
| 10            | 32  | fp64  | 37,165,589  | 833.59k/s   |
| 100           | 32  | fp64  | 131,243,121 | 3.13M/s     |
| 1             | 128 | fp64  | 29,690,909  | 96.35k/s    |
| 10            | 128 | fp64  | 65,359,653  | 605.30k/s   |
| 100           | 128 | fp64  | 404,874,395 | 1.44M/s     |
| 1             | 512 | fp64  | 41,775,068  | 82.71k/s    |
| 10            | 512 | fp64  | 178,460,212 | 288.93k/s   |
| 100           | 512 | fp64  | 1,946,268,909 | 149.92k/s |
| 1             | 768 | fp64  | 48,850,047  | 75.08k/s    |
| 10            | 768 | fp64  | 253,516,484 | 219.28k/s   |
| 100           | 768 | fp64  | 2,849,894,658 | 104.41k/s |

*dtype: embedding element type — fp32 (float) or fp64 (double). BM_DBWalWrite_VectorDB does not vary sync; all runs use default WriteOptions (no per-write fsync).*
