#pragma once

#include <map>
#include <mutex>

#include <libnuraft/nuraft.hxx>

namespace mwal {
namespace replication {

// In-memory log store for NuRaft metadata-only Raft log.
// Lightweight — stores only cluster config changes and ISR metadata.
class RaftLogStore : public nuraft::log_store {
 public:
  RaftLogStore();

  uint64_t next_slot() const override;
  uint64_t start_index() const override;
  nuraft::ptr<nuraft::log_entry> last_entry() const override;

  uint64_t append(nuraft::ptr<nuraft::log_entry>& entry) override;
  void write_at(uint64_t index,
                nuraft::ptr<nuraft::log_entry>& entry) override;

  nuraft::ptr<std::vector<nuraft::ptr<nuraft::log_entry>>>
  log_entries(uint64_t start, uint64_t end) override;

  nuraft::ptr<std::vector<nuraft::ptr<nuraft::log_entry>>>
  log_entries_ext(uint64_t start, uint64_t end,
                  int64_t batch_size_hint = 0) override;

  nuraft::ptr<nuraft::log_entry> entry_at(uint64_t index) override;
  uint64_t term_at(uint64_t index) override;

  nuraft::ptr<nuraft::buffer> pack(uint64_t index, int32_t cnt) override;
  void apply_pack(uint64_t index, nuraft::buffer& pack) override;

  bool compact(uint64_t last_log_index) override;
  bool flush() override;

 private:
  static nuraft::ptr<nuraft::log_entry> MakeDummyEntry();

  mutable std::mutex mu_;
  std::map<uint64_t, nuraft::ptr<nuraft::log_entry>> logs_;
  uint64_t start_idx_ = 1;
};

}  // namespace replication
}  // namespace mwal
