#include "flashringc/key_index.h"
#include "flashringc/common.h"

#include <cstdlib>
#include <stdexcept>

#define XXH_INLINE_ALL
#include <xxhash.h>

// ---------------------------------------------------------------------------
// hash_key  (xxHash 128-bit, split into lo/hi)
// ---------------------------------------------------------------------------

Hash128 hash_key(const void* key, size_t len) {
    XXH128_hash_t h = XXH3_128bits(key, len);
    return {h.low64, h.high64};
}

// ---------------------------------------------------------------------------
// KeyIndex
// ---------------------------------------------------------------------------

KeyIndex::KeyIndex(uint32_t capacity)
    : capacity_(capacity), ring_(capacity) {
    if (capacity == 0)
        throw std::invalid_argument("KeyIndex capacity must be > 0");
    map_.reserve(capacity);
}

// ---------------------------------------------------------------------------
// put
// ---------------------------------------------------------------------------

uint32_t KeyIndex::put(Hash128 h, uint32_t mem_id,
                       uint32_t offset, uint32_t length,
                       uint16_t now_delta, uint16_t ttl_seconds) {
    auto it = map_.find(h.lo);
    if (it != map_.end()) {
        uint32_t idx = it->second;
        Entry& e = ring_[idx];
        if (e.flags == Entry::kOccupied && e.hash_hi == h.hi) {
            e.mem_id = mem_id;
            e.offset = offset;
            e.length = length;
            e.last_access.store(now_delta, std::memory_order_relaxed);
            e.freq.store(0, std::memory_order_relaxed);
            e.ttl_seconds = ttl_seconds;
            return idx;
        }
    }

    if (ring_used_ >= capacity_)
        evict_oldest(1);

    uint32_t idx = tail_;
    tail_ = next(tail_);

    Entry& e = ring_[idx];
    e.hash_lo = h.lo;
    e.hash_hi = h.hi;
    e.mem_id  = mem_id;
    e.offset  = offset;
    e.length  = length;
    e.last_access.store(now_delta, std::memory_order_relaxed);
    e.freq.store(0, std::memory_order_relaxed);
    e.ttl_seconds = ttl_seconds;
    e.flags   = Entry::kOccupied;

    map_[h.lo] = idx;
    ++size_;
    ++ring_used_;

    return idx;
}

// ---------------------------------------------------------------------------
// get
// ---------------------------------------------------------------------------

bool KeyIndex::get(Hash128 h, uint16_t now_delta, LookupResult& out) {
    auto it = map_.find(h.lo);
    if (it == map_.end()) return false;

    uint32_t idx = it->second;
    Entry& e = ring_[idx];
    if (e.flags != Entry::kOccupied || e.hash_hi != h.hi)
        return false;

    // TTL: 0 = no expiry. Otherwise expire when age >= ttl_seconds (wrap-safe 16-bit).
    if (e.ttl_seconds != 0) {
        uint16_t insert_time = e.last_access.load(std::memory_order_relaxed);
        uint16_t age = static_cast<uint16_t>(now_delta - insert_time);
        if (age >= e.ttl_seconds) {
            remove_entry(idx);
            return false;
        }
    }

    out.mem_id      = e.mem_id;
    out.offset      = e.offset;
    out.length      = e.length;
    out.last_access = e.last_access.load(std::memory_order_relaxed);
    out.freq        = e.freq.load(std::memory_order_relaxed);

    e.last_access.store(now_delta, std::memory_order_relaxed);
    morris_increment(e.freq);

    return true;
}

// ---------------------------------------------------------------------------
// remove
// ---------------------------------------------------------------------------

bool KeyIndex::remove(Hash128 h) {
    auto it = map_.find(h.lo);
    if (it == map_.end()) return false;

    Entry& e = ring_[it->second];
    if (e.hash_hi != h.hi) return false;

    e.flags = Entry::kDeleted;
    map_.erase(it);
    --size_;
    return true;
}

// ---------------------------------------------------------------------------
// evict_oldest
// ---------------------------------------------------------------------------

uint32_t KeyIndex::evict_oldest(uint32_t count, uint64_t* evicted_bytes) {
    uint32_t evicted = 0;
    while (evicted < count && ring_used_ > 0) {
        Entry& e = ring_[head_];
        if (e.flags == Entry::kOccupied) {
            if (evicted_bytes)
                *evicted_bytes += AlignedBuffer::align_up(e.length);
            map_.erase(e.hash_lo);
            e.flags = Entry::kDeleted;
            --size_;
        }
        head_ = next(head_);
        --ring_used_;
        ++evicted;
    }
    return evicted;
}

// ---------------------------------------------------------------------------
// entry_at
// ---------------------------------------------------------------------------

Entry* KeyIndex::entry_at(uint32_t ring_index) {
    if (ring_index >= capacity_) return nullptr;
    Entry& e = ring_[ring_index];
    if (e.flags != Entry::kOccupied) return nullptr;
    return &e;
}

const Entry* KeyIndex::entry_at(uint32_t ring_index) const {
    if (ring_index >= capacity_) return nullptr;
    const Entry& e = ring_[ring_index];
    if (e.flags != Entry::kOccupied) return nullptr;
    return &e;
}

// ---------------------------------------------------------------------------
// remove_entry — remove by ring index, compact head if possible
// ---------------------------------------------------------------------------

void KeyIndex::remove_entry(uint32_t ring_index) {
    if (ring_index >= capacity_) return;
    Entry& e = ring_[ring_index];
    if (e.flags != Entry::kOccupied) return;

    e.flags = Entry::kDeleted;
    map_.erase(e.hash_lo);
    --size_;

    // Compact head: advance past contiguous deleted/empty slots
    while (ring_used_ > 0 && ring_[head_].flags != Entry::kOccupied) {
        head_ = next(head_);
        --ring_used_;
    }
}

// ---------------------------------------------------------------------------
// Morris log counter increment (member xorshift RNG — single-threaded)
// ---------------------------------------------------------------------------

void KeyIndex::morris_increment(std::atomic<uint8_t>& freq) {
    uint8_t cur = freq.load(std::memory_order_relaxed);
    if (cur >= 255) return;
    rand_state_ ^= rand_state_ << 13;
    rand_state_ ^= rand_state_ >> 17;
    rand_state_ ^= rand_state_ << 5;
    if ((rand_state_ & ((1u << cur) - 1)) == 0)
        freq.store(cur + 1, std::memory_order_relaxed);
}
