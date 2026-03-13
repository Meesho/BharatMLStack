# Eigenix — K-Means Benchmarking & Distributed BLAS

## What This Is

A C++17 K-Means clustering library benchmarking three backends (BLAS, SIMD, FAISS) on CPU.
BLAS won the single-node POC. A distributed extension (`distributed/`) scales BLAS K-Means
across multiple VMs via raw TCP sockets (no MPI/ZMQ).

Part of the [BharatMLStack](https://github.com/BharatMLStack) project — building toward a vector database.

## Build

```bash
cmake -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build -j$(nproc)
```

Dependencies: OpenBLAS, OpenMP, FAISS (CPU), GTest (optional). Install via `./setup.sh`.

Produces: `eigenix_bench`, `eigenix_tests`, `dist_coordinator`, `dist_worker`.

## Architecture

### Core Library (`eigenix_kmeans` static lib)

All backends implement `KMeansBase` (pure virtual):
- `train()`, `assign()`, `centroids()`, `inertia()`, `iterations()`, `cluster_stats()`, `name()`

Protected members in `KMeansBase`: `k_`, `dim_`, `inertia_`, `iterations_`, `centroids_`.

Config: `TrainConfig { max_iter=100, tol=1e-4, seed=42, verbose=false, nredo=1 }`.

### Backends

| Backend | File | Strategy |
|---------|------|----------|
| **BlasKMeans** | `src/kmeans_blas.cpp` | `cblas_sgemm` for distance cross-term, OpenMP argmin. BATCH_SIZE=100k. |
| **SimdKMeans** | `src/kmeans_simd.cpp` | Hand-rolled AVX-512/AVX2/SSE4.2 L2 kernels with FMA + prefetch |
| **FaissKMeans** | `src/kmeans_faiss.cpp` | Wraps `faiss::Clustering` + `IndexFlatL2` |

### BlasKMeans Key Internals

- **Distance formula**: `||x-c||² = ||x||² - 2·x·cᵀ + ||c||²` (sgemm for the cross-term)
- `assign_batch()` — **protected**, processes BATCH_SIZE points at a time via sgemm
- `compute_centroid_norms()` — **protected**, precomputes `||c||²` for each centroid
- `fix_clusters(centroids, counts, k, dim, n_total)` — **public static**, Phase 1 (fill empty clusters by splitting largest) + Phase 2 (split oversized clusters > 3× mean)
- `centroid_norms_`, `dist_buf_` — **protected** vectors
- `train_once()`, `random_init()`, `kmeanspp_init()` — **private**

### Metrics (`include/metrics.hpp`, `src/metrics.cpp`)

Free functions in `eigenix::`:
- `compute_inertia(data, n, dim, labels, centroids, k)`
- `compute_cluster_sizes(labels, n, k, out_sizes)`
- `compute_imbalance_ratio(sizes, k)`
- `compute_cluster_stats(data, n, dim, labels, centroids, k)` → `vector<ClusterStats>`
- `compute_cluster_size_stddev(sizes, k)`
- `count_empty_clusters(sizes, k)`
- `compute_purity(pred_labels, true_labels, n, k_pred, k_true)`

### Data Generation (`include/data_generator.hpp`)

- `generate_gaussian_mixture(n, dim, n_clusters, seed)` → `vector<float>`
- `generate_ground_truth_labels(n, n_clusters, seed)` → `vector<int>`

## Distributed BLAS K-Means (`distributed/`)

### Protocol (`dist_protocol.hpp`)

Binary, length-prefixed. 16-byte `MsgHeader { magic=0xE16E1E16 (u32), msg_type (u32), payload_len (u64) }`.
`payload_len` is `uint64_t` to support shards > 4 GB (e.g. 5M × 2000-dim × 4 bytes = 40 GB).

| MsgType | Dir | Payload |
|---------|-----|---------|
| `SHARD_CONFIG` (0x01) | C→W | 32-byte `ShardConfig { n_local, dim, k, max_iter, tol, seed, worker_id, n_workers }` |
| `SHARD_DATA` (0x02) | C→W | `n_local × dim` raw floats |
| `CENTROIDS` (0x03) | C→W | `k × dim` floats (per iteration) |
| `DONE` (0x04) | C→W | empty |
| `READY` (0x10) | W→C | empty |
| `LOCAL_STATS` (0x11) | W→C | `k×dim` floats (sums) + `k` uint64_t (counts) |

### Network Layer (`dist_net.hpp/cpp`)

POSIX TCP sockets. Key functions:
- `send_all()` / `recv_all()` — loop until all bytes transferred
- `send_msg()` / `recv_msg()` — header + payload framing
- `make_listener()`, `accept_one()`, `connect_to()` (with retry), `close_fd()`
- `parse_host_port()` — splits `"host:port"` string

### Worker (`dist_worker.cpp`)

`DistWorkerKMeans` subclass of `BlasKMeans` (defined locally in the file):
- `set_centroids()` — injects centroids + calls `compute_centroid_norms()`
- Exposes `assign_batch()` via `using` declaration
- Thread-local sum/count accumulation (same pattern as `kmeans_blas.cpp`)

CLI: `dist_worker --port PORT [--threads T] [--verbose]`

### Coordinator (`dist_coordinator.cpp`)

- Reads `workers.txt` (one `HOST:PORT` per line, `#` comments ignored)
- Connects to each worker (coordinator is TCP client, workers listen)
- Generates full dataset via `generate_gaussian_mixture()`, subsamples `train_fraction` (default 0.3) for training
- Only training subset is sharded across workers (matches single-node bench methodology)
- Random centroid init inline (shuffle + memcpy from training data)
- Iteration loop: broadcast centroids → collect LOCAL_STATS → all-reduce → update → fix_clusters → check convergence
- Final: assign ALL n_total points locally → inertia, cluster sizes min/max/stddev, imbalance, throughput

CLI: `dist_coordinator --workers FILE --n N --k K --dim D [--max-iter I] [--tol T] [--train-fraction F] [--seed S] [--verbose]`

### Running

```bash
# Workers first (they listen), then coordinator connects:
./build/distributed/dist_worker --port 9001 &
./build/distributed/dist_worker --port 9002 &
./build/distributed/dist_coordinator --workers distributed/workers.txt.example \
  --n 1000000 --k 256 --dim 128 --verbose
```

Or: `./distributed/run_distributed.sh build distributed/workers.txt.example 1000000 256 128`

## File Layout

```
eigenix/
├── CMakeLists.txt           # Root build — eigenix_kmeans lib + bench + tests + distributed subdir
├── setup.sh                 # Installs OpenBLAS, FAISS v1.9.0, GTest, OpenMP
├── include/
│   ├── kmeans_base.hpp      # KMeansBase abstract class, TrainConfig, ClusterStats
│   ├── kmeans_blas.hpp      # BlasKMeans (protected: assign_batch, compute_centroid_norms)
│   ├── kmeans_simd.hpp      # SimdKMeans (runtime ISA detection)
│   ├── kmeans_faiss.hpp     # FaissKMeans
│   ├── metrics.hpp          # Free functions: inertia, sizes, stddev, purity, imbalance
│   ├── data_generator.hpp   # Gaussian mixture generator
│   └── bench_utils.hpp      # ScopedTimer, peak RSS, CsvWriter, BenchResult
├── src/                     # Implementations (~1100 lines total)
├── bench/main_bench.cpp     # Benchmark harness
├── tests/test_correctness.cpp  # 9 GTest cases
├── distributed/
│   ├── CMakeLists.txt       # dist_coordinator + dist_worker targets (link eigenix_kmeans)
│   ├── include/dist_protocol.hpp  # MsgType, MsgHeader, ShardConfig
│   ├── include/dist_net.hpp       # TCP helper declarations
│   ├── src/dist_net.cpp           # POSIX socket implementation
│   ├── src/dist_coordinator.cpp   # Coordinator main
│   ├── src/dist_worker.cpp        # Worker main + DistWorkerKMeans subclass
│   ├── workers.txt.example
│   └── run_distributed.sh
└── results/                 # CSV output directory
```

## Key Conventions

- C++17, `-O3 -ffast-math -march=native` (x86 only for march)
- OpenMP for intra-node threading; raw TCP for inter-node
- All float data is row-major `float*` (not `double`)
- BATCH_SIZE = 100,000 for sgemm calls
- Convergence: `max_shift ≤ tol` where shift = L2 movement of each centroid
- Cluster repair: Phase 1 (empty → split largest) + Phase 2 (oversized > 3× mean → split)
- Thread-local buffers for centroid sum accumulation to avoid OpenMP contention
- `eigenix_kmeans` static lib links OpenBLAS + OpenMP + FAISS publicly — downstream targets only need `target_link_libraries(... eigenix_kmeans)`

## Tests

```bash
./build/eigenix_tests   # or: cd build && ctest --output-on-failure
```

9 GTest cases: purity, convergence monotonicity, determinism, cross-backend consistency,
throughput regression, no empty clusters, radius sanity, size balance, outlier detection.

## Environment Variables (Benchmarks)

| Variable | Default | Description |
|----------|---------|-------------|
| `EIGENIX_BENCH_N` | `10000000` | Dataset sizes (comma-separated) |
| `EIGENIX_BENCH_K` | `1000` | Number of clusters |
| `EIGENIX_BENCH_RUNS` | `1` | Runs per config |
| `EIGENIX_BENCH_WARMUP` | `10000` | Warmup set size |
| `EIGENIX_BENCH_DATA_SEED` | `42` | RNG seed (`random` for non-deterministic) |
