#include "bench_utils.hpp"
#include "data_generator.hpp"
#include "kmeans_base.hpp"
#include "kmeans_blas.hpp"
#include "kmeans_faiss.hpp"
#include "kmeans_simd.hpp"
#include "kmeans_streaming.hpp"
#include "metrics.hpp"

#include <algorithm>
#include <cstdio>
#include <limits>
#include <memory>
#include <numeric>
#include <omp.h>
#include <string>
#include <thread>
#include <vector>

using namespace eigenix;

static constexpr int DIM = 128;
static constexpr int K = 1000;
static constexpr int NUM_GAUSSIANS = 50;
static constexpr unsigned DATA_SEED = 42;
static constexpr int NUM_RUNS = 3;
static constexpr size_t WARMUP_N = 100000;

struct RunResult {
    double train_ms;
    double assign_ms;
    float inertia;
    int iters;
    double throughput_mvecs;
    double mem_mb;
};

static RunResult run_one(KMeansBase& backend, const float* data, size_t n,
                         int dim, int k) {
    RunResult r{};
    TrainConfig cfg;
    cfg.max_iter = 100;
    cfg.seed = 42;

    ScopedTimer t;
    backend.train(data, n, dim, k, cfg);
    r.train_ms = t.elapsed_ms();

    std::vector<int> labels;
    t.reset();
    backend.assign(data, n, dim, labels);
    r.assign_ms = t.elapsed_ms();

    r.inertia = backend.inertia();
    r.iters = backend.iterations();
    r.throughput_mvecs = (static_cast<double>(n) / 1e6) / (r.assign_ms / 1000.0);
    r.mem_mb = get_peak_rss_mb();
    return r;
}

static void print_table_header() {
    std::printf("%-18s | %8s | %10s | %10s | %12s | %5s | %8s | %8s\n",
        "Backend", "N", "Train(ms)", "Assign(ms)", "Inertia",
        "Iters", "MVec/s", "Mem(MB)");
    std::printf("%s\n", std::string(100, '-').c_str());
}

static void print_table_row(const std::string& name, size_t n,
                            const RunResult& r) {
    std::printf("%-18s | %8zu | %10.1f | %10.1f | %12.4e | %5d | %8.1f | %8.1f\n",
        name.c_str(), n, r.train_ms, r.assign_ms, r.inertia,
        r.iters, r.throughput_mvecs, r.mem_mb);
}

static void print_cluster_health(const std::string& backend_name, size_t n,
                                 const std::vector<ClusterStats>& stats, int k) {
    std::vector<size_t> sizes(stats.size());
    float global_min_dist = std::numeric_limits<float>::max();
    float global_max_dist = 0.0f;
    float global_mean_sum = 0.0f;
    int empty = 0;
    float max_ratio = 0.0f;
    int max_ratio_cluster = 0;

    for (size_t i = 0; i < stats.size(); ++i) {
        sizes[i] = stats[i].count;
        if (stats[i].count == 0) { empty++; continue; }
        if (stats[i].min_dist < global_min_dist)
            global_min_dist = stats[i].min_dist;
        if (stats[i].max_dist > global_max_dist)
            global_max_dist = stats[i].max_dist;
        global_mean_sum += stats[i].mean_dist;
        if (stats[i].radius_ratio > max_ratio) {
            max_ratio = stats[i].radius_ratio;
            max_ratio_cluster = static_cast<int>(i);
        }
    }

    float stddev = compute_cluster_size_stddev(sizes.data(), k);
    size_t min_sz = *std::min_element(sizes.begin(), sizes.end());
    size_t max_sz = *std::max_element(sizes.begin(), sizes.end());
    double mean_sz = static_cast<double>(
        std::accumulate(sizes.begin(), sizes.end(), size_t(0))) / k;
    float global_mean = global_mean_sum / static_cast<float>(k - empty);

    std::printf("\nCluster Health Report (%s, N=%zu):\n", backend_name.c_str(), n);
    std::printf("  Empty clusters   : %d / %d\n", empty, k);
    std::printf("  Size range       : %zu - %zu  (mean: %.0f, stddev: %.0f)\n",
        min_sz, max_sz, mean_sz, static_cast<double>(stddev));
    std::printf("  Dist min/max/mean: %.3f / %.2f / %.2f\n",
        global_min_dist, global_max_dist, global_mean);
    std::printf("  Max radius ratio : %.2f  (cluster #%d)\n\n",
        max_ratio, max_ratio_cluster);
}

int main() {
    int nthreads = static_cast<int>(std::thread::hardware_concurrency());
    if (nthreads <= 0) nthreads = 1;
    omp_set_num_threads(nthreads);

    std::printf("Eigenix KMeans Benchmark Suite\n");
    std::printf("Threads: %d, D=%d, K=%d\n\n", nthreads, DIM, K);
    check_cpu_governor();

    const std::vector<size_t> dataset_sizes = {
        1'000'000, 2'000'000, 5'000'000, 10'000'000
    };

    // Generate largest dataset once; smaller runs use a prefix.
    size_t max_n = *std::max_element(dataset_sizes.begin(), dataset_sizes.end());
    std::printf("Generating %zu vectors (D=%d, %d Gaussians, seed=%u)...\n",
        max_n, DIM, NUM_GAUSSIANS, DATA_SEED);
    auto all_data = generate_gaussian_mixture(max_n, DIM, NUM_GAUSSIANS, DATA_SEED);
    std::printf("Data generation complete (%.1f MB)\n\n",
        static_cast<double>(all_data.size() * sizeof(float)) / (1024.0 * 1024.0));

    // CSV output
    CsvWriter csv("results/benchmark_results.csv");
    if (!csv.is_open()) {
        std::fprintf(stderr, "WARNING: could not open results/benchmark_results.csv\n");
    }
    csv.write_header({"backend", "n", "run", "train_ms", "assign_ms", "inertia",
                       "iterations", "throughput_mvecs_per_sec", "memory_mb",
                       "cluster_size_min", "cluster_size_max",
                       "cluster_size_stddev", "empty_clusters"});

    // Warmup data
    std::printf("Warming up backends with %zu vectors...\n\n", WARMUP_N);
    auto warmup_data = generate_gaussian_mixture(WARMUP_N, DIM, NUM_GAUSSIANS, 99);

    // Backends
    BlasKMeans blas;
    FaissKMeans faiss_km;
    SimdKMeans simd;
    StreamingKMeans streaming(nullptr, 50000);

    std::vector<KMeansBase*> backends = {&faiss_km, &blas, &simd, &streaming};

    // Warmup each backend.
    int warmup_k = std::min(K, static_cast<int>(WARMUP_N / 10));
    for (auto* b : backends) {
        TrainConfig wcfg;
        wcfg.max_iter = 10;
        wcfg.seed = 99;
        b->train(warmup_data.data(), WARMUP_N, DIM, warmup_k, wcfg);
    }
    std::printf("Warmup complete.\n\n");

    print_table_header();

    for (size_t n : dataset_sizes) {
        const float* data = all_data.data();  // use prefix of length n

        for (auto* backend : backends) {
            std::vector<RunResult> runs(NUM_RUNS);

            for (int r = 0; r < NUM_RUNS; ++r) {
                runs[r] = run_one(*backend, data, n, DIM, K);

                // CSV row
                std::vector<int> labels;
                backend->assign(data, n, DIM, labels);
                std::vector<size_t> sizes(K);
                compute_cluster_sizes(labels.data(), n, K, sizes.data());
                float sd = compute_cluster_size_stddev(sizes.data(), K);
                int empty = count_empty_clusters(sizes.data(), K);
                size_t min_sz = *std::min_element(sizes.begin(), sizes.end());
                size_t max_sz = *std::max_element(sizes.begin(), sizes.end());

                csv.write_row(backend->name(), n, r + 1,
                    runs[r].train_ms, runs[r].assign_ms,
                    runs[r].inertia, runs[r].iters,
                    runs[r].throughput_mvecs, runs[r].mem_mb,
                    min_sz, max_sz, sd, empty);
            }

            // Report mean across runs.
            RunResult mean{};
            for (auto& r : runs) {
                mean.train_ms += r.train_ms;
                mean.assign_ms += r.assign_ms;
                mean.inertia += r.inertia;
                mean.iters += r.iters;
                mean.throughput_mvecs += r.throughput_mvecs;
                mean.mem_mb += r.mem_mb;
            }
            mean.train_ms /= NUM_RUNS;
            mean.assign_ms /= NUM_RUNS;
            mean.inertia /= NUM_RUNS;
            mean.iters /= NUM_RUNS;
            mean.throughput_mvecs /= NUM_RUNS;
            mean.mem_mb /= NUM_RUNS;

            print_table_row(backend->name(), n, mean);

            // Cluster health for last run.
            auto stats = backend->cluster_stats(data, n, DIM);
            print_cluster_health(backend->name(), n, stats, K);
        }
    }

    csv.flush();
    std::printf("\nResults written to results/benchmark_results.csv\n");
    return 0;
}
