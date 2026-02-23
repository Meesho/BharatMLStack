#pragma once

#include <atomic>
#include <condition_variable>
#include <cstddef>
#include <cstdint>
#include <mutex>
#include <new>
#include <type_traits>
#include <utility>

// ---------------------------------------------------------------------------
// MPSCQueue — bounded, lock-free, multi-producer / single-consumer queue.
//
// Capacity must be a power of two.
// Producers CAS on head_ (wait-free try_push).
// The single consumer advances tail_ (no contention).
//
// Each slot has a sequence number to distinguish full/empty without ABA.
// ---------------------------------------------------------------------------

#ifdef __cpp_lib_hardware_destructive_interference_size
    inline constexpr size_t kCacheLine = std::hardware_destructive_interference_size;
#else
    inline constexpr size_t kCacheLine = 64;
#endif

template <typename T>
class MPSCQueue {
public:
    explicit MPSCQueue(uint32_t capacity)
        : mask_(next_pow2(capacity) - 1),
          slots_(new Slot[mask_ + 1]) {
        for (uint32_t i = 0; i <= mask_; ++i)
            slots_[i].seq.store(i, std::memory_order_relaxed);
    }

    ~MPSCQueue() { delete[] slots_; }

    MPSCQueue(const MPSCQueue&) = delete;
    MPSCQueue& operator=(const MPSCQueue&) = delete;

    // Push from any thread. Returns false if the queue is full.
    bool try_push(T&& item) {
        uint32_t pos = head_.load(std::memory_order_relaxed);
        for (;;) {
            Slot& slot = slots_[pos & mask_];
            uint32_t seq = slot.seq.load(std::memory_order_acquire);
            int32_t diff = static_cast<int32_t>(seq) -
                           static_cast<int32_t>(pos);
            if (diff == 0) {
                if (head_.compare_exchange_weak(pos, pos + 1,
                        std::memory_order_relaxed))
                {
                    slot.data = std::move(item);
                    slot.seq.store(pos + 1, std::memory_order_release);
                    {
                        std::lock_guard<std::mutex> lk(mu_);
                        cv_.notify_one();
                    }
                    return true;
                }
            } else if (diff < 0) {
                return false;  // full
            } else {
                pos = head_.load(std::memory_order_relaxed);
            }
        }
    }

    // Pop from consumer thread only. Returns false if empty.
    bool try_pop(T& item) {
        Slot& slot = slots_[tail_ & mask_];
        uint32_t seq = slot.seq.load(std::memory_order_acquire);
        int32_t diff = static_cast<int32_t>(seq) -
                       static_cast<int32_t>(tail_ + 1);
        if (diff < 0)
            return false;  // empty
        item = std::move(slot.data);
        slot.seq.store(tail_ + mask_ + 1, std::memory_order_release);
        ++tail_;
        return true;
    }

    // Drain up to `max` items (consumer only). Returns count popped.
    size_t drain(T* out, size_t max) {
        size_t n = 0;
        while (n < max && try_pop(out[n]))
            ++n;
        return n;
    }

    // Blocking push — spin-retries try_push() until success.
    // Used by stop() for the shutdown sentinel which must not fail.
    void push(T&& item) {
        while (!try_push(std::move(item))) {}
    }

    // Blocking pop (consumer only). Fast path: lock-free try_pop.
    // Slow path: condvar wait. Only used in State 2 (idle reactor).
    bool pop_blocking(T& item) {
        if (try_pop(item))
            return true;
        std::unique_lock<std::mutex> lk(mu_);
        cv_.wait(lk, [&] { return try_pop(item); });
        return true;
    }

    uint32_t capacity() const { return mask_ + 1; }

private:
    struct Slot {
        std::atomic<uint32_t> seq{0};
        T data{};
    };

    static uint32_t next_pow2(uint32_t v) {
        if (v == 0) return 1;
        --v;
        v |= v >> 1;  v |= v >> 2;
        v |= v >> 4;  v |= v >> 8;
        v |= v >> 16;
        return v + 1;
    }

    const uint32_t mask_;
    Slot*          slots_;

    alignas(kCacheLine) std::atomic<uint32_t> head_{0};
    alignas(kCacheLine) uint32_t              tail_{0};

    std::mutex              mu_;
    std::condition_variable cv_;
};
