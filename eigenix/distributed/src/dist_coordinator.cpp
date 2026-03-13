#include "dist_net.hpp"
#include "dist_protocol.hpp"
#include "data_generator.hpp"
#include "kmeans_blas.hpp"
#include "metrics.hpp"

#include <algorithm>
#include <chrono>
#include <cmath>
#include <cstdio>
#include <cstring>
#include <fstream>
#include <limits>
#include <numeric>
#include <random>
#include <string>
#include <thread>
#include <vector>

namespace {

// Thin subclass to inject centroids and use BLAS-accelerated assign.
class CoordKMeans : public eigenix::BlasKMeans {
public:
    void set_centroids(const float* c, int k, int dim) {
        k_ = k;
        dim_ = dim;
        centroids_.assign(c, c + static_cast<size_t>(k) * dim);
        compute_centroid_norms();
    }
    void train(const float*, size_t, int, int, const eigenix::TrainConfig&) override {}
    std::string name() const override { return "CoordFinalAssign"; }
};

struct CoordArgs {
    std::string workers_file;
    size_t n = 1000000;
    int k = 256;
    int dim = 128;
    size_t max_iter = 100;
    float tol = 0.01f;
    float train_fraction = 0.3f;
    unsigned seed = 42;
    bool verbose = false;
};

CoordArgs parse_args(int argc, char* argv[]) {
    CoordArgs args;
    for (int i = 1; i < argc; ++i) {
        std::string a = argv[i];
        if (a == "--workers" && i + 1 < argc) args.workers_file = argv[++i];
        else if (a == "--n" && i + 1 < argc) args.n = std::stoull(argv[++i]);
        else if (a == "--k" && i + 1 < argc) args.k = std::stoi(argv[++i]);
        else if (a == "--dim" && i + 1 < argc) args.dim = std::stoi(argv[++i]);
        else if (a == "--max-iter" && i + 1 < argc) args.max_iter = std::stoull(argv[++i]);
        else if (a == "--tol" && i + 1 < argc) args.tol = std::stof(argv[++i]);
        else if (a == "--train-fraction" && i + 1 < argc) args.train_fraction = std::stof(argv[++i]);
        else if (a == "--seed" && i + 1 < argc) args.seed = static_cast<unsigned>(std::stoul(argv[++i]));
        else if (a == "--verbose") args.verbose = true;
        else {
            std::fprintf(stderr,
                "Usage: dist_coordinator --workers FILE --n N --k K --dim D\n"
                "       [--max-iter I] [--tol T] [--train-fraction F] [--seed S] [--verbose]\n");
            std::exit(1);
        }
    }
    if (args.workers_file.empty()) {
        std::fprintf(stderr, "Error: --workers is required\n");
        std::exit(1);
    }
    if (args.train_fraction <= 0.0f || args.train_fraction > 1.0f) {
        std::fprintf(stderr, "Error: --train-fraction must be in (0, 1]\n");
        std::exit(1);
    }
    return args;
}

struct WorkerAddr {
    std::string host;
    uint16_t port;
};

std::vector<WorkerAddr> load_workers(const std::string& path) {
    std::vector<WorkerAddr> addrs;
    std::ifstream f(path);
    if (!f.is_open()) {
        std::fprintf(stderr, "Cannot open workers file: %s\n", path.c_str());
        std::exit(1);
    }
    std::string line;
    while (std::getline(f, line)) {
        // Trim whitespace.
        while (!line.empty() && (line.back() == ' ' || line.back() == '\t' || line.back() == '\r'))
            line.pop_back();
        if (line.empty() || line[0] == '#') continue;
        WorkerAddr wa;
        if (!eigenix::dist::parse_host_port(line, wa.host, wa.port)) {
            std::fprintf(stderr, "Bad worker address: %s\n", line.c_str());
            std::exit(1);
        }
        addrs.push_back(wa);
    }
    return addrs;
}

}  // namespace

int main(int argc, char* argv[]) {
    auto args = parse_args(argc, argv);
    auto worker_addrs = load_workers(args.workers_file);
    int N = static_cast<int>(worker_addrs.size());

    if (N == 0) {
        std::fprintf(stderr, "No workers in %s\n", args.workers_file.c_str());
        return 1;
    }

    // Compute n_train from fraction (matching single-node bench methodology).
    size_t n_train = static_cast<size_t>(static_cast<double>(args.n) * args.train_fraction);
    if (n_train < static_cast<size_t>(args.k)) n_train = static_cast<size_t>(args.k);
    if (n_train > args.n) n_train = args.n;

    std::fprintf(stderr, "[COORD] %d workers, n_total=%zu, n_train=%zu (%.0f%%), k=%d, dim=%d, max_iter=%zu, tol=%.4f, seed=%u\n",
                 N, args.n, n_train, args.train_fraction * 100.0f,
                 args.k, args.dim, args.max_iter, args.tol, args.seed);

    using namespace eigenix::dist;
    using Clock = std::chrono::steady_clock;

    // ========== Generate full dataset ==========
    auto t_gen = Clock::now();
    auto data = eigenix::generate_gaussian_mixture(args.n, args.dim, args.k, args.seed);
    double gen_ms = std::chrono::duration<double, std::milli>(Clock::now() - t_gen).count();
    std::fprintf(stderr, "[COORD] Data generated in %.0fms (%.1f MB)\n",
                 gen_ms, static_cast<double>(data.size() * sizeof(float)) / (1024.0 * 1024.0));

    // Build a randomly-shuffled training subsample — identical to main_bench.cpp methodology.
    // generate_gaussian_mixture produces points in cluster order, so contiguous slicing would
    // give a biased subsample covering only a fraction of the Gaussian components.
    std::vector<size_t> shuffle_indices(args.n);
    std::iota(shuffle_indices.begin(), shuffle_indices.end(), size_t(0));
    {
        std::mt19937 shuffle_rng(args.seed);
        std::shuffle(shuffle_indices.begin(), shuffle_indices.end(), shuffle_rng);
    }

    // Build contiguous train buffer by copying rows in shuffled order (matching bench).
    auto t_subsample = Clock::now();
    std::vector<float> train_data(n_train * static_cast<size_t>(args.dim));
    #pragma omp parallel for schedule(static)
    for (size_t i = 0; i < n_train; ++i)
        std::memcpy(train_data.data() + i * args.dim,
                    data.data() + shuffle_indices[i] * args.dim,
                    static_cast<size_t>(args.dim) * sizeof(float));
    std::fprintf(stderr, "[COORD] Training subsample built in %.0fms (%.1f MB, random shuffle)\n",
                 std::chrono::duration<double, std::milli>(Clock::now() - t_subsample).count(),
                 static_cast<double>(train_data.size() * sizeof(float)) / (1024.0 * 1024.0));

    // ========== Compute shard offsets over n_train ==========
    std::vector<uint32_t> shard_n(N);
    std::vector<size_t> shard_offset(N);
    size_t base_sz = n_train / static_cast<size_t>(N);
    size_t remainder = n_train % static_cast<size_t>(N);
    for (int i = 0; i < N; ++i) {
        shard_n[i] = static_cast<uint32_t>(base_sz + (static_cast<size_t>(i) < remainder ? 1 : 0));
        shard_offset[i] = (i == 0) ? 0 : shard_offset[i - 1] + shard_n[i - 1];
    }

    // ========== Connect to workers ==========
    std::vector<int> worker_fds(N);
    for (int i = 0; i < N; ++i) {
        std::fprintf(stderr, "[COORD] Connecting to worker %d at %s:%d ...\n",
                     i, worker_addrs[i].host.c_str(), worker_addrs[i].port);
        worker_fds[i] = connect_to(worker_addrs[i].host, worker_addrs[i].port);
        if (worker_fds[i] < 0) {
            std::fprintf(stderr, "[COORD] Failed to connect to worker %d\n", i);
            return 1;
        }
    }
    std::fprintf(stderr, "[COORD] All workers connected\n");

    // ========== Send training shards (from shuffled train_data) ==========
    auto t_shard = Clock::now();
    for (int i = 0; i < N; ++i) {
        ShardConfig cfg{};
        cfg.n_local = shard_n[i];
        cfg.dim = static_cast<uint32_t>(args.dim);
        cfg.k = static_cast<uint32_t>(args.k);
        cfg.max_iter = static_cast<uint32_t>(args.max_iter);
        cfg.tol = args.tol;
        cfg.seed = args.seed;
        cfg.worker_id = static_cast<uint32_t>(i);
        cfg.n_workers = static_cast<uint32_t>(N);

        send_msg(worker_fds[i], MsgType::SHARD_CONFIG, &cfg, sizeof(ShardConfig));

        // Use header + raw send_all for SHARD_DATA to support payloads > 4 GB.
        uint64_t shard_bytes = static_cast<uint64_t>(shard_n[i]) * args.dim * sizeof(float);
        if (!send_msg_header(worker_fds[i], MsgType::SHARD_DATA, shard_bytes) ||
            !send_all(worker_fds[i], train_data.data() + shard_offset[i] * args.dim,
                      static_cast<size_t>(shard_bytes))) {
            std::fprintf(stderr, "[COORD] Failed sending shard to worker %d\n", i);
            return 1;
        }
    }

    // Wait for READY from all workers.
    for (int i = 0; i < N; ++i) {
        MsgHeader hdr{};
        std::vector<uint8_t> payload;
        if (!recv_msg(worker_fds[i], hdr, payload) || MsgType(hdr.msg_type) != MsgType::READY) {
            std::fprintf(stderr, "[COORD] Worker %d did not send READY\n", i);
            return 1;
        }
    }
    double shard_ms = std::chrono::duration<double, std::milli>(Clock::now() - t_shard).count();
    std::fprintf(stderr, "[COORD] Training shards distributed in %.0fms\n", shard_ms);

    // ========== Initialize centroids (random from shuffled training data) ==========
    int k = args.k;
    int dim = args.dim;
    size_t kd = static_cast<size_t>(k) * dim;
    std::vector<float> coord_centroids(kd);

    {
        // train_data is already randomly shuffled — just pick the first k rows.
        for (int c = 0; c < k; ++c)
            std::memcpy(coord_centroids.data() + static_cast<size_t>(c) * dim,
                        train_data.data() + static_cast<size_t>(c) * dim,
                        static_cast<size_t>(dim) * sizeof(float));
    }

    // ========== Training iteration loop ==========
    std::vector<float> prev_centroids(kd);
    std::vector<double> centroid_sums_d(kd);
    std::vector<uint64_t> centroid_counts(k);
    std::vector<size_t> centroid_counts_sz(k);  // for fix_clusters

    double total_comm_ms = 0.0;
    double total_compute_ms = 0.0;

    auto t_train = Clock::now();
    int final_iter = 0;
    float final_shift = 0.0f;

    for (size_t iter = 0; iter < args.max_iter; ++iter) {
        // --- Broadcast centroids (parallel) ---
        auto t_comm = Clock::now();
        {
            std::vector<bool> send_ok(N, false);
            std::vector<std::thread> send_threads;
            for (int i = 0; i < N; ++i) {
                send_threads.emplace_back([&, i]() {
                    send_ok[i] = send_msg(worker_fds[i], MsgType::CENTROIDS,
                                          coord_centroids.data(), static_cast<uint64_t>(kd * sizeof(float)));
                });
            }
            for (auto& t : send_threads) t.join();
            for (int i = 0; i < N; ++i) {
                if (!send_ok[i]) {
                    std::fprintf(stderr, "[COORD] Failed sending centroids to worker %d\n", i);
                    return 1;
                }
            }
        }

        // --- Collect LOCAL_STATS (parallel) ---
        std::fill(centroid_sums_d.begin(), centroid_sums_d.end(), 0.0);
        std::fill(centroid_counts.begin(), centroid_counts.end(), 0ULL);

        // Per-worker buffers to receive into in parallel, then reduce.
        std::vector<std::vector<uint8_t>> worker_payloads(N);
        {
            std::vector<bool> recv_ok(N, false);
            std::vector<std::thread> recv_threads;
            for (int i = 0; i < N; ++i) {
                recv_threads.emplace_back([&, i]() {
                    MsgHeader hdr{};
                    if (recv_msg(worker_fds[i], hdr, worker_payloads[i]) &&
                        MsgType(hdr.msg_type) == MsgType::LOCAL_STATS) {
                        recv_ok[i] = true;
                    }
                });
            }
            for (auto& t : recv_threads) t.join();
            for (int i = 0; i < N; ++i) {
                if (!recv_ok[i]) {
                    std::fprintf(stderr, "[COORD] Worker %d did not send LOCAL_STATS at iter %zu\n", i, iter);
                    return 1;
                }
            }
        }

        // Reduce worker stats (sequential — small data).
        for (int i = 0; i < N; ++i) {
            const float* wsum = reinterpret_cast<const float*>(worker_payloads[i].data());
            const uint64_t* wcnt = reinterpret_cast<const uint64_t*>(
                worker_payloads[i].data() + kd * sizeof(float));
            for (int c = 0; c < k; ++c) {
                centroid_counts[c] += wcnt[c];
                for (int j = 0; j < dim; ++j)
                    centroid_sums_d[static_cast<size_t>(c) * dim + j] +=
                        static_cast<double>(wsum[static_cast<size_t>(c) * dim + j]);
            }
        }
        double comm_ms = std::chrono::duration<double, std::milli>(Clock::now() - t_comm).count();
        total_comm_ms += comm_ms;

        // --- Update centroids ---
        auto t_compute = Clock::now();
        std::copy(coord_centroids.begin(), coord_centroids.end(), prev_centroids.begin());

        for (int c = 0; c < k; ++c) {
            if (centroid_counts[c] == 0) continue;
            double inv = 1.0 / static_cast<double>(centroid_counts[c]);
            for (int j = 0; j < dim; ++j)
                coord_centroids[static_cast<size_t>(c) * dim + j] =
                    static_cast<float>(centroid_sums_d[static_cast<size_t>(c) * dim + j] * inv);
        }

        // Phase 1 + Phase 2 cluster fix.
        for (int c = 0; c < k; ++c)
            centroid_counts_sz[c] = static_cast<size_t>(centroid_counts[c]);
        eigenix::BlasKMeans::fix_clusters(coord_centroids.data(), centroid_counts_sz.data(),
                                          k, dim, n_train);

        // Compute max_shift.
        float max_shift = 0.0f;
        for (int c = 0; c < k; ++c) {
            float shift = 0.0f;
            for (int j = 0; j < dim; ++j) {
                float d = coord_centroids[static_cast<size_t>(c) * dim + j]
                        - prev_centroids[static_cast<size_t>(c) * dim + j];
                shift += d * d;
            }
            shift = std::sqrt(shift);
            if (shift > max_shift) max_shift = shift;
        }

        double compute_ms = std::chrono::duration<double, std::milli>(Clock::now() - t_compute).count();
        total_compute_ms += compute_ms;

        final_iter = static_cast<int>(iter + 1);
        final_shift = max_shift;

        if (args.verbose && (iter == 0 || (iter + 1) % 5 == 0 ||
                             max_shift <= args.tol || iter + 1 == args.max_iter)) {
            double elapsed = std::chrono::duration<double, std::milli>(Clock::now() - t_train).count();
            // Per-iter bytes: N * (centroid broadcast + LOCAL_STATS reply) per worker
            double bytes_per_iter = static_cast<double>(N) * 2.0 * kd * sizeof(float)
                                  + static_cast<double>(N) * k * sizeof(uint64_t);
            double bw_mbps = (bytes_per_iter / (1024.0 * 1024.0)) / (comm_ms / 1000.0);
            std::fprintf(stderr, "[COORD] iter %zu/%zu  max_shift=%.4f  comm=%.1fms (%.1f MB/s)  update=%.1fms  elapsed=%.0fms\n",
                         iter + 1, args.max_iter, max_shift, comm_ms, bw_mbps, compute_ms, elapsed);
        }

        if (max_shift <= args.tol) {
            // Broadcast DONE to all workers.
            for (int i = 0; i < N; ++i)
                send_msg(worker_fds[i], MsgType::DONE);
            break;
        }

        // If last iteration, also send DONE.
        if (iter + 1 == args.max_iter) {
            for (int i = 0; i < N; ++i)
                send_msg(worker_fds[i], MsgType::DONE);
        }
    }

    double train_ms = std::chrono::duration<double, std::milli>(Clock::now() - t_train).count();

    // ========== Final assign on ALL n_total points (BLAS-accelerated) ==========
    auto t_assign = Clock::now();
    CoordKMeans final_km;
    final_km.set_centroids(coord_centroids.data(), k, dim);
    std::vector<int> final_labels;
    final_km.assign(data.data(), args.n, dim, final_labels);
    double assign_ms = std::chrono::duration<double, std::milli>(Clock::now() - t_assign).count();

    // ========== Final metrics (over all n_total) ==========
    float inertia = eigenix::compute_inertia(data.data(), args.n, dim,
                                              final_labels.data(), coord_centroids.data(), k);

    std::vector<size_t> sizes(k);
    eigenix::compute_cluster_sizes(final_labels.data(), args.n, k, sizes.data());
    int n_empty = eigenix::count_empty_clusters(sizes.data(), k);
    float imbalance = eigenix::compute_imbalance_ratio(sizes.data(), k);
    float size_stddev = eigenix::compute_cluster_size_stddev(sizes.data(), k);

    size_t size_min = *std::min_element(sizes.begin(), sizes.end());
    size_t size_max = *std::max_element(sizes.begin(), sizes.end());

    double throughput = static_cast<double>(args.n) / (assign_ms / 1000.0) / 1e6;

    // ========== Report ==========
    std::fprintf(stderr, "\n========== Distributed K-Means Results ==========\n");
    std::fprintf(stderr, "Workers:           %d\n", N);
    std::fprintf(stderr, "Points (total):    %zu\n", args.n);
    std::fprintf(stderr, "Points (train):    %zu (%.0f%%)\n", n_train, args.train_fraction * 100.0f);
    std::fprintf(stderr, "Dimensions:        %d\n", dim);
    std::fprintf(stderr, "Clusters:          %d\n", k);
    std::fprintf(stderr, "Convergence:       iter=%d, max_shift=%.4f\n", final_iter, final_shift);
    std::fprintf(stderr, "Inertia:           %.6e\n", inertia);
    std::fprintf(stderr, "Empty clusters:    %d / %d\n", n_empty, k);
    std::fprintf(stderr, "Imbalance ratio:   %.2f\n", imbalance);
    std::fprintf(stderr, "Cluster size min:  %zu\n", size_min);
    std::fprintf(stderr, "Cluster size max:  %zu\n", size_max);
    std::fprintf(stderr, "Cluster size std:  %.1f\n", size_stddev);
    std::fprintf(stderr, "Data gen time:     %.0fms\n", gen_ms);
    std::fprintf(stderr, "Shard dist time:   %.0fms\n", shard_ms);
    std::fprintf(stderr, "Training time:     %.0fms\n", train_ms);
    std::fprintf(stderr, "Assign time:       %.0fms (all %zu points)\n", assign_ms, args.n);
    std::fprintf(stderr, "Throughput:        %.2f Mvecs/sec\n", throughput);
    std::fprintf(stderr, "Avg comm/iter:     %.1fms\n", total_comm_ms / final_iter);
    std::fprintf(stderr, "Avg compute/iter:  %.1fms\n", total_compute_ms / final_iter);
    std::fprintf(stderr, "================================================\n");

    // Cleanup.
    for (int i = 0; i < N; ++i)
        close_fd(worker_fds[i]);

    return 0;
}
