#include "flashringc/delete_manager.h"

DeleteManager::DeleteManager(RingDevice& ring, KeyIndex& index,
                             double high_watermark, double low_watermark,
                             uint32_t n_delete, uint64_t discard_batch)
    : ring_(ring), index_(index),
      high_watermark_(high_watermark),
      low_watermark_(low_watermark),
      n_delete_(n_delete),
      discard_batch_(discard_batch) {}

uint64_t DeleteManager::ring_usage() const {
    uint64_t wo = ring_.write_offset();
    if (wo >= discard_cursor_)
        return wo - discard_cursor_;
    return ring_.capacity() - discard_cursor_ + wo;
}

uint32_t DeleteManager::maybe_evict() {
    double usage_ratio =
        static_cast<double>(ring_usage()) / static_cast<double>(ring_.capacity());
    if (usage_ratio < high_watermark_)
        return 0;
    if (usage_ratio <= low_watermark_)
        return 0;

    uint64_t evicted_bytes = 0;
    uint32_t evicted = index_.evict_oldest(n_delete_, &evicted_bytes);
    pending_discard_bytes_ += evicted_bytes;
    return evicted;
}

void DeleteManager::flush_discards() {
    if (pending_discard_bytes_ < discard_batch_)
        return;

    uint64_t cap = ring_.capacity();
    uint64_t offset = discard_cursor_ % cap;
    uint64_t length = pending_discard_bytes_;

    // Clamp to avoid wrapping past capacity in a single discard call.
    if (offset + length > cap)
        length = cap - offset;

    ring_.discard(offset, length);
    discard_cursor_ += pending_discard_bytes_;
    pending_discard_bytes_ = 0;
}
