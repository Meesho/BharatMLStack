#include "ikmeans.hpp"
#include "kmeans_blas.hpp"
#include "kmeans_faiss.hpp"
#include "metrics.hpp"
#include <chrono>
#include <cstdio>
#include <cstdlib>
#include <omp.h>
#include <random>
#include <thread>
#include <vector>

namespace {

void generate_synthetic(float* data, size_t n, size_t dim, unsigned seed) {
    std::mt19937 rng(seed);
    std::normal_distribution<float> dist(0.0f, 1.0f);
    for (size_t i = 0; i < n * dim; ++i)
        data[i] = dist(rng);
}

void print_cluster_sizes(const size_t* sizes, size_t k, const char* prefix) {
    std::printf("%s cluster sizes:", prefix);
    size_t min_sz = sizes[0], max_sz = sizes[0];
    for (size_t i = 0; i < k; ++i) {
        if (i < 5)
            std::printf(" %zu", sizes[i]);
        else if (i == 5)
            std::printf(" ...");
        if (sizes[i] < min_sz) min_sz = sizes[i];
        if (sizes[i] > max_sz) max_sz = sizes[i];
    }
    std::printf(" (min=%zu max=%zu)\n", min_sz, max_sz);
}

}  // namespace

int main(int argc, char** argv) {
    size_t n_train = 100000;
    size_t dim = 128;
    size_t k = 256;
    if (argc >= 2) n_train = static_cast<size_t>(std::atoi(argv[1]));
    if (argc >= 3) dim = static_cast<size_t>(std::atoi(argv[2]));
    if (argc >= 4) k = static_cast<size_t>(std::atoi(argv[3]));

    int nthreads = static_cast<int>(std::thread::hardware_concurrency());
    if (nthreads <= 0) nthreads = 1;
    omp_set_num_threads(nthreads);
    std::printf("Threads: %d\n\n", nthreads);

    std::vector<float> data(n_train * dim);
    generate_synthetic(data.data(), n_train, dim, 12345u);

    std::vector<int> labels_blas(n_train), labels_faiss(n_train);
    std::vector<size_t> sizes(k);

    const size_t max_iter = 300;

    // ---- Custom BLAS KMeans ----
    std::printf("==== Custom BLAS KMeans ====\n");
    eigenix::BlasKMeans blas_kmeans(k, max_iter, 1e-4f);

    auto t0 = std::chrono::steady_clock::now();
    blas_kmeans.train(data.data(), n_train, dim);
    auto t1 = std::chrono::steady_clock::now();
    double train_blas_ms = 1e-6 * std::chrono::duration_cast<std::chrono::nanoseconds>(t1 - t0).count();

    t0 = std::chrono::steady_clock::now();
    blas_kmeans.assign(data.data(), n_train, labels_blas.data());
    t1 = std::chrono::steady_clock::now();
    double assign_blas_ms = 1e-6 * std::chrono::duration_cast<std::chrono::nanoseconds>(t1 - t0).count();

    float inertia_blas = eigenix::compute_inertia(
        data.data(), n_train, dim, labels_blas.data(),
        blas_kmeans.centroids(), blas_kmeans.k());
    eigenix::compute_cluster_sizes(labels_blas.data(), n_train, k, sizes.data());
    float imbalance_blas = eigenix::compute_imbalance_ratio(sizes.data(), k);

    std::printf("Training time: %.2f ms\n", train_blas_ms);
    std::printf("Assignment time: %.2f ms\n", assign_blas_ms);
    std::printf("Inertia: %.4e\n", inertia_blas);
    std::printf("Imbalance ratio: %.4f\n", imbalance_blas);
    print_cluster_sizes(sizes.data(), k, "  ");
    std::printf("\n");

    // ---- FAISS KMeans ----
    std::printf("==== FAISS KMeans ====\n");
    eigenix::FaissKMeans faiss_kmeans(k, max_iter);

    t0 = std::chrono::steady_clock::now();
    faiss_kmeans.train(data.data(), n_train, dim);
    t1 = std::chrono::steady_clock::now();
    double train_faiss_ms = 1e-6 * std::chrono::duration_cast<std::chrono::nanoseconds>(t1 - t0).count();

    t0 = std::chrono::steady_clock::now();
    faiss_kmeans.assign(data.data(), n_train, labels_faiss.data());
    t1 = std::chrono::steady_clock::now();
    double assign_faiss_ms = 1e-6 * std::chrono::duration_cast<std::chrono::nanoseconds>(t1 - t0).count();

    float inertia_faiss = eigenix::compute_inertia(
        data.data(), n_train, dim, labels_faiss.data(),
        faiss_kmeans.centroids(), faiss_kmeans.k());
    eigenix::compute_cluster_sizes(labels_faiss.data(), n_train, k, sizes.data());
    float imbalance_faiss = eigenix::compute_imbalance_ratio(sizes.data(), k);

    std::printf("Training time: %.2f ms\n", train_faiss_ms);
    std::printf("Assignment time: %.2f ms\n", assign_faiss_ms);
    std::printf("Inertia: %.4e\n", inertia_faiss);
    std::printf("Imbalance ratio: %.4f\n", imbalance_faiss);
    print_cluster_sizes(sizes.data(), k, "  ");
    std::printf("\n");

    // ---- Comparison table ----
    std::printf("========================================\n");
    std::printf("%-20s %12s %12s\n", "Metric", "BLAS", "FAISS");
    std::printf("----------------------------------------\n");
    std::printf("%-20s %10.2f ms %10.2f ms\n", "Training time", train_blas_ms, train_faiss_ms);
    std::printf("%-20s %10.2f ms %10.2f ms\n", "Assignment time", assign_blas_ms, assign_faiss_ms);
    std::printf("%-20s %12.4e %12.4e\n", "Inertia", inertia_blas, inertia_faiss);
    std::printf("%-20s %12.4f %12.4f\n", "Imbalance ratio", imbalance_blas, imbalance_faiss);
    std::printf("========================================\n");

    return 0;
}
