#pragma once

#include "flashringc/common.h"
#include "flashringc/request.h"
#include "flashringc/semaphore_pool.h"
#include "flashringc/shard_reactor.h"

#include <cstdint>
#include <memory>
#include <string>
#include <string_view>
#include <thread>
#include <vector>

struct CacheConfig {
    std::string device_path;
    uint64_t    ring_capacity     = 0;          // 0 = auto-detect (block devices)
    size_t      memtable_size     = 64 << 20;   // 64 MB total
    uint32_t    index_capacity    = 1'000'000;
    uint32_t    num_shards        = 0;          // 0 = hardware_concurrency()
    uint32_t    queue_capacity    = kDefaultQueueDepth;
    uint32_t    uring_queue_depth = 256;
    size_t      sem_pool_capacity = 1024;
    double      eviction_threshold = 0.75;     // start evicting when usage >= this
    double      clear_threshold    = 0.65;     // stop evicting when usage <= this
};

class Cache {
public:
    static Cache open(const CacheConfig& cfg);
    ~Cache();

    Cache(Cache&&) noexcept;
    Cache& operator=(Cache&&) noexcept;
    Cache(const Cache&) = delete;
    Cache& operator=(const Cache&) = delete;

    Result get(std::string_view key);
    Result put(std::string_view key, std::string_view value);
    Result del(std::string_view key);

    std::vector<Result> batch_get(const std::vector<std::string_view>& keys);
    std::vector<Result> batch_put(const std::vector<KVPair>& pairs);

    bool put_sync(const void* key, size_t key_len,
                  const void* val, size_t val_len);
    bool get_sync(const void* key, size_t key_len,
                  void* val_buf, size_t val_buf_len, size_t* actual_len);
    bool remove_sync(const void* key, size_t key_len);

    // Flush all shards synchronously.
    void flush();

    uint32_t key_count() const;
    uint64_t ring_usage() const;
    uint64_t ring_wrap_count() const;
    uint32_t num_shards() const { return num_shards_; }

private:
    Cache() = default;

    ShardReactor& shard_for(Hash128 h) {
        return *shards_[h.lo % num_shards_];
    }

    std::unique_ptr<SemaphorePool>             sem_pool_;
    std::vector<std::unique_ptr<ShardReactor>> shards_;
    std::vector<std::thread>                   threads_;
    uint32_t num_shards_ = 0;
};
