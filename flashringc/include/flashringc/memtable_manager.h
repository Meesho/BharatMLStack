#pragma once

#include "flashringc/common.h"
#include "flashringc/ring_device.h"

#include <cstdint>
#include <unordered_map>

struct WriteResult {
    uint32_t mem_id;
    uint32_t offset;
    uint32_t length;
};

// Returned by swap_and_get_flush_buffer() for the reactor to submit io_uring.
struct FlushBuffer {
    const void* data;
    size_t      len;       // block-aligned size to write
    uint32_t    mem_id;    // generation id of the memtable being flushed
    int         slot_idx;  // which mt_[] slot was flushed (for complete_flush)
};

// A single aligned write buffer that accumulates records.
class Memtable {
public:
    explicit Memtable(size_t capacity);

    int32_t append(const void* data, size_t len);
    bool read(void* buf, size_t len, uint32_t offset) const;

    // Synchronous flush (used by macOS fallback path).
    int64_t flush(RingDevice& dev);

    void     reset(uint32_t new_id);
    uint32_t id()        const { return id_; }
    size_t   used()      const { return used_; }
    size_t   remaining() const { return buf_.size() - used_; }

    // True if one more record of size len can fit with block-aligned packing.
    bool can_fit(size_t len) const;

    const void* data()   const { return buf_.data(); }

private:
    AlignedBuffer buf_;
    uint32_t      id_   = 0;
    size_t        used_ = 0;
};

// Single-threaded double-buffered memtable manager (owned by ShardReactor).
// No internal locks or background threads — the reactor drives everything.
class MemtableManager {
public:
    MemtableManager(RingDevice& dev, size_t mt_size);
    ~MemtableManager() = default;

    MemtableManager(const MemtableManager&) = delete;
    MemtableManager& operator=(const MemtableManager&) = delete;

    // Append a record to the active memtable.
    WriteResult put(const void* data, size_t len);

    // True if the active memtable cannot fit `len` more bytes.
    bool needs_flush(size_t next_record_len) const;

    // Swap active ↔ flushing, return the full buffer for io_uring write.
    // Caller must eventually call complete_flush() after the IO completes.
    FlushBuffer swap_and_get_flush_buffer();

    // Called after io_uring flush CQE: records the file offset, resets buffer.
    void complete_flush(uint32_t mem_id, int64_t file_offset);

    // Synchronous flush via pwrite (macOS fallback / shutdown path).
    void flush_sync();

    // Serve a read from in-memory memtables (active or flushing).
    bool try_read_from_memory(void* buf, size_t len,
                              uint32_t mem_id, uint32_t offset);

    // Resolve mem_id → file offset. Returns -1 if not flushed yet.
    int64_t file_offset_for(uint32_t mem_id) const;

    uint32_t active_mem_id() const;

    bool is_flush_pending() const { return flush_pending_; }

private:
    RingDevice& dev_;

    Memtable mt_[2];
    int      active_idx_   = 0;
    int      flushing_idx_ = -1;
    bool     flush_pending_ = false;

    uint32_t next_id_ = 0;

    std::unordered_map<uint32_t, int64_t> flush_info_;
};
