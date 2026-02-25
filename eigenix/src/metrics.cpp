#include "metrics.hpp"
#include <algorithm>
#include <cmath>
#include <cstring>
#include <limits>
#include <numeric>
#include <vector>

namespace eigenix {

float compute_inertia(const float* data, size_t n, int dim,
                      const int* labels, const float* centroids, int k) {
    float total = 0.0f;
    #pragma omp parallel for reduction(+:total) schedule(static)
    for (size_t i = 0; i < n; ++i) {
        int l = labels[i];
        if (l < 0 || l >= k) continue;
        const float* x = data + i * dim;
        const float* c = centroids + static_cast<size_t>(l) * dim;
        float s = 0.0f;
        for (int j = 0; j < dim; ++j) {
            float d = x[j] - c[j];
            s += d * d;
        }
        total += s;
    }
    return total;
}

void compute_cluster_sizes(const int* labels, size_t n, int k,
                           size_t* out_sizes) {
    std::memset(out_sizes, 0, static_cast<size_t>(k) * sizeof(size_t));
    for (size_t i = 0; i < n; ++i) {
        int l = labels[i];
        if (l >= 0 && l < k)
            out_sizes[static_cast<size_t>(l)]++;
    }
}

float compute_imbalance_ratio(const size_t* sizes, int k) {
    if (k <= 0) return 0.0f;
    size_t mn = std::numeric_limits<size_t>::max();
    size_t mx = 0;
    for (int i = 0; i < k; ++i) {
        if (sizes[i] < mn) mn = sizes[i];
        if (sizes[i] > mx) mx = sizes[i];
    }
    if (mn == 0) return 0.0f;
    return static_cast<float>(mx) / static_cast<float>(mn);
}

std::vector<ClusterStats> compute_cluster_stats(
    const float* data, size_t n, int dim,
    const int* labels, const float* centroids, int k) {

    std::vector<ClusterStats> stats(static_cast<size_t>(k));
    for (int c = 0; c < k; ++c) {
        stats[c].min_dist = std::numeric_limits<float>::max();
        stats[c].max_dist = 0.0f;
        stats[c].mean_dist = 0.0f;
        stats[c].radius_ratio = 0.0f;
        stats[c].count = 0;
    }

    // Per-cluster accumulators (not parallelised for simplicity on stats path).
    for (size_t i = 0; i < n; ++i) {
        int l = labels[i];
        if (l < 0 || l >= k) continue;

        const float* x = data + i * dim;
        const float* c = centroids + static_cast<size_t>(l) * dim;
        float d2 = 0.0f;
        for (int j = 0; j < dim; ++j) {
            float d = x[j] - c[j];
            d2 += d * d;
        }
        float dist = std::sqrt(d2);

        auto& s = stats[static_cast<size_t>(l)];
        if (dist < s.min_dist) s.min_dist = dist;
        if (dist > s.max_dist) s.max_dist = dist;
        s.mean_dist += dist;
        s.count++;
    }

    for (int c = 0; c < k; ++c) {
        auto& s = stats[static_cast<size_t>(c)];
        if (s.count > 0) {
            s.mean_dist /= static_cast<float>(s.count);
            s.radius_ratio = (s.mean_dist > 0.0f)
                ? s.max_dist / s.mean_dist : 0.0f;
        } else {
            s.min_dist = 0.0f;
        }
    }

    return stats;
}

float compute_cluster_size_stddev(const size_t* sizes, int k) {
    if (k <= 0) return 0.0f;
    double sum = 0.0;
    for (int i = 0; i < k; ++i) sum += static_cast<double>(sizes[i]);
    double mean = sum / k;
    double var = 0.0;
    for (int i = 0; i < k; ++i) {
        double d = static_cast<double>(sizes[i]) - mean;
        var += d * d;
    }
    return static_cast<float>(std::sqrt(var / k));
}

int count_empty_clusters(const size_t* sizes, int k) {
    int cnt = 0;
    for (int i = 0; i < k; ++i)
        if (sizes[i] == 0) ++cnt;
    return cnt;
}

float compute_purity(const int* pred_labels, const int* true_labels,
                     size_t n, int k_pred, int k_true) {
    // For each predicted cluster, find the majority true label.
    // Purity = sum(max-matches) / n
    std::vector<std::vector<int>> counts(
        static_cast<size_t>(k_pred),
        std::vector<int>(static_cast<size_t>(k_true), 0));

    for (size_t i = 0; i < n; ++i) {
        int p = pred_labels[i];
        int t = true_labels[i];
        if (p >= 0 && p < k_pred && t >= 0 && t < k_true)
            counts[p][t]++;
    }

    size_t correct = 0;
    for (int c = 0; c < k_pred; ++c) {
        int mx = *std::max_element(counts[c].begin(), counts[c].end());
        correct += static_cast<size_t>(mx);
    }
    return static_cast<float>(correct) / static_cast<float>(n);
}

}  // namespace eigenix
