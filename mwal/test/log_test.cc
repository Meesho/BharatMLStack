#include <gtest/gtest.h>

#include <cstring>
#include <fcntl.h>
#include <memory>
#include <random>
#include <string>
#include <unistd.h>

#include "file/sequential_file_reader.h"
#include "file/writable_file_writer.h"
#include "log/log_format.h"
#include "log/log_reader.h"
#include "log/log_writer.h"
#include "mwal/env.h"
#include "mwal/slice.h"
#include "test_util.h"

namespace mwal {
namespace log {

class LogTest : public ::testing::Test {
 protected:
  test::TempDir tmpdir_;
  Env* env_ = Env::Default();

  std::unique_ptr<Writer> OpenWriter(uint64_t log_num, bool recycle = false) {
    std::string path = test::TempFileName(tmpdir_.path(), log_num);
    std::unique_ptr<WritableFile> file;
    EXPECT_TRUE(env_->NewWritableFile(path, &file, EnvOptions()).ok());
    auto writer = std::make_unique<WritableFileWriter>(std::move(file), path);
    return std::make_unique<Writer>(std::move(writer), log_num, recycle);
  }

  std::unique_ptr<Reader> OpenReader(uint64_t log_num,
                                     Reader::Reporter* reporter = nullptr,
                                     bool checksum = true) {
    std::string path = test::TempFileName(tmpdir_.path(), log_num);
    std::unique_ptr<SequentialFile> file;
    EXPECT_TRUE(env_->NewSequentialFile(path, &file, EnvOptions()).ok());
    auto reader = std::make_unique<SequentialFileReader>(std::move(file), path);
    return std::make_unique<Reader>(std::move(reader), reporter, checksum,
                                    log_num);
  }

  class StringReporter : public Reader::Reporter {
   public:
    std::string message;
    size_t dropped_bytes = 0;
    void Corruption(size_t bytes, const Status& status) override {
      dropped_bytes += bytes;
      message.append(status.ToString());
    }
  };
};

TEST_F(LogTest, EmptyRecord) {
  auto writer = OpenWriter(1);
  ASSERT_TRUE(writer->AddRecord(Slice()).ok());
  writer->Close();

  auto reader = OpenReader(1);
  Slice record;
  std::string scratch;
  ASSERT_TRUE(reader->ReadRecord(&record, &scratch));
  EXPECT_EQ(record.size(), 0u);
  EXPECT_FALSE(reader->ReadRecord(&record, &scratch));
}

TEST_F(LogTest, SmallRecord) {
  auto writer = OpenWriter(1);
  ASSERT_TRUE(writer->AddRecord(Slice("hello")).ok());
  writer->Close();

  auto reader = OpenReader(1);
  Slice record;
  std::string scratch;
  ASSERT_TRUE(reader->ReadRecord(&record, &scratch));
  EXPECT_EQ(record.ToString(), "hello");
}

TEST_F(LogTest, MultipleRecords) {
  auto writer = OpenWriter(1);
  for (int i = 0; i < 100; i++) {
    std::string s = "record_" + std::to_string(i);
    ASSERT_TRUE(writer->AddRecord(Slice(s)).ok());
  }
  writer->Close();

  auto reader = OpenReader(1);
  Slice record;
  std::string scratch;
  for (int i = 0; i < 100; i++) {
    ASSERT_TRUE(reader->ReadRecord(&record, &scratch));
    EXPECT_EQ(record.ToString(), "record_" + std::to_string(i));
  }
  EXPECT_FALSE(reader->ReadRecord(&record, &scratch));
}

TEST_F(LogTest, LargeRecord) {
  std::string big(kBlockSize * 3, 'x');
  auto writer = OpenWriter(1);
  ASSERT_TRUE(writer->AddRecord(Slice(big)).ok());
  writer->Close();

  auto reader = OpenReader(1);
  Slice record;
  std::string scratch;
  ASSERT_TRUE(reader->ReadRecord(&record, &scratch));
  EXPECT_EQ(record.size(), big.size());
  EXPECT_EQ(record.ToString(), big);
}

TEST_F(LogTest, ExactBlockSize) {
  size_t payload = kBlockSize - kHeaderSize;
  std::string data(payload, 'a');

  auto writer = OpenWriter(1);
  ASSERT_TRUE(writer->AddRecord(Slice(data)).ok());
  writer->Close();

  auto reader = OpenReader(1);
  Slice record;
  std::string scratch;
  ASSERT_TRUE(reader->ReadRecord(&record, &scratch));
  EXPECT_EQ(record.size(), payload);
}

TEST_F(LogTest, RandomRecords) {
  std::mt19937 rng(42);
  std::vector<std::string> records;
  auto writer = OpenWriter(1);

  for (int i = 0; i < 200; i++) {
    size_t len = rng() % (kBlockSize * 2);
    std::string s(len, 'A' + (i % 26));
    records.push_back(s);
    ASSERT_TRUE(writer->AddRecord(Slice(s)).ok());
  }
  writer->Close();

  auto reader = OpenReader(1);
  Slice record;
  std::string scratch;
  for (size_t i = 0; i < records.size(); i++) {
    ASSERT_TRUE(reader->ReadRecord(&record, &scratch))
        << "Failed to read record " << i;
    EXPECT_EQ(record.ToString(), records[i]) << "Mismatch at record " << i;
  }
  EXPECT_FALSE(reader->ReadRecord(&record, &scratch));
}

TEST_F(LogTest, RecyclableRecord) {
  auto writer = OpenWriter(42, true);
  ASSERT_TRUE(writer->AddRecord(Slice("recycled")).ok());
  writer->Close();

  auto reader = OpenReader(42);
  Slice record;
  std::string scratch;
  ASSERT_TRUE(reader->ReadRecord(&record, &scratch));
  EXPECT_EQ(record.ToString(), "recycled");
}

TEST_F(LogTest, ChecksumMismatch) {
  auto writer = OpenWriter(1);
  ASSERT_TRUE(writer->AddRecord(Slice("data")).ok());
  writer->Close();

  std::string path = test::TempFileName(tmpdir_.path(), 1);
  // Corrupt one payload byte in-place so checksum verification fails.
  // Record layout is [7-byte header][payload], so offset 7 targets payload[0].
  int fd = open(path.c_str(), O_RDWR);
  ASSERT_GE(fd, 0);
  ASSERT_EQ(lseek(fd, static_cast<off_t>(kHeaderSize), SEEK_SET),
            static_cast<off_t>(kHeaderSize));
  unsigned char corrupt = 0xff;
  ASSERT_EQ(write(fd, &corrupt, 1), 1);
  close(fd);

  StringReporter reporter;
  auto reader = OpenReader(1, &reporter, true);
  Slice record;
  std::string scratch;
  bool read = reader->ReadRecord(&record, &scratch);
  EXPECT_FALSE(read);
  EXPECT_GT(reporter.dropped_bytes, 0u);
}

TEST_F(LogTest, MarginalTrailer) {
  auto writer = OpenWriter(1);
  size_t payload_to_fill = kBlockSize - kHeaderSize - 3;
  std::string fill(payload_to_fill, 'f');
  ASSERT_TRUE(writer->AddRecord(Slice(fill)).ok());
  ASSERT_TRUE(writer->AddRecord(Slice("next")).ok());
  writer->Close();

  auto reader = OpenReader(1);
  Slice record;
  std::string scratch;
  ASSERT_TRUE(reader->ReadRecord(&record, &scratch));
  EXPECT_EQ(record.size(), fill.size());
  ASSERT_TRUE(reader->ReadRecord(&record, &scratch));
  EXPECT_EQ(record.ToString(), "next");
}

}  // namespace log
}  // namespace mwal
