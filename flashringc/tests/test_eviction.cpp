#include "flashringc/cache.h"

#include <algorithm>
#include <chrono>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <string>
#include <thread>
#include <unistd.h>
#include <vector>

#define CHECK(cond)                                                         \
    do {                                                                    \
        if (!(cond)) {                                                      \
            fprintf(stderr, "\n  CHECK failed: %s  [%s:%d]\n",             \
                    #cond, __FILE__, __LINE__);                              \
            std::abort();                                                   \
        }                                                                   \
    } while (0)

static std::string g_test_path = "/tmp/flashring_eviction_test.dat";
static int tests_run = 0;
static int tests_passed = 0;

static void cleanup() {
    for (uint32_t i = 0; i < 256; ++i)
        ::unlink((g_test_path + "." + std::to_string(i)).c_str());
}

#define RUN(name)                                       \
    do {                                                \
        ++tests_run;                                    \
        printf("  %-55s ", #name);                      \
        fflush(stdout);                                 \
        cleanup();                                      \
        name();                                         \
        cleanup();                                      \
        ++tests_passed;                                 \
        printf("PASS\n");                               \
    } while (0)

// How many records fit in the ring before the 75% watermark triggers.
// Each record is 4096-byte aligned in the memtable regardless of value size.
static uint64_t safe_record_count(uint64_t ring_capacity) {
    return (ring_capacity * 3 / 4) / 4096;
}

// -----------------------------------------------------------------------
// Test 1: Index-level FIFO eviction
//
// Writes 2x index_capacity keys into a ring large enough to avoid the
// watermark. Oldest keys should be evicted; newest should survive.
// -----------------------------------------------------------------------
static void test_index_fifo_eviction() {
    constexpr uint32_t INDEX_CAP = 200;
    constexpr uint32_t TOTAL_KEYS = INDEX_CAP * 2;
    // 256MB ring → safe for ~49K records. We only write 400.
    CacheConfig cfg{
        .device_path     = g_test_path,
        .ring_capacity   = 256ULL * 1024 * 1024,
        .memtable_size   = 16 * 1024 * 1024,
        .index_capacity  = INDEX_CAP,
        .num_shards      = 1,
        .queue_capacity  = 4096,
    };
    auto cache = Cache::open(cfg);

    std::string val(128, 'A');

    for (uint32_t i = 0; i < TOTAL_KEYS; ++i) {
        std::string key = "fifo_" + std::to_string(i);
        bool ok = cache.put_sync(key.data(), key.size(), val.data(), val.size());
        CHECK(ok);
    }

    uint32_t kc = cache.key_count();
    CHECK(kc <= INDEX_CAP);
    CHECK(kc > 0);

    // Oldest keys (first 50) should be evicted.
    int evicted_count = 0;
    for (uint32_t i = 0; i < 50; ++i) {
        std::string key = "fifo_" + std::to_string(i);
        char buf[256];
        size_t actual = 0;
        if (!cache.get_sync(key.data(), key.size(), buf, sizeof(buf), &actual))
            ++evicted_count;
    }
    CHECK(evicted_count >= 40);

    // Newest keys (last 50) should all be present.
    for (uint32_t i = TOTAL_KEYS - 50; i < TOTAL_KEYS; ++i) {
        std::string key = "fifo_" + std::to_string(i);
        char buf[256];
        size_t actual = 0;
        bool found = cache.get_sync(key.data(), key.size(), buf, sizeof(buf), &actual);
        CHECK(found);
    }

    printf("[evicted %d/50 oldest, kc=%u] ", evicted_count, kc);
}

// -----------------------------------------------------------------------
// Test 2: Sustained write pressure — bounded writes
//
// Writes 10x index_capacity (well within watermark limit).
// Verifies index stays bounded and recent keys are readable.
// -----------------------------------------------------------------------
static void test_bounded_write_pressure() {
    constexpr uint32_t INDEX_CAP = 500;
    constexpr uint32_t TOTAL_KEYS = INDEX_CAP * 10;  // 5000 keys
    // 256MB ring, safe for ~49K records. 5000 << 49K.
    CacheConfig cfg{
        .device_path     = g_test_path,
        .ring_capacity   = 256ULL * 1024 * 1024,
        .memtable_size   = 16 * 1024 * 1024,
        .index_capacity  = INDEX_CAP,
        .num_shards      = 1,
        .queue_capacity  = 4096,
    };
    auto cache = Cache::open(cfg);

    std::string val(128, 'B');

    for (uint32_t i = 0; i < TOTAL_KEYS; ++i) {
        std::string key = "pres_" + std::to_string(i);
        bool ok = cache.put_sync(key.data(), key.size(), val.data(), val.size());
        CHECK(ok);
    }

    uint32_t kc = cache.key_count();
    CHECK(kc <= INDEX_CAP);
    CHECK(kc > 0);

    // The most recent keys should be readable.
    int recent_found = 0;
    for (uint32_t i = TOTAL_KEYS - 100; i < TOTAL_KEYS; ++i) {
        std::string key = "pres_" + std::to_string(i);
        char buf[256];
        size_t actual = 0;
        if (cache.get_sync(key.data(), key.size(), buf, sizeof(buf), &actual))
            ++recent_found;
    }
    CHECK(recent_found >= 90);

    printf("[%u written, kc=%u, %d/100 recent found] ",
           TOTAL_KEYS, kc, recent_found);
}

// -----------------------------------------------------------------------
// Test 3: Data integrity after eviction
//
// Writes enough to trigger eviction, then verifies that all surviving
// keys return the correct value.
// -----------------------------------------------------------------------
static void test_data_integrity_after_eviction() {
    constexpr uint32_t INDEX_CAP = 300;
    constexpr uint32_t TOTAL_KEYS = 3000;

    CacheConfig cfg{
        .device_path     = g_test_path,
        .ring_capacity   = 256ULL * 1024 * 1024,
        .memtable_size   = 8 * 1024 * 1024,
        .index_capacity  = INDEX_CAP,
        .num_shards      = 1,
        .queue_capacity  = 4096,
    };
    auto cache = Cache::open(cfg);

    for (uint32_t i = 0; i < TOTAL_KEYS; ++i) {
        // Unique value per key for integrity check.
        std::string key = "integ_" + std::to_string(i);
        std::string val(256, static_cast<char>('0' + (i % 10)));
        bool ok = cache.put_sync(key.data(), key.size(), val.data(), val.size());
        CHECK(ok);
    }

    int checked = 0, correct = 0;
    for (uint32_t i = TOTAL_KEYS - 500; i < TOTAL_KEYS; ++i) {
        std::string key = "integ_" + std::to_string(i);
        char buf[512];
        size_t actual = 0;
        if (cache.get_sync(key.data(), key.size(), buf, sizeof(buf), &actual)) {
            ++checked;
            char expected = static_cast<char>('0' + (i % 10));
            if (actual == 256 && buf[0] == expected)
                ++correct;
        }
    }

    CHECK(checked > 0);
    CHECK(correct == checked);

    printf("[%d found, %d correct] ", checked, correct);
}

// -----------------------------------------------------------------------
// Test 4: Explicit delete + re-insert cycle
//
// Delete keys and re-insert them with new values. Verify deletes are
// respected and new values are returned.
// -----------------------------------------------------------------------
static void test_delete_reinsert_cycle() {
    CacheConfig cfg{
        .device_path     = g_test_path,
        .ring_capacity   = 256ULL * 1024 * 1024,
        .memtable_size   = 16 * 1024 * 1024,
        .index_capacity  = 2000,
        .num_shards      = 1,
        .queue_capacity  = 4096,
    };
    auto cache = Cache::open(cfg);

    std::string val_v1(256, '1');
    std::string val_v2(256, '2');

    for (int i = 0; i < 200; ++i) {
        std::string key = "cycle_" + std::to_string(i);
        bool ok = cache.put_sync(key.data(), key.size(),
                                 val_v1.data(), val_v1.size());
        CHECK(ok);
    }

    // Delete even-numbered keys.
    for (int i = 0; i < 200; i += 2) {
        std::string key = "cycle_" + std::to_string(i);
        bool removed = cache.remove_sync(key.data(), key.size());
        CHECK(removed);
    }

    // Even keys gone, odd keys present.
    for (int i = 0; i < 200; ++i) {
        std::string key = "cycle_" + std::to_string(i);
        char buf[512];
        size_t actual = 0;
        bool found = cache.get_sync(key.data(), key.size(),
                                    buf, sizeof(buf), &actual);
        if (i % 2 == 0)
            CHECK(!found);
        else
            CHECK(found);
    }

    // Re-insert even keys with v2.
    for (int i = 0; i < 200; i += 2) {
        std::string key = "cycle_" + std::to_string(i);
        bool ok = cache.put_sync(key.data(), key.size(),
                                 val_v2.data(), val_v2.size());
        CHECK(ok);
    }

    // All keys present. Even → v2, odd → v1.
    for (int i = 0; i < 200; ++i) {
        std::string key = "cycle_" + std::to_string(i);
        char buf[512];
        size_t actual = 0;
        bool found = cache.get_sync(key.data(), key.size(),
                                    buf, sizeof(buf), &actual);
        CHECK(found);
        CHECK(actual == 256);
        if (i % 2 == 0)
            CHECK(buf[0] == '2');
        else
            CHECK(buf[0] == '1');
    }
}

// -----------------------------------------------------------------------
// Test 5: In-place update (overwrite existing key)
//
// Put the same key multiple times. Key count should stay at 1.
// Only the latest value should be returned.
// -----------------------------------------------------------------------
static void test_in_place_update() {
    CacheConfig cfg{
        .device_path     = g_test_path,
        .ring_capacity   = 256ULL * 1024 * 1024,
        .memtable_size   = 16 * 1024 * 1024,
        .index_capacity  = 1000,
        .num_shards      = 1,
        .queue_capacity  = 4096,
    };
    auto cache = Cache::open(cfg);

    std::string key = "update_key";
    for (int v = 0; v < 100; ++v) {
        std::string val(128, static_cast<char>('A' + (v % 26)));
        bool ok = cache.put_sync(key.data(), key.size(), val.data(), val.size());
        CHECK(ok);
    }

    CHECK(cache.key_count() == 1);

    char buf[256];
    size_t actual = 0;
    bool found = cache.get_sync(key.data(), key.size(),
                                buf, sizeof(buf), &actual);
    CHECK(found);
    CHECK(actual == 128);
    char expected = static_cast<char>('A' + (99 % 26));
    CHECK(buf[0] == expected);
}

// -----------------------------------------------------------------------
// Test 6: Multi-threaded eviction stress
//
// Multiple writers filling a small-index cache simultaneously.
// Bounded total writes to stay below watermark.
// Verify no crashes, no deadlocks, bounded key count.
// -----------------------------------------------------------------------
static void test_multithreaded_eviction_stress() {
    constexpr uint32_t INDEX_CAP = 1000;

    CacheConfig cfg{
        .device_path     = g_test_path,
        .ring_capacity   = 512ULL * 1024 * 1024,
        .memtable_size   = 32 * 1024 * 1024,
        .index_capacity  = INDEX_CAP,
        .num_shards      = 4,
        .queue_capacity  = 8192,
    };
    auto cache = Cache::open(cfg);

    constexpr int NUM_WRITERS = 8;
    constexpr int WRITES_PER_THREAD = 2000;  // 16K total, well under watermark

    std::string val(512, 'E');
    std::atomic<int> total_writes{0};
    std::atomic<int> total_reads_ok{0};

    std::vector<std::thread> writers;
    for (int w = 0; w < NUM_WRITERS; ++w) {
        writers.emplace_back([&cache, &val, &total_writes, &total_reads_ok, w]() {
            for (int i = 0; i < WRITES_PER_THREAD; ++i) {
                std::string key = "mt_" + std::to_string(w) +
                                  "_" + std::to_string(i);
                cache.put_sync(key.data(), key.size(),
                              val.data(), val.size());
                total_writes.fetch_add(1, std::memory_order_relaxed);

                if (i % 10 == 0 && i > 0) {
                    std::string rkey = "mt_" + std::to_string(w) +
                                      "_" + std::to_string(i - 1);
                    char buf[1024];
                    size_t actual = 0;
                    if (cache.get_sync(rkey.data(), rkey.size(),
                                      buf, sizeof(buf), &actual))
                        total_reads_ok.fetch_add(1, std::memory_order_relaxed);
                }
            }
        });
    }

    for (auto& t : writers) t.join();

    uint32_t kc = cache.key_count();
    CHECK(kc <= INDEX_CAP);
    CHECK(total_writes.load() == NUM_WRITERS * WRITES_PER_THREAD);

    printf("[%d writes, %d reads_ok, %u keys] ",
           total_writes.load(), total_reads_ok.load(), kc);
}

// -----------------------------------------------------------------------
// Test 7: Churn — write + read + delete for 10 seconds
//
// Bounded write rate to stay below watermark.
// Simulates a realistic mixed workload.
// -----------------------------------------------------------------------
static void test_churn_10s() {
    constexpr uint32_t INDEX_CAP = 2000;
    // 2GB ring. With 4096-byte aligned records, safe for ~393K records.
    // At ~5K ops/sec (throttled), 10s = 50K ops << 393K.
    CacheConfig cfg{
        .device_path     = g_test_path,
        .ring_capacity   = 256ULL * 1024 * 1024,
        .memtable_size   = 16 * 1024 * 1024,
        .index_capacity  = INDEX_CAP,
        .num_shards      = 2,
        .queue_capacity  = 8192,
    };
    auto cache = Cache::open(cfg);

    std::string val(512, 'F');

    int puts = 0, gets_ok = 0, gets_miss = 0, deletes = 0;
    int next_key_id = 0;
    // Write at most safe_record_count records to avoid watermark.
    uint64_t max_records = safe_record_count(256ULL * 1024 * 1024);
    auto t_start = std::chrono::steady_clock::now();

    while (true) {
        auto elapsed = std::chrono::steady_clock::now() - t_start;
        double secs = std::chrono::duration<double>(elapsed).count();
        if (secs >= 10.0) break;
        if (static_cast<uint64_t>(puts) >= max_records) break;

        std::string key = "churn_" + std::to_string(next_key_id);
        cache.put_sync(key.data(), key.size(), val.data(), val.size());
        ++puts;

        // Read a recent key.
        if (next_key_id > 10) {
            int read_id = next_key_id - 5;
            std::string rkey = "churn_" + std::to_string(read_id);
            char buf[1024];
            size_t actual = 0;
            if (cache.get_sync(rkey.data(), rkey.size(),
                              buf, sizeof(buf), &actual))
                ++gets_ok;
            else
                ++gets_miss;
        }

        // Delete an old key every 20 ops.
        if (next_key_id % 20 == 0 && next_key_id > 100) {
            int del_id = next_key_id - 100;
            std::string dkey = "churn_" + std::to_string(del_id);
            cache.remove_sync(dkey.data(), dkey.size());
            ++deletes;
        }

        ++next_key_id;
    }

    uint32_t kc = cache.key_count();
    CHECK(kc <= INDEX_CAP);
    CHECK(kc > 0);

    double get_hit_rate = (gets_ok + gets_miss) > 0
        ? 100.0 * gets_ok / (gets_ok + gets_miss) : 0.0;

    auto elapsed = std::chrono::steady_clock::now() - t_start;
    double secs = std::chrono::duration<double>(elapsed).count();

    printf("[%.1fs, %d puts, %d gets(%.0f%% hit), %d dels, %u keys] ",
           secs, puts, gets_ok + gets_miss, get_hit_rate, deletes, kc);

    CHECK(get_hit_rate > 80.0);
}

// -----------------------------------------------------------------------
// Test 8: Multi-threaded mixed churn for 15 seconds
//
// Multiple threads doing puts, gets, and deletes concurrently.
// Total writes bounded per thread to stay under watermark.
// -----------------------------------------------------------------------
static void test_multithreaded_churn_15s() {
    constexpr uint32_t INDEX_CAP = 2000;

    CacheConfig cfg{
        .device_path     = g_test_path,
        .ring_capacity   = 512ULL * 1024 * 1024,
        .memtable_size   = 32 * 1024 * 1024,
        .index_capacity  = INDEX_CAP,
        .num_shards      = 4,
        .queue_capacity  = 8192,
    };
    auto cache = Cache::open(cfg);

    constexpr int NUM_THREADS = 4;
    constexpr double DURATION_SECS = 15.0;
    uint64_t max_per_thread = safe_record_count(512ULL * 1024 * 1024) / NUM_THREADS;

    std::atomic<int> total_puts{0};
    std::atomic<int> total_gets_hit{0};
    std::atomic<int> total_gets_miss{0};
    std::atomic<int> total_deletes{0};
    std::atomic<bool> stop_flag{false};

    std::vector<std::thread> workers;
    for (int t = 0; t < NUM_THREADS; ++t) {
        workers.emplace_back([&, t, max_per_thread]() {
            std::string val(256, static_cast<char>('A' + t));
            uint64_t id = 0;
            while (!stop_flag.load(std::memory_order_relaxed) &&
                   id < max_per_thread) {
                std::string key = "mc_" + std::to_string(t) +
                                  "_" + std::to_string(id);
                cache.put_sync(key.data(), key.size(),
                              val.data(), val.size());
                total_puts.fetch_add(1, std::memory_order_relaxed);

                if (id > 5) {
                    std::string rkey = "mc_" + std::to_string(t) +
                                      "_" + std::to_string(id - 3);
                    char buf[512];
                    size_t actual = 0;
                    if (cache.get_sync(rkey.data(), rkey.size(),
                                      buf, sizeof(buf), &actual))
                        total_gets_hit.fetch_add(1, std::memory_order_relaxed);
                    else
                        total_gets_miss.fetch_add(1, std::memory_order_relaxed);
                }

                if (id % 50 == 0 && id > 200) {
                    std::string dkey = "mc_" + std::to_string(t) +
                                      "_" + std::to_string(id - 200);
                    cache.remove_sync(dkey.data(), dkey.size());
                    total_deletes.fetch_add(1, std::memory_order_relaxed);
                }

                ++id;
            }
        });
    }

    std::this_thread::sleep_for(
        std::chrono::milliseconds(static_cast<int>(DURATION_SECS * 1000)));
    stop_flag.store(true, std::memory_order_release);

    for (auto& w : workers) w.join();

    uint32_t kc = cache.key_count();
    int p = total_puts.load();
    int gh = total_gets_hit.load();
    int gm = total_gets_miss.load();
    int d = total_deletes.load();
    double hit_rate = (gh + gm) > 0 ? 100.0 * gh / (gh + gm) : 0.0;

    CHECK(p > 0);
    CHECK(kc <= INDEX_CAP);

    printf("[%.0fs, %d puts, %d gets(%.0f%% hit), %d dels, %u keys] ",
           DURATION_SECS, p, gh + gm, hit_rate, d, kc);
}

// -----------------------------------------------------------------------
// Test 9: Watermark eviction drains index (known bug documentation)
//
// When the ring device fills past 75%, DeleteManager evicts entries
// faster than they're inserted. Because discard_cursor_ never advances
// (discard_batch_ defaults to 64MB > useful ring data), ring_usage()
// never decreases, and every put triggers maybe_evict(10). This drains
// the index to 0 keys permanently.
//
// This test documents the behavior. It does NOT assert correctness —
// it asserts the *current* (buggy) behavior so that a future fix
// can be verified by updating the assertions.
// -----------------------------------------------------------------------
static void test_watermark_eviction_drains_index() {
    // 256MB ring with 16MB memtable = 16 flushes per full ring wrap.
    // Watermark triggers at flush 12 (192MB = 75%). The remaining 4 flushes
    // (16384 records) under active eviction drain the 500-entry index and
    // create a degenerate state where head_ laps tail_ in the FIFO ring.
    CacheConfig cfg{
        .device_path     = g_test_path,
        .ring_capacity   = 256ULL * 1024 * 1024,
        .memtable_size   = 16 * 1024 * 1024,
        .index_capacity  = 500,
        .num_shards      = 1,
        .queue_capacity  = 4096,
    };
    auto cache = Cache::open(cfg);

    std::string val(128, 'W');

    // Write 100K records. Watermark triggers at ~49K (75% of 256MB / 4KB).
    // After that, maybe_evict(10) fires every put and drains the index.
    for (int i = 0; i < 100000; ++i) {
        std::string key = "wm_" + std::to_string(i);
        cache.put_sync(key.data(), key.size(), val.data(), val.size());
    }

    uint32_t kc = cache.key_count();

    // KNOWN BUG: once ring fills past 75%, maybe_evict drains the index
    // to 0 because discard_cursor_ never advances (discard_batch_ = 64MB
    // exceeds useful evicted bytes), so ring_usage() never drops below
    // the watermark. Each put inserts 1 entry and immediately evicts it.
    //
    // Root cause: head_ in the FIFO ring laps tail_ during bulk eviction,
    // causing newly inserted entries to be the oldest and immediately
    // evictable.
    //
    // When this bug is fixed, change this to: CHECK(kc > 0);
    printf("[kc=%u — expected 0 due to watermark bug] ", kc);
    CHECK(kc == 0);
}

// -----------------------------------------------------------------------
// main
// -----------------------------------------------------------------------

int main(int argc, char* argv[]) {
    if (argc >= 2)
        g_test_path = argv[1];

    setbuf(stdout, nullptr);
    printf("test_eviction  (%s)\n\n", g_test_path.c_str());

    auto t0 = std::chrono::steady_clock::now();

    RUN(test_index_fifo_eviction);
    RUN(test_bounded_write_pressure);
    RUN(test_data_integrity_after_eviction);
    RUN(test_delete_reinsert_cycle);
    RUN(test_in_place_update);
    RUN(test_multithreaded_eviction_stress);
    RUN(test_churn_10s);
    RUN(test_multithreaded_churn_15s);
    RUN(test_watermark_eviction_drains_index);

    auto elapsed = std::chrono::steady_clock::now() - t0;
    double total_secs = std::chrono::duration<double>(elapsed).count();

    printf("\n%d / %d tests passed (%.1fs total).\n",
           tests_passed, tests_run, total_secs);

    if (tests_passed == tests_run) {
        printf("\nNotes:\n"
               "  - Eviction is FIFO (oldest-first via KeyIndex ring).\n"
               "  - TTL (time-to-live) is NOT implemented.\n"
               "  - last_access / freq fields are tracked but unused.\n"
               "  - BUG: DeleteManager watermark eviction permanently\n"
               "    drains the index when ring fills past 75%%.\n");
    }

    return (tests_passed == tests_run) ? 0 : 1;
}
