# Eigenix K-Means Benchmarking Suite

Production-level C++ benchmarking suite comparing K-Means clustering across three backends on CPU. Part of the [BharatMLStack](https://github.com/BharatMLStack) project.

## Backends

| Backend | Strategy |
|---------|----------|
| **FAISS** | `faiss::Clustering` + `IndexFlatL2` for assignment |
| **BLAS** | `cblas_sgemm` for the distance cross-term (`\|\|x-c\|\|^2 = \|\|x\|\|^2 - 2x^Tc + \|\|c\|\|^2`), OpenMP argmin |
| **SIMD** | Hand-rolled AVX-512 / AVX2 / SSE4.2 L2 kernels with FMA and cache prefetch, runtime ISA detection via CPUID |

All backends share a common abstract interface (`KMeansBase`). BLAS and SIMD use random initialization (with optional KMeans++); FAISS uses its internal random init. See [KMEANS_IMPLEMENTATION.md](KMEANS_IMPLEMENTATION.md) for a full implementation summary and comparison with FAISS.

## Project Structure

```
eigenix/
├── CMakeLists.txt              # CMake 3.20+, C++17
├── setup.sh                    # Dependency installer (Ubuntu 22.04)
├── run_bench.sh                # Build + run full suite
├── include/
│   ├── kmeans_base.hpp         # Abstract interface + ClusterStats + TrainConfig
│   ├── kmeans_blas.hpp         # BLAS backend
│   ├── kmeans_faiss.hpp        # FAISS backend
│   ├── kmeans_simd.hpp         # SIMD backend (AVX-512 / AVX2 / SSE4.2)
│   ├── data_generator.hpp      # Mixture-of-Gaussians synthetic data
│   ├── metrics.hpp             # Inertia, cluster stats, purity, balance metrics
│   └── bench_utils.hpp         # Timer, peak RSS, CSV writer, governor check
├── src/
│   ├── kmeans_blas.cpp
│   ├── kmeans_faiss.cpp
│   ├── kmeans_simd.cpp
│   ├── data_generator.cpp
│   └── metrics.cpp
├── bench/
│   └── main_bench.cpp          # Full benchmark harness
├── tests/
│   └── test_correctness.cpp    # GTest suite (9 test cases)
└── results/
    └── .gitkeep
```

## Prerequisites

- C++17 compiler (GCC 9+ or Clang 10+)
- CMake 3.20+
- OpenBLAS (or MKL)
- FAISS (CPU-only build)
- OpenMP
- Google Test (for tests)

### Quick Install (Ubuntu 22.04)

```bash
chmod +x setup.sh
./setup.sh
```

This installs OpenBLAS, FAISS (CPU, built from source), GTest, and CMake.

## Build

```bash
cmake -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build -j$(nproc)
```

Compile flags applied automatically: `-O3 -march=native -ffast-math -fopenmp`.

## Run Benchmarks

```bash
chmod +x run_bench.sh
./run_bench.sh
```

Or manually:

```bash
export OMP_PROC_BIND=close
export OMP_PLACES=cores
./build/eigenix_bench
```

Results are written to `results/benchmark_results.csv` and printed to stdout.

## Run Tests

```bash
cd build && ctest --output-on-failure
```

Or directly:

```bash
./build/eigenix_tests
```

## Dataset Specification

- Synthetic float32 vectors generated from a mixture of 50 Gaussians
- Dimensionality: **D = 128**
- Default benchmark size: **10M vectors** (overridable via `EIGENIX_BENCH_N`)
- Number of centroids: **K = 1000** (overridable via `EIGENIX_BENCH_K`)
- Seeded RNG for reproducibility (seed = 42)
- All data in-memory, no disk I/O during benchmarks

**Environment variables** (for `eigenix_bench` / `run_bench.sh`):

| Variable | Default | Description |
|----------|---------|-------------|
| `EIGENIX_BENCH_N` | `10000000` | Comma-separated dataset sizes (e.g. `10000000` or `50000000,10000000`) |
| `EIGENIX_BENCH_K` | `1000` | Number of clusters |
| `EIGENIX_BENCH_RUNS` | `1` | Runs per backend per N (use 3 for averaged timings) |
| `EIGENIX_BENCH_WARMUP` | `10000` | Warmup set size |
| `EIGENIX_BENCH_DATA_SEED` | `42` (unset) | Data generation seed. Unset = 42 (reproducible). Set to `random` or `rand` for a random seed each run; or set to a number (e.g. `123`) for that seed. |

FAISS training uses the **full** N (no internal subsampling); other backends train on all N as well.

## Metrics Collected

| Metric | Description |
|--------|-------------|
| `train_time_ms` | Full training until convergence or max 100 iterations |
| `assign_time_ms` | Time to assign all N vectors to nearest centroid |
| `inertia` | Final within-cluster sum of squared distances |
| `iterations` | Actual iterations to converge |
| `throughput_Mvecs_per_sec` | Vectors assigned per second (millions) |
| `memory_MB` | Peak resident set size |
| `min_dist_per_cluster` | Closest point to centroid per cluster |
| `max_dist_per_cluster` | Furthest point from centroid per cluster (radius) |
| `mean_dist_per_cluster` | Mean L2 distance within each cluster |
| `cluster_size_min` | Smallest cluster by point count |
| `cluster_size_max` | Largest cluster by point count |
| `cluster_size_stddev` | Std deviation of cluster sizes |
| `empty_clusters` | Count of centroids with zero assigned points |

Streaming additionally records `batch_size` and streaming vs batch throughput.

## Benchmark Methodology

- Each backend is **warmed up** with 100K vectors before timed runs
- Each configuration runs **3 times**; min/mean/max reported
- Thread pinning via `OMP_PROC_BIND=close` / `OMP_PLACES=cores`
- CPU governor check on Linux (warns if not `performance`)
- CSV output + pretty-printed comparison table + per-backend cluster health report

### Output Example

```
Backend            |        N |  Train(ms) |  Assign(ms) |      Inertia | Iters |   MVec/s |  Mem(MB)
----------------------------------------------------------------------------------------------------
FAISS              |  1000000 |     1234.0 |      210.0  |   4.3200e+07 |    87 |   4761.0 |    512.0
BLAS               |  1000000 |     1456.0 |      198.0  |   4.3100e+07 |    92 |   5050.0 |    490.0
SIMD-AVX512        |  1000000 |      987.0 |      145.0  |   4.3300e+07 |    91 |   6896.0 |    480.0
```

```
Cluster Health Report (SIMD-AVX512, N=5000000):
  Empty clusters   : 0 / 1000
  Size range       : 3821 - 6104  (mean: 5000, stddev: 312)
  Dist min/max/mean: 0.003 / 14.72 / 3.41
  Max radius ratio : 4.31  (cluster #847)
```

## Test Suite

9 correctness tests (Google Test):

| # | Test | Assertion |
|---|------|-----------|
| 1 | Two-cluster purity | > 95% purity on well-separated 2-cluster data |
| 2 | Convergence | Inertia decreases monotonically across iterations |
| 3 | Determinism | Same seed produces identical centroids |
| 4 | FAISS vs SIMD consistency | Reasonable assignment agreement on same data |
| 5 | Throughput regression | Assignment exceeds 500M float ops/sec |
| 6 | No empty clusters | Zero empty clusters on balanced data |
| 7 | Radius sanity | `max_dist >= min_dist` for every cluster |
| 8 | Size balance | `stddev / mean < 0.5` on uniform data |
| 9 | Outlier detection | Injected outliers are the max-dist points in their clusters |

## Abstract Interface

All backends implement:

```cpp
namespace eigenix {

struct ClusterStats {
    float min_dist;       // closest point to centroid
    float max_dist;       // furthest point (cluster radius)
    float mean_dist;
    float radius_ratio;   // max_dist / mean_dist
    size_t count;
};

struct TrainConfig {
    size_t max_iter = 100;
    float tol = 1e-4f;
    unsigned seed = 42;
};

class KMeansBase {
public:
    virtual void train(const float* data, size_t n, int dim, int k,
                       const TrainConfig& cfg = {}) = 0;
    virtual void assign(const float* data, size_t n, int dim,
                        std::vector<int>& labels) const = 0;
    virtual const float* centroids() const = 0;
    virtual float inertia() const = 0;
    virtual int iterations() const = 0;
    virtual std::vector<ClusterStats> cluster_stats(
        const float* data, size_t n, int dim) const = 0;
    virtual std::string name() const = 0;
};

}  // namespace eigenix
```

## SIMD Backend Details

Runtime ISA detection via CPUID with three kernel tiers:

| ISA | Register Width | Floats/Iteration | Key Intrinsic |
|-----|---------------|-------------------|---------------|
| AVX-512 | 512-bit | 16 | `_mm512_fmadd_ps` |
| AVX2 | 256-bit | 8 | `_mm256_fmadd_ps` |
| SSE4.2 | 128-bit | 4 | `_mm_mul_ps` + `_mm_add_ps` |

All kernels use `_mm_prefetch` for the next cache line during distance loops. The outer (per-vector) loop is parallelised with OpenMP.

## Expected Performance Characteristics

| Backend | Training | Assignment | Memory |
|---------|----------|------------|--------|
| **BLAS** | Fastest at large N (batched SGEMM, cache-friendly) | Fast | Moderate (batched) |
| **FAISS** | Slower (index overhead per iteration) | Fast (IndexFlatL2) | High |
| **SIMD-AVX512** | Good | Fastest (widest registers, FMA, prefetch) | Moderate |

## Thread Control

- Set thread count via `OMP_NUM_THREADS` or let it auto-detect
- For BLAS, set `OPENBLAS_NUM_THREADS=1` to avoid nested parallelism (our code manages OpenMP threading)
- Pin threads with `OMP_PROC_BIND=close` for stable benchmark results

## License

See the root repository license.
