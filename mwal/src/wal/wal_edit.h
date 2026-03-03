// Derived from RocksDB — WAL metadata tracking for mwal.

#pragma once

#include <cstdint>
#include <map>
#include <string>

#include "mwal/status.h"

namespace mwal {

struct WalAddition {
  uint64_t log_number = 0;
  uint64_t size_bytes = 0;
  bool synced = false;

  WalAddition() = default;
  WalAddition(uint64_t log_num, uint64_t sz, bool s)
      : log_number(log_num), size_bytes(sz), synced(s) {}

  std::string DebugString() const;
  void EncodeTo(std::string* dst) const;
  Status DecodeFrom(Slice* src);
};

struct WalDeletion {
  uint64_t log_number = 0;

  WalDeletion() = default;
  explicit WalDeletion(uint64_t log_num) : log_number(log_num) {}

  std::string DebugString() const;
  void EncodeTo(std::string* dst) const;
  Status DecodeFrom(Slice* src);
};

class WalSet {
 public:
  void AddWal(const WalAddition& wal);
  void DeleteWalsBefore(uint64_t min_log_number);

  const std::map<uint64_t, WalAddition>& GetWals() const { return wals_; }
  bool HasWal(uint64_t log_number) const;
  uint64_t GetMinLogNumber() const;

  Status CheckWals() const;

 private:
  std::map<uint64_t, WalAddition> wals_;
};

}  // namespace mwal
