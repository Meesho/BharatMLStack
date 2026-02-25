#ifndef EIGENIX_METRICS_HPP
#define EIGENIX_METRICS_HPP

#include "kmeans_base.hpp"
#include <cstddef>
#include <vector>

namespace eigenix {

float compute_inertia(const float* data, size_t n, int dim,
                      const int* labels, const float* centroids, int k);

void compute_cluster_sizes(const int* labels, size_t n, int k,
                           size_t* out_sizes);

float compute_imbalance_ratio(const size_t* cluster_sizes, int k);

std::vector<ClusterStats> compute_cluster_stats(
    const float* data, size_t n, int dim,
    const int* labels, const float* centroids, int k);

float compute_cluster_size_stddev(const size_t* sizes, int k);

int count_empty_clusters(const size_t* sizes, int k);

// Purity score: fraction of points whose cluster assignment matches the
// majority ground-truth label within that predicted cluster.
float compute_purity(const int* pred_labels, const int* true_labels,
                     size_t n, int k_pred, int k_true);

}  // namespace eigenix

#endif  // EIGENIX_METRICS_HPP
