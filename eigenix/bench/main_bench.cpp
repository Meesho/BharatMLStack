#include "bench_utils.hpp"
#include "data_generator.hpp"
#include "kmeans_base.hpp"
#include "kmeans_blas.hpp"
#include "kmeans_faiss.hpp"
#include "kmeans_simd.hpp"
#include "metrics.hpp"

#include <algorithm>
#include <cstdarg>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <limits>
#include <memory>
#include <numeric>
#include <omp.h>
#include <random>
#include <string>
#include <thread>
#include <vector>
#if defined(__linux__)
#include <malloc.h>
#endif

using namespace eigenix;

static constexpr int DIM = 2000;
static constexpr int DEFAULT_K = 1000;
static constexpr int NUM_GAUSSIANS = 50;
static constexpr unsigned DEFAULT_DATA_SEED = 42;
static constexpr int NUM_RUNS = 1;       // overridden by EIGENIX_BENCH_RUNS
static constexpr size_t DEFAULT_WARMUP_N = 10000;
static constexpr double DEFAULT_TRAIN_RATIO = 0.3;  // overridden by EIGENIX_BENCH_TRAIN_RATIO

// Unbuffered progress log to stderr so output appears immediately
// even when stdout is piped or redirected.
static void log_progress(const char* fmt, ...) {
    va_list args;
    va_start(args, fmt);
    std::fprintf(stderr, "[BENCH] ");
    std::vfprintf(stderr, fmt, args);
    std::fprintf(stderr, "\n");
    va_end(args);
}

struct RunResult {
    double train_ms;
    double assign_ms;
    float inertia;
    int iters;
    double throughput_mvecs;
    double mem_mb;
};

static RunResult run_one(KMeansBase& backend, const float* data, size_t n,
                         int dim, int k, int nredo = 1) {
    RunResult r{};
    TrainConfig cfg;
    cfg.max_iter = 100;
    cfg.tol = 0.01f;
    cfg.seed = 42;
    cfg.nredo = nredo;
    cfg.verbose = true;

    log_progress("  [train] %s starting (N=%zu, K=%d, max_iter=%zu, tol=%.2e, nredo=%d)...",
                 backend.name().c_str(), n, k, cfg.max_iter, static_cast<double>(cfg.tol), nredo);
    ScopedTimer t;
    backend.train(data, n, dim, k, cfg);
    r.train_ms = t.elapsed_ms();
    log_progress("  [train] %s done in %.1fms (%d iters)",
                 backend.name().c_str(), r.train_ms, backend.iterations());

    log_progress("  [assign] %s starting (N=%zu)...", backend.name().c_str(), n);
    std::vector<int> labels;
    t.reset();
    backend.assign(data, n, dim, labels);
    r.assign_ms = t.elapsed_ms();
    log_progress("  [assign] %s done in %.1fms", backend.name().c_str(), r.assign_ms);

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
    std::fflush(stdout);
}

static void print_table_row(const std::string& name, size_t n,
                            const RunResult& r) {
    std::printf("%-18s | %8zu | %10.1f | %10.1f | %12.4e | %5d | %8.1f | %8.1f\n",
        name.c_str(), n, r.train_ms, r.assign_ms, r.inertia,
        r.iters, r.throughput_mvecs, r.mem_mb);
    std::fflush(stdout);
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
    std::fflush(stdout);
}

int main() {
    int nthreads = static_cast<int>(std::thread::hardware_concurrency());
    if (nthreads <= 0) nthreads = 1;
    omp_set_num_threads(nthreads);

    const char* env_k = std::getenv("EIGENIX_BENCH_K");
    int K = env_k ? std::stoi(env_k) : DEFAULT_K;

    const char* env_warmup = std::getenv("EIGENIX_BENCH_WARMUP");
    size_t WARMUP_N = env_warmup
        ? static_cast<size_t>(std::stoul(env_warmup)) : DEFAULT_WARMUP_N;

    const char* env_runs = std::getenv("EIGENIX_BENCH_RUNS");
    int num_runs = env_runs ? std::stoi(env_runs) : NUM_RUNS;
    if (num_runs < 1) num_runs = 1;

    const char* env_nredo = std::getenv("EIGENIX_BENCH_NREDO");
    int bench_nredo = env_nredo ? std::stoi(env_nredo) : 1;
    if (bench_nredo < 1) bench_nredo = 1;

    const char* env_ratio = std::getenv("EIGENIX_BENCH_TRAIN_RATIO");
    double train_ratio = env_ratio ? std::stod(env_ratio) : DEFAULT_TRAIN_RATIO;
    if (train_ratio <= 0.0 || train_ratio > 1.0) train_ratio = DEFAULT_TRAIN_RATIO;

    // Dataset sizes: override with EIGENIX_BENCH_N env var (comma-separated).
    std::vector<size_t> dataset_sizes;
    const char* env_n = std::getenv("EIGENIX_BENCH_N");
    if (env_n) {
        std::string s(env_n);
        size_t pos = 0;
        while (pos < s.size()) {
            size_t end = s.find(',', pos);
            if (end == std::string::npos) end = s.size();
            dataset_sizes.push_back(
                static_cast<size_t>(std::stoul(s.substr(pos, end - pos))));
            pos = end + 1;
        }
    }
    if (dataset_sizes.empty()) {
        dataset_sizes = {10'000'000};  // 10M points; use EIGENIX_BENCH_N=50e6,10e6 for 50M gen + 10M run
    }

    // Data seed: EIGENIX_BENCH_DATA_SEED unset = 42 (reproducible). Set to "random"/"rand" for random each run; or set to a number for that seed.
    unsigned data_seed;
    const char* env_data_seed = std::getenv("EIGENIX_BENCH_DATA_SEED");
    if (env_data_seed && env_data_seed[0] != '\0' &&
        (std::strcmp(env_data_seed, "random") == 0 || std::strcmp(env_data_seed, "rand") == 0)) {
        data_seed = static_cast<unsigned>(std::random_device{}());
        log_progress("Data seed: random (%u)", data_seed);
    } else if (env_data_seed && env_data_seed[0] != '\0') {
        try {
            data_seed = static_cast<unsigned>(std::stoul(env_data_seed));
        } catch (...) {
            data_seed = DEFAULT_DATA_SEED;
            log_progress("EIGENIX_BENCH_DATA_SEED invalid, using %u", data_seed);
        }
    } else {
        data_seed = DEFAULT_DATA_SEED;
    }

    // Log resolved config so it's visible immediately.
    std::string sizes_str;
    for (size_t i = 0; i < dataset_sizes.size(); ++i) {
        if (i > 0) sizes_str += ",";
        sizes_str += std::to_string(dataset_sizes[i]);
    }
    log_progress("Config: N=[%s] K=%d train_ratio=%.2f warmup=%zu runs=%d nredo=%d threads=%d dim=%d data_seed=%u",
                 sizes_str.c_str(), K, train_ratio, WARMUP_N, num_runs, bench_nredo, nthreads, DIM, data_seed);

    std::printf("Eigenix KMeans Benchmark Suite\n");
    std::printf("Threads: %d, D=%d, K=%d, train_ratio=%.0f%%\n\n",
                nthreads, DIM, K, train_ratio * 100.0);
    std::fflush(stdout);
    check_cpu_governor();

    // --- Data generation ---
    size_t max_n = *std::max_element(dataset_sizes.begin(), dataset_sizes.end());
    double expected_mb = static_cast<double>(max_n) * DIM * sizeof(float) / (1024.0 * 1024.0);
    log_progress("Generating %zu vectors (D=%d, %d Gaussians, seed=%u, ~%.0f MB)...",
                 max_n, DIM, NUM_GAUSSIANS, data_seed, expected_mb);

    ScopedTimer gen_timer;
    auto all_data = generate_gaussian_mixture(max_n, DIM, NUM_GAUSSIANS, data_seed);
    double gen_elapsed = gen_timer.elapsed_ms();

    double actual_mb = static_cast<double>(all_data.size() * sizeof(float)) / (1024.0 * 1024.0);
    log_progress("Data generation complete: %.1f MB in %.1fs",
                 actual_mb, gen_elapsed / 1000.0);

    std::printf("Generating %zu vectors (D=%d, %d Gaussians, seed=%u)...\n",
        max_n, DIM, NUM_GAUSSIANS, data_seed);
    std::printf("Data generation complete (%.1f MB)\n\n", actual_mb);
    std::fflush(stdout);

    // --- CSV output ---
    CsvWriter csv("results/benchmark_results.csv");
    if (!csv.is_open()) {
        std::fprintf(stderr, "WARNING: could not open results/benchmark_results.csv\n");
    }
    csv.write_header({"backend", "n_total", "n_train", "run", "train_ms", "assign_ms",
                       "inertia", "iterations", "throughput_mvecs_per_sec", "memory_mb",
                       "cluster_size_min", "cluster_size_max",
                       "cluster_size_stddev", "empty_clusters"});

    // --- Warmup ---
    log_progress("Generating warmup data: %zu vectors...", WARMUP_N);
    auto warmup_data = generate_gaussian_mixture(WARMUP_N, DIM, NUM_GAUSSIANS, 99);

    BlasKMeans blas;
    FaissKMeans faiss_km;
    SimdKMeans simd;

    std::vector<KMeansBase*> backends = {&faiss_km, &blas, &simd};
    int num_backends = static_cast<int>(backends.size());

    // FAISS requires >= 39 points per centroid; use that as the divisor so
    // warmup never triggers a "too few training points" warning.
    int warmup_k = std::min(K, static_cast<int>(WARMUP_N / 39));
    if (warmup_k < 2) warmup_k = 2;
    for (int bi = 0; bi < num_backends; ++bi) {
        auto* b = backends[bi];
        log_progress("Warmup: %s (%d/%d)...", b->name().c_str(), bi + 1, num_backends);
        ScopedTimer wt;
        TrainConfig wcfg;
        wcfg.max_iter = 10;
        wcfg.seed = 99;
        b->train(warmup_data.data(), WARMUP_N, DIM, warmup_k, wcfg);
        log_progress("Warmup: %s done (%.1fs)", b->name().c_str(), wt.elapsed_ms() / 1000.0);
    }

    std::printf("Warming up backends with %zu vectors...\n\n", WARMUP_N);
    std::printf("Warmup complete.\n\n");
    std::fflush(stdout);
    log_progress("Warmup complete for all %d backends", num_backends);

    // --- Main benchmark ---
    print_table_header();
    int num_datasets = static_cast<int>(dataset_sizes.size());

    for (int di = 0; di < num_datasets; ++di) {
        size_t n_total = dataset_sizes[di];
        const float* full_data = all_data.data();

        // Subsample train_ratio of the total data for training.
        size_t n_train = static_cast<size_t>(static_cast<double>(n_total) * train_ratio);
        if (n_train < 1) n_train = 1;

        // Build a random subsample (without replacement) by shuffling indices.
        std::vector<size_t> indices(n_total);
        std::iota(indices.begin(), indices.end(), size_t(0));
        std::mt19937 shuffle_rng(data_seed);
        std::shuffle(indices.begin(), indices.end(), shuffle_rng);

        std::vector<float> train_data(n_train * DIM);
        for (size_t i = 0; i < n_train; ++i)
            std::memcpy(train_data.data() + i * DIM,
                        full_data + indices[i] * DIM,
                        DIM * sizeof(float));

        int k = std::min(K, static_cast<int>(n_train / 10));
        if (k < 2) k = 2;

        log_progress("=== Dataset %d/%d: N_total=%zu, N_train=%zu (%.0f%%), K=%d ===",
                     di + 1, num_datasets, n_total, n_train, train_ratio * 100.0, k);

        for (int bi = 0; bi < num_backends; ++bi) {
            auto* backend = backends[bi];
            log_progress("Starting %s (%d/%d) on N_train=%zu / N_total=%zu",
                         backend->name().c_str(), bi + 1, num_backends, n_train, n_total);

            std::vector<RunResult> runs(static_cast<size_t>(num_runs));

            for (int r = 0; r < num_runs; ++r) {
                log_progress("%s run %d/%d starting...", backend->name().c_str(), r + 1, num_runs);
                ScopedTimer run_timer;

                // Train on the subsample, assign on the full dataset.
                runs[r] = run_one(*backend, train_data.data(), n_train, DIM, k, bench_nredo);

                log_progress("  [metrics] assigning full dataset (N=%zu)...", n_total);
                std::vector<int> labels;
                backend->assign(full_data, n_total, DIM, labels);
                std::vector<size_t> sizes(k);
                compute_cluster_sizes(labels.data(), n_total, k, sizes.data());
                float sd = compute_cluster_size_stddev(sizes.data(), k);
                int empty = count_empty_clusters(sizes.data(), k);
                size_t min_sz = *std::min_element(sizes.begin(), sizes.end());
                size_t max_sz = *std::max_element(sizes.begin(), sizes.end());

                csv.write_row(backend->name(), n_total, n_train, r + 1,
                    runs[r].train_ms, runs[r].assign_ms,
                    runs[r].inertia, runs[r].iters,
                    runs[r].throughput_mvecs, runs[r].mem_mb,
                    min_sz, max_sz, sd, empty);

                log_progress("%s run %d/%d done: train=%.0fms assign=%.0fms inertia=%.4e (total %.1fs)",
                    backend->name().c_str(), r + 1, num_runs,
                    runs[r].train_ms, runs[r].assign_ms, runs[r].inertia,
                    run_timer.elapsed_ms() / 1000.0);
            }

            RunResult mean{};
            for (int r = 0; r < num_runs; ++r) {
                mean.train_ms += runs[r].train_ms;
                mean.assign_ms += runs[r].assign_ms;
                mean.inertia += runs[r].inertia;
                mean.iters += runs[r].iters;
                mean.throughput_mvecs += runs[r].throughput_mvecs;
                mean.mem_mb += runs[r].mem_mb;
            }
            mean.train_ms /= num_runs;
            mean.assign_ms /= num_runs;
            mean.inertia /= num_runs;
            mean.iters /= num_runs;
            mean.throughput_mvecs /= num_runs;
            mean.mem_mb /= num_runs;

            print_table_row(backend->name(), n_train, mean);

            log_progress("  [health] computing cluster health for %s on full N=%zu...",
                         backend->name().c_str(), n_total);
            auto stats = backend->cluster_stats(full_data, n_total, DIM);
            print_cluster_health(backend->name(), n_total, stats, k);
            log_progress("Finished %s on N_total=%zu N_train=%zu", backend->name().c_str(), n_total, n_train);

#if defined(__linux__)
            // Return freed memory (e.g. FAISS training buffers) to the OS so the next
            // backend (e.g. BLAS) has more RAM available and is less likely to be OOM-killed.
            malloc_trim(0);
#endif
        }
    }

    csv.flush();
    std::printf("\nResults written to results/benchmark_results.csv\n");
    std::fflush(stdout);
    log_progress("All benchmarks complete. Results in results/benchmark_results.csv");
    return 0;
}
