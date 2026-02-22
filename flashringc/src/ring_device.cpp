#include "ring_device.h"

#include <cerrno>
#include <cstring>
#include <stdexcept>

#include <fcntl.h>
#include <sys/stat.h>
#include <unistd.h>

#ifdef __linux__
#include <linux/fs.h>
#include <sys/ioctl.h>
#endif

#ifdef __APPLE__
#include <sys/disk.h>
#endif

// ---------------------------------------------------------------------------
// Platform helpers
// ---------------------------------------------------------------------------

namespace {

int open_direct(const char* path, int extra_flags, mode_t mode = 0) {
#ifdef __linux__
    return ::open(path, extra_flags | O_DIRECT, mode);
#else
    // macOS: open normally, then disable kernel caching.
    int fd = ::open(path, extra_flags, mode);
    if (fd >= 0) ::fcntl(fd, F_NOCACHE, 1);
    return fd;
#endif
}

int preallocate_file(int fd, uint64_t size) {
#ifdef __linux__
    return ::fallocate(fd, 0, 0, static_cast<off_t>(size));
#else
    return ::ftruncate(fd, static_cast<off_t>(size));
#endif
}

uint64_t block_device_size(int fd) {
    uint64_t sz = 0;
#ifdef __linux__
    if (::ioctl(fd, BLKGETSIZE64, &sz) < 0) sz = 0;
#elif defined(__APPLE__)
    uint32_t blk_size = 0;
    uint64_t blk_count = 0;
    if (::ioctl(fd, DKIOCGETBLOCKSIZE, &blk_size) == 0 &&
        ::ioctl(fd, DKIOCGETBLOCKCOUNT, &blk_count) == 0)
        sz = blk_size * blk_count;
#endif
    return sz;
}

} // namespace

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

RingDevice::~RingDevice() {
    if (fd_ >= 0) ::close(fd_);
}

RingDevice::RingDevice(RingDevice&& o) noexcept
    : fd_(o.fd_), capacity_(o.capacity_), write_offset_(o.write_offset_),
      wrapped_(o.wrapped_), is_blk_(o.is_blk_) {
    o.fd_ = -1;
}

RingDevice& RingDevice::operator=(RingDevice&& o) noexcept {
    if (this != &o) {
        if (fd_ >= 0) ::close(fd_);
        fd_           = o.fd_;
        capacity_     = o.capacity_;
        write_offset_ = o.write_offset_;
        wrapped_      = o.wrapped_;
        is_blk_       = o.is_blk_;
        o.fd_ = -1;
    }
    return *this;
}

// ---------------------------------------------------------------------------
// open
// ---------------------------------------------------------------------------

RingDevice RingDevice::open(const std::string& path, uint64_t capacity) {
    RingDevice dev;
    struct stat st{};

    bool exists = (::stat(path.c_str(), &st) == 0);

    if (exists && S_ISBLK(st.st_mode)) {
        // --- Block device ---
        dev.fd_ = open_direct(path.c_str(), O_RDWR);
        if (dev.fd_ < 0)
            throw std::runtime_error("open block device: " +
                                     std::string(std::strerror(errno)));
        dev.is_blk_   = true;
        dev.capacity_ = block_device_size(dev.fd_);
        if (dev.capacity_ == 0)
            throw std::runtime_error("cannot determine block device size");
    } else {
        // --- Regular file ---
        if (capacity == 0)
            throw std::invalid_argument("capacity required for file-backed ring");

        capacity = AlignedBuffer::align_up(capacity);

        dev.fd_ = open_direct(path.c_str(), O_RDWR | O_CREAT, 0644);
        if (dev.fd_ < 0)
            throw std::runtime_error("open file: " +
                                     std::string(std::strerror(errno)));

        if (preallocate_file(dev.fd_, capacity) < 0)
            throw std::runtime_error("preallocate: " +
                                     std::string(std::strerror(errno)));

        dev.is_blk_   = false;
        dev.capacity_ = capacity;
    }

    return dev;
}

// ---------------------------------------------------------------------------
// write  (sequential, wrap-around)
// ---------------------------------------------------------------------------

int64_t RingDevice::write(const void* buf, size_t len) {
    if (len == 0 || (len & (kBlockSize - 1)) != 0) return -1;

    // If remaining tail can't fit this write, wrap to 0.
    if (write_offset_ + len > capacity_) {
        write_offset_ = 0;
        wrapped_ = true;
    }

    ssize_t n = ::pwrite(fd_, buf, len, static_cast<off_t>(write_offset_));
    if (n != static_cast<ssize_t>(len)) return -1;

    int64_t written_at = static_cast<int64_t>(write_offset_);
    write_offset_ += len;

    if (write_offset_ >= capacity_) {
        write_offset_ = 0;
        wrapped_ = true;
    }

    return written_at;
}

// ---------------------------------------------------------------------------
// read  (block-aligned)
// ---------------------------------------------------------------------------

ssize_t RingDevice::read(void* buf, size_t len, uint64_t offset) const {
    if (len == 0 || (len & (kBlockSize - 1)) != 0) return -1;
    if ((offset & (kBlockSize - 1)) != 0) return -1;
    if (offset + len > capacity_) return -1;

    return ::pread(fd_, buf, len, static_cast<off_t>(offset));
}

// ---------------------------------------------------------------------------
// read_unaligned  (arbitrary offset/length, handles alignment internally)
// ---------------------------------------------------------------------------

ssize_t RingDevice::read_unaligned(void* buf, size_t len, uint64_t offset) const {
    if (len == 0 || offset + len > capacity_) return -1;

    uint64_t aligned_start = offset & ~(uint64_t)(kBlockSize - 1);
    uint64_t aligned_end   = AlignedBuffer::align_up(offset + len);
    if (aligned_end > capacity_) aligned_end = capacity_;
    size_t   aligned_len   = static_cast<size_t>(aligned_end - aligned_start);

    auto tmp = AlignedBuffer::allocate(aligned_len);
    if (!tmp.valid()) return -1;

    ssize_t n = ::pread(fd_, tmp.data(), aligned_len,
                        static_cast<off_t>(aligned_start));
    if (n < static_cast<ssize_t>(aligned_len)) return -1;

    size_t skip = static_cast<size_t>(offset - aligned_start);
    std::memcpy(buf, tmp.bytes() + skip, len);
    return static_cast<ssize_t>(len);
}

// ---------------------------------------------------------------------------
// discard  (BLKDISCARD on block devices, no-op otherwise)
// ---------------------------------------------------------------------------

int RingDevice::discard(uint64_t offset, uint64_t length) {
#ifdef __linux__
    if (!is_blk_) return 0;
    uint64_t range[2] = {offset, length};
    return ::ioctl(fd_, BLKDISCARD, range);
#else
    (void)offset; (void)length;
    return 0;
#endif
}
