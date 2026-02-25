#include "flashringc/semaphore_pool.h"
#include "flashringc/shard_reactor.h"

#include <cassert>
#include <chrono>
#include <cstdio>
#include <cstring>
#include <string>
#include <thread>
#include <unistd.h>
#include <vector>

static const char* TEST_PATH = "/tmp/flashring_reactor_test.dat";
static int tests_run = 0;
static int tests_passed = 0;

static std::vector<uint8_t> to_bytes(const std::string& s) {
    return {s.begin(), s.end()};
}

static void cleanup() {
    std::string path = std::string(TEST_PATH) + ".0";
    ::unlink(path.c_str());
}

#define RUN(name)                                       \
    do {                                                \
        ++tests_run;                                    \
        printf("  %-45s ", #name);                      \
        cleanup();                                      \
        name();                                         \
        cleanup();                                      \
        ++tests_passed;                                 \
        printf("PASS\n");                               \
    } while (0)

static ShardReactor make_reactor(SemaphorePool* pool, uint32_t index_cap = 1024) {
    auto ring = RingDevice::open(std::string(TEST_PATH) + ".0",
                                 4 * 1024 * 1024);
    return ShardReactor(std::move(ring),
                        256 * 1024,   // memtable size
                        index_cap,
                        pool,
                        1024,         // queue capacity
                        64);          // uring depth
}

static Result submit_put(ShardReactor& reactor, SemaphorePool& pool,
                        const std::string& key, const std::string& val) {
    uint32_t slot = pool.acquire();
    Result result;
    Request req;
    req.type = OpType::Put;
    req.key = key;
    req.value = val;
    req.hash = hash_key(key.data(), key.size());
    req.result = &result;
    req.sem_slot = slot;
    req.batch_remaining = nullptr;
    bool ok = reactor.submit(std::move(req));
    assert(ok);
    pool.wait(slot);
    pool.release(slot);
    return result;
}

static Result submit_get(ShardReactor& reactor, SemaphorePool& pool,
                        const std::string& key) {
    uint32_t slot = pool.acquire();
    Result result;
    Request req;
    req.type = OpType::Get;
    req.key = key;
    req.hash = hash_key(key.data(), key.size());
    req.result = &result;
    req.sem_slot = slot;
    req.batch_remaining = nullptr;
    bool ok = reactor.submit(std::move(req));
    assert(ok);
    pool.wait(slot);
    pool.release(slot);
    return result;
}

static Result submit_del(ShardReactor& reactor, SemaphorePool& pool,
                        const std::string& key) {
    uint32_t slot = pool.acquire();
    Result result;
    Request req;
    req.type = OpType::Delete;
    req.key = key;
    req.hash = hash_key(key.data(), key.size());
    req.result = &result;
    req.sem_slot = slot;
    req.batch_remaining = nullptr;
    bool ok = reactor.submit(std::move(req));
    assert(ok);
    pool.wait(slot);
    pool.release(slot);
    return result;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

static void test_put_get_memtable_hit() {
    SemaphorePool pool(64);
    auto reactor = make_reactor(&pool);
    std::thread t([&]() { reactor.run(); });

    Result pr = submit_put(reactor, pool, "hello", "world");
    assert(pr.status == Status::Ok);

    Result gr = submit_get(reactor, pool, "hello");
    assert(gr.status == Status::Ok);
    assert(gr.value == to_bytes("world"));

    reactor.stop();
    t.join();
}

static void test_get_miss() {
    SemaphorePool pool(64);
    auto reactor = make_reactor(&pool);
    std::thread t([&]() { reactor.run(); });

    Result r = submit_get(reactor, pool, "nonexistent");
    assert(r.status == Status::NotFound);

    reactor.stop();
    t.join();
}

static void test_put_overwrite() {
    SemaphorePool pool(64);
    auto reactor = make_reactor(&pool);
    std::thread t([&]() { reactor.run(); });

    submit_put(reactor, pool, "key1", "val1");
    submit_put(reactor, pool, "key1", "val2_updated");

    Result r = submit_get(reactor, pool, "key1");
    assert(r.status == Status::Ok);
    assert(r.value == to_bytes("val2_updated"));

    reactor.stop();
    t.join();
}

static void test_delete() {
    SemaphorePool pool(64);
    auto reactor = make_reactor(&pool);
    std::thread t([&]() { reactor.run(); });

    submit_put(reactor, pool, "delme", "temp");

    Result r1 = submit_get(reactor, pool, "delme");
    assert(r1.status == Status::Ok);

    Result dr = submit_del(reactor, pool, "delme");
    assert(dr.status == Status::Ok);

    Result r2 = submit_get(reactor, pool, "delme");
    assert(r2.status == Status::NotFound);

    reactor.stop();
    t.join();
}

static void test_many_keys() {
    SemaphorePool pool(256);
    auto reactor = make_reactor(&pool, 4096);
    std::thread t([&]() { reactor.run(); });

    for (int i = 0; i < 200; ++i) {
        std::string key = "k_" + std::to_string(i);
        std::string val = "v_" + std::to_string(i) + "_payload";
        Result r = submit_put(reactor, pool, key, val);
        assert(r.status == Status::Ok);
    }

    for (int i = 0; i < 200; ++i) {
        std::string key = "k_" + std::to_string(i);
        std::string expected = "v_" + std::to_string(i) + "_payload";
        Result r = submit_get(reactor, pool, key);
        assert(r.status == Status::Ok);
        assert(r.value == to_bytes(expected));
    }

    reactor.stop();
    t.join();
}

static void test_eviction() {
    SemaphorePool pool(128);
    auto reactor = make_reactor(&pool, 64);
    std::thread t([&]() { reactor.run(); });

    for (int i = 0; i < 128; ++i) {
        std::string key = "ev_" + std::to_string(i);
        std::string val = "val_" + std::to_string(i);
        submit_put(reactor, pool, key, val);
    }

    assert(reactor.key_count() <= 64);

    Result r = submit_get(reactor, pool, "ev_127");
    assert(r.status == Status::Ok);
    assert(r.value == to_bytes("val_127"));

    reactor.stop();
    t.join();
}

static void test_multi_producer_stress() {
    SemaphorePool pool(512);
    auto reactor = make_reactor(&pool, 4096);
    std::thread reactor_thread([&]() { reactor.run(); });

    constexpr int NUM_PRODUCERS = 4;
    constexpr int ITEMS = 100;

    std::vector<std::thread> producers;
    for (int p = 0; p < NUM_PRODUCERS; ++p) {
        producers.emplace_back([&reactor, &pool, p]() {
            for (int i = 0; i < ITEMS; ++i) {
                std::string key = "p" + std::to_string(p) + "_k" + std::to_string(i);
                std::string val = "v_" + std::to_string(p * ITEMS + i);
                Result r = submit_put(reactor, pool, key, val);
                assert(r.status == Status::Ok);
            }
        });
    }

    for (auto& th : producers) th.join();

    for (int p = 0; p < NUM_PRODUCERS; ++p) {
        std::string key = "p" + std::to_string(p) + "_k0";
        std::string expected = "v_" + std::to_string(p * ITEMS);
        Result r = submit_get(reactor, pool, key);
        assert(r.status == Status::Ok);
        assert(r.value == to_bytes(expected));
    }

    reactor.stop();
    reactor_thread.join();
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

int main() {
    printf("test_shard_reactor\n");

    RUN(test_put_get_memtable_hit);
    RUN(test_get_miss);
    RUN(test_put_overwrite);
    RUN(test_delete);
    RUN(test_many_keys);
    RUN(test_eviction);
    RUN(test_multi_producer_stress);

    printf("\n%d / %d tests passed.\n", tests_passed, tests_run);
    return (tests_passed == tests_run) ? 0 : 1;
}
