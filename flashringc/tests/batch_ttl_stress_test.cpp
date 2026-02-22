// Stress test: 50M key space, 4KB values, get_batch(100) with miss->put and 15min TTL.
// Runs 30 min; logs num_batches, avg, p90, p95, p99, p99.9, hit_rate every 1 min.
//
// Usage: ./batch_ttl_stress_test <device_path> [ring_mb] [index_cap] [memtable_mb] [num_shards] [use_io_uring]
//   ring_mb        default 4096   (0 = auto for block device)
//   index_cap      default 2000000
//   memtable_mb    default 64
//   num_shards     default 8
//   use_io_uring   default 1 (0=pread, 1=io_uring)

#include "cache.h"

#include <algorithm>
#include <chrono>
#include <cinttypes>
#include <cstdint>
#include <cstdio>
#include <cstring>
#include <random>
#include <string>
#include <vector>

static const char* DEFAULT_PATH = "/tmp/flashring_batch_ttl_stress.dat";

static constexpr uint64_t KEY_SPACE    = 50'000'000ULL;
static constexpr size_t   VAL_SIZE     = 4096;
static constexpr size_t   BATCH_SIZE   = 100;
static constexpr uint64_t TTL_MS       = 15ULL * 60 * 1000;
static constexpr int      RUN_MINUTES  = 30;
static constexpr int      LOG_INTERVAL_SEC = 60;

using Clock  = std::chrono::steady_clock;
using ms_d   = std::chrono::duration<double, std::milli>;
using ns_d   = std::chrono::duration<double, std::nano>;

static constexpr size_t KEY_LEN = sizeof(uint64_t);

// Value layout: first 8 bytes = timestamp (ms since test start), rest = payload
static constexpr size_t TS_OFFSET = 0;
static constexpr size_t TS_SIZE   = 8;

static uint64_t now_ms(const Clock::time_point& start) {
    return static_cast<uint64_t>(
        std::chrono::duration_cast<std::chrono::milliseconds>(
            Clock::now() - start).count());
}

int main(int argc, char* argv[]) {
    const char* path         = argc > 1 ? argv[1] : DEFAULT_PATH;
    uint64_t ring_cap_mb     = argc > 2 ? static_cast<uint64_t>(std::stoull(argv[2])) : 4096;
    uint32_t index_cap       = argc > 3 ? static_cast<uint32_t>(std::stoul(argv[3])) : 2'000'000;
    uint64_t memtable_mb     = argc > 4 ? static_cast<uint64_t>(std::stoull(argv[4])) : 64;
    uint32_t num_shards      = argc > 5 ? static_cast<uint32_t>(std::stoul(argv[5])) : 8;
    bool use_io_uring        = argc > 6 ? (std::stoul(argv[6]) != 0) : true;

    uint64_t ring_capacity   = ring_cap_mb * 1024ULL * 1024;
    size_t memtable_size     = memtable_mb * 1024ULL * 1024;

    CacheConfig cfg{
        .device_path    = path,
        .ring_capacity  = ring_capacity,
        .memtable_size  = memtable_size,
        .index_capacity = index_cap,
        .num_shards     = num_shards,
        .use_io_uring   = use_io_uring,
    };

    std::printf("batch_ttl_stress_test\n");
    std::printf("  device_path=%s ring_capacity=%" PRIu64 " MB memtable_size=%zu MB index_capacity=%u num_shards=%u use_io_uring=%d\n",
                cfg.device_path.c_str(), ring_cap_mb, memtable_mb, cfg.index_capacity, cfg.num_shards, cfg.use_io_uring ? 1 : 0);
    std::printf("  keyspace=%" PRIu64 " val_sz=%zu batch=%zu ttl_min=15 run_min=%d log_interval=%ds\n\n",
                KEY_SPACE, VAL_SIZE, BATCH_SIZE, RUN_MINUTES, LOG_INTERVAL_SEC);

    auto cache = Cache::open(cfg);

    Clock::time_point run_start = Clock::now();
    std::mt19937_64 rng(12345);
    std::uniform_int_distribution<uint64_t> key_dist(0, KEY_SPACE - 1);

    // Buffers for one batch
    std::vector<uint64_t> key_ids(BATCH_SIZE);
    std::vector<uint8_t> key_storage(BATCH_SIZE * KEY_LEN);
    std::vector<uint8_t> val_storage(BATCH_SIZE * VAL_SIZE);
    std::vector<BatchGetRequest> reqs(BATCH_SIZE);
    std::vector<BatchGetResult>  results(BATCH_SIZE);

    // Value template: timestamp (filled per put) + payload
    std::vector<uint8_t> value_template(VAL_SIZE);
    for (size_t i = TS_SIZE; i < VAL_SIZE; ++i)
        value_template[i] = static_cast<uint8_t>((i * 7 + 13) & 0xFF);

    // Sliding window for 1-min stats
    struct Sample {
        double     latency_ns;
        uint64_t   hits;
        uint64_t   misses;
        uint64_t   end_time_ms;
    };
    std::vector<Sample> window;
    int next_log_sec = LOG_INTERVAL_SEC;
    uint64_t total_batches = 0;
    uint64_t total_hits = 0;
    uint64_t total_misses = 0;

    while (true) {
        uint64_t elapsed_ms = now_ms(run_start);
        if (elapsed_ms >= static_cast<uint64_t>(RUN_MINUTES * 60 * 1000))
            break;

        // Pick 100 random keys
        for (size_t i = 0; i < BATCH_SIZE; ++i)
            key_ids[i] = key_dist(rng);

        for (size_t i = 0; i < BATCH_SIZE; ++i) {
            std::memcpy(key_storage.data() + i * KEY_LEN, &key_ids[i], KEY_LEN);
            reqs[i] = {
                .key         = key_storage.data() + i * KEY_LEN,
                .key_len     = KEY_LEN,
                .val_buf     = val_storage.data() + i * VAL_SIZE,
                .val_buf_len = VAL_SIZE,
            };
        }

        auto t0 = Clock::now();
        cache.get_batch(reqs.data(), results.data(), BATCH_SIZE);

        uint64_t hits = 0, misses = 0;

        for (size_t i = 0; i < BATCH_SIZE; ++i) {
            if (!results[i].found) {
                ++misses;
                std::memcpy(value_template.data(), &elapsed_ms, TS_SIZE);
                cache.put(reqs[i].key, KEY_LEN, value_template.data(), VAL_SIZE);
                continue;
            }
            if (results[i].actual_len < TS_SIZE) {
                ++misses;
                std::memcpy(value_template.data(), &elapsed_ms, TS_SIZE);
                cache.put(reqs[i].key, KEY_LEN, value_template.data(), VAL_SIZE);
                continue;
            }
            uint64_t stored_ts = 0;
            std::memcpy(&stored_ts, val_storage.data() + i * VAL_SIZE, TS_SIZE);
            if (elapsed_ms - stored_ts > TTL_MS) {
                ++misses;
                std::memcpy(value_template.data(), &elapsed_ms, TS_SIZE);
                cache.put(reqs[i].key, KEY_LEN, value_template.data(), VAL_SIZE);
            } else {
                ++hits;
            }
        }

        auto t1 = Clock::now();
        double batch_ns = std::chrono::duration_cast<ns_d>(t1 - t0).count();

        total_batches++;
        total_hits += hits;
        total_misses += misses;

        window.push_back({
            batch_ns,
            hits,
            misses,
            elapsed_ms,
        });

        // Log every LOG_INTERVAL_SEC
        uint64_t elapsed_sec = elapsed_ms / 1000;
        if (elapsed_sec >= static_cast<uint64_t>(next_log_sec)) {
            uint64_t window_end_ms = elapsed_ms;
            uint64_t window_start_ms = (window_end_ms > static_cast<uint64_t>(LOG_INTERVAL_SEC) * 1000)
                ? window_end_ms - static_cast<uint64_t>(LOG_INTERVAL_SEC) * 1000
                : 0;

            std::vector<double> lats;
            uint64_t wh = 0, wm = 0;
            for (const auto& s : window) {
                if (s.end_time_ms >= window_start_ms) {
                    lats.push_back(s.latency_ns);
                    wh += s.hits;
                    wm += s.misses;
                }
            }

            double avg_ns = 0, p90_ns = 0, p95_ns = 0, p99_ns = 0, p999_ns = 0;
            double hitrate = (wh + wm) > 0 ? static_cast<double>(wh) / (wh + wm) : 0.0;

            if (!lats.empty()) {
                std::sort(lats.begin(), lats.end());
                size_t n = lats.size();
                double sum = 0;
                for (double x : lats) sum += x;
                avg_ns  = sum / n;
                p90_ns  = lats[std::min(n - 1, static_cast<size_t>(n * 0.90))];
                p95_ns  = lats[std::min(n - 1, static_cast<size_t>(n * 0.95))];
                p99_ns  = lats[std::min(n - 1, static_cast<size_t>(n * 0.99))];
                p999_ns = lats[std::min(n - 1, static_cast<size_t>(n * 0.999))];
            }

            std::printf("min=%-3lu batches=%-8lu avg_us=%-8.2f p90_us=%-8.2f p95_us=%-8.2f p99_us=%-8.2f p999_us=%-8.2f hitrate=%.4f\n",
                        static_cast<unsigned long>(elapsed_sec / 60),
                        static_cast<unsigned long>(lats.size()),
                        avg_ns / 1000.,
                        p90_ns / 1000.,
                        p95_ns / 1000.,
                        p99_ns / 1000.,
                        p999_ns / 1000.,
                        hitrate);

            next_log_sec += LOG_INTERVAL_SEC;

            // Trim window to last 1 min
            auto it = std::remove_if(window.begin(), window.end(),
                [window_start_ms](const Sample& s) { return s.end_time_ms < window_start_ms; });
            window.erase(it, window.end());
        }
    }

    std::printf("\ndone. total_batches=%lu total_hits=%lu total_misses=%lu hitrate=%.4f\n",
                static_cast<unsigned long>(total_batches),
                static_cast<unsigned long>(total_hits),
                static_cast<unsigned long>(total_misses),
                (total_hits + total_misses) > 0
                    ? static_cast<double>(total_hits) / (total_hits + total_misses)
                    : 0.0);
    return 0;
}
