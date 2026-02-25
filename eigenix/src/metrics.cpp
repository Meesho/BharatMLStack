#include "metrics.hpp"
#include <algorithm>
#include <cmath>
#include <cstring>
#include <limits>

namespace eigenix {

float compute_inertia(const float* data,
                     size_t n,
                     size_t dim,
                     const int* labels,
                     const float* centroids,
                     size_t k) {
    float total = 0.0f;
    for (size_t i = 0; i < n; ++i) {
        int label = labels[i];
        if (label < 0 || static_cast<size_t>(label) >= k)
            continue;
        const float* x = data + i * dim;
        const float* c = centroids + static_cast<size_t>(label) * dim;
        float sum = 0.0f;
        for (size_t j = 0; j < dim; ++j) {
            float d = x[j] - c[j];
            sum += d * d;
        }
        total += sum;
    }
    return total;
}

void compute_cluster_sizes(const int* labels, size_t n, size_t k, size_t* out_sizes) {
    std::memset(out_sizes, 0, k * sizeof(size_t));
    for (size_t i = 0; i < n; ++i) {
        int label = labels[i];
        if (label >= 0 && static_cast<size_t>(label) < k)
            out_sizes[static_cast<size_t>(label)]++;
    }
}

float compute_imbalance_ratio(const size_t* cluster_sizes, size_t k) {
    if (k == 0) return 0.0f;
    size_t min_sz = std::numeric_limits<size_t>::max();
    size_t max_sz = 0;
    for (size_t i = 0; i < k; ++i) {
        size_t s = cluster_sizes[i];
        if (s < min_sz) min_sz = s;
        if (s > max_sz) max_sz = s;
    }
    if (min_sz == 0) return 0.0f;
    return static_cast<float>(max_sz) / static_cast<float>(min_sz);
}

}  // namespace eigenix
