#include "data_generator.hpp"
#include <cmath>
#include <random>
#include <stdexcept>

namespace eigenix {

std::vector<float> generate_gaussian_mixture(
    size_t n, int dim, int num_gaussians, unsigned seed) {
    if (n == 0 || dim <= 0 || num_gaussians <= 0)
        throw std::invalid_argument("generate_gaussian_mixture: invalid parameters");

    std::mt19937 rng(seed);
    std::uniform_real_distribution<float> mean_dist(-10.0f, 10.0f);
    std::uniform_real_distribution<float> scale_dist(0.5f, 2.0f);

    // Generate cluster centres and per-cluster scales.
    std::vector<float> centres(static_cast<size_t>(num_gaussians) * dim);
    std::vector<float> scales(num_gaussians);
    for (int g = 0; g < num_gaussians; ++g) {
        scales[g] = scale_dist(rng);
        for (int d = 0; d < dim; ++d)
            centres[static_cast<size_t>(g) * dim + d] = mean_dist(rng);
    }

    // Sample points round-robin across clusters then add Gaussian noise.
    std::vector<float> data(n * dim);
    std::normal_distribution<float> noise(0.0f, 1.0f);

    for (size_t i = 0; i < n; ++i) {
        int g = static_cast<int>(i % static_cast<size_t>(num_gaussians));
        float s = scales[g];
        const float* c = centres.data() + static_cast<size_t>(g) * dim;
        float* row = data.data() + i * dim;
        for (int d = 0; d < dim; ++d)
            row[d] = c[d] + s * noise(rng);
    }

    return data;
}

std::vector<int> generate_ground_truth_labels(
    size_t n, int num_gaussians, unsigned /*seed*/) {
    std::vector<int> labels(n);
    for (size_t i = 0; i < n; ++i)
        labels[i] = static_cast<int>(i % static_cast<size_t>(num_gaussians));
    return labels;
}

}  // namespace eigenix
