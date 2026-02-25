# Eigenix K-Means Benchmarking Suite

Production-level C++ benchmarking suite comparing K-Means clustering across four backends on CPU. Part of the [BharatMLStack](https://github.com/BharatMLStack) project.

## Backends

| Backend | Strategy |
|---------|----------|
| **FAISS** | `faiss::Clustering` + `IndexFlatL2` for assignment |
| **BLAS** | `cblas_sgemm` for the distance cross-term (`\|\|x-c\|\|^2 = \|\|x\|\|^2 - 2x^Tc + \|\|c\|\|^2`), OpenMP argmin |
| **SIMD** | Hand-rolled AVX-512 / AVX2 / SSE4.2 L2 kernels with FMA and cache prefetch, runtime ISA detection via CPUID |
| **Streaming** | Mini-batch K-Means (Sculley 2010) with per-centroid learning-rate decay, pluggable assignment backend |

All backends share a common abstract interface (`KMeansBase`) and use KMeans++ initialisation.

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
│   ├── kmeans_streaming.hpp    # Streaming mini-batch backend
│   ├── data_generator.hpp      # Mixture-of-Gaussians synthetic data
│   ├── metrics.hpp             # Inertia, cluster stats, purity, balance metrics
│   └── bench_utils.hpp         # Timer, peak RSS, CSV writer, governor check
├── src/
│   ├── kmeans_blas.cpp
│   ├── kmeans_faiss.cpp
│   ├── kmeans_simd.cpp
│   ├── kmeans_streaming.cpp
│   ├── data_generator.cpp
│   └── metrics.cpp
├── bench/
│   └── main_bench.cpp          # Full benchmark harness
├── tests/
│   └── test_correctness.cpp    # GTest suite (10 test cases)
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
- Benchmark sizes: **1M, 2M, 5M, 10M vectors**
- Number of centroids: **K = 1000**
- Seeded RNG for reproducibility (seed = 42)
- All data in-memory, no disk I/O during benchmarks

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
Streaming-Brute    |  1000000 |     1100.0 |      145.0  |   4.5100e+07 |    10 |   6896.0 |    160.0
```

```
Cluster Health Report (SIMD-AVX512, N=5000000):
  Empty clusters   : 0 / 1000
  Size range       : 3821 - 6104  (mean: 5000, stddev: 312)
  Dist min/max/mean: 0.003 / 14.72 / 3.41
  Max radius ratio : 4.31  (cluster #847)
```

## Test Suite

10 correctness tests (Google Test):

| # | Test | Assertion |
|---|------|-----------|
| 1 | Two-cluster purity | > 95% purity on well-separated 2-cluster data |
| 2 | Convergence | Inertia decreases monotonically across iterations |
| 3 | Determinism | Same seed produces identical centroids |
| 4 | FAISS vs SIMD consistency | Reasonable assignment agreement on same data |
| 5 | Streaming vs batch delta | Streaming inertia within 15% of batch |
| 6 | Throughput regression | Assignment exceeds 500M float ops/sec |
| 7 | No empty clusters | Zero empty clusters on balanced data |
| 8 | Radius sanity | `max_dist >= min_dist` for every cluster |
| 9 | Size balance | `stddev / mean < 0.5` on uniform data |
| 10 | Outlier detection | Injected outliers are the max-dist points in their clusters |

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

## Streaming K-Means Details

Implements the Sculley (2010) mini-batch update rule:

```
for each point x assigned to centroid c:
    v[c] += 1
    c += (1 / v[c]) * (x - c)
```

- Default batch size: 50,000 vectors
- Per-centroid lifetime count `v[c]` provides natural learning-rate decay
- Centroids seeded via KMeans++ on the first batch
- Supports pluggable assignment backend (any `KMeansBase*`)

## Expected Performance Characteristics

| Backend | Training | Assignment | Memory |
|---------|----------|------------|--------|
| **BLAS** | Fastest at large N (SGEMM is cache-oblivious, heavily optimized) | Fast | High (N x K distance matrix) |
| **FAISS** | Competitive with BLAS | Fast (optimized internal routines) | High |
| **SIMD-AVX512** | Good | Fastest (widest registers, FMA, prefetch) | Moderate |
| **Streaming** | Moderate | Same as underlying backend | Lowest (one batch at a time) |

- Streaming produces 2-5% higher inertia than batch methods because it never sees the full dataset simultaneously.
- Batch methods produce tighter clusters; streaming may show higher radius ratios.
- Cluster size balance is slightly worse with streaming due to online update convergence.

## Thread Control

- Set thread count via `OMP_NUM_THREADS` or let it auto-detect
- For BLAS, set `OPENBLAS_NUM_THREADS=1` to avoid nested parallelism (our code manages OpenMP threading)
- Pin threads with `OMP_PROC_BIND=close` for stable benchmark results

## License

See the root repository license.
