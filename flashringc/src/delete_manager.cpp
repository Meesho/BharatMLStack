#include "flashringc/delete_manager.h"
#include "flashringc/key_index.h"
#include "flashringc/ring_device.h"

DeleteManager::DeleteManager(const Config& cfg,
                             KeyIndex& index,
                             RingDevice& device)
    : cfg_(cfg), index_(index), device_(device) {}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

void DeleteManager::on_put() {
    maybe_discard_blocks();

    if (cleaning_in_progress_) {
        int delta = static_cast<int>(discarded_through_mem_id_ - cleaning_mem_id_);
        int effective_k = cfg_.base_k * (1 + delta);
        amortize_index_cleanup(effective_k);
    }
}

void DeleteManager::on_tick() {
    if (cleaning_in_progress_) {
        int delta = static_cast<int>(discarded_through_mem_id_ - cleaning_mem_id_);
        int effective_k = cfg_.base_k * (1 + delta);
        amortize_index_cleanup(effective_k);
    }
}

bool DeleteManager::has_backlog() const {
    return cleaning_in_progress_;
}

bool DeleteManager::is_discarded(uint32_t mem_id) const {
    return has_discarded_ && mem_id <= discarded_through_mem_id_;
}

uint32_t DeleteManager::discarded_through_mem_id() const {
    return discarded_through_mem_id_;
}

bool DeleteManager::is_evicting() const {
    return evicting_;
}

void DeleteManager::reset() {
    evicting_                 = false;
    oldest_mem_id_            = 0;
    discarded_through_mem_id_ = 0;
    has_discarded_            = false;
    cleaning_mem_id_          = 0;
    cleaning_cursor_          = 0;
    cleaning_in_progress_     = false;
}

// ---------------------------------------------------------------------------
// Hysteresis-Aware Block Discard
// ---------------------------------------------------------------------------

bool DeleteManager::needs_discard() const {
    double dev_util = device_.utilization();
    double idx_util = index_.utilization();

    if (evicting_) {
        return dev_util > cfg_.clear_threshold
            || (dev_util > 0.0 && idx_util > cfg_.clear_threshold);
    } else {
        return dev_util > cfg_.eviction_threshold
            || (dev_util > 0.0 && idx_util > cfg_.eviction_threshold);
    }
}

void DeleteManager::maybe_discard_blocks() {
    if (!evicting_ && !needs_discard()) return;

    evicting_ = true;

    int discards = 0;
    while (needs_discard() && discards < cfg_.max_discards_per_put) {
        discard_oldest_block();
        discards++;
    }

    if (!needs_discard()) {
        evicting_ = false;
    }
}

void DeleteManager::discard_oldest_block() {
    uint32_t mem_id = oldest_mem_id_;
    uint64_t offset = memid_to_block_offset(mem_id);
    uint64_t size = device_.memtable_size();

    device_.discard(offset, size);
    device_.advance_discard_cursor(size);

    oldest_mem_id_++;
    discarded_through_mem_id_ = mem_id;
    has_discarded_ = true;

    if (!cleaning_in_progress_) {
        cleaning_mem_id_ = mem_id;
        cleaning_cursor_ = index_.ring_head();
        cleaning_in_progress_ = true;
    }
}

// ---------------------------------------------------------------------------
// Amortized Index Cleanup
// ---------------------------------------------------------------------------

void DeleteManager::amortize_index_cleanup(int effective_k) {
    int cleaned = 0;

    while (cleaned < effective_k && cleaning_in_progress_) {
        if (cleaning_cursor_ == index_.ring_tail()) {
            advance_to_next_memid();
            break;
        }

        auto* entry = index_.entry_at(cleaning_cursor_);
        if (entry == nullptr) {
            cleaning_cursor_ = (cleaning_cursor_ + 1) % index_.capacity();
            continue;
        }

        if (entry->mem_id == cleaning_mem_id_) {
            index_.remove_entry(cleaning_cursor_);
            cleaning_cursor_ = (cleaning_cursor_ + 1) % index_.capacity();
            cleaned++;
        } else if (entry->mem_id > cleaning_mem_id_) {
            advance_to_next_memid();
            if (!cleaning_in_progress_) break;
        } else {
            cleaning_cursor_ = (cleaning_cursor_ + 1) % index_.capacity();
        }
    }
}

void DeleteManager::advance_to_next_memid() {
    if (cleaning_mem_id_ >= discarded_through_mem_id_) {
        cleaning_in_progress_ = false;
    } else {
        cleaning_mem_id_++;
    }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

uint64_t DeleteManager::memid_to_block_offset(uint32_t mem_id) const {
    uint64_t mt_size = device_.memtable_size();
    if (mt_size == 0) return 0;
    uint64_t num_blocks = device_.capacity() / mt_size;
    if (num_blocks == 0) return 0;
    return (static_cast<uint64_t>(mem_id) % num_blocks) * mt_size;
}
