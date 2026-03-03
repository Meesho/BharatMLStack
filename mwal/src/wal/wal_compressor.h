// WAL record-level compression/decompression for mwal.
// Uses a 1-byte prefix: 0x00 = uncompressed, 0x07 = zstd.
// Legacy records (no prefix) are detected by checking if the first bytes
// match the WriteBatch header pattern.

#pragma once

#include <string>

#include "mwal/compression_type.h"
#include "mwal/slice.h"
#include "mwal/status.h"

namespace mwal {

class WalCompressor {
 public:
  static Status Compress(CompressionType type, const Slice& input,
                         std::string* output);

  static bool IsCompressed(const Slice& data);
};

class WalDecompressor {
 public:
  // Decompresses if the record has a compression prefix.
  // For legacy uncompressed records, copies data as-is.
  // On success, *output contains the decompressed data and *result
  // points into *output.
  static Status Decompress(const Slice& input, std::string* output,
                            Slice* result);
};

}  // namespace mwal
