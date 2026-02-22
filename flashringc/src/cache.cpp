#include "cache.h"
#include "record.h"

#include <algorithm>
#include <chrono>
#include <cstring>
#include <stdexcept>

// ---------------------------------------------------------------------------
// open / move / destructor
// ---------------------------------------------------------------------------

Cache Cache::open(const CacheConfig& cfg) {
    Cache c;
    c.ring_      = std::make_unique<RingDevice>(
                       RingDevice::open(cfg.device_path, cfg.ring_capacity));
    c.memtables_ = std::make_unique<MemtableManager>(*c.ring_, cfg.memtable_size);
    c.index_     = std::make_unique<KeyIndex>(cfg.index_capacity);
    return c;
}

Cache::~Cache() = default;

Cache::Cache(Cache&& o) noexcept
    : ring_(std::move(o.ring_)),
      memtables_(std::move(o.memtables_)),
      index_(std::move(o.index_)) {}

Cache& Cache::operator=(Cache&& o) noexcept {
    if (this != &o) {
        ring_      = std::move(o.ring_);
        memtables_ = std::move(o.memtables_);
        index_     = std::move(o.index_);
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

    WriteResult wr = memtables_->put(rec.data(), rec_len);

    {
        std::unique_lock lk(mu_);
        index_->put(h, wr.mem_id, wr.offset, wr.length);
    }
    return true;
}

// ---------------------------------------------------------------------------
// get
// ---------------------------------------------------------------------------

bool Cache::get(const void* key, size_t key_len,
                void* val_buf, size_t val_buf_len, size_t* actual_len) {
    Hash128 h = hash_key(key, key_len);

    LookupResult lr{};
    {
        auto now = std::chrono::steady_clock::now().time_since_epoch();
        auto delta = static_cast<uint16_t>(
            std::chrono::duration_cast<std::chrono::seconds>(now).count() & 0xFFFF);
        std::shared_lock lk(mu_);
        if (!index_->get(h, delta, lr))
            return false;
    }

    std::vector<uint8_t> buf(lr.length);
    if (!read_record(lr, buf))
        return false;

    DecodedRecord dr = decode_record(buf.data());

    // Collision guard: verify the key matches.
    if (dr.key_len != key_len ||
        std::memcmp(dr.key, key, key_len) != 0)
        return false;

    if (actual_len)
        *actual_len = dr.val_len;

    size_t copy_len = std::min(static_cast<size_t>(dr.val_len), val_buf_len);
    if (copy_len > 0)
        std::memcpy(val_buf, dr.val, copy_len);

    return true;
}

// ---------------------------------------------------------------------------
// remove
// ---------------------------------------------------------------------------

bool Cache::remove(const void* key, size_t key_len) {
    Hash128 h = hash_key(key, key_len);
    std::unique_lock lk(mu_);
    return index_->remove(h);
}

// ---------------------------------------------------------------------------
// flush
// ---------------------------------------------------------------------------

void Cache::flush() {
    memtables_->flush();
}

// ---------------------------------------------------------------------------
// stats
// ---------------------------------------------------------------------------

uint32_t Cache::key_count() const {
    std::shared_lock lk(mu_);
    return index_->size();
}

uint64_t Cache::ring_usage() const {
    return ring_->write_offset();
}

// ---------------------------------------------------------------------------
// read_record  (internal helper)
// ---------------------------------------------------------------------------

bool Cache::read_record(const LookupResult& lr, std::vector<uint8_t>& buf) {
    buf.resize(lr.length);

    // Fast path: data may still be in a memtable.
    if (memtables_->try_read_from_memory(buf.data(), lr.length,
                                          lr.mem_id, lr.offset))
        return true;

    // Slow path: read from ring device.
    int64_t file_off = memtables_->file_offset_for(lr.mem_id);
    if (file_off < 0)
        return false;

    uint64_t abs_offset = static_cast<uint64_t>(file_off) + lr.offset;
    ssize_t n = ring_->read_unaligned(buf.data(), lr.length, abs_offset);
    return n == static_cast<ssize_t>(lr.length);
}
