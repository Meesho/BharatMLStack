#include "flashringc/mpsc_queue.h"

#include <atomic>
#include <cassert>
#include <chrono>
#include <cstdio>
#include <thread>
#include <vector>

static int tests_run = 0;
static int tests_passed = 0;

#define RUN(name)                                       \
    do {                                                \
        ++tests_run;                                    \
        printf("  %-45s ", #name);                      \
        name();                                         \
        ++tests_passed;                                 \
        printf("PASS\n");                               \
    } while (0)

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

static void test_single_push_pop() {
    MPSCQueue<int> q(4);
    int val = 0;
    bool ok;

    ok = q.try_pop(val);
    assert(!ok);

    ok = q.try_push(42);
    assert(ok);

    ok = q.try_pop(val);
    assert(ok);
    assert(val == 42);

    ok = q.try_pop(val);
    assert(!ok);
}

static void test_fill_and_drain() {
    MPSCQueue<int> q(8);
    for (int i = 0; i < 8; ++i) {
        bool ok = q.try_push(std::move(i));
        assert(ok);
    }

    int overflow = 99;
    bool full = q.try_push(std::move(overflow));
    assert(!full);

    for (int i = 0; i < 8; ++i) {
        int val = -1;
        bool ok = q.try_pop(val);
        assert(ok);
        assert(val == i);
    }

    int val = -1;
    bool ok = q.try_pop(val);
    assert(!ok);
}

static void test_drain_batch() {
    MPSCQueue<int> q(16);
    for (int i = 0; i < 10; ++i) {
        int v = i;
        bool ok = q.try_push(std::move(v));
        assert(ok);
    }

    int buf[16] = {};
    size_t n = q.drain(buf, 5);
    assert(n == 5);
    for (int i = 0; i < 5; ++i)
        assert(buf[i] == i);

    n = q.drain(buf, 16);
    assert(n == 5);
    for (int i = 0; i < 5; ++i)
        assert(buf[i] == i + 5);

    n = q.drain(buf, 16);
    assert(n == 0);
}

static void test_power_of_two_rounding() {
    MPSCQueue<int> q(3);
    assert(q.capacity() == 4);

    MPSCQueue<int> q2(1);
    assert(q2.capacity() == 1);

    MPSCQueue<int> q3(5);
    assert(q3.capacity() == 8);
}

static void test_multi_producer() {
    constexpr int NUM_PRODUCERS = 4;
    constexpr int ITEMS_PER_PRODUCER = 10000;
    constexpr int TOTAL = NUM_PRODUCERS * ITEMS_PER_PRODUCER;

    MPSCQueue<int> q(TOTAL * 2);
    std::atomic<int> push_failures{0};

    std::vector<std::thread> producers;
    for (int p = 0; p < NUM_PRODUCERS; ++p) {
        producers.emplace_back([&q, &push_failures, p]() {
            for (int i = 0; i < ITEMS_PER_PRODUCER; ++i) {
                int val = p * ITEMS_PER_PRODUCER + i;
                if (!q.try_push(std::move(val)))
                    push_failures.fetch_add(1, std::memory_order_relaxed);
            }
        });
    }

    for (auto& t : producers)
        t.join();

    assert(push_failures.load() == 0);

    std::vector<bool> seen(TOTAL, false);
    int val = 0;
    int count = 0;
    while (q.try_pop(val)) {
        assert(val >= 0 && val < TOTAL);
        assert(!seen[val]);
        seen[val] = true;
        ++count;
    }
    assert(count == TOTAL);
}

static void test_move_only_type() {
    struct MoveOnly {
        int val = 0;
        MoveOnly() = default;
        explicit MoveOnly(int v) : val(v) {}
        MoveOnly(MoveOnly&& o) noexcept : val(o.val) { o.val = -1; }
        MoveOnly& operator=(MoveOnly&& o) noexcept { val = o.val; o.val = -1; return *this; }
        MoveOnly(const MoveOnly&) = delete;
        MoveOnly& operator=(const MoveOnly&) = delete;
    };

    MPSCQueue<MoveOnly> q(4);
    bool ok;

    ok = q.try_push(MoveOnly(42));
    assert(ok);

    MoveOnly out;
    ok = q.try_pop(out);
    assert(ok);
    assert(out.val == 42);
}

static void test_pop_blocking() {
    MPSCQueue<int> q(4);

    std::atomic<bool> popped{false};
    int result = -1;

    std::thread consumer([&]() {
        q.pop_blocking(result);
        popped.store(true, std::memory_order_release);
    });

    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    assert(!popped.load(std::memory_order_acquire));

    int val = 77;
    bool ok = q.try_push(std::move(val));
    assert(ok);

    consumer.join();
    assert(popped.load());
    assert(result == 77);
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

int main() {
    setbuf(stdout, NULL);
    printf("test_mpsc_queue\n");

    RUN(test_single_push_pop);
    RUN(test_fill_and_drain);
    RUN(test_drain_batch);
    RUN(test_power_of_two_rounding);
    RUN(test_multi_producer);
    RUN(test_move_only_type);
    RUN(test_pop_blocking);

    printf("\n%d / %d tests passed.\n", tests_passed, tests_run);
    return (tests_passed == tests_run) ? 0 : 1;
}
