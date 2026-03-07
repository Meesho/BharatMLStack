# Eigenix K-Means: Implementation Summary and Comparison with FAISS

This document describes what each algorithm does, what changes were made, why BLAS and SIMD can outperform FAISS on our benchmark, and a cross-check of the FAISS integration against the [official FAISS library](https://github.com/facebookresearch/faiss).

---

## 1. Algorithm Overview

| Backend | Initialization | Assignment | Centroid update | Empty/oversized handling |
|--------|----------------|------------|------------------|--------------------------|
| **FAISS** | Random (internal) | `IndexFlatL2` search | Lloyd inside `Clustering::train` | Empty-cluster split (internal) |
| **BLAS** | Random (shuffle K points) | Batched `cblas_sgemm` + argmin | Parallel sum/count, then mean | Empty split + oversized split (3× mean) |
| **SIMD** | Random (same as BLAS) | Per-point L2² via AVX-512/AVX2/SSE4.2/NEON | Same as BLAS (thread-local reduce) | Same as BLAS |

All three implement **Lloyd’s k-means**: alternate (1) assign each point to nearest centroid (L2²), (2) set each centroid to the mean of its assigned points. Convergence is driven by **centroid shift tolerance** (`max_shift <= tol`) and **max iterations**.

---

## 2. FAISS Backend

### What it does

- **Training:** Uses `faiss::Clustering` with `faiss::IndexFlatL2` as the assignment index. FAISS runs its own Lloyd loop (random init, assign via index, update centroids, handle empty clusters internally).
- **Assignment:** Builds an `IndexFlatL2`, adds the `k` centroids, then runs `index.search(n, data, 1, distances, indices)` to get the nearest centroid index per point.
- **Inertia:** Computed after training using our own `compute_inertia()` on the labels from our `assign()` for consistency with other backends.

### Parameters we set

- `cp.niter` = `cfg.max_iter` (e.g. 100)
- `cp.verbose` = `cfg.verbose`
- `cp.seed` = `cfg.seed`
- `cp.nredo` = `cfg.nredo` (same as BLAS/SIMD: multiple restarts when `EIGENIX_BENCH_NREDO` > 1, keep best inertia)
- `cp.max_points_per_centroid` = ceil(n/k) so that no subsampling is applied — we want to train on the full provided sample.

### Cross-check with FAISS library

- **Clustering API:** FAISS’s [building blocks](https://github.com/facebookresearch/faiss/wiki/Faiss-building-blocks:-clustering,-PCA,-quantization) and C++ API describe `Clustering(dim, k, cp)` and `clus.train(n, x, index)`. Our code uses exactly this: we construct `faiss::Clustering clus(dim, k, cp)` and call `clus.train(static_cast<faiss::idx_t>(n), data, index)` with an `IndexFlatL2 index(dim)`.
- **Index usage:** `IndexFlatL2` is the exact L2 distance index; `index.add(k, centroids)` adds the centroids; `index.search(n, data, 1, distances, idx)` returns the single nearest centroid index per vector. So assignment is correct L2 nearest-centroid.
- **Centroids:** We copy `clus.centroids` into our `centroids_` and use them for `assign()` and inertia. FAISS stores centroids in the same row-major `k × dim` layout we use.
- **Iteration count:** We set `iterations_` from `clus.iteration_stats.size()`, which matches the number of Lloyd iterations FAISS ran.

**Conclusion:** The FAISS backend is implemented correctly and uses the public FAISS clustering and flat L2 index APIs as intended. All three backends (FAISS, BLAS, SIMD) use the same `nredo` from `TrainConfig` for a fair comparison.

---

## 3. BLAS Backend

### What it does

- **Initialization:** **Random init** — shuffle indices and pick the first `k` data points as centroids. This matches FAISS’s default behavior and gives better cluster balance than KMeans++ on our benchmarks (KMeans++ is still available as `kmeanspp_init()`).
- **Assignment:** Uses the identity `||x - c||² = ||x||² - 2 x·c + ||c||²`. Precomputes `||x||²` and `||c||²`. The cross-term is a matrix multiply: for a batch of points `X` and centroids `C`, the (i,c) entry is `-2 * (X C^T)(i,c)`. Implemented with `cblas_sgemm` (RowMajor, NoTrans, Trans, -2.0, …). Then adds norms and does an argmin over `k` per point. Batched with `BATCH_SIZE = 100000` to control memory.
- **Centroid update:** Each thread accumulates point sums and counts per cluster in thread-local buffers; then a parallel reduction over clusters merges into global sums/counts. New centroid = sum / count.
- **Convergence:** Stops when `max_shift <= cfg.tol` (e.g. 0.01) or after `max_iter` iterations.
- **Empty clusters:** Phase 1 — any cluster with count 0: pick the largest cluster as donor, copy its centroid to the empty one, perturb both (SPLIT_EPS = 1/1024) so they separate, and split counts 50/50.
- **Oversized clusters:** Phase 2 — any cluster larger than `3 × mean_sz` is split: its centroid is copied to the smallest cluster, both perturbed, counts split. Repeated until no cluster is above the threshold.

### Changes made (and why)

1. **Random init** — Better balance (lower cluster-size stddev) and closer to FAISS; avoids KMeans++ favoring outliers and dense “mega-clusters.”
2. **Batched SGEMM** — Avoids allocating an n×k distance matrix; keeps memory and cache use under control for large n.
3. **Parallel centroid reduction** — Replaced a single `omp critical` with per-thread sums/counts and a parallel reduction over clusters for scalability.
4. **Empty + oversized handling** — Phase 1 fixes empties (FAISS-style); Phase 2 actively rebalances so no cluster stays >> mean size. This yields cluster-size distributions closer to FAISS or better (e.g. lower stddev).
5. **nredo** — Optional multiple restarts (different seeds), keeping the run with lowest inertia; improves quality when enabled.

---

## 4. SIMD Backend

### What it does

- **Initialization:** Same **random init** as BLAS (shuffle, first `k` points).
- **Assignment:** For each point, computes L2² to all `k` centroids using SIMD:
  - x86: AVX-512 (16 floats), AVX2+FMA (8 floats), or SSE4.2 (4 floats) depending on CPU; runtime detection via CPUID.
  - ARM: NEON (4 floats).
  - Scalar fallback when no SIMD.
- **Centroid update:** Identical to BLAS: thread-local sums/counts, then parallel reduction, then mean; same empty and oversized cluster handling (Phase 1 + Phase 2).
- **Convergence and nredo:** Same as BLAS (tol, max_iter, optional nredo).

### Changes made (and why)

- **Random init** — Same rationale as BLAS; aligns with FAISS and improves balance.
- **Empty + oversized handling** — Same two-phase logic as BLAS for consistent cluster quality.
- **Parallel reduction** — Same pattern as BLAS for centroid aggregation.
- **nredo** — Same multi-restart option as BLAS.

---

## 5. Why BLAS and SIMD Can Be “Better” Than FAISS Here

- **Speed:** FAISS’s clustering path builds and uses an index each iteration and has more internal overhead. Our BLAS path uses a single batched SGEMM plus argmin per batch; SIMD uses a tight per-point L2² loop. Both scale well with OpenMP. So for the same data and same (or higher) iteration count, BLAS/SIMD often finish training much faster (e.g. ~5× in some benchmarks).
- **Cluster balance:** We added explicit **oversized-cluster splitting** (Phase 2). FAISS only fixes empty clusters. So for our synthetic data we often get:
  - Similar or better **inertia** (same or lower).
  - **Tighter cluster-size distribution** (lower stddev, smaller max cluster) and **no empty clusters**.
- **Convergence:** We stop on centroid shift (`max_shift <= tol`). BLAS/SIMD can converge in fewer iterations (e.g. 72–82) while FAISS is run for a fixed 100; combined with faster per-iteration cost, wall time is much lower.
- **Fair comparison:** All three are trained on the same data (e.g. 30% sample), and we use the same brute-force assignment on the full set to compare centroid quality (inertia and purity vs ground truth). So “better” here means: same or better objective and balance, with much lower training time.

---

## 6. Brute-Force Comparison Benchmark

After each backend runs, we save its centroids and run a **single shared brute-force L2 assign** on the full dataset for all three. We then report:

- **Brute-force inertia** — Same assign algorithm for every method; lower is better.
- **Purity vs ground truth** — For data generated from 50 Gaussians with known labels; higher is better.
- **Pairwise agreement** — Fraction of points where two methods assign the same label (FAISS vs BLAS, FAISS vs SIMD, BLAS vs SIMD).

This removes any difference in assignment implementation and isolates centroid quality. Results are written to `results/bruteforce_comparison.csv`.

---

## 7. File-Level Summary of Changes

| Component | Change |
|----------|--------|
| **FAISS** | Correct use of `Clustering` + `IndexFlatL2`; `max_points_per_centroid` set so no subsampling; inertia from our `assign` + `compute_inertia`. |
| **BLAS** | Random init (default), batched SGEMM, parallel centroid reduction, Phase 1 (empty) + Phase 2 (oversized) splits, nredo, centroid-shift convergence. |
| **SIMD** | Random init (default), same two-phase split and reduction as BLAS, nredo, centroid-shift convergence; assignment is SIMD L2² only. |
| **Benchmark** | Same train data for all backends; brute-force comparison phase; optional `results/bruteforce_comparison.csv`. |
| **Metrics** | `brute_force_assign()` for shared L2 nearest-centroid assign. |
| **Streaming** | Removed from the suite; only FAISS, BLAS, and SIMD are compared. |

---

## 8. References

- [FAISS: A library for efficient similarity search and clustering](https://github.com/facebookresearch/faiss)
- [Faiss building blocks: clustering, PCA, quantization](https://github.com/facebookresearch/faiss/wiki/Faiss-building-blocks:-clustering,-PCA,-quantization)
