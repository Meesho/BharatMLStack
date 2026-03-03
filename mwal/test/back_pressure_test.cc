// Section 10: Back-Pressure Behavior
// TC-BP-01 through TC-BP-03.

#include <gtest/gtest.h>

#include <atomic>
#include <chrono>
#include <string>
#include <thread>
#include <vector>

#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/wal_file_info.h"
#include "mwal/write_batch.h"
#include "test_util.h"

namespace mwal {

class BackPressureTest : public ::testing::Test {
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

// TC-BP-01: Write succeeds after purge clears size
TEST_F(BackPressureTest, TC_BP_01_WriteSucceedsAfterPurgeClearsSize) {
  constexpr uint64_t kLimit = 512 * 1024;  // 512 KiB
  WALOptions opts = MakeOptions();
  opts.max_total_wal_size = kLimit;
  opts.max_wal_file_size = 64 * 1024;  // 64 KiB per file to allow rotation

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  std::string val(1024, 'x');  // 1 KiB per batch
  Status s;
  int writes_before_busy = 0;
  while (true) {
    WriteBatch batch;
    batch.Put("k" + std::to_string(writes_before_busy), val);
    s = wal->Write(WriteOptions(), &batch);
    if (s.IsBusy()) break;
    ASSERT_TRUE(s.ok()) << s.ToString();
    writes_before_busy++;
  }
  EXPECT_TRUE(s.IsBusy()) << s.ToString();

  std::vector<WalFileInfo> files_before;
  ASSERT_TRUE(wal->GetLiveWalFiles(&files_before).ok());
  uint64_t total_before = 0;
  for (const auto& f : files_before) total_before += f.size_bytes;
  EXPECT_GT(total_before, kLimit);

  wal->SetMinLogNumberToKeep(wal->GetCurrentLogNumber());
  ASSERT_TRUE(wal->PurgeObsoleteFiles().ok());

  std::vector<WalFileInfo> files_after;
  ASSERT_TRUE(wal->GetLiveWalFiles(&files_after).ok());
  uint64_t total_after = 0;
  for (const auto& f : files_after) total_after += f.size_bytes;
  EXPECT_LT(total_after, kLimit) << "Purge should reduce size below limit";

  WriteBatch batch;
  batch.Put("after_purge", "value");
  s = wal->Write(WriteOptions(), &batch);
  ASSERT_TRUE(s.ok()) << s.ToString();

  wal->Close();

  WALOptions recover_opts = opts;
  bool found_after_purge = false;
  recover_opts.recovery_callback = [&found_after_purge](SequenceNumber,
                                                        WriteBatch* b) {
    struct Collector : public WriteBatch::Handler {
      bool* found;
      Status Put(const Slice& key, const Slice& value) override {
        if (key == Slice("after_purge") && value == Slice("value")) *found = true;
        return Status::OK();
      }
      Status Delete(const Slice&) override { return Status::OK(); }
    } handler;
    handler.found = &found_after_purge;
    return b->Iterate(&handler);
  };
  std::unique_ptr<DBWal> wal2;
  ASSERT_TRUE(DBWal::Open(recover_opts, env_, &wal2).ok());
  EXPECT_TRUE(found_after_purge) << "Post-purge write must be visible in recovery";
  wal2->Close();
}

// TC-BP-02: Multiple threads blocked on size stall, unblock after purge
TEST_F(BackPressureTest, TC_BP_02_MultipleThreadsUnblockAfterPurge) {
  constexpr uint64_t kLimit = 256 * 1024;  // 256 KiB
  WALOptions opts = MakeOptions();
  opts.max_total_wal_size = kLimit;
  opts.max_wal_file_size = 32 * 1024;

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  std::string val(2048, 'x');
  Status fill_s;
  do {
    WriteBatch b;
    b.Put("fill", val);
    fill_s = wal->Write(WriteOptions(), &b);
    if (fill_s.ok()) continue;
    if (fill_s.IsBusy()) break;
    ASSERT_TRUE(fill_s.ok()) << fill_s.ToString();
  } while (true);
  EXPECT_TRUE(fill_s.IsBusy());

  constexpr int kThreads = 10;
  constexpr int kRetries = 50;
  std::atomic<int> success_count{0};
  std::atomic<int> busy_count{0};
  std::vector<std::thread> threads;

  for (int t = 0; t < kThreads; t++) {
    threads.emplace_back([&, t] {
      for (int r = 0; r < kRetries; r++) {
        WriteBatch batch;
        batch.Put("t" + std::to_string(t) + "_r" + std::to_string(r), "v");
        Status s = wal->Write(WriteOptions(), &batch);
        if (s.ok()) {
          success_count.fetch_add(1);
          return;
        }
        if (s.IsBusy()) {
          busy_count.fetch_add(1);
          std::this_thread::sleep_for(std::chrono::milliseconds(10));
        } else {
          FAIL() << "Unexpected status: " << s.ToString();
        }
      }
    });
  }

  std::this_thread::sleep_for(std::chrono::milliseconds(100));

  wal->SetMinLogNumberToKeep(wal->GetCurrentLogNumber());
  ASSERT_TRUE(wal->PurgeObsoleteFiles().ok());

  for (auto& th : threads) th.join();

  EXPECT_GT(success_count.load(), 0) << "At least one thread should succeed after purge";
  EXPECT_GE(busy_count.load(), 0);

  wal->Close();
}

// TC-BP-03: max_total_wal_size = 0 disables back-pressure
TEST_F(BackPressureTest, TC_BP_03_ZeroDisablesBackPressure) {
  WALOptions opts = MakeOptions();
  opts.max_total_wal_size = 0;
  opts.max_wal_file_size = 256 * 1024;

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  constexpr int kNumWrites = 5000;
  std::string val(4096, 'x');  // 4 KiB each -> ~20 MB total
  for (int i = 0; i < kNumWrites; i++) {
    WriteBatch batch;
    batch.Put("k" + std::to_string(i), val);
    Status s = wal->Write(WriteOptions(), &batch);
    ASSERT_TRUE(s.ok()) << "Write " << i << " returned " << s.ToString();
    EXPECT_FALSE(s.IsBusy()) << "max_total_wal_size=0 should never return Busy";
  }

  wal->Close();
}

}  // namespace mwal
