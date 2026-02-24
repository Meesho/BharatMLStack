#pragma once

#include <cstdint>
#include <cstring>

#include "absl/crc/crc32c.h"
#include "absl/strings/string_view.h"

// On-disk / in-memtable record layout:
//   [key_len: 4B] [val_len: 4B] [key_data: key_len B] [val_data: val_len B] [crc32: 4B]
// CRC32C is computed over the first (8 + key_len + val_len) bytes, stored LE at end.

static constexpr size_t kRecordHeaderSize = 8;
static constexpr size_t kRecordCrcSize = 4;

inline size_t record_size(size_t key_len, size_t val_len) {
    return kRecordHeaderSize + key_len + val_len + kRecordCrcSize;
}

inline void encode_record(void* dst,
                           const void* key, uint32_t key_len,
                           const void* val, uint32_t val_len) {
    auto* p = static_cast<uint8_t*>(dst);
    std::memcpy(p, &key_len, 4);
    std::memcpy(p + 4, &val_len, 4);
    std::memcpy(p + 8, key, key_len);
    std::memcpy(p + 8 + key_len, val, val_len);

    size_t payload_len = kRecordHeaderSize + key_len + val_len;
    uint32_t crc = static_cast<uint32_t>(
        absl::ComputeCrc32c(absl::string_view(reinterpret_cast<const char*>(p), payload_len)));
    p[payload_len + 0] = static_cast<uint8_t>(crc >> 0);
    p[payload_len + 1] = static_cast<uint8_t>(crc >> 8);
    p[payload_len + 2] = static_cast<uint8_t>(crc >> 16);
    p[payload_len + 3] = static_cast<uint8_t>(crc >> 24);
}

inline bool verify_record_crc(const void* src, size_t len) {
    if (len < kRecordHeaderSize + kRecordCrcSize) return false;
    const auto* p = static_cast<const uint8_t*>(src);
    uint32_t key_len, val_len;
    std::memcpy(&key_len, p, 4);
    std::memcpy(&val_len, p + 4, 4);
    if (len < kRecordHeaderSize + key_len + val_len + kRecordCrcSize)
        return false;
    size_t payload_len = len - kRecordCrcSize;
    uint32_t computed = static_cast<uint32_t>(
        absl::ComputeCrc32c(absl::string_view(reinterpret_cast<const char*>(src), payload_len)));
    uint32_t stored = static_cast<uint32_t>(p[payload_len + 0]) << 0
                    | static_cast<uint32_t>(p[payload_len + 1]) << 8
                    | static_cast<uint32_t>(p[payload_len + 2]) << 16
                    | static_cast<uint32_t>(p[payload_len + 3]) << 24;
    return computed == stored;
}

struct DecodedRecord {
    const uint8_t* key;
    uint32_t       key_len;
    const uint8_t* val;
    uint32_t       val_len;
};

inline DecodedRecord decode_record(const void* src) {
    auto* p = static_cast<const uint8_t*>(src);
    DecodedRecord r{};
    std::memcpy(&r.key_len, p, 4);
    std::memcpy(&r.val_len, p + 4, 4);
    r.key = p + 8;
    r.val = p + 8 + r.key_len;
    return r;
}
