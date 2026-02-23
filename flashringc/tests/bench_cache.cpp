#include "flashringc/cache.h"

#include <algorithm>
#include <chrono>
#include <cstdio>
#include <cstring>
#include <string>
#include <thread>
#include <unistd.h>
#include <vector>

static std::string g_bench_path = "/tmp/flashring_bench_v2.dat";

static void cleanup() {
    for (uint32_t i = 0; i < 256; ++i)
        ::unlink((g_bench_path + "." + std::to_string(i)).c_str());
}

struct BenchResult {
    double avg_us;
    double p50_us;
    double p99_us;
    double mops;
};

static BenchResult compute_stats(std::vector<double>& latencies_us) {
    std::sort(latencies_us.begin(), latencies_us.end());
    size_t n = latencies_us.size();
    double sum = 0;
    for (auto l : latencies_us) sum += l;

    return {
        sum / static_cast<double>(n),
        latencies_us[n / 2],
        latencies_us[static_cast<size_t>(n * 0.99)],
        static_cast<double>(n) / sum  // Mops/s (latencies are in µs)
    };
}

static void bench_put(uint32_t num_keys, size_t val_size, uint32_t num_shards) {
    cleanup();
    CacheConfig cfg{
        .device_path      = g_bench_path,
        .ring_capacity    = 256ULL * 1024 * 1024,
        .memtable_size    = 16 * 1024 * 1024,
        .index_capacity   = num_keys * 2,
        .num_shards       = num_shards,
        .queue_capacity   = 4096,
        .uring_queue_depth = 256,
    };
    auto cache = Cache::open(cfg);

    std::string val(val_size, 'x');
    std::vector<double> latencies;
    latencies.reserve(num_keys);

    for (uint32_t i = 0; i < num_keys; ++i) {
        std::string key = "k_" + std::to_string(i);
        auto t0 = std::chrono::high_resolution_clock::now();
        cache.put_sync(key.data(), key.size(), val.data(), val.size());
        auto t1 = std::chrono::high_resolution_clock::now();
        latencies.push_back(
            std::chrono::duration<double, std::micro>(t1 - t0).count());
    }

    auto stats = compute_stats(latencies);
    printf("  PUT  %6u keys × %4zuB val, %u shards | "
           "avg=%.1fµs  p50=%.1fµs  p99=%.1fµs  %.2f Mops/s\n",
           num_keys, val_size, num_shards,
           stats.avg_us, stats.p50_us, stats.p99_us, stats.mops);
    cleanup();
}

static void bench_get_hot(uint32_t num_keys, size_t val_size,
                          uint32_t num_shards) {
    cleanup();
    CacheConfig cfg{
        .device_path      = g_bench_path,
        .ring_capacity    = 256ULL * 1024 * 1024,
        .memtable_size    = 64 * 1024 * 1024,
        .index_capacity   = num_keys * 2,
        .num_shards       = num_shards,
        .queue_capacity   = 4096,
        .uring_queue_depth = 256,
    };
    auto cache = Cache::open(cfg);

    std::string val(val_size, 'x');
    for (uint32_t i = 0; i < num_keys; ++i) {
        std::string key = "k_" + std::to_string(i);
        cache.put_sync(key.data(), key.size(), val.data(), val.size());
    }

    std::vector<double> latencies;
    latencies.reserve(num_keys);

    for (uint32_t i = 0; i < num_keys; ++i) {
        std::string key = "k_" + std::to_string(i);
        char buf[65536];
        size_t actual = 0;
        auto t0 = std::chrono::high_resolution_clock::now();
        cache.get_sync(key.data(), key.size(), buf, sizeof(buf), &actual);
        auto t1 = std::chrono::high_resolution_clock::now();
        latencies.push_back(
            std::chrono::duration<double, std::micro>(t1 - t0).count());
    }

    auto stats = compute_stats(latencies);
    printf("  GET  %6u keys × %4zuB val, %u shards | "
           "avg=%.1fµs  p50=%.1fµs  p99=%.1fµs  %.2f Mops/s\n",
           num_keys, val_size, num_shards,
           stats.avg_us, stats.p50_us, stats.p99_us, stats.mops);
    cleanup();
}

static void bench_mixed_mt(uint32_t num_keys, size_t val_size,
                           uint32_t num_shards, int num_threads) {
    cleanup();
    CacheConfig cfg{
        .device_path      = g_bench_path,
        .ring_capacity    = 256ULL * 1024 * 1024,
        .memtable_size    = 64 * 1024 * 1024,
        .index_capacity   = num_keys * 2,
        .num_shards       = num_shards,
        .queue_capacity   = 8192,
        .uring_queue_depth = 256,
    };
    auto cache = Cache::open(cfg);

    std::string val(val_size, 'x');
    // Pre-fill.
    for (uint32_t i = 0; i < num_keys; ++i) {
        std::string key = "k_" + std::to_string(i);
        cache.put_sync(key.data(), key.size(), val.data(), val.size());
    }

    std::atomic<uint64_t> total_ops{0};
    auto t0 = std::chrono::high_resolution_clock::now();

    std::vector<std::thread> workers;
    for (int w = 0; w < num_threads; ++w) {
        workers.emplace_back([&cache, num_keys, val_size, &total_ops, &val]() {
            uint64_t ops = 0;
            for (uint32_t i = 0; i < 10000; ++i) {
                uint32_t idx = i % num_keys;
                std::string key = "k_" + std::to_string(idx);
                if (i % 10 == 0) {
                    cache.put_sync(key.data(), key.size(),
                                   val.data(), val.size());
                } else {
                    char buf[65536];
                    size_t actual = 0;
                    cache.get_sync(key.data(), key.size(),
                                   buf, sizeof(buf), &actual);
                }
                ++ops;
            }
            total_ops.fetch_add(ops, std::memory_order_relaxed);
        });
    }

    for (auto& t : workers) t.join();

    auto t1 = std::chrono::high_resolution_clock::now();
    double elapsed_s = std::chrono::duration<double>(t1 - t0).count();
    double mops = static_cast<double>(total_ops.load()) / elapsed_s / 1e6;

    printf("  MIXED  %u keys × %zuB, %u shards, %d threads | "
           "%.2f Mops/s  (%.1fs)\n",
           num_keys, val_size, num_shards, num_threads,
           mops, elapsed_s);
    cleanup();
}

int main(int argc, char* argv[]) {
    if (argc >= 2)
        g_bench_path = argv[1];

    printf("bench_cache (reactor-based)\n");
    printf("device_path: %s\n\n", g_bench_path.c_str());

    printf("--- PUT throughput ---\n");
    bench_put(10000, 128, 4);
    bench_put(10000, 1024, 4);
    bench_put(10000, 4096, 4);

    printf("\n--- GET (memtable hit) ---\n");
    bench_get_hot(10000, 128, 4);
    bench_get_hot(10000, 1024, 4);

    printf("\n--- Mixed 90/10 read/write (multi-threaded) ---\n");
    bench_mixed_mt(10000, 128, 4, 4);
    bench_mixed_mt(10000, 128, 4, 8);
    bench_mixed_mt(10000, 1024, 4, 8);

    printf("\nDone.\n");
    return 0;
}
