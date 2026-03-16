#pragma once

#include <cstdint>
#include <string>
#include <vector>

namespace mwal {
namespace replication {

struct NodeInfo {
  uint64_t node_id = 0;
  std::string endpoint;  // host:port for gRPC
  std::string zone;      // failure domain for ISR diversity
};

struct RaftConfig {
  uint64_t election_timeout_lower_ms = 200;
  uint64_t election_timeout_upper_ms = 400;
  uint64_t heartbeat_interval_ms = 75;
  uint64_t node_id = 0;
  std::string raft_endpoint;              // host:port for NuRaft
  // NodeInfo::endpoint here must be Raft (host:port), not gRPC — NuRaft uses it for Raft protocol connections.
  std::vector<NodeInfo> initial_cluster;  // initial peer list
};

struct ReplicationConfig {
  uint32_t min_insync_replicas = 2;
  uint64_t replication_timeout_ms = 150;
  uint64_t isr_check_interval_ms = 1000;
  uint64_t max_lag_entries = 5000;
  uint64_t replica_timeout_ms = 3000;
  uint32_t batch_max_entries = 100;
  uint64_t batch_max_bytes = 1048576;  // 1 MiB
  uint64_t progress_report_interval_ms = 500;
  uint64_t max_replica_lag_before_snapshot = 50000;

  RaftConfig raft;
  NodeInfo self;
  std::vector<NodeInfo> peers;
};

}  // namespace replication
}  // namespace mwal
