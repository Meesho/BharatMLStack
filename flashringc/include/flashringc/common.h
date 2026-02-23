#pragma once

#include <cstddef>
#include <cstdint>
#include <cstdlib>
#include <cstring>
#include <utility>

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

constexpr size_t   kBlockSize         = 4096;
constexpr uint32_t kDefaultQueueDepth = 1024;
constexpr int      kMaxInboxBatch     = 64;

// ---------------------------------------------------------------------------
// Status
// ---------------------------------------------------------------------------

enum class Status : uint8_t { Ok, NotFound, Error };

// ---------------------------------------------------------------------------
// Hash128
// ---------------------------------------------------------------------------

struct Hash128 {
    uint64_t lo;
    uint64_t hi;
};

Hash128 hash_key(const void* key, size_t len);

// ---------------------------------------------------------------------------
// AlignedBuffer — RAII wrapper for block-aligned memory (O_DIRECT)
// ---------------------------------------------------------------------------

class AlignedBuffer {
public:
    AlignedBuffer() = default;

    static AlignedBuffer allocate(size_t size) {
        size_t aligned = align_up(size);
        void* ptr = nullptr;
        if (posix_memalign(&ptr, kBlockSize, aligned) != 0)
            return {};
        std::memset(ptr, 0, aligned);
        return AlignedBuffer(ptr, aligned);
    }

    ~AlignedBuffer() { std::free(data_); }

    AlignedBuffer(AlignedBuffer&& o) noexcept
        : data_(o.data_), size_(o.size_) {
        o.data_ = nullptr;
        o.size_ = 0;
    }

    AlignedBuffer& operator=(AlignedBuffer&& o) noexcept {
        if (this != &o) {
            std::free(data_);
            data_ = o.data_;
            size_ = o.size_;
            o.data_ = nullptr;
            o.size_ = 0;
        }
        return *this;
    }

    AlignedBuffer(const AlignedBuffer&) = delete;
    AlignedBuffer& operator=(const AlignedBuffer&) = delete;

    void*          data()       { return data_; }
    const void*    data() const { return data_; }
    uint8_t*       bytes()       { return static_cast<uint8_t*>(data_); }
    const uint8_t* bytes() const { return static_cast<const uint8_t*>(data_); }
    size_t size()  const { return size_; }
    bool   valid() const { return data_ != nullptr; }

    static constexpr size_t align_up(size_t n) {
        return (n + kBlockSize - 1) & ~(kBlockSize - 1);
    }

private:
    AlignedBuffer(void* d, size_t s) : data_(d), size_(s) {}
    void*  data_ = nullptr;
    size_t size_ = 0;
};
