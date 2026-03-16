#include "replication/raft_state_machine.h"

namespace mwal {
namespace replication {

RaftStateMachine::RaftStateMachine(LeadershipCallback cb)
    : leadership_cb_(std::move(cb)) {}

nuraft::ptr<nuraft::buffer> RaftStateMachine::pre_commit(
    const nuraft::ulong /*log_idx*/, nuraft::buffer& /*data*/) {
  return nullptr;
}

nuraft::ptr<nuraft::buffer> RaftStateMachine::commit(
    const nuraft::ulong log_idx, nuraft::buffer& /*data*/) {
  last_committed_idx_.store(log_idx, std::memory_order_release);
  return nullptr;
}

void RaftStateMachine::rollback(const nuraft::ulong /*log_idx*/,
                                nuraft::buffer& /*data*/) {}

nuraft::ulong RaftStateMachine::last_commit_index() {
  return last_committed_idx_.load(std::memory_order_acquire);
}

void RaftStateMachine::create_snapshot(
    nuraft::snapshot& s,
    nuraft::async_result<bool>::handler_type& when_done) {
  {
    std::lock_guard<std::mutex> lk(snap_mu_);
    last_snapshot_ = nuraft::cs_new<nuraft::snapshot>(
        s.get_last_log_idx(), s.get_last_log_term(), s.get_last_config());
  }
  nuraft::ptr<std::exception> nil;
  bool ret = true;
  when_done(ret, nil);
}

void RaftStateMachine::save_logical_snp_obj(
    nuraft::snapshot& s, nuraft::ulong& /*obj_id*/, nuraft::buffer& /*data*/,
    bool /*is_first_obj*/, bool /*is_last_obj*/) {
  std::lock_guard<std::mutex> lk(snap_mu_);
  last_snapshot_ = nuraft::cs_new<nuraft::snapshot>(
      s.get_last_log_idx(), s.get_last_log_term(), s.get_last_config());
}

bool RaftStateMachine::apply_snapshot(nuraft::snapshot& s) {
  std::lock_guard<std::mutex> lk(snap_mu_);
  last_snapshot_ = nuraft::cs_new<nuraft::snapshot>(
      s.get_last_log_idx(), s.get_last_log_term(), s.get_last_config());
  last_committed_idx_.store(s.get_last_log_idx(), std::memory_order_release);
  return true;
}

nuraft::ptr<nuraft::snapshot> RaftStateMachine::last_snapshot() {
  std::lock_guard<std::mutex> lk(snap_mu_);
  return last_snapshot_;
}

int RaftStateMachine::read_logical_snp_obj(
    nuraft::snapshot& /*s*/, void*& /*user_ctx*/, nuraft::ulong /*obj_id*/,
    nuraft::ptr<nuraft::buffer>& data_out, bool& is_last_obj) {
  data_out = nuraft::buffer::alloc(sizeof(int32_t));
  nuraft::buffer_serializer bs(data_out);
  bs.put_i32(0);
  is_last_obj = true;
  return 0;
}

void RaftStateMachine::OnLeadershipChange(uint64_t term, bool is_leader) {
  if (leadership_cb_) {
    leadership_cb_(term, is_leader);
  }
}

}  // namespace replication
}  // namespace mwal
