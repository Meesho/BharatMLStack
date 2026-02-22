#pragma once

#include <atomic>
#include <cstdint>
#include <cstring>
#include <vector>

#include "absl/container/flat_hash_map.h"

// 128-bit hash produced by xxHash; split into lo (map key) and hi (collision check).
struct Hash128 {
    uint64_t lo;
    uint64_t hi;
};

// Compute a 128-bit hash of `key` (length `len`).
Hash128 hash_key(const void* key, size_t len);

// Per-key metadata stored in the entry ring.
// 32-byte aligned so two entries never share a cache line.
struct alignas(32) Entry {
    uint64_t                 hash_lo;      // map key, needed for reverse erase on eviction
    uint64_t                 hash_hi;      // collision check
    uint32_t                 mem_id;
    uint32_t                 offset;
    uint16_t                 length;
    std::atomic<uint16_t>    last_access;  // delta from epoch, updated with relaxed store
    std::atomic<uint8_t>     freq;         // Morris log counter, updated with relaxed store
    uint8_t                  flags;        // kEmpty / kOccupied / kDeleted
    uint16_t                 reserved;

    static constexpr uint8_t kEmpty    = 0;
    static constexpr uint8_t kOccupied = 1;
    static constexpr uint8_t kDeleted  = 2;
};

static_assert(sizeof(Entry) == 32, "Entry must be 32 bytes");

// Result of a successful get().
struct LookupResult {
    uint32_t mem_id;
    uint32_t offset;
    uint16_t length;
    uint16_t last_access;
    uint8_t  freq;
};

// Key index backed by absl::flat_hash_map (hash_lo -> ring index) and a flat
// circular entry ring.  Caller is responsible for external synchronization:
//   - get() is safe under a shared_lock (only atomic relaxed stores on freq/lastAccess)
//   - put(), remove(), evict_oldest() require an exclusive lock
class KeyIndex {
public:
    explicit KeyIndex(uint32_t capacity);

    // Insert or update.  Returns the ring slot index.
    uint32_t put(Hash128 h, uint32_t mem_id, uint32_t offset, uint16_t length);

    // Lookup.  Returns true and fills `out` if found.
    // Also bumps freq (Morris counter) and stores `now_delta` as last_access.
    bool get(Hash128 h, uint16_t now_delta, LookupResult& out);

    // Remove a single key.
    bool remove(Hash128 h);

    // Evict up to `count` oldest entries from the ring head.
    // Returns the number actually evicted.
    uint32_t evict_oldest(uint32_t count);

    uint32_t capacity() const { return capacity_; }
    uint32_t size()     const { return size_; }

private:
    // Advance `idx` by one position in the ring.
    uint32_t next(uint32_t idx) const { return (idx + 1) % capacity_; }

    // Probabilistic Morris counter increment.
    static void morris_increment(std::atomic<uint8_t>& freq);

    uint32_t capacity_;
    uint32_t size_ = 0;
    uint32_t head_ = 0;   // oldest occupied slot
    uint32_t tail_ = 0;   // next free slot

    std::vector<Entry> ring_;
    absl::flat_hash_map<uint64_t, uint32_t> map_;  // hash_lo -> ring index
};
