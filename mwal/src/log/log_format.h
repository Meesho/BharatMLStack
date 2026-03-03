// Copyright (c) 2011-present, Facebook, Inc.  All rights reserved.
// Derived from RocksDB — log record format definitions for mwal.

#pragma once

#include <cstdint>

namespace mwal {
namespace log {

enum RecordType : uint8_t {
  kZeroType = 0,
  kFullType = 1,
  kFirstType = 2,
  kMiddleType = 3,
  kLastType = 4,

  kRecyclableFullType = 5,
  kRecyclableFirstType = 6,
  kRecyclableMiddleType = 7,
  kRecyclableLastType = 8,
};

constexpr uint8_t kMaxRecordType = kRecyclableLastType;

constexpr unsigned int kBlockSize = 32768;

// Header: checksum (4B) + length (2B) + type (1B)
constexpr int kHeaderSize = 4 + 2 + 1;

// Recyclable header adds log number (4B)
constexpr int kRecyclableHeaderSize = 4 + 2 + 1 + 4;

}  // namespace log
}  // namespace mwal
