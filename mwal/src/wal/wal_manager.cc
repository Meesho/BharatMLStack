// Derived from RocksDB — WAL file lifecycle management for mwal.

#include "wal/wal_manager.h"

#include <algorithm>
#include <cstdlib>
#include <string>

namespace mwal {

namespace {

bool ParseLogNumber(const std::string& fname, uint64_t* number) {
  // WAL files are named like 000042.log
  size_t dot_pos = fname.find('.');
  if (dot_pos == std::string::npos) return false;
  std::string suffix = fname.substr(dot_pos + 1);
  if (suffix != "log") return false;
  std::string num_str = fname.substr(0, dot_pos);
  if (num_str.empty()) return false;
  char* end = nullptr;
  *number = std::strtoull(num_str.c_str(), &end, 10);
  return (end != nullptr && *end == '\0');
}

}  // namespace

WalManager::WalManager(Env* env, const WALOptions& options,
                       std::function<uint64_t()> get_min_log_number_to_keep)
    : env_(env),
      wal_dir_(options.wal_dir),
      options_(options),
      get_min_log_number_to_keep_(std::move(get_min_log_number_to_keep)) {}

Status WalManager::GetSortedWalFiles(std::vector<WalFileInfo>* files) {
  std::lock_guard<std::mutex> lock(mu_);
  Status s = ScanWalDir(files);
  if (s.ok()) {
    std::sort(files->begin(), files->end());
  }
  return s;
}

Status WalManager::ScanWalDir(std::vector<WalFileInfo>* files) {
  files->clear();
  std::vector<std::string> children;
  Status s = env_->GetChildren(wal_dir_, &children);
  if (!s.ok()) return s;

  for (const auto& fname : children) {
    uint64_t log_num = 0;
    if (ParseLogNumber(fname, &log_num)) {
      std::string full_path = wal_dir_ + "/" + fname;
      uint64_t fsize = 0;
      Status fs = env_->GetFileSize(full_path, &fsize);
      if (fs.ok()) {
        files->push_back({log_num, fsize, full_path});
      }
    }
  }
  return Status::OK();
}

Status WalManager::PurgeObsoleteWALFiles() {
  std::lock_guard<std::mutex> lock(mu_);

  uint64_t min_log_to_keep = 0;
  if (get_min_log_number_to_keep_) {
    min_log_to_keep = get_min_log_number_to_keep_();
  }
  uint64_t now_seconds = env_->NowMicros() / 1000000ULL;

  std::vector<WalFileInfo> files;
  Status s = ScanWalDir(&files);
  if (!s.ok()) return s;

  std::sort(files.begin(), files.end());

  // Compute total WAL size for size-limit purge.
  uint64_t total_size = 0;
  for (const auto& f : files) {
    total_size += f.size_bytes;
  }

  uint64_t size_limit_bytes =
      options_.WAL_size_limit_MB > 0
          ? options_.WAL_size_limit_MB * 1024ULL * 1024ULL
          : 0;

  for (const auto& f : files) {
    if (ShouldPurgeWal(f, min_log_to_keep, now_seconds, total_size,
                       size_limit_bytes)) {
      total_size -= f.size_bytes;
      Status ds = env_->DeleteFile(f.path);
      if (!ds.ok()) return ds;
    }
  }
  return Status::OK();
}

bool WalManager::ShouldPurgeWal(const WalFileInfo& info,
                                uint64_t min_log_to_keep,
                                uint64_t now_seconds,
                                uint64_t total_wal_size,
                                uint64_t size_limit_bytes) const {
  // Never purge files that are still needed for recovery.
  if (info.log_number >= min_log_to_keep) {
    return false;
  }

  // TTL-based purge.
  if (options_.WAL_ttl_seconds > 0) {
    uint64_t mtime = 0;
    Status s = env_->GetFileModificationTime(info.path, &mtime);
    if (!s.ok()) {
      return false;  // Conservative: when in doubt, keep.
    }
    if (now_seconds > mtime &&
        (now_seconds - mtime) > options_.WAL_ttl_seconds) {
      return true;
    }
    return false;  // Within TTL, keep.
  }

  // Size-based purge: delete oldest files until under the limit.
  if (size_limit_bytes > 0 && total_wal_size > size_limit_bytes) {
    return true;
  }

  // Default: purge if below min_log_to_keep (already checked above).
  return true;
}

Status WalManager::DeleteWalFile(const std::string& path) {
  return env_->DeleteFile(path);
}

}  // namespace mwal
