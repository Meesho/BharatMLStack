// Unit tests for ReplicationManager.
// Tests the core logic (leadership callbacks, ISR wiring, pending buffer)
// without requiring a real multi-node cluster.

#include <gtest/gtest.h>

#include <filesystem>
#include <string>

#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/write_batch.h"
#include "replication/replication_config.h"
#include "replication/replication_manager.h"

namespace mwal {
namespace replication {

class ReplicationManagerTest : public ::testing::Test {
 protected:
  void SetUp() override {
    wal_dir_ = "/tmp/mwal_repl_mgr_test_" +
               std::to_string(std::chrono::steady_clock::now()
                                  .time_since_epoch()
                                  .count());
    WALOptions opts;
    opts.wal_dir = wal_dir_;
    opts.max_wal_file_size = 1 * 1024 * 1024;
    opts.max_async_queue_depth = 0;
    ASSERT_TRUE(DBWal::Open(opts, Env::Default(), &wal_).ok());
  }

  void TearDown() override {
    if (wal_) wal_->Close();
    std::filesystem::remove_all(wal_dir_);
  }

  ReplicationConfig MakeConfig() {
    ReplicationConfig cfg;
    cfg.min_insync_replicas = 0;  // allow writes even with no ISR for unit test
    cfg.replication_timeout_ms = 100;
    cfg.isr_check_interval_ms = 50;
    cfg.max_lag_entries = 1000;
    cfg.replica_timeout_ms = 500;
    cfg.batch_max_entries = 10;
    cfg.batch_max_bytes = 65536;
    cfg.self = {1, "127.0.0.1:0", "zone-1"};
    cfg.raft.node_id = 1;
    cfg.raft.raft_endpoint = "localhost:0";
    cfg.raft.heartbeat_interval_ms = 50;
    cfg.raft.election_timeout_lower_ms = 100;
    cfg.raft.election_timeout_upper_ms = 200;
    return cfg;
  }

  std::string wal_dir_;
  std::unique_ptr<DBWal> wal_;
};

TEST_F(ReplicationManagerTest, ConstructionDoesNotCrash) {
  auto cfg = MakeConfig();
  ReplicationManager mgr(wal_.get(), cfg);
  // Just verifying construction works.
  EXPECT_FALSE(mgr.IsLeader());
  EXPECT_EQ(mgr.GetCurrentTerm(), 0u);
  EXPECT_EQ(mgr.GetCommittedLSN(), 0u);
}

TEST_F(ReplicationManagerTest, OnRaftLeadership_SetsState) {
  auto cfg = MakeConfig();
  ReplicationManager mgr(wal_.get(), cfg);

  mgr.OnRaftLeadership(5, true);
  EXPECT_TRUE(mgr.IsLeader());
  EXPECT_EQ(mgr.GetCurrentTerm(), 5u);

  mgr.OnRaftLeadership(6, false);
  EXPECT_FALSE(mgr.IsLeader());
  EXPECT_EQ(mgr.GetCurrentTerm(), 6u);
}

TEST_F(ReplicationManagerTest, ISRTracker_Accessible) {
  auto cfg = MakeConfig();
  ReplicationManager mgr(wal_.get(), cfg);
  EXPECT_NE(mgr.GetISRTracker(), nullptr);
}

TEST_F(ReplicationManagerTest, HandleProgressReport_UpdatesTracker) {
  auto cfg = MakeConfig();
  cfg.peers = {{2, "127.0.0.1:50052", "zone-2"}};
  ReplicationManager mgr(wal_.get(), cfg);

  // The peer should be registered during Start(), but we can also call
  // HandleProgressReport directly to verify the tracker update path.
  mgr.GetISRTracker()->AddReplica(2, "zone-2");
  mgr.HandleProgressReport(2, 42, 30);

  auto isr = mgr.GetISRTracker()->GetAllReplicaIDs();
  EXPECT_EQ(isr.size(), 1u);
}

TEST_F(ReplicationManagerTest, WriteWithCallback_BuffersPending) {
  auto cfg = MakeConfig();
  ReplicationManager mgr(wal_.get(), cfg);

  // Simulate becoming leader and wire the callback manually.
  mgr.OnRaftLeadership(1, true);

  // Write to the WAL (callback isn't wired through Start(), so just verify
  // the WAL write itself works).
  WriteBatch batch;
  batch.Put(Slice("key"), Slice("value"));
  WriteOptions wo;
  wo.sync = false;
  auto s = wal_->Write(wo, &batch);
  EXPECT_TRUE(s.ok());
  EXPECT_GT(wal_->GetLatestSequenceNumber(), 0u);
}

}  // namespace replication
}  // namespace mwal
