// Section 7: Bit-flip / data corruption tests (TC-CORRUPT-01 through TC-CORRUPT-06).
// Simulates corruption by XOR/overwrite of specific bytes, then runs recovery
// under WALRecoveryMode and asserts expected behavior per WAL_DESIGN.md Section 11.

#include <gtest/gtest.h>

#include <fcntl.h>
#include <unistd.h>

#include <algorithm>
#include <string>
#include <utility>
#include <vector>

#include "log/log_format.h"
#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/write_batch.h"
#include "test_util.h"
#include "util/coding.h"

namespace mwal {

namespace {

constexpr int kHeaderSize = log::kHeaderSize;  // 7 bytes, non-recyclable

// Parse WAL file and return (start, end) byte offsets for each physical record.
std::vector<std::pair<uint64_t, uint64_t>> GetWalRecordOffsets(
    Env* env, const std::string& path) {
  std::vector<std::pair<uint64_t, uint64_t>> offsets;
  uint64_t file_size = 0;
  Status s = env->GetFileSize(path, &file_size);
  if (!s.ok() || file_size == 0) return offsets;

  std::unique_ptr<SequentialFile> seq_file;
  s = env->NewSequentialFile(path, &seq_file, EnvOptions());
  if (!s.ok()) return offsets;

  std::string data;
  data.reserve(static_cast<size_t>(file_size));
  constexpr size_t kChunk = 65536;
  std::vector<char> scratch(kChunk);
  uint64_t total_read = 0;
  while (total_read < file_size) {
    size_t to_read = static_cast<size_t>(
        std::min<uint64_t>(kChunk, file_size - total_read));
    Slice result;
    IOStatus ios = seq_file->Read(to_read, &result, scratch.data());
    if (!ios.ok()) return offsets;
    data.append(result.data(), result.size());
    total_read += result.size();
    if (result.size() < to_read) break;
  }

  const char* buf = data.data();
  const size_t size = data.size();
  uint64_t pos = 0;

  while (pos + kHeaderSize <= size) {
    uint32_t length = DecodeFixed16(buf + pos + 4);
    uint64_t record_end = pos + kHeaderSize + length;
    if (record_end > size) break;
    offsets.push_back({pos, record_end});
    pos = record_end;
  }
  return offsets;
}

// Overwrite one byte at offset (test-only; POSIX).
bool OverwriteByteAt(const std::string& path, uint64_t offset,
                     unsigned char byte) {
  int fd = open(path.c_str(), O_RDWR);
  if (fd < 0) return false;
  if (lseek(fd, static_cast<off_t>(offset), SEEK_SET) !=
      static_cast<off_t>(offset)) {
    close(fd);
    return false;
  }
  unsigned char b = byte;
  ssize_t n = write(fd, &b, 1);
  close(fd);
  return n == 1;
}

// XOR one byte at offset with mask (test-only; POSIX). Used for bit-flip.
bool XorByteAt(const std::string& path, uint64_t offset, unsigned char mask) {
  int fd = open(path.c_str(), O_RDWR);
  if (fd < 0) return false;
  if (lseek(fd, static_cast<off_t>(offset), SEEK_SET) !=
      static_cast<off_t>(offset)) {
    close(fd);
    return false;
  }
  unsigned char b = 0;
  ssize_t n = read(fd, &b, 1);
  if (n != 1) {
    close(fd);
    return false;
  }
  b ^= mask;
  if (lseek(fd, static_cast<off_t>(offset), SEEK_SET) !=
      static_cast<off_t>(offset)) {
    close(fd);
    return false;
  }
  n = write(fd, &b, 1);
  close(fd);
  return n == 1;
}

// Run recovery with given mode; return Open status and set recovered_count.
Status RunRecoveryWithMode(const std::string& wal_dir, WALRecoveryMode mode,
                           int* recovered_count, Env* env) {
  *recovered_count = 0;
  WALOptions opts;
  opts.wal_dir = wal_dir;
  opts.wal_recovery_mode = mode;
  opts.recovery_callback = [recovered_count](SequenceNumber, WriteBatch*) {
    (*recovered_count)++;
    return Status::OK();
  };
  std::unique_ptr<DBWal> wal;
  Status s = DBWal::Open(opts, env, &wal);
  if (s.ok()) {
    wal->Close();
  }
  return s;
}

}  // namespace

class BitFlipCorruptionTest : public ::testing::Test {
 protected:
  void SetUp() override { env_ = Env::Default(); }

  WALOptions MakeOptions() {
    WALOptions opts;
    opts.wal_dir = tmpdir_.path();
    return opts;
  }

  test::TempDir tmpdir_;
  Env* env_ = nullptr;
};

// TC-CORRUPT-01: Bit flip in CRC field of record 2.
// Run recovery under each of the four WALRecoveryMode values.
// Use record 2 large enough to push record 3 to the next block, so kSkipAnyCorruptedRecords
// can skip only record 2 and recover record 3 (reader drops block on checksum error).
TEST_F(BitFlipCorruptionTest, TC_CORRUPT_01_BitFlipInCrcField) {
  constexpr size_t kBlockSize = log::kBlockSize;
  // Record 2 must span past block boundary so record 3 is in block 2.
  // Record 1 ~50 bytes; need record 2 > (kBlockSize - 50 - 7) to overflow.
  constexpr size_t kRecord2PayloadSize = kBlockSize;
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    WriteBatch batch1;
    batch1.Put("k0", "v");
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch1).ok());
    WriteBatch batch2;
    batch2.Put("k1", std::string(kRecord2PayloadSize, 'x'));
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch2).ok());
    WriteBatch batch3;
    batch3.Put("k2", "v");
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch3).ok());
    ASSERT_TRUE(wal->Close().ok());
  }

  std::string path = test::TempFileName(tmpdir_.path(), 1);
  auto offsets = GetWalRecordOffsets(env_, path);
  ASSERT_GE(offsets.size(), 3u);
  uint64_t crc_offset = offsets[1].first;  // CRC is bytes 0-3 of record 2
  ASSERT_TRUE(XorByteAt(path, crc_offset, 0x01));

  int count = 0;

  // kTolerateCorruptedTailRecords: Records 2 and 3 may be lost; record 1 recovered. No error.
  Status s_tolerate =
      RunRecoveryWithMode(tmpdir_.path(),
                         WALRecoveryMode::kTolerateCorruptedTailRecords, &count,
                         env_);
  ASSERT_TRUE(s_tolerate.ok()) << s_tolerate.ToString();
  EXPECT_EQ(count, 1);

  // kAbsoluteConsistency: Record 2 corrupt. Recover() returns Status::Corruption.
  count = 0;
  Status s_abs = RunRecoveryWithMode(
      tmpdir_.path(), WALRecoveryMode::kAbsoluteConsistency, &count, env_);
  EXPECT_TRUE(s_abs.IsCorruption()) << s_abs.ToString();

  // kPointInTimeRecovery: Corruption reported; recovery stops at record 2. Record 1 returned. OK.
  count = 0;
  Status s_pitr = RunRecoveryWithMode(
      tmpdir_.path(), WALRecoveryMode::kPointInTimeRecovery, &count, env_);
  ASSERT_TRUE(s_pitr.ok()) << s_pitr.ToString();
  EXPECT_EQ(count, 1);

  // kSkipAnyCorruptedRecords: Record 2 skipped. Records 1 and 3 recovered. OK.
  // Record 3 is in a different block, so block drop on record 2 does not lose it.
  count = 0;
  Status s_skip = RunRecoveryWithMode(
      tmpdir_.path(), WALRecoveryMode::kSkipAnyCorruptedRecords, &count, env_);
  ASSERT_TRUE(s_skip.ok()) << s_skip.ToString();
  EXPECT_EQ(count, 2);  // Records 1 and 3
}

// TC-CORRUPT-02: Bit flip in length field of record 2.
// kPointInTimeRecovery: record 1 recovered, 2 and 3 not; Recover() OK.
TEST_F(BitFlipCorruptionTest, TC_CORRUPT_02_BitFlipInLengthField) {
  constexpr int N = 3;
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    for (int i = 0; i < N; i++) {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), "v");
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    ASSERT_TRUE(wal->Close().ok());
  }

  std::string path = test::TempFileName(tmpdir_.path(), 1);
  auto offsets = GetWalRecordOffsets(env_, path);
  ASSERT_GE(offsets.size(), static_cast<size_t>(N));
  uint64_t len_offset = offsets[1].first + 4;  // Length is bytes 4-5
  ASSERT_TRUE(XorByteAt(path, len_offset, 0x80));

  int count = 0;
  Status s = RunRecoveryWithMode(tmpdir_.path(),
                                WALRecoveryMode::kPointInTimeRecovery, &count,
                                env_);
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(count, 1);  // Only record 1 recovered
}

// TC-CORRUPT-03: Bit flip in payload of record 2.
// kPointInTimeRecovery: CRC fails on record 2; record 1 recovered.
TEST_F(BitFlipCorruptionTest, TC_CORRUPT_03_BitFlipInPayload) {
  constexpr int N = 3;
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    for (int i = 0; i < N; i++) {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), "v");
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    ASSERT_TRUE(wal->Close().ok());
  }

  std::string path = test::TempFileName(tmpdir_.path(), 1);
  auto offsets = GetWalRecordOffsets(env_, path);
  ASSERT_GE(offsets.size(), static_cast<size_t>(N));
  // Payload starts after 7-byte header
  uint64_t payload_offset = offsets[1].first + 7;
  uint64_t payload_end = offsets[1].second;
  ASSERT_LT(payload_offset, payload_end);
  uint64_t mid_offset = payload_offset + (payload_end - payload_offset) / 2;
  ASSERT_TRUE(XorByteAt(path, mid_offset, 0x01));

  int count = 0;
  Status s = RunRecoveryWithMode(tmpdir_.path(),
                                WALRecoveryMode::kPointInTimeRecovery, &count,
                                env_);
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(count, 1);  // Record 1 recovered; record 2 CRC fails
}

// TC-CORRUPT-04: Middle fragment type byte corrupted to kFullType.
// Write 100 KiB record (spans 3 blocks), corrupt Middle fragment type.
// kPointInTimeRecovery: fragment sequence error; recovery stops.
TEST_F(BitFlipCorruptionTest, TC_CORRUPT_04_MiddleFragmentTypeToFullType) {
  constexpr size_t kPayloadSize = 100 * 1024;  // 100 KiB
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    WriteBatch batch;
    std::string val(kPayloadSize, 'x');
    batch.Put("large_key", val);
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    ASSERT_TRUE(wal->Close().ok());
  }

  std::string path = test::TempFileName(tmpdir_.path(), 1);
  auto offsets = GetWalRecordOffsets(env_, path);
  // 100 KiB spans 3 blocks: First, Middle, Last fragments
  ASSERT_GE(offsets.size(), 3u);
  // Second physical record is the Middle fragment
  uint64_t type_offset = offsets[1].first + 6;  // Type is byte 6 of header
  ASSERT_TRUE(OverwriteByteAt(path, type_offset, log::kFullType));

  int count = 0;
  Status s = RunRecoveryWithMode(tmpdir_.path(),
                                WALRecoveryMode::kPointInTimeRecovery, &count,
                                env_);
  ASSERT_TRUE(s.ok()) << s.ToString();
  // Fragment sequence error; recovery stops. Record 1 (First fragment) may or may not
  // be returned as complete; typically we get 0 or partial. Spec: "recovery stops at this point".
  EXPECT_EQ(count, 0);
}

// TC-CORRUPT-05: Unknown compression prefix byte (0x00 -> 0x05).
// Run under all modes; document actual behavior.
TEST_F(BitFlipCorruptionTest, TC_CORRUPT_05_UnknownCompressionPrefix) {
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    WriteBatch batch;
    batch.Put("k", "v");
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    ASSERT_TRUE(wal->Close().ok());
  }

  std::string path = test::TempFileName(tmpdir_.path(), 1);
  auto offsets = GetWalRecordOffsets(env_, path);
  ASSERT_GE(offsets.size(), 1u);
  // First payload byte (after 7-byte log header) is compression prefix
  uint64_t prefix_offset = offsets[0].first + 7;
  ASSERT_TRUE(OverwriteByteAt(path, prefix_offset, 0x05));

  for (WALRecoveryMode mode :
       {WALRecoveryMode::kTolerateCorruptedTailRecords,
        WALRecoveryMode::kAbsoluteConsistency,
        WALRecoveryMode::kPointInTimeRecovery,
        WALRecoveryMode::kSkipAnyCorruptedRecords}) {
    int count = 0;
    Status s = RunRecoveryWithMode(tmpdir_.path(), mode, &count, env_);
    // Spec: characterize actual behavior. Unknown prefix falls through to legacy
    // path; WriteBatch parsing may fail or produce garbage. Expect Corruption or OK with wrong count.
    (void)count;
    (void)s;
    // Document: if s.IsCorruption(), decompression/WriteBatch rejected it.
    // If s.ok() and count wrong, silent garbage (bug).
    EXPECT_TRUE(s.ok() || s.IsCorruption())
        << "mode=" << static_cast<int>(mode) << " " << s.ToString();
  }
}

// TC-CORRUPT-06: Valid -> corrupt -> valid under kSkipAnyCorruptedRecords.
// Corrupt record 2's CRC; records 1 and 3 recovered; callback exactly twice; Recover() OK.
// Use record 2 large enough to push record 3 to the next block (reader drops block on checksum).
TEST_F(BitFlipCorruptionTest, TC_CORRUPT_06_ValidCorruptValidSkipMode) {
  constexpr size_t kBlockSize = log::kBlockSize;
  constexpr size_t kRecord2PayloadSize = kBlockSize;
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    WriteBatch batch1;
    batch1.Put("k0", "v");
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch1).ok());
    WriteBatch batch2;
    batch2.Put("k1", std::string(kRecord2PayloadSize, 'x'));
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch2).ok());
    WriteBatch batch3;
    batch3.Put("k2", "v");
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch3).ok());
    ASSERT_TRUE(wal->Close().ok());
  }

  std::string path = test::TempFileName(tmpdir_.path(), 1);
  auto offsets = GetWalRecordOffsets(env_, path);
  ASSERT_GE(offsets.size(), 3u);
  uint64_t crc_offset = offsets[1].first;  // Corrupt CRC of record 2
  ASSERT_TRUE(XorByteAt(path, crc_offset, 0x01));

  int count = 0;
  Status s = RunRecoveryWithMode(tmpdir_.path(),
                                WALRecoveryMode::kSkipAnyCorruptedRecords,
                                &count, env_);
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(count, 2);  // Records 1 and 3; record 2 skipped
}

}  // namespace mwal
