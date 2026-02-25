#include "flashringc/memtable_manager.h"

#include <cstring>
#include <stdexcept>

// ===========================================================================
// Memtable
// ===========================================================================

Memtable::Memtable(size_t capacity)
    : buf_(AlignedBuffer::allocate(capacity)) {
    if (!buf_.valid())
        throw std::runtime_error("failed to allocate aligned memtable buffer");
}

int32_t Memtable::append(const void* data, size_t len) {
    if (len <= kBlockSize) {
        // Small record: pack within current block, never cross boundary
        size_t block_used = used_ % kBlockSize;
        size_t block_remaining =
            (block_used == 0) ? kBlockSize : kBlockSize - block_used;
        if (len > block_remaining) {
            // Pad to next block boundary
            size_t pad = block_remaining;
            if (used_ + pad + len > buf_.size()) return -1;
            std::memset(buf_.bytes() + used_, 0, pad);
            used_ += pad;
        } else {
            if (used_ + len > buf_.size()) return -1;
        }
        auto off = static_cast<int32_t>(used_);
        std::memcpy(buf_.bytes() + used_, data, len);
        used_ += len;
        return off;
    } else {
        // Large record: align start to 4KB boundary
        size_t aligned_start = AlignedBuffer::align_up(used_);
        size_t aligned_len = AlignedBuffer::align_up(len);
        if (aligned_start + aligned_len > buf_.size()) return -1;
        if (aligned_start > used_)
            std::memset(buf_.bytes() + used_, 0, aligned_start - used_);
        std::memcpy(buf_.bytes() + aligned_start, data, len);
        if (aligned_len > len)
            std::memset(buf_.bytes() + aligned_start + len, 0,
                        aligned_len - len);
        used_ = aligned_start + aligned_len;
        return static_cast<int32_t>(aligned_start);
    }
}

bool Memtable::can_fit(size_t len) const {
    if (len <= kBlockSize) {
        size_t block_used = used_ % kBlockSize;
        size_t block_remaining =
            (block_used == 0) ? kBlockSize : kBlockSize - block_used;
        size_t space_needed = (len > block_remaining) ? (block_remaining + len) : len;
        return used_ + space_needed <= buf_.size();
    } else {
        size_t aligned_start = AlignedBuffer::align_up(used_);
        size_t aligned_len = AlignedBuffer::align_up(len);
        return aligned_start + aligned_len <= buf_.size();
    }
}

bool Memtable::read(void* buf, size_t len, uint32_t offset) const {
    if (offset + len > used_) return false;
    std::memcpy(buf, buf_.bytes() + offset, len);
    return true;
}

int64_t Memtable::flush(RingDevice& dev) {
    if (used_ == 0) return -1;

    size_t aligned = AlignedBuffer::align_up(used_);
    if (aligned > used_)
        std::memset(buf_.bytes() + used_, 0, aligned - used_);

    return dev.write(buf_.data(), aligned);
}

void Memtable::reset(uint32_t new_id) {
    id_   = new_id;
    used_ = 0;
}

// ===========================================================================
// MemtableManager (single-threaded, reactor-driven)
// ===========================================================================

MemtableManager::MemtableManager(RingDevice& dev, size_t mt_size)
    : dev_(dev),
      mt_{Memtable(AlignedBuffer::align_up(mt_size)),
          Memtable(AlignedBuffer::align_up(mt_size))} {
    mt_[0].reset(next_id_++);
    mt_[1].reset(next_id_++);
    active_idx_ = 0;
}

// ---------------------------------------------------------------------------
// put
// ---------------------------------------------------------------------------

WriteResult MemtableManager::put(const void* data, size_t len) {
    Memtable& active = mt_[active_idx_];
    int32_t off = active.append(data, len);
    if (off < 0)
        throw std::runtime_error("record too large for memtable");

    return {active.id(), static_cast<uint32_t>(off),
            static_cast<uint32_t>(len)};
}

// ---------------------------------------------------------------------------
// needs_flush
// ---------------------------------------------------------------------------

bool MemtableManager::needs_flush(size_t next_record_len) const {
    return !mt_[active_idx_].can_fit(next_record_len);
}

// ---------------------------------------------------------------------------
// swap_and_get_flush_buffer
// ---------------------------------------------------------------------------

FlushBuffer MemtableManager::swap_and_get_flush_buffer() {
    flushing_idx_  = active_idx_;
    active_idx_    = 1 - active_idx_;
    flush_pending_ = true;

    Memtable& flushing = mt_[flushing_idx_];
    size_t aligned = AlignedBuffer::align_up(flushing.used());

    // Zero-pad tail for O_DIRECT alignment.
    if (aligned > flushing.used()) {
        auto* p = const_cast<uint8_t*>(
            static_cast<const uint8_t*>(flushing.data()));
        std::memset(p + flushing.used(), 0, aligned - flushing.used());
    }

    // Compute the ring offset where this write will land.
    // RingDevice::write() handles wrap-around internally, but we need the
    // offset for flush_info bookkeeping. We'll get the real offset from
    // the reactor after the write completes.
    return {flushing.data(), aligned, flushing.id(), flushing_idx_};
}

// ---------------------------------------------------------------------------
// complete_flush
// ---------------------------------------------------------------------------

void MemtableManager::complete_flush(uint32_t mem_id, int64_t file_offset) {
    flush_info_[mem_id] = file_offset;
    mt_[flushing_idx_].reset(next_id_++);
    flushing_idx_  = -1;
    flush_pending_ = false;
}

// ---------------------------------------------------------------------------
// flush_sync — synchronous flush (macOS fallback / shutdown)
// ---------------------------------------------------------------------------

void MemtableManager::flush_sync() {
    if (mt_[active_idx_].used() == 0) return;

    flushing_idx_  = active_idx_;
    active_idx_    = 1 - active_idx_;
    flush_pending_ = true;

    int64_t file_off = mt_[flushing_idx_].flush(dev_);
    flush_info_[mt_[flushing_idx_].id()] = file_off;
    mt_[flushing_idx_].reset(next_id_++);
    flushing_idx_  = -1;
    flush_pending_ = false;
}

// ---------------------------------------------------------------------------
// try_read_from_memory
// ---------------------------------------------------------------------------

bool MemtableManager::try_read_from_memory(void* buf, size_t len,
                                           uint32_t mem_id, uint32_t offset) {
    for (int i = 0; i < 2; ++i) {
        if (mt_[i].id() == mem_id)
            return mt_[i].read(buf, len, offset);
    }
    return false;
}

// ---------------------------------------------------------------------------
// file_offset_for
// ---------------------------------------------------------------------------

int64_t MemtableManager::file_offset_for(uint32_t mem_id) const {
    auto it = flush_info_.find(mem_id);
    return it != flush_info_.end() ? it->second : -1;
}

uint32_t MemtableManager::active_mem_id() const {
    return mt_[active_idx_].id();
}
