#include "flashringc/mpsc_queue.h"

#include <atomic>
#include <cassert>
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
    assert(!q.try_pop(val));

    assert(q.try_push(42));
    assert(q.try_pop(val));
    assert(val == 42);
    assert(!q.try_pop(val));
}

static void test_fill_and_drain() {
    MPSCQueue<int> q(8);
    for (int i = 0; i < 8; ++i)
        assert(q.try_push(std::move(i)));

    // Queue is full (capacity rounded to 8).
    int overflow = 99;
    assert(!q.try_push(std::move(overflow)));

    for (int i = 0; i < 8; ++i) {
        int val = -1;
        assert(q.try_pop(val));
        assert(val == i);
    }

    int val = -1;
    assert(!q.try_pop(val));
}

static void test_drain_batch() {
    MPSCQueue<int> q(16);
    for (int i = 0; i < 10; ++i) {
        int v = i;
        assert(q.try_push(std::move(v)));
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

    // Drain all items — verify we got exactly TOTAL.
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
    assert(q.try_push(MoveOnly(42)));
    MoveOnly out;
    assert(q.try_pop(out));
    assert(out.val == 42);
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

int main() {
    printf("test_mpsc_queue\n");

    RUN(test_single_push_pop);
    RUN(test_fill_and_drain);
    RUN(test_drain_batch);
    RUN(test_power_of_two_rounding);
    RUN(test_multi_producer);
    RUN(test_move_only_type);

    printf("\n%d / %d tests passed.\n", tests_passed, tests_run);
    return (tests_passed == tests_run) ? 0 : 1;
}
