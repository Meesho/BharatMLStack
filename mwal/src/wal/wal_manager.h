// Derived from RocksDB — WAL file lifecycle management for mwal.

#pragma once

#include <cstdint>
#include <functional>
#include <mutex>
#include <string>
#include <vector>

#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/status.h"
#include "mwal/wal_file_info.h"

namespace mwal {

class WalManager {
 public:
  WalManager(Env* env, const WALOptions& options,
             std::function<uint64_t()> get_min_log_number_to_keep);

  Status GetSortedWalFiles(std::vector<WalFileInfo>* files);
  Status PurgeObsoleteWALFiles();
  Status DeleteWalFile(const std::string& path);

  const std::string& GetWalDir() const { return wal_dir_; }

 private:
  Status ScanWalDir(std::vector<WalFileInfo>* files);
  bool ShouldPurgeWal(const WalFileInfo& info, uint64_t min_log_to_keep,
                      uint64_t now_seconds, uint64_t total_wal_size,
                      uint64_t size_limit_bytes) const;

  Env* env_;
  std::string wal_dir_;
  WALOptions options_;
  std::function<uint64_t()> get_min_log_number_to_keep_;
  std::mutex mu_;
};

}  // namespace mwal
