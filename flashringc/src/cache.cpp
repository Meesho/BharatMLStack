#include "cache.h"
#include "record.h"

#include <cstring>
#include <future>
#include <stdexcept>
#include <vector>

// ---------------------------------------------------------------------------
// open / move / destructor
// ---------------------------------------------------------------------------

Cache Cache::open(const CacheConfig& cfg) {
    if (cfg.num_shards == 0)
        throw std::invalid_argument("num_shards must be > 0");

    Cache c;
    c.num_shards_ = cfg.num_shards;
    c.shards_.reserve(cfg.num_shards);

    bool is_blk = RingDevice::is_block_device_path(cfg.device_path);

    uint64_t total_capacity = cfg.ring_capacity;
    if (is_blk && total_capacity == 0) {
        auto probe = RingDevice::open(cfg.device_path, 0);
        total_capacity = probe.capacity();
    }

    uint64_t per_shard_capacity =
        AlignedBuffer::align_up(total_capacity / cfg.num_shards);
    size_t per_shard_mt =
        AlignedBuffer::align_up(cfg.memtable_size / cfg.num_shards);
    uint32_t per_shard_keys = cfg.index_capacity / cfg.num_shards;
    if (per_shard_keys == 0) per_shard_keys = 1;

    for (uint32_t i = 0; i < cfg.num_shards; ++i) {
        RingDevice ring = is_blk
            ? RingDevice::open_region(cfg.device_path,
                                      i * per_shard_capacity,
                                      per_shard_capacity)
            : RingDevice::open(cfg.device_path + "." + std::to_string(i),
                               per_shard_capacity);
        c.shards_.push_back(
            std::make_unique<Shard>(std::move(ring), per_shard_mt,
                                   per_shard_keys, cfg.use_io_uring));
    }

    return c;
}

Cache::~Cache() = default;

Cache::Cache(Cache&& o) noexcept
    : shards_(std::move(o.shards_)),
      num_shards_(o.num_shards_) {
    o.num_shards_ = 0;
}

Cache& Cache::operator=(Cache&& o) noexcept {
    if (this != &o) {
        shards_     = std::move(o.shards_);
        num_shards_ = o.num_shards_;
        o.num_shards_ = 0;
    }
    return *this;
}

// ---------------------------------------------------------------------------
// put
// ---------------------------------------------------------------------------

bool Cache::put(const void* key, size_t key_len,
                const void* val, size_t val_len) {
    size_t rec_len = record_size(key_len, val_len);
    std::vector<uint8_t> rec(rec_len);
    encode_record(rec.data(), key, static_cast<uint32_t>(key_len),
                  val, static_cast<uint32_t>(val_len));

    Hash128 h = hash_key(key, key_len);
    return shard_for(h).put(h, rec.data(), rec_len);
}

// ---------------------------------------------------------------------------
// get
// ---------------------------------------------------------------------------

bool Cache::get(const void* key, size_t key_len,
                void* val_buf, size_t val_buf_len, size_t* actual_len) {
    Hash128 h = hash_key(key, key_len);
    return shard_for(h).get(h, key, key_len, val_buf, val_buf_len, actual_len);
}

// ---------------------------------------------------------------------------
// get_batch
// ---------------------------------------------------------------------------

void Cache::get_batch(const BatchGetRequest* reqs, BatchGetResult* results,
                      size_t count) {
    if (count == 0) return;

    // Hash all keys and group by shard
    std::vector<Hash128> hashes(count);
    std::vector<std::vector<size_t>> shard_groups(num_shards_);

    for (size_t i = 0; i < count; ++i) {
        hashes[i] = hash_key(reqs[i].key, reqs[i].key_len);
        uint32_t shard_id = hashes[i].lo % num_shards_;
        shard_groups[shard_id].push_back(i);
    }

    // Dispatch each shard's batch in parallel (batch latency = max over shards, not sum)
    std::vector<std::future<void>> futures;
    futures.reserve(num_shards_);
    for (uint32_t s = 0; s < num_shards_; ++s) {
        const std::vector<size_t>& group = shard_groups[s];
        if (group.empty()) continue;

        futures.push_back(std::async(std::launch::async, [this, s, &reqs, &results, &hashes, &shard_groups]() {
            const std::vector<size_t>& g = shard_groups[s];
            const size_t n = g.size();
            std::vector<BatchGetRequest> sub_reqs(n);
            std::vector<BatchGetResult>  sub_results(n);
            std::vector<Hash128>         sub_hashes(n);
            for (size_t j = 0; j < n; ++j) {
                sub_reqs[j]   = reqs[g[j]];
                sub_hashes[j] = hashes[g[j]];
            }
            shards_[s]->get_batch(sub_reqs.data(), sub_results.data(), n, sub_hashes.data());
            for (size_t j = 0; j < n; ++j)
                results[g[j]] = sub_results[j];
        }));
    }
    for (auto& f : futures)
        f.get();
}

// ---------------------------------------------------------------------------
// remove
// ---------------------------------------------------------------------------

bool Cache::remove(const void* key, size_t key_len) {
    Hash128 h = hash_key(key, key_len);
    return shard_for(h).remove(h);
}

// ---------------------------------------------------------------------------
// flush
// ---------------------------------------------------------------------------

void Cache::flush() {
    for (auto& s : shards_)
        s->flush();
}

// ---------------------------------------------------------------------------
// stats
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
