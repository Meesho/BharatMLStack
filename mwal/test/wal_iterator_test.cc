// WalIterator tests: corruption behavior, snapshot semantics, start_seq beyond last.

#include <gtest/gtest.h>

#include <fcntl.h>
#include <unistd.h>

#include <string>
#include <vector>

#include "file/writable_file_writer.h"
#include "log/log_format.h"
#include "log/log_writer.h"
#include "mwal/compression_type.h"
#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/wal_iterator.h"
#include "mwal/write_batch.h"
#include "test_util.h"
#include "util/coding.h"

namespace mwal {

namespace {

constexpr int kHeaderSize = log::kHeaderSize;

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

bool TruncateWalFileAt(const std::string& path, uint64_t size) {
  int fd = open(path.c_str(), O_RDWR);
  if (fd < 0) return false;
  int ret = ftruncate(fd, static_cast<off_t>(size));
  close(fd);
  return ret == 0;
}

}  // namespace

class WalIteratorTest : public ::testing::Test {
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

// Iterator on corrupt file under kAbsoluteConsistency: Valid() becomes false, status() returns error.
TEST_F(WalIteratorTest, IteratorCorruptionAbsoluteConsistency) {
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
  ASSERT_TRUE(TruncateWalFileAt(path, offsets[1].first + 3));

  WALOptions opts = MakeOptions();
  opts.wal_recovery_mode = WALRecoveryMode::kAbsoluteConsistency;
  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  std::unique_ptr<WalIterator> iter;
  ASSERT_TRUE(wal->NewWalIterator(0, &iter).ok());
  ASSERT_TRUE(iter->Valid());
  iter->Next();
  EXPECT_FALSE(iter->Valid());
  EXPECT_TRUE(iter->status().IsCorruption()) << iter->status().ToString();
  wal->Close();
}

// Iterator on corrupt file under kPointInTimeRecovery: skips that file, continues (no more files).
TEST_F(WalIteratorTest, IteratorCorruptionPointInTimeSkipsFile) {
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
  std::string path2 = test::TempFileName(tmpdir_.path(), 2);
  {
    std::unique_ptr<WritableFile> wf;
    ASSERT_TRUE(env_->NewWritableFile(path2, &wf, EnvOptions()).ok());
    wf->Append(Slice("\x00\x00\x00\x00\x01"));
    ASSERT_TRUE(wf->Close().ok());
  }

  WALOptions opts = MakeOptions();
  opts.wal_recovery_mode = WALRecoveryMode::kPointInTimeRecovery;
  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  std::unique_ptr<WalIterator> iter;
  ASSERT_TRUE(wal->NewWalIterator(0, &iter).ok());
  int count = 0;
  while (iter->Valid()) {
    count++;
    iter->Next();
  }
  EXPECT_EQ(count, 2);
  EXPECT_TRUE(iter->status().ok()) << iter->status().ToString();
  wal->Close();
}

// Iterator start_seq beyond last record in all files: Valid() immediately false.
TEST_F(WalIteratorTest, IteratorStartSeqBeyondLast) {
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
    for (int i = 0; i < 10; i++) {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), "v");
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    ASSERT_TRUE(wal->Close().ok());
  }

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
  std::unique_ptr<WalIterator> iter;
  ASSERT_TRUE(wal->NewWalIterator(100, &iter).ok());
  EXPECT_FALSE(iter->Valid());
  EXPECT_TRUE(iter->status().ok()) << iter->status().ToString();
  wal->Close();
}

// Snapshot semantics: create iterator, write more records, iterator does NOT see them.
TEST_F(WalIteratorTest, IteratorSnapshotSemantics) {
  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());
  for (int i = 0; i < 5; i++) {
    WriteBatch batch;
    batch.Put("k" + std::to_string(i), "v");
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
  }

  std::unique_ptr<WalIterator> iter;
  ASSERT_TRUE(wal->NewWalIterator(0, &iter).ok());

  for (int i = 5; i < 10; i++) {
    WriteBatch batch;
    batch.Put("k" + std::to_string(i), "v");
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
  }

  int count = 0;
  while (iter->Valid()) {
    count++;
    iter->Next();
  }
  EXPECT_EQ(count, 5);
  EXPECT_TRUE(iter->status().ok());
  wal->Close();
}

#ifdef MWAL_HAVE_ZSTD
// Iterator where decompression fails: Valid() false, status() has Corruption.
TEST_F(WalIteratorTest, IteratorDecompressionFails) {
  // Write a physical WAL record with a valid checksum but invalid zstd payload:
  // [0x07][garbage...] so log::Reader succeeds and WalDecompressor fails.
  const std::string path = test::TempFileName(tmpdir_.path(), 1);
  {
    std::unique_ptr<WritableFile> wf;
    ASSERT_TRUE(env_->NewWritableFile(path, &wf, EnvOptions()).ok());
    auto writer_file = std::make_unique<WritableFileWriter>(std::move(wf), path);
    log::Writer writer(std::move(writer_file), 1, false);

    std::string bad_payload;
    bad_payload.push_back(static_cast<char>(kZSTD));
    bad_payload.append("not_a_valid_zstd_frame");
    ASSERT_TRUE(writer.AddRecord(Slice(bad_payload)).ok());
    ASSERT_TRUE(writer.Close().ok());
  }

  WALOptions opts = MakeOptions();
  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  std::unique_ptr<WalIterator> iter;
  ASSERT_TRUE(wal->NewWalIterator(0, &iter).ok());
  EXPECT_FALSE(iter->Valid());
  EXPECT_TRUE(iter->status().IsCorruption()) << iter->status().ToString();
  wal->Close();
}
#endif

// Iterator across multiple files, corruption in second file under AbsoluteConsistency:
// we get all records from file 1, then Valid() false (and status() may be set per 11.4).
TEST_F(WalIteratorTest, IteratorMultiFileCorruptionInSecond) {
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
  std::string path2 = test::TempFileName(tmpdir_.path(), 2);
  {
    std::unique_ptr<WritableFile> wf;
    ASSERT_TRUE(env_->NewWritableFile(path2, &wf, EnvOptions()).ok());
    wf->Append(Slice("\x00\x00\x00\x00\x01"));
    ASSERT_TRUE(wf->Close().ok());
  }

  WALOptions opts = MakeOptions();
  opts.wal_recovery_mode = WALRecoveryMode::kAbsoluteConsistency;
  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  std::unique_ptr<WalIterator> iter;
  ASSERT_TRUE(wal->NewWalIterator(0, &iter).ok());
  int count = 0;
  while (iter->Valid()) {
    count++;
    iter->Next();
  }
  EXPECT_EQ(count, 2);
  EXPECT_TRUE(iter->status().IsCorruption()) << iter->status().ToString();
  wal->Close();
}

}  // namespace mwal
