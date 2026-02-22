#include "io_engine.h"

#include <unistd.h>
#include <algorithm>

#if defined(__linux__) && defined(HAVE_IO_URING)
#include <liburing.h>
#endif

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

IoEngine::IoEngine(bool use_uring, uint32_t queue_depth)
    : queue_depth_(queue_depth) {
#if defined(__linux__) && defined(HAVE_IO_URING)
    if (use_uring) {
        auto* ring = new struct io_uring{};
        int ret = io_uring_queue_init(queue_depth_, ring, 0);
        if (ret == 0) {
            ring_ptr_ = ring;
            uring_ok_ = true;
        } else {
            delete ring;
        }
    }
#else
    (void)use_uring;
#endif
}

IoEngine::~IoEngine() {
#if defined(__linux__) && defined(HAVE_IO_URING)
    if (ring_ptr_) {
        io_uring_queue_exit(static_cast<struct io_uring*>(ring_ptr_));
        delete static_cast<struct io_uring*>(ring_ptr_);
    }
#endif
}

IoEngine::IoEngine(IoEngine&& o) noexcept
    : queue_depth_(o.queue_depth_), uring_ok_(o.uring_ok_) {
#if defined(__linux__) && defined(HAVE_IO_URING)
    ring_ptr_   = o.ring_ptr_;
    o.ring_ptr_ = nullptr;
#endif
    o.uring_ok_ = false;
}

IoEngine& IoEngine::operator=(IoEngine&& o) noexcept {
    if (this != &o) {
#if defined(__linux__) && defined(HAVE_IO_URING)
        if (ring_ptr_) {
            io_uring_queue_exit(static_cast<struct io_uring*>(ring_ptr_));
            delete static_cast<struct io_uring*>(ring_ptr_);
        }
        ring_ptr_   = o.ring_ptr_;
        o.ring_ptr_ = nullptr;
#endif
        queue_depth_ = o.queue_depth_;
        uring_ok_    = o.uring_ok_;
        o.uring_ok_  = false;
    }
    return *this;
}

// ---------------------------------------------------------------------------
// pread fallback
// ---------------------------------------------------------------------------

uint32_t IoEngine::read_batch_pread(ReadOp* ops, uint32_t count) {
    uint32_t ok_count = 0;
    for (uint32_t i = 0; i < count; ++i) {
        auto& op = ops[i];
        ssize_t n = ::pread(op.fd, op.buf, op.len,
                            static_cast<off_t>(op.offset));
        op.ok = (n == static_cast<ssize_t>(op.len));
        if (op.ok) ++ok_count;
    }
    return ok_count;
}

// ---------------------------------------------------------------------------
// io_uring path
// ---------------------------------------------------------------------------

#if defined(__linux__) && defined(HAVE_IO_URING)
uint32_t IoEngine::read_batch_uring(ReadOp* ops, uint32_t count) {
    auto* ring = static_cast<struct io_uring*>(ring_ptr_);
    uint32_t ok_count = 0;
    uint32_t pos = 0;

    while (pos < count) {
        uint32_t chunk = std::min(queue_depth_, count - pos);

        for (uint32_t i = 0; i < chunk; ++i) {
            auto& op = ops[pos + i];
            struct io_uring_sqe* sqe = io_uring_get_sqe(ring);
            if (!sqe) {
                // SQ full — submit what we have and retry
                io_uring_submit(ring);
                sqe = io_uring_get_sqe(ring);
                if (!sqe) {
                    op.ok = false;
                    continue;
                }
            }
            io_uring_prep_read(sqe, op.fd, op.buf, op.len, op.offset);
            io_uring_sqe_set_data(sqe, &op);
        }

        int submitted = io_uring_submit(ring);
        if (submitted < 0) {
            for (uint32_t i = 0; i < chunk; ++i)
                ops[pos + i].ok = false;
            pos += chunk;
            continue;
        }

        for (int i = 0; i < submitted; ++i) {
            struct io_uring_cqe* cqe = nullptr;
            int ret = io_uring_wait_cqe(ring, &cqe);
            if (ret < 0) continue;
            auto* op = static_cast<ReadOp*>(io_uring_cqe_get_data(cqe));
            if (op) {
                op->ok = (cqe->res == static_cast<int>(op->len));
                if (op->ok) ++ok_count;
            }
            io_uring_cqe_seen(ring, cqe);
        }

        pos += chunk;
    }

    return ok_count;
}
#endif

// ---------------------------------------------------------------------------
// Dispatch
// ---------------------------------------------------------------------------

uint32_t IoEngine::read_batch(ReadOp* ops, uint32_t count) {
    if (count == 0) return 0;
#if defined(__linux__) && defined(HAVE_IO_URING)
    if (uring_ok_)
        return read_batch_uring(ops, count);
#endif
    return read_batch_pread(ops, count);
}
