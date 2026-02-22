#include "cache.h"

#include <algorithm>
#include <chrono>
#include <cstdint>
#include <cstdio>
#include <cstring>
#include <mutex>
#include <numeric>
#include <random>
#include <shared_mutex>
#include <string>
#include <thread>
#include <sys/stat.h>
#include <unistd.h>
#include <vector>

static const char* DEFAULT_PATH = "/tmp/flashring_cache_perf.dat";
static const char* g_path       = DEFAULT_PATH;
static bool        g_is_blk     = false;

static bool is_block_device(const char* path) {
    struct stat st{};
    return (::stat(path, &st) == 0 && S_ISBLK(st.st_mode));
}

using Clock = std::chrono::high_resolution_clock;
using ns_d  = std::chrono::duration<double, std::nano>;
using us_d  = std::chrono::duration<double, std::micro>;

static constexpr uint32_t kMaxMeasure = 200'000;

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
        1e3 / avg,
    };
}

static void print_header() {
    std::printf(
        "%-18s %8s %6s %6s | %9s %9s %9s %9s\n",
        "benchmark", "keys", "valSz", "thr",
        "avg(ns)", "p50(ns)", "p99(ns)", "Mops/s");
    std::printf(
        "%-18s %8s %6s %6s | %9s %9s %9s %9s\n",
        "", "", "(B)", "", "", "", "", "");
    std::printf(
        "--------------------------------------------"
        "+----------------------------------------\n");
}

static void print_row(const char* label, uint32_t n, size_t val_sz,
                      int threads, Stats& s) {
    std::printf(
        "%-18s %8u %6zu %6d | %9.1f %9.1f %9.1f %9.2f\n",
        label, n, val_sz, threads,
        s.avg_ns, s.p50_ns, s.p99_ns, s.mops);
}

// Pre-generate keys as strings.
static std::vector<std::string> gen_keys(uint32_t count) {
    std::vector<std::string> out(count);
    for (uint32_t i = 0; i < count; ++i)
        out[i] = "key_" + std::to_string(i);
    return out;
}

// Generate a deterministic value of the given size.
static std::vector<uint8_t> gen_value(size_t sz) {
    std::vector<uint8_t> v(sz);
    for (size_t i = 0; i < sz; ++i)
        v[i] = static_cast<uint8_t>((i * 7 + 13) & 0xFF);
    return v;
}

// ─── Benchmarks ──────────────────────────────────────────────────────────────

static constexpr uint32_t PERF_SHARDS = 4;

static void cleanup() {
    if (g_is_blk) return;
    for (uint32_t i = 0; i < 256; ++i) {
        std::string path = std::string(g_path) + "." + std::to_string(i);
        ::unlink(path.c_str());
    }
}

static void bench_put(uint32_t n, size_t val_sz) {
    cleanup();
    auto cache = Cache::open({
        .device_path    = g_path,
        .ring_capacity  = g_is_blk ? 0 : static_cast<uint64_t>(n) * (val_sz + 64) * 2,
        .memtable_size  = 4 << 20,
        .index_capacity = n,
        .num_shards     = PERF_SHARDS,
    });

    auto keys = gen_keys(n);
    auto val  = gen_value(val_sz);
    uint32_t m = std::min(n, kMaxMeasure);

    std::vector<double> lat(m);
    for (uint32_t i = 0; i < m; ++i) {
        auto t0 = Clock::now();
        cache.put(keys[i].data(), keys[i].size(), val.data(), val.size());
        auto t1 = Clock::now();
        lat[i] = std::chrono::duration_cast<ns_d>(t1 - t0).count();
    }
    auto s = summarise(lat);
    print_row("put", n, val_sz, 1, s);
    cleanup();
}

static void bench_get_mem(uint32_t n, size_t val_sz) {
    cleanup();
    auto cache = Cache::open({
        .device_path    = g_path,
        .ring_capacity  = g_is_blk ? 0 : static_cast<uint64_t>(n) * (val_sz + 64) * 2,
        .memtable_size  = static_cast<size_t>(n) * (val_sz + 64) + (1 << 20),
        .index_capacity = n,
        .num_shards     = PERF_SHARDS,
    });

    auto keys = gen_keys(n);
    auto val  = gen_value(val_sz);
    for (uint32_t i = 0; i < n; ++i)
        cache.put(keys[i].data(), keys[i].size(), val.data(), val.size());

    uint32_t m = std::min(n, kMaxMeasure);
    std::mt19937 rng(42);
    std::uniform_int_distribution<uint32_t> dist(0, n - 1);

    std::vector<uint8_t> buf(val_sz);
    std::vector<double> lat(m);
    for (uint32_t i = 0; i < m; ++i) {
        uint32_t k = dist(rng);
        size_t actual = 0;
        auto t0 = Clock::now();
        cache.get(keys[k].data(), keys[k].size(), buf.data(), buf.size(), &actual);
        auto t1 = Clock::now();
        lat[i] = std::chrono::duration_cast<ns_d>(t1 - t0).count();
    }
    auto s = summarise(lat);
    print_row("get_mem", n, val_sz, 1, s);
    cleanup();
}

static void bench_get_disk(uint32_t n, size_t val_sz) {
    cleanup();
    auto cache = Cache::open({
        .device_path    = g_path,
        .ring_capacity  = g_is_blk ? 0 : static_cast<uint64_t>(n) * (val_sz + 64) * 2,
        .memtable_size  = 256 * 1024,
        .index_capacity = n,
        .num_shards     = PERF_SHARDS,
    });

    auto keys = gen_keys(n);
    auto val  = gen_value(val_sz);
    for (uint32_t i = 0; i < n; ++i)
        cache.put(keys[i].data(), keys[i].size(), val.data(), val.size());
    cache.flush();

    uint32_t m = std::min(n, kMaxMeasure);
    std::mt19937 rng(42);
    std::uniform_int_distribution<uint32_t> dist(0, n - 1);

    std::vector<uint8_t> buf(val_sz);
    std::vector<double> lat(m);
    for (uint32_t i = 0; i < m; ++i) {
        uint32_t k = dist(rng);
        size_t actual = 0;
        auto t0 = Clock::now();
        cache.get(keys[k].data(), keys[k].size(), buf.data(), buf.size(), &actual);
        auto t1 = Clock::now();
        lat[i] = std::chrono::duration_cast<ns_d>(t1 - t0).count();
    }
    auto s = summarise(lat);
    print_row("get_disk", n, val_sz, 1, s);
    cleanup();
}

static void bench_get_miss(uint32_t n, size_t val_sz) {
    cleanup();
    auto cache = Cache::open({
        .device_path    = g_path,
        .ring_capacity  = g_is_blk ? 0 : static_cast<uint64_t>(16 << 20),
        .memtable_size  = 4 << 20,
        .index_capacity = n,
        .num_shards     = PERF_SHARDS,
    });

    uint32_t m = std::min(n, kMaxMeasure);
    auto keys = gen_keys(m);
    std::vector<uint8_t> buf(val_sz);
    std::vector<double> lat(m);
    for (uint32_t i = 0; i < m; ++i) {
        size_t actual = 0;
        auto t0 = Clock::now();
        cache.get(keys[i].data(), keys[i].size(), buf.data(), buf.size(), &actual);
        auto t1 = Clock::now();
        lat[i] = std::chrono::duration_cast<ns_d>(t1 - t0).count();
    }
    auto s = summarise(lat);
    print_row("get_miss", n, val_sz, 1, s);
    cleanup();
}

static void bench_remove(uint32_t n, size_t val_sz) {
    cleanup();
    auto cache = Cache::open({
        .device_path    = g_path,
        .ring_capacity  = g_is_blk ? 0 : static_cast<uint64_t>(n) * (val_sz + 64) * 2,
        .memtable_size  = 4 << 20,
        .index_capacity = n,
        .num_shards     = PERF_SHARDS,
    });

    auto keys = gen_keys(n);
    auto val  = gen_value(val_sz);
    for (uint32_t i = 0; i < n; ++i)
        cache.put(keys[i].data(), keys[i].size(), val.data(), val.size());

    uint32_t m = std::min(n, kMaxMeasure);
    std::mt19937 rng(99);
    std::uniform_int_distribution<uint32_t> dist(0, n - 1);

    std::vector<double> lat(m);
    for (uint32_t i = 0; i < m; ++i) {
        uint32_t k = dist(rng);
        auto t0 = Clock::now();
        cache.remove(keys[k].data(), keys[k].size());
        auto t1 = Clock::now();
        lat[i] = std::chrono::duration_cast<ns_d>(t1 - t0).count();
    }
    auto s = summarise(lat);
    print_row("remove", n, val_sz, 1, s);
    cleanup();
}

static void bench_mixed(uint32_t n, size_t val_sz, int num_threads) {
    cleanup();
    auto cache = Cache::open({
        .device_path    = g_path,
        .ring_capacity  = g_is_blk ? 0 : static_cast<uint64_t>(n) * (val_sz + 64) * 4,
        .memtable_size  = 4 << 20,
        .index_capacity = n,
        .num_shards     = PERF_SHARDS,
    });

    auto keys = gen_keys(n);
    auto val  = gen_value(val_sz);

    // Pre-populate half the keys.
    for (uint32_t i = 0; i < n / 2; ++i)
        cache.put(keys[i].data(), keys[i].size(), val.data(), val.size());

    uint32_t ops_per_thread = std::min(n, kMaxMeasure);
    std::vector<std::vector<double>> all_lat(num_threads);
    std::vector<std::thread> threads;

    for (int t = 0; t < num_threads; ++t) {
        threads.emplace_back([&, t]() {
            std::mt19937 rng(100 + t);
            std::uniform_int_distribution<uint32_t> key_dist(0, n - 1);
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
    print_row("mixed_90r", n, val_sz, num_threads, s);
    cleanup();
}

// ─── main ────────────────────────────────────────────────────────────────────

int main(int argc, char* argv[]) {
    if (argc > 1) g_path = argv[1];
    g_is_blk = is_block_device(g_path);

    const char* mode = g_is_blk ? "raw block device" : "file-backed";
    std::printf("flashringc cache perf benchmark  (%s)\n", mode);
    std::printf("path: %s\n\n", g_path);

    struct Scenario {
        uint32_t keys;
        size_t   val_sz;
    };

    const Scenario scenarios[] = {
        {10'000,   128},
        {10'000,  1024},
        {10'000,  4096},
        {100'000,  128},
        {100'000, 1024},
        {100'000, 4096},
    };

    for (const auto& sc : scenarios) {
        std::printf("── keys=%u  val_sz=%zu ──\n", sc.keys, sc.val_sz);
        print_header();

        bench_put(sc.keys, sc.val_sz);
        bench_get_mem(sc.keys, sc.val_sz);
        bench_get_disk(sc.keys, sc.val_sz);
        bench_get_miss(sc.keys, sc.val_sz);
        bench_remove(sc.keys, sc.val_sz);

        int hw = static_cast<int>(std::thread::hardware_concurrency());
        if (hw < 1) hw = 4;

        for (int t : {1, 2, 4, hw}) {
            if (t > hw) continue;
            bench_mixed(sc.keys, sc.val_sz, t);
        }

        std::printf("\n");
    }

    std::printf("done.\n");
    return 0;
}
