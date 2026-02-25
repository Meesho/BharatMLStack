#pragma once

#include "flashringc/common.h"

#include <cstdint>
#include <string>

// Circular ring buffer backed by a raw block device or pre-allocated file.
//
// Uses separate file descriptors for reads and writes so they can proceed
// on independent NVMe submission queues without kernel-level fd contention.
// Both fds use O_DIRECT to bypass the kernel page cache.
// On block devices, BLKDISCARD sends TRIM to the SSD controller.
class RingDevice {
public:
    ~RingDevice();

    RingDevice(RingDevice&& o) noexcept;
    RingDevice& operator=(RingDevice&& o) noexcept;
    RingDevice(const RingDevice&) = delete;
    RingDevice& operator=(const RingDevice&) = delete;

    static RingDevice open(const std::string& path, uint64_t capacity = 0);

    static RingDevice open_region(const std::string& path,
                                  uint64_t base_offset,
                                  uint64_t region_capacity);

    static bool is_block_device_path(const std::string& path);

    int64_t write(const void* buf, size_t len);
    ssize_t read(void* buf, size_t len, uint64_t offset) const;
    ssize_t read_unaligned(void* buf, size_t len, uint64_t offset) const;
    int discard(uint64_t offset, uint64_t length);

    uint64_t capacity()        const { return capacity_; }
    uint64_t write_offset()    const { return write_offset_; }
    uint64_t wrap_count()      const { return wrap_count_; }
    bool     wrapped()         const { return wrapped_; }
    bool     is_block_device() const { return is_blk_; }
    int      read_fd()         const { return read_fd_; }
    int      write_fd()        const { return write_fd_; }
    uint64_t base_offset()     const { return base_offset_; }

    uint64_t memtable_size()      const { return memtable_size_; }
    void     set_memtable_size(uint64_t sz) { memtable_size_ = sz; }

    uint64_t discard_cursor()     const { return discard_cursor_; }
    void     advance_discard_cursor(uint64_t bytes);

    double   utilization() const;

private:
    RingDevice() = default;

    int      write_fd_       = -1;
    int      read_fd_        = -1;
    uint64_t capacity_       = 0;
    uint64_t base_offset_    = 0;
    uint64_t write_offset_   = 0;
    uint64_t wrap_count_     = 0;
    uint64_t discard_cursor_ = 0;
    uint64_t memtable_size_  = 0;
    bool     wrapped_        = false;
    bool     is_blk_         = false;
};
