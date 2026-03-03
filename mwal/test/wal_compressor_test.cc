#include <gtest/gtest.h>

#include <string>

#include "mwal/compression_type.h"
#include "mwal/slice.h"
#include "mwal/status.h"
#include "wal/wal_compressor.h"

namespace mwal {

TEST(WalCompressorTest, RoundtripUncompressed) {
  std::string input = "hello world, this is a WAL record";
  std::string compressed;
  Status s = WalCompressor::Compress(kNoCompression, Slice(input), &compressed);
  ASSERT_TRUE(s.ok()) << s.ToString();

  // Should have a 0x00 prefix + original data.
  EXPECT_EQ(compressed.size(), 1 + input.size());
  EXPECT_EQ(static_cast<uint8_t>(compressed[0]), 0x00);

  std::string decompressed;
  Slice result;
  s = WalDecompressor::Decompress(Slice(compressed), &decompressed, &result);
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(result.ToString(), input);
}

TEST(WalCompressorTest, RoundtripEmpty) {
  std::string compressed;
  Status s = WalCompressor::Compress(kNoCompression, Slice(), &compressed);
  ASSERT_TRUE(s.ok());
  EXPECT_EQ(compressed.size(), 1u);

  std::string decompressed;
  Slice result;
  s = WalDecompressor::Decompress(Slice(compressed), &decompressed, &result);
  ASSERT_TRUE(s.ok());
  EXPECT_TRUE(result.empty());
}

TEST(WalCompressorTest, DecompressEmpty) {
  std::string decompressed;
  Slice result;
  Status s = WalDecompressor::Decompress(Slice(), &decompressed, &result);
  ASSERT_TRUE(s.ok());
  EXPECT_TRUE(result.empty());
}

TEST(WalCompressorTest, LegacyUncompressedPassthrough) {
  // Simulate a legacy WriteBatch record: starts with sequence number bytes.
  // Sequence numbers won't start with 0x00 or 0x07 in practice for non-zero
  // sequences, so this tests the passthrough path.
  std::string legacy_data;
  legacy_data.resize(12, '\0');
  legacy_data[0] = 0x01;  // non-zero first byte = not a known compression tag
  legacy_data.append("some_payload");

  std::string decompressed;
  Slice result;
  Status s =
      WalDecompressor::Decompress(Slice(legacy_data), &decompressed, &result);
  ASSERT_TRUE(s.ok());
  EXPECT_EQ(result.ToString(), legacy_data);
}

#ifdef MWAL_HAVE_ZSTD

TEST(WalCompressorTest, ZstdRoundtripSmall) {
  std::string input = "small zstd test data";
  std::string compressed;
  Status s = WalCompressor::Compress(kZSTD, Slice(input), &compressed);
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(static_cast<uint8_t>(compressed[0]), 0x07);

  std::string decompressed;
  Slice result;
  s = WalDecompressor::Decompress(Slice(compressed), &decompressed, &result);
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(result.ToString(), input);
}

TEST(WalCompressorTest, ZstdRoundtripLarge) {
  std::string input(1024 * 1024, 'A');  // 1MB of repeated data
  for (size_t i = 0; i < input.size(); i++) {
    input[i] = static_cast<char>('A' + (i % 26));
  }

  std::string compressed;
  Status s = WalCompressor::Compress(kZSTD, Slice(input), &compressed);
  ASSERT_TRUE(s.ok()) << s.ToString();

  // Repeated pattern compresses well.
  EXPECT_LT(compressed.size(), input.size());

  std::string decompressed;
  Slice result;
  s = WalDecompressor::Decompress(Slice(compressed), &decompressed, &result);
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(result.ToString(), input);
}

TEST(WalCompressorTest, ZstdCompressionRatio) {
  std::string input(4096, 'X');
  std::string compressed;
  Status s = WalCompressor::Compress(kZSTD, Slice(input), &compressed);
  ASSERT_TRUE(s.ok());
  EXPECT_LT(compressed.size(), input.size() / 2);
}

TEST(WalCompressorTest, ZstdCorruptedInput) {
  std::string bad;
  bad.push_back(static_cast<char>(0x07));  // zstd tag
  bad.append("this is not valid zstd data");

  std::string decompressed;
  Slice result;
  Status s = WalDecompressor::Decompress(Slice(bad), &decompressed, &result);
  EXPECT_TRUE(s.IsCorruption() || s.IsNotSupported()) << s.ToString();
}

TEST(WalCompressorTest, IsCompressed) {
  std::string uncompressed;
  WalCompressor::Compress(kNoCompression, Slice("test"), &uncompressed);
  EXPECT_FALSE(WalCompressor::IsCompressed(Slice(uncompressed)));

  std::string compressed;
  WalCompressor::Compress(kZSTD, Slice("test"), &compressed);
  EXPECT_TRUE(WalCompressor::IsCompressed(Slice(compressed)));
}

#else

TEST(WalCompressorTest, ZstdNotAvailable) {
  std::string compressed;
  Status s = WalCompressor::Compress(kZSTD, Slice("test"), &compressed);
  EXPECT_TRUE(s.IsNotSupported()) << s.ToString();
}

#endif

TEST(WalCompressorTest, UnsupportedType) {
  std::string compressed;
  Status s =
      WalCompressor::Compress(kSnappyCompression, Slice("test"), &compressed);
  EXPECT_TRUE(s.IsNotSupported()) << s.ToString();
}

}  // namespace mwal
