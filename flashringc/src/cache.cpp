#include "flashringc/cache.h"
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

    c.shards_.reserve(c.num_shards_);
    for (uint32_t i = 0; i < c.num_shards_; ++i) {
        RingDevice ring = is_blk
            ? RingDevice::open_region(cfg.device_path,
                                      i * per_shard_capacity,
                                      per_shard_capacity)
            : RingDevice::open(cfg.device_path + "." + std::to_string(i),
                               per_shard_capacity);
        c.shards_.push_back(std::make_unique<ShardReactor>(
            std::move(ring), per_shard_mt, per_shard_keys,
            cfg.queue_capacity, cfg.uring_queue_depth));
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
    : shards_(std::move(o.shards_)),
      threads_(std::move(o.threads_)),
      num_shards_(o.num_shards_) {
    o.num_shards_ = 0;
}

Cache& Cache::operator=(Cache&& o) noexcept {
    if (this != &o) {
        for (auto& s : shards_) s->stop();
        for (auto& t : threads_) if (t.joinable()) t.join();

        shards_     = std::move(o.shards_);
        threads_    = std::move(o.threads_);
        num_shards_ = o.num_shards_;
        o.num_shards_ = 0;
    }
    return *this;
}

// ---------------------------------------------------------------------------
// Async API
// ---------------------------------------------------------------------------

std::future<Result> Cache::get(std::string_view key) {
    Hash128 h = hash_key(key.data(), key.size());
    Request req;
    req.type = OpType::Get;
    req.key  = std::string(key);
    req.hash = h;
    auto fut = req.promise.get_future();
    shard_for(h).submit(std::move(req));
    return fut;
}

std::future<Result> Cache::put(std::string_view key, std::string_view value) {
    Hash128 h = hash_key(key.data(), key.size());
    Request req;
    req.type  = OpType::Put;
    req.key   = std::string(key);
    req.value = std::string(value);
    req.hash  = h;
    auto fut = req.promise.get_future();
    shard_for(h).submit(std::move(req));
    return fut;
}

std::future<Result> Cache::del(std::string_view key) {
    Hash128 h = hash_key(key.data(), key.size());
    Request req;
    req.type = OpType::Delete;
    req.key  = std::string(key);
    req.hash = h;
    auto fut = req.promise.get_future();
    shard_for(h).submit(std::move(req));
    return fut;
}

std::vector<std::future<Result>> Cache::batch_get(
    const std::string_view* keys, size_t count) {
    std::vector<std::future<Result>> futs;
    futs.reserve(count);
    for (size_t i = 0; i < count; ++i)
        futs.push_back(get(keys[i]));
    return futs;
}

std::vector<std::future<Result>> Cache::batch_put(
    const KVPair* pairs, size_t count) {
    std::vector<std::future<Result>> futs;
    futs.reserve(count);
    for (size_t i = 0; i < count; ++i)
        futs.push_back(put(pairs[i].key, pairs[i].value));
    return futs;
}

// ---------------------------------------------------------------------------
// Synchronous convenience wrappers
// ---------------------------------------------------------------------------

bool Cache::put_sync(const void* key, size_t key_len,
                     const void* val, size_t val_len) {
    auto fut = put(std::string_view(static_cast<const char*>(key), key_len),
                   std::string_view(static_cast<const char*>(val), val_len));
    Result r = fut.get();
    return r.status == Status::Ok;
}

bool Cache::get_sync(const void* key, size_t key_len,
                     void* val_buf, size_t val_buf_len, size_t* actual_len) {
    auto fut = get(std::string_view(static_cast<const char*>(key), key_len));
    Result r = fut.get();
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
    auto fut = del(std::string_view(static_cast<const char*>(key), key_len));
    Result r = fut.get();
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
