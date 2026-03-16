#pragma once

#include <chrono>
#include <cstdint>
#include <mutex>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include "replication/replication_config.h"

namespace mwal {
namespace replication {

// Per-replica state tracked by the leader.
struct ReplicaState {
  uint64_t node_id = 0;
  std::string zone;
  uint64_t last_persisted_lsn = 0;
  uint64_t applied_lsn = 0;
  std::chrono::steady_clock::time_point last_ack_time{};
};

// Manages ISR (In-Sync Replicas) and OSR (Out-of-Sync Replicas) sets.
// Thread-safe: all public methods acquire an internal mutex.
class ISRTracker {
 public:
  explicit ISRTracker(const ReplicationConfig& config);

  // Update a replica's progress (called on Replicate ack or ReportProgress).
  void UpdateReplicaProgress(uint64_t node_id, uint64_t persisted_lsn,
                             uint64_t applied_lsn = 0);

  // Register a replica (called when cluster membership changes).
  void AddReplica(uint64_t node_id, const std::string& zone);

  // Remove a replica from all tracking.
  void RemoveReplica(uint64_t node_id);

  // Run periodic ISR maintenance: evict laggards, promote caught-up OSR nodes.
  // |leader_last_seq| is the leader's current last_sequence_.
  void RunMaintenance(uint64_t leader_last_seq);

  // Perform initial ISR selection from the current set of replicas.
  void SelectInitialISR(uint64_t leader_last_seq);

  // Check whether writes should be accepted (|ISR| >= min_insync_replicas).
  bool IsWriteAllowed() const;

  // Compute committed LSN = min persisted_lsn across ISR members.
  uint64_t ComputeCommittedLSN() const;

  // Get the minimum persisted_lsn across all replicas (for WAL retention).
  uint64_t GetMinReplicaLSN() const;

  // Snapshot of the ISR/OSR sets.
  std::unordered_set<uint64_t> GetISRSet() const;
  std::unordered_set<uint64_t> GetOSRSet() const;

  // Get all replica node IDs (ISR + OSR).
  std::vector<uint64_t> GetAllReplicaIDs() const;

 private:
  mutable std::mutex mu_;
  ReplicationConfig config_;

  std::unordered_map<uint64_t, ReplicaState> replicas_;
  std::unordered_set<uint64_t> isr_set_;
  std::unordered_set<uint64_t> osr_set_;
};

}  // namespace replication
}  // namespace mwal
