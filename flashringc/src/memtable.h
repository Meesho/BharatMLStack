#pragma once

#include "aligned_buffer.h"
#include "ring_device.h"

#include <atomic>
#include <condition_variable>
#include <cstdint>
#include <mutex>
#include <shared_mutex>
#include <thread>
#include <unordered_map>

// Location of a record produced by MemtableManager::put().
struct WriteResult {
    uint32_t mem_id;   // memtable generation
    uint32_t offset;   // byte offset inside that memtable
    uint16_t length;   // record length
};

// A single aligned write buffer that accumulates records and flushes to disk
// as one sequential O_DIRECT write.
class Memtable {
public:
    explicit Memtable(size_t capacity);

    // Append `len` bytes.  Returns offset inside the buffer, or -1 if full.
    int32_t append(const void* data, size_t len);

    // Copy record at `offset` into `buf`.
    bool read(void* buf, size_t len, uint32_t offset) const;

    // Flush to ring device.  Returns the file offset, or -1 on error.
    int64_t flush(RingDevice& dev);

    void     reset(uint32_t new_id);
    uint32_t id()        const { return id_; }
    size_t   used()      const { return used_; }
    size_t   remaining() const { return buf_.size() - used_; }

private:
    AlignedBuffer buf_;
    uint32_t      id_   = 0;
    size_t        used_ = 0;
};

// Manages two memtables with double-buffering: one is active (receiving
// writes), the other is either idle or being flushed in the background.
class MemtableManager {
public:
    // `mt_size` should be block-aligned and much smaller than ring capacity.
    MemtableManager(RingDevice& dev, size_t mt_size);
    ~MemtableManager();

    MemtableManager(const MemtableManager&) = delete;
    MemtableManager& operator=(const MemtableManager&) = delete;

    // Append a record.  Triggers swap+flush when the active memtable is full.
    WriteResult put(const void* data, size_t len);

    // Try to serve a read from in-memory memtables (active or flushing).
    bool try_read_from_memory(void* buf, size_t len,
                              uint32_t mem_id, uint32_t offset);

    // Resolve a memtable id to the file offset where it was flushed.
    // Returns -1 if the id hasn't been flushed or has been overwritten.
    int64_t file_offset_for(uint32_t mem_id) const;

    // Force-flush the active memtable (blocks until done).
    void flush();

    uint32_t active_mem_id() const;

private:
    void flush_loop();

    RingDevice& dev_;

    Memtable mt_[2];
    int      active_idx_   = 0;   // index into mt_[]
    int      flushing_idx_ = -1;  // -1 = nothing flushing

    uint32_t next_id_ = 0;

    // flush-info: mem_id -> file offset (populated after flush)
    mutable std::mutex info_mu_;
    std::unordered_map<uint32_t, int64_t> flush_info_;

    // Synchronisation for the background flush thread.
    // shared_mutex: readers (try_read_from_memory) take shared_lock,
    // writers (put/flush/flush_loop) take unique_lock.
    mutable std::shared_mutex       mu_;
    std::condition_variable_any     flush_cv_;   // signals flush thread
    std::condition_variable_any     swap_cv_;    // signals put() after flush completes
    std::thread                     flush_thread_;
    bool                            flush_pending_ = false;
    bool                            shutdown_      = false;
};
