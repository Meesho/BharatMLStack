#include <gtest/gtest.h>

#include <atomic>
#include <chrono>
#include <set>
#include <string>
#include <thread>
#include <vector>

#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/wal_file_info.h"
#include "mwal/wal_iterator.h"
#include "mwal/write_batch.h"
#include "test_util.h"

namespace mwal {

class DBWalTest : public ::testing::Test {
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

// --- Helper to collect recovered entries ---

struct RecoveredEntry {
  SequenceNumber seq;
  std::string key;
  std::string value;
  bool is_delete;
};

class RecoveryCollector : public WriteBatch::Handler {
 public:
  SequenceNumber current_seq = 0;
  std::vector<RecoveredEntry> entries;

  Status Put(const Slice& key, const Slice& value) override {
    entries.push_back({current_seq, key.ToString(), value.ToString(), false});
    return Status::OK();
  }
  Status Delete(const Slice& key) override {
    entries.push_back({current_seq, key.ToString(), "", true});
    return Status::OK();
  }
};

// ========================
// Existing tests
// ========================

TEST_F(DBWalTest, OpenClose) {
  std::unique_ptr<DBWal> wal;
  Status s = DBWal::Open(MakeOptions(), env_, &wal);
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(wal->GetLatestSequenceNumber(), 0u);
  s = wal->Close();
  ASSERT_TRUE(s.ok()) << s.ToString();
}

TEST_F(DBWalTest, SingleWrite) {
  auto opts = MakeOptions();

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    WriteBatch batch;
    batch.Put("hello", "world");
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    EXPECT_EQ(wal->GetLatestSequenceNumber(), 1u);
    wal->Close();
  }

  // Recover and verify.
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    RecoveryCollector collector;
    Status s = wal->Recover([&](SequenceNumber seq, WriteBatch* batch) {
      collector.current_seq = seq;
      return batch->Iterate(&collector);
    });
    ASSERT_TRUE(s.ok()) << s.ToString();
    ASSERT_EQ(collector.entries.size(), 1u);
    EXPECT_EQ(collector.entries[0].key, "hello");
    EXPECT_EQ(collector.entries[0].value, "world");
    EXPECT_EQ(collector.entries[0].seq, 1u);
    wal->Close();
  }
}

TEST_F(DBWalTest, MultipleWrites) {
  auto opts = MakeOptions();
  constexpr int kNumWrites = 100;

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    for (int i = 0; i < kNumWrites; i++) {
      WriteBatch batch;
      batch.Put("key_" + std::to_string(i), "val_" + std::to_string(i));
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    EXPECT_EQ(wal->GetLatestSequenceNumber(),
              static_cast<uint64_t>(kNumWrites));
    wal->Close();
  }

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    int count = 0;
    Status s = wal->Recover([&](SequenceNumber, WriteBatch*) {
      count++;
      return Status::OK();
    });
    ASSERT_TRUE(s.ok()) << s.ToString();
    EXPECT_EQ(count, kNumWrites);
    wal->Close();
  }
}

TEST_F(DBWalTest, SequenceNumberMonotonicity) {
  auto opts = MakeOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  WriteBatch b1;
  b1.Put("a", "1");
  ASSERT_TRUE(wal->Write(WriteOptions(), &b1).ok());
  EXPECT_EQ(b1.Sequence(), 1u);

  WriteBatch b2;
  b2.Put("b", "2");
  b2.Put("c", "3");
  ASSERT_TRUE(wal->Write(WriteOptions(), &b2).ok());
  EXPECT_EQ(b2.Sequence(), 2u);
  EXPECT_EQ(wal->GetLatestSequenceNumber(), 3u);

  WriteBatch b3;
  b3.Put("d", "4");
  ASSERT_TRUE(wal->Write(WriteOptions(), &b3).ok());
  EXPECT_EQ(b3.Sequence(), 4u);
  EXPECT_EQ(wal->GetLatestSequenceNumber(), 4u);

  wal->Close();
}

TEST_F(DBWalTest, DisableWAL) {
  auto opts = MakeOptions();

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    WriteBatch b1;
    b1.Put("visible", "yes");
    ASSERT_TRUE(wal->Write(WriteOptions(), &b1).ok());

    WriteOptions wo;
    wo.disableWAL = true;
    WriteBatch b2;
    b2.Put("invisible", "no");
    ASSERT_TRUE(wal->Write(wo, &b2).ok());

    EXPECT_EQ(wal->GetLatestSequenceNumber(), 2u);
    wal->Close();
  }

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    RecoveryCollector collector;
    wal->Recover([&](SequenceNumber seq, WriteBatch* batch) {
      collector.current_seq = seq;
      return batch->Iterate(&collector);
    });
    ASSERT_EQ(collector.entries.size(), 1u);
    EXPECT_EQ(collector.entries[0].key, "visible");
    wal->Close();
  }
}

TEST_F(DBWalTest, LogRotation) {
  auto opts = MakeOptions();
  opts.max_wal_file_size = 1024;

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    std::string big_value(200, 'X');
    for (int i = 0; i < 20; i++) {
      WriteBatch batch;
      batch.Put("key_" + std::to_string(i), big_value);
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }

    EXPECT_GT(wal->GetCurrentLogNumber(), 1u);
    wal->Close();
  }

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    int count = 0;
    wal->Recover([&](SequenceNumber, WriteBatch*) {
      count++;
      return Status::OK();
    });
    EXPECT_EQ(count, 20);
    wal->Close();
  }
}

TEST_F(DBWalTest, RecoverEmpty) {
  auto opts = MakeOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  int count = 0;
  Status s = wal->Recover([&](SequenceNumber, WriteBatch*) {
    count++;
    return Status::OK();
  });
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(count, 0);
  wal->Close();
}

TEST_F(DBWalTest, ConcurrentWrites) {
  auto opts = MakeOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  constexpr int kThreads = 4;
  constexpr int kWritesPerThread = 50;
  std::atomic<int> errors{0};

  std::vector<std::thread> threads;
  for (int t = 0; t < kThreads; t++) {
    threads.emplace_back([&, t] {
      for (int i = 0; i < kWritesPerThread; i++) {
        WriteBatch batch;
        std::string key =
            "t" + std::to_string(t) + "_k" + std::to_string(i);
        batch.Put(key, "value");
        Status s = wal->Write(WriteOptions(), &batch);
        if (!s.ok()) errors.fetch_add(1);
      }
    });
  }

  for (auto& t : threads) t.join();
  EXPECT_EQ(errors.load(), 0);

  uint64_t expected_seq = kThreads * kWritesPerThread;
  EXPECT_EQ(wal->GetLatestSequenceNumber(), expected_seq);
  wal->Close();

  // Recover and count individual entries (batches may be merged in group commit).
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  RecoveryCollector collector;
  wal->Recover([&](SequenceNumber seq, WriteBatch* batch) {
    collector.current_seq = seq;
    return batch->Iterate(&collector);
  });
  EXPECT_EQ(static_cast<int>(collector.entries.size()),
            kThreads * kWritesPerThread);
  wal->Close();
}

TEST_F(DBWalTest, WriteAfterRecovery) {
  auto opts = MakeOptions();

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    WriteBatch b;
    b.Put("before", "recovery");
    ASSERT_TRUE(wal->Write(WriteOptions(), &b).ok());
    wal->Close();
  }

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    wal->Recover([](SequenceNumber, WriteBatch*) { return Status::OK(); });

    SequenceNumber after_recovery = wal->GetLatestSequenceNumber();
    EXPECT_EQ(after_recovery, 1u);

    WriteBatch b;
    b.Put("after", "recovery");
    ASSERT_TRUE(wal->Write(WriteOptions(), &b).ok());
    EXPECT_EQ(wal->GetLatestSequenceNumber(), 2u);
    EXPECT_GT(b.Sequence(), after_recovery);
    wal->Close();
  }
}

TEST_F(DBWalTest, LargeBatch) {
  auto opts = MakeOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  WriteBatch batch;
  std::string big_value(100000, 'Z');
  batch.Put("big_key", big_value);
  ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
  wal->Close();

  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  RecoveryCollector collector;
  wal->Recover([&](SequenceNumber seq, WriteBatch* b) {
    collector.current_seq = seq;
    return b->Iterate(&collector);
  });
  ASSERT_EQ(collector.entries.size(), 1u);
  EXPECT_EQ(collector.entries[0].key, "big_key");
  EXPECT_EQ(collector.entries[0].value.size(), 100000u);
  wal->Close();
}

TEST_F(DBWalTest, FlushWAL) {
  auto opts = MakeOptions();
  opts.manual_wal_flush = true;

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  WriteBatch batch;
  batch.Put("flushed", "data");
  ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());

  ASSERT_TRUE(wal->FlushWAL(false).ok());
  ASSERT_TRUE(wal->SyncWAL().ok());
  wal->Close();
}

TEST_F(DBWalTest, SyncWrite) {
  auto opts = MakeOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  WriteOptions wo;
  wo.sync = true;

  WriteBatch batch;
  batch.Put("synced", "value");
  ASSERT_TRUE(wal->Write(wo, &batch).ok());
  wal->Close();

  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  RecoveryCollector collector;
  wal->Recover([&](SequenceNumber seq, WriteBatch* b) {
    collector.current_seq = seq;
    return b->Iterate(&collector);
  });
  ASSERT_EQ(collector.entries.size(), 1u);
  EXPECT_EQ(collector.entries[0].key, "synced");
  wal->Close();
}

TEST_F(DBWalTest, NullBatchReturnsError) {
  auto opts = MakeOptions();
  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  Status s = wal->Write(WriteOptions(), nullptr);
  EXPECT_TRUE(s.IsInvalidArgument());
  wal->Close();
}

TEST_F(DBWalTest, WriteAfterClose) {
  auto opts = MakeOptions();
  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  wal->Close();

  WriteBatch batch;
  batch.Put("k", "v");
  Status s = wal->Write(WriteOptions(), &batch);
  EXPECT_TRUE(s.IsAborted());
}

#ifdef MWAL_HAVE_ZSTD

TEST_F(DBWalTest, CompressionRoundtrip) {
  auto opts = MakeOptions();
  opts.wal_compression = kZSTD;

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    for (int i = 0; i < 50; i++) {
      WriteBatch batch;
      batch.Put("key_" + std::to_string(i),
                std::string(256, static_cast<char>('a' + i % 26)));
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    wal->Close();
  }

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    int count = 0;
    RecoveryCollector collector;
    wal->Recover([&](SequenceNumber seq, WriteBatch* batch) {
      collector.current_seq = seq;
      count++;
      return batch->Iterate(&collector);
    });
    EXPECT_EQ(count, 50);
    EXPECT_EQ(collector.entries.size(), 50u);
    EXPECT_EQ(collector.entries[0].key, "key_0");
    wal->Close();
  }
}

#endif

// ========================================
// New tests for hardening & completion
// ========================================

TEST_F(DBWalTest, CloseRaceWithWriters) {
  auto opts = MakeOptions();
  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  constexpr int kThreads = 8;
  constexpr int kWritesPerThread = 100;
  std::atomic<int> success{0};
  std::atomic<int> aborted{0};

  std::vector<std::thread> threads;
  for (int t = 0; t < kThreads; t++) {
    threads.emplace_back([&, t] {
      for (int i = 0; i < kWritesPerThread; i++) {
        WriteBatch batch;
        batch.Put("t" + std::to_string(t) + "_k" + std::to_string(i), "v");
        Status s = wal->Write(WriteOptions(), &batch);
        if (s.ok()) {
          success.fetch_add(1);
        } else if (s.IsAborted()) {
          aborted.fetch_add(1);
        }
      }
    });
  }

  std::this_thread::sleep_for(std::chrono::milliseconds(5));
  Status cs = wal->Close();
  EXPECT_TRUE(cs.ok()) << cs.ToString();

  for (auto& t : threads) t.join();

  EXPECT_GT(success.load() + aborted.load(), 0);
  EXPECT_EQ(success.load() + aborted.load(), kThreads * kWritesPerThread);
}

TEST_F(DBWalTest, SetMinLogToKeep) {
  auto opts = MakeOptions();
  opts.max_wal_file_size = 512;

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  std::string val(200, 'A');
  for (int i = 0; i < 20; i++) {
    WriteBatch batch;
    batch.Put("k" + std::to_string(i), val);
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
  }

  uint64_t current_log = wal->GetCurrentLogNumber();
  EXPECT_GT(current_log, 1u);

  wal->SetMinLogNumberToKeep(current_log);

  std::vector<WalFileInfo> before;
  ASSERT_TRUE(wal->GetLiveWalFiles(&before).ok());

  ASSERT_TRUE(wal->PurgeObsoleteFiles().ok());

  std::vector<WalFileInfo> after;
  ASSERT_TRUE(wal->GetLiveWalFiles(&after).ok());

  for (const auto& f : after) {
    EXPECT_GE(f.log_number, current_log);
  }
  EXPECT_LE(after.size(), before.size());
  wal->Close();
}

TEST_F(DBWalTest, DirectoryLock) {
  auto opts = MakeOptions();

  std::unique_ptr<DBWal> wal1;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal1).ok());

  std::unique_ptr<DBWal> wal2;
  Status s = DBWal::Open(opts, env_, &wal2);
  EXPECT_TRUE(s.IsIOError()) << s.ToString();

  wal1->Close();

  // After closing, a new Open should succeed.
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal2).ok());
  wal2->Close();
}

TEST_F(DBWalTest, PurgeObsoleteExposed) {
  auto opts = MakeOptions();
  opts.max_wal_file_size = 512;

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  std::string val(200, 'B');
  for (int i = 0; i < 15; i++) {
    WriteBatch batch;
    batch.Put("k" + std::to_string(i), val);
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
  }

  uint64_t current_log = wal->GetCurrentLogNumber();
  wal->SetMinLogNumberToKeep(current_log);

  Status s = wal->PurgeObsoleteFiles();
  EXPECT_TRUE(s.ok()) << s.ToString();

  std::vector<WalFileInfo> files;
  ASSERT_TRUE(wal->GetLiveWalFiles(&files).ok());
  for (const auto& f : files) {
    EXPECT_GE(f.log_number, current_log);
  }
  wal->Close();
}

TEST_F(DBWalTest, GetLiveWalFiles) {
  auto opts = MakeOptions();
  opts.max_wal_file_size = 512;

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  std::string val(200, 'C');
  for (int i = 0; i < 10; i++) {
    WriteBatch batch;
    batch.Put("k" + std::to_string(i), val);
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
  }

  std::vector<WalFileInfo> files;
  ASSERT_TRUE(wal->GetLiveWalFiles(&files).ok());
  EXPECT_GE(files.size(), 1u);

  for (size_t i = 1; i < files.size(); i++) {
    EXPECT_GT(files[i].log_number, files[i - 1].log_number);
  }
  wal->Close();
}

TEST_F(DBWalTest, WalIteratorBasic) {
  auto opts = MakeOptions();

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    for (int i = 0; i < 10; i++) {
      WriteBatch batch;
      batch.Put("key_" + std::to_string(i), "val_" + std::to_string(i));
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    wal->Close();
  }

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    std::unique_ptr<WalIterator> iter;
    ASSERT_TRUE(wal->NewWalIterator(0, &iter).ok());

    int count = 0;
    while (iter->Valid()) {
      count++;
      EXPECT_GT(iter->GetSequenceNumber(), 0u);
      iter->Next();
    }
    EXPECT_TRUE(iter->status().ok()) << iter->status().ToString();
    EXPECT_EQ(count, 10);
    wal->Close();
  }
}

TEST_F(DBWalTest, WalIteratorFromMiddle) {
  auto opts = MakeOptions();

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    for (int i = 0; i < 20; i++) {
      WriteBatch batch;
      batch.Put("key_" + std::to_string(i), "val");
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    wal->Close();
  }

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    SequenceNumber start_from = 11;
    std::unique_ptr<WalIterator> iter;
    ASSERT_TRUE(wal->NewWalIterator(start_from, &iter).ok());

    int count = 0;
    while (iter->Valid()) {
      EXPECT_GE(iter->GetSequenceNumber(), start_from);
      count++;
      iter->Next();
    }
    EXPECT_TRUE(iter->status().ok());
    EXPECT_EQ(count, 10);
    wal->Close();
  }
}

TEST_F(DBWalTest, AutoRecoveryOnOpen) {
  auto opts = MakeOptions();

  // Write some data
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
    for (int i = 0; i < 5; i++) {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), "v" + std::to_string(i));
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    wal->Close();
  }

  // Reopen with recovery_callback
  int recovered_count = 0;
  opts.recovery_callback = [&](SequenceNumber, WriteBatch*) {
    recovered_count++;
    return Status::OK();
  };

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    EXPECT_EQ(recovered_count, 5);
    EXPECT_EQ(wal->GetLatestSequenceNumber(), 5u);

    WriteBatch b;
    b.Put("after_auto_recovery", "yes");
    ASSERT_TRUE(wal->Write(WriteOptions(), &b).ok());
    EXPECT_EQ(wal->GetLatestSequenceNumber(), 6u);
    wal->Close();
  }
}

TEST_F(DBWalTest, MergedGroupCommit) {
  auto opts = MakeOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  constexpr int kThreads = 4;
  constexpr int kWritesPerThread = 50;
  std::atomic<int> errors{0};

  std::vector<std::thread> threads;
  for (int t = 0; t < kThreads; t++) {
    threads.emplace_back([&, t] {
      for (int i = 0; i < kWritesPerThread; i++) {
        WriteBatch batch;
        batch.Put("t" + std::to_string(t) + "_" + std::to_string(i), "val");
        Status s = wal->Write(WriteOptions(), &batch);
        if (!s.ok()) errors.fetch_add(1);
      }
    });
  }

  for (auto& t : threads) t.join();
  EXPECT_EQ(errors.load(), 0);
  wal->Close();

  // Recover all entries
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  RecoveryCollector collector;
  int batch_count = 0;
  wal->Recover([&](SequenceNumber seq, WriteBatch* batch) {
    batch_count++;
    collector.current_seq = seq;
    return batch->Iterate(&collector);
  });
  EXPECT_EQ(static_cast<int>(collector.entries.size()),
            kThreads * kWritesPerThread);
  wal->Close();
}

TEST_F(DBWalTest, BackgroundPurge) {
  auto opts = MakeOptions();
  opts.max_wal_file_size = 512;
  opts.purge_interval_seconds = 1;

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  std::string val(200, 'D');
  for (int i = 0; i < 20; i++) {
    WriteBatch batch;
    batch.Put("k" + std::to_string(i), val);
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
  }

  uint64_t current_log = wal->GetCurrentLogNumber();
  EXPECT_GT(current_log, 1u);

  wal->SetMinLogNumberToKeep(current_log);

  std::vector<WalFileInfo> before;
  ASSERT_TRUE(wal->GetLiveWalFiles(&before).ok());

  // Wait for background purge to fire.
  std::this_thread::sleep_for(std::chrono::seconds(2));

  std::vector<WalFileInfo> after;
  ASSERT_TRUE(wal->GetLiveWalFiles(&after).ok());
  EXPECT_LE(after.size(), before.size());
  wal->Close();
}

TEST_F(DBWalTest, WriteStallOnGroupSize) {
  auto opts = MakeOptions();
  opts.max_write_group_size = 2;

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  constexpr int kThreads = 8;
  constexpr int kWritesPerThread = 20;
  std::atomic<int> ok_count{0};

  std::vector<std::thread> threads;
  for (int t = 0; t < kThreads; t++) {
    threads.emplace_back([&, t] {
      for (int i = 0; i < kWritesPerThread; i++) {
        WriteBatch batch;
        batch.Put("t" + std::to_string(t) + "_" + std::to_string(i), "v");
        Status s = wal->Write(WriteOptions(), &batch);
        if (s.ok()) ok_count.fetch_add(1);
      }
    });
  }

  for (auto& t : threads) t.join();
  EXPECT_EQ(ok_count.load(), kThreads * kWritesPerThread);
  wal->Close();
}

TEST_F(DBWalTest, WriteStallOnTotalSize) {
  auto opts = MakeOptions();
  opts.max_total_wal_size = 1;  // 1 byte — immediately stalls

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  // First write: the WAL file starts small enough.
  WriteBatch b1;
  b1.Put("k1", "v1");
  Status s1 = wal->Write(WriteOptions(), &b1);
  // May or may not succeed depending on timing.

  WriteBatch b2;
  b2.Put("k2", "v2");
  Status s2 = wal->Write(WriteOptions(), &b2);
  // At least one should be Busy given the 1-byte limit.
  EXPECT_TRUE(s1.IsBusy() || s2.IsBusy())
      << "s1=" << s1.ToString() << " s2=" << s2.ToString();
  if (s1.ok()) {
    EXPECT_TRUE(s2.IsBusy()) << s2.ToString();
  }
  wal->Close();
}

TEST_F(DBWalTest, TTLPurgeWorks) {
  auto opts = MakeOptions();
  opts.WAL_ttl_seconds = 1;

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    WriteBatch batch;
    batch.Put("old", "data");
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());

    wal->SetMinLogNumberToKeep(wal->GetCurrentLogNumber() + 1);
    wal->Close();
  }

  // Wait for files to age past the TTL.
  std::this_thread::sleep_for(std::chrono::seconds(2));

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    wal->SetMinLogNumberToKeep(wal->GetCurrentLogNumber() + 1);

    ASSERT_TRUE(wal->PurgeObsoleteFiles().ok());

    std::vector<WalFileInfo> files;
    ASSERT_TRUE(wal->GetLiveWalFiles(&files).ok());

    for (const auto& f : files) {
      EXPECT_GE(f.log_number, wal->GetCurrentLogNumber());
    }
    wal->Close();
  }
}

}  // namespace mwal
