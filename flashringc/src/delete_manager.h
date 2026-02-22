#pragma once

#include "key_index.h"
#include "ring_device.h"

#include <cstdint>

class DeleteManager {
public:
    DeleteManager(RingDevice& ring, KeyIndex& index,
                  double high_watermark = 0.75,
                  uint32_t n_delete = 10,
                  uint64_t discard_batch = 64 * 1024 * 1024);

    // Called inside put()'s unique_lock.
    // Evicts n_delete entries if ring usage exceeds the watermark.
    // Returns number of entries evicted.
    uint32_t maybe_evict();

    // Issue pending BLKDISCARD if accumulated bytes >= discard_batch.
    // Safe to call outside the lock.
    void flush_discards();

private:
    uint64_t ring_usage() const;

    RingDevice& ring_;
    KeyIndex&   index_;
    double      high_watermark_;
    uint32_t    n_delete_;
    uint64_t    discard_batch_;
    uint64_t    pending_discard_bytes_ = 0;
    uint64_t    discard_cursor_        = 0;
};
