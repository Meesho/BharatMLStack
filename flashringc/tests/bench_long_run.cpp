#include "flashringc/cache.h"

#include <algorithm>
#include <atomic>
#include <chrono>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <mutex>
#include <random>
#include <string>
#include <thread>
#include <unistd.h>
#include <vector>

using Clock = std::chrono::high_resolution_clock;

// Fixed working set: we only ever touch this many distinct keys.
// With index_capacity >= num_keys, most keys stay cached → meaningful hit rate.
static constexpr uint64_t kNumKeys       = 1'000'000;
static constexpr double   kDurationSec   = 300.0;   // 5 min
static constexpr double   kReportIntervalSec = 15.0;
static constexpr size_t   kBatchSize     = 100;
static constexpr size_t   kValSize       = 256;

static std::string g_bench_path = "/tmp/flashring_longrun.dat";
static double g_duration_sec = kDurationSec;
static double g_report_interval_sec = kReportIntervalSec;
static uint64_t g_num_keys = kNumKeys;

static void cleanup() {
    for (uint32_t i = 0; i < 256; ++i)
        ::unlink((g_bench_path + "." + std::to_string(i)).c_str());
}

static void compute_percentiles(std::vector<double>& lats,
                                double* avg, double* p90, double* p99) {
    if (lats.empty()) {
        *avg = *p90 = *p99 = 0;
        return;
    }
    std::sort(lats.begin(), lats.end());
    size_t n = lats.size();
    double sum = 0;
    for (double x : lats) sum += x;
    *avg = sum / static_cast<double>(n);
    *p90 = lats[static_cast<size_t>(static_cast<double>(n) * 0.90)];
    *p99 = lats[std::min(static_cast<size_t>(static_cast<double>(n) * 0.99), n - 1)];
}

int main(int argc, char* argv[]) {
    if (argc >= 2)
        g_bench_path = argv[1];
    if (argc >= 3)
        g_duration_sec = std::max(1.0, atof(argv[2]));
    if (argc >= 4)
        g_report_interval_sec = std::max(1.0, atof(argv[3]));
    if (argc >= 5)
        g_num_keys = std::max(1ULL, static_cast<uint64_t>(atoll(argv[4])));

    cleanup();

    // Fixed working set of g_num_keys. Size ring so working set stays below 75% watermark
    // (records are 4KB-aligned in memtable, so ~4KB per key when flushed).
    const uint64_t working_set_bytes = g_num_keys * 4096ULL;
    uint64_t ring_cap = (working_set_bytes * 4ULL) / 3ULL;  // working_set / 0.75
    if (ring_cap < 512ULL * 1024 * 1024) ring_cap = 512ULL * 1024 * 1024;
    if (ring_cap > 8ULL * 1024 * 1024 * 1024) ring_cap = 8ULL * 1024 * 1024 * 1024;
    size_t mt_size    = 64ULL * 1024 * 1024;
    uint32_t index_cap = static_cast<uint32_t>(std::min(g_num_keys * 2ULL, 4'000'000ULL));

    CacheConfig cfg{
        .device_path    = g_bench_path,
        .ring_capacity  = ring_cap,
        .memtable_size  = mt_size,
        .index_capacity = index_cap,
        .num_shards     = 0,
        .queue_capacity = 16384,
        .uring_queue_depth = 512,
        .sem_pool_capacity = 4096,
    };

    printf("bench_long_run: get-batch, put-if-miss, fixed set of %lu keys, %.0fs, report every %.0fs\n",
           (unsigned long)g_num_keys, g_duration_sec, g_report_interval_sec);
    printf("  batch_size=%zu, val_size=%zu, path=%s\n\n",
           kBatchSize, kValSize, g_bench_path.c_str());

    auto cache = Cache::open(cfg);

    std::string val(kValSize, 'V');

    std::atomic<uint64_t> total_gets{0};
    std::atomic<uint64_t> total_puts{0};
    std::atomic<uint64_t> total_hits{0};
    std::atomic<uint64_t> total_batch_ops{0};

    std::mutex lat_mu;
    std::vector<double> window_latencies;

    std::atomic<bool> stop{false};
    auto deadline = Clock::now() + std::chrono::duration<double>(g_duration_sec);

    // Reporter thread: every Ns print stats
    std::thread reporter([&]() {
        uint64_t prev_gets = 0, prev_puts = 0, prev_batch_ops = 0;
        uint64_t prev_wrap = 0;
        int interval_num = 0;

        while (!stop.load(std::memory_order_relaxed)) {
            std::this_thread::sleep_for(
                std::chrono::milliseconds(static_cast<int>(g_report_interval_sec * 1000)));

            if (stop.load(std::memory_order_relaxed)) break;

            uint64_t gets = total_gets.load(std::memory_order_relaxed);
            uint64_t puts = total_puts.load(std::memory_order_relaxed);
            uint64_t batch_ops = total_batch_ops.load(std::memory_order_relaxed);
            uint64_t hits = total_hits.load(std::memory_order_relaxed);
            uint64_t wrap = cache.ring_wrap_count();

            uint64_t d_gets = gets - prev_gets;
            uint64_t d_puts = puts - prev_puts;
            uint64_t d_batch = batch_ops - prev_batch_ops;
            uint64_t d_wrap = wrap - prev_wrap;

            double hit_rate = gets > 0 ? 100.0 * static_cast<double>(hits) / static_cast<double>(gets) : 0.0;

            double avg_us = 0, p90_us = 0, p99_us = 0;
            {
                std::lock_guard<std::mutex> lock(lat_mu);
                compute_percentiles(window_latencies, &avg_us, &p90_us, &p99_us);
                window_latencies.clear();
            }

            double elapsed = std::chrono::duration<double>(Clock::now() - (deadline - std::chrono::duration<double>(g_duration_sec))).count();
            ++interval_num;

            printf("[%3.0fs] gets=%lu puts=%lu batch_ops=%lu | "
                   "lat_avg=%.1fus p90=%.1fus p99=%.1fus | "
                   "hit_rate=%.1f%% wrap_count=%lu (+%lu)\n",
                   elapsed,
                   (unsigned long)gets, (unsigned long)puts, (unsigned long)batch_ops,
                   avg_us, p90_us, p99_us,
                   hit_rate, (unsigned long)wrap, (unsigned long)d_wrap);

            prev_gets = gets;
            prev_puts = puts;
            prev_batch_ops = batch_ops;
            prev_wrap = wrap;
        }
    });

    // Worker: repeatedly do batch get, then batch put for misses.
    // Keys drawn uniformly from [0, g_num_keys) — fixed working set.
    std::mt19937 rng(12345);
    std::uniform_int_distribution<uint64_t> key_dist(0, g_num_keys - 1);

    std::vector<std::string> key_storage(kBatchSize);
    std::vector<std::string_view> keys(kBatchSize);
    std::vector<KVPair> put_pairs;
    put_pairs.reserve(kBatchSize);

    while (Clock::now() < deadline) {
        for (size_t i = 0; i < kBatchSize; ++i) {
            uint64_t k = key_dist(rng);
            key_storage[i] = "lk_" + std::to_string(k);
            keys[i] = key_storage[i];
        }

        auto t0 = Clock::now();
        std::vector<Result> results = cache.batch_get(keys);
        uint64_t hits_this_batch = 0;
        put_pairs.clear();
        for (size_t i = 0; i < results.size(); ++i) {
            if (results[i].status == Status::Ok)
                ++hits_this_batch;
            else if (results[i].status == Status::NotFound)
                put_pairs.push_back(KVPair{keys[i], std::string_view(val)});
        }
        if (!put_pairs.empty())
            cache.batch_put(put_pairs);
        auto t1 = Clock::now();

        double us = std::chrono::duration<double, std::micro>(t1 - t0).count();
        {
            std::lock_guard<std::mutex> lock(lat_mu);
            window_latencies.push_back(us);
        }

        total_gets.fetch_add(static_cast<uint64_t>(kBatchSize), std::memory_order_relaxed);
        total_puts.fetch_add(static_cast<uint64_t>(put_pairs.size()), std::memory_order_relaxed);
        total_hits.fetch_add(hits_this_batch, std::memory_order_relaxed);
        total_batch_ops.fetch_add(1, std::memory_order_relaxed);
    }

    stop.store(true, std::memory_order_release);
    reporter.join();

    // Final totals
    uint64_t gets = total_gets.load();
    uint64_t puts = total_puts.load();
    uint64_t batch_ops = total_batch_ops.load();
    uint64_t hits = total_hits.load();
    double hit_rate = gets > 0 ? 100.0 * static_cast<double>(hits) / static_cast<double>(gets) : 0.0;

    printf("\n--- Final ---\n");
    printf("  gets=%lu puts=%lu batch_ops=%lu hit_rate=%.2f%% wrap_count=%lu\n",
           (unsigned long)gets, (unsigned long)puts, (unsigned long)batch_ops,
           hit_rate, (unsigned long)cache.ring_wrap_count());

    cleanup();
    return 0;
}
