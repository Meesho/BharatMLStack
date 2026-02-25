# Eigenix KMeans

Modular KMeans component for vector clustering, designed to integrate with an HNSW-based vector database.

## Layout

- **include/** — Public headers: `ikmeans.hpp`, `kmeans_blas.hpp`, `kmeans_faiss.hpp`, `metrics.hpp`
- **src/** — Implementations: `kmeans_blas.cpp`, `kmeans_faiss.cpp`, `metrics.cpp`
- **benchmark/** — `benchmark_kmeans.cpp` (synthetic data, BLAS vs FAISS comparison)

## Build

Requirements: BLAS (e.g. OpenBLAS), FAISS, OpenMP.

```bash
make
./benchmark/benchmark_kmeans [N_train] [D] [K]
# Default: N_train=100000, D=128, K=256
```

**macOS (Apple Clang):** Install OpenMP and use compatible flags, e.g.:

```bash
brew install libomp
# Then either use a g++ that supports -fopenmp, or set:
# CXXFLAGS="-O3 -std=c++17 -Xpreprocessor -fopenmp -I include" and link -lomp
```

## Thread control

- Set `omp_set_num_threads()` before training (e.g. to `std::thread::hardware_concurrency()`).
- Do not let BLAS use its own threading; use a single-threaded BLAS or set `OPENBLAS_NUM_THREADS=1` (or equivalent) so OpenMP in our code is the only parallelism.
- No nested parallelism in the library.

## Interface

- `eigenix::IKMeans` — abstract base: `train()`, `assign()`, `centroids()`, `k()`.
- `eigenix::BlasKMeans` — BLAS + OpenMP, KMeans++, early stopping.
- `eigenix::FaissKMeans` — FAISS Clustering + IndexFlatL2; `assign()` implemented for fair comparison.
- `eigenix::compute_inertia()`, `compute_cluster_sizes()`, `compute_imbalance_ratio()`.

All data is contiguous row-major float32.
