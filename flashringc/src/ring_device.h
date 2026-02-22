#pragma once

#include "aligned_buffer.h"
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

    // Open a block device (capacity auto-detected) or file (pre-allocated to
    // `capacity` bytes, must be > 0).
    static RingDevice open(const std::string& path, uint64_t capacity = 0);

    // Open a region of an existing block device or file.
    // The ring operates within [base_offset, base_offset + region_capacity).
    static RingDevice open_region(const std::string& path,
                                  uint64_t base_offset,
                                  uint64_t region_capacity);

    // Returns true if `path` refers to a block device.
    static bool is_block_device_path(const std::string& path);

    // Append `len` bytes (must be block-aligned).
    // Returns the offset where data was written, or -1 on error.
    // Wraps to offset 0 when remaining space < len.
    int64_t write(const void* buf, size_t len);

    // Block-aligned read at an arbitrary offset.
    ssize_t read(void* buf, size_t len, uint64_t offset) const;

    // Convenience: read arbitrary (non-aligned) offset+length.
    // Allocates a temporary aligned buffer internally.
    ssize_t read_unaligned(void* buf, size_t len, uint64_t offset) const;

    // BLKDISCARD on block devices; no-op on files.
    int discard(uint64_t offset, uint64_t length);

    uint64_t capacity()     const { return capacity_; }
    uint64_t write_offset() const { return write_offset_; }
    bool     wrapped()      const { return wrapped_; }
    bool     is_block_device() const { return is_blk_; }
    int      read_fd()      const { return read_fd_; }
    uint64_t base_offset()  const { return base_offset_; }

private:
    RingDevice() = default;

    int      write_fd_     = -1;   // O_DIRECT fd used exclusively for pwrite
    int      read_fd_      = -1;   // O_DIRECT fd used exclusively for pread
    uint64_t capacity_     = 0;
    uint64_t base_offset_  = 0;    // start of this ring's region on the device
    uint64_t write_offset_ = 0;
    bool     wrapped_      = false;
    bool     is_blk_       = false;
};
