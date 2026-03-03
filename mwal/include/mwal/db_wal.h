// Top-level WAL API for mwal.
// Provides atomic writes, group commit, log rotation, crash recovery,
// sequence number management, and optional compression.

#pragma once

#include <atomic>
#include <condition_variable>
#include <cstdint>
#include <functional>
#include <memory>
#include <mutex>
#include <thread>
#include <vector>

#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/status.h"
#include "mwal/types.h"
#include "mwal/wal_file_info.h"
#include "mwal/write_batch.h"

namespace mwal {

namespace log {
class Writer;
}
class WalManager;
class WriteCoalescer;
class WriteThread;
class WalIterator;

class DBWal {
 public:
  ~DBWal();

  DBWal(const DBWal&) = delete;
  DBWal& operator=(const DBWal&) = delete;

  static Status Open(const WALOptions& options, Env* env,
                     std::unique_ptr<DBWal>* result);

  Status Write(const WriteOptions& options, WriteBatch* batch);

  // Replays all WAL records in order, calling callback for each batch.
  // After recovery, new writes continue from the latest sequence.
  Status Recover(
      std::function<Status(SequenceNumber seq, WriteBatch* batch)> callback);

  Status FlushWAL(bool sync);
  Status SyncWAL();
  Status Close();

  SequenceNumber GetLatestSequenceNumber() const;
  uint64_t GetCurrentLogNumber() const;

  // --- Public management APIs ---

  void SetMinLogNumberToKeep(uint64_t n);

  Status PurgeObsoleteFiles();

  Status GetLiveWalFiles(std::vector<WalFileInfo>* files);

  // Snapshot-based iterator starting from start_seq.
  Status NewWalIterator(SequenceNumber start_seq,
                        std::unique_ptr<WalIterator>* result);

 private:
  DBWal();

  Status NewLogFile();
  Status RotateLogFile();

  uint64_t TotalWalSize();

  // Callback for the WriteCoalescer: merges batches and writes a single record.
  Status WriteCoalescedBatches(std::vector<WriteBatch*>& batches);

  WALOptions options_;
  Env* env_ = nullptr;
  std::unique_ptr<WalManager> wal_manager_;
  std::unique_ptr<log::Writer> log_writer_;
  std::unique_ptr<WriteThread> write_thread_;
  std::unique_ptr<WriteCoalescer> write_coalescer_;
  std::atomic<SequenceNumber> last_sequence_{0};
  std::atomic<uint64_t> log_number_{0};
  std::mutex writer_mu_;
  bool closed_ = false;

  // A3: min log number safe to purge
  std::atomic<uint64_t> min_log_to_keep_{0};

  // A4: directory lock
  std::unique_ptr<FileLock> dir_lock_;

  // C5: background purge thread
  std::thread bg_purge_thread_;
  std::mutex bg_purge_mu_;
  std::condition_variable bg_purge_cv_;
  std::atomic<bool> shutdown_{false};
};

}  // namespace mwal
