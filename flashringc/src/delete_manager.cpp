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
    const uint64_t cap = ring_.capacity();
    uint32_t total_evicted = 0;

    while (true) {
        double usage_ratio =
            static_cast<double>(ring_usage()) / static_cast<double>(cap);
        if (usage_ratio < high_watermark_)
            return total_evicted;
        if (usage_ratio <= low_watermark_)
            return total_evicted;

        uint64_t evicted_bytes = 0;
        uint32_t evicted = index_.evict_oldest(n_delete_, &evicted_bytes);
        if (evicted == 0) {
            flush_discards_remainder();
            break;
        }
        total_evicted += evicted;
        pending_discard_bytes_ += evicted_bytes;

        // Flush when we have enough pending so discard_cursor_ advances and usage drops.
        flush_discards();
    }
    return total_evicted;
}

void DeleteManager::flush_discards() {
    if (pending_discard_bytes_ < discard_batch_)
        return;

    const uint64_t cap = ring_.capacity();

    // Flush one or two chunks so we never advance cursor by more than we actually discard.
    // First chunk may be clamped at wrap; then flush remainder from start of ring.
    uint64_t offset = discard_cursor_ % cap;
    uint64_t length = pending_discard_bytes_;
    if (offset + length > cap)
        length = cap - offset;

    ring_.discard(offset, length);
    discard_cursor_ += length;
    pending_discard_bytes_ -= length;

    // If we clamped, flush the remainder from logical start (physical offset 0).
    if (pending_discard_bytes_ > 0) {
        length = pending_discard_bytes_;
        if (length > cap)
            length = cap;
        ring_.discard(0, length);
        discard_cursor_ += length;
        pending_discard_bytes_ -= length;
    }
}

void DeleteManager::flush_discards_remainder() {
    if (pending_discard_bytes_ == 0)
        return;
    const uint64_t cap = ring_.capacity();
    uint64_t offset = discard_cursor_ % cap;
    uint64_t length = pending_discard_bytes_;
    if (offset + length > cap)
        length = cap - offset;
    if (length == 0)
        return;
    ring_.discard(offset, length);
    discard_cursor_ += length;
    pending_discard_bytes_ -= length;
    if (pending_discard_bytes_ > 0) {
        length = std::min(pending_discard_bytes_, cap);
        ring_.discard(0, length);
        discard_cursor_ += length;
        pending_discard_bytes_ -= length;
    }
}
