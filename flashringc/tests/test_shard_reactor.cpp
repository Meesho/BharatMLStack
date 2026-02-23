#include "flashringc/shard_reactor.h"

#include <cassert>
#include <chrono>
#include <cstdio>
#include <cstring>
#include <future>
#include <string>
#include <thread>
#include <unistd.h>
#include <vector>

static const char* TEST_PATH = "/tmp/flashring_reactor_test.dat";
static int tests_run = 0;
static int tests_passed = 0;

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

static ShardReactor make_reactor(uint32_t index_cap = 1024) {
    auto ring = RingDevice::open(std::string(TEST_PATH) + ".0",
                                 4 * 1024 * 1024);
    return ShardReactor(std::move(ring),
                        256 * 1024,   // memtable size
                        index_cap,
                        1024,         // queue capacity
                        64);          // uring depth
}

// Helper: submit a request and return its future.
static std::future<Result> submit_put(ShardReactor& reactor,
                                      const std::string& key,
                                      const std::string& val) {
    Request req;
    req.type  = OpType::Put;
    req.key   = key;
    req.value = val;
    req.hash  = hash_key(key.data(), key.size());
    auto fut  = req.promise.get_future();
    assert(reactor.submit(std::move(req)));
    return fut;
}

static std::future<Result> submit_get(ShardReactor& reactor,
                                      const std::string& key) {
    Request req;
    req.type = OpType::Get;
    req.key  = key;
    req.hash = hash_key(key.data(), key.size());
    auto fut = req.promise.get_future();
    assert(reactor.submit(std::move(req)));
    return fut;
}

static std::future<Result> submit_del(ShardReactor& reactor,
                                      const std::string& key) {
    Request req;
    req.type = OpType::Delete;
    req.key  = key;
    req.hash = hash_key(key.data(), key.size());
    auto fut = req.promise.get_future();
    assert(reactor.submit(std::move(req)));
    return fut;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

static void test_put_get_memtable_hit() {
    auto reactor = make_reactor();
    std::thread t([&]() { reactor.run(); });

    auto put_fut = submit_put(reactor, "hello", "world");
    Result pr = put_fut.get();
    assert(pr.status == Status::Ok);

    auto get_fut = submit_get(reactor, "hello");
    Result gr = get_fut.get();
    assert(gr.status == Status::Ok);
    assert(gr.value == "world");

    reactor.stop();
    t.join();
}

static void test_get_miss() {
    auto reactor = make_reactor();
    std::thread t([&]() { reactor.run(); });

    auto fut = submit_get(reactor, "nonexistent");
    Result r = fut.get();
    assert(r.status == Status::NotFound);

    reactor.stop();
    t.join();
}

static void test_put_overwrite() {
    auto reactor = make_reactor();
    std::thread t([&]() { reactor.run(); });

    submit_put(reactor, "key1", "val1").get();
    submit_put(reactor, "key1", "val2_updated").get();

    auto r = submit_get(reactor, "key1").get();
    assert(r.status == Status::Ok);
    assert(r.value == "val2_updated");

    reactor.stop();
    t.join();
}

static void test_delete() {
    auto reactor = make_reactor();
    std::thread t([&]() { reactor.run(); });

    submit_put(reactor, "delme", "temp").get();

    auto r1 = submit_get(reactor, "delme").get();
    assert(r1.status == Status::Ok);

    auto dr = submit_del(reactor, "delme").get();
    assert(dr.status == Status::Ok);

    auto r2 = submit_get(reactor, "delme").get();
    assert(r2.status == Status::NotFound);

    reactor.stop();
    t.join();
}

static void test_many_keys() {
    auto reactor = make_reactor(4096);
    std::thread t([&]() { reactor.run(); });

    for (int i = 0; i < 200; ++i) {
        std::string key = "k_" + std::to_string(i);
        std::string val = "v_" + std::to_string(i) + "_payload";
        auto r = submit_put(reactor, key, val).get();
        assert(r.status == Status::Ok);
    }

    for (int i = 0; i < 200; ++i) {
        std::string key = "k_" + std::to_string(i);
        std::string expected = "v_" + std::to_string(i) + "_payload";
        auto r = submit_get(reactor, key).get();
        assert(r.status == Status::Ok);
        assert(r.value == expected);
    }

    reactor.stop();
    t.join();
}

static void test_eviction() {
    auto reactor = make_reactor(64);
    std::thread t([&]() { reactor.run(); });

    for (int i = 0; i < 128; ++i) {
        std::string key = "ev_" + std::to_string(i);
        std::string val = "val_" + std::to_string(i);
        submit_put(reactor, key, val).get();
    }

    assert(reactor.key_count() <= 64);

    // Recent keys should still be accessible.
    auto r = submit_get(reactor, "ev_127").get();
    assert(r.status == Status::Ok);
    assert(r.value == "val_127");

    reactor.stop();
    t.join();
}

static void test_multi_producer_stress() {
    auto reactor = make_reactor(4096);
    std::thread reactor_thread([&]() { reactor.run(); });

    constexpr int NUM_PRODUCERS = 4;
    constexpr int ITEMS = 100;

    std::vector<std::thread> producers;
    for (int p = 0; p < NUM_PRODUCERS; ++p) {
        producers.emplace_back([&reactor, p]() {
            for (int i = 0; i < ITEMS; ++i) {
                std::string key = "p" + std::to_string(p) + "_k" + std::to_string(i);
                std::string val = "v_" + std::to_string(p * ITEMS + i);
                auto r = submit_put(reactor, key, val).get();
                assert(r.status == Status::Ok);
            }
        });
    }

    for (auto& t : producers) t.join();

    // Verify some keys.
    for (int p = 0; p < NUM_PRODUCERS; ++p) {
        std::string key = "p" + std::to_string(p) + "_k0";
        std::string expected = "v_" + std::to_string(p * ITEMS);
        auto r = submit_get(reactor, key).get();
        assert(r.status == Status::Ok);
        assert(r.value == expected);
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
