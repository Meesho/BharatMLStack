#include "replication/raft_log_store.h"

#include <cstring>

namespace mwal {
namespace replication {

RaftLogStore::RaftLogStore() {
  auto dummy = MakeDummyEntry();
  logs_[0] = dummy;
}

nuraft::ptr<nuraft::log_entry> RaftLogStore::MakeDummyEntry() {
  nuraft::ptr<nuraft::buffer> buf = nuraft::buffer::alloc(sizeof(uint64_t));
  return nuraft::cs_new<nuraft::log_entry>(0, buf);
}

uint64_t RaftLogStore::next_slot() const {
  std::lock_guard<std::mutex> lk(mu_);
  if (logs_.empty()) return start_idx_;
  return logs_.rbegin()->first + 1;
}

uint64_t RaftLogStore::start_index() const {
  std::lock_guard<std::mutex> lk(mu_);
  return start_idx_;
}

nuraft::ptr<nuraft::log_entry> RaftLogStore::last_entry() const {
  std::lock_guard<std::mutex> lk(mu_);
  if (logs_.empty()) return MakeDummyEntry();
  return logs_.rbegin()->second;
}

uint64_t RaftLogStore::append(nuraft::ptr<nuraft::log_entry>& entry) {
  std::lock_guard<std::mutex> lk(mu_);
  uint64_t idx = logs_.empty() ? start_idx_ : logs_.rbegin()->first + 1;
  logs_[idx] = entry;
  return idx;
}

void RaftLogStore::write_at(uint64_t index,
                            nuraft::ptr<nuraft::log_entry>& entry) {
  std::lock_guard<std::mutex> lk(mu_);
  // Remove everything from index onward.
  auto it = logs_.lower_bound(index);
  logs_.erase(it, logs_.end());
  logs_[index] = entry;
}

nuraft::ptr<std::vector<nuraft::ptr<nuraft::log_entry>>>
RaftLogStore::log_entries(uint64_t start, uint64_t end) {
  auto ret =
      nuraft::cs_new<std::vector<nuraft::ptr<nuraft::log_entry>>>();
  std::lock_guard<std::mutex> lk(mu_);
  for (auto it = logs_.lower_bound(start);
       it != logs_.end() && it->first < end; ++it) {
    ret->push_back(it->second);
  }
  return ret;
}

nuraft::ptr<std::vector<nuraft::ptr<nuraft::log_entry>>>
RaftLogStore::log_entries_ext(uint64_t start, uint64_t end,
                              int64_t /*batch_size_hint*/) {
  return log_entries(start, end);
}

nuraft::ptr<nuraft::log_entry> RaftLogStore::entry_at(uint64_t index) {
  std::lock_guard<std::mutex> lk(mu_);
  auto it = logs_.find(index);
  if (it == logs_.end()) return nullptr;
  return it->second;
}

uint64_t RaftLogStore::term_at(uint64_t index) {
  auto entry = entry_at(index);
  if (!entry) return 0;
  return entry->get_term();
}

nuraft::ptr<nuraft::buffer> RaftLogStore::pack(uint64_t index, int32_t cnt) {
  std::lock_guard<std::mutex> lk(mu_);
  std::vector<nuraft::ptr<nuraft::buffer>> bufs;
  auto it = logs_.lower_bound(index);
  for (int32_t i = 0; i < cnt && it != logs_.end(); ++i, ++it) {
    nuraft::ptr<nuraft::buffer> b = it->second->serialize();
    bufs.push_back(b);
  }

  size_t total = sizeof(int32_t);
  for (auto& b : bufs) total += sizeof(int32_t) + b->size();

  auto result = nuraft::buffer::alloc(total);
  nuraft::buffer_serializer bs(result);
  bs.put_i32(static_cast<int32_t>(bufs.size()));
  for (auto& b : bufs) {
    bs.put_i32(static_cast<int32_t>(b->size()));
    bs.put_raw(b->data(), b->size());
  }
  return result;
}

void RaftLogStore::apply_pack(uint64_t index, nuraft::buffer& pack) {
  std::lock_guard<std::mutex> lk(mu_);
  nuraft::buffer_serializer bs(pack);
  int32_t count = bs.get_i32();
  for (int32_t i = 0; i < count; ++i) {
    int32_t sz = bs.get_i32();
    nuraft::ptr<nuraft::buffer> buf = nuraft::buffer::alloc(sz);
    void* raw = bs.get_raw(sz);
    ::memcpy(buf->data(), raw, sz);
    auto entry = nuraft::log_entry::deserialize(*buf);
    logs_[index + static_cast<uint64_t>(i)] = entry;
  }
}

bool RaftLogStore::compact(uint64_t last_log_index) {
  std::lock_guard<std::mutex> lk(mu_);
  auto it = logs_.begin();
  while (it != logs_.end() && it->first <= last_log_index) {
    it = logs_.erase(it);
  }
  if (!logs_.empty()) {
    start_idx_ = logs_.begin()->first;
  } else {
    start_idx_ = last_log_index + 1;
  }
  return true;
}

bool RaftLogStore::flush() { return true; }

}  // namespace replication
}  // namespace mwal
