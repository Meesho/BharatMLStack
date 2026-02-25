#ifndef EIGENIX_KMEANS_STREAMING_HPP
#define EIGENIX_KMEANS_STREAMING_HPP

#include "kmeans_base.hpp"
#include <memory>
#include <vector>

namespace eigenix {

// Mini-batch / Streaming K-Means (Sculley 2010).
// Uses a pluggable KMeansBase backend for the per-batch assignment step.
// Centroids are updated online with per-centroid learning-rate decay:
//   v[c] += 1;  c += (1/v[c]) * (x - c)
class StreamingKMeans : public KMeansBase {
public:
    // assign_backend: any KMeansBase* used for the nearest-centroid query
    //   within each mini-batch.  Ownership is NOT transferred.
    // batch_size: vectors per mini-batch (default 50 000).
    explicit StreamingKMeans(KMeansBase* assign_backend = nullptr,
                             size_t batch_size = 50000);

    void set_backend(KMeansBase* backend);
    void set_batch_size(size_t bs);
    size_t batch_size() const { return batch_size_; }

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

private:
    KMeansBase* backend_;
    size_t batch_size_;
    std::vector<size_t> centroid_counts_;  // lifetime per-centroid assignment count

    void init_centroids_from_batch(const float* batch, size_t batch_n,
                                   int dim, int k, unsigned seed);
};

}  // namespace eigenix

#endif  // EIGENIX_KMEANS_STREAMING_HPP
