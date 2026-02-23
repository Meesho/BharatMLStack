#include "cache.h"
#include "record.h"

#include <cstring>
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

    // Hash all keys and group by shard (caller thread only)
    std::vector<Hash128> hashes(count);
    std::vector<std::vector<size_t>> shard_groups(num_shards_);

    for (size_t i = 0; i < count; ++i) {
        hashes[i] = hash_key(reqs[i].key, reqs[i].key_len);
        uint32_t shard_id = hashes[i].lo % num_shards_;
        shard_groups[shard_id].push_back(i);
    }

    uint32_t num_shards_with_keys = 0;
    for (uint32_t s = 0; s < num_shards_; ++s)
        if (!shard_groups[s].empty()) ++num_shards_with_keys;

    BatchCompletion completion(num_shards_with_keys);

    // Keep sub_reqs/sub_hashes alive until reactors finish
    std::vector<std::vector<BatchGetRequest>> sub_reqs_storage;
    std::vector<std::vector<Hash128>> sub_hashes_storage;
    sub_reqs_storage.reserve(num_shards_with_keys);
    sub_hashes_storage.reserve(num_shards_with_keys);

    for (uint32_t s = 0; s < num_shards_; ++s) {
        const std::vector<size_t>& group = shard_groups[s];
        if (group.empty()) continue;

        const size_t n = group.size();
        sub_reqs_storage.emplace_back(n);
        sub_hashes_storage.emplace_back(n);
        auto& sub_reqs = sub_reqs_storage.back();
        auto& sub_hashes = sub_hashes_storage.back();
        for (size_t j = 0; j < n; ++j) {
            sub_reqs[j] = reqs[group[j]];
            sub_hashes[j] = hashes[group[j]];
        }

        ShardBatchRequest shard_req;
        shard_req.completion = &completion;
        shard_req.count = n;
        shard_req.reqs = sub_reqs.data();
        shard_req.hashes = sub_hashes.data();
        shard_req.results = results;
        shard_req.result_indices = group.data();
        shards_[s]->submit_batch(std::move(shard_req));
    }

    completion.wait();
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
