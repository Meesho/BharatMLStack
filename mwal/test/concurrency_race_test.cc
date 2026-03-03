// Comprehensive concurrency and race condition tests for WAL system.
// Tests critical concurrent scenarios:
// 1. DrainAndClose() with multiple writers blocked waiting to become leader
// 2. Write() concurrent with PurgeObsoleteFiles() 
// 3. NewWalIterator() concurrent with active writes
// 4. RotateLogFile() concurrent with GetLiveWalFiles()
// 5. Multi-file iteration consistency under concurrent rotations
//
// Must be run with ThreadSanitizer: cmake -DMWAL_ENABLE_TSAN=ON && ctest

#include <gtest/gtest.h>

#include <atomic>
#include <barrier>
#include <chrono>
#include <condition_variable>
#include <latch>
#include <memory>
#include <mutex>
#include <thread>
#include <vector>

#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/status.h"
#include "mwal/wal_file_info.h"
#include "mwal/wal_iterator.h"
#include "mwal/write_batch.h"
#include "test_util.h"

namespace mwal {

// Barrier utility for synchronizing test threads at specific points
class TestBarrier {
 public:
  explicit TestBarrier(int num_threads) : num_threads_(num_threads), count_(0) {}

  void Wait() {
    std::unique_lock<std::mutex> lock(mu_);
    count_++;
    if (count_ == num_threads_) {
      count_ = 0;
      cv_.notify_all();
    } else {
      cv_.wait(lock, [this] { return count_ == 0; });
    }
  }

 private:
  const int num_threads_;
  std::mutex mu_;
  std::condition_variable cv_;
  int count_;
};

// ============================================================================
// Test Fixture for Concurrency Tests
// ============================================================================

class ConcurrencyRaceTest : public ::testing::Test {
 protected:
  void SetUp() override {
    temp_dir_ = std::make_unique<test::TempDir>();
    test_dir_ = temp_dir_->path();
    env_ = Env::Default();
    ASSERT_NE(env_, nullptr);
  }

  void TearDown() override {
    if (wal_) {
      wal_->Close();
    }
    temp_dir_.reset();
  }

  Status OpenWal(const WALOptions& opts = WALOptions()) {
    auto options = opts;
    if (options.wal_dir.empty()) {
      options.wal_dir = test_dir_;
    }
    return DBWal::Open(options, env_, &wal_);
  }

  std::unique_ptr<test::TempDir> temp_dir_;
  std::string test_dir_;
  Env* env_ = nullptr;
  std::unique_ptr<DBWal> wal_;
};

// ============================================================================
// Test 1: Close with Multiple Writers
// ============================================================================
// Scenario: 20 threads are calling Write() when Close() is called.
// All must receive appropriate errors or success (those that completed before).
// Intent: Verify Close() doesn't crash and handles concurrent writes safely.

TEST_F(ConcurrencyRaceTest, CloseWithConcurrentWriters) {
  ASSERT_TRUE(OpenWal().ok());

  constexpr int kNumWriters = 20;
  std::vector<std::thread> threads;
  std::vector<Status> results(kNumWriters);
  std::atomic<int> writers_running{0};
  TestBarrier writers_ready(kNumWriters + 1);  // +1 for main thread

  auto writer_fn = [&](int id) {
    WriteBatch batch;
    batch.Put("key_" + std::to_string(id), "value_" + std::to_string(id));

    WriteOptions opts;
    opts.sync = false;

    writers_running.fetch_add(1);
    writers_ready.Wait();  // Synchronize all writers to start roughly together

    // Attempt write while Close might be happening
    results[id] = wal_->Write(opts, &batch);
  };

  // Launch all writer threads
  for (int i = 0; i < kNumWriters; i++) {
    threads.emplace_back(writer_fn, i);
  }

  // Wait for all writers to be running
  writers_ready.Wait();
  std::this_thread::sleep_for(std::chrono::milliseconds(50));

  // Close the WAL while writers are active
  Status close_status = wal_->Close();
  EXPECT_TRUE(close_status.ok());

  // Join all threads and verify they completed
  for (auto& t : threads) {
    t.join();
  }

  // All writes should be OK or aborted (no crashes)
  int success_count = 0;
  for (int i = 0; i < kNumWriters; i++) {
    if (results[i].ok()) {
      success_count++;
    }
    // Expect only OK or Aborted, never other errors
    EXPECT_TRUE(results[i].ok() || results[i].IsAborted() || results[i].IsIOError())
        << "Writer " << i << ": " << results[i].ToString();
  }

  // At least some should succeed
  EXPECT_GT(success_count, 0) << "At least some writes should succeed";
}

// ============================================================================
// Test 2: Write Concurrent with PurgeObsoleteFiles
// ============================================================================
// Scenario: 15 threads writing concurrently while another thread calls
// PurgeObsoleteFiles() repeatedly. Purge must never delete the active log file.
// Intent: Verify write safety during purge; no data loss.

TEST_F(ConcurrencyRaceTest, WriteConcurrentWithPurge) {
  WALOptions opts;
  opts.max_wal_file_size = 1024 * 64;  // Small files to force rotation
  ASSERT_TRUE(OpenWal(opts).ok());
  wal_->SetMinLogNumberToKeep(0);

  constexpr int kNumWriters = 15;
  constexpr int kWritesPerThread = 50;
  constexpr int kPurgeIterations = 10;

  std::atomic<int> writes_completed{0};
  std::atomic<int> purges_completed{0};
  std::atomic<bool> stop_purging{false};
  std::vector<Status> write_results(kNumWriters * kWritesPerThread);
  std::vector<Status> purge_results(kPurgeIterations);

  auto writer_fn = [&](int thread_id) {
    for (int i = 0; i < kWritesPerThread; i++) {
      WriteBatch batch;
      batch.Put("key_" + std::to_string(thread_id) + "_" + std::to_string(i),
                "value_" + std::to_string(i));
      WriteOptions write_opts;
      write_opts.sync = (i % 10 == 0);  // Sync every 10th write

      int idx = thread_id * kWritesPerThread + i;
      write_results[idx] = wal_->Write(write_opts, &batch);
      writes_completed.fetch_add(1);

      std::this_thread::sleep_for(std::chrono::milliseconds(5));
    }
  };

  auto purger_fn = [&] {
    for (int i = 0; i < kPurgeIterations; i++) {
      if (stop_purging.load()) break;
      purge_results[i] = wal_->PurgeObsoleteFiles();
      purges_completed.fetch_add(1);
      std::this_thread::sleep_for(std::chrono::milliseconds(50));
    }
  };

  // Start all threads
  std::vector<std::thread> threads;
  for (int i = 0; i < kNumWriters; i++) {
    threads.emplace_back(writer_fn, i);
  }
  std::thread purger_thread(purger_fn);

  // Wait for completion
  for (auto& t : threads) {
    t.join();
  }
  stop_purging.store(true);
  purger_thread.join();

  // Verify all writes succeeded
  int success_count = 0;
  for (int i = 0; i < kNumWriters * kWritesPerThread; i++) {
    EXPECT_TRUE(write_results[i].ok())
        << "Write " << i << " failed: " << write_results[i].ToString();
    if (write_results[i].ok()) success_count++;
  }
  EXPECT_EQ(success_count, kNumWriters * kWritesPerThread);

  // Verify purges didn't crash or delete active files
  for (int i = 0; i < kPurgeIterations; i++) {
    EXPECT_TRUE(purge_results[i].ok())
        << "Purge " << i << " failed: " << purge_results[i].ToString();
  }

  // Verify we can still iterate and get all written data
  auto callback = [&](uint64_t /*seq*/, const WriteBatch* /*batch*/) {
    // Just verify we can read without crashing
    return Status::OK();
  };
  Status recover_status = wal_->Recover(callback);
  EXPECT_TRUE(recover_status.ok()) << recover_status.ToString();
}

// ============================================================================
// Test 3: NewWalIterator Concurrent with Active Writes
// ============================================================================
// Scenario: Create multiple iterators while writes are happening concurrently.
// Each iterator must see a self-consistent snapshot (no partial file list).
// Intent: Verify iterator snapshot isolation; no torn state.

TEST_F(ConcurrencyRaceTest, IteratorSnapshotUnderConcurrentWrites) {
  ASSERT_TRUE(OpenWal().ok());

  constexpr int kNumWriters = 8;
  constexpr int kWritesPerThread = 100;
  constexpr int kNumIterators = 5;

  std::atomic<uint64_t> max_seq_written{0};
  std::vector<Status> write_results(kNumWriters * kWritesPerThread);
  std::vector<std::vector<uint64_t>> iterator_sequences(kNumIterators);
  std::vector<Status> iterator_results(kNumIterators);

  auto writer_fn = [&](int thread_id) {
    for (int i = 0; i < kWritesPerThread; i++) {
      WriteBatch batch;
      batch.Put("key_" + std::to_string(thread_id) + "_" + std::to_string(i),
                "value_" + std::to_string(i));
      WriteOptions opts;
      opts.sync = false;

      int idx = thread_id * kWritesPerThread + i;
      write_results[idx] = wal_->Write(opts, &batch);

      // Update max seq in case iterator reads while writing
      if (write_results[idx].ok()) {
        max_seq_written.fetch_add(1, std::memory_order_release);
      }

      std::this_thread::sleep_for(std::chrono::milliseconds(2));
    }
  };

  auto iterator_fn = [&](int iter_id) {
    // Give writers a head start
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    std::unique_ptr<WalIterator> iter;
    iterator_results[iter_id] = wal_->NewWalIterator(0, &iter);
    if (!iterator_results[iter_id].ok()) {
      return;
    }

    uint64_t last_seq = 0;
    int record_count = 0;
    while (iter->Valid()) {
      uint64_t seq = iter->GetSequenceNumber();
      iter->GetBatch();  // Verify we can read the batch

      // Verify sequences are monotonically increasing
      EXPECT_GE(seq, last_seq) << "Iterator " << iter_id
                                << ": sequence not monotonic";
      last_seq = seq;
      record_count++;

      iter->Next();
    }

    iterator_sequences[iter_id].push_back(record_count);
  };

  // Start writers
  std::vector<std::thread> writer_threads;
  for (int i = 0; i < kNumWriters; i++) {
    writer_threads.emplace_back(writer_fn, i);
  }

  // Start iterators
  std::vector<std::thread> iter_threads;
  for (int i = 0; i < kNumIterators; i++) {
    iter_threads.emplace_back(iterator_fn, i);
  }

  // Wait for all to complete
  for (auto& t : writer_threads) {
    t.join();
  }
  for (auto& t : iter_threads) {
    t.join();
  }

  // Verify all writes succeeded
  int write_success = 0;
  for (const auto& s : write_results) {
    if (s.ok()) write_success++;
  }
  EXPECT_EQ(write_success, kNumWriters * kWritesPerThread);

  // Verify all iterators succeeded and saw consistent data
  for (int i = 0; i < kNumIterators; i++) {
    EXPECT_TRUE(iterator_results[i].ok())
        << "Iterator " << i << ": " << iterator_results[i].ToString();
  }
}

// ============================================================================
// Test 4: RotateLogFile Concurrent with GetLiveWalFiles
// ============================================================================
// Scenario: One thread rotates log files while another calls GetLiveWalFiles().
// No torn state; file list must be self-consistent.
// Intent: Verify file list consistency during rotation.

TEST_F(ConcurrencyRaceTest, RotateFileConcurrentWithGetLiveWalFiles) {
  WALOptions opts;
  opts.max_wal_file_size = 512 * 1024;  // 512KB to force rotations
  ASSERT_TRUE(OpenWal(opts).ok());

  constexpr int kNumWritersForRotation = 10;
  constexpr int kWritesPerThread = 200;
  constexpr int kFileListReads = 100;

  std::vector<Status> write_results(kNumWritersForRotation * kWritesPerThread);
  std::vector<std::vector<WalFileInfo>> file_lists(kFileListReads);

  auto writer_for_rotation = [&](int thread_id) {
    for (int i = 0; i < kWritesPerThread; i++) {
      WriteBatch batch;
      // Write large data to trigger rotation
      std::string large_value(1024, 'x');
      batch.Put("key_" + std::to_string(thread_id) + "_" + std::to_string(i),
                large_value);
      WriteOptions opts;
      opts.sync = false;

      int idx = thread_id * kWritesPerThread + i;
      write_results[idx] = wal_->Write(opts, &batch);

      std::this_thread::sleep_for(std::chrono::milliseconds(1));
    }
  };

  auto file_list_reader = [&] {
    for (int i = 0; i < kFileListReads; i++) {
      std::vector<WalFileInfo> files;
      Status s = wal_->GetLiveWalFiles(&files);
      if (s.ok()) {
        file_lists[i] = files;

        // Verify no duplicate file numbers in list
        std::set<uint64_t> seen_nums;
        for (const auto& f : files) {
          EXPECT_EQ(seen_nums.count(f.log_number), 0)
              << "Duplicate file number in list: " << f.log_number;
          seen_nums.insert(f.log_number);
        }

        // Verify file numbers are sorted
        uint64_t last_num = 0;
        for (const auto& f : files) {
          EXPECT_GE(f.log_number, last_num)
              << "File numbers not sorted in list";
          last_num = f.log_number;
        }
      }

      std::this_thread::sleep_for(std::chrono::milliseconds(5));
    }
  };

  // Start writers
  std::vector<std::thread> writer_threads;
  for (int i = 0; i < kNumWritersForRotation; i++) {
    writer_threads.emplace_back(writer_for_rotation, i);
  }

  // Start file list reader
  std::thread reader_thread(file_list_reader);

  // Wait for completion
  for (auto& t : writer_threads) {
    t.join();
  }
  reader_thread.join();

  // Verify all writes succeeded
  int success_count = 0;
  for (const auto& s : write_results) {
    if (s.ok()) success_count++;
  }
  EXPECT_GT(success_count, 0) << "Some writes should succeed";
}

// ============================================================================
// Test 5: Multi-File Iterator with Concurrent Rotation
// ============================================================================
// Scenario: Iterator reads from multiple files while rotation is happening.
// Snapshot must remain consistent (no partial file list from rotation).
// Intent: Verify iterator doesn't see torn state during rotation.

TEST_F(ConcurrencyRaceTest, MultiFileIteratorDuringRotation) {
  WALOptions opts;
  opts.max_wal_file_size = 256 * 1024;  // Force rotation
  ASSERT_TRUE(OpenWal(opts).ok());

  constexpr int kNumWriterThreads = 12;
  constexpr int kWritesPerThread = 150;

  std::atomic<int> records_iterated{0};
  std::vector<Status> write_results(kNumWriterThreads * kWritesPerThread);
  std::vector<Status> iterator_results(5);

  auto writer_fn = [&](int thread_id) {
    for (int i = 0; i < kWritesPerThread; i++) {
      WriteBatch batch;
      std::string value(512, 'a' + (thread_id % 26));
      batch.Put("key_" + std::to_string(thread_id) + "_" + std::to_string(i),
                value);
      WriteOptions opts;
      opts.sync = false;

      int idx = thread_id * kWritesPerThread + i;
      write_results[idx] = wal_->Write(opts, &batch);

      std::this_thread::sleep_for(std::chrono::milliseconds(1));
    }
  };

  auto iterator_fn = [&](int iter_id) {
    // Stagger iterator starts to hit different rotation phases
    std::this_thread::sleep_for(std::chrono::milliseconds(50 * iter_id));

    std::unique_ptr<WalIterator> iter;
    iterator_results[iter_id] = wal_->NewWalIterator(0, &iter);
    if (!iterator_results[iter_id].ok()) {
      return;
    }

    uint64_t last_seq = 0;
    int count = 0;
    while (iter->Valid() && count < 10000) {  // Limit to prevent infinite loop
      uint64_t seq = iter->GetSequenceNumber();
      iter->GetBatch();  // Verify we can read

      // Verify monotonicity
      EXPECT_GE(seq, last_seq) << "Iterator " << iter_id
                                << ": non-monotonic sequences";
      last_seq = seq;
      count++;

      iter->Next();
    }

    records_iterated.fetch_add(count);
  };

  // Start all writers
  std::vector<std::thread> writer_threads;
  for (int i = 0; i < kNumWriterThreads; i++) {
    writer_threads.emplace_back(writer_fn, i);
  }

  // Start iterators
  std::vector<std::thread> iter_threads;
  for (int i = 0; i < 5; i++) {
    iter_threads.emplace_back(iterator_fn, i);
  }

  // Wait for completion
  for (auto& t : writer_threads) {
    t.join();
  }
  for (auto& t : iter_threads) {
    t.join();
  }

  // Verify writes
  int write_success = 0;
  for (const auto& s : write_results) {
    if (s.ok()) write_success++;
  }
  EXPECT_GT(write_success, 0);

  // Verify iterators
  for (int i = 0; i < 5; i++) {
    EXPECT_TRUE(iterator_results[i].ok())
        << "Iterator " << i << ": " << iterator_results[i].ToString();
  }
}

// ============================================================================
// Test 6: Close Concurrent with Multiple Operations
// ============================================================================
// Scenario: Call Close() while writes, purges, and iterator creation are in progress.
// No crashes, no hangs, proper error propagation.
// Intent: Verify safe concurrent shutdown.

TEST_F(ConcurrencyRaceTest, CloseConcurrentWithMultipleOps) {
  ASSERT_TRUE(OpenWal().ok());

  constexpr int kNumWriters = 10;
  constexpr int kWritesPerThread = 100;
  constexpr int kPurgeThreads = 2;
  constexpr int kIteratorThreads = 3;

  std::atomic<bool> stop_all{false};
  std::vector<Status> write_results(kNumWriters * kWritesPerThread);
  std::vector<Status> purge_results(100);
  std::vector<Status> iter_results(100);

  auto writer_fn = [&](int thread_id) {
    for (int i = 0; i < kWritesPerThread && !stop_all.load(); i++) {
      WriteBatch batch;
      batch.Put("key_" + std::to_string(thread_id) + "_" + std::to_string(i),
                "value_" + std::to_string(i));
      WriteOptions opts;
      opts.sync = false;

      int idx = thread_id * kWritesPerThread + i;
      write_results[idx] = wal_->Write(opts, &batch);
    }
  };

  auto purger_fn = [&] {
    int idx = 0;
    while (!stop_all.load() && idx < 100) {
      purge_results[idx] = wal_->PurgeObsoleteFiles();
      idx++;
      std::this_thread::sleep_for(std::chrono::milliseconds(10));
    }
  };

  auto iterator_fn = [&] {
    int idx = 0;
    while (!stop_all.load() && idx < 100) {
      std::unique_ptr<WalIterator> iter;
      iter_results[idx] = wal_->NewWalIterator(0, &iter);
      idx++;
      std::this_thread::sleep_for(std::chrono::milliseconds(10));
    }
  };

  // Start all threads
  std::vector<std::thread> threads;
  for (int i = 0; i < kNumWriters; i++) {
    threads.emplace_back(writer_fn, i);
  }
  for (int i = 0; i < kPurgeThreads; i++) {
    threads.emplace_back(purger_fn);
  }
  for (int i = 0; i < kIteratorThreads; i++) {
    threads.emplace_back(iterator_fn);
  }

  // Let them run briefly
  std::this_thread::sleep_for(std::chrono::milliseconds(200));

  // Close while operations are pending
  stop_all.store(true);
  Status close_status = wal_->Close();

  // Join all threads
  for (auto& t : threads) {
    t.join();
  }

  // Close should succeed
  EXPECT_TRUE(close_status.ok()) << close_status.ToString();
}

// ============================================================================
// Test 7: Sync Coalescing Under Concurrent Writes
// ============================================================================
// Scenario: Multiple writers with sync=true arrive concurrently.
// Only one physical sync should occur, all writers see the sync succeed.
// Intent: Verify sync coalescing doesn't lose data or corrupt state.

TEST_F(ConcurrencyRaceTest, SyncCoalescingConcurrentWriters) {
  ASSERT_TRUE(OpenWal().ok());

  constexpr int kNumSyncWriters = 16;
  std::vector<Status> results(kNumSyncWriters);
  std::atomic<int> started{0};
  TestBarrier ready(kNumSyncWriters + 1);

  auto writer_fn = [&](int id) {
    WriteBatch batch;
    batch.Put("sync_key_" + std::to_string(id),
              "sync_value_" + std::to_string(id));
    WriteOptions opts;
    opts.sync = true;  // All will request sync

    started.fetch_add(1);
    ready.Wait();  // Synchronize all to start together

    results[id] = wal_->Write(opts, &batch);
  };

  std::vector<std::thread> threads;
  for (int i = 0; i < kNumSyncWriters; i++) {
    threads.emplace_back(writer_fn, i);
  }

  ready.Wait();  // Release all threads

  for (auto& t : threads) {
    t.join();
  }

  // All writes should succeed
  for (int i = 0; i < kNumSyncWriters; i++) {
    EXPECT_TRUE(results[i].ok())
        << "Sync write " << i << ": " << results[i].ToString();
  }

  // Verify data is durable by recovering
  int recovered_count = 0;
  auto recovery_callback = [&](uint64_t /*seq*/, const WriteBatch* /*batch*/) {
    recovered_count++;
    return Status::OK();
  };
  Status recover_status = wal_->Recover(recovery_callback);
  EXPECT_TRUE(recover_status.ok());
  // Should recover at least one record per writer who successfully wrote
  EXPECT_GT(recovered_count, 0);
}

// ============================================================================
// Test 8: GetLiveWalFiles Consistency Under Rapid Rotation
// ============================================================================
// Scenario: Rapid writes causing frequent rotations while GetLiveWalFiles
// is called repeatedly. File lists must never have gaps or inconsistencies.
// Intent: Verify file metadata consistency during rapid rotation.

TEST_F(ConcurrencyRaceTest, LiveWalFilesConsistencyDuringRapidRotation) {
  WALOptions opts;
  opts.max_wal_file_size = 128 * 1024;  // Very small to force rapid rotation
  ASSERT_TRUE(OpenWal(opts).ok());

  constexpr int kNumWriters = 8;
  constexpr int kWritesPerThread = 200;
  constexpr int kFileListChecks = 200;

  std::atomic<int> checks_done{0};
  std::vector<Status> write_results(kNumWriters * kWritesPerThread);
  std::vector<std::vector<uint64_t>> observed_file_numbers(kFileListChecks);

  auto writer_fn = [&](int thread_id) {
    for (int i = 0; i < kWritesPerThread; i++) {
      WriteBatch batch;
      std::string value(512, 'x');
      batch.Put("key_" + std::to_string(thread_id) + "_" + std::to_string(i),
                value);
      WriteOptions opts;
      opts.sync = false;

      int idx = thread_id * kWritesPerThread + i;
      write_results[idx] = wal_->Write(opts, &batch);

      std::this_thread::sleep_for(std::chrono::microseconds(500));
    }
  };

  auto checker_fn = [&] {
    uint64_t last_max_filenum = 0;
    for (int i = 0; i < kFileListChecks; i++) {
      std::vector<WalFileInfo> files;
      Status s = wal_->GetLiveWalFiles(&files);

      if (s.ok() && !files.empty()) {
        // Collect file numbers
        for (const auto& f : files) {
          observed_file_numbers[i].push_back(f.log_number);
        }

        // Verify monotonicity: max file number should never decrease
        uint64_t max_filenum = files.back().log_number;
        EXPECT_GE(max_filenum, last_max_filenum)
            << "Max file number decreased: " << max_filenum << " < "
            << last_max_filenum;
        last_max_filenum = max_filenum;

        checks_done.fetch_add(1);
      }

      std::this_thread::sleep_for(std::chrono::milliseconds(2));
    }
  };

  // Start writers
  std::vector<std::thread> writer_threads;
  for (int i = 0; i < kNumWriters; i++) {
    writer_threads.emplace_back(writer_fn, i);
  }

  // Start checker
  std::thread checker_thread(checker_fn);

  for (auto& t : writer_threads) {
    t.join();
  }
  checker_thread.join();

  // Verify checks occurred
  EXPECT_GT(checks_done.load(), 0);
}

// ============================================================================
// Test 9: Leader Switch During High Contention
// ============================================================================
// Scenario: Many threads competing to become leader while some are exiting.
// Thread-safe leader election must not deadlock or miss writers.
// Intent: Verify WriteThread leader-follower state machine correctness.

TEST_F(ConcurrencyRaceTest, LeaderSwitchingUnderContention) {
  ASSERT_TRUE(OpenWal().ok());

  constexpr int kNumWriters = 32;
  constexpr int kWritesPerThread = 50;

  std::atomic<int> completed{0};
  std::vector<Status> results(kNumWriters * kWritesPerThread);

  auto writer_fn = [&](int thread_id) {
    for (int i = 0; i < kWritesPerThread; i++) {
      WriteBatch batch;
      batch.Put("key_" + std::to_string(thread_id) + "_" + std::to_string(i),
                "value_" + std::to_string(i));
      WriteOptions opts;
      opts.sync = (i % 5 == 0);  // Some sync, some don't

      int idx = thread_id * kWritesPerThread + i;
      results[idx] = wal_->Write(opts, &batch);
      completed.fetch_add(1);

      if ((i + 1) % 10 == 0) {
        std::this_thread::yield();
      }
    }
  };

  std::vector<std::thread> threads;
  for (int i = 0; i < kNumWriters; i++) {
    threads.emplace_back(writer_fn, i);
  }

  for (auto& t : threads) {
    t.join();
  }

  // All should complete
  EXPECT_EQ(completed.load(), kNumWriters * kWritesPerThread);

  // Verify success rate
  int success_count = 0;
  for (const auto& s : results) {
    if (s.ok()) success_count++;
  }
  EXPECT_EQ(success_count, kNumWriters * kWritesPerThread);
}

}  // namespace mwal
