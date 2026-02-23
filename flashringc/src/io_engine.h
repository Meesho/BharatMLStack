#pragma once

#include <cstddef>
#include <cstdint>
#include <mutex>

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

    // Non-blocking: submit only, then call reap_ready() later. io_uring only.
    void submit_only(ReadOp* ops, uint32_t count);
    // Non-blocking reap; returns number of completions processed. Sets op->ok.
    uint32_t reap_ready(uint32_t max_reap = 0);

    bool uring_enabled() const { return uring_ok_; }

private:
    uint32_t read_batch_pread(ReadOp* ops, uint32_t count);

#if defined(__linux__) && defined(HAVE_IO_URING)
    uint32_t read_batch_uring(ReadOp* ops, uint32_t count);
    void     submit_only_uring(ReadOp* ops, uint32_t count);
    uint32_t reap_ready_uring(uint32_t max_reap);
    void*    ring_ptr_   = nullptr;
#endif

    std::mutex mu_;           // serializes submit/wait; io_uring is not thread-safe
    uint32_t   queue_depth_ = 64;
    bool       uring_ok_    = false;
};
