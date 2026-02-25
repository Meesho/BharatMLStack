#include "kmeans_blas.hpp"
#include "metrics.hpp"
#include <cblas.h>
#include <cmath>
#include <cstring>
#include <omp.h>
#include <random>
#include <stdexcept>

namespace eigenix {

namespace {

inline float sqnorm(const float* x, int dim) {
    float s = 0.0f;
    for (int j = 0; j < dim; ++j) s += x[j] * x[j];
    return s;
}

}  // namespace

void BlasKMeans::kmeanspp_init(const float* data, size_t n, int dim, int k,
                               std::mt19937& rng) {
    centroids_.resize(static_cast<size_t>(k) * dim);

    std::uniform_int_distribution<size_t> uidx(0, n - 1);
    size_t first = uidx(rng);
    std::memcpy(centroids_.data(), data + first * dim,
                static_cast<size_t>(dim) * sizeof(float));

    std::vector<float> min_dist(n);
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
        float total = 0.0f;
        for (size_t i = 0; i < n; ++i) total += min_dist[i];
        if (total <= 0.0f) total = 1.0f;

        std::uniform_real_distribution<float> u(0.0f, total);
        float r = u(rng);
        size_t chosen = 0;
        for (; chosen < n && r >= 0.0f; ++chosen) r -= min_dist[chosen];
        if (chosen > 0) chosen--;
        chosen = std::min(chosen, n - 1);

        std::memcpy(centroids_.data() + static_cast<size_t>(cc) * dim,
                     data + chosen * dim,
                     static_cast<size_t>(dim) * sizeof(float));

        const float* c_new = centroids_.data() + static_cast<size_t>(cc) * dim;
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

void BlasKMeans::train(const float* data, size_t n, int dim, int k,
                       const TrainConfig& cfg) {
    if (!data || n == 0 || dim <= 0 || k <= 0)
        throw std::invalid_argument("BlasKMeans::train: invalid arguments");
    if (n < static_cast<size_t>(k))
        throw std::invalid_argument("BlasKMeans::train: n must be >= k");

    k_ = k;
    dim_ = dim;
    iterations_ = 0;
    inertia_ = 0.0f;

    std::mt19937 rng(cfg.seed);
    kmeanspp_init(data, n, dim, k, rng);
    compute_centroid_norms();

    // Pre-compute data norms.
    std::vector<float> data_norms(n);
    #pragma omp parallel for schedule(static)
    for (size_t i = 0; i < n; ++i)
        data_norms[i] = sqnorm(data + i * dim, dim);

    dist_buf_.resize(n * static_cast<size_t>(k));
    std::vector<float> centroid_sums(static_cast<size_t>(k) * dim);
    std::vector<size_t> centroid_counts(static_cast<size_t>(k));
    std::vector<int> labels(n);

    for (size_t iter = 0; iter < cfg.max_iter; ++iter) {
        // dist_buf[i,c] = -2 * x_i . c_c  (via SGEMM)
        cblas_sgemm(CblasRowMajor, CblasNoTrans, CblasTrans,
                    static_cast<CBLAS_INT>(n),
                    static_cast<CBLAS_INT>(k),
                    static_cast<CBLAS_INT>(dim),
                    -2.0f,
                    data, static_cast<CBLAS_INT>(dim),
                    centroids_.data(), static_cast<CBLAS_INT>(dim),
                    0.0f,
                    dist_buf_.data(), static_cast<CBLAS_INT>(k));

        // Complete ||x-c||^2 = ||x||^2 - 2 x.c + ||c||^2  and find argmin.
        #pragma omp parallel for schedule(static)
        for (size_t i = 0; i < n; ++i) {
            float* row = dist_buf_.data() + i * k;
            float dn = data_norms[i];
            for (int c = 0; c < k; ++c)
                row[c] += dn + centroid_norms_[c];

            int best = 0;
            float best_val = row[0];
            for (int c = 1; c < k; ++c) {
                if (row[c] < best_val) { best_val = row[c]; best = c; }
            }
            labels[i] = best;
        }

        // Centroid update with thread-local accumulators.
        std::memset(centroid_sums.data(), 0,
                    static_cast<size_t>(k) * dim * sizeof(float));
        std::memset(centroid_counts.data(), 0,
                    static_cast<size_t>(k) * sizeof(size_t));

        #pragma omp parallel
        {
            std::vector<float> local_sums(static_cast<size_t>(k) * dim, 0.0f);
            std::vector<size_t> local_counts(static_cast<size_t>(k), 0);
            #pragma omp for schedule(static)
            for (size_t i = 0; i < n; ++i) {
                int l = labels[i];
                const float* x = data + i * dim;
                float* s = local_sums.data() + static_cast<size_t>(l) * dim;
                for (int j = 0; j < dim; ++j) s[j] += x[j];
                local_counts[l]++;
            }
            #pragma omp critical
            {
                for (int c = 0; c < k; ++c) {
                    centroid_counts[c] += local_counts[c];
                    const float* src = local_sums.data() + static_cast<size_t>(c) * dim;
                    float* dst = centroid_sums.data() + static_cast<size_t>(c) * dim;
                    for (int j = 0; j < dim; ++j) dst[j] += src[j];
                }
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

        compute_centroid_norms();
        iterations_ = static_cast<int>(iter + 1);

        if (max_shift <= cfg.tol) break;
    }

    inertia_ = compute_inertia(data, n, dim, labels.data(),
                               centroids_.data(), k);
}

void BlasKMeans::assign(const float* data, size_t n, int dim,
                        std::vector<int>& labels) const {
    if (!data || dim != dim_ || k_ == 0)
        throw std::runtime_error("BlasKMeans::assign: not trained or dim mismatch");

    labels.resize(n);
    if (n == 0) return;

    dist_buf_.resize(n * static_cast<size_t>(k_));

    std::vector<float> qnorms(n);
    #pragma omp parallel for schedule(static)
    for (size_t i = 0; i < n; ++i)
        qnorms[i] = sqnorm(data + i * dim, dim);

    cblas_sgemm(CblasRowMajor, CblasNoTrans, CblasTrans,
                static_cast<CBLAS_INT>(n),
                static_cast<CBLAS_INT>(k_),
                static_cast<CBLAS_INT>(dim),
                -2.0f,
                data, static_cast<CBLAS_INT>(dim),
                centroids_.data(), static_cast<CBLAS_INT>(dim),
                0.0f,
                dist_buf_.data(), static_cast<CBLAS_INT>(k_));

    #pragma omp parallel for schedule(static)
    for (size_t i = 0; i < n; ++i) {
        float* row = dist_buf_.data() + i * k_;
        float qn = qnorms[i];
        int best = 0;
        float best_val = row[0] + qn + centroid_norms_[0];
        for (int c = 0; c < k_; ++c) {
            float v = row[c] + qn + centroid_norms_[c];
            if (v < best_val) { best_val = v; best = c; }
        }
        labels[i] = best;
    }
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
