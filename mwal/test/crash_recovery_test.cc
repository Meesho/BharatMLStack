// Crash / truncation recovery tests: simulate crash by truncating WAL files
// at specific offsets, then run recovery under all four WALRecoveryMode values.

#include <gtest/gtest.h>

#include <fcntl.h>
#include <unistd.h>

#include <algorithm>
#include <string>
#include <utility>
#include <vector>

#include "file/writable_file_writer.h"
#include "log/log_format.h"
#include "log/log_writer.h"
#include "mwal/compression_type.h"
#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/write_batch.h"
#include "test_util.h"
#include "util/coding.h"
#include "wal/wal_compressor.h"

namespace mwal {

namespace {

constexpr int kHeaderSize = log::kHeaderSize;  // 7 bytes, non-recyclable

// Parse WAL file and return (start, end) byte offsets for each physical record.
// Uses 7-byte header only (mwal uses non-recyclable).
std::vector<std::pair<uint64_t, uint64_t>> GetWalRecordOffsets(Env* env,
                                                               const std::string& path) {
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
    size_t to_read = static_cast<size_t>(std::min<uint64_t>(kChunk, file_size - total_read));
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

// Truncate file at path to given size (test-only; POSIX).
bool TruncateWalFileAt(const std::string& path, uint64_t size) {
  int fd = open(path.c_str(), O_RDWR);
  if (fd < 0) return false;
  int ret = ftruncate(fd, static_cast<off_t>(size));
  close(fd);
  return ret == 0;
}

// Overwrite one byte at offset (test-only; POSIX). Used for checksum corruption.
bool OverwriteByteAt(const std::string& path, uint64_t offset, unsigned char byte) {
  int fd = open(path.c_str(), O_RDWR);
  if (fd < 0) return false;
  if (lseek(fd, static_cast<off_t>(offset), SEEK_SET) != static_cast<off_t>(offset)) {
    close(fd);
    return false;
  }
  unsigned char b = byte;
  ssize_t n = write(fd, &b, 1);
  close(fd);
  return n == 1;
}

// XOR one byte at offset with mask (test-only; POSIX). Used for bit-flip corruption.
bool XorByteAt(const std::string& path, uint64_t offset, unsigned char mask) {
  int fd = open(path.c_str(), O_RDWR);
  if (fd < 0) return false;
  if (lseek(fd, static_cast<off_t>(offset), SEEK_SET) != static_cast<off_t>(offset)) {
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
  if (lseek(fd, static_cast<off_t>(offset), SEEK_SET) != static_cast<off_t>(offset)) {
    close(fd);
    return false;
  }
  n = write(fd, &b, 1);
  close(fd);
  return n == 1;
}

bool WriteRawWalRecord(Env* env, const std::string& path, uint64_t log_num,
                       const std::string& record_data) {
  std::unique_ptr<WritableFile> file;
  Status s = env->NewWritableFile(path, &file, EnvOptions());
  if (!s.ok()) return false;
  auto writer_file = std::make_unique<WritableFileWriter>(std::move(file), path);
  log::Writer writer(std::move(writer_file), log_num, false);
  IOStatus ws = writer.AddRecord(Slice(record_data));
  if (!ws.ok()) return false;
  IOStatus cs = writer.Close();
  return cs.ok();
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

class CrashRecoveryTest : public ::testing::Test {
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

// --- Clean shutdown: normal Close(), full recovery, zero data loss ---
TEST_F(CrashRecoveryTest, CleanShutdown) {
  constexpr int N = 5;
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    for (int i = 0; i < N; i++) {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), "v" + std::to_string(i));
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    ASSERT_TRUE(wal->Close().ok());
  }

  for (WALRecoveryMode mode : {
         WALRecoveryMode::kTolerateCorruptedTailRecords,
         WALRecoveryMode::kAbsoluteConsistency,
         WALRecoveryMode::kPointInTimeRecovery,
         WALRecoveryMode::kSkipAnyCorruptedRecords,
       }) {
    int count = 0;
    Status s = RunRecoveryWithMode(tmpdir_.path(), mode, &count, env_);
    ASSERT_TRUE(s.ok()) << "mode=" << static_cast<int>(mode) << " " << s.ToString();
    EXPECT_EQ(count, N) << "mode=" << static_cast<int>(mode);
  }
}

// --- Truncate last N bytes (crash after write but before flush) ---
TEST_F(CrashRecoveryTest, TruncateLastBytes) {
  constexpr int N = 4;
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
  uint64_t last_start = offsets[N - 1].first;
  ASSERT_TRUE(TruncateWalFileAt(path, last_start + 5));

  int count_tolerate = 0, count_abs = 0, count_pitr = 0, count_skip = 0;
  Status s_tolerate = RunRecoveryWithMode(tmpdir_.path(),
                                          WALRecoveryMode::kTolerateCorruptedTailRecords,
                                          &count_tolerate, env_);
  Status s_abs = RunRecoveryWithMode(tmpdir_.path(),
                                    WALRecoveryMode::kAbsoluteConsistency,
                                    &count_abs, env_);
  Status s_pitr = RunRecoveryWithMode(tmpdir_.path(),
                                      WALRecoveryMode::kPointInTimeRecovery,
                                      &count_pitr, env_);
  Status s_skip = RunRecoveryWithMode(tmpdir_.path(),
                                      WALRecoveryMode::kSkipAnyCorruptedRecords,
                                      &count_skip, env_);

  ASSERT_TRUE(s_tolerate.ok());
  ASSERT_TRUE(s_pitr.ok());
  ASSERT_TRUE(s_skip.ok());
  EXPECT_TRUE(s_abs.IsCorruption()) << s_abs.ToString();

  EXPECT_EQ(count_tolerate, N - 1);
  EXPECT_EQ(count_pitr, N - 1);
  EXPECT_EQ(count_skip, N - 1);
}

// --- Truncate mid-header (inside 7-byte record header) ---
TEST_F(CrashRecoveryTest, TruncateMidHeader) {
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    for (int i = 0; i < 3; i++) {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), "v");
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    ASSERT_TRUE(wal->Close().ok());
  }

  std::string path = test::TempFileName(tmpdir_.path(), 1);
  auto offsets = GetWalRecordOffsets(env_, path);
  ASSERT_GE(offsets.size(), 2u);
  uint64_t trunc_at = offsets[1].first + 3;
  ASSERT_TRUE(TruncateWalFileAt(path, trunc_at));

  for (WALRecoveryMode mode : {
         WALRecoveryMode::kTolerateCorruptedTailRecords,
         WALRecoveryMode::kPointInTimeRecovery,
         WALRecoveryMode::kSkipAnyCorruptedRecords,
       }) {
    int count = 0;
    Status s = RunRecoveryWithMode(tmpdir_.path(), mode, &count, env_);
    ASSERT_TRUE(s.ok()) << s.ToString();
    EXPECT_EQ(count, 1);
  }
  int count_abs = 0;
  Status s_abs = RunRecoveryWithMode(tmpdir_.path(),
                                     WALRecoveryMode::kAbsoluteConsistency,
                                     &count_abs, env_);
  EXPECT_TRUE(s_abs.IsCorruption()) << s_abs.ToString();
}

// --- Truncate mid-payload (header complete, payload cut short) ---
TEST_F(CrashRecoveryTest, TruncateMidPayload) {
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    for (int i = 0; i < 3; i++) {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), "v");
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    ASSERT_TRUE(wal->Close().ok());
  }

  std::string path = test::TempFileName(tmpdir_.path(), 1);
  auto offsets = GetWalRecordOffsets(env_, path);
  ASSERT_GE(offsets.size(), 2u);
  uint64_t trunc_at = offsets[1].first + kHeaderSize + 5;
  ASSERT_TRUE(TruncateWalFileAt(path, trunc_at));

  for (WALRecoveryMode mode : {
         WALRecoveryMode::kTolerateCorruptedTailRecords,
         WALRecoveryMode::kPointInTimeRecovery,
         WALRecoveryMode::kSkipAnyCorruptedRecords,
       }) {
    int count = 0;
    Status s = RunRecoveryWithMode(tmpdir_.path(), mode, &count, env_);
    ASSERT_TRUE(s.ok());
    EXPECT_EQ(count, 1);
  }
  int count_abs = 0;
  Status s_abs = RunRecoveryWithMode(tmpdir_.path(),
                                     WALRecoveryMode::kAbsoluteConsistency,
                                     &count_abs, env_);
  EXPECT_TRUE(s_abs.IsCorruption()) << s_abs.ToString();
}

// --- Truncate between First and Last fragment (crash mid-multi-block record) ---
TEST_F(CrashRecoveryTest, TruncateBetweenFirstAndLast) {
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    WriteBatch small;
    small.Put("first", "record");
    ASSERT_TRUE(wal->Write(WriteOptions(), &small).ok());
    std::string big_val(log::kBlockSize - kHeaderSize + 1000, 'X');
    WriteBatch big_batch;
    big_batch.Put("big_key", big_val);
    ASSERT_TRUE(wal->Write(WriteOptions(), &big_batch).ok());
    ASSERT_TRUE(wal->Close().ok());
  }

  std::string path = test::TempFileName(tmpdir_.path(), 1);
  auto offsets = GetWalRecordOffsets(env_, path);
  ASSERT_GE(offsets.size(), 2u);
  uint64_t first_fragment_end = offsets[1].second;
  ASSERT_TRUE(TruncateWalFileAt(path, first_fragment_end + 100));

  for (WALRecoveryMode mode : {
         WALRecoveryMode::kTolerateCorruptedTailRecords,
         WALRecoveryMode::kPointInTimeRecovery,
         WALRecoveryMode::kSkipAnyCorruptedRecords,
       }) {
    int count = 0;
    Status s = RunRecoveryWithMode(tmpdir_.path(), mode, &count, env_);
    ASSERT_TRUE(s.ok());
    EXPECT_EQ(count, 1);
  }
  int count_abs = 0;
  Status s_abs = RunRecoveryWithMode(tmpdir_.path(),
                                     WALRecoveryMode::kAbsoluteConsistency,
                                     &count_abs, env_);
  EXPECT_TRUE(s_abs.IsCorruption()) << s_abs.ToString();
}

// --- Truncate exactly at record boundary (clean cut — recover all) ---
TEST_F(CrashRecoveryTest, TruncateAtRecordBoundary) {
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    WriteBatch b1, b2;
    b1.Put("a", "1");
    b2.Put("b", "2");
    ASSERT_TRUE(wal->Write(WriteOptions(), &b1).ok());
    ASSERT_TRUE(wal->Write(WriteOptions(), &b2).ok());
    ASSERT_TRUE(wal->Close().ok());
  }

  std::string path = test::TempFileName(tmpdir_.path(), 1);
  auto offsets = GetWalRecordOffsets(env_, path);
  ASSERT_GE(offsets.size(), 2u);
  uint64_t after_second = offsets[1].second;
  ASSERT_TRUE(TruncateWalFileAt(path, after_second));

  for (WALRecoveryMode mode : {
         WALRecoveryMode::kTolerateCorruptedTailRecords,
         WALRecoveryMode::kAbsoluteConsistency,
         WALRecoveryMode::kPointInTimeRecovery,
         WALRecoveryMode::kSkipAnyCorruptedRecords,
       }) {
    int count = 0;
    Status s = RunRecoveryWithMode(tmpdir_.path(), mode, &count, env_);
    ASSERT_TRUE(s.ok()) << "mode=" << static_cast<int>(mode) << " " << s.ToString();
    EXPECT_EQ(count, 2) << "mode=" << static_cast<int>(mode);
  }
}

// --- Truncate file to zero bytes ---
TEST_F(CrashRecoveryTest, TruncateZeroBytes) {
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    WriteBatch batch;
    batch.Put("x", "y");
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    ASSERT_TRUE(wal->Close().ok());
  }

  std::string path = test::TempFileName(tmpdir_.path(), 1);
  ASSERT_TRUE(TruncateWalFileAt(path, 0));

  for (WALRecoveryMode mode : {
         WALRecoveryMode::kTolerateCorruptedTailRecords,
         WALRecoveryMode::kAbsoluteConsistency,
         WALRecoveryMode::kPointInTimeRecovery,
         WALRecoveryMode::kSkipAnyCorruptedRecords,
       }) {
    int count = 0;
    Status s = RunRecoveryWithMode(tmpdir_.path(), mode, &count, env_);
    ASSERT_TRUE(s.ok()) << s.ToString();
    EXPECT_EQ(count, 0);
  }
}

// --- Mid-rotation: (a) old file full, new file zero bytes ---
TEST_F(CrashRecoveryTest, MidRotationNewFileZeroBytes) {
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    for (int i = 0; i < 3; i++) {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), "v");
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    ASSERT_TRUE(wal->Close().ok());
  }
  std::string path2 = test::TempFileName(tmpdir_.path(), 2);
  {
    std::unique_ptr<WritableFile> wf;
    ASSERT_TRUE(env_->NewWritableFile(path2, &wf, EnvOptions()).ok());
    ASSERT_TRUE(wf->Close().ok());
  }

  for (WALRecoveryMode mode : {
         WALRecoveryMode::kTolerateCorruptedTailRecords,
         WALRecoveryMode::kAbsoluteConsistency,
         WALRecoveryMode::kPointInTimeRecovery,
         WALRecoveryMode::kSkipAnyCorruptedRecords,
       }) {
    int count = 0;
    Status s = RunRecoveryWithMode(tmpdir_.path(), mode, &count, env_);
    ASSERT_TRUE(s.ok()) << s.ToString();
    EXPECT_EQ(count, 3);
  }
}

// --- Mid-rotation: (b) new file has partial first record ---
TEST_F(CrashRecoveryTest, MidRotationNewFilePartialFirstRecord) {
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    for (int i = 0; i < 2; i++) {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), "v");
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    ASSERT_TRUE(wal->Close().ok());
  }
  // Create 000002 with valid data, then truncate to 5 bytes (partial header).
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    WriteBatch batch;
    batch.Put("x", "y");
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    ASSERT_TRUE(wal->Close().ok());
  }
  ASSERT_TRUE(TruncateWalFileAt(test::TempFileName(tmpdir_.path(), 2), 5));

  // Run AbsoluteConsistency first (before any successful Open adds 000003.log).
  // For a partial first record in file 2, AbsoluteConsistency must fail.
  int count_abs = 0;
  Status s_abs = RunRecoveryWithMode(tmpdir_.path(),
                                     WALRecoveryMode::kAbsoluteConsistency,
                                     &count_abs, env_);
  EXPECT_TRUE(s_abs.IsCorruption()) << s_abs.ToString();

  for (WALRecoveryMode mode : {
         WALRecoveryMode::kTolerateCorruptedTailRecords,
         WALRecoveryMode::kPointInTimeRecovery,
         WALRecoveryMode::kSkipAnyCorruptedRecords,
       }) {
    int count = 0;
    Status s = RunRecoveryWithMode(tmpdir_.path(), mode, &count, env_);
    ASSERT_TRUE(s.ok());
    EXPECT_EQ(count, 2);
  }
}

// --- WALRecoveryMode matrix: same corrupt file, four modes, four outcomes ---
TEST_F(CrashRecoveryTest, RecoveryModeMatrix_FourModesTailCorruption) {
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
  uint64_t trunc_at = offsets[N - 1].first + 4;
  ASSERT_TRUE(TruncateWalFileAt(path, trunc_at));

  int count = 0;
  EXPECT_TRUE(RunRecoveryWithMode(tmpdir_.path(),
                                  WALRecoveryMode::kTolerateCorruptedTailRecords,
                                  &count, env_).ok());
  EXPECT_EQ(count, N - 1);

  EXPECT_TRUE(RunRecoveryWithMode(tmpdir_.path(),
                                  WALRecoveryMode::kPointInTimeRecovery,
                                  &count, env_).ok());
  EXPECT_EQ(count, N - 1);

  EXPECT_TRUE(RunRecoveryWithMode(tmpdir_.path(),
                                  WALRecoveryMode::kSkipAnyCorruptedRecords,
                                  &count, env_).ok());
  EXPECT_EQ(count, N - 1);

  Status s_abs = RunRecoveryWithMode(tmpdir_.path(),
                                     WALRecoveryMode::kAbsoluteConsistency,
                                     &count, env_);
  EXPECT_TRUE(s_abs.IsCorruption()) << s_abs.ToString();
}

// --- Corrupt mid-file (checksum) under TolerateCorruptedTailRecords ---
// Reader reports checksum error, skips one block, and continues.
TEST_F(CrashRecoveryTest, RecoveryModeMatrix_ChecksumCorruptMidFile) {
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    for (int i = 0; i < 3; i++) {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), "v");
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    ASSERT_TRUE(wal->Close().ok());
  }
  std::string path = test::TempFileName(tmpdir_.path(), 1);
  auto offsets = GetWalRecordOffsets(env_, path);
  ASSERT_GE(offsets.size(), 2u);
  uint64_t corrupt_offset = offsets[1].first;
  ASSERT_TRUE(OverwriteByteAt(path, corrupt_offset, 0xff));

  int count = 0;
  Status s = RunRecoveryWithMode(tmpdir_.path(),
                                 WALRecoveryMode::kTolerateCorruptedTailRecords,
                                 &count, env_);
  ASSERT_TRUE(s.ok());
  EXPECT_EQ(count, 1);
}

// --- Multi-file WAL: corruption in middle file; AbsoluteConsistency fails, others skip and continue ---
TEST_F(CrashRecoveryTest, RecoveryModeMatrix_MultiFileCorruptionInMiddle) {
  // Build deterministic layout:
  //   000001.log -> 2 valid records
  //   000002.log -> partial/corrupt (5 bytes)
  //   000003.log -> 2 valid records
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    for (int i = 0; i < 2; i++) {
      WriteBatch b;
      b.Put("f1_k" + std::to_string(i), "v");
      ASSERT_TRUE(wal->Write(WriteOptions(), &b).ok());
    }
    ASSERT_TRUE(wal->Close().ok());
  }

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    WriteBatch b;
    b.Put("f2", "v");
    ASSERT_TRUE(wal->Write(WriteOptions(), &b).ok());
    ASSERT_TRUE(wal->Close().ok());
  }
  ASSERT_TRUE(TruncateWalFileAt(test::TempFileName(tmpdir_.path(), 2), 5));

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    for (int i = 0; i < 2; i++) {
      WriteBatch b;
      b.Put("f3_k" + std::to_string(i), "v");
      ASSERT_TRUE(wal->Write(WriteOptions(), &b).ok());
    }
    ASSERT_TRUE(wal->Close().ok());
  }

  int count = 0;
  Status s_abs = RunRecoveryWithMode(tmpdir_.path(),
                                     WALRecoveryMode::kAbsoluteConsistency,
                                     &count, env_);
  EXPECT_TRUE(s_abs.IsCorruption()) << s_abs.ToString();

  for (WALRecoveryMode mode : {
         WALRecoveryMode::kTolerateCorruptedTailRecords,
         WALRecoveryMode::kPointInTimeRecovery,
         WALRecoveryMode::kSkipAnyCorruptedRecords,
       }) {
    int c = 0;
    Status s = RunRecoveryWithMode(tmpdir_.path(), mode, &c, env_);
    ASSERT_TRUE(s.ok()) << s.ToString();
    EXPECT_EQ(c, 4);
  }
}

TEST_F(CrashRecoveryTest, RecoverPropagatesCallbackCorruptionAndStops) {
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    for (int i = 0; i < 3; i++) {
      WriteBatch b;
      b.Put("k" + std::to_string(i), "v");
      ASSERT_TRUE(wal->Write(WriteOptions(), &b).ok());
    }
    ASSERT_TRUE(wal->Close().ok());
  }

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());

  int callbacks = 0;
  Status s = wal->Recover([&](SequenceNumber, WriteBatch*) {
    callbacks++;
    return Status::Corruption("callback corruption");
  });
  EXPECT_TRUE(s.IsCorruption()) << s.ToString();
  EXPECT_EQ(callbacks, 1);
  wal->Close();
}

TEST_F(CrashRecoveryTest, RecoverFailsOnMalformedBatchIterate) {
  // Build malformed WriteBatch payload:
  // [12-byte batch header with count=1] + [unknown tag byte 0x7f].
  std::string malformed_batch(12, '\0');
  EncodeFixed32(&malformed_batch[8], 1);
  malformed_batch.push_back(static_cast<char>(0x7f));

  // Wrap with explicit no-compression prefix so decompressor returns payload.
  std::string record;
  record.push_back(static_cast<char>(kNoCompression));
  record.append(malformed_batch);

  const std::string path = test::TempFileName(tmpdir_.path(), 1);
  ASSERT_TRUE(WriteRawWalRecord(env_, path, 1, record));

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());

  class NoopHandler : public WriteBatch::Handler {
   public:
    Status Put(const Slice&, const Slice&) override { return Status::OK(); }
    Status Delete(const Slice&) override { return Status::OK(); }
  };
  NoopHandler handler;
  int callbacks = 0;
  Status s = wal->Recover([&](SequenceNumber, WriteBatch* batch) {
    callbacks++;
    return batch->Iterate(&handler);
  });

  EXPECT_TRUE(s.IsCorruption()) << s.ToString();
  EXPECT_EQ(callbacks, 1);
  wal->Close();
}

TEST_F(CrashRecoveryTest, RecoverStopsOnDecompressionFailureAllModes) {
  // Physical WAL record with zstd tag + invalid frame.
  std::string bad_record;
  bad_record.push_back(static_cast<char>(kZSTD));
  bad_record.append("not_a_valid_zstd_frame");

  const std::string path = test::TempFileName(tmpdir_.path(), 1);
  ASSERT_TRUE(WriteRawWalRecord(env_, path, 1, bad_record));

  for (WALRecoveryMode mode : {
         WALRecoveryMode::kTolerateCorruptedTailRecords,
         WALRecoveryMode::kAbsoluteConsistency,
         WALRecoveryMode::kPointInTimeRecovery,
         WALRecoveryMode::kSkipAnyCorruptedRecords,
       }) {
    WALOptions opts = MakeOptions();
    opts.wal_recovery_mode = mode;
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    int callbacks = 0;
    Status s = wal->Recover([&](SequenceNumber, WriteBatch*) {
      callbacks++;
      return Status::OK();
    });
    EXPECT_TRUE(s.IsCorruption() || s.IsNotSupported()) << s.ToString();
    EXPECT_EQ(callbacks, 0);
    wal->Close();
  }
}

TEST_F(CrashRecoveryTest, RecoverAcrossThreeFilesMiddleCorrupt) {
  // 000001.log valid, 000002.log corrupt header, 000003.log valid.
  WriteBatch b1, b3;
  b1.Put("f1", "v1");
  b3.Put("f3", "v3");

  std::string rec1;
  ASSERT_TRUE(
      WalCompressor::Compress(kNoCompression, WriteBatchInternal::Contents(&b1), &rec1).ok());
  std::string rec3;
  ASSERT_TRUE(
      WalCompressor::Compress(kNoCompression, WriteBatchInternal::Contents(&b3), &rec3).ok());

  ASSERT_TRUE(WriteRawWalRecord(env_, test::TempFileName(tmpdir_.path(), 1), 1, rec1));
  ASSERT_TRUE(WriteRawWalRecord(env_, test::TempFileName(tmpdir_.path(), 2), 2, rec3));
  ASSERT_TRUE(WriteRawWalRecord(env_, test::TempFileName(tmpdir_.path(), 3), 3, rec3));

  // Truncate 000002 to 5 bytes (partial header) to simulate corruption.
  ASSERT_TRUE(TruncateWalFileAt(test::TempFileName(tmpdir_.path(), 2), 5));

  // AbsoluteConsistency: fail on middle corrupt file.
  {
    WALOptions opts = MakeOptions();
    opts.wal_recovery_mode = WALRecoveryMode::kAbsoluteConsistency;
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
    int callbacks = 0;
    Status s = wal->Recover([&](SequenceNumber, WriteBatch*) {
      callbacks++;
      return Status::OK();
    });
    EXPECT_TRUE(s.IsCorruption()) << s.ToString();
    EXPECT_EQ(callbacks, 1);  // 000001 before corrupt 000002
    wal->Close();
  }

  // PointInTime: skip middle file, continue to file 3.
  {
    WALOptions opts = MakeOptions();
    opts.wal_recovery_mode = WALRecoveryMode::kPointInTimeRecovery;
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
    int callbacks = 0;
    Status s = wal->Recover([&](SequenceNumber, WriteBatch*) {
      callbacks++;
      return Status::OK();
    });
    EXPECT_TRUE(s.ok()) << s.ToString();
    EXPECT_EQ(callbacks, 2);
    wal->Close();
  }
}

TEST_F(CrashRecoveryTest, RecoverHandlesDiscoveredEmptyLogFile) {
  WriteBatch b;
  b.Put("k", "v");
  std::string rec;
  ASSERT_TRUE(
      WalCompressor::Compress(kNoCompression, WriteBatchInternal::Contents(&b), &rec).ok());
  ASSERT_TRUE(WriteRawWalRecord(env_, test::TempFileName(tmpdir_.path(), 1), 1, rec));

  // Discovered empty log file (000002.log) should be harmless.
  {
    std::unique_ptr<WritableFile> wf;
    ASSERT_TRUE(
        env_->NewWritableFile(test::TempFileName(tmpdir_.path(), 2), &wf, EnvOptions()).ok());
    ASSERT_TRUE(wf->Close().ok());
  }

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
  int callbacks = 0;
  Status s = wal->Recover([&](SequenceNumber, WriteBatch*) {
    callbacks++;
    return Status::OK();
  });
  EXPECT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(callbacks, 1);
  wal->Close();
}

// --- Directory lock: second Open() on same dir must fail (flock) ---
TEST_F(CrashRecoveryTest, DirectoryLock) {
  auto opts = MakeOptions();
  std::unique_ptr<DBWal> wal1;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal1).ok());

  std::unique_ptr<DBWal> wal2;
  Status s = DBWal::Open(opts, env_, &wal2);
  EXPECT_TRUE(s.IsIOError()) << s.ToString();

  wal1->Close();
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal2).ok());
  wal2->Close();
}

}  // namespace mwal
