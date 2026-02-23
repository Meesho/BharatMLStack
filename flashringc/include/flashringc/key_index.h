#pragma once

#include "flashringc/common.h"

#include <atomic>
#include <cstdint>
#include <cstring>
#include <vector>

#include "absl/container/flat_hash_map.h"

// Per-key metadata stored in the entry ring.
// 32-byte aligned so two entries never share a cache line.
struct alignas(32) Entry {
    uint64_t                 hash_lo;
    uint64_t                 hash_hi;
    uint32_t                 mem_id;
    uint32_t                 offset;
    uint32_t                 length;
    std::atomic<uint16_t>    last_access;
    std::atomic<uint8_t>     freq;
    uint8_t                  flags;

    static constexpr uint8_t kEmpty    = 0;
    static constexpr uint8_t kOccupied = 1;
    static constexpr uint8_t kDeleted  = 2;
};

static_assert(sizeof(Entry) == 32, "Entry must be 32 bytes");

struct LookupResult {
    uint32_t mem_id;
    uint32_t offset;
    uint32_t length;
    uint16_t last_access;
    uint8_t  freq;
};

// Single-threaded key index (owned by a ShardReactor).
// No internal locking — the reactor is the sole accessor.
class KeyIndex {
public:
    explicit KeyIndex(uint32_t capacity);

    uint32_t put(Hash128 h, uint32_t mem_id, uint32_t offset, uint32_t length);
    bool get(Hash128 h, uint16_t now_delta, LookupResult& out);
    bool remove(Hash128 h);
    uint32_t evict_oldest(uint32_t count, uint64_t* evicted_bytes = nullptr);

    uint32_t capacity() const { return capacity_; }
    uint32_t size()     const { return size_; }

private:
    uint32_t next(uint32_t idx) const { return (idx + 1) % capacity_; }
    void morris_increment(std::atomic<uint8_t>& freq);

    uint32_t capacity_;
    uint32_t size_      = 0;
    uint32_t ring_used_ = 0;
    uint32_t head_      = 0;
    uint32_t tail_      = 0;
    uint32_t rand_state_ = 2654435769u;

    std::vector<Entry> ring_;
    absl::flat_hash_map<uint64_t, uint32_t> map_;
};
