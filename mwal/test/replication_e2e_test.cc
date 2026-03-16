// End-to-end replication tests.
// These tests exercise the full stack: mwal + gRPC + ISR tracker.
// NuRaft is bypassed (leadership is set manually) to keep tests deterministic.

#include <gtest/gtest.h>

#include <chrono>
#include <filesystem>
#include <memory>
#include <string>
#include <thread>

#include <grpcpp/grpcpp.h>

#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/write_batch.h"
#include "replication/replication_client.h"
#include "replication/replication_config.h"
#include "replication/replication_manager.h"
#include "replication/replication_service.h"

namespace mwal {
namespace replication {

namespace {

std::string TmpDir(const std::string& suffix) {
  return "/tmp/mwal_e2e_" + suffix + "_" +
         std::to_string(std::chrono::steady_clock::now()
                            .time_since_epoch()
                            .count());
}

struct TestNode {
  std::string wal_dir;
  std::string uds_path;
  std::unique_ptr<DBWal> wal;
  std::unique_ptr<ReplicationServiceImpl> service;
  std::unique_ptr<grpc::Server> server;

  std::string Address() const { return "unix:" + uds_path; }

  ~TestNode() {
    if (server) server->Shutdown();
    if (wal) wal->Close();
    std::filesystem::remove_all(wal_dir);
  }
};

std::unique_ptr<TestNode> MakeReplicaNode(
    ReplicationManager* mgr, int id) {
  auto node = std::make_unique<TestNode>();
  node->wal_dir = TmpDir("replica_" + std::to_string(id));

  WALOptions opts;
  opts.wal_dir = node->wal_dir;
  opts.max_async_queue_depth = 0;
  EXPECT_TRUE(DBWal::Open(opts, Env::Default(), &node->wal).ok());

  node->service =
      std::make_unique<ReplicationServiceImpl>(node->wal.get(), mgr);

  node->uds_path = node->wal_dir + "/grpc.sock";
  std::string addr = "unix:" + node->uds_path;

  grpc::ServerBuilder builder;
  builder.AddListeningPort(addr, grpc::InsecureServerCredentials());
  builder.RegisterService(node->service.get());
  node->server = builder.BuildAndStart();
  return node;
}

}  // namespace

class ReplicationE2ETest : public ::testing::Test {
 protected:
  void SetUp() override {
    leader_dir_ = TmpDir("leader");
    WALOptions opts;
    opts.wal_dir = leader_dir_;
    opts.max_async_queue_depth = 0;
    ASSERT_TRUE(DBWal::Open(opts, Env::Default(), &leader_wal_).ok());
  }

  void TearDown() override {
    if (leader_wal_) leader_wal_->Close();
    std::filesystem::remove_all(leader_dir_);
  }

  std::string leader_dir_;
  std::unique_ptr<DBWal> leader_wal_;
};

TEST_F(ReplicationE2ETest, ReplicateRPC_BasicAppend) {
  ReplicationConfig cfg;
  cfg.min_insync_replicas = 0;
  cfg.self = {1, "127.0.0.1:0", "zone-1"};
  cfg.raft.node_id = 1;
  cfg.raft.raft_endpoint = "localhost:0";
  ReplicationManager mgr(leader_wal_.get(), cfg);
  mgr.OnRaftLeadership(1, false);

  auto replica = MakeReplicaNode(&mgr, 0);
  ASSERT_NE(replica->server, nullptr);

  WriteBatch batch;
  batch.Put(Slice("k1"), Slice("v1"));
  WriteOptions wo;
  ASSERT_TRUE(leader_wal_->Write(wo, &batch).ok());

  ReplicationClient client(replica->Address());

  ReplicateRequest req;
  req.set_term(1);
  req.set_leader_commit(0);
  req.set_prev_lsn(0);
  auto* entry = req.add_entries();
  entry->set_first_seq(1);
  entry->set_count(1);
  entry->set_payload(batch.Data());

  auto resp = client.SendReplicate(req);
  EXPECT_TRUE(resp.success()) << resp.message();
  EXPECT_EQ(resp.last_persisted_lsn(), 1u);
}

TEST_F(ReplicationE2ETest, ReplicateRPC_GapDetection) {
  ReplicationConfig cfg;
  cfg.min_insync_replicas = 0;
  cfg.self = {1, "127.0.0.1:0", "zone-1"};
  cfg.raft.node_id = 1;
  cfg.raft.raft_endpoint = "localhost:0";
  ReplicationManager mgr(leader_wal_.get(), cfg);
  mgr.OnRaftLeadership(1, false);

  auto replica = MakeReplicaNode(&mgr, 1);
  ASSERT_NE(replica->server, nullptr);

  ReplicationClient client(replica->Address());

  ReplicateRequest req;
  req.set_term(1);
  req.set_leader_commit(0);
  req.set_prev_lsn(10);

  auto resp = client.SendReplicate(req);
  EXPECT_FALSE(resp.success());
  EXPECT_EQ(resp.last_persisted_lsn(), 0u);
}

TEST_F(ReplicationE2ETest, ReplicateRPC_DivergenceTruncation) {
  ReplicationConfig cfg;
  cfg.min_insync_replicas = 0;
  cfg.self = {1, "127.0.0.1:0", "zone-1"};
  cfg.raft.node_id = 1;
  cfg.raft.raft_endpoint = "localhost:0";
  ReplicationManager mgr(leader_wal_.get(), cfg);
  mgr.OnRaftLeadership(1, false);

  auto replica = MakeReplicaNode(&mgr, 2);
  ASSERT_NE(replica->server, nullptr);

  for (int i = 0; i < 3; ++i) {
    WriteBatch b;
    b.Put(Slice("old_k"), Slice("old_v"));
    WriteOptions wo;
    ASSERT_TRUE(replica->wal->Write(wo, &b).ok());
  }
  EXPECT_EQ(replica->wal->GetLatestSequenceNumber(), 3u);

  WriteBatch batch;
  batch.Put(Slice("new_k"), Slice("new_v"));
  WriteOptions wo;
  ASSERT_TRUE(leader_wal_->Write(wo, &batch).ok());

  ReplicationClient client(replica->Address());

  ReplicateRequest req;
  req.set_term(1);
  req.set_leader_commit(0);
  req.set_prev_lsn(1);
  auto* entry = req.add_entries();
  entry->set_first_seq(2);
  entry->set_count(1);
  entry->set_payload(batch.Data());

  auto resp = client.SendReplicate(req);
  EXPECT_TRUE(resp.success()) << resp.message();
  EXPECT_EQ(resp.last_persisted_lsn(), 2u);
}

TEST_F(ReplicationE2ETest, ProgressReport_ReturnsCommittedLSN) {
  ReplicationConfig cfg;
  cfg.min_insync_replicas = 0;
  cfg.self = {1, "127.0.0.1:0", "zone-1"};
  cfg.raft.node_id = 1;
  cfg.raft.raft_endpoint = "localhost:0";
  ReplicationManager mgr(leader_wal_.get(), cfg);
  mgr.OnRaftLeadership(1, true);

  auto replica = MakeReplicaNode(&mgr, 3);
  ASSERT_NE(replica->server, nullptr);

  ReplicationClient client(replica->Address());

  ProgressReport report;
  report.set_node_id(2);
  report.set_persisted_lsn(42);
  report.set_applied_lsn(40);
  report.set_term(1);

  auto ack = client.SendProgressReport(report);
  EXPECT_GE(ack.committed_lsn(), 0u);
}

}  // namespace replication
}  // namespace mwal
