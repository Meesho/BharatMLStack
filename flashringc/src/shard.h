#pragma once

#include "delete_manager.h"
#include "io_engine.h"
#include "key_index.h"
#include "memtable.h"
#include "record.h"
#include "ring_device.h"

#include <condition_variable>
#include <cstdint>
#include <memory>
#include <mutex>
#include <queue>
#include <shared_mutex>
#include <thread>
#include <vector>

struct BatchGetRequest {
    const void* key;
    size_t      key_len;
    void*       val_buf;
    size_t      val_buf_len;
};

struct BatchGetResult {
    bool   found      = false;
    size_t actual_len  = 0;
};

// One shard slice of a get_batch. Caller enqueues to shard's reactor; reactor
// processes and calls completion->mark_one_done() when this slice is finished.
// If result_indices is non-null, reactor writes to results[result_indices[j]];
// otherwise writes to results[j].
struct ShardBatchRequest {
    uint64_t                    batch_id = 0;
    class BatchCompletion*      completion = nullptr;
    size_t                      count = 0;
    const BatchGetRequest*      reqs = nullptr;
    const Hash128*              hashes = nullptr;
    BatchGetResult*             results = nullptr;       // base array
    const size_t*               result_indices = nullptr;  // indices into results, or null for 0..count-1
};

// Per-batch completion: caller waits until all shard slices have called mark_one_done().
class BatchCompletion {
public:
    explicit BatchCompletion(uint32_t num_shards_to_wait)
        : remaining_(num_shards_to_wait) {}

    void mark_one_done();
    void wait();

private:
    std::mutex              mu_;
    std::condition_variable cv_;
    uint32_t                remaining_;
};

// One independent partition of the cache. Owns its own ring device, memtable
// manager, key index, delete manager, io engine, and lock.
class Shard {
public:
    Shard(RingDevice ring, size_t mt_size, uint32_t index_cap,
          bool use_io_uring = false);
    ~Shard();

    Shard(const Shard&) = delete;
    Shard& operator=(const Shard&) = delete;

    bool put(Hash128 h, const void* rec, size_t rec_len);

    bool get(Hash128 h, const void* key, size_t key_len,
             void* val_buf, size_t val_buf_len, size_t* actual_len);

    void get_batch(const BatchGetRequest* reqs, BatchGetResult* results,
                   size_t count, const Hash128* hashes);

    // Reactor path: enqueue a slice for the reactor thread to process.
    void submit_batch(ShardBatchRequest req);

    bool remove(Hash128 h);

    void flush();

    uint32_t key_count() const;
    uint64_t ring_usage() const;

private:
    bool read_record(const LookupResult& lr, std::vector<uint8_t>& buf);
    void reactor_loop();

    RingDevice                ring_;
    MemtableManager           memtables_;
    KeyIndex                  index_;
    DeleteManager             delete_mgr_;
    IoEngine                  io_engine_;
    mutable std::shared_mutex mu_;

    std::thread               reactor_thread_;
    std::mutex                queue_mu_;
    std::queue<ShardBatchRequest> request_queue_;
    std::condition_variable   queue_cv_;
    bool                      shutdown_ = false;
};
