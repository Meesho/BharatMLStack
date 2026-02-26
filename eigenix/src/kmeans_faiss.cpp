#include "kmeans_faiss.hpp"
#include "metrics.hpp"
#include <faiss/Clustering.h>
#include <faiss/IndexFlat.h>
#include <cstring>
#include <stdexcept>
#include <vector>

namespace eigenix {

void FaissKMeans::train(const float* data, size_t n, int dim, int k,
                        const TrainConfig& cfg) {
    if (!data || n == 0 || dim <= 0 || k <= 0)
        throw std::invalid_argument("FaissKMeans::train: invalid arguments");
    if (n < static_cast<size_t>(k))
        throw std::invalid_argument("FaissKMeans::train: n must be >= k");

    k_ = k;
    dim_ = dim;
    iterations_ = 0;

    faiss::ClusteringParameters cp;
    cp.niter = static_cast<int>(cfg.max_iter);
    cp.verbose = cfg.verbose;
    cp.seed = static_cast<int>(cfg.seed);
    cp.nredo = 1;

    faiss::Clustering clus(dim, k, cp);
    faiss::IndexFlatL2 index(dim);
    clus.train(static_cast<faiss::idx_t>(n), data, index);

    centroids_.resize(static_cast<size_t>(k) * dim);
    std::memcpy(centroids_.data(), clus.centroids.data(),
                centroids_.size() * sizeof(float));

    iterations_ = static_cast<int>(clus.iteration_stats.size());

    // Compute inertia on training data.
    std::vector<int> labels;
    assign(data, n, dim, labels);
    inertia_ = compute_inertia(data, n, dim, labels.data(),
                               centroids_.data(), k);
}

void FaissKMeans::assign(const float* data, size_t n, int dim,
                         std::vector<int>& labels) const {
    if (!data || dim != dim_ || k_ == 0)
        throw std::runtime_error("FaissKMeans::assign: not trained or dim mismatch");

    labels.resize(n);
    if (n == 0) return;

    faiss::IndexFlatL2 index(dim);
    index.add(static_cast<faiss::idx_t>(k_), centroids_.data());

    std::vector<float> distances(n);
    std::vector<faiss::idx_t> idx(n);
    index.search(static_cast<faiss::idx_t>(n), data, 1,
                 distances.data(), idx.data());

    for (size_t i = 0; i < n; ++i)
        labels[i] = static_cast<int>(idx[i]);
}

const float* FaissKMeans::centroids() const { return centroids_.data(); }
float FaissKMeans::inertia() const { return inertia_; }
int FaissKMeans::iterations() const { return iterations_; }
std::string FaissKMeans::name() const { return "FAISS"; }

std::vector<ClusterStats> FaissKMeans::cluster_stats(
    const float* data, size_t n, int dim) const {
    std::vector<int> labels;
    assign(data, n, dim, labels);
    return compute_cluster_stats(data, n, dim, labels.data(),
                                 centroids_.data(), k_);
}

}  // namespace eigenix
