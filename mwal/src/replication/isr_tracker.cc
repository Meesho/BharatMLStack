#include "replication/isr_tracker.h"

#include <algorithm>
#include <limits>

namespace mwal {
namespace replication {

ISRTracker::ISRTracker(const ReplicationConfig& config) : config_(config) {}

void ISRTracker::AddReplica(uint64_t node_id, const std::string& zone) {
  std::lock_guard<std::mutex> lk(mu_);
  ReplicaState& rs = replicas_[node_id];
  rs.node_id = node_id;
  rs.zone = zone;
  rs.last_ack_time = std::chrono::steady_clock::now();
  // New replicas start in OSR until proven in-sync.
  osr_set_.insert(node_id);
}

void ISRTracker::RemoveReplica(uint64_t node_id) {
  std::lock_guard<std::mutex> lk(mu_);
  replicas_.erase(node_id);
  isr_set_.erase(node_id);
  osr_set_.erase(node_id);
}

void ISRTracker::UpdateReplicaProgress(uint64_t node_id,
                                       uint64_t persisted_lsn,
                                       uint64_t applied_lsn) {
  std::lock_guard<std::mutex> lk(mu_);
  auto it = replicas_.find(node_id);
  if (it == replicas_.end()) return;

  auto& rs = it->second;
  if (persisted_lsn > rs.last_persisted_lsn) {
    rs.last_persisted_lsn = persisted_lsn;
  }
  if (applied_lsn > rs.applied_lsn) {
    rs.applied_lsn = applied_lsn;
  }
  rs.last_ack_time = std::chrono::steady_clock::now();
}

void ISRTracker::SelectInitialISR(uint64_t leader_last_seq) {
  std::lock_guard<std::mutex> lk(mu_);
  isr_set_.clear();
  osr_set_.clear();

  // Collect eligible replicas and sort by zone diversity then lag.
  struct Candidate {
    uint64_t node_id;
    std::string zone;
    uint64_t lag;
  };
  std::vector<Candidate> candidates;
  for (const auto& [id, rs] : replicas_) {
    uint64_t lag =
        (leader_last_seq > rs.last_persisted_lsn)
            ? (leader_last_seq - rs.last_persisted_lsn)
            : 0;
    candidates.push_back({id, rs.zone, lag});
  }

  // Sort: zone diversity first (unique zones prioritised), then by lag.
  std::unordered_set<std::string> seen_zones;
  std::stable_sort(candidates.begin(), candidates.end(),
                   [&](const Candidate& a, const Candidate& b) {
                     bool a_new = seen_zones.find(a.zone) == seen_zones.end();
                     bool b_new = seen_zones.find(b.zone) == seen_zones.end();
                     if (a_new != b_new) return a_new > b_new;
                     return a.lag < b.lag;
                   });

  for (const auto& c : candidates) {
    if (c.lag <= config_.max_lag_entries) {
      isr_set_.insert(c.node_id);
      seen_zones.insert(c.zone);
    } else {
      osr_set_.insert(c.node_id);
    }
  }

  // Ensure OSR catches any remaining.
  for (const auto& [id, _] : replicas_) {
    if (isr_set_.count(id) == 0) {
      osr_set_.insert(id);
    }
  }
}

void ISRTracker::RunMaintenance(uint64_t leader_last_seq) {
  std::lock_guard<std::mutex> lk(mu_);
  auto now = std::chrono::steady_clock::now();

  // Evict from ISR: lag too high or no ack for too long.
  std::vector<uint64_t> to_evict;
  for (uint64_t id : isr_set_) {
    auto it = replicas_.find(id);
    if (it == replicas_.end()) {
      to_evict.push_back(id);
      continue;
    }
    const auto& rs = it->second;
    uint64_t lag =
        (leader_last_seq > rs.last_persisted_lsn)
            ? (leader_last_seq - rs.last_persisted_lsn)
            : 0;

    bool lag_exceeded = lag > config_.max_lag_entries;
    bool timeout_exceeded =
        std::chrono::duration_cast<std::chrono::milliseconds>(
            now - rs.last_ack_time)
            .count() > static_cast<int64_t>(config_.replica_timeout_ms);

    if (lag_exceeded || timeout_exceeded) {
      to_evict.push_back(id);
    }
  }
  for (uint64_t id : to_evict) {
    isr_set_.erase(id);
    osr_set_.insert(id);
  }

  // Promote from OSR: caught up and healthy.
  std::vector<uint64_t> to_promote;
  for (uint64_t id : osr_set_) {
    auto it = replicas_.find(id);
    if (it == replicas_.end()) continue;
    const auto& rs = it->second;
    uint64_t lag =
        (leader_last_seq > rs.last_persisted_lsn)
            ? (leader_last_seq - rs.last_persisted_lsn)
            : 0;
    bool recent_ack =
        std::chrono::duration_cast<std::chrono::milliseconds>(
            now - rs.last_ack_time)
            .count() <= static_cast<int64_t>(config_.replica_timeout_ms);

    if (lag <= config_.max_lag_entries && recent_ack) {
      to_promote.push_back(id);
    }
  }
  for (uint64_t id : to_promote) {
    osr_set_.erase(id);
    isr_set_.insert(id);
  }
}

bool ISRTracker::IsWriteAllowed() const {
  std::lock_guard<std::mutex> lk(mu_);
  return isr_set_.size() >= config_.min_insync_replicas;
}

uint64_t ISRTracker::ComputeCommittedLSN() const {
  std::lock_guard<std::mutex> lk(mu_);
  uint64_t min_lsn = std::numeric_limits<uint64_t>::max();
  for (uint64_t id : isr_set_) {
    auto it = replicas_.find(id);
    if (it == replicas_.end()) continue;
    min_lsn = std::min(min_lsn, it->second.last_persisted_lsn);
  }
  return (min_lsn == std::numeric_limits<uint64_t>::max()) ? 0 : min_lsn;
}

uint64_t ISRTracker::GetMinReplicaLSN() const {
  std::lock_guard<std::mutex> lk(mu_);
  uint64_t min_lsn = std::numeric_limits<uint64_t>::max();
  for (const auto& [_, rs] : replicas_) {
    min_lsn = std::min(min_lsn, rs.last_persisted_lsn);
  }
  return (min_lsn == std::numeric_limits<uint64_t>::max()) ? 0 : min_lsn;
}

std::unordered_set<uint64_t> ISRTracker::GetISRSet() const {
  std::lock_guard<std::mutex> lk(mu_);
  return isr_set_;
}

std::unordered_set<uint64_t> ISRTracker::GetOSRSet() const {
  std::lock_guard<std::mutex> lk(mu_);
  return osr_set_;
}

std::vector<uint64_t> ISRTracker::GetAllReplicaIDs() const {
  std::lock_guard<std::mutex> lk(mu_);
  std::vector<uint64_t> ids;
  ids.reserve(replicas_.size());
  for (const auto& [id, _] : replicas_) {
    ids.push_back(id);
  }
  return ids;
}

}  // namespace replication
}  // namespace mwal
