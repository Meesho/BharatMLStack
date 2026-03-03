// Unit tests for WriteCoalescer: submit, back-pressure, shutdown, error propagation.

#include <gtest/gtest.h>

#include <atomic>
#include <chrono>
#include <string>
#include <thread>
#include <vector>

#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/write_batch.h"
#include "wal/write_coalescer.h"
#include "test_util.h"

namespace mwal {

// --- Unit tests for WriteCoalescer directly ---

class WriteCoalescerTest : public ::testing::Test {
 protected:
  void SetUp() override { env_ = Env::Default(); }

  Env* env_ = nullptr;
};

TEST_F(WriteCoalescerTest, SingleSubmit) {
  WriteCoalescer coalescer(100);
  std::atomic<int> call_count{0};
  Status callback_status = Status::OK();

  coalescer.Start([&](std::vector<WriteBatch*>& batches) -> Status {
    call_count.fetch_add(1);
    EXPECT_EQ(batches.size(), 1u);
    return callback_status;
  });

  WriteBatch batch;
  batch.Put("key", "value");
  Status s = coalescer.Submit(&batch);
  EXPECT_TRUE(s.ok()) << s.ToString();
  EXPECT_GE(call_count.load(), 1);

  coalescer.Stop();
}

TEST_F(WriteCoalescerTest, MultipleProducers) {
  constexpr int kProducers = 8;
  constexpr int kWritesPerProducer = 100;
  WriteCoalescer coalescer(10000);

  std::atomic<int> total_batches_written{0};

  coalescer.Start([&](std::vector<WriteBatch*>& batches) -> Status {
    total_batches_written.fetch_add(static_cast<int>(batches.size()));
    return Status::OK();
  });

  std::vector<std::thread> producers;
  std::atomic<int> errors{0};

  for (int p = 0; p < kProducers; p++) {
    producers.emplace_back([&, p] {
      for (int i = 0; i < kWritesPerProducer; i++) {
        WriteBatch batch;
        batch.Put("p" + std::to_string(p) + "_k" + std::to_string(i), "v");
        Status s = coalescer.Submit(&batch);
        if (!s.ok()) errors.fetch_add(1);
      }
    });
  }

  for (auto& t : producers) t.join();
  coalescer.Stop();

  EXPECT_EQ(errors.load(), 0);
  EXPECT_EQ(total_batches_written.load(), kProducers * kWritesPerProducer);
}

TEST_F(WriteCoalescerTest, BackPressureBlocksWhenFull) {
  constexpr size_t kMaxDepth = 2;
  WriteCoalescer coalescer(kMaxDepth);

  std::mutex gate_mu;
  std::condition_variable gate_cv;
  bool gate_open = false;

  coalescer.Start([&](std::vector<WriteBatch*>& batches) -> Status {
    // Block the writer thread until we release the gate
    std::unique_lock<std::mutex> lock(gate_mu);
    gate_cv.wait(lock, [&] { return gate_open; });
    (void)batches;
    return Status::OK();
  });

  // Fill the queue to max depth from background threads
  std::atomic<int> submitted{0};
  std::vector<std::thread> threads;
  constexpr int kOverflow = 6;

  for (int i = 0; i < kOverflow; i++) {
    threads.emplace_back([&] {
      WriteBatch batch;
      batch.Put("key", "val");
      submitted.fetch_add(1);
      coalescer.Submit(&batch);
    });
  }

  // Give threads time to attempt submission
  std::this_thread::sleep_for(std::chrono::milliseconds(100));

  // Release the gate so the writer thread can drain
  {
    std::lock_guard<std::mutex> lock(gate_mu);
    gate_open = true;
  }
  gate_cv.notify_one();

  for (auto& t : threads) t.join();
  coalescer.Stop();

  EXPECT_EQ(submitted.load(), kOverflow);
}

TEST_F(WriteCoalescerTest, ShutdownDrainsPending) {
  WriteCoalescer coalescer(1000);
  std::atomic<int> written{0};

  coalescer.Start([&](std::vector<WriteBatch*>& batches) -> Status {
    written.fetch_add(static_cast<int>(batches.size()));
    return Status::OK();
  });

  constexpr int kWrites = 50;
  std::vector<std::thread> threads;
  for (int i = 0; i < kWrites; i++) {
    threads.emplace_back([&, i] {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), "v");
      coalescer.Submit(&batch);
    });
  }

  for (auto& t : threads) t.join();
  coalescer.Stop();

  EXPECT_EQ(written.load(), kWrites);
}

TEST_F(WriteCoalescerTest, ErrorPropagation) {
  WriteCoalescer coalescer(100);
  Status injected_error = Status::IOError("disk full");

  coalescer.Start([&](std::vector<WriteBatch*>&) -> Status {
    return injected_error;
  });

  constexpr int kProducers = 4;
  std::vector<std::thread> threads;
  std::vector<Status> results(kProducers);

  for (int i = 0; i < kProducers; i++) {
    threads.emplace_back([&, i] {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), "v");
      results[i] = coalescer.Submit(&batch);
    });
  }

  for (auto& t : threads) t.join();
  coalescer.Stop();

  for (int i = 0; i < kProducers; i++) {
    EXPECT_TRUE(results[i].IsIOError())
        << "Producer " << i << " got: " << results[i].ToString();
  }
}

// --- Time-based flush tests ---

TEST_F(WriteCoalescerTest, TimeBasedFlushFiresWithPendingBatches) {
  constexpr auto kInterval = std::chrono::milliseconds(50);
  WriteCoalescer coalescer(100, kInterval);
  std::atomic<int> call_count{0};

  coalescer.Start([&](std::vector<WriteBatch*>& batches) -> Status {
    call_count.fetch_add(1);
    EXPECT_EQ(batches.size(), 1u);
    return Status::OK();
  });

  WriteBatch batch;
  batch.Put("key", "value");
  auto t0 = std::chrono::steady_clock::now();
  Status s = coalescer.Submit(&batch);
  auto t1 = std::chrono::steady_clock::now();
  EXPECT_TRUE(s.ok()) << s.ToString();
  EXPECT_GE(call_count.load(), 1);
  auto elapsed_ms = std::chrono::duration_cast<std::chrono::milliseconds>(t1 - t0).count();
  EXPECT_LE(elapsed_ms, 150) << "Flush should complete within ~2x interval";

  coalescer.Stop();
}

TEST_F(WriteCoalescerTest, TimeBasedFlushDisabledWhenIntervalZero) {
  WriteCoalescer coalescer(100, std::chrono::milliseconds(0));
  std::atomic<int> call_count{0};

  coalescer.Start([&](std::vector<WriteBatch*>& batches) -> Status {
    call_count.fetch_add(1);
    EXPECT_EQ(batches.size(), 1u);
    return Status::OK();
  });

  WriteBatch batch;
  batch.Put("key", "value");
  Status s = coalescer.Submit(&batch);
  EXPECT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(call_count.load(), 1);

  coalescer.Stop();
}

TEST_F(WriteCoalescerTest, MultipleBatchesFlushedTogetherOnTimeout) {
  constexpr auto kInterval = std::chrono::milliseconds(50);
  constexpr int kN = 3;
  WriteCoalescer coalescer(100, kInterval);
  std::atomic<int> call_count{0};
  std::vector<size_t> batch_sizes;
  std::mutex batch_sizes_mu;

  coalescer.Start([&](std::vector<WriteBatch*>& batches) -> Status {
    call_count.fetch_add(1);
    {
      std::lock_guard<std::mutex> lock(batch_sizes_mu);
      batch_sizes.push_back(batches.size());
    }
    return Status::OK();
  });

  std::vector<WriteBatch> batches(kN);
  for (int i = 0; i < kN; i++) {
    batches[i].Put("k" + std::to_string(i), "v");
  }
  auto t0 = std::chrono::steady_clock::now();
  for (int i = 0; i < kN; i++) {
    Status s = coalescer.Submit(&batches[i]);
    EXPECT_TRUE(s.ok()) << s.ToString();
  }
  auto t1 = std::chrono::steady_clock::now();
  coalescer.Stop();

  EXPECT_GE(call_count.load(), 1);
  size_t total_written = 0;
  {
    std::lock_guard<std::mutex> lock(batch_sizes_mu);
    for (size_t n : batch_sizes) total_written += n;
  }
  EXPECT_EQ(total_written, static_cast<size_t>(kN));
  auto elapsed_ms = std::chrono::duration_cast<std::chrono::milliseconds>(t1 - t0).count();
  EXPECT_LE(elapsed_ms, 200) << "All submits should complete within ~2x interval";
}

TEST_F(WriteCoalescerTest, SubmitBeforeTimeoutWakesWriter) {
  constexpr auto kInterval = std::chrono::milliseconds(100);
  WriteCoalescer coalescer(100, kInterval);
  std::atomic<int> total_batches{0};

  coalescer.Start([&](std::vector<WriteBatch*>& batches) -> Status {
    total_batches.fetch_add(static_cast<int>(batches.size()));
    return Status::OK();
  });

  WriteBatch b1, b2;
  b1.Put("a", "1");
  b2.Put("b", "2");
  Status s1 = coalescer.Submit(&b1);
  std::this_thread::sleep_for(std::chrono::milliseconds(20));
  Status s2 = coalescer.Submit(&b2);

  EXPECT_TRUE(s1.ok()) << s1.ToString();
  EXPECT_TRUE(s2.ok()) << s2.ToString();
  EXPECT_EQ(total_batches.load(), 2);

  coalescer.Stop();
}

TEST_F(WriteCoalescerTest, TimeoutAndStopDrainsPending) {
  constexpr auto kInterval = std::chrono::milliseconds(50);
  WriteCoalescer coalescer(100, kInterval);
  std::atomic<int> written{0};

  coalescer.Start([&](std::vector<WriteBatch*>& batches) -> Status {
    written.fetch_add(static_cast<int>(batches.size()));
    return Status::OK();
  });

  constexpr int kN = 5;
  std::vector<std::thread> threads;
  for (int i = 0; i < kN; i++) {
    threads.emplace_back([&, i] {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), "v");
      Status s = coalescer.Submit(&batch);
      EXPECT_TRUE(s.ok()) << s.ToString();
    });
  }

  for (auto& t : threads) t.join();
  coalescer.Stop();

  EXPECT_EQ(written.load(), kN);
}

TEST_F(WriteCoalescerTest, BackPressureWithTimeout) {
  constexpr size_t kMaxDepth = 2;
  constexpr auto kInterval = std::chrono::milliseconds(50);
  WriteCoalescer coalescer(kMaxDepth, kInterval);

  std::mutex gate_mu;
  std::condition_variable gate_cv;
  bool gate_open = false;
  std::atomic<int> written{0};

  coalescer.Start([&](std::vector<WriteBatch*>& batches) -> Status {
    written.fetch_add(static_cast<int>(batches.size()));
    std::unique_lock<std::mutex> lock(gate_mu);
    gate_cv.wait(lock, [&] { return gate_open; });
    return Status::OK();
  });

  std::vector<std::thread> threads;
  constexpr int kNumWrites = 4;
  for (int i = 0; i < kNumWrites; i++) {
    threads.emplace_back([&] {
      WriteBatch batch;
      batch.Put("key", "val");
      coalescer.Submit(&batch);
    });
  }

  std::this_thread::sleep_for(std::chrono::milliseconds(150));
  {
    std::lock_guard<std::mutex> lock(gate_mu);
    gate_open = true;
  }
  gate_cv.notify_one();

  for (auto& t : threads) t.join();
  coalescer.Stop();

  EXPECT_EQ(written.load(), kNumWrites);
}

TEST_F(WriteCoalescerTest, ErrorPropagationOnTimeoutFlush) {
  constexpr auto kInterval = std::chrono::milliseconds(50);
  WriteCoalescer coalescer(100, kInterval);
  Status injected_error = Status::IOError("disk full");

  coalescer.Start([&](std::vector<WriteBatch*>&) -> Status {
    return injected_error;
  });

  WriteBatch batch;
  batch.Put("key", "val");
  Status s = coalescer.Submit(&batch);
  EXPECT_TRUE(s.IsIOError()) << s.ToString();
  coalescer.Stop();
}

TEST_F(WriteCoalescerTest, NoDoubleFlush) {
  constexpr auto kInterval = std::chrono::milliseconds(100);
  WriteCoalescer coalescer(100, kInterval);
  std::atomic<int> call_count{0};

  coalescer.Start([&](std::vector<WriteBatch*>& batches) -> Status {
    call_count.fetch_add(1);
    EXPECT_EQ(batches.size(), 1u);
    return Status::OK();
  });

  WriteBatch batch;
  batch.Put("key", "value");
  Status s = coalescer.Submit(&batch);
  EXPECT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(call_count.load(), 1);

  coalescer.Stop();
}

TEST_F(WriteCoalescerTest, IdleLatencyBoundedByInterval) {
  constexpr auto kInterval = std::chrono::milliseconds(50);
  WriteCoalescer coalescer(100, kInterval);
  std::atomic<int> call_count{0};

  coalescer.Start([&](std::vector<WriteBatch*>&) -> Status {
    call_count.fetch_add(1);
    return Status::OK();
  });

  WriteBatch batch;
  batch.Put("key", "value");
  auto t0 = std::chrono::steady_clock::now();
  Status s = coalescer.Submit(&batch);
  auto t1 = std::chrono::steady_clock::now();
  EXPECT_TRUE(s.ok()) << s.ToString();

  auto latency_ms = std::chrono::duration_cast<std::chrono::milliseconds>(t1 - t0).count();
  EXPECT_LE(latency_ms, 200) << "Latency should be bounded by ~2x interval";

  coalescer.Stop();
}

// --- Integration tests via DBWal ---

class CoalescerIntegrationTest : public ::testing::Test {
 protected:
  void SetUp() override { env_ = Env::Default(); }

  WALOptions MakeOptions(size_t queue_depth = 10000,
                        uint64_t flush_interval_ms = 0) {
    WALOptions opts;
    opts.wal_dir = tmpdir_.path();
    opts.max_async_queue_depth = queue_depth;
    opts.max_async_flush_interval_ms = flush_interval_ms;
    return opts;
  }

  test::TempDir tmpdir_;
  Env* env_ = nullptr;
};

TEST_F(CoalescerIntegrationTest, AsyncWritesThroughCoalescer) {
  WALOptions opts = MakeOptions();
  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  constexpr int kWrites = 100;
  for (int i = 0; i < kWrites; i++) {
    WriteBatch batch;
    batch.Put("key" + std::to_string(i), "val" + std::to_string(i));
    WriteOptions wo;
    wo.sync = false;
    Status s = wal->Write(wo, &batch);
    ASSERT_TRUE(s.ok()) << s.ToString();
  }

  EXPECT_EQ(wal->GetLatestSequenceNumber(),
            static_cast<SequenceNumber>(kWrites));
  wal->Close();
}

TEST_F(CoalescerIntegrationTest, SyncWritesBypassCoalescer) {
  WALOptions opts = MakeOptions();
  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  constexpr int kWrites = 20;
  for (int i = 0; i < kWrites; i++) {
    WriteBatch batch;
    batch.Put("sync_key" + std::to_string(i), "val");
    WriteOptions wo;
    wo.sync = true;
    Status s = wal->Write(wo, &batch);
    ASSERT_TRUE(s.ok()) << s.ToString();
  }

  EXPECT_EQ(wal->GetLatestSequenceNumber(),
            static_cast<SequenceNumber>(kWrites));
  wal->Close();
}

TEST_F(CoalescerIntegrationTest, MixedSyncAsyncWrites) {
  WALOptions opts = MakeOptions();
  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  constexpr int kThreads = 8;
  constexpr int kWritesPerThread = 100;
  std::atomic<int> errors{0};
  std::vector<std::thread> threads;

  for (int t = 0; t < kThreads; t++) {
    threads.emplace_back([&, t] {
      for (int i = 0; i < kWritesPerThread; i++) {
        WriteBatch batch;
        batch.Put("t" + std::to_string(t) + "_k" + std::to_string(i), "v");
        WriteOptions wo;
        wo.sync = (t % 4 == 0);  // 25% sync
        Status s = wal->Write(wo, &batch);
        if (!s.ok()) errors.fetch_add(1);
      }
    });
  }

  for (auto& th : threads) th.join();
  wal->Close();

  EXPECT_EQ(errors.load(), 0);
}

TEST_F(CoalescerIntegrationTest, DisabledCoalescerUsesGroupCommit) {
  WALOptions opts = MakeOptions(0);  // disabled
  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  constexpr int kWrites = 50;
  for (int i = 0; i < kWrites; i++) {
    WriteBatch batch;
    batch.Put("key" + std::to_string(i), "val");
    Status s = wal->Write(WriteOptions(), &batch);
    ASSERT_TRUE(s.ok()) << s.ToString();
  }

  EXPECT_EQ(wal->GetLatestSequenceNumber(),
            static_cast<SequenceNumber>(kWrites));
  wal->Close();
}

TEST_F(CoalescerIntegrationTest, RecoveryAfterCoalescedWrites) {
  WALOptions opts = MakeOptions();
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    constexpr int kWrites = 50;
    for (int i = 0; i < kWrites; i++) {
      WriteBatch batch;
      batch.Put("rk" + std::to_string(i), "rv" + std::to_string(i));
      Status s = wal->Write(WriteOptions(), &batch);
      ASSERT_TRUE(s.ok()) << s.ToString();
    }
    wal->Close();
  }

  // Recover and verify all records
  int recovered = 0;
  WALOptions recover_opts;
  recover_opts.wal_dir = tmpdir_.path();
  recover_opts.max_async_queue_depth = 10000;
  recover_opts.recovery_callback = [&recovered](SequenceNumber, WriteBatch*) {
    recovered++;
    return Status::OK();
  };
  std::unique_ptr<DBWal> wal2;
  ASSERT_TRUE(DBWal::Open(recover_opts, env_, &wal2).ok());
  EXPECT_GT(recovered, 0);
  wal2->Close();
}

TEST_F(CoalescerIntegrationTest, IntegrationTimeBasedFlushRecovery) {
  WALOptions opts = MakeOptions(10000, 50);  // 50 ms flush interval
  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  constexpr int kWrites = 20;
  for (int i = 0; i < kWrites; i++) {
    WriteBatch batch;
    batch.Put("tk" + std::to_string(i), "tv" + std::to_string(i));
    WriteOptions wo;
    wo.sync = false;
    Status s = wal->Write(wo, &batch);
    ASSERT_TRUE(s.ok()) << s.ToString();
  }

  EXPECT_EQ(wal->GetLatestSequenceNumber(),
            static_cast<SequenceNumber>(kWrites));
  wal->Close();

  int recovered = 0;
  WALOptions recover_opts;
  recover_opts.wal_dir = tmpdir_.path();
  recover_opts.max_async_queue_depth = 10000;
  recover_opts.max_async_flush_interval_ms = 50;
  recover_opts.recovery_callback = [&recovered](SequenceNumber, WriteBatch*) {
    recovered++;
    return Status::OK();
  };
  std::unique_ptr<DBWal> wal2;
  ASSERT_TRUE(DBWal::Open(recover_opts, env_, &wal2).ok());
  EXPECT_EQ(recovered, kWrites);
  wal2->Close();
}

}  // namespace mwal
