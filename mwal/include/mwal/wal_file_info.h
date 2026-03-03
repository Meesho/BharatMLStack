// WAL file metadata — public type used by GetLiveWalFiles and WalIterator.

#pragma once

#include <cstdint>
#include <string>

namespace mwal {

struct WalFileInfo {
  uint64_t log_number;
  uint64_t size_bytes;
  std::string path;

  bool operator<(const WalFileInfo& other) const {
    return log_number < other.log_number;
  }
};

}  // namespace mwal
