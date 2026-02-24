#include "flashringc/semaphore_pool.h"

#include <cassert>
#include <cstdio>
#include <thread>
#include <vector>

static int tests_run = 0;
static int tests_passed = 0;

#define RUN(name)                                   \
    do {                                            \
        ++tests_run;                                \
        printf("  %-45s ", #name);                  \
        name();                                     \
        ++tests_passed;                             \
        printf("PASS\n");                           \
    } while (0)

static void test_acquire_release() {
    SemaphorePool pool(8);
    std::vector<uint32_t> slots;
    for (int i = 0; i < 8; ++i) {
        uint32_t s = pool.acquire();
        slots.push_back(s);
    }
    for (uint32_t s : slots)
        pool.release(s);
    for (int i = 0; i < 8; ++i) {
        uint32_t s = pool.acquire();
        (void)s;
    }
}

static void test_post_wait() {
    SemaphorePool pool(4);
    uint32_t slot = pool.acquire();
    int done = 0;
    std::thread t([&]() {
        done = 1;
        pool.post(slot);
    });
    pool.wait(slot);
    t.join();
    assert(done == 1);
    pool.release(slot);
}

static void test_pool_exhaustion() {
    SemaphorePool pool(2);
    uint32_t a = pool.acquire();
    uint32_t b = pool.acquire();
    std::atomic<bool> released{false};
    std::thread t([&]() {
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
        pool.release(a);
        released = true;
    });
    uint32_t c = pool.acquire();  // spins until a is released
    assert(released);
    assert(c == a || c == b);
    t.join();
    pool.release(b);
    pool.release(c);
}

static void test_concurrent_acquire_release() {
    SemaphorePool pool(64);
    std::atomic<int> count{0};
    std::vector<std::thread> threads;
    for (int t = 0; t < 8; ++t) {
        threads.emplace_back([&, t]() {
            for (int i = 0; i < 100; ++i) {
                uint32_t slot = pool.acquire();
                count++;
                pool.release(slot);
                count--;
            }
        });
    }
    for (auto& th : threads) th.join();
    assert(count.load() == 0);
}

int main() {
    printf("test_semaphore_pool\n");
    RUN(test_acquire_release);
    RUN(test_post_wait);
    RUN(test_pool_exhaustion);
    RUN(test_concurrent_acquire_release);
    printf("\n%d / %d tests passed.\n", tests_passed, tests_run);
    return (tests_passed == tests_run) ? 0 : 1;
}
