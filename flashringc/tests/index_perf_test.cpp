#include "key_index.h"

#include <algorithm>
#include <chrono>
#include <cstdint>
#include <cstdio>
#include <cstring>
#include <numeric>
#include <random>
#include <shared_mutex>
#include <string>
#include <thread>
#include <vector>

using Clock = std::chrono::high_resolution_clock;
using ns_d  = std::chrono::duration<double, std::nano>;
using us_d  = std::chrono::duration<double, std::micro>;

struct Stats {
    double avg_ns, p50_ns, p99_ns, mops;
};

static Stats summarise(std::vector<double>& v) {
    std::sort(v.begin(), v.end());
    size_t n = v.size();
    double total = std::accumulate(v.begin(), v.end(), 0.0);
    double avg   = total / static_cast<double>(n);
    return {
        avg,
        v[n / 2],
        v[std::min(n - 1, static_cast<size_t>(n * 0.99))],
        1e3 / avg,  // Mops = 1e9 ns/s  /  avg_ns  /  1e6  =  1e3 / avg_ns
    };
}

// Max iterations for per-op latency measurement (keeps runtime bounded).
static constexpr uint32_t kMaxMeasure = 500'000;

static uint32_t measure_count(uint32_t n) { return std::min(n, kMaxMeasure); }

// Pre-generate hashes so hashing cost isn't measured in the hot loop.
static std::vector<Hash128> gen_hashes(uint32_t count) {
    std::vector<Hash128> out(count);
    for (uint32_t i = 0; i < count; ++i) {
        out[i] = hash_key(&i, sizeof(i));
    }
    return out;
}

static void print_header() {
    std::printf(
        "%-14s %10s %8s | %9s %9s %9s %9s\n",
        "benchmark", "n", "threads",
        "avg(ns)", "p50(ns)", "p99(ns)", "Mops/s");
    std::printf(
        "%-14s %10s %8s | %9s %9s %9s %9s\n",
        "", "", "", "", "", "", "");
    std::printf(
        "-------------------------------------------"
        "+----------------------------------------\n");
}

static void print_row(const char* label, uint32_t n, int threads, Stats& s) {
    std::printf(
        "%-14s %10u %8d | %9.1f %9.1f %9.1f %9.2f\n",
        label, n, threads, s.avg_ns, s.p50_ns, s.p99_ns, s.mops);
}

// ─── Benchmarks ──────────────────────────────────────────────────────────────

static void bench_put(uint32_t n) {
    auto hashes = gen_hashes(n);
    KeyIndex idx(n);
    uint32_t m = measure_count(n);

    std::vector<double> lat(m);
    for (uint32_t i = 0; i < m; ++i) {
        auto t0 = Clock::now();
        idx.put(hashes[i], i, i, static_cast<uint16_t>(i & 0xFFFF));
        auto t1 = Clock::now();
        lat[i] = std::chrono::duration_cast<ns_d>(t1 - t0).count();
    }
    auto s = summarise(lat);
    print_row("put", n, 1, s);
}

static void bench_get_hit(uint32_t n) {
    auto hashes = gen_hashes(n);
    KeyIndex idx(n);
    for (uint32_t i = 0; i < n; ++i)
        idx.put(hashes[i], i, i, static_cast<uint16_t>(i & 0xFFFF));

    uint32_t m = measure_count(n);
    std::vector<uint32_t> order(m);
    std::mt19937 rng(42);
    std::uniform_int_distribution<uint32_t> dist(0, n - 1);
    for (uint32_t i = 0; i < m; ++i) order[i] = dist(rng);

    LookupResult lr{};
    std::vector<double> lat(m);
    for (uint32_t i = 0; i < m; ++i) {
        auto t0 = Clock::now();
        idx.get(hashes[order[i]], static_cast<uint16_t>(i), lr);
        auto t1 = Clock::now();
        lat[i] = std::chrono::duration_cast<ns_d>(t1 - t0).count();
    }
    auto s = summarise(lat);
    print_row("get_hit", n, 1, s);
}

static void bench_get_miss(uint32_t n) {
    auto hashes = gen_hashes(n);
    KeyIndex idx(n);
    for (uint32_t i = 0; i < n; ++i)
        idx.put(hashes[i], i, i, static_cast<uint16_t>(i & 0xFFFF));

    uint32_t m = measure_count(n);
    std::vector<Hash128> miss_hashes(m);
    for (uint32_t i = 0; i < m; ++i) {
        uint32_t key = n + i;
        miss_hashes[i] = hash_key(&key, sizeof(key));
    }

    LookupResult lr{};
    std::vector<double> lat(m);
    for (uint32_t i = 0; i < m; ++i) {
        auto t0 = Clock::now();
        idx.get(miss_hashes[i], 0, lr);
        auto t1 = Clock::now();
        lat[i] = std::chrono::duration_cast<ns_d>(t1 - t0).count();
    }
    auto s = summarise(lat);
    print_row("get_miss", n, 1, s);
}

static void bench_remove(uint32_t n) {
    auto hashes = gen_hashes(n);
    KeyIndex idx(n);
    for (uint32_t i = 0; i < n; ++i)
        idx.put(hashes[i], i, i, static_cast<uint16_t>(i & 0xFFFF));

    uint32_t m = measure_count(n);
    std::vector<uint32_t> order(m);
    std::mt19937 rng(99);
    std::uniform_int_distribution<uint32_t> dist(0, n - 1);
    for (uint32_t i = 0; i < m; ++i) order[i] = dist(rng);

    std::vector<double> lat(m);
    for (uint32_t i = 0; i < m; ++i) {
        auto t0 = Clock::now();
        idx.remove(hashes[order[i]]);
        auto t1 = Clock::now();
        lat[i] = std::chrono::duration_cast<ns_d>(t1 - t0).count();
    }
    auto s = summarise(lat);
    print_row("remove", n, 1, s);
}

static void bench_evict(uint32_t n) {
    auto hashes = gen_hashes(n);
    KeyIndex idx(n);
    for (uint32_t i = 0; i < n; ++i)
        idx.put(hashes[i], i, i, static_cast<uint16_t>(i & 0xFFFF));

    constexpr uint32_t kBatch = 1000;
    uint32_t batches = n / kBatch;
    if (batches == 0) batches = 1;

    std::vector<double> lat(batches);
    for (uint32_t b = 0; b < batches; ++b) {
        uint32_t count = std::min(kBatch, idx.size());
        if (count == 0) break;
        auto t0 = Clock::now();
        idx.evict_oldest(count);
        auto t1 = Clock::now();
        lat[b] = std::chrono::duration_cast<ns_d>(t1 - t0).count()
                 / static_cast<double>(count);
    }
    lat.resize(batches);
    auto s = summarise(lat);
    print_row("evict(1k)", n, 1, s);
}

static void bench_put_update(uint32_t n) {
    auto hashes = gen_hashes(n);
    KeyIndex idx(n);
    for (uint32_t i = 0; i < n; ++i)
        idx.put(hashes[i], i, i, static_cast<uint16_t>(i & 0xFFFF));

    uint32_t m = measure_count(n);
    std::mt19937 rng(77);
    std::uniform_int_distribution<uint32_t> dist(0, n - 1);

    std::vector<double> lat(m);
    for (uint32_t i = 0; i < m; ++i) {
        uint32_t k = dist(rng);
        auto t0 = Clock::now();
        idx.put(hashes[k], i + n, i, static_cast<uint16_t>(i & 0xFFFF));
        auto t1 = Clock::now();
        lat[i] = std::chrono::duration_cast<ns_d>(t1 - t0).count();
    }
    auto s = summarise(lat);
    print_row("put_update", n, 1, s);
}

// Concurrent get: multiple threads reading from a shared index with shared_mutex.
static void bench_concurrent_get(uint32_t n, int num_threads) {
    auto hashes = gen_hashes(n);
    KeyIndex idx(n);
    for (uint32_t i = 0; i < n; ++i)
        idx.put(hashes[i], i, i, static_cast<uint16_t>(i & 0xFFFF));

    std::shared_mutex mu;
    const uint32_t ops_per_thread = measure_count(n);

    std::vector<std::vector<double>> all_lat(num_threads);
    std::vector<std::thread> threads;

    for (int t = 0; t < num_threads; ++t) {
        threads.emplace_back([&, t]() {
            std::mt19937 rng(42 + t);
            std::uniform_int_distribution<uint32_t> dist(0, n - 1);
            auto& lat = all_lat[t];
            lat.resize(ops_per_thread);
            LookupResult lr{};

            for (uint32_t i = 0; i < ops_per_thread; ++i) {
                uint32_t k = dist(rng);
                auto t0 = Clock::now();
                {
                    std::shared_lock lk(mu);
                    idx.get(hashes[k], static_cast<uint16_t>(i), lr);
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
    print_row("get_conc", n, num_threads, s);
}

// Mixed: 90% get / 10% put with exclusive lock.
static void bench_mixed_rw(uint32_t n, int num_threads) {
    auto hashes = gen_hashes(n);
    KeyIndex idx(n);
    for (uint32_t i = 0; i < n; ++i)
        idx.put(hashes[i], i, i, static_cast<uint16_t>(i & 0xFFFF));

    std::shared_mutex mu;
    const uint32_t ops_per_thread = measure_count(n);

    std::vector<std::vector<double>> all_lat(num_threads);
    std::vector<std::thread> threads;

    for (int t = 0; t < num_threads; ++t) {
        threads.emplace_back([&, t]() {
            std::mt19937 rng(100 + t);
            std::uniform_int_distribution<uint32_t> key_dist(0, n - 1);
            std::uniform_int_distribution<int> op_dist(0, 9);
            auto& lat = all_lat[t];
            lat.resize(ops_per_thread);
            LookupResult lr{};

            for (uint32_t i = 0; i < ops_per_thread; ++i) {
                uint32_t k = key_dist(rng);
                bool is_write = (op_dist(rng) == 0);
                auto t0 = Clock::now();
                if (is_write) {
                    std::unique_lock lk(mu);
                    idx.put(hashes[k], i, i, static_cast<uint16_t>(i & 0xFFFF));
                } else {
                    std::shared_lock lk(mu);
                    idx.get(hashes[k], static_cast<uint16_t>(i), lr);
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
    print_row("mixed_90r", n, num_threads, s);
}

// ─── hash_key throughput ─────────────────────────────────────────────────────

static void bench_hash(uint32_t n) {
    uint32_t m = measure_count(n);
    std::vector<uint32_t> keys(m);
    std::iota(keys.begin(), keys.end(), 0);

    std::vector<double> lat(m);
    for (uint32_t i = 0; i < m; ++i) {
        auto t0 = Clock::now();
        auto h = hash_key(&keys[i], sizeof(keys[i]));
        auto t1 = Clock::now();
        lat[i] = std::chrono::duration_cast<ns_d>(t1 - t0).count();
        (void)h;
    }
    auto s = summarise(lat);
    print_row("xxh3_128", n, 1, s);
}

// ─── main ────────────────────────────────────────────────────────────────────

int main() {
    std::printf("flashringc index perf benchmark\n");
    std::printf("Entry size: %zu bytes\n\n", sizeof(Entry));

    const uint32_t sizes[] = {100'000, 1'000'000, 10'000'000};

    for (uint32_t n : sizes) {
        std::printf("── n = %u ──\n", n);
        print_header();

        bench_hash(n);
        bench_put(n);
        bench_put_update(n);
        bench_get_hit(n);
        bench_get_miss(n);
        bench_remove(n);
        bench_evict(n);

        int hw = static_cast<int>(std::thread::hardware_concurrency());
        if (hw < 1) hw = 4;

        for (int t : {1, 2, 4, hw}) {
            bench_concurrent_get(n, t);
        }
        for (int t : {1, 2, 4, hw}) {
            bench_mixed_rw(n, t);
        }

        std::printf("\n");
    }

    std::printf("done.\n");
    return 0;
}
