// Copyright (c) 2011-present, Facebook, Inc.  All rights reserved.
// Derived from RocksDB — adapted for mwal standalone WAL library.

#pragma once

namespace mwal {

enum CompressionType : unsigned char {
  kNoCompression = 0x00,
  kSnappyCompression = 0x01,
  kZlibCompression = 0x02,
  kBZip2Compression = 0x03,
  kLZ4Compression = 0x04,
  kLZ4HCCompression = 0x05,
  kXpressCompression = 0x06,
  kZSTD = 0x07,
  kDisableCompressionOption = 0xff,
};

}  // namespace mwal
