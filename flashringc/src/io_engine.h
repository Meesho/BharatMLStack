#pragma once

#include <cstddef>
#include <cstdint>

struct ReadOp {
    void*    buf;
    size_t   len;
    uint64_t offset;
    int      fd;
    bool     ok = false;
};

class IoEngine {
public:
    explicit IoEngine(bool use_uring = false, uint32_t queue_depth = 64);
    ~IoEngine();

    IoEngine(IoEngine&& o) noexcept;
    IoEngine& operator=(IoEngine&& o) noexcept;
    IoEngine(const IoEngine&) = delete;
    IoEngine& operator=(const IoEngine&) = delete;

    uint32_t read_batch(ReadOp* ops, uint32_t count);

    bool uring_enabled() const { return uring_ok_; }

private:
    uint32_t read_batch_pread(ReadOp* ops, uint32_t count);

#if defined(__linux__) && defined(HAVE_IO_URING)
    uint32_t read_batch_uring(ReadOp* ops, uint32_t count);
    void*    ring_ptr_   = nullptr;
#endif

    uint32_t queue_depth_ = 64;
    bool     uring_ok_    = false;
};
