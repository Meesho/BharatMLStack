#pragma once

#include "key_index.h"
#include "shard.h"

#include <cstdint>
#include <memory>
#include <string>
#include <vector>

struct CacheConfig {
    std::string device_path;
    uint64_t    ring_capacity  = 0;          // 0 = auto-detect (block devices)
    size_t      memtable_size  = 64 << 20;   // 64 MB
    uint32_t    index_capacity = 1'000'000;  // max keys
    uint32_t    num_shards     = 64;
    bool        use_io_uring   = true;      // enable io_uring for reads (Linux only)
};

class Cache {
public:
    static Cache open(const CacheConfig& cfg);
    ~Cache();

    Cache(Cache&&) noexcept;
    Cache& operator=(Cache&&) noexcept;
    Cache(const Cache&) = delete;
    Cache& operator=(const Cache&) = delete;

    bool put(const void* key, size_t key_len,
             const void* val, size_t val_len);

    bool get(const void* key, size_t key_len,
             void* val_buf, size_t val_buf_len, size_t* actual_len);

    void get_batch(const BatchGetRequest* reqs, BatchGetResult* results,
                   size_t count);

    bool remove(const void* key, size_t key_len);

    void flush();

    uint32_t key_count()  const;
    uint64_t ring_usage() const;

private:
    Cache() = default;

    Shard& shard_for(Hash128 h) { return *shards_[h.lo % num_shards_]; }

    std::vector<std::unique_ptr<Shard>> shards_;
    uint32_t num_shards_ = 0;
};
