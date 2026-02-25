#ifndef EIGENIX_KMEANS_FAISS_HPP
#define EIGENIX_KMEANS_FAISS_HPP

#include "kmeans_base.hpp"
#include <vector>

namespace eigenix {

// FAISS-backed K-Means: faiss::Clustering for training,
// faiss::IndexFlatL2 for assignment.
class FaissKMeans : public KMeansBase {
public:
    FaissKMeans() = default;

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
};

}  // namespace eigenix

#endif  // EIGENIX_KMEANS_FAISS_HPP
