#include "cache.h"

#include <algorithm>
#include <chrono>
#include <cstdio>
#include <cstring>
#include <numeric>
#include <random>
#include <string>
#include <thread>
#include <unistd.h>
#include <vector>

static const char* BENCH_PATH = "/tmp/flashring_shard_bench.dat";

using Clock = std::chrono::high_resolution_clock;
using ns_d  = std::chrono::duration<double, std::nano>;

static constexpr uint32_t NUM_KEYS  = 10'000;
static constexpr uint32_t MEASURE   = 50'000;

struct Stats { double avg_ns, p50_ns, p99_ns, mops; };

static Stats summarise(std::vector<double>& v) {
    std::sort(v.begin(), v.end());
    size_t n = v.size();
    double total = std::accumulate(v.begin(), v.end(), 0.0);
    double avg = total / static_cast<double>(n);
    return { avg, v[n/2], v[static_cast<size_t>(n * 0.99)], 1e3 / avg };
}

static void cleanup(uint32_t max_shards) {
    for (uint32_t i = 0; i < max_shards; ++i) {
        std::string path = std::string(BENCH_PATH) + "." + std::to_string(i);
        ::unlink(path.c_str());
    }
}

static std::vector<std::string> gen_keys(uint32_t count) {
    std::vector<std::string> out(count);
    for (uint32_t i = 0; i < count; ++i)
        out[i] = "key_" + std::to_string(i);
    return out;
}

static std::vector<uint8_t> gen_value(size_t sz) {
    std::vector<uint8_t> v(sz);
    for (size_t i = 0; i < sz; ++i)
        v[i] = static_cast<uint8_t>((i * 7 + 13) & 0xFF);
    return v;
}

static void run_bench(uint32_t num_shards, size_t val_sz) {
    cleanup(64);

    uint64_t ring_cap = static_cast<uint64_t>(NUM_KEYS) * (val_sz + 64) * 2;
    size_t mt_size = static_cast<size_t>(NUM_KEYS) * (val_sz + 64) + (1 << 20);

    auto cache = Cache::open({
        .device_path    = BENCH_PATH,
        .ring_capacity  = ring_cap,
        .memtable_size  = mt_size,
        .index_capacity = NUM_KEYS,
        .num_shards     = num_shards,
    });

    auto keys = gen_keys(NUM_KEYS);
    auto val  = gen_value(val_sz);

    // --- PUT benchmark ---
    {
        std::vector<double> lat(NUM_KEYS);
        for (uint32_t i = 0; i < NUM_KEYS; ++i) {
            auto t0 = Clock::now();
            cache.put(keys[i].data(), keys[i].size(), val.data(), val.size());
            auto t1 = Clock::now();
            lat[i] = std::chrono::duration_cast<ns_d>(t1 - t0).count();
        }
        auto s = summarise(lat);
        std::printf("  put          %9.0f %9.0f %9.0f %9.2f\n",
                    s.avg_ns, s.p50_ns, s.p99_ns, s.mops);
    }

    // --- GET (memory) benchmark ---
    {
        std::mt19937 rng(42);
        std::uniform_int_distribution<uint32_t> dist(0, NUM_KEYS - 1);
        std::vector<uint8_t> buf(val_sz);
        std::vector<double> lat(MEASURE);
        for (uint32_t i = 0; i < MEASURE; ++i) {
            uint32_t k = dist(rng);
            size_t actual = 0;
            auto t0 = Clock::now();
            cache.get(keys[k].data(), keys[k].size(), buf.data(), buf.size(), &actual);
            auto t1 = Clock::now();
            lat[i] = std::chrono::duration_cast<ns_d>(t1 - t0).count();
        }
        auto s = summarise(lat);
        std::printf("  get_mem      %9.0f %9.0f %9.0f %9.2f\n",
                    s.avg_ns, s.p50_ns, s.p99_ns, s.mops);
    }

    // --- GET (disk) benchmark ---
    cache.flush();
    {
        std::mt19937 rng(42);
        std::uniform_int_distribution<uint32_t> dist(0, NUM_KEYS - 1);
        std::vector<uint8_t> buf(val_sz);
        std::vector<double> lat(MEASURE);
        for (uint32_t i = 0; i < MEASURE; ++i) {
            uint32_t k = dist(rng);
            size_t actual = 0;
            auto t0 = Clock::now();
            cache.get(keys[k].data(), keys[k].size(), buf.data(), buf.size(), &actual);
            auto t1 = Clock::now();
            lat[i] = std::chrono::duration_cast<ns_d>(t1 - t0).count();
        }
        auto s = summarise(lat);
        std::printf("  get_disk     %9.0f %9.0f %9.0f %9.2f\n",
                    s.avg_ns, s.p50_ns, s.p99_ns, s.mops);
    }

    // --- MIXED (4 threads, 90% read / 10% write) ---
    {
        int num_threads = 4;
        uint32_t ops_per_thread = MEASURE;
        std::vector<std::vector<double>> all_lat(num_threads);
        std::vector<std::thread> threads;

        for (int t = 0; t < num_threads; ++t) {
            threads.emplace_back([&, t]() {
                std::mt19937 rng(100 + t);
                std::uniform_int_distribution<uint32_t> key_dist(0, NUM_KEYS - 1);
                std::uniform_int_distribution<int> op_dist(0, 9);
                auto& lat = all_lat[t];
                lat.resize(ops_per_thread);
                std::vector<uint8_t> rbuf(val_sz);

                for (uint32_t i = 0; i < ops_per_thread; ++i) {
                    uint32_t k = key_dist(rng);
                    bool is_write = (op_dist(rng) == 0);
                    auto t0 = Clock::now();
                    if (is_write) {
                        cache.put(keys[k].data(), keys[k].size(),
                                  val.data(), val.size());
                    } else {
                        size_t actual = 0;
                        cache.get(keys[k].data(), keys[k].size(),
                                  rbuf.data(), rbuf.size(), &actual);
                    }
                    auto t1 = Clock::now();
                    lat[i] = std::chrono::duration_cast<ns_d>(t1 - t0).count();
                }
            });
        }
        for (auto& th : threads) th.join();

        std::vector<double> merged;
        merged.reserve(static_cast<size_t>(num_threads) * ops_per_thread);
        for (auto& v : all_lat)
            merged.insert(merged.end(), v.begin(), v.end());
        auto s = summarise(merged);
        s.mops *= num_threads;
        std::printf("  mixed_4t     %9.0f %9.0f %9.0f %9.2f\n",
                    s.avg_ns, s.p50_ns, s.p99_ns, s.mops);
    }

    cleanup(64);
}

int main() {
    const size_t val_sizes[]    = {1024, 2048, 4096};
    const uint32_t shard_counts[] = {1, 2, 4, 8};

    std::printf("flashringc shard benchmark  (10K keys, 50K measure ops)\n\n");

    for (size_t val_sz : val_sizes) {
        for (uint32_t shards : shard_counts) {
            std::printf("── val_sz=%zuB  shards=%u ──\n", val_sz, shards);
            std::printf("  %-14s %9s %9s %9s %9s\n",
                        "operation", "avg(ns)", "p50(ns)", "p99(ns)", "Mops/s");
            std::printf("  %-14s %9s %9s %9s %9s\n",
                        "", "", "", "", "");
            run_bench(shards, val_sz);
            std::printf("\n");
        }
    }

    std::printf("done.\n");
    return 0;
}
