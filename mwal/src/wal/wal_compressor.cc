// WAL record-level compression/decompression for mwal.

#include "wal/wal_compressor.h"

#include <cstring>

#ifdef MWAL_HAVE_ZSTD
#include <zstd.h>
#endif

namespace mwal {

static constexpr uint8_t kCompressionNone = 0x00;
static constexpr uint8_t kCompressionZSTD = 0x07;

Status WalCompressor::Compress(CompressionType type, const Slice& input,
                               std::string* output) {
  if (type == kNoCompression) {
    output->clear();
    output->reserve(1 + input.size());
    output->push_back(static_cast<char>(kCompressionNone));
    output->append(input.data(), input.size());
    return Status::OK();
  }

  if (type != kZSTD) {
    return Status::NotSupported("Only zstd WAL compression is supported");
  }

#ifdef MWAL_HAVE_ZSTD
  size_t bound = ZSTD_compressBound(input.size());
  output->resize(1 + bound);
  (*output)[0] = static_cast<char>(kCompressionZSTD);

  size_t compressed_size =
      ZSTD_compress(&(*output)[1], bound, input.data(), input.size(), 1);
  if (ZSTD_isError(compressed_size)) {
    return Status::Corruption("ZSTD compression failed",
                              ZSTD_getErrorName(compressed_size));
  }
  output->resize(1 + compressed_size);
  return Status::OK();
#else
  (void)input;
  (void)output;
  return Status::NotSupported("mwal built without zstd support");
#endif
}

bool WalCompressor::IsCompressed(const Slice& data) {
  if (data.empty()) return false;
  uint8_t tag = static_cast<uint8_t>(data[0]);
  return tag == kCompressionZSTD;
}

Status WalDecompressor::Decompress(const Slice& input, std::string* output,
                                    Slice* result) {
  if (input.empty()) {
    output->clear();
    *result = Slice(*output);
    return Status::OK();
  }

  uint8_t tag = static_cast<uint8_t>(input[0]);

  if (tag == kCompressionNone) {
    // Explicit uncompressed prefix: strip the tag byte.
    *result = Slice(input.data() + 1, input.size() - 1);
    return Status::OK();
  }

  if (tag == kCompressionZSTD) {
#ifdef MWAL_HAVE_ZSTD
    const char* compressed = input.data() + 1;
    size_t compressed_len = input.size() - 1;

    unsigned long long decompressed_size =
        ZSTD_getFrameContentSize(compressed, compressed_len);
    if (decompressed_size == ZSTD_CONTENTSIZE_UNKNOWN ||
        decompressed_size == ZSTD_CONTENTSIZE_ERROR) {
      return Status::Corruption("Cannot determine decompressed size");
    }

    output->resize(static_cast<size_t>(decompressed_size));
    size_t actual = ZSTD_decompress(output->data(), output->size(), compressed,
                                    compressed_len);
    if (ZSTD_isError(actual)) {
      return Status::Corruption("ZSTD decompression failed",
                                ZSTD_getErrorName(actual));
    }
    output->resize(actual);
    *result = Slice(*output);
    return Status::OK();
#else
    (void)output;
    (void)result;
    return Status::NotSupported("mwal built without zstd support");
#endif
  }

  // Legacy record: no compression prefix. Treat entire input as raw data.
  *result = input;
  return Status::OK();
}

}  // namespace mwal
