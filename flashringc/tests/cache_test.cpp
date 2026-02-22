#include "cache.h"

#include <cassert>
#include <cstdio>
#include <cstring>
#include <string>
#include <unistd.h>
#include <vector>

static const char* TEST_PATH = "/tmp/flashring_cache_test.dat";
static constexpr uint32_t TEST_SHARDS = 4;

static int tests_run = 0;
static int tests_passed = 0;

static void cleanup_shard_files() {
    for (uint32_t i = 0; i < 256; ++i) {
        std::string path = std::string(TEST_PATH) + "." + std::to_string(i);
        ::unlink(path.c_str());
    }
}

#define RUN(name)                                       \
    do {                                                \
        ++tests_run;                                    \
        printf("  %-40s ", #name);                      \
        cleanup_shard_files();                          \
        name();                                         \
        cleanup_shard_files();                          \
        ++tests_passed;                                 \
        printf("PASS\n");                               \
    } while (0)

static CacheConfig test_cfg(uint32_t index_cap = 1024) {
    return {
        .device_path    = TEST_PATH,
        .ring_capacity  = 16 * 1024 * 1024,   // 16 MB total
        .memtable_size  = 1024 * 1024,         // 1 MB total (256 KB per shard)
        .index_capacity = index_cap,
        .num_shards     = TEST_SHARDS,
    };
}

// ─── Tests ───────────────────────────────────────────────────────────────────

static void test_put_get_small() {
    auto cache = Cache::open(test_cfg());

    std::string key = "hello";
    std::string val = "world";

    assert(cache.put(key.data(), key.size(), val.data(), val.size()));

    char buf[64] = {};
    size_t actual = 0;
    assert(cache.get(key.data(), key.size(), buf, sizeof(buf), &actual));
    assert(actual == val.size());
    assert(std::memcmp(buf, val.data(), val.size()) == 0);
}

static void test_put_get_large() {
    auto cache = Cache::open(test_cfg());

    std::string key = "large_key";
    std::vector<uint8_t> val(100'000);
    for (size_t i = 0; i < val.size(); ++i)
        val[i] = static_cast<uint8_t>((i * 7 + 13) & 0xFF);

    assert(cache.put(key.data(), key.size(), val.data(), val.size()));

    std::vector<uint8_t> out(val.size());
    size_t actual = 0;
    assert(cache.get(key.data(), key.size(), out.data(), out.size(), &actual));
    assert(actual == val.size());
    assert(std::memcmp(out.data(), val.data(), val.size()) == 0);
}

static void test_get_miss() {
    auto cache = Cache::open(test_cfg());

    std::string key = "nonexistent";
    char buf[16];
    size_t actual = 0;
    assert(!cache.get(key.data(), key.size(), buf, sizeof(buf), &actual));
}

static void test_put_overwrite() {
    auto cache = Cache::open(test_cfg());

    std::string key = "mykey";
    std::string val1 = "first_value";
    std::string val2 = "second_value_longer";

    assert(cache.put(key.data(), key.size(), val1.data(), val1.size()));
    assert(cache.put(key.data(), key.size(), val2.data(), val2.size()));

    char buf[64] = {};
    size_t actual = 0;
    assert(cache.get(key.data(), key.size(), buf, sizeof(buf), &actual));
    assert(actual == val2.size());
    assert(std::memcmp(buf, val2.data(), val2.size()) == 0);
}

static void test_remove() {
    auto cache = Cache::open(test_cfg());

    std::string key = "to_delete";
    std::string val = "temporary";

    assert(cache.put(key.data(), key.size(), val.data(), val.size()));

    char buf[32];
    size_t actual = 0;
    assert(cache.get(key.data(), key.size(), buf, sizeof(buf), &actual));

    assert(cache.remove(key.data(), key.size()));

    assert(!cache.get(key.data(), key.size(), buf, sizeof(buf), &actual));
    assert(!cache.remove(key.data(), key.size()));
}

static void test_flush_then_get() {
    auto cache = Cache::open(test_cfg());

    std::string key = "persist";
    std::string val = "on_disk_value";

    assert(cache.put(key.data(), key.size(), val.data(), val.size()));

    cache.flush();

    char buf[64] = {};
    size_t actual = 0;
    assert(cache.get(key.data(), key.size(), buf, sizeof(buf), &actual));
    assert(actual == val.size());
    assert(std::memcmp(buf, val.data(), val.size()) == 0);
}

static void test_many_keys() {
    auto cache = Cache::open(test_cfg(4096));

    for (int i = 0; i < 500; ++i) {
        std::string key = "key_" + std::to_string(i);
        std::string val = "value_" + std::to_string(i) + "_payload";
        assert(cache.put(key.data(), key.size(), val.data(), val.size()));
    }

    cache.flush();

    for (int i = 0; i < 500; ++i) {
        std::string key = "key_" + std::to_string(i);
        std::string expected = "value_" + std::to_string(i) + "_payload";
        char buf[128] = {};
        size_t actual = 0;
        bool found = cache.get(key.data(), key.size(), buf, sizeof(buf), &actual);
        assert(found);
        assert(actual == expected.size());
        assert(std::memcmp(buf, expected.data(), expected.size()) == 0);
    }
}

static void test_key_count() {
    auto cache = Cache::open(test_cfg());

    assert(cache.key_count() == 0);

    std::string k1 = "a", v1 = "1";
    std::string k2 = "b", v2 = "2";
    cache.put(k1.data(), k1.size(), v1.data(), v1.size());
    assert(cache.key_count() == 1);

    cache.put(k2.data(), k2.size(), v2.data(), v2.size());
    assert(cache.key_count() == 2);

    cache.put(k1.data(), k1.size(), v2.data(), v2.size());
    assert(cache.key_count() == 2);

    cache.remove(k1.data(), k1.size());
    assert(cache.key_count() == 1);
}

static void test_eviction_on_full_index() {
    // 128 total keys / 4 shards = 32 per shard.
    auto cache = Cache::open(test_cfg(128));

    for (int i = 0; i < 256; ++i) {
        std::string key = "ek_" + std::to_string(i);
        std::string val = "ev_" + std::to_string(i);
        assert(cache.put(key.data(), key.size(), val.data(), val.size()));
    }

    assert(cache.key_count() <= 128);

    std::string last_key = "ek_255";
    std::string last_val = "ev_255";
    char buf[32] = {};
    size_t actual = 0;
    assert(cache.get(last_key.data(), last_key.size(), buf, sizeof(buf), &actual));
    assert(actual == last_val.size());
    assert(std::memcmp(buf, last_val.data(), last_val.size()) == 0);
}

static void test_truncated_read() {
    auto cache = Cache::open(test_cfg());

    std::string key = "trunc";
    std::string val = "a_longer_value_string";

    cache.put(key.data(), key.size(), val.data(), val.size());

    char buf[5] = {};
    size_t actual = 0;
    assert(cache.get(key.data(), key.size(), buf, sizeof(buf), &actual));
    assert(actual == val.size());
    assert(std::memcmp(buf, val.data(), 5) == 0);
}

static void test_batch_get_memory() {
    auto cache = Cache::open(test_cfg(4096));

    for (int i = 0; i < 100; ++i) {
        std::string key = "bk_" + std::to_string(i);
        std::string val = "bv_" + std::to_string(i) + "_payload";
        assert(cache.put(key.data(), key.size(), val.data(), val.size()));
    }

    // Batch get 10 keys (from memtable)
    constexpr int BATCH = 10;
    std::vector<std::string> keys(BATCH);
    std::vector<std::string> expected(BATCH);
    std::vector<char> bufs(BATCH * 64, 0);
    std::vector<BatchGetRequest> reqs(BATCH);
    std::vector<BatchGetResult>  results(BATCH);

    for (int i = 0; i < BATCH; ++i) {
        keys[i] = "bk_" + std::to_string(i * 10);
        expected[i] = "bv_" + std::to_string(i * 10) + "_payload";
        reqs[i] = {keys[i].data(), keys[i].size(),
                   bufs.data() + i * 64, 64};
    }

    cache.get_batch(reqs.data(), results.data(), BATCH);

    for (int i = 0; i < BATCH; ++i) {
        assert(results[i].found);
        assert(results[i].actual_len == expected[i].size());
        assert(std::memcmp(bufs.data() + i * 64,
                           expected[i].data(), expected[i].size()) == 0);
    }
}

static void test_batch_get_disk() {
    auto cache = Cache::open(test_cfg(4096));

    for (int i = 0; i < 50; ++i) {
        std::string key = "dk_" + std::to_string(i);
        std::string val = "dv_" + std::to_string(i) + "_diskdata";
        assert(cache.put(key.data(), key.size(), val.data(), val.size()));
    }

    cache.flush();

    constexpr int BATCH = 5;
    std::vector<std::string> keys(BATCH);
    std::vector<std::string> expected(BATCH);
    std::vector<char> bufs(BATCH * 64, 0);
    std::vector<BatchGetRequest> reqs(BATCH);
    std::vector<BatchGetResult>  results(BATCH);

    for (int i = 0; i < BATCH; ++i) {
        keys[i] = "dk_" + std::to_string(i * 10);
        expected[i] = "dv_" + std::to_string(i * 10) + "_diskdata";
        reqs[i] = {keys[i].data(), keys[i].size(),
                   bufs.data() + i * 64, 64};
    }

    cache.get_batch(reqs.data(), results.data(), BATCH);

    for (int i = 0; i < BATCH; ++i) {
        assert(results[i].found);
        assert(results[i].actual_len == expected[i].size());
        assert(std::memcmp(bufs.data() + i * 64,
                           expected[i].data(), expected[i].size()) == 0);
    }
}

static void test_batch_get_mixed_hit_miss() {
    auto cache = Cache::open(test_cfg(4096));

    for (int i = 0; i < 20; ++i) {
        std::string key = "mk_" + std::to_string(i);
        std::string val = "mv_" + std::to_string(i);
        assert(cache.put(key.data(), key.size(), val.data(), val.size()));
    }

    constexpr int BATCH = 4;
    std::string k0 = "mk_0",  k1 = "mk_5", k2 = "nonexistent", k3 = "mk_19";
    char bufs[BATCH][32] = {};
    BatchGetRequest reqs[BATCH] = {
        {k0.data(), k0.size(), bufs[0], 32},
        {k1.data(), k1.size(), bufs[1], 32},
        {k2.data(), k2.size(), bufs[2], 32},
        {k3.data(), k3.size(), bufs[3], 32},
    };
    BatchGetResult results[BATCH] = {};

    cache.get_batch(reqs, results, BATCH);

    assert(results[0].found);
    assert(results[1].found);
    assert(!results[2].found);
    assert(results[3].found);
}

static void test_delete_manager_eviction() {
    // Ring large enough to hold all data; index small enough to force eviction.
    CacheConfig cfg{
        .device_path    = TEST_PATH,
        .ring_capacity  = 4 * 1024 * 1024,   // 4 MB total
        .memtable_size  = 256 * 1024,         // 256 KB total
        .index_capacity = 128,
        .num_shards     = 2,
    };
    auto cache = Cache::open(cfg);

    for (int i = 0; i < 200; ++i) {
        std::string key = "dm_" + std::to_string(i);
        std::string val = "val_" + std::to_string(i);
        assert(cache.put(key.data(), key.size(), val.data(), val.size()));
    }

    // Eviction should have pruned older entries.
    assert(cache.key_count() <= 128);

    // Recent keys should still be retrievable.
    std::string last_key = "dm_199";
    std::string last_val = "val_199";
    char buf[32] = {};
    size_t actual = 0;
    assert(cache.get(last_key.data(), last_key.size(), buf, sizeof(buf), &actual));
    assert(actual == last_val.size());
    assert(std::memcmp(buf, last_val.data(), last_val.size()) == 0);
}

// ─── main ────────────────────────────────────────────────────────────────────

int main() {
    printf("cache_test\n");

    RUN(test_put_get_small);
    RUN(test_put_get_large);
    RUN(test_get_miss);
    RUN(test_put_overwrite);
    RUN(test_remove);
    RUN(test_flush_then_get);
    RUN(test_many_keys);
    RUN(test_key_count);
    RUN(test_eviction_on_full_index);
    RUN(test_truncated_read);
    RUN(test_batch_get_memory);
    RUN(test_batch_get_disk);
    RUN(test_batch_get_mixed_hit_miss);
    RUN(test_delete_manager_eviction);

    printf("\n%d / %d tests passed.\n", tests_passed, tests_run);
    return (tests_passed == tests_run) ? 0 : 1;
}
