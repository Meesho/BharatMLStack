// Edge case: Empty log file after Recover() + next Open()
// Tests that GetSortedWalFiles correctly handles zero-byte logs created by Recover().

#include <gtest/gtest.h>

#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/write_batch.h"
#include "test_util.h"

namespace mwal {

class RecoverEmptyLogTest : public ::testing::Test {
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

// EmptyLogAfterRecovery: Recover() creates a new log file. If Open() then fails
// or something crashes before the first Write(), the empty log persists.
// Next Open() must handle the zero-byte log gracefully.
TEST_F(RecoverEmptyLogTest, EmptyLogAfterRecovery) {
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());

    WriteBatch batch;
    batch.Put("k1", "v1");
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());

    ASSERT_TRUE(wal->Close().ok());
  }

  {
    WALOptions opts = MakeOptions();
    int count = 0;
    opts.recovery_callback = [&count](SequenceNumber, WriteBatch*) {
      count++;
      return Status::OK();
    };

    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
    EXPECT_EQ(count, 1);

    std::vector<WalFileInfo> files;
    ASSERT_TRUE(wal->GetLiveWalFiles(&files).ok());

    EXPECT_EQ(files.size(), 2u) << "Should have original log (000001) + new empty log (000002)";
    EXPECT_EQ(files[0].log_number, 1u);
    EXPECT_GT(files[0].size_bytes, 0u) << "Original file should have data";
    EXPECT_EQ(files[1].log_number, 2u);
    EXPECT_EQ(files[1].size_bytes, 0u) << "New log should be empty";

    wal->Close();
  }

  {
    WALOptions opts = MakeOptions();
    int count = 0;
    opts.recovery_callback = [&count](SequenceNumber, WriteBatch*) {
      count++;
      return Status::OK();
    };

    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, env_, &wal);
    ASSERT_TRUE(s.ok()) << "Should handle zero-byte log gracefully: " << s.ToString();
    EXPECT_EQ(count, 1) << "Should recover original write, not from empty log";

    std::vector<WalFileInfo> files;
    ASSERT_TRUE(wal->GetLiveWalFiles(&files).ok());
    EXPECT_GE(files.size(), 1u);

    WriteBatch batch;
    batch.Put("k2", "v2");
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());

    wal->Close();
  }
}

}  // namespace mwal
