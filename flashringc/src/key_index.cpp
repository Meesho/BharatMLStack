#include "key_index.h"

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
// Fast thread-local xorshift for Morris counter probability check.
// ---------------------------------------------------------------------------

namespace {

uint32_t fast_rand() {
    // Per-thread state avoids atomic contention.
    thread_local uint32_t s = 2654435769u ^ static_cast<uint32_t>(
        reinterpret_cast<uintptr_t>(&s));
    s ^= s << 13;
    s ^= s >> 17;
    s ^= s << 5;
    return s;
}

} // namespace

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
                       uint32_t offset, uint16_t length) {
    // Check for existing entry (update case).
    auto it = map_.find(h.lo);
    if (it != map_.end()) {
        uint32_t idx = it->second;
        Entry& e = ring_[idx];
        if (e.flags == Entry::kOccupied && e.hash_hi == h.hi) {
            e.mem_id = mem_id;
            e.offset = offset;
            e.length = length;
            e.last_access.store(0, std::memory_order_relaxed);
            e.freq.store(0, std::memory_order_relaxed);
            return idx;
        }
    }

    // If ring is full, evict one entry at head to make room.
    if (size_ >= capacity_)
        evict_oldest(1);

    uint32_t idx = tail_;
    tail_ = next(tail_);

    Entry& e = ring_[idx];
    e.hash_lo = h.lo;
    e.hash_hi = h.hi;
    e.mem_id  = mem_id;
    e.offset  = offset;
    e.length  = length;
    e.last_access.store(0, std::memory_order_relaxed);
    e.freq.store(0, std::memory_order_relaxed);
    e.flags   = Entry::kOccupied;

    map_[h.lo] = idx;
    ++size_;

    return idx;
}

// ---------------------------------------------------------------------------
// get
// ---------------------------------------------------------------------------

bool KeyIndex::get(Hash128 h, uint16_t now_delta, LookupResult& out) {
    auto it = map_.find(h.lo);
    if (it == map_.end()) return false;

    Entry& e = ring_[it->second];
    if (e.flags != Entry::kOccupied || e.hash_hi != h.hi)
        return false;

    out.mem_id      = e.mem_id;
    out.offset      = e.offset;
    out.length      = e.length;
    out.last_access = e.last_access.load(std::memory_order_relaxed);
    out.freq        = e.freq.load(std::memory_order_relaxed);

    // Update access metadata (safe under shared_lock — relaxed atomics).
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

uint32_t KeyIndex::evict_oldest(uint32_t count) {
    uint32_t evicted = 0;
    while (evicted < count && head_ != tail_) {
        Entry& e = ring_[head_];
        if (e.flags == Entry::kOccupied) {
            map_.erase(e.hash_lo);
            e.flags = Entry::kDeleted;
            --size_;
        }
        head_ = next(head_);
        ++evicted;
    }
    return evicted;
}

// ---------------------------------------------------------------------------
// Morris log counter increment
// ---------------------------------------------------------------------------

void KeyIndex::morris_increment(std::atomic<uint8_t>& freq) {
    uint8_t cur = freq.load(std::memory_order_relaxed);
    if (cur >= 255) return;
    // Increment with probability 1 / 2^cur.
    if ((fast_rand() & ((1u << cur) - 1)) == 0)
        freq.store(cur + 1, std::memory_order_relaxed);
}
