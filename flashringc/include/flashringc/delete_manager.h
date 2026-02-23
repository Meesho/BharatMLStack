#pragma once

#include "flashringc/key_index.h"
#include "flashringc/ring_device.h"

#include <cstdint>

class DeleteManager {
public:
    DeleteManager(RingDevice& ring, KeyIndex& index,
                  double high_watermark = 0.75,
                  uint32_t n_delete = 10,
                  uint64_t discard_batch = 64 * 1024 * 1024);

    uint32_t maybe_evict();
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
