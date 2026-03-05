#include "kmeans_blas.hpp"
#include "metrics.hpp"
#include <cblas.h>
#include <chrono>
#include <cmath>
#include <cstdio>
#include <cstring>
#include <omp.h>
#include <algorithm>
#include <limits>
#include <numeric>
#include <random>
#include <stdexcept>

// Older OpenBLAS headers may not define CBLAS_INT (added in newer versions).
#ifndef CBLAS_INT
using cblas_int = int;
#else
using cblas_int = CBLAS_INT;
#endif

namespace eigenix {

namespace {

inline float sqnorm(const float* x, int dim) {
    float s = 0.0f;
    for (int j = 0; j < dim; ++j) s += x[j] * x[j];
    return s;
}

}  // namespace

void BlasKMeans::random_init(const float* data, size_t n, int dim, int k,
                             std::mt19937& rng) {
    centroids_.resize(static_cast<size_t>(k) * dim);
    std::vector<size_t> indices(n);
    std::iota(indices.begin(), indices.end(), size_t(0));
    std::shuffle(indices.begin(), indices.end(), rng);
    for (int c = 0; c < k; ++c)
        std::memcpy(centroids_.data() + static_cast<size_t>(c) * dim,
                     data + indices[c] * dim,
                     static_cast<size_t>(dim) * sizeof(float));
}

void BlasKMeans::kmeanspp_init(const float* data, size_t n, int dim, int k,
                               std::mt19937& rng) {
    centroids_.resize(static_cast<size_t>(k) * dim);

    std::uniform_int_distribution<size_t> uidx(0, n - 1);
    size_t first = uidx(rng);
    std::memcpy(centroids_.data(), data + first * dim,
                static_cast<size_t>(dim) * sizeof(float));

    std::vector<float> min_dist(n);
    #pragma omp parallel for schedule(static)
    for (size_t i = 0; i < n; ++i) {
        const float* x = data + i * dim;
        float d = 0.0f;
        for (int j = 0; j < dim; ++j) {
            float t = x[j] - centroids_[j];
            d += t * t;
        }
        min_dist[i] = d;
    }

    for (int cc = 1; cc < k; ++cc) {
        double total = 0.0;
        #pragma omp parallel for schedule(static) reduction(+:total)
        for (size_t i = 0; i < n; ++i) total += min_dist[i];
        if (total <= 0.0) total = 1.0;

        std::uniform_real_distribution<double> u(0.0, total);
        double r = u(rng);
        size_t chosen = 0;
        for (; chosen < n && r >= 0.0; ++chosen) r -= min_dist[chosen];
        if (chosen > 0) chosen--;
        chosen = std::min(chosen, n - 1);

        std::memcpy(centroids_.data() + static_cast<size_t>(cc) * dim,
                     data + chosen * dim,
                     static_cast<size_t>(dim) * sizeof(float));

        const float* c_new = centroids_.data() + static_cast<size_t>(cc) * dim;
        #pragma omp parallel for schedule(static)
        for (size_t i = 0; i < n; ++i) {
            const float* x = data + i * dim;
            float d = 0.0f;
            for (int j = 0; j < dim; ++j) {
                float t = x[j] - c_new[j];
                d += t * t;
            }
            if (d < min_dist[i]) min_dist[i] = d;
        }
    }
}

void BlasKMeans::compute_centroid_norms() {
    centroid_norms_.resize(static_cast<size_t>(k_));
    for (int c = 0; c < k_; ++c)
        centroid_norms_[c] = sqnorm(centroids_.data() + static_cast<size_t>(c) * dim_, dim_);
}

void BlasKMeans::train_once(const float* data, size_t n, int dim, int k,
                            const TrainConfig& cfg) {
    k_ = k;
    dim_ = dim;
    iterations_ = 0;
    inertia_ = 0.0f;

    std::mt19937 rng(cfg.seed);
    random_init(data, n, dim, k, rng);
    compute_centroid_norms();

    std::vector<float> data_norms(n);
    #pragma omp parallel for schedule(static)
    for (size_t i = 0; i < n; ++i)
        data_norms[i] = sqnorm(data + i * dim, dim);

    size_t bs = std::min(BATCH_SIZE, n);
    dist_buf_.resize(bs * static_cast<size_t>(k));
    std::vector<float> centroid_sums(static_cast<size_t>(k) * dim);
    std::vector<size_t> centroid_counts(static_cast<size_t>(k));
    std::vector<int> labels(n);

    auto train_start = std::chrono::steady_clock::now();

    for (size_t iter = 0; iter < cfg.max_iter; ++iter) {
        assign_batch(data, data_norms.data(), n, labels.data());

        std::memset(centroid_sums.data(), 0,
                    static_cast<size_t>(k) * dim * sizeof(float));
        std::memset(centroid_counts.data(), 0,
                    static_cast<size_t>(k) * sizeof(size_t));

        int nthreads = 1;
        #pragma omp parallel
        { nthreads = omp_get_num_threads(); }

        std::vector<float> all_sums(static_cast<size_t>(nthreads) * k * dim, 0.0f);
        std::vector<size_t> all_counts(static_cast<size_t>(nthreads) * k, 0);

        #pragma omp parallel
        {
            int tid = omp_get_thread_num();
            float* lsums = all_sums.data() + static_cast<size_t>(tid) * k * dim;
            size_t* lcounts = all_counts.data() + static_cast<size_t>(tid) * k;
            #pragma omp for schedule(static)
            for (size_t i = 0; i < n; ++i) {
                int l = labels[i];
                const float* x = data + i * dim;
                float* s = lsums + static_cast<size_t>(l) * dim;
                for (int j = 0; j < dim; ++j) s[j] += x[j];
                lcounts[l]++;
            }
        }

        // Reduce thread-local buffers in parallel over clusters.
        #pragma omp parallel for schedule(static)
        for (int c = 0; c < k; ++c) {
            for (int t = 0; t < nthreads; ++t) {
                centroid_counts[c] += all_counts[static_cast<size_t>(t) * k + c];
                const float* src = all_sums.data() + (static_cast<size_t>(t) * k + c) * dim;
                float* dst = centroid_sums.data() + static_cast<size_t>(c) * dim;
                for (int j = 0; j < dim; ++j) dst[j] += src[j];
            }
        }

        float max_shift = 0.0f;
        for (int c = 0; c < k; ++c) {
            size_t cnt = centroid_counts[c];
            float* cv = centroids_.data() + static_cast<size_t>(c) * dim;
            if (cnt == 0) continue;
            float inv = 1.0f / static_cast<float>(cnt);
            float shift = 0.0f;
            for (int j = 0; j < dim; ++j) {
                float old_c = cv[j];
                float new_c = centroid_sums[static_cast<size_t>(c) * dim + j] * inv;
                cv[j] = new_c;
                float d = new_c - old_c;
                shift += d * d;
            }
            shift = std::sqrt(shift);
            if (shift > max_shift) max_shift = shift;
        }

        iterations_ = static_cast<int>(iter + 1);

        constexpr float SPLIT_EPS = 1.0f / 1024.0f;

        // Phase 1: fix empty clusters by splitting the largest.
        for (int c = 0; c < k; ++c) {
            if (centroid_counts[c] != 0) continue;
            int donor = 0;
            for (int j = 1; j < k; ++j)
                if (centroid_counts[j] > centroid_counts[donor]) donor = j;
            if (centroid_counts[donor] <= 1) continue;

            float* dst = centroids_.data() + static_cast<size_t>(c) * dim;
            float* src = centroids_.data() + static_cast<size_t>(donor) * dim;
            std::memcpy(dst, src, static_cast<size_t>(dim) * sizeof(float));
            for (int j = 0; j < dim; ++j) {
                float sign = (j % 2 == 0) ? 1.0f : -1.0f;
                dst[j] *= (1.0f + sign * SPLIT_EPS);
                src[j] *= (1.0f - sign * SPLIT_EPS);
            }
            centroid_counts[c] = centroid_counts[donor] / 2;
            centroid_counts[donor] -= centroid_counts[c];
        }

        // Phase 2: split oversized clusters into the smallest ones.
        size_t mean_sz = n / static_cast<size_t>(k);
        constexpr size_t IMBALANCE_FACTOR = 3;
        size_t threshold = mean_sz * IMBALANCE_FACTOR;
        for (int pass = 0; pass < k; ++pass) {
            int largest = 0, smallest = 0;
            for (int c = 1; c < k; ++c) {
                if (centroid_counts[c] > centroid_counts[largest]) largest = c;
                if (centroid_counts[c] < centroid_counts[smallest]) smallest = c;
            }
            if (centroid_counts[largest] <= threshold) break;
            if (largest == smallest) break;

            float* dst = centroids_.data() + static_cast<size_t>(smallest) * dim;
            float* src = centroids_.data() + static_cast<size_t>(largest) * dim;
            std::memcpy(dst, src, static_cast<size_t>(dim) * sizeof(float));
            for (int j = 0; j < dim; ++j) {
                float sign = (j % 2 == 0) ? 1.0f : -1.0f;
                dst[j] *= (1.0f + sign * SPLIT_EPS);
                src[j] *= (1.0f - sign * SPLIT_EPS);
            }
            centroid_counts[smallest] = centroid_counts[largest] / 2;
            centroid_counts[largest] -= centroid_counts[smallest];
        }

        compute_centroid_norms();

        if (cfg.verbose && (iter == 0 || (iter + 1) % 5 == 0
                            || max_shift <= cfg.tol || iter + 1 == cfg.max_iter)) {
            auto now = std::chrono::steady_clock::now();
            double elapsed = std::chrono::duration<double, std::milli>(now - train_start).count();
            std::fprintf(stderr, "[BLAS] iter %zu/%zu  max_shift=%.4f  elapsed=%.0fms\n",
                         iter + 1, cfg.max_iter, max_shift, elapsed);
        }

        if (max_shift <= cfg.tol) break;
    }

    inertia_ = compute_inertia(data, n, dim, labels.data(), centroids_.data(), k);
}

void BlasKMeans::train(const float* data, size_t n, int dim, int k,
                       const TrainConfig& cfg) {
    if (!data || n == 0 || dim <= 0 || k <= 0)
        throw std::invalid_argument("BlasKMeans::train: invalid arguments");
    if (n < static_cast<size_t>(k))
        throw std::invalid_argument("BlasKMeans::train: n must be >= k");

    int nredo = std::max(1, cfg.nredo);
    float best_inertia = std::numeric_limits<float>::max();
    std::vector<float> best_centroids;
    int best_iters = 0;

    for (int r = 0; r < nredo; ++r) {
        TrainConfig run_cfg = cfg;
        run_cfg.seed = cfg.seed + static_cast<unsigned>(r);
        run_cfg.nredo = 1;
        if (cfg.verbose && nredo > 1)
            std::fprintf(stderr, "[BLAS] nredo %d/%d (seed=%u)\n", r + 1, nredo, run_cfg.seed);
        train_once(data, n, dim, k, run_cfg);
        if (inertia_ < best_inertia) {
            best_inertia = inertia_;
            best_centroids = centroids_;
            best_iters = iterations_;
        }
    }

    centroids_ = std::move(best_centroids);
    inertia_ = best_inertia;
    iterations_ = best_iters;
}

void BlasKMeans::assign_batch(const float* data, const float* data_norms,
                              size_t n, int* labels) const {
    size_t bs = std::min(BATCH_SIZE, n);
    dist_buf_.resize(bs * static_cast<size_t>(k_));

    for (size_t off = 0; off < n; off += bs) {
        size_t cur = std::min(bs, n - off);
        cblas_sgemm(CblasRowMajor, CblasNoTrans, CblasTrans,
                    static_cast<cblas_int>(cur),
                    static_cast<cblas_int>(k_),
                    static_cast<cblas_int>(dim_),
                    -2.0f,
                    data + off * dim_, static_cast<cblas_int>(dim_),
                    centroids_.data(), static_cast<cblas_int>(dim_),
                    0.0f,
                    dist_buf_.data(), static_cast<cblas_int>(k_));

        #pragma omp parallel for schedule(static)
        for (size_t i = 0; i < cur; ++i) {
            float* row = dist_buf_.data() + i * k_;
            float dn = data_norms[off + i];
            int best = 0;
            float best_val = row[0] + dn + centroid_norms_[0];
            for (int c = 0; c < k_; ++c) {
                float v = row[c] + dn + centroid_norms_[c];
                if (v < best_val) { best_val = v; best = c; }
            }
            labels[off + i] = best;
        }
    }
}

void BlasKMeans::assign(const float* data, size_t n, int dim,
                        std::vector<int>& labels) const {
    if (!data || dim != dim_ || k_ == 0)
        throw std::runtime_error("BlasKMeans::assign: not trained or dim mismatch");

    labels.resize(n);
    if (n == 0) return;

    std::vector<float> qnorms(n);
    #pragma omp parallel for schedule(static)
    for (size_t i = 0; i < n; ++i)
        qnorms[i] = sqnorm(data + i * dim, dim);

    assign_batch(data, qnorms.data(), n, labels.data());
}

const float* BlasKMeans::centroids() const { return centroids_.data(); }
float BlasKMeans::inertia() const { return inertia_; }
int BlasKMeans::iterations() const { return iterations_; }
std::string BlasKMeans::name() const { return "BLAS"; }

std::vector<ClusterStats> BlasKMeans::cluster_stats(
    const float* data, size_t n, int dim) const {
    std::vector<int> labels;
    assign(data, n, dim, labels);
    return compute_cluster_stats(data, n, dim, labels.data(),
                                 centroids_.data(), k_);
}

}  // namespace eigenix
