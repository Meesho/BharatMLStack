#include "dist_net.hpp"
#include "dist_protocol.hpp"
#include "kmeans_blas.hpp"

#include <cblas.h>
#include <chrono>
#include <cmath>
#include <cstdio>
#include <cstring>
#include <omp.h>
#include <vector>

namespace {

inline float sqnorm(const float* x, int dim) {
    float s = 0.0f;
    for (int j = 0; j < dim; ++j) s += x[j] * x[j];
    return s;
}

// Thin subclass to access protected members of BlasKMeans.
class DistWorkerKMeans : public eigenix::BlasKMeans {
public:
    void set_centroids(const float* c, int k, int dim) {
        k_ = k;
        dim_ = dim;
        centroids_.assign(c, c + static_cast<size_t>(k) * dim);
        compute_centroid_norms();
        // Ensure dist_buf_ is sized for assign_batch
        size_t bs = std::min(BATCH_SIZE, static_cast<size_t>(1));  // will be resized in assign_batch
        (void)bs;
    }

    using BlasKMeans::assign_batch;

    // Unused pure virtuals — worker never calls these.
    void train(const float*, size_t, int, int, const eigenix::TrainConfig&) override {}
    std::string name() const override { return "DistWorker"; }
};

struct WorkerArgs {
    uint16_t port = 9001;
    int threads = 0;  // 0 = use OMP default
    bool verbose = false;
};

WorkerArgs parse_args(int argc, char* argv[]) {
    WorkerArgs args;
    for (int i = 1; i < argc; ++i) {
        std::string a = argv[i];
        if (a == "--port" && i + 1 < argc) args.port = static_cast<uint16_t>(std::stoi(argv[++i]));
        else if (a == "--threads" && i + 1 < argc) args.threads = std::stoi(argv[++i]);
        else if (a == "--verbose") args.verbose = true;
        else {
            std::fprintf(stderr, "Usage: dist_worker --port PORT [--threads T] [--verbose]\n");
            std::exit(1);
        }
    }
    return args;
}

}  // namespace

int main(int argc, char* argv[]) {
    auto args = parse_args(argc, argv);

    if (args.threads > 0) omp_set_num_threads(args.threads);

    using namespace eigenix::dist;

    // Listen for coordinator connection.
    int listen_fd = make_listener(args.port);
    if (listen_fd < 0) return 1;
    std::fprintf(stderr, "[WORKER] Listening on port %d\n", args.port);

    int coord_fd = accept_one(listen_fd);
    if (coord_fd < 0) return 1;
    close_fd(listen_fd);
    std::fprintf(stderr, "[WORKER] Coordinator connected\n");

    // Receive SHARD_CONFIG.
    MsgHeader hdr{};
    std::vector<uint8_t> payload;
    if (!recv_msg(coord_fd, hdr, payload) || MsgType(hdr.msg_type) != MsgType::SHARD_CONFIG) {
        std::fprintf(stderr, "[WORKER] Expected SHARD_CONFIG\n");
        close_fd(coord_fd);
        return 1;
    }
    ShardConfig cfg{};
    std::memcpy(&cfg, payload.data(), sizeof(ShardConfig));

    if (args.verbose)
        std::fprintf(stderr, "[WORKER] Config: worker_id=%u n_local=%u dim=%u k=%u max_iter=%u\n",
                     cfg.worker_id, cfg.n_local, cfg.dim, cfg.k, cfg.max_iter);

    // Receive SHARD_DATA — use header-only recv to avoid allocating a
    // temporary vector<uint8_t> (shard can exceed 4 GB).
    if (!recv_msg_header(coord_fd, hdr) || MsgType(hdr.msg_type) != MsgType::SHARD_DATA) {
        std::fprintf(stderr, "[WORKER] Expected SHARD_DATA\n");
        close_fd(coord_fd);
        return 1;
    }

    size_t expected_bytes = static_cast<size_t>(cfg.n_local) * cfg.dim * sizeof(float);
    if (hdr.payload_len != expected_bytes) {
        std::fprintf(stderr, "[WORKER] Shard size mismatch: got %llu, expected %zu\n",
                     static_cast<unsigned long long>(hdr.payload_len), expected_bytes);
        close_fd(coord_fd);
        return 1;
    }

    // Receive shard data directly into the float vector (no intermediate copy).
    std::vector<float> shard(static_cast<size_t>(cfg.n_local) * cfg.dim);
    if (!recv_all(coord_fd, shard.data(), expected_bytes)) {
        std::fprintf(stderr, "[WORKER] Failed to receive shard data\n");
        close_fd(coord_fd);
        return 1;
    }

    // Precompute data norms (done once).
    std::vector<float> data_norms(cfg.n_local);
    #pragma omp parallel for schedule(static)
    for (uint32_t i = 0; i < cfg.n_local; ++i)
        data_norms[i] = sqnorm(shard.data() + static_cast<size_t>(i) * cfg.dim, static_cast<int>(cfg.dim));

    // Send READY.
    if (!send_msg(coord_fd, MsgType::READY)) {
        std::fprintf(stderr, "[WORKER] Failed to send READY\n");
        close_fd(coord_fd);
        return 1;
    }
    std::fprintf(stderr, "[WORKER %u] Ready, shard has %u points\n", cfg.worker_id, cfg.n_local);

    // Prepare worker k-means object and buffers.
    DistWorkerKMeans wkm;
    std::vector<int> labels(cfg.n_local);
    int k = static_cast<int>(cfg.k);
    int dim = static_cast<int>(cfg.dim);
    size_t kd = static_cast<size_t>(k) * dim;

    // Pre-allocate buffers outside the loop to avoid per-iteration heap churn.
    int nthreads = 1;
    #pragma omp parallel
    { nthreads = omp_get_num_threads(); }

    std::vector<double> all_sums(static_cast<size_t>(nthreads) * kd);
    std::vector<uint64_t> all_counts(static_cast<size_t>(nthreads) * k);
    std::vector<float> local_sums(kd);
    std::vector<uint64_t> local_counts(k);
    size_t payload_bytes = kd * sizeof(float) + static_cast<size_t>(k) * sizeof(uint64_t);
    std::vector<uint8_t> out(payload_bytes);

    // Iteration loop.
    for (uint32_t iter = 0; ; ++iter) {
        MsgHeader iter_hdr{};
        std::vector<uint8_t> iter_payload;
        if (!recv_msg(coord_fd, iter_hdr, iter_payload)) {
            std::fprintf(stderr, "[WORKER %u] Connection lost at iter %u\n", cfg.worker_id, iter);
            break;
        }

        if (MsgType(iter_hdr.msg_type) == MsgType::DONE) {
            if (args.verbose)
                std::fprintf(stderr, "[WORKER %u] Received DONE at iter %u\n", cfg.worker_id, iter);
            break;
        }

        if (MsgType(iter_hdr.msg_type) != MsgType::CENTROIDS) {
            std::fprintf(stderr, "[WORKER %u] Unexpected msg type 0x%X\n",
                         cfg.worker_id, iter_hdr.msg_type);
            break;
        }

        auto t0 = std::chrono::steady_clock::now();

        // Load centroids and run assignment.
        wkm.set_centroids(reinterpret_cast<const float*>(iter_payload.data()), k, dim);
        wkm.assign_batch(shard.data(), data_norms.data(), cfg.n_local, labels.data());

        // Accumulate local sums + counts using thread-local buffers.
        std::fill(all_sums.begin(), all_sums.end(), 0.0);
        std::fill(all_counts.begin(), all_counts.end(), 0ULL);

        #pragma omp parallel
        {
            int tid = omp_get_thread_num();
            double* lsums = all_sums.data() + static_cast<size_t>(tid) * kd;
            uint64_t* lcounts = all_counts.data() + static_cast<size_t>(tid) * k;
            #pragma omp for schedule(static)
            for (uint32_t i = 0; i < cfg.n_local; ++i) {
                int l = labels[i];
                const float* x = shard.data() + static_cast<size_t>(i) * dim;
                double* s = lsums + static_cast<size_t>(l) * dim;
                for (int j = 0; j < dim; ++j) s[j] += static_cast<double>(x[j]);
                lcounts[l]++;
            }
        }

        // Reduce thread-local buffers (parallelized over clusters).
        std::fill(local_sums.begin(), local_sums.end(), 0.0f);
        std::fill(local_counts.begin(), local_counts.end(), 0ULL);
        #pragma omp parallel for schedule(static)
        for (int c = 0; c < k; ++c) {
            for (int j = 0; j < dim; ++j) {
                double acc = 0.0;
                for (int t = 0; t < nthreads; ++t)
                    acc += all_sums[static_cast<size_t>(t) * kd + static_cast<size_t>(c) * dim + j];
                local_sums[static_cast<size_t>(c) * dim + j] = static_cast<float>(acc);
            }
            for (int t = 0; t < nthreads; ++t)
                local_counts[c] += all_counts[static_cast<size_t>(t) * k + c];
        }

        auto t1 = std::chrono::steady_clock::now();
        double compute_ms = std::chrono::duration<double, std::milli>(t1 - t0).count();

        if (args.verbose && (iter % 5 == 0))
            std::fprintf(stderr, "[WORKER %u] iter %u compute=%.1fms\n",
                         cfg.worker_id, iter, compute_ms);

        // Pack and send LOCAL_STATS: [k*dim floats | k uint64_t counts].
        std::memcpy(out.data(), local_sums.data(), kd * sizeof(float));
        std::memcpy(out.data() + kd * sizeof(float), local_counts.data(),
                    static_cast<size_t>(k) * sizeof(uint64_t));

        if (!send_msg(coord_fd, MsgType::LOCAL_STATS, out.data(), static_cast<uint64_t>(payload_bytes))) {
            std::fprintf(stderr, "[WORKER %u] Failed to send LOCAL_STATS\n", cfg.worker_id);
            break;
        }
    }

    close_fd(coord_fd);
    std::fprintf(stderr, "[WORKER %u] Done\n", cfg.worker_id);
    return 0;
}
