// Sequence number correctness tests (Section 6).
// TC-SEQ-01: Continuity across log rotation
// TC-SEQ-02: Continuity after recovery
// TC-SEQ-03: Concurrent write sequence correctness
// TC-SEQ-04: disableWAL still advances sequence

#include <gtest/gtest.h>

#include <set>
#include <thread>
#include <vector>

#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/wal_file_info.h"
#include "mwal/write_batch.h"
#include "test_util.h"

namespace mwal {

class SequenceNumberTest : public ::testing::Test {
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

// TC-SEQ-01: Sequence continuity across log rotation
// Intent: Sequences are strictly contiguous across file boundaries; no gaps or
// duplicates; GetCurrentLogNumber() increments at each rotation.
TEST_F(SequenceNumberTest, SequenceContinuityAcrossLogRotation) {
  WALOptions opts = MakeOptions();
  opts.max_wal_file_size = 4096;  // 4 KiB to force rotation

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  // Write until we have at least 4 files (3 rotations)
  std::string value(256, 'X');  // ~260 bytes per record with overhead
  int write_count = 0;
  while (true) {
    std::vector<WalFileInfo> files;
    Status s = wal->GetLiveWalFiles(&files);
    ASSERT_TRUE(s.ok()) << s.ToString();
    if (files.size() >= 4) break;

    WriteBatch batch;
    batch.Put("key_" + std::to_string(write_count), value);
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok()) << "Write " << write_count;
    write_count++;
    if (write_count > 500) {
      FAIL() << "Did not reach 4 files after 500 writes; got " << files.size();
    }
  }

  // Extra writes so we have content across all files
  for (int i = 0; i < 20; i++) {
    WriteBatch batch;
    batch.Put("key_extra_" + std::to_string(i), value);
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
  }

  std::vector<WalFileInfo> files;
  ASSERT_TRUE(wal->GetLiveWalFiles(&files).ok());
  ASSERT_GE(files.size(), 4u) << "Need at least 4 files";
  EXPECT_GE(wal->GetCurrentLogNumber(), 4u);

  wal->Close();

  // Verify sequence continuity via recovery (covers all file boundaries)
  std::vector<std::pair<SequenceNumber, int>> recovered;
  opts.recovery_callback = [&recovered](SequenceNumber seq, WriteBatch* batch) {
    recovered.push_back({seq, batch->Count()});
    return Status::OK();
  };
  std::unique_ptr<DBWal> wal2;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal2).ok());
  ASSERT_GE(recovered.size(), 1u) << "Must recover at least 1 batch";

  for (size_t i = 1; i < recovered.size(); i++) {
    SequenceNumber prev_first = recovered[i - 1].first;
    int prev_count = recovered[i - 1].second;
    SequenceNumber curr_first = recovered[i].first;
    EXPECT_EQ(curr_first, prev_first + prev_count)
        << "Batch " << i << ": first_seq=" << curr_first
        << " expected prev_first+prev_count=" << (prev_first + prev_count);
  }

  wal2->Close();
}

// TC-SEQ-02: Sequence continuity after recovery
// Intent: New writes after recovery start exactly where recovery left off;
// GetLatestSequenceNumber() matches the highest sequence seen in the callback.
TEST_F(SequenceNumberTest, SequenceContinuityAfterRecovery) {
  WALOptions opts = MakeOptions();

  // Write 10 batches with varying op counts
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    for (int i = 0; i < 10; i++) {
      WriteBatch batch;
      for (int j = 0; j <= i; j++) {
        batch.Put("key_" + std::to_string(i) + "_" + std::to_string(j),
                  "val");
      }
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok()) << "Write " << i;
    }
    wal->Close();
  }

  std::vector<std::pair<SequenceNumber, int>> recovered;
  {
    std::unique_ptr<DBWal> wal;
    opts.recovery_callback = [&recovered](SequenceNumber seq, WriteBatch* batch) {
      recovered.push_back({seq, batch->Count()});
      return Status::OK();
    };
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    ASSERT_GE(recovered.size(), 1u) << "Recovery must see at least 1 batch";

    SequenceNumber last_recovered_seq = recovered.back().first;
    int last_batch_count = recovered.back().second;
    SequenceNumber expected_max_after_recovery =
        last_recovered_seq + last_batch_count - 1;

    EXPECT_EQ(wal->GetLatestSequenceNumber(), expected_max_after_recovery)
        << "GetLatestSequenceNumber after recovery";

    WriteBatch new_batch;
    new_batch.Put("new_key", "new_val");
    ASSERT_TRUE(wal->Write(WriteOptions(), &new_batch).ok());

    EXPECT_EQ(new_batch.Sequence(), last_recovered_seq + last_batch_count)
        << "New write must start at last_recovered_seq + last_batch_count";

    SequenceNumber expected_max_after_write =
        expected_max_after_recovery + new_batch.Count();
    EXPECT_EQ(wal->GetLatestSequenceNumber(), expected_max_after_write)
        << "GetLatestSequenceNumber after new write";

    wal->Close();
  }
}

// TC-SEQ-03: Concurrent write sequence correctness
// Intent: 32 threads x 100 batches x 1 op each; every sequence 1..3200 appears
// exactly once; no gaps, no duplicates.
TEST_F(SequenceNumberTest, ConcurrentWriteSequenceCorrectness) {
  WALOptions opts = MakeOptions();

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    constexpr int kNumThreads = 32;
    constexpr int kBatchesPerThread = 100;

    std::vector<std::thread> threads;
    for (int t = 0; t < kNumThreads; t++) {
      threads.emplace_back([&wal, t] {
        for (int i = 0; i < kBatchesPerThread; i++) {
          WriteBatch batch;
          batch.Put("key_" + std::to_string(t) + "_" + std::to_string(i),
                    "val");
          Status s = wal->Write(WriteOptions(), &batch);
          ASSERT_TRUE(s.ok()) << "Thread " << t << " batch " << i << ": "
                              << s.ToString();
        }
      });
    }
    for (auto& th : threads) th.join();

    wal->Close();
  }

  std::set<SequenceNumber> sequence_set;
  {
    std::unique_ptr<DBWal> wal;
    opts.recovery_callback = [&sequence_set](SequenceNumber seq,
                                              WriteBatch* batch) {
      int count = batch->Count();
      for (int i = 0; i < count; i++) {
        sequence_set.insert(seq + i);
      }
      return Status::OK();
    };
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
    wal->Close();
  }

  EXPECT_EQ(sequence_set.size(), 3200u)
      << "Expected 3200 unique sequences, got " << sequence_set.size();

  if (!sequence_set.empty()) {
    SequenceNumber min_seq = *sequence_set.begin();
    SequenceNumber max_seq = *sequence_set.rbegin();
    EXPECT_EQ(min_seq, 1u) << "Min sequence must be 1";
    EXPECT_EQ(max_seq, 3200u) << "Max sequence must be 3200";
    EXPECT_EQ(max_seq - min_seq + 1, 3200u)
        << "No gaps: max - min + 1 must equal 3200";
  }
}

// TC-SEQ-04: disableWAL still advances sequence
// Intent: With disableWAL=true, sequence advances but no WAL I/O occurs; after
// Close/Open, only WAL-persisted batches are replayed; GetLatestSequenceNumber()
// after recovery reflects last persisted sequence.
TEST_F(SequenceNumberTest, DisableWALStillAdvancesSequence) {
  WALOptions opts = MakeOptions();

  uint64_t file_size_before = 0;
  uint64_t file_size_after = 0;

  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    // Write 5 batches with WAL enabled
    for (int i = 0; i < 5; i++) {
      WriteBatch batch;
      batch.Put("key_" + std::to_string(i), "value");
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    EXPECT_EQ(wal->GetLatestSequenceNumber(), 5u)
        << "After 5 WAL writes, sequence must be 5";

    // Get WAL file size before disableWAL writes
    std::vector<WalFileInfo> files;
    ASSERT_TRUE(wal->GetLiveWalFiles(&files).ok());
    for (const auto& f : files) {
      file_size_before += f.size_bytes;
    }

    // Write 5 more with disableWAL
    WriteOptions wo;
    wo.disableWAL = true;
    for (int i = 5; i < 10; i++) {
      WriteBatch batch;
      batch.Put("key_" + std::to_string(i), "value");
      ASSERT_TRUE(wal->Write(wo, &batch).ok());
    }
    EXPECT_EQ(wal->GetLatestSequenceNumber(), 10u)
        << "After 5 disableWAL writes, sequence must be 10";

    // Get WAL file size after disableWAL writes
    files.clear();
    ASSERT_TRUE(wal->GetLiveWalFiles(&files).ok());
    for (const auto& f : files) {
      file_size_after += f.size_bytes;
    }

    EXPECT_EQ(file_size_after, file_size_before)
        << "WAL file size must not change during disableWAL writes";

    wal->Close();
  }

  // Reopen and recover
  int recovered_count = 0;
  {
    std::unique_ptr<DBWal> wal;
    opts.recovery_callback = [&recovered_count](SequenceNumber, WriteBatch*) {
      recovered_count++;
      return Status::OK();
    };
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    EXPECT_EQ(recovered_count, 5)
        << "Only 5 WAL-persisted batches must be replayed";
    EXPECT_EQ(wal->GetLatestSequenceNumber(), 5u)
        << "After recovery, GetLatestSequenceNumber must be 5 (last persisted)";

    wal->Close();
  }
}

}  // namespace mwal
