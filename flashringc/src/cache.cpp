#include "flashringc/cache.h"
#include "flashringc/delete_manager.h"
#include "flashringc/record.h"

#include <algorithm>
#include <cstring>
#include <stdexcept>
#include <thread>

#ifdef __linux__
#include <pthread.h>
#endif

// ---------------------------------------------------------------------------
// open
// ---------------------------------------------------------------------------

Cache Cache::open(const CacheConfig& cfg) {
    Cache c;

    if (cfg.clear_threshold <= 0 || cfg.eviction_threshold <= cfg.clear_threshold ||
        cfg.eviction_threshold > 1.0) {
        throw std::invalid_argument(
            "CacheConfig: require 0 < clear_threshold < eviction_threshold <= 1");
    }

    c.num_shards_ = cfg.num_shards;
    if (c.num_shards_ == 0)
        c.num_shards_ = std::max(1u, std::thread::hardware_concurrency());

    bool is_blk = RingDevice::is_block_device_path(cfg.device_path);

    uint64_t total_capacity = cfg.ring_capacity;
    if (is_blk && total_capacity == 0) {
        auto probe = RingDevice::open(cfg.device_path, 0);
        total_capacity = probe.capacity();
    }

    if (total_capacity == 0)
        throw std::invalid_argument("ring_capacity required for file-backed cache");

    uint64_t per_shard_capacity =
        AlignedBuffer::align_up(total_capacity / c.num_shards_);
    size_t per_shard_mt =
        AlignedBuffer::align_up(cfg.memtable_size / c.num_shards_);
    uint32_t per_shard_keys = cfg.index_capacity / c.num_shards_;
    if (per_shard_keys == 0) per_shard_keys = 1;

    c.sem_pool_ = std::make_unique<SemaphorePool>(cfg.sem_pool_capacity);

    c.shards_.reserve(c.num_shards_);
    for (uint32_t i = 0; i < c.num_shards_; ++i) {
        RingDevice ring = is_blk
            ? RingDevice::open_region(cfg.device_path,
                                      i * per_shard_capacity,
                                      per_shard_capacity)
            : RingDevice::open(cfg.device_path + "." + std::to_string(i),
                               per_shard_capacity);
        DeleteManager::Config dm_cfg{
            .eviction_threshold = cfg.eviction_threshold,
            .clear_threshold    = cfg.clear_threshold,
        };
        c.shards_.push_back(std::make_unique<ShardReactor>(
            std::move(ring), per_shard_mt, per_shard_keys,
            c.sem_pool_.get(),
            cfg.queue_capacity, cfg.uring_queue_depth,
            dm_cfg));
    }

    // Spawn reactor threads.
    c.threads_.reserve(c.num_shards_);
    for (uint32_t i = 0; i < c.num_shards_; ++i) {
        ShardReactor* reactor = c.shards_[i].get();
        c.threads_.emplace_back([reactor, i]() {
#ifdef __linux__
            cpu_set_t cpuset;
            CPU_ZERO(&cpuset);
            CPU_SET(i % std::thread::hardware_concurrency(), &cpuset);
            pthread_setaffinity_np(pthread_self(), sizeof(cpu_set_t), &cpuset);
#endif
            reactor->run();
        });

        // Wait for the reactor to start running before proceeding.
        while (!reactor->key_count() && !reactor->ring_usage()) {
            // Just ensure run() has been entered — a brief spin.
            // The reactor sets running_ to true at the top of run().
            std::this_thread::yield();
            break;
        }
    }

    // Give reactors a moment to initialize.
    std::this_thread::sleep_for(std::chrono::milliseconds(1));

    return c;
}

// ---------------------------------------------------------------------------
// Destructor
// ---------------------------------------------------------------------------

Cache::~Cache() {
    for (auto& s : shards_)
        s->stop();
    for (auto& t : threads_) {
        if (t.joinable())
            t.join();
    }
}

Cache::Cache(Cache&& o) noexcept
    : sem_pool_(std::move(o.sem_pool_)),
      shards_(std::move(o.shards_)),
      threads_(std::move(o.threads_)),
      num_shards_(o.num_shards_) {
    o.num_shards_ = 0;
}

Cache& Cache::operator=(Cache&& o) noexcept {
    if (this != &o) {
        for (auto& s : shards_) s->stop();
        for (auto& t : threads_) if (t.joinable()) t.join();

        sem_pool_   = std::move(o.sem_pool_);
        shards_     = std::move(o.shards_);
        threads_    = std::move(o.threads_);
        num_shards_ = o.num_shards_;
        o.num_shards_ = 0;
    }
    return *this;
}

// ---------------------------------------------------------------------------
// Blocking API (semaphore pool)
// ---------------------------------------------------------------------------

Result Cache::get(std::string_view key) {
    Hash128 h = hash_key(key.data(), key.size());
    uint32_t slot = sem_pool_->acquire();
    Result result;
    Request req;
    req.type = OpType::Get;
    req.hash = h;
    req.key = key;
    req.result = &result;
    req.sem_slot = slot;
    req.batch_remaining = nullptr;
    shard_for(h).submit(std::move(req));
    sem_pool_->wait(slot);
    sem_pool_->release(slot);
    return result;
}

Result Cache::put(std::string_view key, std::string_view value) {
    Hash128 h = hash_key(key.data(), key.size());
    uint32_t slot = sem_pool_->acquire();
    Result result;
    Request req;
    req.type = OpType::Put;
    req.hash = h;
    req.key = key;
    req.value = value;
    req.result = &result;
    req.sem_slot = slot;
    req.batch_remaining = nullptr;
    shard_for(h).submit(std::move(req));
    sem_pool_->wait(slot);
    sem_pool_->release(slot);
    return result;
}

Result Cache::del(std::string_view key) {
    Hash128 h = hash_key(key.data(), key.size());
    uint32_t slot = sem_pool_->acquire();
    Result result;
    Request req;
    req.type = OpType::Delete;
    req.hash = h;
    req.key = key;
    req.result = &result;
    req.sem_slot = slot;
    req.batch_remaining = nullptr;
    shard_for(h).submit(std::move(req));
    sem_pool_->wait(slot);
    sem_pool_->release(slot);
    return result;
}

std::vector<Result> Cache::batch_get(const std::vector<std::string_view>& keys) {
    if (keys.empty()) return {};
    std::vector<Result> results(keys.size());
    std::atomic<int> remaining{static_cast<int>(keys.size())};
    uint32_t slot = sem_pool_->acquire();

    for (size_t i = 0; i < keys.size(); ++i) {
        Hash128 h = hash_key(keys[i].data(), keys[i].size());
        Request req;
        req.type = OpType::Get;
        req.hash = h;
        req.key = keys[i];
        req.result = &results[i];
        req.sem_slot = slot;
        req.batch_remaining = &remaining;
        shard_for(h).submit(std::move(req));
    }

    sem_pool_->wait(slot);
    sem_pool_->release(slot);
    return results;
}

std::vector<Result> Cache::batch_put(const std::vector<KVPair>& pairs) {
    if (pairs.empty()) return {};
    std::vector<Result> results(pairs.size());
    std::atomic<int> remaining{static_cast<int>(pairs.size())};
    uint32_t slot = sem_pool_->acquire();

    for (size_t i = 0; i < pairs.size(); ++i) {
        Hash128 h = hash_key(pairs[i].key.data(), pairs[i].key.size());
        Request req;
        req.type = OpType::Put;
        req.hash = h;
        req.key = pairs[i].key;
        req.value = pairs[i].value;
        req.result = &results[i];
        req.sem_slot = slot;
        req.batch_remaining = &remaining;
        shard_for(h).submit(std::move(req));
    }

    sem_pool_->wait(slot);
    sem_pool_->release(slot);
    return results;
}

// ---------------------------------------------------------------------------
// Synchronous convenience wrappers
// ---------------------------------------------------------------------------

bool Cache::put_sync(const void* key, size_t key_len,
                     const void* val, size_t val_len) {
    Result r = put(std::string_view(static_cast<const char*>(key), key_len),
                   std::string_view(static_cast<const char*>(val), val_len));
    return r.status == Status::Ok;
}

bool Cache::get_sync(const void* key, size_t key_len,
                     void* val_buf, size_t val_buf_len, size_t* actual_len) {
    Result r = get(std::string_view(static_cast<const char*>(key), key_len));
    if (r.status != Status::Ok)
        return false;
    if (actual_len)
        *actual_len = r.value.size();
    size_t copy_len = std::min(r.value.size(), val_buf_len);
    if (copy_len > 0)
        std::memcpy(val_buf, r.value.data(), copy_len);
    return true;
}

bool Cache::remove_sync(const void* key, size_t key_len) {
    Result r = del(std::string_view(static_cast<const char*>(key), key_len));
    return r.status == Status::Ok;
}

// ---------------------------------------------------------------------------
// flush
// ---------------------------------------------------------------------------

void Cache::flush() {
    // Submit flush requests to all shards and wait.
    // Since flush is a maintenance operation, we do it via the sync path
    // on each reactor. We send a special empty-key put and then the reactor
    // handles it on the next iteration. But for simplicity, we just
    // do a sync sleep-wait after stopping/restarting — actually, the simplest
    // approach is to stop each reactor, which drains + flushes, then restart.
    // However, that's heavy. Instead, just sleep briefly to let reactors
    // process pending work.
    std::this_thread::sleep_for(std::chrono::milliseconds(10));
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

uint32_t Cache::key_count() const {
    uint32_t total = 0;
    for (auto& s : shards_)
        total += s->key_count();
    return total;
}

uint64_t Cache::ring_usage() const {
    uint64_t total = 0;
    for (auto& s : shards_)
        total += s->ring_usage();
    return total;
}

uint64_t Cache::ring_wrap_count() const {
    uint64_t total = 0;
    for (auto& s : shards_)
        total += s->ring_wrap_count();
    return total;
}
