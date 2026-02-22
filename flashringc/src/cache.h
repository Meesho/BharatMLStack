#pragma once

#include "key_index.h"
#include "memtable.h"
#include "ring_device.h"

#include <cstdint>
#include <memory>
#include <shared_mutex>
#include <string>

struct CacheConfig {
    std::string device_path;
    uint64_t    ring_capacity  = 0;          // 0 = auto-detect (block devices)
    size_t      memtable_size  = 64 << 20;   // 64 MB
    uint32_t    index_capacity = 1'000'000;  // max keys
};

class Cache {
public:
    static Cache open(const CacheConfig& cfg);
    ~Cache();

    Cache(Cache&&) noexcept;
    Cache& operator=(Cache&&) noexcept;
    Cache(const Cache&) = delete;
    Cache& operator=(const Cache&) = delete;

    bool put(const void* key, size_t key_len,
             const void* val, size_t val_len);

    // Returns true if found.  value is copied into val_buf (up to val_buf_len).
    // *actual_len is set to the stored value's length regardless of truncation.
    bool get(const void* key, size_t key_len,
             void* val_buf, size_t val_buf_len, size_t* actual_len);

    bool remove(const void* key, size_t key_len);

    void flush();

    uint32_t key_count()  const;
    uint64_t ring_usage() const;

private:
    Cache() = default;

    std::unique_ptr<RingDevice>      ring_;
    std::unique_ptr<MemtableManager> memtables_;
    std::unique_ptr<KeyIndex>        index_;
    mutable std::shared_mutex        mu_;   // protects index_

    bool read_record(const LookupResult& lr, std::vector<uint8_t>& buf);
};
