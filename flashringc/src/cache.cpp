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

    uint64_t per_shard_capacity =
        AlignedBuffer::align_up(cfg.ring_capacity / cfg.num_shards);
    size_t per_shard_mt =
        AlignedBuffer::align_up(cfg.memtable_size / cfg.num_shards);
    uint32_t per_shard_keys = cfg.index_capacity / cfg.num_shards;
    if (per_shard_keys == 0) per_shard_keys = 1;

    for (uint32_t i = 0; i < cfg.num_shards; ++i) {
        std::string path = cfg.device_path + "." + std::to_string(i);
        auto ring = RingDevice::open(path, per_shard_capacity);
        c.shards_.push_back(
            std::make_unique<Shard>(std::move(ring), per_shard_mt, per_shard_keys));
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
