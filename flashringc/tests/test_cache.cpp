#include "flashringc/cache.h"

#include <cassert>
#include <cstdio>
#include <cstring>
#include <string>
#include <unistd.h>
#include <vector>

static const char* TEST_PATH = "/tmp/flashring_cache_v2_test.dat";
static constexpr uint32_t TEST_SHARDS = 4;

static int tests_run = 0;
static int tests_passed = 0;

static std::vector<uint8_t> to_bytes(const std::string& s) {
    return {s.begin(), s.end()};
}

static void cleanup_shard_files() {
    for (uint32_t i = 0; i < 256; ++i) {
        std::string path = std::string(TEST_PATH) + "." + std::to_string(i);
        ::unlink(path.c_str());
    }
}

#define RUN(name)                                       \
    do {                                                \
        ++tests_run;                                    \
        printf("  %-45s ", #name);                      \
        cleanup_shard_files();                          \
        name();                                         \
        cleanup_shard_files();                          \
        ++tests_passed;                                 \
        printf("PASS\n");                               \
    } while (0)

static CacheConfig test_cfg(uint32_t index_cap = 1024) {
    return {
        .device_path      = TEST_PATH,
        .ring_capacity    = 16 * 1024 * 1024,
        .memtable_size    = 1024 * 1024,
        .index_capacity   = index_cap,
        .num_shards       = TEST_SHARDS,
        .queue_capacity   = 1024,
        .uring_queue_depth = 64,
    };
}

// ---------------------------------------------------------------------------
// Tests (using sync wrappers for simplicity)
// ---------------------------------------------------------------------------

static void test_put_get_small() {
    auto cache = Cache::open(test_cfg());

    std::string key = "hello";
    std::string val = "world";

    assert(cache.put_sync(key.data(), key.size(), val.data(), val.size()));

    char buf[64] = {};
    size_t actual = 0;
    assert(cache.get_sync(key.data(), key.size(), buf, sizeof(buf), &actual));
    assert(actual == val.size());
    assert(std::memcmp(buf, val.data(), val.size()) == 0);
}

static void test_put_get_large() {
    auto cache = Cache::open(test_cfg());

    std::string key = "large_key";
    std::vector<uint8_t> val(50'000);
    for (size_t i = 0; i < val.size(); ++i)
        val[i] = static_cast<uint8_t>((i * 7 + 13) & 0xFF);

    assert(cache.put_sync(key.data(), key.size(), val.data(), val.size()));

    std::vector<uint8_t> out(val.size());
    size_t actual = 0;
    assert(cache.get_sync(key.data(), key.size(), out.data(), out.size(), &actual));
    assert(actual == val.size());
    assert(std::memcmp(out.data(), val.data(), val.size()) == 0);
}

static void test_get_miss() {
    auto cache = Cache::open(test_cfg());

    std::string key = "nonexistent";
    char buf[16];
    size_t actual = 0;
    assert(!cache.get_sync(key.data(), key.size(), buf, sizeof(buf), &actual));
}

static void test_put_overwrite() {
    auto cache = Cache::open(test_cfg());

    std::string key = "mykey";
    std::string val1 = "first_value";
    std::string val2 = "second_value_longer";

    assert(cache.put_sync(key.data(), key.size(), val1.data(), val1.size()));
    assert(cache.put_sync(key.data(), key.size(), val2.data(), val2.size()));

    char buf[64] = {};
    size_t actual = 0;
    assert(cache.get_sync(key.data(), key.size(), buf, sizeof(buf), &actual));
    assert(actual == val2.size());
    assert(std::memcmp(buf, val2.data(), val2.size()) == 0);
}

static void test_remove() {
    auto cache = Cache::open(test_cfg());

    std::string key = "to_delete";
    std::string val = "temporary";

    assert(cache.put_sync(key.data(), key.size(), val.data(), val.size()));

    char buf[32];
    size_t actual = 0;
    assert(cache.get_sync(key.data(), key.size(), buf, sizeof(buf), &actual));

    assert(cache.remove_sync(key.data(), key.size()));
    assert(!cache.get_sync(key.data(), key.size(), buf, sizeof(buf), &actual));
}

static void test_many_keys() {
    auto cache = Cache::open(test_cfg(4096));

    for (int i = 0; i < 500; ++i) {
        std::string key = "key_" + std::to_string(i);
        std::string val = "value_" + std::to_string(i) + "_payload";
        assert(cache.put_sync(key.data(), key.size(), val.data(), val.size()));
    }

    for (int i = 0; i < 500; ++i) {
        std::string key = "key_" + std::to_string(i);
        std::string expected = "value_" + std::to_string(i) + "_payload";
        char buf[128] = {};
        size_t actual = 0;
        bool found = cache.get_sync(key.data(), key.size(), buf, sizeof(buf), &actual);
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
    cache.put_sync(k1.data(), k1.size(), v1.data(), v1.size());
    // Give reactor a moment to process.
    std::this_thread::sleep_for(std::chrono::milliseconds(5));
    assert(cache.key_count() == 1);

    cache.put_sync(k2.data(), k2.size(), v2.data(), v2.size());
    std::this_thread::sleep_for(std::chrono::milliseconds(5));
    assert(cache.key_count() == 2);

    cache.put_sync(k1.data(), k1.size(), v2.data(), v2.size());
    std::this_thread::sleep_for(std::chrono::milliseconds(5));
    assert(cache.key_count() == 2);

    cache.remove_sync(k1.data(), k1.size());
    std::this_thread::sleep_for(std::chrono::milliseconds(5));
    assert(cache.key_count() == 1);
}

static void test_eviction_on_full_index() {
    auto cache = Cache::open(test_cfg(128));

    for (int i = 0; i < 256; ++i) {
        std::string key = "ek_" + std::to_string(i);
        std::string val = "ev_" + std::to_string(i);
        assert(cache.put_sync(key.data(), key.size(), val.data(), val.size()));
    }

    assert(cache.key_count() <= 128);

    std::string last_key = "ek_255";
    std::string last_val = "ev_255";
    char buf[32] = {};
    size_t actual = 0;
    assert(cache.get_sync(last_key.data(), last_key.size(), buf, sizeof(buf), &actual));
    assert(actual == last_val.size());
    assert(std::memcmp(buf, last_val.data(), last_val.size()) == 0);
}

static void test_truncated_read() {
    auto cache = Cache::open(test_cfg());

    std::string key = "trunc";
    std::string val = "a_longer_value_string";

    cache.put_sync(key.data(), key.size(), val.data(), val.size());

    char buf[5] = {};
    size_t actual = 0;
    assert(cache.get_sync(key.data(), key.size(), buf, sizeof(buf), &actual));
    assert(actual == val.size());
    assert(std::memcmp(buf, val.data(), 5) == 0);
}

static void test_async_api() {
    auto cache = Cache::open(test_cfg());

    Result pr = cache.put("async_key", "async_value");
    assert(pr.status == Status::Ok);

    Result gr = cache.get("async_key");
    assert(gr.status == Status::Ok);
    assert(gr.value == to_bytes("async_value"));

    Result dr = cache.del("async_key");
    assert(dr.status == Status::Ok);

    Result mr = cache.get("async_key");
    assert(mr.status == Status::NotFound);
}

static void test_batch_api() {
    auto cache = Cache::open(test_cfg(4096));

    for (int i = 0; i < 50; ++i) {
        std::string key = "bk_" + std::to_string(i);
        std::string val = "bv_" + std::to_string(i) + "_payload";
        cache.put_sync(key.data(), key.size(), val.data(), val.size());
    }

    constexpr int BATCH = 5;
    std::vector<std::string> key_strs(BATCH);
    std::vector<std::string_view> keys(BATCH);
    for (int i = 0; i < BATCH; ++i) {
        key_strs[i] = "bk_" + std::to_string(i * 10);
        keys[i] = key_strs[i];
    }

    std::vector<Result> results = cache.batch_get(keys);
    assert(results.size() == BATCH);

    for (int i = 0; i < BATCH; ++i) {
        assert(results[i].status == Status::Ok);
        std::string expected = "bv_" + std::to_string(i * 10) + "_payload";
        assert(results[i].value == to_bytes(expected));
    }
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

int main() {
    printf("test_cache (reactor-based)\n");

    RUN(test_put_get_small);
    RUN(test_put_get_large);
    RUN(test_get_miss);
    RUN(test_put_overwrite);
    RUN(test_remove);
    RUN(test_many_keys);
    RUN(test_key_count);
    RUN(test_eviction_on_full_index);
    RUN(test_truncated_read);
    RUN(test_async_api);
    RUN(test_batch_api);

    printf("\n%d / %d tests passed.\n", tests_passed, tests_run);
    return (tests_passed == tests_run) ? 0 : 1;
}
