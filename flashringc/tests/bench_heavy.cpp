#include "flashringc/cache.h"

#include <algorithm>
#include <atomic>
#include <chrono>
#include <cstdio>
#include <cstring>
#include <mutex>
#include <random>
#include <string>
#include <thread>
#include <unistd.h>
#include <vector>

#ifdef __linux__
#include <pthread.h>
#endif

using Clock = std::chrono::high_resolution_clock;

static std::string g_bench_path = "/tmp/flashring_heavy.dat";

static constexpr size_t VAL_SIZE = 4096;

static void cleanup() {
    for (uint32_t i = 0; i < 256; ++i)
        ::unlink((g_bench_path + "." + std::to_string(i)).c_str());
}

static void pin_thread(int cpu) {
#ifdef __linux__
    cpu_set_t cpuset;
    CPU_ZERO(&cpuset);
    CPU_SET(cpu % static_cast<int>(std::thread::hardware_concurrency()), &cpuset);
    pthread_setaffinity_np(pthread_self(), sizeof(cpu_set_t), &cpuset);
#else
    (void)cpu;
#endif
}

struct Percentiles {
    double p50_us;
    double p90_us;
    double p99_us;
    double avg_us;
};

static Percentiles compute_percentiles(std::vector<double>& lats) {
    if (lats.empty()) return {0, 0, 0, 0};
    std::sort(lats.begin(), lats.end());
    size_t n = lats.size();
    double sum = 0;
    for (double l : lats) sum += l;
    return {
        lats[n / 2],
        lats[static_cast<size_t>(n * 0.90)],
        lats[std::min(static_cast<size_t>(n * 0.99), n - 1)],
        sum / static_cast<double>(n),
    };
}

struct HeavyResult {
    uint64_t total_ops;
    uint64_t put_ops;
    uint64_t get_ops;
    uint64_t get_hits;
    double   elapsed_s;
    Percentiles batch_lat;  // latency of 100-op batch (submit to last future)
};

static HeavyResult run_heavy_bench(uint32_t total_unique_keys,
                                   uint32_t num_shards,
                                   int num_producers,
                                   int ops_per_producer) {
    cleanup();

    uint64_t ring_cap = std::max<uint64_t>(
        512ULL * 1024 * 1024,
        static_cast<uint64_t>(total_unique_keys) * (VAL_SIZE + 128) * 2);

    size_t mt_size = std::min<size_t>(4ULL * 1024 * 1024, ring_cap / 32);

    CacheConfig cfg{
        .device_path       = g_bench_path,
        .ring_capacity     = ring_cap,
        .memtable_size     = mt_size,
        .index_capacity    = total_unique_keys * 2,
        .num_shards        = num_shards,
        .queue_capacity    = 16384,
        .uring_queue_depth = 512,
    };
    auto cache = Cache::open(cfg);

    std::string val(VAL_SIZE, 'V');

    for (uint32_t i = 0; i < total_unique_keys; ++i) {
        std::string key = "hk_" + std::to_string(i);
        cache.put_sync(key.data(), key.size(), val.data(), val.size());
    }

    std::atomic<uint64_t> total_puts{0};
    std::atomic<uint64_t> total_gets{0};
    std::atomic<uint64_t> total_hits{0};

    std::mutex lat_mu;
    std::vector<double> all_lats;

    auto t0 = Clock::now();

    std::vector<std::thread> workers;
    workers.reserve(num_producers);

    for (int w = 0; w < num_producers; ++w) {
        workers.emplace_back([&, w]() {
            pin_thread(static_cast<int>(num_shards) + w);

            std::mt19937 rng(static_cast<unsigned>(w * 31 + 7));
            std::uniform_int_distribution<uint32_t> key_dist(0, total_unique_keys - 1);

            uint64_t puts = 0, gets = 0, hits = 0;

            // Batch latencies: time from first submit to last future resolved.
            std::vector<double> local_batch_lats;
            int num_batches = ops_per_producer / 100;
            local_batch_lats.reserve(num_batches);

            for (int batch = 0; batch < num_batches; ++batch) {
                int batch_puts = 0, batch_gets = 0;

                auto t_batch_start = Clock::now();

                for (int j = 0; j < 100; ++j) {
                    uint32_t idx = key_dist(rng);
                    std::string key = "hk_" + std::to_string(idx);

                    if (j % 10 == 0) {
                        Result r = cache.put(key, std::string_view(val));
                        if (r.status == Status::Ok) ++hits;
                        ++batch_puts;
                    } else {
                        Result r = cache.get(key);
                        if (r.status == Status::Ok && !r.value.empty())
                            ++hits;
                        ++batch_gets;
                    }
                }

                auto t_batch_end = Clock::now();
                double batch_us = std::chrono::duration<double, std::micro>(
                    t_batch_end - t_batch_start).count();
                local_batch_lats.push_back(batch_us);

                puts += batch_puts;
                gets += batch_gets;
            }

            total_puts.fetch_add(puts, std::memory_order_relaxed);
            total_gets.fetch_add(gets, std::memory_order_relaxed);
            total_hits.fetch_add(hits, std::memory_order_relaxed);

            std::lock_guard<std::mutex> lk(lat_mu);
            all_lats.insert(all_lats.end(),
                            local_batch_lats.begin(), local_batch_lats.end());
        });
    }

    for (auto& t : workers) t.join();

    auto t1 = Clock::now();
    double elapsed = std::chrono::duration<double>(t1 - t0).count();

    cleanup();

    return {
        total_puts.load() + total_gets.load(),
        total_puts.load(), total_gets.load(), total_hits.load(),
        elapsed,
        compute_percentiles(all_lats),
    };
}

int main(int argc, char* argv[]) {
    if (argc >= 2)
        g_bench_path = argv[1];

    setbuf(stdout, nullptr);
    printf("bench_heavy (reactor-based, 4KB values)\n");
    printf("device_path: %s\n", g_bench_path.c_str());
    printf("hw_threads:  %u\n\n", std::thread::hardware_concurrency());

    static const int kProducerCounts[] = {4, 8, 16, 32, 64, 128};
    static const uint32_t kShardCounts[] = {4, 8, 16};

    static const uint32_t kTotalKeys = 50000;
    static const int kOpsPerProducer = 10000;

    printf("Batch = 100 ops (90%% GET / 10%% PUT). Latency = wall time per batch.\n\n");

    printf("%-6s %-5s %8s %8s %8s │ %10s %10s %10s %10s │ %7s %6s\n",
           "Shrd", "Thrd", "Ops", "PUTs", "GETs",
           "Bat avg", "Bat p50", "Bat p90", "Bat p99",
           "Mops/s", "Hit%");
    printf("%-6s %-5s %8s %8s %8s │ %10s %10s %10s %10s │ %7s %6s\n",
           "-----", "----", "------", "------", "------",
           "--------", "--------", "--------", "--------",
           "------", "----");

    for (uint32_t shards : kShardCounts) {
        for (int producers : kProducerCounts) {
            auto r = run_heavy_bench(kTotalKeys, shards,
                                     producers, kOpsPerProducer);
            double mops = static_cast<double>(r.total_ops) / r.elapsed_s / 1e6;
            double hit_rate = r.get_ops > 0
                ? 100.0 * static_cast<double>(r.get_hits)
                        / static_cast<double>(r.get_ops)
                : 0.0;

            auto& b = r.batch_lat;
            printf("%-6u %-5d %8lu %8lu %8lu │ %8.0fµs %8.0fµs %8.0fµs %8.0fµs │ %7.2f %5.1f%%\n",
                   shards, producers,
                   (unsigned long)r.total_ops,
                   (unsigned long)r.put_ops,
                   (unsigned long)r.get_ops,
                   b.avg_us, b.p50_us, b.p90_us, b.p99_us,
                   mops, hit_rate);
        }
        printf("\n");
    }

    printf("Done.\n");
    return 0;
}
