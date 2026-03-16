#include "replication/replication_manager.h"

#include <algorithm>
#include <chrono>
#include <thread>

#include "replication.pb.h"

namespace mwal {
namespace replication {

ReplicationManager::ReplicationManager(DBWal* wal,
                                       const ReplicationConfig& config)
    : wal_(wal), config_(config), isr_tracker_(config) {}

ReplicationManager::~ReplicationManager() { Stop(); }

Status ReplicationManager::Start() {
  // Register peers in the ISR tracker and create gRPC clients.
  for (const auto& peer : config_.peers) {
    isr_tracker_.AddReplica(peer.node_id, peer.zone);
    clients_[peer.node_id] =
        std::make_unique<ReplicationClient>(peer.endpoint);
  }

  // Start gRPC server.
  grpc_service_ = std::make_unique<ReplicationServiceImpl>(wal_, this);
  grpc::ServerBuilder builder;
  builder.AddListeningPort(config_.self.endpoint,
                           grpc::InsecureServerCredentials());
  builder.RegisterService(grpc_service_.get());
  grpc_server_ = builder.BuildAndStart();
  if (!grpc_server_) {
    return Status::IOError("failed to start gRPC server on " +
                           config_.self.endpoint);
  }

  // Initialise NuRaft.
  auto s = InitRaft();
  if (!s.ok()) return s;

  // Start background threads.
  shutdown_.store(false, std::memory_order_relaxed);
  repl_thread_ = std::thread([this] { ReplicationLoop(); });
  isr_thread_ = std::thread([this] { ISRMaintenanceLoop(); });

  return Status::OK();
}

void ReplicationManager::Stop() {
  shutdown_.store(true, std::memory_order_release);
  pending_cv_.notify_all();
  if (repl_thread_.joinable()) repl_thread_.join();
  if (isr_thread_.joinable()) isr_thread_.join();
  launcher_.shutdown();
  if (grpc_server_) grpc_server_->Shutdown();
}

void ReplicationManager::OnRaftLeadership(uint64_t term, bool is_leader) {
  current_term_.store(term, std::memory_order_release);
  is_leader_.store(is_leader, std::memory_order_release);

  if (is_leader) {
    isr_tracker_.SelectInitialISR(wal_->GetLatestSequenceNumber());
  }
}

void ReplicationManager::HandleProgressReport(uint64_t node_id,
                                              uint64_t persisted_lsn,
                                              uint64_t applied_lsn) {
  isr_tracker_.UpdateReplicaProgress(node_id, persisted_lsn, applied_lsn);
}

uint64_t ReplicationManager::GetCurrentTerm() const {
  return current_term_.load(std::memory_order_acquire);
}

uint64_t ReplicationManager::GetCommittedLSN() const {
  return committed_lsn_.load(std::memory_order_acquire);
}

bool ReplicationManager::IsLeader() const {
  return is_leader_.load(std::memory_order_acquire);
}

void ReplicationManager::OnWriteRecord(SequenceNumber first_seq, uint32_t count,
                                       const Slice& payload) {
  if (!is_leader_.load(std::memory_order_acquire)) return;
  std::lock_guard<std::mutex> lk(pending_mu_);
  pending_entries_.push_back(
      {first_seq, count, std::string(payload.data(), payload.size())});
  pending_cv_.notify_one();
}

void ReplicationManager::ReplicationLoop() {
  uint64_t interval_ms = config_.replication_timeout_ms;
  if (interval_ms == 0) interval_ms = 50;

  while (!shutdown_.load(std::memory_order_acquire)) {
    std::vector<PendingEntry> batch;
    {
      std::unique_lock<std::mutex> lk(pending_mu_);
      pending_cv_.wait_for(lk, std::chrono::milliseconds(interval_ms),
                          [this] { return shutdown_.load(std::memory_order_acquire) || !pending_entries_.empty(); });
      if (shutdown_.load(std::memory_order_acquire)) break;
      if (!pending_entries_.empty()) {
        batch.swap(pending_entries_);
      }
    }
    if (!batch.empty() && is_leader_.load(std::memory_order_acquire)) {
      FanOutEntries(batch);
    }
    committed_lsn_.store(isr_tracker_.ComputeCommittedLSN(),
                         std::memory_order_release);
  }
}

void ReplicationManager::ISRMaintenanceLoop() {
  uint64_t interval_ms = config_.isr_check_interval_ms;
  if (interval_ms == 0) interval_ms = 1000;

  while (!shutdown_.load(std::memory_order_acquire)) {
    std::this_thread::sleep_for(std::chrono::milliseconds(interval_ms));
    if (shutdown_.load(std::memory_order_acquire)) break;
    if (is_leader_.load(std::memory_order_acquire)) {
      isr_tracker_.RunMaintenance(wal_->GetLatestSequenceNumber());
      committed_lsn_.store(isr_tracker_.ComputeCommittedLSN(),
                           std::memory_order_release);
    }
  }
}

void ReplicationManager::FanOutEntries(std::vector<PendingEntry>& entries) {
  if (entries.empty()) return;

  uint64_t term = current_term_.load(std::memory_order_acquire);
  uint64_t commit = committed_lsn_.load(std::memory_order_acquire);
  uint64_t my_lsn = wal_->GetLatestSequenceNumber();
  uint64_t prev_lsn = entries.front().first_seq > 0
                          ? entries.front().first_seq - 1
                          : 0;

  ReplicateRequest req;
  req.set_term(term);
  req.set_leader_commit(commit);
  req.set_prev_lsn(prev_lsn);
  for (auto& e : entries) {
    auto* entry = req.add_entries();
    entry->set_first_seq(e.first_seq);
    entry->set_count(e.count);
    entry->set_payload(e.payload);
  }

  auto isr = isr_tracker_.GetISRSet();
  auto osr = isr_tracker_.GetOSRSet();

  for (const auto& [node_id, client] : clients_) {
    if (!client || !client->IsConnected()) continue;
    ReplicateResponse resp = client->SendReplicate(req);
    if (resp.success()) {
      isr_tracker_.UpdateReplicaProgress(node_id, resp.last_persisted_lsn(),
                                        resp.last_persisted_lsn());
    } else if (resp.last_persisted_lsn() < prev_lsn && isr.count(node_id) == 0) {
      CatchUpReplica(node_id, resp.last_persisted_lsn() + 1);
    }
  }
}

void ReplicationManager::CatchUpReplica(uint64_t /*node_id*/,
                                        uint64_t /*start_lsn*/) {
  // Replica catch-up is pull-based: the replica calls the leader's StreamWAL
  // when it detects it is behind (e.g. from Replicate response). The leader
  // does not push StreamWAL to replicas.
}

// Lightweight in-memory state_mgr for NuRaft.
// Initial cluster must contain all nodes so no node becomes leader until it has a real majority.
class InMemStateMgr : public nuraft::state_mgr {
 public:
  InMemStateMgr(int srv_id, const std::string& endpoint,
                nuraft::ptr<nuraft::log_store> ls,
                const std::vector<nuraft::ptr<nuraft::srv_config>>& initial_servers)
      : my_id_(srv_id), log_store_(std::move(ls)) {
    my_srv_config_ = nuraft::cs_new<nuraft::srv_config>(srv_id, endpoint);
    saved_config_ = nuraft::cs_new<nuraft::cluster_config>();
    for (const auto& srv : initial_servers) {
      saved_config_->get_servers().push_back(srv);
    }
  }

  nuraft::ptr<nuraft::cluster_config> load_config() override {
    return saved_config_;
  }
  void save_config(const nuraft::cluster_config& config) override {
    auto buf = config.serialize();
    saved_config_ = nuraft::cluster_config::deserialize(*buf);
  }
  void save_state(const nuraft::srv_state& state) override {
    auto buf = state.serialize();
    saved_state_ = nuraft::srv_state::deserialize(*buf);
  }
  nuraft::ptr<nuraft::srv_state> read_state() override {
    return saved_state_;
  }
  nuraft::ptr<nuraft::log_store> load_log_store() override {
    return log_store_;
  }
  nuraft::int32 server_id() override { return my_id_; }
  void system_exit(const int /*exit_code*/) override {}

 private:
  int my_id_;
  nuraft::ptr<nuraft::log_store> log_store_;
  nuraft::ptr<nuraft::srv_config> my_srv_config_;
  nuraft::ptr<nuraft::cluster_config> saved_config_;
  nuraft::ptr<nuraft::srv_state> saved_state_;
};

// No-op logger for NuRaft.
class NullLogger : public nuraft::logger {
 public:
  void put_details(int, const char*, const char*, size_t,
                   const std::string&) override {}
};

Status ReplicationManager::InitRaft() {
  sm_ = nuraft::cs_new<RaftStateMachine>(
      [this](uint64_t term, bool leader) {
        OnRaftLeadership(term, leader);
      });
  log_store_ = nuraft::cs_new<RaftLogStore>();

  nuraft::raft_params params;
  params.heart_beat_interval_ =
      static_cast<int>(config_.raft.heartbeat_interval_ms);
  params.election_timeout_lower_bound_ =
      static_cast<int>(config_.raft.election_timeout_lower_ms);
  params.election_timeout_upper_bound_ =
      static_cast<int>(config_.raft.election_timeout_upper_ms);
  params.snapshot_distance_ = 0;
  params.max_append_size_ = 100;
  params.return_method_ = nuraft::raft_params::blocking;

  // Build initial cluster config (all nodes) so Raft starts with full quorum.
  std::vector<nuraft::ptr<nuraft::srv_config>> initial_servers;
  for (const auto& p : config_.raft.initial_cluster) {
    initial_servers.push_back(nuraft::cs_new<nuraft::srv_config>(
        static_cast<int>(p.node_id), p.endpoint));
  }

  auto smgr = nuraft::cs_new<InMemStateMgr>(
      static_cast<int>(config_.raft.node_id),
      config_.raft.raft_endpoint,
      log_store_,
      initial_servers);

  // v3 API: init(sm, smgr, logger, port, asio_options, params, init_options)
  int port = 0;
  std::string ep = config_.raft.raft_endpoint;
  auto colon = ep.rfind(':');
  if (colon != std::string::npos) {
    port = std::atoi(ep.substr(colon + 1).c_str());
  }

  nuraft::raft_server::init_options init_opt;
  init_opt.raft_callback_ = [this](nuraft::cb_func::Type type,
                                    nuraft::cb_func::Param* param) {
    if (!param || !param->ctx) return nuraft::cb_func::Ok;
    uint64_t term = *static_cast<uint64_t*>(param->ctx);
    auto* rsm = static_cast<RaftStateMachine*>(sm_.get());
    if (type == nuraft::cb_func::BecomeLeader) {
      rsm->OnLeadershipChange(term, true);
    } else if (type == nuraft::cb_func::BecomeFollower) {
      rsm->OnLeadershipChange(term, false);
    }
    return nuraft::cb_func::Ok;
  };

  auto raft_instance = launcher_.init(
      sm_, smgr,
      nuraft::cs_new<NullLogger>(),
      port,
      nuraft::asio_service::options{},
      params,
      init_opt);

  if (!raft_instance) {
    return Status::IOError("NuRaft init failed");
  }

  // Ensure peers are registered for connection.
  for (auto& p : initial_servers) {
    if (p->get_id() == static_cast<int>(config_.raft.node_id)) continue;
    raft_instance->add_srv(*p);
  }

  // Block until Raft server is initialized so election can proceed.
  while (!raft_instance->is_initialized()) {
    std::this_thread::sleep_for(std::chrono::milliseconds(10));
  }

  return Status::OK();
}

}  // namespace replication
}  // namespace mwal
