// Example: N-node replicated WAL cluster.
//
// Usage:
//   replicated_node <node_id> <grpc_port> <raft_port> [num_nodes] [wal_dir] [raft_base] [grpc_base]
//
// If wal_dir is not given, defaults to /tmp/mwal_repl_node_<node_id>.
// raft_base / grpc_base: base for building peer lists (default 9000 / 50050).

#include <cstdlib>
#include <iostream>
#include <string>
#include <thread>

#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/write_batch.h"
#include "replication/replication_config.h"
#include "replication/replication_manager.h"

int main(int argc, char** argv) {
  if (argc < 4) {
    std::cerr << "Usage: " << argv[0]
              << " <node_id> <grpc_port> <raft_port> [num_nodes] [wal_dir] [raft_base] [grpc_base]\n";
    return 1;
  }

  uint64_t node_id = std::stoull(argv[1]);
  std::string grpc_port = argv[2];
  std::string raft_port = argv[3];
  int num_nodes = (argc >= 5) ? std::atoi(argv[4]) : 3;
  std::string wal_dir = (argc >= 6)
      ? argv[5]
      : "/tmp/mwal_repl_node_" + std::to_string(node_id);
  int raft_base = (argc >= 7) ? std::atoi(argv[6]) : 9000;
  int grpc_base = (argc >= 8) ? std::atoi(argv[7]) : 50050;

  mwal::WALOptions wal_opts;
  wal_opts.wal_dir = wal_dir;
  wal_opts.max_wal_file_size = 4 * 1024 * 1024;
  // Run recovery when opening existing WAL so last_sequence_ is set correctly.
  // Otherwise a restarted node reports last_persisted_lsn=0 and catch-up appends
  // from seq 1 into a new file while old files remain, producing duplicate entries.
  wal_opts.recovery_callback = [](mwal::SequenceNumber /*seq*/,
                                  mwal::WriteBatch* /*batch*/) {
    return mwal::Status::OK();
  };

  mwal::replication::ReplicationManager* mgr_ptr = nullptr;

  wal_opts.write_record_callback =
      [&mgr_ptr](mwal::SequenceNumber first_seq, uint32_t count,
                 const mwal::Slice& payload) {
        if (mgr_ptr) {
          mgr_ptr->OnWriteRecord(first_seq, count, payload);
        }
      };

  std::unique_ptr<mwal::DBWal> wal;
  auto s = mwal::DBWal::Open(wal_opts, mwal::Env::Default(), &wal);
  if (!s.ok()) {
    std::cerr << "Failed to open WAL: " << s.ToString() << "\n";
    return 1;
  }

  mwal::replication::ReplicationConfig config;
  config.min_insync_replicas = (num_nodes <= 3) ? 1 : 2;
  config.replication_timeout_ms = 500;
  config.isr_check_interval_ms = 1000;
  config.max_lag_entries = 5000;
  config.replica_timeout_ms = 3000;
  config.batch_max_entries = 50;
  config.batch_max_bytes = 512 * 1024;
  config.progress_report_interval_ms = 500;

  config.self = {node_id, "0.0.0.0:" + grpc_port,
                 "zone-" + std::to_string(node_id)};

  // gRPC peer list (for ReplicationManager gRPC clients)
  std::vector<mwal::replication::NodeInfo> all_nodes;
  for (int i = 1; i <= num_nodes; ++i) {
    all_nodes.push_back(
        {static_cast<uint64_t>(i),
         "127.0.0.1:" + std::to_string(grpc_base + i),
         "zone-" + std::to_string(i)});
  }
  for (const auto& n : all_nodes) {
    if (n.node_id != node_id) {
      config.peers.push_back(n);
    }
  }

  // Raft peer list: NuRaft connects over Raft ports (raft_base + node_id)
  std::vector<mwal::replication::NodeInfo> raft_initial_cluster;
  for (int i = 1; i <= num_nodes; ++i) {
    raft_initial_cluster.push_back(
        {static_cast<uint64_t>(i),
         "127.0.0.1:" + std::to_string(raft_base + i),
         "zone-" + std::to_string(i)});
  }

  config.raft.node_id = node_id;
  config.raft.raft_endpoint = "localhost:" + raft_port;
  config.raft.heartbeat_interval_ms = 50;
  config.raft.election_timeout_lower_ms = 150;
  config.raft.election_timeout_upper_ms = 300;
  config.raft.initial_cluster = raft_initial_cluster;

  mwal::replication::ReplicationManager mgr(wal.get(), config);
  mgr_ptr = &mgr;
  s = mgr.Start();
  if (!s.ok()) {
    std::cerr << "Failed to start replication: " << s.ToString() << "\n";
    return 1;
  }

  std::cout << "Node " << node_id << " started (gRPC=" << grpc_port
            << " Raft=" << raft_port << " cluster=" << num_nodes
            << " wal=" << wal_dir << ")" << std::endl;

  std::string line;
  while (std::getline(std::cin, line)) {
    if (line.empty()) continue;
    if (line == "quit") break;
    if (line == "status") {
      std::cout << "leader=" << mgr.IsLeader()
                << " term=" << mgr.GetCurrentTerm()
                << " committed=" << mgr.GetCommittedLSN()
                << " last_seq=" << wal->GetLatestSequenceNumber()
                << std::endl
                << std::flush;
    } else if (line.size() > 4 && line.substr(0, 4) == "put ") {
      size_t sp1 = line.find(' ', 4);
      if (sp1 != std::string::npos) {
        std::string k = line.substr(4, sp1 - 4);
        std::string v = line.substr(sp1 + 1);
        mwal::WriteBatch batch;
        batch.Put(mwal::Slice(k), mwal::Slice(v));
        mwal::WriteOptions wo;
        wo.sync = false;
        s = wal->Write(wo, &batch);
        if (!s.ok()) {
          std::cout << "Write failed: " << s.ToString() << std::endl;
        }
      }
    } else {
      std::cout << "Commands: put <k> <v> | status | quit" << std::endl;
    }
  }

  mgr_ptr = nullptr;
  mgr.Stop();
  wal->Close();
  return 0;
}
