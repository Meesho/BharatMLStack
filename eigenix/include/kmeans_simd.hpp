#ifndef EIGENIX_KMEANS_SIMD_HPP
#define EIGENIX_KMEANS_SIMD_HPP

#include "kmeans_base.hpp"
#include <random>
#include <string>
#include <vector>

namespace eigenix {

enum class SimdISA { SSE42, AVX2, AVX512, NEON };

// Runtime-dispatched SIMD K-Means: AVX-512 → AVX2 → SSE4.2 fallback.
// FMA-fused L2 squared distance kernels with cache-line prefetch.
// OpenMP parallel outer loop for assignment and centroid update.
class SimdKMeans : public KMeansBase {
public:
    SimdKMeans();

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

    SimdISA detected_isa() const { return isa_; }

private:
    SimdISA isa_;

    void kmeanspp_init(const float* data, size_t n, int dim, int k,
                       std::mt19937& rng);

    // Distance from a single vector to a single centroid (squared L2).
    float l2sq(const float* a, const float* b, int dim) const;
};

}  // namespace eigenix

#endif  // EIGENIX_KMEANS_SIMD_HPP
