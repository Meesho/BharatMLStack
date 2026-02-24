#pragma once

#include <condition_variable>
#include <cstdint>
#include <memory>
#include <mutex>
#include <vector>
#include <atomic>
#include <stdexcept>

#if !defined(__APPLE__)
#include <semaphore.h>
#endif

class SemaphorePool {
public:
    explicit SemaphorePool(size_t capacity = 1024);
    ~SemaphorePool();

    SemaphorePool(const SemaphorePool&) = delete;
    SemaphorePool& operator=(const SemaphorePool&) = delete;

    uint32_t acquire();
    void release(uint32_t slot);

#if !defined(__APPLE__)
    sem_t* get(uint32_t slot);
#endif

    void post(uint32_t slot);
    void wait(uint32_t slot);

private:
#if defined(__APPLE__)
    struct Slot {
        std::mutex mtx;
        std::condition_variable cv;
        bool signaled = false;
    };
    std::unique_ptr<Slot[]> slots_;
    size_t capacity_ = 0;
#else
    std::vector<sem_t> sems_;
#endif
    std::vector<uint32_t> free_stack_;
    std::atomic<int32_t> free_top_;
};
