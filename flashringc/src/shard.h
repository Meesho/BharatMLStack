#pragma once

#include "delete_manager.h"
#include "io_engine.h"
#include "key_index.h"
#include "memtable.h"
#include "record.h"
#include "ring_device.h"

#include <cstdint>
#include <shared_mutex>
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

// One independent partition of the cache. Owns its own ring device, memtable
// manager, key index, delete manager, io engine, and lock.
class Shard {
public:
    Shard(RingDevice ring, size_t mt_size, uint32_t index_cap,
          bool use_io_uring = false);

    Shard(const Shard&) = delete;
    Shard& operator=(const Shard&) = delete;

    bool put(Hash128 h, const void* rec, size_t rec_len);

    bool get(Hash128 h, const void* key, size_t key_len,
             void* val_buf, size_t val_buf_len, size_t* actual_len);

    void get_batch(const BatchGetRequest* reqs, BatchGetResult* results,
                   size_t count, const Hash128* hashes);

    bool remove(Hash128 h);

    void flush();

    uint32_t key_count() const;
    uint64_t ring_usage() const;

private:
    bool read_record(const LookupResult& lr, std::vector<uint8_t>& buf);

    RingDevice                ring_;
    MemtableManager           memtables_;
    KeyIndex                  index_;
    DeleteManager             delete_mgr_;
    IoEngine                  io_engine_;
    mutable std::shared_mutex mu_;
};
