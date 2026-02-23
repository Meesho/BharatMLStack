#pragma once

#include "flashringc/common.h"
#include "flashringc/request.h"
#include "flashringc/shard_reactor.h"

#include <cstdint>
#include <future>
#include <memory>
#include <string>
#include <string_view>
#include <thread>
#include <vector>

struct CacheConfig {
    std::string device_path;
    uint64_t    ring_capacity    = 0;          // 0 = auto-detect (block devices)
    size_t      memtable_size    = 64 << 20;   // 64 MB total
    uint32_t    index_capacity   = 1'000'000;
    uint32_t    num_shards       = 0;          // 0 = hardware_concurrency()
    uint32_t    queue_capacity   = kDefaultQueueDepth;
    uint32_t    uring_queue_depth = 256;
};

class Cache {
public:
    static Cache open(const CacheConfig& cfg);
    ~Cache();

    Cache(Cache&&) noexcept;
    Cache& operator=(Cache&&) noexcept;
    Cache(const Cache&) = delete;
    Cache& operator=(const Cache&) = delete;

    std::future<Result> get(std::string_view key);
    std::future<Result> put(std::string_view key, std::string_view value);
    std::future<Result> del(std::string_view key);

    std::vector<std::future<Result>> batch_get(
        const std::string_view* keys, size_t count);
    std::vector<std::future<Result>> batch_put(
        const KVPair* pairs, size_t count);

    // Synchronous convenience wrappers (block on future).
    bool put_sync(const void* key, size_t key_len,
                  const void* val, size_t val_len);
    bool get_sync(const void* key, size_t key_len,
                  void* val_buf, size_t val_buf_len, size_t* actual_len);
    bool remove_sync(const void* key, size_t key_len);

    // Flush all shards synchronously.
    void flush();

    uint32_t key_count() const;
    uint64_t ring_usage() const;
    uint32_t num_shards() const { return num_shards_; }

private:
    Cache() = default;

    ShardReactor& shard_for(Hash128 h) {
        return *shards_[h.lo % num_shards_];
    }

    std::vector<std::unique_ptr<ShardReactor>> shards_;
    std::vector<std::thread>                   threads_;
    uint32_t num_shards_ = 0;
};
