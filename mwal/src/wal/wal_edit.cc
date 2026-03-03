// Derived from RocksDB — WAL metadata tracking for mwal.

#include "wal/wal_edit.h"

#include <sstream>

#include "util/coding.h"

namespace mwal {

std::string WalAddition::DebugString() const {
  std::ostringstream oss;
  oss << "WalAddition{log=" << log_number << ", size=" << size_bytes
      << ", synced=" << (synced ? "true" : "false") << "}";
  return oss.str();
}

void WalAddition::EncodeTo(std::string* dst) const {
  PutVarint64(dst, log_number);
  PutVarint64(dst, size_bytes);
  dst->push_back(synced ? 1 : 0);
}

Status WalAddition::DecodeFrom(Slice* src) {
  if (!GetVarint64(src, &log_number)) {
    return Status::Corruption("bad WalAddition log_number");
  }
  if (!GetVarint64(src, &size_bytes)) {
    return Status::Corruption("bad WalAddition size_bytes");
  }
  if (src->empty()) {
    return Status::Corruption("bad WalAddition synced");
  }
  synced = ((*src)[0] != 0);
  src->remove_prefix(1);
  return Status::OK();
}

std::string WalDeletion::DebugString() const {
  std::ostringstream oss;
  oss << "WalDeletion{log=" << log_number << "}";
  return oss.str();
}

void WalDeletion::EncodeTo(std::string* dst) const {
  PutVarint64(dst, log_number);
}

Status WalDeletion::DecodeFrom(Slice* src) {
  if (!GetVarint64(src, &log_number)) {
    return Status::Corruption("bad WalDeletion log_number");
  }
  return Status::OK();
}

void WalSet::AddWal(const WalAddition& wal) {
  wals_[wal.log_number] = wal;
}

void WalSet::DeleteWalsBefore(uint64_t min_log_number) {
  auto it = wals_.begin();
  while (it != wals_.end() && it->first < min_log_number) {
    it = wals_.erase(it);
  }
}

bool WalSet::HasWal(uint64_t log_number) const {
  return wals_.count(log_number) > 0;
}

uint64_t WalSet::GetMinLogNumber() const {
  if (wals_.empty()) return 0;
  return wals_.begin()->first;
}

Status WalSet::CheckWals() const {
  for (auto& [num, wal] : wals_) {
    if (wal.log_number != num) {
      return Status::Corruption("WalSet inconsistency");
    }
  }
  return Status::OK();
}

}  // namespace mwal
