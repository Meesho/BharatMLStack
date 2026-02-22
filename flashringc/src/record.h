#pragma once

#include <cstdint>
#include <cstring>

// On-disk / in-memtable record layout:
//   [key_len: 4B] [val_len: 4B] [key_data: key_len B] [val_data: val_len B]

static constexpr size_t kRecordHeaderSize = 8;

inline size_t record_size(size_t key_len, size_t val_len) {
    return kRecordHeaderSize + key_len + val_len;
}

inline void encode_record(void* dst,
                           const void* key, uint32_t key_len,
                           const void* val, uint32_t val_len) {
    auto* p = static_cast<uint8_t*>(dst);
    std::memcpy(p, &key_len, 4);
    std::memcpy(p + 4, &val_len, 4);
    std::memcpy(p + 8, key, key_len);
    std::memcpy(p + 8 + key_len, val, val_len);
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
