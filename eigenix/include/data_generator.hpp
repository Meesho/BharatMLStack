#ifndef EIGENIX_DATA_GENERATOR_HPP
#define EIGENIX_DATA_GENERATOR_HPP

#include <cstddef>
#include <vector>

namespace eigenix {

// Mixture-of-Gaussians synthetic data for benchmarking.
// Generates n vectors of dimension dim from num_gaussians clusters.
// Each Gaussian centre is uniformly sampled in [-10, 10]^dim.
// Points are drawn with identity covariance scaled per cluster.
std::vector<float> generate_gaussian_mixture(
    size_t n, int dim, int num_gaussians, unsigned seed);

// Returns the ground-truth cluster label (0..num_gaussians-1) for each point
// using the same seed so assignments are reproducible.
std::vector<int> generate_ground_truth_labels(
    size_t n, int num_gaussians, unsigned seed);

}  // namespace eigenix

#endif  // EIGENIX_DATA_GENERATOR_HPP
