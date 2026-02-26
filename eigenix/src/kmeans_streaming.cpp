#include "kmeans_streaming.hpp"
#include "metrics.hpp"
#include <algorithm>
#include <chrono>
#include <cmath>
#include <cstdio>
#include <cstring>
#include <random>
#include <stdexcept>

namespace eigenix {

StreamingKMeans::StreamingKMeans(KMeansBase* assign_backend, size_t batch_size)
    : backend_(assign_backend), batch_size_(batch_size) {}

void StreamingKMeans::set_backend(KMeansBase* backend) { backend_ = backend; }
void StreamingKMeans::set_batch_size(size_t bs) { batch_size_ = bs; }

// KMeans++ on the first batch to seed centroids.
void StreamingKMeans::init_centroids_from_batch(
    const float* batch, size_t batch_n, int dim, int k, unsigned seed) {

    centroids_.resize(static_cast<size_t>(k) * dim);
    centroid_counts_.assign(static_cast<size_t>(k), 0);

    std::mt19937 rng(seed);
    std::uniform_int_distribution<size_t> uidx(0, batch_n - 1);
    size_t first = uidx(rng);
    std::memcpy(centroids_.data(), batch + first * dim,
                static_cast<size_t>(dim) * sizeof(float));

    std::vector<float> min_dist(batch_n);
    for (size_t i = 0; i < batch_n; ++i) {
        const float* x = batch + i * dim;
        float d = 0.0f;
        for (int j = 0; j < dim; ++j) {
            float t = x[j] - centroids_[j];
            d += t * t;
        }
        min_dist[i] = d;
    }

    for (int cc = 1; cc < k; ++cc) {
        float total = 0.0f;
        for (size_t i = 0; i < batch_n; ++i) total += min_dist[i];
        if (total <= 0.0f) total = 1.0f;

        std::uniform_real_distribution<float> u(0.0f, total);
        float r = u(rng);
        size_t chosen = 0;
        for (; chosen < batch_n && r >= 0.0f; ++chosen) r -= min_dist[chosen];
        if (chosen > 0) chosen--;
        chosen = std::min(chosen, batch_n - 1);

        std::memcpy(centroids_.data() + static_cast<size_t>(cc) * dim,
                     batch + chosen * dim,
                     static_cast<size_t>(dim) * sizeof(float));

        const float* c_new = centroids_.data() + static_cast<size_t>(cc) * dim;
        for (size_t i = 0; i < batch_n; ++i) {
            const float* x = batch + i * dim;
            float d = 0.0f;
            for (int j = 0; j < dim; ++j) {
                float t = x[j] - c_new[j];
                d += t * t;
            }
            if (d < min_dist[i]) min_dist[i] = d;
        }
    }
}

void StreamingKMeans::train(const float* data, size_t n, int dim, int k,
                            const TrainConfig& cfg) {
    if (!data || n == 0 || dim <= 0 || k <= 0)
        throw std::invalid_argument("StreamingKMeans::train: invalid arguments");
    if (n < static_cast<size_t>(k))
        throw std::invalid_argument("StreamingKMeans::train: n must be >= k");

    k_ = k;
    dim_ = dim;
    iterations_ = 0;
    inertia_ = 0.0f;

    size_t bs = std::min(batch_size_, n);

    // Seed centroids from first batch using KMeans++.
    init_centroids_from_batch(data, bs, dim, k, cfg.seed);

    // We need a temporary "inner backend" to run assign() on each batch.
    // If user supplied a backend, copy our centroids into it;
    // otherwise we do a brute-force assign inline.

    std::vector<int> batch_labels;

    size_t total_batches = (n + bs - 1) / bs;
    auto train_start = std::chrono::steady_clock::now();

    for (size_t iter = 0; iter < cfg.max_iter; ++iter) {
        float max_shift = 0.0f;
        size_t batch_idx = 0;

        for (size_t start = 0; start < n; start += bs) {
            ++batch_idx;
            size_t batch_n = std::min(bs, n - start);
            const float* batch = data + start * dim;

            // Assign batch points to nearest centroid.
            batch_labels.resize(batch_n);
            for (size_t i = 0; i < batch_n; ++i) {
                const float* x = batch + i * dim;
                int best = 0;
                float best_d = 0.0f;
                for (int j = 0; j < dim; ++j) {
                    float d = x[j] - centroids_[j];
                    best_d += d * d;
                }
                for (int c = 1; c < k; ++c) {
                    const float* cv = centroids_.data() + static_cast<size_t>(c) * dim;
                    float d2 = 0.0f;
                    for (int j = 0; j < dim; ++j) {
                        float d = x[j] - cv[j];
                        d2 += d * d;
                    }
                    if (d2 < best_d) { best_d = d2; best = c; }
                }
                batch_labels[i] = best;
            }

            // Sculley (2010) online update:
            //   v[c] += 1
            //   c += (1/v[c]) * (x - c)
            for (size_t i = 0; i < batch_n; ++i) {
                int l = batch_labels[i];
                centroid_counts_[l]++;
                float eta = 1.0f / static_cast<float>(centroid_counts_[l]);
                float* cv = centroids_.data() + static_cast<size_t>(l) * dim;
                const float* x = batch + i * dim;
                float shift = 0.0f;
                for (int j = 0; j < dim; ++j) {
                    float delta = x[j] - cv[j];
                    cv[j] += eta * delta;
                    shift += delta * delta * eta * eta;
                }
                if (shift > max_shift) max_shift = shift;
            }
        }

        iterations_ = static_cast<int>(iter + 1);

        if (cfg.verbose && (iter == 0 || (iter + 1) % 5 == 0
                            || std::sqrt(max_shift) <= cfg.tol || iter + 1 == cfg.max_iter)) {
            auto now = std::chrono::steady_clock::now();
            double elapsed = std::chrono::duration<double, std::milli>(now - train_start).count();
            std::fprintf(stderr, "[Streaming] epoch %zu/%zu  batches=%zu  max_shift=%.4f  elapsed=%.0fms\n",
                         iter + 1, cfg.max_iter, total_batches, std::sqrt(max_shift), elapsed);
        }

        if (std::sqrt(max_shift) <= cfg.tol) break;
    }

    // Compute final inertia over full dataset.
    std::vector<int> labels;
    assign(data, n, dim, labels);
    inertia_ = compute_inertia(data, n, dim, labels.data(),
                               centroids_.data(), k);
}

void StreamingKMeans::assign(const float* data, size_t n, int dim,
                             std::vector<int>& labels) const {
    if (!data || dim != dim_ || k_ == 0)
        throw std::runtime_error("StreamingKMeans::assign: not trained or dim mismatch");

    labels.resize(n);
    if (n == 0) return;

    // If a backend is set and trained with same k/dim, delegate.
    if (backend_ && backend_->k() == k_ && backend_->dim() == dim_) {
        // Copy our centroids into a temporary to leverage the backend's
        // optimised assign().  This is a bit hacky but avoids coupling.
        // For simplicity, just do brute-force here — the streaming train
        // is the expensive path, not the final assign.
    }

    #pragma omp parallel for schedule(static)
    for (size_t i = 0; i < n; ++i) {
        const float* x = data + i * dim;
        int best = 0;
        float best_d = 0.0f;
        for (int j = 0; j < dim; ++j) {
            float d = x[j] - centroids_[j];
            best_d += d * d;
        }
        for (int c = 1; c < k_; ++c) {
            const float* cv = centroids_.data() + static_cast<size_t>(c) * dim;
            float d2 = 0.0f;
            for (int j = 0; j < dim; ++j) {
                float d = x[j] - cv[j];
                d2 += d * d;
            }
            if (d2 < best_d) { best_d = d2; best = c; }
        }
        labels[i] = best;
    }
}

const float* StreamingKMeans::centroids() const { return centroids_.data(); }
float StreamingKMeans::inertia() const { return inertia_; }
int StreamingKMeans::iterations() const { return iterations_; }

std::string StreamingKMeans::name() const {
    std::string suffix = backend_ ? backend_->name() : "Brute";
    return "Streaming-" + suffix;
}

std::vector<ClusterStats> StreamingKMeans::cluster_stats(
    const float* data, size_t n, int dim) const {
    std::vector<int> labels;
    assign(data, n, dim, labels);
    return compute_cluster_stats(data, n, dim, labels.data(),
                                 centroids_.data(), k_);
}

}  // namespace eigenix
