#include "memtable.h"
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
    if (used_ + len > buf_.size()) return -1;
    std::memcpy(buf_.bytes() + used_, data, len);
    auto off = static_cast<int32_t>(used_);
    used_ += len;
    return off;
}

bool Memtable::read(void* buf, size_t len, uint32_t offset) const {
    if (offset + len > used_) return false;
    std::memcpy(buf, buf_.bytes() + offset, len);
    return true;
}

int64_t Memtable::flush(RingDevice& dev) {
    if (used_ == 0) return -1;

    // Pad to block alignment (zero-fill tail).
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
// MemtableManager
// ===========================================================================

MemtableManager::MemtableManager(RingDevice& dev, size_t mt_size)
    : dev_(dev),
      mt_{Memtable(AlignedBuffer::align_up(mt_size)),
          Memtable(AlignedBuffer::align_up(mt_size))} {
    mt_[0].reset(next_id_++);
    mt_[1].reset(next_id_++);
    active_idx_ = 0;

    flush_thread_ = std::thread(&MemtableManager::flush_loop, this);
}

MemtableManager::~MemtableManager() {
    {
        std::lock_guard lk(mu_);
        shutdown_ = true;
    }
    flush_cv_.notify_one();
    if (flush_thread_.joinable()) flush_thread_.join();
}

// ---------------------------------------------------------------------------
// put  (hot path)
// ---------------------------------------------------------------------------

WriteResult MemtableManager::put(const void* data, size_t len) {
    std::unique_lock lk(mu_);

    // If active memtable can't fit this record, swap + schedule flush.
    if (mt_[active_idx_].remaining() < len) {
        // Back-pressure: wait until previous flush is done so the other
        // memtable is free to accept writes.
        swap_cv_.wait(lk, [&] { return !flush_pending_; });

        flushing_idx_ = active_idx_;
        active_idx_   = 1 - active_idx_;
        flush_pending_ = true;
        flush_cv_.notify_one();
    }

    Memtable& active = mt_[active_idx_];
    int32_t off = active.append(data, len);
    if (off < 0)
        throw std::runtime_error("record too large for memtable");

    return {active.id(), static_cast<uint32_t>(off),
            static_cast<uint16_t>(len)};
}

// ---------------------------------------------------------------------------
// try_read_from_memory
// ---------------------------------------------------------------------------

bool MemtableManager::try_read_from_memory(void* buf, size_t len,
                                           uint32_t mem_id, uint32_t offset) {
    std::shared_lock lk(mu_);
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
    std::lock_guard lk(info_mu_);
    auto it = flush_info_.find(mem_id);
    return it != flush_info_.end() ? it->second : -1;
}

// ---------------------------------------------------------------------------
// flush  (public: force-flush active memtable, synchronous)
// ---------------------------------------------------------------------------

void MemtableManager::flush() {
    std::unique_lock lk(mu_);
    if (mt_[active_idx_].used() == 0) return;

    swap_cv_.wait(lk, [&] { return !flush_pending_; });

    flushing_idx_ = active_idx_;
    active_idx_   = 1 - active_idx_;
    flush_pending_ = true;
    flush_cv_.notify_one();

    // Wait for this flush to complete.
    swap_cv_.wait(lk, [&] { return !flush_pending_; });
}

uint32_t MemtableManager::active_mem_id() const {
    std::shared_lock lk(mu_);
    return mt_[active_idx_].id();
}

// ---------------------------------------------------------------------------
// Background flush thread
// ---------------------------------------------------------------------------

void MemtableManager::flush_loop() {
    while (true) {
        std::unique_lock lk(mu_);
        flush_cv_.wait(lk, [&] { return flush_pending_ || shutdown_; });

        if (shutdown_ && !flush_pending_) return;

        int idx = flushing_idx_;
        lk.unlock();

        // Heavy I/O happens outside the lock.
        int64_t file_off = mt_[idx].flush(dev_);

        // Record the mapping so callers can resolve mem_id → file offset.
        {
            std::lock_guard ilk(info_mu_);
            flush_info_[mt_[idx].id()] = file_off;
        }

        lk.lock();
        mt_[idx].reset(next_id_++);
        flushing_idx_  = -1;
        flush_pending_ = false;
        lk.unlock();

        swap_cv_.notify_one();
    }
}
