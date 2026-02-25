#ifndef EIGENIX_KMEANS_FAISS_HPP
#define EIGENIX_KMEANS_FAISS_HPP

#include "ikmeans.hpp"
#include <cstddef>
#include <vector>

namespace eigenix {

/**
 * FAISS-backed KMeans: faiss::Clustering + IndexFlatL2.
 * Centroids extracted after training; assign() implemented manually for fair comparison.
 */
class FaissKMeans : public IKMeans {
public:
    explicit FaissKMeans(size_t k, size_t max_iter = 300);

    void train(const float* data, size_t n, size_t dim) override;
    void assign(const float* data, size_t n, int* labels) const override;
    const float* centroids() const override;
    size_t k() const override;

private:
    size_t k_;
    size_t dim_{0};
    size_t max_iter_;

    std::vector<float> centroids_;
};

}  // namespace eigenix

#endif  // EIGENIX_KMEANS_FAISS_HPP
