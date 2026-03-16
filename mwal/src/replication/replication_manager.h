#pragma once

#include <atomic>
#include <condition_variable>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <unordered_map>
#include <vector>

#include <grpcpp/grpcpp.h>
#include <libnuraft/nuraft.hxx>

#include "mwal/db_wal.h"
#include "mwal/slice.h"
#include "replication/isr_tracker.h"
#include "replication/raft_log_store.h"
#include "replication/raft_state_machine.h"
#include "replication/replication_client.h"
#include "replication/replication_config.h"
#include "replication/replication_service.h"

namespace mwal {
namespace replication {

// Buffered WAL entry captured by WriteRecordCallback.
struct PendingEntry {
  SequenceNumber first_seq;
  uint32_t count;
  std::string payload;
};

// Central orchestrator for the replication layer.
// On the leader: captures writes via callback, batches, fans out to ISR/OSR.
// On replicas: runs the gRPC server that accepts Replicate / StreamWAL.
class ReplicationManager {
 public:
  ReplicationManager(DBWal* wal, const ReplicationConfig& config);
  ~ReplicationManager();

  // Start NuRaft, gRPC server, and background threads.
  Status Start();

  // Graceful shutdown.
  void Stop();

  // Called by NuRaft when leadership changes.
  void OnRaftLeadership(uint64_t term, bool is_leader);

  // Called by ReplicationServiceImpl on ReportProgress RPC.
  void HandleProgressReport(uint64_t node_id, uint64_t persisted_lsn,
                            uint64_t applied_lsn);

  // Query methods used by the gRPC service.
  uint64_t GetCurrentTerm() const;
  uint64_t GetCommittedLSN() const;
  bool IsLeader() const;

  // Access to ISR tracker (for tests).
  ISRTracker* GetISRTracker() { return &isr_tracker_; }

  // Called from WALOptions::write_record_callback (leader only). Buffers
  // entries for replication; ReplicationLoop drains and fans out.
  void OnWriteRecord(SequenceNumber first_seq, uint32_t count,
                     const Slice& payload);

 private:

  // Background: drain pending entries and fan out Replicate RPCs.
  void ReplicationLoop();

  // Background: periodic ISR maintenance.
  void ISRMaintenanceLoop();

  // Fan out a batch of entries to all replicas.
  void FanOutEntries(std::vector<PendingEntry>& entries);

  // Initiate catch-up for a specific replica via StreamWAL.
  void CatchUpReplica(uint64_t node_id, uint64_t start_lsn);

  // Initialise NuRaft cluster.
  Status InitRaft();

  DBWal* wal_;
  ReplicationConfig config_;
  ISRTracker isr_tracker_;

  // NuRaft
  nuraft::ptr<RaftStateMachine> sm_;
  nuraft::ptr<RaftLogStore> log_store_;
  nuraft::raft_launcher launcher_;

  // gRPC server
  std::unique_ptr<ReplicationServiceImpl> grpc_service_;
  std::unique_ptr<grpc::Server> grpc_server_;

  // gRPC clients to peers
  std::unordered_map<uint64_t, std::unique_ptr<ReplicationClient>> clients_;

  // Leadership state
  std::atomic<bool> is_leader_{false};
  std::atomic<uint64_t> current_term_{0};
  std::atomic<uint64_t> committed_lsn_{0};

  // Pending entry buffer (filled by WriteRecordCallback on leader).
  std::mutex pending_mu_;
  std::condition_variable pending_cv_;
  std::vector<PendingEntry> pending_entries_;

  // Background threads
  std::thread repl_thread_;
  std::thread isr_thread_;
  std::atomic<bool> shutdown_{false};
};

}  // namespace replication
}  // namespace mwal
