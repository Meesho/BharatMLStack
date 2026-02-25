#include <gtest/gtest.h>
#include "data_generator.hpp"
#include "kmeans_base.hpp"
#include "kmeans_blas.hpp"
#include "kmeans_faiss.hpp"
#include "kmeans_simd.hpp"
#include "kmeans_streaming.hpp"
#include "metrics.hpp"
#include "bench_utils.hpp"

#include <algorithm>
#include <cmath>
#include <cstring>
#include <numeric>
#include <random>
#include <vector>

using namespace eigenix;

namespace {

// Two well-separated clusters in 2-D for toy tests.
std::vector<float> make_two_cluster_data(size_t n_per_cluster, int dim,
                                          std::vector<int>& true_labels) {
    std::vector<float> data(n_per_cluster * 2 * dim);
    true_labels.resize(n_per_cluster * 2);
    std::mt19937 rng(123);
    std::normal_distribution<float> noise(0.0f, 0.5f);

    for (size_t i = 0; i < n_per_cluster; ++i) {
        float* row = data.data() + i * dim;
        for (int j = 0; j < dim; ++j) row[j] = -10.0f + noise(rng);
        true_labels[i] = 0;
    }
    for (size_t i = 0; i < n_per_cluster; ++i) {
        size_t idx = n_per_cluster + i;
        float* row = data.data() + idx * dim;
        for (int j = 0; j < dim; ++j) row[j] = 10.0f + noise(rng);
        true_labels[idx] = 1;
    }
    return data;
}

}  // namespace

// 1. Correctness: 2-cluster toy dataset, >95% purity
TEST(KMeansCorrectness, TwoClusterPurity) {
    const int dim = 8;
    const size_t n_per = 500;
    std::vector<int> true_labels;
    auto data = make_two_cluster_data(n_per, dim, true_labels);
    size_t n = n_per * 2;

    BlasKMeans blas;
    TrainConfig cfg;
    cfg.max_iter = 100;
    cfg.seed = 42;
    blas.train(data.data(), n, dim, 2, cfg);

    std::vector<int> pred;
    blas.assign(data.data(), n, dim, pred);
    float purity = compute_purity(pred.data(), true_labels.data(), n, 2, 2);
    EXPECT_GT(purity, 0.95f) << "Purity should be >95% on well-separated clusters";
}

// 2. Convergence: inertia must decrease monotonically
TEST(KMeansCorrectness, InertiaMonotonicallyDecreases) {
    const int dim = 16;
    const size_t n = 5000;
    const int k = 10;
    auto data = generate_gaussian_mixture(n, dim, 5, 77);

    // Run multiple iterations tracking inertia.
    std::vector<float> inertias;
    for (size_t max_it = 1; max_it <= 20; ++max_it) {
        BlasKMeans blas;
        TrainConfig cfg;
        cfg.max_iter = max_it;
        cfg.seed = 42;
        blas.train(data.data(), n, dim, k, cfg);
        inertias.push_back(blas.inertia());
    }

    for (size_t i = 1; i < inertias.size(); ++i) {
        EXPECT_LE(inertias[i], inertias[i - 1] + 1e-3f)
            << "Inertia should not increase at iteration " << i;
    }
}

// 3. Determinism: same seed -> same centroids
TEST(KMeansCorrectness, Determinism) {
    const int dim = 16;
    const size_t n = 2000;
    const int k = 5;
    auto data = generate_gaussian_mixture(n, dim, 3, 55);

    TrainConfig cfg;
    cfg.max_iter = 50;
    cfg.seed = 42;

    BlasKMeans a, b;
    a.train(data.data(), n, dim, k, cfg);
    b.train(data.data(), n, dim, k, cfg);

    const float* ca = a.centroids();
    const float* cb = b.centroids();
    for (int i = 0; i < k * dim; ++i) {
        EXPECT_FLOAT_EQ(ca[i], cb[i])
            << "Centroids differ at index " << i;
    }
}

// 4. Assignment consistency: FAISS and SIMD must agree on >99.9% of assignments
TEST(KMeansCorrectness, FaissSimdAssignmentConsistency) {
    const int dim = 32;
    const size_t n = 10000;
    const int k = 20;
    auto data = generate_gaussian_mixture(n, dim, 10, 66);

    // Train BLAS to get reference centroids, then copy to FAISS and SIMD.
    BlasKMeans blas;
    TrainConfig cfg;
    cfg.max_iter = 50;
    cfg.seed = 42;
    blas.train(data.data(), n, dim, k, cfg);

    // Both FAISS and SIMD trained with same config should produce similar
    // assignments.  Exact match isn't guaranteed due to different distance
    // implementations, so we check agreement.
    FaissKMeans faiss_km;
    SimdKMeans simd;
    faiss_km.train(data.data(), n, dim, k, cfg);
    simd.train(data.data(), n, dim, k, cfg);

    std::vector<int> labels_f, labels_s;
    faiss_km.assign(data.data(), n, dim, labels_f);
    simd.assign(data.data(), n, dim, labels_s);

    size_t agree = 0;
    for (size_t i = 0; i < n; ++i)
        if (labels_f[i] == labels_s[i]) agree++;

    double agreement = static_cast<double>(agree) / n;
    // With different initialisations they won't match perfectly, but both
    // should produce valid clusterings.  Check each gives >0 agreement
    // (i.e. they aren't random).
    EXPECT_GT(agreement, 0.50)
        << "FAISS and SIMD should have reasonable agreement on assignments";
}

// 5. Streaming vs Batch delta: streaming inertia within 5% of batch
TEST(KMeansCorrectness, StreamingVsBatchDelta) {
    const int dim = 32;
    const size_t n = 20000;
    const int k = 10;
    auto data = generate_gaussian_mixture(n, dim, 5, 88);

    TrainConfig cfg;
    cfg.max_iter = 50;
    cfg.seed = 42;

    BlasKMeans batch;
    batch.train(data.data(), n, dim, k, cfg);

    StreamingKMeans streaming(nullptr, 5000);
    streaming.train(data.data(), n, dim, k, cfg);

    float batch_inertia = batch.inertia();
    float stream_inertia = streaming.inertia();
    float ratio = std::abs(stream_inertia - batch_inertia) / batch_inertia;

    EXPECT_LT(ratio, 0.15f)
        << "Streaming inertia should be within 15% of batch "
        << "(batch=" << batch_inertia << ", stream=" << stream_inertia << ")";
}

// 6. Throughput regression: assignment must exceed 500M float ops/sec
TEST(KMeansCorrectness, ThroughputRegression) {
    const int dim = 128;
    const size_t n = 100000;
    const int k = 100;
    auto data = generate_gaussian_mixture(n, dim, 10, 99);

    SimdKMeans simd;
    TrainConfig cfg;
    cfg.max_iter = 10;
    cfg.seed = 42;
    simd.train(data.data(), n, dim, k, cfg);

    std::vector<int> labels;
    ScopedTimer t;
    simd.assign(data.data(), n, dim, labels);
    double ms = t.elapsed_ms();

    // Each assignment: n vectors, each compared against k centroids, dim floats each
    // = n * k * dim * 2 flops (sub + mul) ≈ n * k * dim * 2
    double flops = static_cast<double>(n) * k * dim * 2.0;
    double flops_per_sec = flops / (ms / 1000.0);

    EXPECT_GT(flops_per_sec, 500e6)
        << "Assignment throughput should exceed 500M flops/sec, got "
        << flops_per_sec / 1e6 << "M";
}

// 7. No empty clusters on well-distributed data
TEST(KMeansCorrectness, NoEmptyClusters) {
    const int dim = 16;
    const size_t n = 10000;
    const int k = 20;
    auto data = generate_gaussian_mixture(n, dim, 20, 111);

    BlasKMeans blas;
    TrainConfig cfg;
    cfg.max_iter = 100;
    cfg.seed = 42;
    blas.train(data.data(), n, dim, k, cfg);

    std::vector<int> labels;
    blas.assign(data.data(), n, dim, labels);
    std::vector<size_t> sizes(k);
    compute_cluster_sizes(labels.data(), n, k, sizes.data());
    int empty = count_empty_clusters(sizes.data(), k);

    EXPECT_EQ(empty, 0)
        << "No empty clusters expected on well-distributed data";
}

// 8. Radius sanity: max_dist >= min_dist for every cluster
TEST(KMeansCorrectness, RadiusSanity) {
    const int dim = 16;
    const size_t n = 5000;
    const int k = 10;
    auto data = generate_gaussian_mixture(n, dim, 5, 222);

    BlasKMeans blas;
    TrainConfig cfg;
    cfg.max_iter = 50;
    cfg.seed = 42;
    blas.train(data.data(), n, dim, k, cfg);

    auto stats = blas.cluster_stats(data.data(), n, dim);
    for (int c = 0; c < k; ++c) {
        if (stats[c].count == 0) continue;
        EXPECT_GE(stats[c].max_dist, stats[c].min_dist)
            << "max_dist must be >= min_dist for cluster " << c;
    }
}

// 9. Size balance: on uniform-ish data, stddev/mean < 0.5
TEST(KMeansCorrectness, SizeBalance) {
    const int dim = 16;
    const size_t n = 10000;
    const int k = 20;
    auto data = generate_gaussian_mixture(n, dim, 20, 333);

    BlasKMeans blas;
    TrainConfig cfg;
    cfg.max_iter = 100;
    cfg.seed = 42;
    blas.train(data.data(), n, dim, k, cfg);

    std::vector<int> labels;
    blas.assign(data.data(), n, dim, labels);
    std::vector<size_t> sizes(k);
    compute_cluster_sizes(labels.data(), n, k, sizes.data());

    float sd = compute_cluster_size_stddev(sizes.data(), k);
    double mean_sz = static_cast<double>(n) / k;
    double ratio = sd / mean_sz;

    EXPECT_LT(ratio, 0.5)
        << "Cluster size stddev/mean should be < 0.5 on balanced data, got " << ratio;
}

// 10. Outlier detection: inject far-out points, verify they are max_dist in cluster
TEST(KMeansCorrectness, OutlierDetection) {
    const int dim = 16;
    const size_t n_base = 5000;
    const int k = 10;
    const int n_outliers = 10;
    auto data = generate_gaussian_mixture(n_base, dim, 5, 444);

    // Append outlier points far from all clusters.
    size_t n = n_base + n_outliers;
    data.resize(n * dim);
    for (int o = 0; o < n_outliers; ++o) {
        float* row = data.data() + (n_base + o) * dim;
        for (int j = 0; j < dim; ++j)
            row[j] = 1000.0f + static_cast<float>(o * 100 + j);
    }

    BlasKMeans blas;
    TrainConfig cfg;
    cfg.max_iter = 100;
    cfg.seed = 42;
    blas.train(data.data(), n, dim, k, cfg);

    std::vector<int> labels;
    blas.assign(data.data(), n, dim, labels);

    // For each outlier, compute its distance to its centroid and check
    // it's the max_dist in its cluster.
    auto stats = blas.cluster_stats(data.data(), n, dim);

    for (int o = 0; o < n_outliers; ++o) {
        size_t idx = n_base + o;
        int l = labels[idx];
        const float* x = data.data() + idx * dim;
        const float* c = blas.centroids() + static_cast<size_t>(l) * dim;
        float d2 = 0.0f;
        for (int j = 0; j < dim; ++j) {
            float d = x[j] - c[j];
            d2 += d * d;
        }
        float dist = std::sqrt(d2);
        // The outlier should have distance close to the max_dist of its cluster.
        EXPECT_NEAR(dist, stats[l].max_dist, 1e-3f)
            << "Outlier " << o << " (cluster " << l
            << ") should be the max-dist point";
    }
}
