#pragma once

#include <functional>
#include <mutex>
#include <vector>

#include <libnuraft/nuraft.hxx>

namespace mwal {
namespace replication {

using LeadershipCallback =
    std::function<void(uint64_t term, bool is_leader)>;

class RaftStateMachine : public nuraft::state_machine {
 public:
  explicit RaftStateMachine(LeadershipCallback cb);

  nuraft::ptr<nuraft::buffer> pre_commit(const nuraft::ulong log_idx,
                                          nuraft::buffer& data) override;
  nuraft::ptr<nuraft::buffer> commit(const nuraft::ulong log_idx,
                                      nuraft::buffer& data) override;
  void rollback(const nuraft::ulong log_idx,
                nuraft::buffer& data) override;

  void save_logical_snp_obj(nuraft::snapshot& s, nuraft::ulong& obj_id,
                            nuraft::buffer& data, bool is_first_obj,
                            bool is_last_obj) override;
  bool apply_snapshot(nuraft::snapshot& s) override;
  nuraft::ptr<nuraft::snapshot> last_snapshot() override;
  int read_logical_snp_obj(nuraft::snapshot& s, void*& user_ctx,
                           nuraft::ulong obj_id,
                           nuraft::ptr<nuraft::buffer>& data_out,
                           bool& is_last_obj) override;

  nuraft::ulong last_commit_index() override;
  void create_snapshot(nuraft::snapshot& s,
                       nuraft::async_result<bool>::handler_type& when_done) override;

  void OnLeadershipChange(uint64_t term, bool is_leader);

 private:
  LeadershipCallback leadership_cb_;
  std::atomic<uint64_t> last_committed_idx_{0};
  std::mutex snap_mu_;
  nuraft::ptr<nuraft::snapshot> last_snapshot_;
};

}  // namespace replication
}  // namespace mwal
