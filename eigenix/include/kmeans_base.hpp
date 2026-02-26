#ifndef EIGENIX_KMEANS_BASE_HPP
#define EIGENIX_KMEANS_BASE_HPP

#include <cstddef>
#include <cstdint>
#include <string>
#include <vector>

namespace eigenix {

struct ClusterStats {
    float min_dist;
    float max_dist;
    float mean_dist;
    float radius_ratio;   // max_dist / mean_dist — detect outlier-heavy clusters
    size_t count;
};

struct TrainConfig {
    size_t max_iter = 100;
    float tol = 1e-4f;
    unsigned seed = 42;
    bool verbose = false;
};

class KMeansBase {
public:
    virtual ~KMeansBase() = default;

    virtual void train(const float* data, size_t n, int dim, int k,
                       const TrainConfig& cfg = {}) = 0;

    virtual void assign(const float* data, size_t n, int dim,
                        std::vector<int>& labels) const = 0;

    virtual const float* centroids() const = 0;

    virtual float inertia() const = 0;

    virtual int iterations() const = 0;

    virtual std::vector<ClusterStats> cluster_stats(
        const float* data, size_t n, int dim) const = 0;

    virtual std::string name() const = 0;

    int k() const { return k_; }
    int dim() const { return dim_; }

protected:
    int k_ = 0;
    int dim_ = 0;
    float inertia_ = 0.0f;
    int iterations_ = 0;
    std::vector<float> centroids_;
};

}  // namespace eigenix

#endif  // EIGENIX_KMEANS_BASE_HPP
