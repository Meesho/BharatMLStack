#include "flashringc/cache.h"

#include <algorithm>
#include <atomic>
#include <chrono>
#include <cstdio>
#include <cstring>
#include <random>
#include <string>
#include <thread>
#include <unistd.h>
#include <vector>

#ifdef __linux__
#include <pthread.h>
#endif

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

struct HeavyResult {
    uint64_t total_ops;
    uint64_t put_ops;
    uint64_t get_ops;
    uint64_t get_hits;
    double   elapsed_s;
};

// Each producer thread: PUTs a batch of unique keys, then GETs them back.
// Keys are large enough (with 4KB values) to overflow the memtable,
// forcing disk reads on the GET path.
static HeavyResult run_heavy_bench(uint32_t total_unique_keys,
                                   uint32_t num_shards,
                                   int num_producers,
                                   int ops_per_producer) {
    cleanup();

    uint64_t ring_cap = std::max<uint64_t>(
        512ULL * 1024 * 1024,
        static_cast<uint64_t>(total_unique_keys) * (VAL_SIZE + 128) * 2);

    // Small memtable to force data to disk quickly.
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

    // Phase 1: pre-fill all keys so GETs have data.
    for (uint32_t i = 0; i < total_unique_keys; ++i) {
        std::string key = "hk_" + std::to_string(i);
        cache.put_sync(key.data(), key.size(), val.data(), val.size());
    }

    // Phase 2: multi-threaded mixed workload.
    std::atomic<uint64_t> total_puts{0};
    std::atomic<uint64_t> total_gets{0};
    std::atomic<uint64_t> total_hits{0};

    auto t0 = std::chrono::high_resolution_clock::now();

    std::vector<std::thread> workers;
    workers.reserve(num_producers);

    for (int w = 0; w < num_producers; ++w) {
        workers.emplace_back([&, w]() {
            pin_thread(static_cast<int>(num_shards) + w);

            std::mt19937 rng(static_cast<unsigned>(w * 31 + 7));
            std::uniform_int_distribution<uint32_t> key_dist(0, total_unique_keys - 1);

            uint64_t puts = 0, gets = 0, hits = 0;
            char read_buf[VAL_SIZE + 256];

            for (int batch = 0; batch < ops_per_producer / 100; ++batch) {
                // Fire 100 async ops per batch.
                std::vector<std::future<Result>> futs;
                futs.reserve(100);

                for (int j = 0; j < 100; ++j) {
                    uint32_t idx = key_dist(rng);
                    std::string key = "hk_" + std::to_string(idx);

                    if (j % 10 == 0) {
                        futs.push_back(cache.put(key, std::string_view(val)));
                        ++puts;
                    } else {
                        futs.push_back(cache.get(key));
                        ++gets;
                    }
                }

                for (auto& f : futs) {
                    Result r = f.get();
                    if (r.status == Status::Ok && !r.value.empty())
                        ++hits;
                }
            }

            total_puts.fetch_add(puts, std::memory_order_relaxed);
            total_gets.fetch_add(gets, std::memory_order_relaxed);
            total_hits.fetch_add(hits, std::memory_order_relaxed);
        });
    }

    for (auto& t : workers) t.join();

    auto t1 = std::chrono::high_resolution_clock::now();
    double elapsed = std::chrono::duration<double>(t1 - t0).count();

    uint64_t ops = total_puts.load() + total_gets.load();
    cleanup();

    return {ops, total_puts.load(), total_gets.load(),
            total_hits.load(), elapsed};
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

    // Large key counts to overflow memtable (4KB vals, small memtable).
    // 50K keys × 4KB ≈ 200MB — well beyond the 4MB memtable.
    static const uint32_t kTotalKeys = 50000;
    static const int kOpsPerProducer = 10000;

    printf("%-8s  %-8s  %-10s  %12s  %12s  %12s  %10s  %10s  %8s\n",
           "Shards", "Threads", "TotalOps", "PUTs", "GETs",
           "GET Hits", "Mops/s", "Elapsed", "HitRate");
    printf("%-8s  %-8s  %-10s  %12s  %12s  %12s  %10s  %10s  %8s\n",
           "------", "-------", "--------", "----", "----",
           "--------", "------", "-------", "-------");

    for (uint32_t shards : kShardCounts) {
        for (int producers : kProducerCounts) {
            auto r = run_heavy_bench(kTotalKeys, shards,
                                     producers, kOpsPerProducer);
            double mops = static_cast<double>(r.total_ops) / r.elapsed_s / 1e6;
            double hit_rate = r.get_ops > 0
                ? 100.0 * static_cast<double>(r.get_hits)
                        / static_cast<double>(r.get_ops)
                : 0.0;

            printf("%-8u  %-8d  %-10lu  %12lu  %12lu  %12lu  %10.2f  %8.2fs  %7.1f%%\n",
                   shards, producers,
                   static_cast<unsigned long>(r.total_ops),
                   static_cast<unsigned long>(r.put_ops),
                   static_cast<unsigned long>(r.get_ops),
                   static_cast<unsigned long>(r.get_hits),
                   mops, r.elapsed_s, hit_rate);
        }
        printf("\n");
    }

    printf("Done.\n");
    return 0;
}
