#ifndef EIGENIX_KMEANS_BLAS_HPP
#define EIGENIX_KMEANS_BLAS_HPP

#include "kmeans_base.hpp"
#include <random>
#include <vector>

namespace eigenix {

// BLAS-backed K-Means: cblas_sgemm for the distance cross-term,
// KMeans++ initialisation, OpenMP-parallel assignment and centroid update.
class BlasKMeans : public KMeansBase {
public:
    BlasKMeans() = default;

    void train(const float* data, size_t n, int dim, int k,
               const TrainConfig& cfg = {}) override;
    void assign(const float* data, size_t n, int dim,
                std::vector<int>& labels) const override;
    const float* centroids() const override;
    float inertia() const override;
    int iterations() const override;
    std::vector<ClusterStats> cluster_stats(
        const float* data, size_t n, int dim) const override;
    std::string name() const override;

    static constexpr size_t BATCH_SIZE = 100000;

private:
    std::vector<float> centroid_norms_;
    mutable std::vector<float> dist_buf_;

    void train_once(const float* data, size_t n, int dim, int k,
                    const TrainConfig& cfg);
    void random_init(const float* data, size_t n, int dim, int k,
                     std::mt19937& rng);
    void kmeanspp_init(const float* data, size_t n, int dim, int k,
                       std::mt19937& rng);
    void compute_centroid_norms();
    void assign_batch(const float* data, const float* data_norms,
                      size_t n, int* labels) const;
};

}  // namespace eigenix

#endif  // EIGENIX_KMEANS_BLAS_HPP
