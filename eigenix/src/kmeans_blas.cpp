#include "kmeans_blas.hpp"
#include <cblas.h>
#include <cmath>
#include <cstring>
#include <omp.h>
#include <random>
#include <stdexcept>

namespace eigenix {

namespace {

inline float sqnorm(const float* x, size_t dim) {
    float sum = 0.0f;
    for (size_t j = 0; j < dim; ++j)
        sum += x[j] * x[j];
    return sum;
}

}  // namespace

BlasKMeans::BlasKMeans(size_t k, size_t max_iter, float tol)
    : k_(k), max_iter_(max_iter), tol_(tol) {
    if (k == 0)
        throw std::invalid_argument("BlasKMeans: k must be > 0");
}

void BlasKMeans::train(const float* data, size_t n, size_t dim) {
    if (data == nullptr || n == 0 || dim == 0)
        throw std::invalid_argument("BlasKMeans::train: invalid data or dimensions");
    if (n < k_)
        throw std::invalid_argument("BlasKMeans::train: n must be >= k");

    dim_ = dim;

    centroids_.resize(k_ * dim_);
    data_norms_.resize(n);
    centroid_norms_.resize(k_);
    dist_buffer_.resize(n * k_);
    centroid_sums_.resize(k_ * dim_);
    centroid_counts_.resize(k_);

    // Precompute data norms: ||x_i||^2
    #pragma omp parallel for schedule(static)
    for (size_t i = 0; i < n; ++i)
        data_norms_[i] = sqnorm(data + i * dim_, dim_);

    // KMeans++ init
    std::mt19937 rng(static_cast<unsigned>(std::random_device{}()));
    std::uniform_int_distribution<size_t> uidx(0, n - 1);
    size_t first = uidx(rng);
    std::memcpy(centroids_.data(), data + first * dim_, dim_ * sizeof(float));

    std::vector<float> min_dist(n);
    for (size_t i = 0; i < n; ++i) {
        const float* x = data + i * dim_;
        float d = 0.0f;
        for (size_t j = 0; j < dim_; ++j) {
            float t = x[j] - centroids_[j];
            d += t * t;
        }
        min_dist[i] = d;
    }

    for (size_t cc = 1; cc < k_; ++cc) {
        float total = 0.0f;
        for (size_t i = 0; i < n; ++i)
            total += min_dist[i];
        if (total <= 0.0f)
            total = 1.0f;
        std::uniform_real_distribution<float> u(0.0f, total);
        float r = u(rng);
        size_t chosen = 0;
        for (; chosen < n && r >= 0.0f; ++chosen)
            r -= min_dist[chosen];
        if (chosen > 0) chosen--;
        chosen = std::min(chosen, n - 1);

        std::memcpy(centroids_.data() + cc * dim_, data + chosen * dim_, dim_ * sizeof(float));

        const float* c_new = centroids_.data() + cc * dim_;
        for (size_t i = 0; i < n; ++i) {
            const float* x = data + i * dim_;
            float d = 0.0f;
            for (size_t j = 0; j < dim_; ++j) {
                float t = x[j] - c_new[j];
                d += t * t;
            }
            if (d < min_dist[i]) min_dist[i] = d;
        }
    }

    for (size_t i = 0; i < k_; ++i)
        centroid_norms_[i] = sqnorm(centroids_.data() + i * dim_, dim_);

    for (size_t iter = 0; iter < max_iter_; ++iter) {
        cblas_sgemm(CblasRowMajor, CblasNoTrans, CblasTrans,
                    static_cast<CBLAS_INT>(n),
                    static_cast<CBLAS_INT>(k_),
                    static_cast<CBLAS_INT>(dim_),
                    -2.0f,
                    data, static_cast<CBLAS_INT>(dim_),
                    centroids_.data(), static_cast<CBLAS_INT>(dim_),
                    0.0f,
                    dist_buffer_.data(), static_cast<CBLAS_INT>(k_));

        #pragma omp parallel for schedule(static)
        for (size_t i = 0; i < n; ++i) {
            float* row = dist_buffer_.data() + i * k_;
            float dn = data_norms_[i];
            for (size_t kk = 0; kk < k_; ++kk)
                row[kk] += dn + centroid_norms_[kk];
        }

        std::vector<int> labels_local(n);
        #pragma omp parallel for schedule(static)
        for (size_t i = 0; i < n; ++i) {
            const float* row = dist_buffer_.data() + i * k_;
            int best = 0;
            float best_val = row[0];
            for (size_t kk = 1; kk < k_; ++kk) {
                if (row[kk] < best_val) {
                    best_val = row[kk];
                    best = static_cast<int>(kk);
                }
            }
            labels_local[i] = best;
        }

        std::memset(centroid_sums_.data(), 0, k_ * dim_ * sizeof(float));
        std::memset(centroid_counts_.data(), 0, k_ * sizeof(size_t));

        #pragma omp parallel
        {
            std::vector<float> local_sums(k_ * dim_, 0.0f);
            std::vector<size_t> local_counts(k_, 0);
            #pragma omp for schedule(static)
            for (size_t i = 0; i < n; ++i) {
                int l = labels_local[i];
                const float* x = data + i * dim_;
                float* s = local_sums.data() + static_cast<size_t>(l) * dim_;
                for (size_t j = 0; j < dim_; ++j)
                    s[j] += x[j];
                local_counts[static_cast<size_t>(l)]++;
            }
            #pragma omp critical
            {
                for (size_t kk = 0; kk < k_; ++kk) {
                    centroid_counts_[kk] += local_counts[kk];
                    const float* src = local_sums.data() + kk * dim_;
                    float* dst = centroid_sums_.data() + kk * dim_;
                    for (size_t j = 0; j < dim_; ++j)
                        dst[j] += src[j];
                }
            }
        }

        float max_shift = 0.0f;
        for (size_t kk = 0; kk < k_; ++kk) {
            size_t cnt = centroid_counts_[kk];
            float* c = centroids_.data() + kk * dim_;
            if (cnt == 0) continue;
            float inv = 1.0f / static_cast<float>(cnt);
            float shift = 0.0f;
            for (size_t j = 0; j < dim_; ++j) {
                float old_c = c[j];
                float new_c = centroid_sums_[kk * dim_ + j] * inv;
                c[j] = new_c;
                float d = new_c - old_c;
                shift += d * d;
            }
            shift = std::sqrt(shift);
            if (shift > max_shift) max_shift = shift;
        }

        for (size_t i = 0; i < k_; ++i)
            centroid_norms_[i] = sqnorm(centroids_.data() + i * dim_, dim_);

        if (max_shift <= tol_)
            break;
    }
}

void BlasKMeans::assign(const float* data, size_t n, int* labels) const {
    if (data == nullptr || labels == nullptr)
        throw std::invalid_argument("BlasKMeans::assign: null data or labels");
    if (dim_ == 0)
        throw std::runtime_error("BlasKMeans::assign: not trained");

    if (n == 0) return;

    dist_buffer_.resize(n * k_);

    std::vector<float> query_norms(n);
    #pragma omp parallel for schedule(static)
    for (size_t i = 0; i < n; ++i)
        query_norms[i] = sqnorm(data + i * dim_, dim_);

    cblas_sgemm(CblasRowMajor, CblasNoTrans, CblasTrans,
                static_cast<CBLAS_INT>(n),
                static_cast<CBLAS_INT>(k_),
                static_cast<CBLAS_INT>(dim_),
                -2.0f,
                data, static_cast<CBLAS_INT>(dim_),
                centroids_.data(), static_cast<CBLAS_INT>(dim_),
                0.0f,
                dist_buffer_.data(), static_cast<CBLAS_INT>(k_));

    #pragma omp parallel for schedule(static)
    for (size_t i = 0; i < n; ++i) {
        float* row = dist_buffer_.data() + i * k_;
        float qn = query_norms[i];
        for (size_t kk = 0; kk < k_; ++kk)
            row[kk] += qn + centroid_norms_[kk];
        int best = 0;
        float best_val = row[0];
        for (size_t kk = 1; kk < k_; ++kk) {
            if (row[kk] < best_val) {
                best_val = row[kk];
                best = static_cast<int>(kk);
            }
        }
        labels[i] = best;
    }
}

const float* BlasKMeans::centroids() const {
    return centroids_.data();
}

size_t BlasKMeans::k() const {
    return k_;
}

}  // namespace eigenix
