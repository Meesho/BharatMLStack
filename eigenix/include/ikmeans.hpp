#ifndef EIGENIX_IKMEANS_HPP
#define EIGENIX_IKMEANS_HPP

#include <cstddef>
#include <cstdint>

namespace eigenix {

/**
 * Abstract interface for KMeans implementations.
 * Data is contiguous row-major float32: data[i * dim + j] = i-th vector, j-th component.
 */
class IKMeans {
public:
    virtual void train(const float* data, size_t n, size_t dim) = 0;

    virtual void assign(const float* data, size_t n, int* labels) const = 0;

    /** Centroids row-major, k() * dim elements. */
    virtual const float* centroids() const = 0;

    virtual size_t k() const = 0;

    virtual ~IKMeans() = default;
};

}  // namespace eigenix

#endif  // EIGENIX_IKMEANS_HPP
