#ifndef EIGENIX_METRICS_HPP
#define EIGENIX_METRICS_HPP

#include <cstddef>

namespace eigenix {

/**
 * Sum of squared distances from each point to its assigned centroid.
 * data: row-major [n * dim], labels: [n], centroids: row-major [k * dim].
 */
float compute_inertia(const float* data,
                     size_t n,
                     size_t dim,
                     const int* labels,
                     const float* centroids,
                     size_t k);

/**
 * Count of points per cluster. out_sizes must have at least k elements.
 */
void compute_cluster_sizes(const int* labels, size_t n, size_t k, size_t* out_sizes);

/**
 * Imbalance ratio: max_cluster_size / min_cluster_size (or 0 if min is 0).
 */
float compute_imbalance_ratio(const size_t* cluster_sizes, size_t k);

}  // namespace eigenix

#endif  // EIGENIX_METRICS_HPP
