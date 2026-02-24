#pragma once

#include "flashringc/common.h"
#include "flashringc/delete_manager.h"
#include "flashringc/key_index.h"
#include "flashringc/memtable_manager.h"
#include "flashringc/mpsc_queue.h"
#include "flashringc/record.h"
#include "flashringc/request.h"
#include "flashringc/ring_device.h"
#include "flashringc/semaphore_pool.h"

#include <atomic>
#include <cstdint>
#include <vector>

#include "absl/container/flat_hash_map.h"

#if defined(__linux__) && defined(HAVE_IO_URING)
#include <liburing.h>
#endif

class ShardReactor {
public:
    ShardReactor(RingDevice ring, size_t mt_size, uint32_t index_cap,
                 SemaphorePool* sem_pool,
                 uint32_t queue_capacity = kDefaultQueueDepth,
                 uint32_t uring_depth = 256,
                 double eviction_threshold = 0.75,
                 double clear_threshold = 0.65);
    ~ShardReactor();

    ShardReactor(const ShardReactor&) = delete;
    ShardReactor& operator=(const ShardReactor&) = delete;

    // Push a request from any thread. Returns false if queue is full.
    bool submit(Request&& req);

    // Main event loop — runs on a dedicated thread, returns on stop().
    void run();

    // Signal the reactor to shut down.
    void stop();

    uint32_t key_count() const { return index_.size(); }
    uint64_t ring_usage() const { return ring_.write_offset(); }
    uint64_t ring_wrap_count() const { return ring_.wrap_count(); }

private:
    void process_inbox(int max_batch);
    void handle_get(Request& req);
    void handle_put(Request& req);
    void handle_delete(Request& req);

    void dispatch(Request& req);
    void complete_request(Request& req, Result result);
    void reap_completions();
    void maybe_flush_memtable();
    void maybe_run_eviction();
    void drain_remaining();

    uint16_t now_delta() const;

    RingDevice        ring_;
    MemtableManager   memtables_;
    KeyIndex          index_;
    DeleteManager     delete_mgr_;

    SemaphorePool*     sem_pool_;
    MPSCQueue<Request> inbox_;

    // In-flight io_uring operations keyed by tag.
    absl::flat_hash_map<uint64_t, PendingOp> inflight_;
    uint64_t next_tag_ = 1;

    // Requests whose disk IO is pending — stored alongside PendingOp.
    // We keep Request objects alive here until the CQE completes.
    absl::flat_hash_map<uint64_t, Request> pending_requests_;

    std::atomic<bool> running_{false};

#if defined(__linux__) && defined(HAVE_IO_URING)
    struct io_uring uring_;
    bool   uring_ok_ = false;
#endif
    uint32_t uring_depth_;
};
