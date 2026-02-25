#include "kmeans_faiss.hpp"
#include <faiss/Clustering.h>
#include <faiss/IndexFlat.h>
#include <cstring>
#include <omp.h>
#include <stdexcept>

namespace eigenix {

namespace {

// Squared L2 distance from one vector to all centroids; return index of nearest.
inline int assign_one(const float* x, size_t dim,
                     const float* centroids, size_t k) {
    int best = 0;
    float best_d2 = 0.0f;
    for (size_t j = 0; j < dim; ++j) {
        float d = x[j] - centroids[j];
        best_d2 += d * d;
    }
    for (size_t c = 1; c < k; ++c) {
        const float* cen = centroids + c * dim;
        float d2 = 0.0f;
        for (size_t j = 0; j < dim; ++j) {
            float d = x[j] - cen[j];
            d2 += d * d;
        }
        if (d2 < best_d2) {
            best_d2 = d2;
            best = static_cast<int>(c);
        }
    }
    return best;
}

}  // namespace

FaissKMeans::FaissKMeans(size_t k, size_t max_iter)
    : k_(k), max_iter_(max_iter) {
    if (k == 0)
        throw std::invalid_argument("FaissKMeans: k must be > 0");
}

void FaissKMeans::train(const float* data, size_t n, size_t dim) {
    if (data == nullptr || n == 0 || dim == 0)
        throw std::invalid_argument("FaissKMeans::train: invalid data or dimensions");
    if (n < k_)
        throw std::invalid_argument("FaissKMeans::train: n must be >= k");

    dim_ = dim;
    centroids_.resize(k_ * dim_);

    faiss::ClusteringParameters cp;
    cp.niter = static_cast<int>(max_iter_);
    cp.verbose = false;
    cp.spherical = false;
    cp.frozen_centroids = false;
    cp.update_index = false;
    cp.nredo = 1;

    faiss::Clustering clus(static_cast<int>(dim_), static_cast<int>(k_), cp);

    faiss::IndexFlatL2 index(static_cast<int>(dim_));
    clus.train(static_cast<faiss::idx_t>(n), data, index);

    const float* faiss_c = clus.centroids.data();
    std::memcpy(centroids_.data(), faiss_c, k_ * dim_ * sizeof(float));
}

void FaissKMeans::assign(const float* data, size_t n, int* labels) const {
    if (data == nullptr || labels == nullptr)
        throw std::invalid_argument("FaissKMeans::assign: null data or labels");
    if (dim_ == 0)
        throw std::runtime_error("FaissKMeans::assign: not trained");

    #pragma omp parallel for schedule(static)
    for (size_t i = 0; i < n; ++i)
        labels[i] = assign_one(data + i * dim_, dim_, centroids_.data(), k_);
}

const float* FaissKMeans::centroids() const {
    return centroids_.data();
}

size_t FaissKMeans::k() const {
    return k_;
}

}  // namespace eigenix
