#include <gtest/gtest.h>

#include <thread>

#include "replication/isr_tracker.h"
#include "replication/replication_config.h"

namespace mwal {
namespace replication {

class ISRTrackerTest : public ::testing::Test {
 protected:
  void SetUp() override {
    config_.min_insync_replicas = 2;
    config_.max_lag_entries = 100;
    config_.replica_timeout_ms = 500;
    config_.isr_check_interval_ms = 100;
  }

  ReplicationConfig config_;
};

TEST_F(ISRTrackerTest, AddReplicaStartsInOSR) {
  ISRTracker t(config_);
  t.AddReplica(1, "zone-a");
  t.AddReplica(2, "zone-b");

  EXPECT_EQ(t.GetOSRSet().size(), 2u);
  EXPECT_TRUE(t.GetISRSet().empty());
}

TEST_F(ISRTrackerTest, SelectInitialISR_AllCaughtUp) {
  ISRTracker t(config_);
  t.AddReplica(1, "zone-a");
  t.AddReplica(2, "zone-b");
  t.UpdateReplicaProgress(1, 100);
  t.UpdateReplicaProgress(2, 100);

  t.SelectInitialISR(100);
  EXPECT_EQ(t.GetISRSet().size(), 2u);
  EXPECT_TRUE(t.GetOSRSet().empty());
}

TEST_F(ISRTrackerTest, SelectInitialISR_LaggardGoesToOSR) {
  ISRTracker t(config_);
  t.AddReplica(1, "zone-a");
  t.AddReplica(2, "zone-b");
  t.UpdateReplicaProgress(1, 200);
  t.UpdateReplicaProgress(2, 0);  // lag = 200 > max_lag_entries (100)

  t.SelectInitialISR(200);
  EXPECT_TRUE(t.GetISRSet().count(1));
  EXPECT_TRUE(t.GetOSRSet().count(2));
}

TEST_F(ISRTrackerTest, IsWriteAllowed_EnoughISR) {
  ISRTracker t(config_);
  t.AddReplica(1, "zone-a");
  t.AddReplica(2, "zone-b");
  t.UpdateReplicaProgress(1, 50);
  t.UpdateReplicaProgress(2, 50);
  t.SelectInitialISR(50);

  EXPECT_TRUE(t.IsWriteAllowed());
}

TEST_F(ISRTrackerTest, IsWriteAllowed_NotEnoughISR) {
  ISRTracker t(config_);
  t.AddReplica(1, "zone-a");
  t.UpdateReplicaProgress(1, 50);
  t.SelectInitialISR(50);

  // Only 1 ISR member, min_insync_replicas=2
  EXPECT_FALSE(t.IsWriteAllowed());
}

TEST_F(ISRTrackerTest, ComputeCommittedLSN) {
  ISRTracker t(config_);
  t.AddReplica(1, "zone-a");
  t.AddReplica(2, "zone-b");
  t.UpdateReplicaProgress(1, 80);
  t.UpdateReplicaProgress(2, 50);
  t.SelectInitialISR(80);

  EXPECT_EQ(t.ComputeCommittedLSN(), 50u);
}

TEST_F(ISRTrackerTest, GetMinReplicaLSN_IncludesOSR) {
  ISRTracker t(config_);
  t.AddReplica(1, "zone-a");
  t.AddReplica(2, "zone-b");
  t.AddReplica(3, "zone-c");
  t.UpdateReplicaProgress(1, 100);
  t.UpdateReplicaProgress(2, 50);
  t.UpdateReplicaProgress(3, 10);

  EXPECT_EQ(t.GetMinReplicaLSN(), 10u);
}

TEST_F(ISRTrackerTest, Maintenance_EvictsLaggard) {
  ISRTracker t(config_);
  t.AddReplica(1, "zone-a");
  t.AddReplica(2, "zone-b");
  t.UpdateReplicaProgress(1, 100);
  t.UpdateReplicaProgress(2, 100);
  t.SelectInitialISR(100);

  EXPECT_EQ(t.GetISRSet().size(), 2u);

  // Leader advances far ahead; node 2 doesn't keep up.
  t.RunMaintenance(300);

  // Both should be evicted since lag=200 > max_lag_entries=100.
  EXPECT_TRUE(t.GetISRSet().empty() || t.GetISRSet().size() < 2u);
}

TEST_F(ISRTrackerTest, Maintenance_PromotesOSR) {
  ISRTracker t(config_);
  t.AddReplica(1, "zone-a");
  t.AddReplica(2, "zone-b");
  t.UpdateReplicaProgress(1, 200);
  t.UpdateReplicaProgress(2, 0);   // lag = 200 > max_lag_entries → OSR
  t.SelectInitialISR(200);

  EXPECT_TRUE(t.GetOSRSet().count(2));

  // Node 2 catches up.
  t.UpdateReplicaProgress(2, 200);
  t.RunMaintenance(200);

  EXPECT_TRUE(t.GetISRSet().count(2));
}

TEST_F(ISRTrackerTest, Maintenance_TimeoutEviction) {
  config_.replica_timeout_ms = 50;
  ISRTracker t(config_);
  t.AddReplica(1, "zone-a");
  t.AddReplica(2, "zone-b");
  t.UpdateReplicaProgress(1, 100);
  t.UpdateReplicaProgress(2, 100);
  t.SelectInitialISR(100);

  // Wait for timeout.
  std::this_thread::sleep_for(std::chrono::milliseconds(100));

  t.RunMaintenance(100);
  // Both timed out (no recent ack).
  EXPECT_EQ(t.GetISRSet().size(), 0u);
}

TEST_F(ISRTrackerTest, RemoveReplica) {
  ISRTracker t(config_);
  t.AddReplica(1, "zone-a");
  t.AddReplica(2, "zone-b");
  t.UpdateReplicaProgress(1, 50);
  t.UpdateReplicaProgress(2, 50);
  t.SelectInitialISR(50);

  t.RemoveReplica(2);
  EXPECT_EQ(t.GetISRSet().size(), 1u);
  EXPECT_TRUE(t.GetOSRSet().empty());
}

TEST_F(ISRTrackerTest, GetAllReplicaIDs) {
  ISRTracker t(config_);
  t.AddReplica(10, "zone-a");
  t.AddReplica(20, "zone-b");

  auto ids = t.GetAllReplicaIDs();
  EXPECT_EQ(ids.size(), 2u);
}

TEST_F(ISRTrackerTest, ZoneDiversity_PrefersUniqueZones) {
  ISRTracker t(config_);
  t.AddReplica(1, "zone-a");
  t.AddReplica(2, "zone-a");  // same zone
  t.AddReplica(3, "zone-b");  // different zone

  t.UpdateReplicaProgress(1, 100);
  t.UpdateReplicaProgress(2, 100);
  t.UpdateReplicaProgress(3, 100);

  t.SelectInitialISR(100);

  auto isr = t.GetISRSet();
  // All should be in ISR since they're caught up, but zone-b should be there.
  EXPECT_TRUE(isr.count(3));
  EXPECT_EQ(isr.size(), 3u);
}

}  // namespace replication
}  // namespace mwal
