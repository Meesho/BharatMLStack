#include "flashringc/shard_reactor.h"
#include "flashringc/semaphore_pool.h"

#include <chrono>
#include <cstring>
#include <vector>


// ---------------------------------------------------------------------------
// Construction / Destruction
// ---------------------------------------------------------------------------

ShardReactor::ShardReactor(RingDevice ring, size_t mt_size, uint32_t index_cap,
                           SemaphorePool* sem_pool,
                           uint32_t queue_capacity, uint32_t uring_depth)
    : ring_(std::move(ring)),
      memtables_(ring_, mt_size),
      index_(index_cap),
      delete_mgr_(ring_, index_),
      sem_pool_(sem_pool),
      inbox_(queue_capacity),
      uring_depth_(uring_depth)
{
#if defined(__linux__) && defined(HAVE_IO_URING)
    std::memset(&uring_, 0, sizeof(uring_));
    if (io_uring_queue_init(uring_depth_, &uring_, 0) == 0)
        uring_ok_ = true;
#endif
}

ShardReactor::~ShardReactor() {
#if defined(__linux__) && defined(HAVE_IO_URING)
    if (uring_ok_)
        io_uring_queue_exit(&uring_);
#endif
}

// ---------------------------------------------------------------------------
// submit — called from any producer thread
// ---------------------------------------------------------------------------

bool ShardReactor::submit(Request&& req) {
    return inbox_.try_push(std::move(req));
}

// ---------------------------------------------------------------------------
// stop
// ---------------------------------------------------------------------------

void ShardReactor::stop() {
    running_.store(false, std::memory_order_release);
    Request sentinel;
    sentinel.type = OpType::Shutdown;
    inbox_.push(std::move(sentinel));
}

// ---------------------------------------------------------------------------
// now_delta — 16-bit seconds-since-epoch for last_access
// ---------------------------------------------------------------------------

uint16_t ShardReactor::now_delta() const {
    auto now = std::chrono::steady_clock::now().time_since_epoch();
    return static_cast<uint16_t>(
        std::chrono::duration_cast<std::chrono::seconds>(now).count() & 0xFFFF);
}

// ---------------------------------------------------------------------------
// run — main event loop
// ---------------------------------------------------------------------------

void ShardReactor::run() {
    running_.store(true, std::memory_order_release);

    while (running_.load(std::memory_order_relaxed)) {
        if (inflight_.empty()) {
            // State 2: idle — block until a request arrives.
            Request req;
            if (!inbox_.pop_blocking(req)) continue;
            if (req.type == OpType::Shutdown) break;
            dispatch(req);
#if defined(__linux__) && defined(HAVE_IO_URING)
            if (uring_ok_)
                io_uring_submit(&uring_);
#endif
        } else {
            // State 1: IO in-flight — tight poll loop.
            process_inbox(kMaxInboxBatch);
            reap_completions();
        }
        maybe_flush_memtable();
        maybe_run_eviction();
    }

    drain_remaining();
}

// ---------------------------------------------------------------------------
// process_inbox
// ---------------------------------------------------------------------------

void ShardReactor::process_inbox(int max_batch) {
    Request req;
    int count = 0;
    while (count < max_batch && inbox_.try_pop(req)) {
        if (req.type == OpType::Shutdown) {
            running_.store(false, std::memory_order_relaxed);
            return;
        }
        dispatch(req);
        ++count;
    }
    if (count > 0) {
#if defined(__linux__) && defined(HAVE_IO_URING)
        if (uring_ok_)
            io_uring_submit(&uring_);
#endif
    }
}

// ---------------------------------------------------------------------------
// complete_request — write result, then post; never use req.key/req.value after post
// ---------------------------------------------------------------------------

void ShardReactor::complete_request(Request& req, Result result) {
    *req.result = std::move(result);
    if (req.batch_remaining) {
        if (req.batch_remaining->fetch_sub(1, std::memory_order_acq_rel) == 1)
            sem_pool_->post(req.sem_slot);
    } else {
        sem_pool_->post(req.sem_slot);
    }
}

// ---------------------------------------------------------------------------
// dispatch
// ---------------------------------------------------------------------------

void ShardReactor::dispatch(Request& req) {
    switch (req.type) {
        case OpType::Get:      handle_get(req);    break;
        case OpType::Put:      handle_put(req);    break;
        case OpType::Delete:   handle_delete(req); break;
        case OpType::Shutdown: break;
    }
}

// ---------------------------------------------------------------------------
// handle_get
// ---------------------------------------------------------------------------

void ShardReactor::handle_get(Request& req) {
    LookupResult lr{};
    if (!index_.get(req.hash, now_delta(), lr)) {
        complete_request(req, {Status::NotFound, {}});
        return;
    }

    // Try memtable first.
    std::vector<uint8_t> buf(lr.length);
    if (memtables_.try_read_from_memory(buf.data(), lr.length,
                                         lr.mem_id, lr.offset)) {
        DecodedRecord dr = decode_record(buf.data());
        if (kRecordHeaderSize + dr.key_len + dr.val_len > lr.length ||
            dr.key_len != req.key.size() ||
            std::memcmp(dr.key, req.key.data(), dr.key_len) != 0) {
            complete_request(req, {Status::NotFound, {}});
            return;
        }
        complete_request(req, {Status::Ok,
            std::string(reinterpret_cast<const char*>(dr.val), dr.val_len)});
        return;
    }

    // Need disk read.
    int64_t file_off = memtables_.file_offset_for(lr.mem_id);
    if (file_off < 0) {
        complete_request(req, {Status::NotFound, {}});
        return;
    }

    uint64_t ring_offset = static_cast<uint64_t>(file_off) + lr.offset;
    size_t aligned_len = AlignedBuffer::align_up(lr.length);
    if (ring_offset + aligned_len > ring_.capacity()) {
        complete_request(req, {Status::Error, {}});
        return;
    }

#if defined(__linux__) && defined(HAVE_IO_URING)
    if (uring_ok_) {
        auto abuf = AlignedBuffer::allocate(aligned_len);
        if (!abuf.valid()) {
            complete_request(req, {Status::Error, {}});
            return;
        }

        uint64_t tag = next_tag_++;

        struct io_uring_sqe* sqe = io_uring_get_sqe(&uring_);
        if (!sqe) {
            io_uring_submit(&uring_);
            sqe = io_uring_get_sqe(&uring_);
            if (!sqe) {
                complete_request(req, {Status::Error, {}});
                return;
            }
        }

        io_uring_prep_read(sqe, ring_.read_fd(), abuf.data(),
                           static_cast<unsigned>(aligned_len),
                           ring_.base_offset() + ring_offset);
        io_uring_sqe_set_data64(sqe, tag);

        PendingOp pop{};
        pop.type      = PendingOpType::GetRead;
        pop.buf       = std::move(abuf);
        pop.mem_id    = lr.mem_id;
        pop.offset    = lr.offset;
        pop.length    = lr.length;

        inflight_.emplace(tag, std::move(pop));
        pending_requests_.emplace(tag, std::move(req));
        return;
    }
#endif

    // Fallback: synchronous pread.
    auto abuf = AlignedBuffer::allocate(aligned_len);
    if (!abuf.valid()) {
        complete_request(req, {Status::Error, {}});
        return;
    }

    ssize_t n = ring_.read(abuf.data(), aligned_len, ring_offset);
    if (n != static_cast<ssize_t>(aligned_len)) {
        complete_request(req, {Status::Error, {}});
        return;
    }

    DecodedRecord dr = decode_record(abuf.bytes());
    if (kRecordHeaderSize + dr.key_len + dr.val_len > lr.length ||
        dr.key_len != req.key.size() ||
        std::memcmp(dr.key, req.key.data(), dr.key_len) != 0) {
        complete_request(req, {Status::NotFound, {}});
        return;
    }
    complete_request(req, {Status::Ok,
        std::string(reinterpret_cast<const char*>(dr.val), dr.val_len)});
}

// ---------------------------------------------------------------------------
// handle_put
// ---------------------------------------------------------------------------

void ShardReactor::handle_put(Request& req) {
    size_t rec_len = record_size(req.key.size(), req.value.size());

    // If active memtable can't fit this record, flush first.
    if (memtables_.needs_flush(rec_len)) {
        if (!memtables_.is_flush_pending()) {
            maybe_flush_memtable();
        } else {
            // Previous flush still inflight — synchronous flush as fallback.
            // This shouldn't happen often with properly sized memtables.
            memtables_.flush_sync();
        }

        // If still can't fit after flush, try again with sync flush.
        if (memtables_.needs_flush(rec_len)) {
            memtables_.flush_sync();
        }
    }

    std::vector<uint8_t> rec(rec_len);
    encode_record(rec.data(),
                  req.key.data(), static_cast<uint32_t>(req.key.size()),
                  req.value.data(), static_cast<uint32_t>(req.value.size()));

    WriteResult wr = memtables_.put(rec.data(), rec_len);
    index_.put(req.hash, wr.mem_id, wr.offset, wr.length);
    delete_mgr_.maybe_evict();

    complete_request(req, {Status::Ok, {}});
}

// ---------------------------------------------------------------------------
// handle_delete
// ---------------------------------------------------------------------------

void ShardReactor::handle_delete(Request& req) {
    bool removed = index_.remove(req.hash);
    complete_request(req, {removed ? Status::Ok : Status::NotFound, {}});
}

// ---------------------------------------------------------------------------
// reap_completions
// ---------------------------------------------------------------------------

void ShardReactor::reap_completions() {
#if defined(__linux__) && defined(HAVE_IO_URING)
    if (!uring_ok_) return;

    struct io_uring_cqe* cqe = nullptr;
    unsigned head;
    unsigned count = 0;

    io_uring_for_each_cqe(&uring_, head, cqe) {
        uint64_t tag = io_uring_cqe_get_data64(cqe);
        auto it = inflight_.find(tag);
        if (it == inflight_.end()) {
            ++count;
            continue;
        }

        PendingOp& pop = it->second;

        switch (pop.type) {
        case PendingOpType::GetRead: {
            auto req_it = pending_requests_.find(tag);
            if (req_it == pending_requests_.end()) break;

            Request& req = req_it->second;
            // Use req.key only before complete_request; caller may wake after post.
            Result res;
            if (cqe->res < 0) {
                res = {Status::Error, {}};
            } else {
                DecodedRecord dr = decode_record(pop.buf.bytes());
                if (kRecordHeaderSize + dr.key_len + dr.val_len > pop.length ||
                    dr.key_len != req.key.size() ||
                    std::memcmp(dr.key, req.key.data(), dr.key_len) != 0) {
                    res = {Status::NotFound, {}};
                } else {
                    res = {Status::Ok,
                           std::string(reinterpret_cast<const char*>(dr.val),
                                       dr.val_len)};
                }
            }
            complete_request(req, std::move(res));
            pending_requests_.erase(req_it);
            break;
        }

        case PendingOpType::Flush: {
            if (cqe->res >= 0) {
                memtables_.complete_flush(pop.mem_id, pop.write_off);
            }
            break;
        }

        case PendingOpType::Trim:
            break;
        }

        inflight_.erase(it);
        ++count;
    }

    if (count > 0)
        io_uring_cq_advance(&uring_, count);
#endif
}

// ---------------------------------------------------------------------------
// maybe_flush_memtable
// ---------------------------------------------------------------------------

void ShardReactor::maybe_flush_memtable() {
    if (memtables_.is_flush_pending()) return;
    if (memtables_.active_mem_id() == 0 &&
        !memtables_.needs_flush(1)) return;

    // Only flush if the active memtable has data and is approaching full,
    // or if it was explicitly triggered by handle_put.
    auto& active_mt = memtables_;
    (void)active_mt;

    // The reactor calls this each iteration, but we only flush when needed.
    // needs_flush is checked by handle_put which calls us.
    // Here we just handle the async submission if a flush was set up.

#if defined(__linux__) && defined(HAVE_IO_URING)
    if (!uring_ok_) return;
    // Check if we need to submit a flush that was prepared.
    // Flush is driven by handle_put detecting needs_flush.
#endif
}

// ---------------------------------------------------------------------------
// submit_flush — called when handle_put detects memtable is full
// ---------------------------------------------------------------------------

// (integrated into handle_put flow via flush_sync or the memtable manager)

// ---------------------------------------------------------------------------
// maybe_run_eviction
// ---------------------------------------------------------------------------

void ShardReactor::maybe_run_eviction() {
    delete_mgr_.flush_discards();
}

// ---------------------------------------------------------------------------
// drain_remaining — shutdown path
// ---------------------------------------------------------------------------

void ShardReactor::drain_remaining() {
    // Process remaining inbox items.
    Request req;
    while (inbox_.try_pop(req)) {
        if (req.type == OpType::Shutdown) continue;
        dispatch(req);
    }

#if defined(__linux__) && defined(HAVE_IO_URING)
    if (uring_ok_) {
        io_uring_submit(&uring_);

        // Wait for all inflight ops to complete.
        while (!inflight_.empty()) {
            struct io_uring_cqe* cqe = nullptr;
            if (io_uring_wait_cqe(&uring_, &cqe) < 0)
                break;
            reap_completions();
        }
    }
#endif

    // Flush any remaining memtable data synchronously.
    memtables_.flush_sync();
}
