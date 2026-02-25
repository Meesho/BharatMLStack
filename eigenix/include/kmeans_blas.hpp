#ifndef EIGENIX_KMEANS_BLAS_HPP
#define EIGENIX_KMEANS_BLAS_HPP

#include "ikmeans.hpp"
#include <cstddef>
#include <vector>

namespace eigenix {

/**
 * BLAS-backed KMeans: cblas_sgemm for distances, KMeans++ init,
 * OpenMP for assignment and centroid update, early stopping.
 * No per-iteration allocation; row-major contiguous data.
 */
class BlasKMeans : public IKMeans {
public:
    explicit BlasKMeans(size_t k,
                       size_t max_iter = 300,
                       float tol = 1e-4f);

    void train(const float* data, size_t n, size_t dim) override;
    void assign(const float* data, size_t n, int* labels) const override;
    const float* centroids() const override;
    size_t k() const override;

private:
    size_t k_;
    size_t dim_{0};
    size_t max_iter_;
    float tol_;

    std::vector<float> centroids_;
    std::vector<float> data_norms_;      // ||x||^2 for training data
    std::vector<float> centroid_norms_;   // ||c||^2
    mutable std::vector<float> dist_buffer_;  // N x K for assignment
    std::vector<float> centroid_sums_;    // K x D for update
    std::vector<size_t> centroid_counts_;  // K
};

}  // namespace eigenix

#endif  // EIGENIX_KMEANS_BLAS_HPP
