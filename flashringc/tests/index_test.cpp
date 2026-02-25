#include "flashringc/key_index.h"

#include <cassert>
#include <cstdio>
#include <cstring>
#include <string>
#include <thread>
#include <vector>

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

static Hash128 make_hash(const std::string& key) {
    return hash_key(key.data(), key.size());
}

static int tests_run = 0;
static int tests_passed = 0;

#define RUN(name)                                       \
    do {                                                \
        ++tests_run;                                    \
        printf("  %-40s ", #name);                      \
        name();                                         \
        ++tests_passed;                                 \
        printf("PASS\n");                               \
    } while (0)

// ---------------------------------------------------------------------------
// tests
// ---------------------------------------------------------------------------

static void test_put_get() {
    KeyIndex idx(64);
    auto h = make_hash("key1");

    uint32_t slot = idx.put(h, /*mem_id=*/1, /*offset=*/100, /*length=*/42, /*now_delta=*/10, /*ttl_seconds=*/0);
    assert(idx.size() == 1);

    LookupResult lr{};
    bool found = idx.get(h, /*now_delta=*/10, lr);
    assert(found);
    assert(lr.mem_id == 1);
    assert(lr.offset == 100);
    assert(lr.length == 42);
    assert(lr.last_access == 10);  // insert time from put
    (void)slot;
}

static void test_put_update() {
    KeyIndex idx(64);
    auto h = make_hash("key1");

    idx.put(h, 1, 100, 42, 0, 0);
    idx.put(h, 2, 200, 84, 0, 0);  // update same key
    assert(idx.size() == 1);

    LookupResult lr{};
    assert(idx.get(h, 5, lr));
    assert(lr.mem_id == 2);
    assert(lr.offset == 200);
    assert(lr.length == 84);
}

static void test_remove() {
    KeyIndex idx(64);
    auto h = make_hash("key1");

    idx.put(h, 1, 100, 42, 0, 0);
    assert(idx.size() == 1);

    bool removed = idx.remove(h);
    assert(removed);
    assert(idx.size() == 0);

    LookupResult lr{};
    assert(!idx.get(h, 0, lr));

    assert(!idx.remove(h));  // double remove returns false
}

static void test_miss() {
    KeyIndex idx(64);
    auto h = make_hash("nonexistent");
    LookupResult lr{};
    assert(!idx.get(h, 0, lr));
}

static void test_collision_same_lo() {
    // Simulate a collision: same hash_lo, different hash_hi.
    KeyIndex idx(64);

    Hash128 h1 = {0xAAAA, 0x1111};
    Hash128 h2 = {0xAAAA, 0x2222};  // same lo, different hi

    idx.put(h1, 1, 100, 10, 0, 0);
    idx.put(h2, 2, 200, 20, 0, 0);

    // h2's put overwrites h1's map slot (same lo).
    // h1 should no longer be found because the map now points to h2's slot.
    LookupResult lr{};
    assert(idx.get(h2, 0, lr));
    assert(lr.mem_id == 2);
    assert(lr.offset == 200);

    // h1 lookup: map points to h2's slot, hash_hi mismatch -> not found.
    assert(!idx.get(h1, 0, lr));
}

static void test_evict_oldest() {
    KeyIndex idx(8);

    // Fill the ring.
    std::vector<Hash128> hashes;
    for (int i = 0; i < 8; ++i) {
        auto h = make_hash("k" + std::to_string(i));
        hashes.push_back(h);
        idx.put(h, static_cast<uint32_t>(i), static_cast<uint32_t>(i * 100), 10, 0, 0);
    }
    assert(idx.size() == 8);

    // Evict the 3 oldest.
    uint32_t evicted = idx.evict_oldest(3);
    assert(evicted == 3);
    assert(idx.size() == 5);

    // The first 3 keys should be gone.
    LookupResult lr{};
    for (int i = 0; i < 3; ++i)
        assert(!idx.get(hashes[i], 0, lr));

    // The remaining 5 should still be present.
    for (int i = 3; i < 8; ++i)
        assert(idx.get(hashes[i], 0, lr));
}

static void test_ring_wrap_auto_evict() {
    KeyIndex idx(4);

    // Insert 4 entries to fill ring.
    for (int i = 0; i < 4; ++i) {
        auto h = make_hash("w" + std::to_string(i));
        idx.put(h, static_cast<uint32_t>(i), static_cast<uint32_t>(i), 1, 0, 0);
    }
    assert(idx.size() == 4);

    // Insert a 5th — should auto-evict the oldest.
    auto h5 = make_hash("w4");
    idx.put(h5, 4, 400, 1, 0, 0);
    assert(idx.size() == 4);

    // w0 should be gone.
    LookupResult lr{};
    assert(!idx.get(make_hash("w0"), 0, lr));

    // w4 should be present.
    assert(idx.get(h5, 0, lr));
    assert(lr.mem_id == 4);
}

static void test_freq_increment() {
    KeyIndex idx(64);
    auto h = make_hash("freq_key");
    idx.put(h, 1, 0, 10, 0, 0);

    // Hit the key many times; freq should increase (probabilistically).
    LookupResult lr{};
    for (int i = 0; i < 1000; ++i)
        idx.get(h, static_cast<uint16_t>(i), lr);

    // After 1000 hits the Morris counter should be at least 5.
    idx.get(h, 0, lr);
    assert(lr.freq >= 5);
}

static void test_last_access_update() {
    KeyIndex idx(64);
    auto h = make_hash("access_key");
    idx.put(h, 1, 0, 10, 0, 0);

    LookupResult lr{};
    idx.get(h, 42, lr);

    // The next get should read the last_access stored by the previous get.
    idx.get(h, 99, lr);
    assert(lr.last_access == 42);
}

static void test_concurrent_freq_update() {
    KeyIndex idx(64);
    auto h = make_hash("conc_key");
    idx.put(h, 1, 0, 10, 0, 0);

    constexpr int kThreads = 4;
    constexpr int kOps     = 5000;
    std::vector<std::thread> threads;

    for (int t = 0; t < kThreads; ++t) {
        threads.emplace_back([&]() {
            LookupResult lr{};
            for (int i = 0; i < kOps; ++i)
                idx.get(h, static_cast<uint16_t>(i & 0xFFFF), lr);
        });
    }
    for (auto& t : threads)
        t.join();

    LookupResult lr{};
    idx.get(h, 0, lr);
    // With 20000 total hits the Morris counter should be well above 0.
    assert(lr.freq >= 5);
}

static void test_ttl_no_expiry() {
    KeyIndex idx(64);
    auto h = make_hash("ttl_key");
    idx.put(h, 1, 100, 10, /*now_delta=*/0, /*ttl_seconds=*/5);

    LookupResult lr{};
    assert(idx.get(h, 3, lr));
    assert(lr.mem_id == 1);
}

static void test_ttl_expired() {
    KeyIndex idx(64);
    auto h = make_hash("ttl_exp_key");
    idx.put(h, 1, 100, 10, /*now_delta=*/0, /*ttl_seconds=*/2);

    LookupResult lr{};
    assert(idx.get(h, 1, lr));
    assert(!idx.get(h, 3, lr));  // expired: age 3 >= 2
    assert(idx.size() == 0);     // lazy-removed
}

static void test_ttl_zero_means_never_expire() {
    KeyIndex idx(64);
    auto h = make_hash("ttl_zero_key");
    idx.put(h, 1, 100, 10, 0, /*ttl_seconds=*/0);

    LookupResult lr{};
    assert(idx.get(h, 1000, lr));
    assert(idx.get(h, 2000, lr));
    assert(lr.mem_id == 1);
}

static void test_hash_key_consistency() {
    const char* key = "deterministic";
    Hash128 a = hash_key(key, strlen(key));
    Hash128 b = hash_key(key, strlen(key));
    assert(a.lo == b.lo);
    assert(a.hi == b.hi);
}

static void test_many_inserts() {
    constexpr uint32_t N = 10000;
    KeyIndex idx(N);

    for (uint32_t i = 0; i < N; ++i) {
        auto h = make_hash("bulk" + std::to_string(i));
        idx.put(h, i, i, static_cast<uint32_t>(i & 0xFFFF), 0, 0);
    }
    assert(idx.size() == N);

    LookupResult lr{};
    for (uint32_t i = 0; i < N; ++i) {
        auto h = make_hash("bulk" + std::to_string(i));
        assert(idx.get(h, 0, lr));
        assert(lr.mem_id == i);
    }
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

int main() {
    printf("index_test\n");

    RUN(test_put_get);
    RUN(test_put_update);
    RUN(test_remove);
    RUN(test_miss);
    RUN(test_collision_same_lo);
    RUN(test_evict_oldest);
    RUN(test_ring_wrap_auto_evict);
    RUN(test_freq_increment);
    RUN(test_last_access_update);
    RUN(test_concurrent_freq_update);
    RUN(test_ttl_no_expiry);
    RUN(test_ttl_expired);
    RUN(test_ttl_zero_means_never_expire);
    RUN(test_hash_key_consistency);
    RUN(test_many_inserts);

    printf("\n%d / %d tests passed.\n", tests_passed, tests_run);
    return (tests_passed == tests_run) ? 0 : 1;
}
