#include "flashringc/semaphore_pool.h"

#include <thread>

#if defined(__x86_64__) || defined(_M_X64)
#include <immintrin.h>
#endif

SemaphorePool::SemaphorePool(size_t capacity) {
    if (capacity == 0)
        throw std::invalid_argument("SemaphorePool capacity must be > 0");
#if defined(__APPLE__)
    capacity_ = capacity;
    slots_ = std::make_unique<Slot[]>(capacity);
#else
    sems_.resize(capacity);
    for (size_t i = 0; i < capacity; ++i) {
        if (sem_init(&sems_[i], 0, 0) != 0)
            throw std::runtime_error("sem_init failed");
    }
#endif
    free_stack_.resize(capacity);

    free_top_.store(0, std::memory_order_relaxed);
    for (size_t i = 0; i < capacity; ++i) {
        free_stack_[i] = (i + 1 < capacity) ? static_cast<uint32_t>(i + 1) : static_cast<uint32_t>(-1);
    }
}

SemaphorePool::~SemaphorePool() {
#if !defined(__APPLE__)
    for (sem_t& s : sems_)
        sem_destroy(&s);
#endif
}

uint32_t SemaphorePool::acquire() {
    int32_t top;
    for (;;) {
        top = free_top_.load(std::memory_order_acquire);
        if (top < 0) {
#if defined(__x86_64__) || defined(_M_X64)
            _mm_pause();
#else
            std::this_thread::yield();
#endif
            continue;
        }
        uint32_t next = (free_stack_[static_cast<size_t>(top)] == static_cast<uint32_t>(-1))
                            ? static_cast<uint32_t>(-1)
                            : free_stack_[static_cast<size_t>(top)];
        int32_t next_i = (next == static_cast<uint32_t>(-1)) ? -1 : static_cast<int32_t>(next);
        if (free_top_.compare_exchange_weak(top, next_i, std::memory_order_acq_rel, std::memory_order_acquire))
            return static_cast<uint32_t>(top);
    }
}

void SemaphorePool::release(uint32_t slot) {
    int32_t prev = free_top_.load(std::memory_order_acquire);
    for (;;) {
        free_stack_[slot] = (prev == -1) ? static_cast<uint32_t>(-1) : static_cast<uint32_t>(prev);
        if (free_top_.compare_exchange_weak(prev, static_cast<int32_t>(slot),
                                            std::memory_order_release,
                                            std::memory_order_relaxed))
            return;
    }
}

#if !defined(__APPLE__)
sem_t* SemaphorePool::get(uint32_t slot) {
    return &sems_[slot];
}
#endif

void SemaphorePool::post(uint32_t slot) {
#if defined(__APPLE__)
    {
        std::lock_guard<std::mutex> lock(slots_[static_cast<size_t>(slot)].mtx);
        slots_[static_cast<size_t>(slot)].signaled = true;
    }
    slots_[static_cast<size_t>(slot)].cv.notify_one();
#else
    sem_post(&sems_[slot]);
#endif
}

void SemaphorePool::wait(uint32_t slot) {
#if defined(__APPLE__)
    size_t s = static_cast<size_t>(slot);
    std::unique_lock<std::mutex> lock(slots_[s].mtx);
    slots_[s].cv.wait(lock, [this, s]() { return slots_[s].signaled; });
    slots_[s].signaled = false;
#else
    sem_wait(&sems_[slot]);
#endif
}
