// Tests for the three new mwal replication APIs:
//   - AppendReplicated(first_seq, count, payload, term)
//   - WriteRecordCallback (in WALOptions)
//   - TruncateAfter(lsn)
//
// These tests are written against the API signatures specified in
// docs/REPLICATION_DESIGN.md. They will not compile until the APIs are
// implemented — the test file acts as the executable specification.

#include <gtest/gtest.h>

#include <atomic>
#include <chrono>
#include <mutex>
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
#include "wal/wal_compressor.h"

namespace mwal {

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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

// Builds a valid WAL payload (compression prefix + WriteBatch) suitable for
// AppendReplicated. Returns the compressed string whose format matches what
// Write() produces internally.
static std::string MakePayload(SequenceNumber first_seq,
                               const std::string& key,
                               const std::string& value) {
  WriteBatch batch;
  batch.Put(key, value);
  batch.SetSequence(first_seq);
  std::string compressed;
  WalCompressor::Compress(kNoCompression, Slice(batch.Data()), &compressed);
  return compressed;
}

// Overload: multi-put batch.
static std::string MakePayloadMulti(
    SequenceNumber first_seq,
    const std::vector<std::pair<std::string, std::string>>& kvs) {
  WriteBatch batch;
  for (const auto& kv : kvs) {
    batch.Put(kv.first, kv.second);
  }
  batch.SetSequence(first_seq);
  std::string compressed;
  WalCompressor::Compress(kNoCompression, Slice(batch.Data()), &compressed);
  return compressed;
}

static std::vector<RecoveredEntry> RecoverAll(DBWal* wal) {
  RecoveryCollector collector;
  Status s = wal->Recover([&](SequenceNumber seq, WriteBatch* batch) {
    collector.current_seq = seq;
    return batch->Iterate(&collector);
  });
  EXPECT_TRUE(s.ok()) << s.ToString();
  return collector.entries;
}

// Stores callback invocations from WriteRecordCallback.
struct CallbackRecord {
  SequenceNumber first_seq;
  uint32_t count;
  std::string payload;
};

// ---------------------------------------------------------------------------
// Test fixture
// ---------------------------------------------------------------------------

class ReplicationApiTest : public ::testing::Test {
 protected:
  void SetUp() override { env_ = Env::Default(); }

  WALOptions LeaderOptions() {
    WALOptions opts;
    opts.wal_dir = leader_dir_.path();
    return opts;
  }

  WALOptions ReplicaOptions() {
    WALOptions opts;
    opts.wal_dir = replica_dir_.path();
    opts.max_async_queue_depth = 0;  // replicas don't need coalescer
    return opts;
  }

  test::TempDir leader_dir_;
  test::TempDir replica_dir_;
  Env* env_ = nullptr;
};

// ===========================================================================
// Group 1: WriteRecordCallback
// ===========================================================================

TEST_F(ReplicationApiTest, CallbackFiresOnWrite) {
  auto opts = LeaderOptions();
  std::vector<CallbackRecord> records;
  std::mutex mu;

  opts.write_record_callback =
      [&](SequenceNumber first_seq, uint32_t count, const Slice& payload) {
        std::lock_guard<std::mutex> lock(mu);
        records.push_back(
            {first_seq, count, std::string(payload.data(), payload.size())});
      };

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  WriteBatch batch;
  batch.Put("hello", "world");
  ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());

  wal->Close();

  ASSERT_EQ(records.size(), 1u);
  EXPECT_EQ(records[0].first_seq, 1u);
  EXPECT_EQ(records[0].count, 1u);
  EXPECT_FALSE(records[0].payload.empty());
}

TEST_F(ReplicationApiTest, CallbackSequenceMatchesBatch) {
  auto opts = LeaderOptions();
  std::vector<CallbackRecord> records;
  std::mutex mu;

  opts.write_record_callback =
      [&](SequenceNumber first_seq, uint32_t count, const Slice& payload) {
        std::lock_guard<std::mutex> lock(mu);
        records.push_back(
            {first_seq, count, std::string(payload.data(), payload.size())});
      };

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  WriteBatch batch;
  batch.Put("a", "1");
  batch.Put("b", "2");
  batch.Put("c", "3");
  ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());

  wal->Close();

  ASSERT_EQ(records.size(), 1u);
  EXPECT_EQ(records[0].first_seq, 1u);
  EXPECT_EQ(records[0].count, 3u);
}

TEST_F(ReplicationApiTest, CallbackFiresPerGroupCommit) {
  auto opts = LeaderOptions();
  opts.max_write_group_size = 4;
  opts.max_async_queue_depth = 0;  // force group commit path

  std::atomic<uint32_t> total_count{0};
  std::atomic<int> callback_invocations{0};
  std::mutex mu;

  opts.write_record_callback =
      [&](SequenceNumber, uint32_t count, const Slice&) {
        callback_invocations.fetch_add(1);
        total_count.fetch_add(count);
      };

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  constexpr int kThreads = 4;
  constexpr int kWritesPerThread = 10;
  std::vector<std::thread> threads;
  for (int t = 0; t < kThreads; t++) {
    threads.emplace_back([&, t] {
      for (int i = 0; i < kWritesPerThread; i++) {
        WriteBatch batch;
        batch.Put("t" + std::to_string(t) + "_" + std::to_string(i), "v");
        wal->Write(WriteOptions(), &batch);
      }
    });
  }
  for (auto& t : threads) t.join();
  wal->Close();

  EXPECT_EQ(total_count.load(), static_cast<uint32_t>(kThreads * kWritesPerThread));
  // Group commit merges batches, so callback fires fewer times than total writes
  EXPECT_LE(callback_invocations.load(), kThreads * kWritesPerThread);
  EXPECT_GE(callback_invocations.load(), 1);
}

#ifdef MWAL_HAVE_ZSTD
TEST_F(ReplicationApiTest, CallbackPayloadIsCompressedRecord) {
  auto opts = LeaderOptions();
  opts.wal_compression = kZSTD;

  std::vector<CallbackRecord> records;
  std::mutex mu;
  opts.write_record_callback =
      [&](SequenceNumber first_seq, uint32_t count, const Slice& payload) {
        std::lock_guard<std::mutex> lock(mu);
        records.push_back(
            {first_seq, count, std::string(payload.data(), payload.size())});
      };

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  WriteBatch batch;
  batch.Put("compressed_key", std::string(256, 'X'));
  ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
  wal->Close();

  ASSERT_EQ(records.size(), 1u);
  // 0x07 = zstd prefix
  EXPECT_EQ(static_cast<uint8_t>(records[0].payload[0]), 0x07);

  // Decompress and verify it's a valid WriteBatch
  std::string decompressed;
  Slice result;
  Status s = WalDecompressor::Decompress(Slice(records[0].payload),
                                          &decompressed, &result);
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_GT(result.size(), 12u);  // at least WriteBatch header
}
#endif

TEST_F(ReplicationApiTest, CallbackPayloadIsValidForAppendReplicated) {
  auto leader_opts = LeaderOptions();
  std::vector<CallbackRecord> records;
  std::mutex mu;

  leader_opts.write_record_callback =
      [&](SequenceNumber first_seq, uint32_t count, const Slice& payload) {
        std::lock_guard<std::mutex> lock(mu);
        records.push_back(
            {first_seq, count, std::string(payload.data(), payload.size())});
      };

  std::unique_ptr<DBWal> leader;
  ASSERT_TRUE(DBWal::Open(leader_opts, env_, &leader).ok());

  WriteBatch batch;
  batch.Put("replicated_key", "replicated_value");
  ASSERT_TRUE(leader->Write(WriteOptions(), &batch).ok());
  leader->Close();

  ASSERT_EQ(records.size(), 1u);

  // Feed callback output to replica
  auto replica_opts = ReplicaOptions();
  std::unique_ptr<DBWal> replica;
  ASSERT_TRUE(DBWal::Open(replica_opts, env_, &replica).ok());

  Status s = replica->AppendReplicated(
      records[0].first_seq, records[0].count,
      Slice(records[0].payload));
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(replica->GetLatestSequenceNumber(), 1u);
  replica->Close();

  // Recover replica and verify
  ASSERT_TRUE(DBWal::Open(replica_opts, env_, &replica).ok());
  auto entries = RecoverAll(replica.get());
  ASSERT_EQ(entries.size(), 1u);
  EXPECT_EQ(entries[0].key, "replicated_key");
  EXPECT_EQ(entries[0].value, "replicated_value");
  replica->Close();
}

TEST_F(ReplicationApiTest, NoCallbackByDefault) {
  auto opts = LeaderOptions();
  // write_record_callback is not set

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  WriteBatch batch;
  batch.Put("key", "value");
  Status s = wal->Write(WriteOptions(), &batch);
  EXPECT_TRUE(s.ok()) << s.ToString();
  wal->Close();
}

TEST_F(ReplicationApiTest, CallbackOnCoalescedWrite) {
  auto opts = LeaderOptions();
  opts.max_async_queue_depth = 100;
  opts.max_async_flush_interval_ms = 50;

  std::atomic<uint32_t> total_count{0};
  std::atomic<int> callback_invocations{0};

  opts.write_record_callback =
      [&](SequenceNumber, uint32_t count, const Slice&) {
        callback_invocations.fetch_add(1);
        total_count.fetch_add(count);
      };

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  WriteOptions wo;
  wo.sync = false;  // async path → coalescer
  for (int i = 0; i < 20; i++) {
    WriteBatch batch;
    batch.Put("async_" + std::to_string(i), "val");
    wal->Write(wo, &batch);
  }

  // Give the coalescer time to drain
  std::this_thread::sleep_for(std::chrono::milliseconds(200));
  wal->Close();

  EXPECT_EQ(total_count.load(), 20u);
  EXPECT_GE(callback_invocations.load(), 1);
}

// ===========================================================================
// Group 2: AppendReplicated
// ===========================================================================

TEST_F(ReplicationApiTest, BasicAppendReplicated) {
  auto opts = ReplicaOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  std::string payload = MakePayload(1, "key1", "val1");
  Status s = wal->AppendReplicated(1, 1, Slice(payload));
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(wal->GetLatestSequenceNumber(), 1u);
  wal->Close();

  // Recover and verify
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  auto entries = RecoverAll(wal.get());
  ASSERT_EQ(entries.size(), 1u);
  EXPECT_EQ(entries[0].key, "key1");
  EXPECT_EQ(entries[0].value, "val1");
  wal->Close();
}

TEST_F(ReplicationApiTest, AppendReplicatedMultipleRecords) {
  auto opts = ReplicaOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  for (int i = 1; i <= 10; i++) {
    std::string payload =
        MakePayload(i, "key_" + std::to_string(i), "val_" + std::to_string(i));
    ASSERT_TRUE(
        wal->AppendReplicated(i, 1, Slice(payload)).ok());
  }
  EXPECT_EQ(wal->GetLatestSequenceNumber(), 10u);
  wal->Close();

  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  auto entries = RecoverAll(wal.get());
  ASSERT_EQ(entries.size(), 10u);
  for (int i = 0; i < 10; i++) {
    EXPECT_EQ(entries[i].key, "key_" + std::to_string(i + 1));
  }
  wal->Close();
}

TEST_F(ReplicationApiTest, AppendReplicatedUpdatesSequence) {
  auto opts = ReplicaOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  // Batch with 3 operations starting at seq 5 → last_sequence_ = 5+3-1 = 7
  std::string payload =
      MakePayloadMulti(5, {{"a", "1"}, {"b", "2"}, {"c", "3"}});
  ASSERT_TRUE(wal->AppendReplicated(5, 3, Slice(payload)).ok());
  EXPECT_EQ(wal->GetLatestSequenceNumber(), 7u);
  wal->Close();
}

TEST_F(ReplicationApiTest, AppendReplicatedWithTerm) {
  auto opts = ReplicaOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  std::string payload = MakePayload(1, "term_key", "term_val");
  Status s = wal->AppendReplicated(1, 1, Slice(payload), /*term=*/42);
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(wal->GetLatestSequenceNumber(), 1u);
  wal->Close();

  // Term does not affect WAL content; recovery should work fine
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  auto entries = RecoverAll(wal.get());
  ASSERT_EQ(entries.size(), 1u);
  EXPECT_EQ(entries[0].key, "term_key");
  wal->Close();
}

TEST_F(ReplicationApiTest, AppendReplicatedZeroTerm) {
  auto opts = ReplicaOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  std::string payload = MakePayload(1, "zero_term", "val");
  Status s = wal->AppendReplicated(1, 1, Slice(payload), /*term=*/0);
  ASSERT_TRUE(s.ok()) << s.ToString();
  wal->Close();

  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  auto entries = RecoverAll(wal.get());
  ASSERT_EQ(entries.size(), 1u);
  EXPECT_EQ(entries[0].key, "zero_term");
  wal->Close();
}

TEST_F(ReplicationApiTest, AppendReplicatedTriggersRotation) {
  auto opts = ReplicaOptions();
  opts.max_wal_file_size = 512;

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  std::string big_value(200, 'R');
  for (int i = 1; i <= 20; i++) {
    std::string payload =
        MakePayload(i, "key_" + std::to_string(i), big_value);
    ASSERT_TRUE(wal->AppendReplicated(i, 1, Slice(payload)).ok());
  }

  EXPECT_GT(wal->GetCurrentLogNumber(), 1u);
  EXPECT_EQ(wal->GetLatestSequenceNumber(), 20u);
  wal->Close();

  // Verify all records survive rotation + recovery
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  auto entries = RecoverAll(wal.get());
  EXPECT_EQ(entries.size(), 20u);
  wal->Close();
}

TEST_F(ReplicationApiTest, AppendReplicatedAfterClose) {
  auto opts = ReplicaOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  wal->Close();

  std::string payload = MakePayload(1, "k", "v");
  Status s = wal->AppendReplicated(1, 1, Slice(payload));
  EXPECT_TRUE(s.IsAborted()) << s.ToString();
}

TEST_F(ReplicationApiTest, AppendReplicatedRecoverMatchesLeader) {
  // Leader writes 20 records with callback
  auto leader_opts = LeaderOptions();
  std::vector<CallbackRecord> records;
  std::mutex mu;

  leader_opts.write_record_callback =
      [&](SequenceNumber first_seq, uint32_t count, const Slice& payload) {
        std::lock_guard<std::mutex> lock(mu);
        records.push_back(
            {first_seq, count, std::string(payload.data(), payload.size())});
      };

  std::unique_ptr<DBWal> leader;
  ASSERT_TRUE(DBWal::Open(leader_opts, env_, &leader).ok());

  for (int i = 0; i < 20; i++) {
    WriteBatch batch;
    batch.Put("key_" + std::to_string(i), "val_" + std::to_string(i));
    ASSERT_TRUE(leader->Write(WriteOptions(), &batch).ok());
  }
  leader->Close();

  // Replica receives all via AppendReplicated
  auto replica_opts = ReplicaOptions();
  std::unique_ptr<DBWal> replica;
  ASSERT_TRUE(DBWal::Open(replica_opts, env_, &replica).ok());

  for (const auto& rec : records) {
    ASSERT_TRUE(replica->AppendReplicated(
        rec.first_seq, rec.count, Slice(rec.payload)).ok());
  }
  replica->Close();

  // Recover both and compare
  ASSERT_TRUE(DBWal::Open(leader_opts, env_, &leader).ok());
  // Re-open without callback to avoid interference
  auto leader_entries = RecoverAll(leader.get());
  leader->Close();

  ASSERT_TRUE(DBWal::Open(replica_opts, env_, &replica).ok());
  auto replica_entries = RecoverAll(replica.get());
  replica->Close();

  ASSERT_EQ(leader_entries.size(), replica_entries.size());
  for (size_t i = 0; i < leader_entries.size(); i++) {
    EXPECT_EQ(leader_entries[i].key, replica_entries[i].key);
    EXPECT_EQ(leader_entries[i].value, replica_entries[i].value);
    EXPECT_EQ(leader_entries[i].seq, replica_entries[i].seq);
  }
}

TEST_F(ReplicationApiTest, AppendReplicatedIteratorWorks) {
  auto opts = ReplicaOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  for (int i = 1; i <= 10; i++) {
    std::string payload =
        MakePayload(i, "key_" + std::to_string(i), "val_" + std::to_string(i));
    ASSERT_TRUE(wal->AppendReplicated(i, 1, Slice(payload)).ok());
  }

  // Iterator from seq 5 should return records 5..10
  std::unique_ptr<WalIterator> iter;
  ASSERT_TRUE(wal->NewWalIterator(5, &iter).ok());

  int count = 0;
  while (iter->Valid()) {
    EXPECT_GE(iter->GetSequenceNumber(), 5u);
    count++;
    iter->Next();
  }
  EXPECT_TRUE(iter->status().ok()) << iter->status().ToString();
  EXPECT_EQ(count, 6);  // seq 5,6,7,8,9,10
  wal->Close();
}

TEST_F(ReplicationApiTest, AppendReplicatedBypassesWriteThread) {
  auto opts = ReplicaOptions();
  opts.max_async_queue_depth = 0;  // coalescer disabled

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  constexpr int kThreads = 4;
  constexpr int kPerThread = 25;
  std::atomic<int> errors{0};
  std::atomic<SequenceNumber> next_seq{1};

  std::vector<std::thread> threads;
  for (int t = 0; t < kThreads; t++) {
    threads.emplace_back([&] {
      for (int i = 0; i < kPerThread; i++) {
        SequenceNumber seq = next_seq.fetch_add(1);
        std::string payload =
            MakePayload(seq, "k_" + std::to_string(seq), "v");
        Status s = wal->AppendReplicated(seq, 1, Slice(payload));
        if (!s.ok()) errors.fetch_add(1);
      }
    });
  }
  for (auto& t : threads) t.join();

  EXPECT_EQ(errors.load(), 0);
  EXPECT_EQ(wal->GetLatestSequenceNumber(),
            static_cast<SequenceNumber>(kThreads * kPerThread));
  wal->Close();

  // Verify recovery
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  auto entries = RecoverAll(wal.get());
  EXPECT_EQ(static_cast<int>(entries.size()), kThreads * kPerThread);
  wal->Close();
}

// ===========================================================================
// Group 3: TruncateAfter
// ===========================================================================

TEST_F(ReplicationApiTest, TruncateAfterBasic) {
  auto opts = ReplicaOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  for (int i = 1; i <= 10; i++) {
    std::string payload =
        MakePayload(i, "key_" + std::to_string(i), "val_" + std::to_string(i));
    ASSERT_TRUE(wal->AppendReplicated(i, 1, Slice(payload)).ok());
  }
  ASSERT_EQ(wal->GetLatestSequenceNumber(), 10u);

  Status s = wal->TruncateAfter(5);
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(wal->GetLatestSequenceNumber(), 5u);
  wal->Close();

  // Recover and verify only records 1..5
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  auto entries = RecoverAll(wal.get());
  ASSERT_EQ(entries.size(), 5u);
  for (int i = 0; i < 5; i++) {
    EXPECT_EQ(entries[i].key, "key_" + std::to_string(i + 1));
  }
  wal->Close();
}

TEST_F(ReplicationApiTest, TruncateAfterAll) {
  auto opts = ReplicaOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  for (int i = 1; i <= 5; i++) {
    std::string payload = MakePayload(i, "key_" + std::to_string(i), "val");
    ASSERT_TRUE(wal->AppendReplicated(i, 1, Slice(payload)).ok());
  }

  Status s = wal->TruncateAfter(0);
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(wal->GetLatestSequenceNumber(), 0u);
  wal->Close();

  // Recover: zero records
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  auto entries = RecoverAll(wal.get());
  EXPECT_EQ(entries.size(), 0u);
  wal->Close();
}

TEST_F(ReplicationApiTest, TruncateAfterNoop) {
  auto opts = ReplicaOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  for (int i = 1; i <= 5; i++) {
    std::string payload = MakePayload(i, "key_" + std::to_string(i), "val");
    ASSERT_TRUE(wal->AppendReplicated(i, 1, Slice(payload)).ok());
  }

  // Truncate at current end — should be a no-op
  Status s = wal->TruncateAfter(5);
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(wal->GetLatestSequenceNumber(), 5u);
  wal->Close();

  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  auto entries = RecoverAll(wal.get());
  EXPECT_EQ(entries.size(), 5u);
  wal->Close();
}

TEST_F(ReplicationApiTest, TruncateAfterBeyondEnd) {
  auto opts = ReplicaOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  for (int i = 1; i <= 5; i++) {
    std::string payload = MakePayload(i, "key_" + std::to_string(i), "val");
    ASSERT_TRUE(wal->AppendReplicated(i, 1, Slice(payload)).ok());
  }

  // Truncate at 100 — nothing to truncate
  Status s = wal->TruncateAfter(100);
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(wal->GetLatestSequenceNumber(), 5u);
  wal->Close();

  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  auto entries = RecoverAll(wal.get());
  EXPECT_EQ(entries.size(), 5u);
  wal->Close();
}

TEST_F(ReplicationApiTest, TruncateAfterThenAppend) {
  auto opts = ReplicaOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  // Append 1..10
  for (int i = 1; i <= 10; i++) {
    std::string payload =
        MakePayload(i, "old_" + std::to_string(i), "val");
    ASSERT_TRUE(wal->AppendReplicated(i, 1, Slice(payload)).ok());
  }

  // Truncate to 5
  ASSERT_TRUE(wal->TruncateAfter(5).ok());
  EXPECT_EQ(wal->GetLatestSequenceNumber(), 5u);

  // Append new records 6..8 (from new leader)
  for (int i = 6; i <= 8; i++) {
    std::string payload =
        MakePayload(i, "new_" + std::to_string(i), "val");
    ASSERT_TRUE(wal->AppendReplicated(i, 1, Slice(payload)).ok());
  }
  EXPECT_EQ(wal->GetLatestSequenceNumber(), 8u);
  wal->Close();

  // Recover: 1..5 (old) + 6..8 (new)
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  auto entries = RecoverAll(wal.get());
  ASSERT_EQ(entries.size(), 8u);
  for (int i = 0; i < 5; i++) {
    EXPECT_EQ(entries[i].key, "old_" + std::to_string(i + 1));
  }
  for (int i = 5; i < 8; i++) {
    EXPECT_EQ(entries[i].key, "new_" + std::to_string(i + 1));
  }
  wal->Close();
}

TEST_F(ReplicationApiTest, TruncateAfterMultipleFiles) {
  auto opts = ReplicaOptions();
  opts.max_wal_file_size = 512;

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  // Append enough records to span 3+ files
  std::string big_value(200, 'T');
  for (int i = 1; i <= 30; i++) {
    std::string payload =
        MakePayload(i, "key_" + std::to_string(i), big_value);
    ASSERT_TRUE(wal->AppendReplicated(i, 1, Slice(payload)).ok());
  }
  EXPECT_GT(wal->GetCurrentLogNumber(), 2u);

  // Truncate to record 10 (should be in an early file)
  ASSERT_TRUE(wal->TruncateAfter(10).ok());
  EXPECT_EQ(wal->GetLatestSequenceNumber(), 10u);
  wal->Close();

  // Recover: only records 1..10
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  auto entries = RecoverAll(wal.get());
  ASSERT_EQ(entries.size(), 10u);
  for (int i = 0; i < 10; i++) {
    EXPECT_EQ(entries[i].key, "key_" + std::to_string(i + 1));
  }
  wal->Close();
}

TEST_F(ReplicationApiTest, TruncateAfterRecoverySequence) {
  // Simulate failover:
  // 1. Leader writes 1..10 with callback
  // 2. Replica gets all 10 via AppendReplicated
  // 3. Replica truncates to 7 (new leader is at seq 7)
  // 4. Replica appends 8..9 (new leader's data)
  // 5. Recover replica: should see 1..7 (old) + 8..9 (new)

  auto leader_opts = LeaderOptions();
  std::vector<CallbackRecord> records;
  std::mutex mu;

  leader_opts.write_record_callback =
      [&](SequenceNumber first_seq, uint32_t count, const Slice& payload) {
        std::lock_guard<std::mutex> lock(mu);
        records.push_back(
            {first_seq, count, std::string(payload.data(), payload.size())});
      };

  std::unique_ptr<DBWal> leader;
  ASSERT_TRUE(DBWal::Open(leader_opts, env_, &leader).ok());
  for (int i = 0; i < 10; i++) {
    WriteBatch batch;
    batch.Put("old_" + std::to_string(i + 1), "val");
    ASSERT_TRUE(leader->Write(WriteOptions(), &batch).ok());
  }
  leader->Close();

  // Replica receives all 10
  auto replica_opts = ReplicaOptions();
  std::unique_ptr<DBWal> replica;
  ASSERT_TRUE(DBWal::Open(replica_opts, env_, &replica).ok());
  for (const auto& rec : records) {
    ASSERT_TRUE(replica->AppendReplicated(
        rec.first_seq, rec.count, Slice(rec.payload)).ok());
  }
  EXPECT_EQ(replica->GetLatestSequenceNumber(), 10u);

  // Simulate new leader at seq 7: truncate divergent tail
  ASSERT_TRUE(replica->TruncateAfter(7).ok());
  EXPECT_EQ(replica->GetLatestSequenceNumber(), 7u);

  // New leader's data for seq 8..9
  for (int i = 8; i <= 9; i++) {
    std::string payload = MakePayload(i, "new_" + std::to_string(i), "val");
    ASSERT_TRUE(replica->AppendReplicated(i, 1, Slice(payload)).ok());
  }
  EXPECT_EQ(replica->GetLatestSequenceNumber(), 9u);
  replica->Close();

  // Recover: 1..7 old + 8..9 new
  ASSERT_TRUE(DBWal::Open(replica_opts, env_, &replica).ok());
  auto entries = RecoverAll(replica.get());
  ASSERT_EQ(entries.size(), 9u);
  for (int i = 0; i < 7; i++) {
    EXPECT_EQ(entries[i].key, "old_" + std::to_string(i + 1));
  }
  EXPECT_EQ(entries[7].key, "new_8");
  EXPECT_EQ(entries[8].key, "new_9");
  replica->Close();
}

TEST_F(ReplicationApiTest, TruncateAfterOnClosedWal) {
  auto opts = ReplicaOptions();

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  wal->Close();

  Status s = wal->TruncateAfter(5);
  EXPECT_TRUE(s.IsAborted()) << s.ToString();
}

// ===========================================================================
// Group 4: Integration / end-to-end
// ===========================================================================

TEST_F(ReplicationApiTest, LeaderToReplicaFullRoundtrip) {
  auto leader_opts = LeaderOptions();
  std::vector<CallbackRecord> records;
  std::mutex mu;

  leader_opts.write_record_callback =
      [&](SequenceNumber first_seq, uint32_t count, const Slice& payload) {
        std::lock_guard<std::mutex> lock(mu);
        records.push_back(
            {first_seq, count, std::string(payload.data(), payload.size())});
      };

  // Leader writes 50 records
  std::unique_ptr<DBWal> leader;
  ASSERT_TRUE(DBWal::Open(leader_opts, env_, &leader).ok());
  for (int i = 0; i < 50; i++) {
    WriteBatch batch;
    batch.Put("key_" + std::to_string(i), "val_" + std::to_string(i));
    ASSERT_TRUE(leader->Write(WriteOptions(), &batch).ok());
  }
  leader->Close();

  // Replica receives all via AppendReplicated
  auto replica_opts = ReplicaOptions();
  std::unique_ptr<DBWal> replica;
  ASSERT_TRUE(DBWal::Open(replica_opts, env_, &replica).ok());
  for (const auto& rec : records) {
    ASSERT_TRUE(replica->AppendReplicated(
        rec.first_seq, rec.count, Slice(rec.payload)).ok());
  }
  replica->Close();

  // Recover both and compare
  ASSERT_TRUE(DBWal::Open(leader_opts, env_, &leader).ok());
  auto leader_entries = RecoverAll(leader.get());
  leader->Close();

  ASSERT_TRUE(DBWal::Open(replica_opts, env_, &replica).ok());
  auto replica_entries = RecoverAll(replica.get());

  ASSERT_EQ(leader_entries.size(), replica_entries.size());
  EXPECT_EQ(leader_entries.size(), 50u);
  for (size_t i = 0; i < leader_entries.size(); i++) {
    EXPECT_EQ(leader_entries[i].key, replica_entries[i].key);
    EXPECT_EQ(leader_entries[i].value, replica_entries[i].value);
  }

  // Iterator on replica from seq 25
  std::unique_ptr<WalIterator> iter;
  ASSERT_TRUE(replica->NewWalIterator(25, &iter).ok());
  int count = 0;
  while (iter->Valid()) {
    EXPECT_GE(iter->GetSequenceNumber(), 25u);
    count++;
    iter->Next();
  }
  EXPECT_TRUE(iter->status().ok());
  EXPECT_EQ(count, 26);  // seq 25..50
  replica->Close();
}

TEST_F(ReplicationApiTest, SimulatedFailover) {
  // Leader A writes 1..20
  auto leader_opts = LeaderOptions();
  std::vector<CallbackRecord> records;
  std::mutex mu;

  leader_opts.write_record_callback =
      [&](SequenceNumber first_seq, uint32_t count, const Slice& payload) {
        std::lock_guard<std::mutex> lock(mu);
        records.push_back(
            {first_seq, count, std::string(payload.data(), payload.size())});
      };

  std::unique_ptr<DBWal> leader;
  ASSERT_TRUE(DBWal::Open(leader_opts, env_, &leader).ok());
  for (int i = 0; i < 20; i++) {
    WriteBatch batch;
    batch.Put("leader_" + std::to_string(i + 1), "val");
    ASSERT_TRUE(leader->Write(WriteOptions(), &batch).ok());
  }
  leader->Close();

  // Replica B gets only 1..15 (partial replication)
  auto replica_opts = ReplicaOptions();
  std::unique_ptr<DBWal> replica;
  ASSERT_TRUE(DBWal::Open(replica_opts, env_, &replica).ok());

  SequenceNumber replicated_up_to = 0;
  for (const auto& rec : records) {
    if (replicated_up_to + rec.count > 15) break;
    ASSERT_TRUE(replica->AppendReplicated(
        rec.first_seq, rec.count, Slice(rec.payload)).ok());
    replicated_up_to = rec.first_seq + rec.count - 1;
  }
  EXPECT_LE(replica->GetLatestSequenceNumber(), 15u);

  // B truncates to its actual end (simulating new leader boundary)
  SequenceNumber b_end = replica->GetLatestSequenceNumber();
  ASSERT_TRUE(replica->TruncateAfter(b_end).ok());

  // B (now leader) writes 16..18 directly via Write()
  // But since B is using AppendReplicated path, simulate with Write()
  // by closing and reopening with normal options for writing
  replica->Close();

  // Reopen replica dir as a "new leader" that accepts writes
  WALOptions new_leader_opts;
  new_leader_opts.wal_dir = replica_dir_.path();
  new_leader_opts.recovery_callback = [](SequenceNumber, WriteBatch*) {
    return Status::OK();
  };

  std::unique_ptr<DBWal> new_leader;
  ASSERT_TRUE(DBWal::Open(new_leader_opts, env_, &new_leader).ok());

  for (int i = 0; i < 3; i++) {
    WriteBatch batch;
    batch.Put("new_leader_" + std::to_string(b_end + 1 + i), "val");
    ASSERT_TRUE(new_leader->Write(WriteOptions(), &batch).ok());
  }
  SequenceNumber final_seq = new_leader->GetLatestSequenceNumber();
  EXPECT_EQ(final_seq, b_end + 3);
  new_leader->Close();

  // Recover: should see b_end original records + 3 new ones
  ASSERT_TRUE(DBWal::Open(new_leader_opts, env_, &new_leader).ok());
  auto entries = RecoverAll(new_leader.get());
  EXPECT_EQ(entries.size(), static_cast<size_t>(b_end + 3));
  new_leader->Close();
}

TEST_F(ReplicationApiTest, CatchUpViaIterator) {
  // Leader writes 1..100
  auto leader_opts = LeaderOptions();
  std::vector<CallbackRecord> records;
  std::mutex mu;

  leader_opts.write_record_callback =
      [&](SequenceNumber first_seq, uint32_t count, const Slice& payload) {
        std::lock_guard<std::mutex> lock(mu);
        records.push_back(
            {first_seq, count, std::string(payload.data(), payload.size())});
      };

  std::unique_ptr<DBWal> leader;
  ASSERT_TRUE(DBWal::Open(leader_opts, env_, &leader).ok());
  for (int i = 0; i < 100; i++) {
    WriteBatch batch;
    batch.Put("key_" + std::to_string(i + 1), "val_" + std::to_string(i + 1));
    ASSERT_TRUE(leader->Write(WriteOptions(), &batch).ok());
  }

  // Replica gets first 50 via callback records
  auto replica_opts = ReplicaOptions();
  std::unique_ptr<DBWal> replica;
  ASSERT_TRUE(DBWal::Open(replica_opts, env_, &replica).ok());

  SequenceNumber replicated_up_to = 0;
  for (const auto& rec : records) {
    if (replicated_up_to + rec.count > 50) break;
    ASSERT_TRUE(replica->AppendReplicated(
        rec.first_seq, rec.count, Slice(rec.payload)).ok());
    replicated_up_to = rec.first_seq + rec.count - 1;
  }
  SequenceNumber replica_seq = replica->GetLatestSequenceNumber();
  EXPECT_GE(replica_seq, 1u);
  EXPECT_LE(replica_seq, 50u);

  // Catch up: use WalIterator on leader from replica_seq+1
  std::unique_ptr<WalIterator> iter;
  ASSERT_TRUE(leader->NewWalIterator(replica_seq + 1, &iter).ok());

  while (iter->Valid()) {
    const WriteBatch& batch = iter->GetBatch();
    SequenceNumber seq = iter->GetSequenceNumber();
    uint32_t count = static_cast<uint32_t>(batch.Count());

    // Build payload from iterator batch (compress it like the leader would)
    std::string compressed;
    WalCompressor::Compress(kNoCompression, Slice(batch.Data()), &compressed);
    ASSERT_TRUE(
        replica->AppendReplicated(seq, count, Slice(compressed)).ok());
    iter->Next();
  }
  EXPECT_TRUE(iter->status().ok()) << iter->status().ToString();

  // Replica should now have all 100 records
  EXPECT_EQ(replica->GetLatestSequenceNumber(), 100u);
  leader->Close();
  replica->Close();

  // Verify via recovery
  ASSERT_TRUE(DBWal::Open(replica_opts, env_, &replica).ok());
  auto entries = RecoverAll(replica.get());
  EXPECT_EQ(entries.size(), 100u);
  for (int i = 0; i < 100; i++) {
    EXPECT_EQ(entries[i].key, "key_" + std::to_string(i + 1));
    EXPECT_EQ(entries[i].value, "val_" + std::to_string(i + 1));
  }
  replica->Close();
}

}  // namespace mwal
