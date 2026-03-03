// Top-level WAL API implementation for mwal.

#include "mwal/db_wal.h"

#include <cassert>
#include <chrono>
#include <cstdio>
#include <string>
#include <vector>

#include "file/sequential_file_reader.h"
#include "file/writable_file_writer.h"
#include "log/log_reader.h"
#include "log/log_writer.h"
#include "mwal/wal_iterator.h"
#include "wal/wal_compressor.h"
#include "wal/wal_manager.h"
#include "wal/write_coalescer.h"
#include "wal/write_thread.h"

namespace mwal {

namespace {

std::string MakeWalPath(const std::string& dir, uint64_t log_num) {
  char buf[64];
  snprintf(buf, sizeof(buf), "%06llu.log", (unsigned long long)log_num);
  return dir + "/" + buf;
}

class RecoveryReporter : public log::Reader::Reporter {
 public:
  Status first_corruption;
  void Corruption(size_t /*bytes*/, const Status& s) override {
    if (first_corruption.ok()) first_corruption = s;
  }
};

Status IOToStatus(const IOStatus& ios) {
  if (ios.ok()) return Status::OK();
  Slice msg = ios.getState() ? Slice(ios.getState()) : Slice("IO error");
  switch (ios.code()) {
    case Status::kIOError: return Status::IOError(msg);
    case Status::kCorruption: return Status::Corruption(msg);
    case Status::kNotSupported: return Status::NotSupported(msg);
    default: return Status::IOError(msg);
  }
}

}  // namespace

DBWal::DBWal() = default;

DBWal::~DBWal() {
  if (!closed_) {
    Close();
  }
}

Status DBWal::Open(const WALOptions& options, Env* env,
                   std::unique_ptr<DBWal>* result) {
  if (!env) return Status::InvalidArgument("Env must not be null");
  if (options.wal_dir.empty()) {
    return Status::InvalidArgument("wal_dir must not be empty");
  }

  Status s = env->CreateDirIfMissing(options.wal_dir);
  if (!s.ok()) return s;

  // A4: acquire directory lock
  std::unique_ptr<FileLock> dir_lock;
  {
    std::string lock_path = options.wal_dir + "/LOCK";
    FileLock* raw_lock = nullptr;
    s = env->LockFile(lock_path, &raw_lock);
    if (!s.ok()) return s;
    dir_lock.reset(raw_lock);
  }

  std::unique_ptr<DBWal> wal(new DBWal());
  wal->options_ = options;
  wal->env_ = env;
  wal->dir_lock_ = std::move(dir_lock);
  wal->write_thread_ = std::make_unique<WriteThread>();

  // A3: wire min_log lambda to the atomic
  auto* raw_wal = wal.get();
  auto min_log_fn = [raw_wal]() -> uint64_t {
    return raw_wal->min_log_to_keep_.load(std::memory_order_acquire);
  };
  wal->wal_manager_ =
      std::make_unique<WalManager>(env, options, std::move(min_log_fn));

  // Discover existing WAL files to avoid overwriting them.
  std::vector<WalFileInfo> existing;
  s = wal->wal_manager_->GetSortedWalFiles(&existing);
  if (s.ok() && !existing.empty()) {
    wal->log_number_.store(existing.back().log_number,
                           std::memory_order_relaxed);
  }

  // B5: auto-recovery if callback is set and WAL files exist
  if (s.ok() && options.recovery_callback && !existing.empty()) {
    s = wal->Recover(options.recovery_callback);
    if (!s.ok()) return s;
    // Recover() already creates a new log file; skip NewLogFile below.
  } else if (s.ok()) {
    s = wal->NewLogFile();
    if (!s.ok()) return s;
  }

  // Start async write coalescer if configured
  if (options.max_async_queue_depth > 0) {
    wal->write_coalescer_ = std::make_unique<WriteCoalescer>(
        options.max_async_queue_depth,
        std::chrono::milliseconds(options.max_async_flush_interval_ms));
    auto* wal_ptr = wal.get();
    wal->write_coalescer_->Start(
        [wal_ptr](std::vector<WriteBatch*>& batches) -> Status {
          return wal_ptr->WriteCoalescedBatches(batches);
        });
  }

  // C5: start background purge thread if configured
  if (options.purge_interval_seconds > 0) {
    wal->shutdown_.store(false, std::memory_order_relaxed);
    auto* wal_ptr = wal.get();
    wal->bg_purge_thread_ = std::thread([wal_ptr]() {
      uint64_t interval = wal_ptr->options_.purge_interval_seconds;
      std::unique_lock<std::mutex> lock(wal_ptr->bg_purge_mu_);
      while (!wal_ptr->shutdown_.load(std::memory_order_acquire)) {
        wal_ptr->bg_purge_cv_.wait_for(
            lock, std::chrono::seconds(interval),
            [wal_ptr] {
              return wal_ptr->shutdown_.load(std::memory_order_acquire);
            });
        if (wal_ptr->shutdown_.load(std::memory_order_acquire)) break;
        wal_ptr->PurgeObsoleteFiles();
      }
    });
  }

  *result = std::move(wal);
  return Status::OK();
}

Status DBWal::NewLogFile() {
  uint64_t new_log_num = log_number_.load(std::memory_order_relaxed) + 1;
  std::string path = MakeWalPath(options_.wal_dir, new_log_num);

  std::unique_ptr<WritableFile> file;
  Status s = env_->NewWritableFile(path, &file, EnvOptions());
  if (!s.ok()) return s;

  auto writer_file =
      std::make_unique<WritableFileWriter>(std::move(file), path);
  log_writer_ = std::make_unique<log::Writer>(std::move(writer_file),
                                               new_log_num, /*recycle=*/false,
                                               options_.manual_wal_flush);
  log_number_.store(new_log_num, std::memory_order_release);
  return Status::OK();
}

Status DBWal::RotateLogFile() {
  if (log_writer_) {
    IOStatus ios = log_writer_->WriteBuffer();
    if (!ios.ok()) return IOToStatus(ios);
    ios = log_writer_->Close();
    if (!ios.ok()) return IOToStatus(ios);
    log_writer_.reset();
  }
  return NewLogFile();
}

uint64_t DBWal::TotalWalSize() {
  std::vector<WalFileInfo> files;
  if (!wal_manager_->GetSortedWalFiles(&files).ok()) return 0;
  uint64_t total = 0;
  for (const auto& f : files) total += f.size_bytes;
  return total;
}

Status DBWal::Write(const WriteOptions& options, WriteBatch* batch) {
  if (closed_) return Status::Aborted("WAL is closed");
  if (!batch) return Status::InvalidArgument("batch is null");

  if (options.disableWAL) {
    SequenceNumber seq =
        last_sequence_.fetch_add(batch->Count(), std::memory_order_relaxed) + 1;
    batch->SetSequence(seq);
    return Status::OK();
  }

  // C6: back-pressure on total WAL size
  if (options_.max_total_wal_size > 0 &&
      TotalWalSize() > options_.max_total_wal_size) {
    return Status::Busy("write stall: total WAL size exceeded");
  }

  // Hybrid routing: async writes go through the coalescer (if enabled)
  if (!options.sync && write_coalescer_) {
    return write_coalescer_->Submit(batch);
  }

  // Sync writes (or all writes when coalescer is disabled) use group commit
  WriteThread::Writer w;
  w.batch = batch;
  w.sync = options.sync;
  w.disable_wal = options.disableWAL;

  WriteThread::WriteGroup group;
  bool is_leader = write_thread_->JoinAndBuildGroup(
      &w, &group, options_.max_write_group_size);
  if (!is_leader) {
    return w.status;
  }

  Status s;
  {
    std::lock_guard<std::mutex> lock(writer_mu_);

    if (closed_) {
      s = Status::Aborted("WAL is closed");
      write_thread_->ExitAsBatchGroupLeader(group, s);
      return s;
    }

    SequenceNumber first_seq =
        last_sequence_.load(std::memory_order_relaxed) + 1;
    uint32_t total_count = 0;
    bool single_writer = (group.size == 1);

    WriteBatch merged;
    Slice record_data;

    if (single_writer) {
      WriteBatch* leader_batch = group.leader->batch;
      if (leader_batch && !group.leader->disable_wal) {
        total_count = static_cast<uint32_t>(leader_batch->Count());
        leader_batch->SetSequence(first_seq);
        record_data = Slice(leader_batch->Data());
      }
    } else {
      WriteThread::Writer* cur = group.leader;
      while (cur != nullptr) {
        if (cur->batch && !cur->disable_wal) {
          if (total_count == 0) {
            WriteBatchInternal::SetContents(
                &merged, WriteBatchInternal::Contents(cur->batch));
          } else {
            WriteBatchInternal::Append(&merged, cur->batch);
          }
          total_count += static_cast<uint32_t>(cur->batch->Count());
        }
        cur = cur->link_newer;
      }
      if (total_count > 0) {
        merged.SetSequence(first_seq);
        record_data = Slice(merged.Data());
      }
    }

    if (total_count > 0) {
      last_sequence_.fetch_add(total_count, std::memory_order_relaxed);

      std::string compressed;
      Status cs = WalCompressor::Compress(options_.wal_compression,
                                          record_data, &compressed);
      if (cs.ok()) {
        record_data = Slice(compressed);
      }

      IOStatus ios = log_writer_->AddRecord(record_data);
      if (!ios.ok()) {
        s = IOToStatus(ios);
      }

      if (s.ok() && !single_writer) {
        SequenceNumber seq = first_seq;
        WriteThread::Writer* cur = group.leader;
        while (cur != nullptr) {
          if (cur->batch && !cur->disable_wal) {
            cur->batch->SetSequence(seq);
            seq += static_cast<SequenceNumber>(cur->batch->Count());
          }
          cur = cur->link_newer;
        }
      }
    }

    // Two-phase exit: release async followers before fsync so they
    // don't pay the durability cost they didn't request.
    if (s.ok() && group.need_sync && group.size > 1) {
      write_thread_->CompleteAsyncFollowers(group, s);
    }

    if (s.ok() && group.need_sync && log_writer_) {
      IOStatus ios = log_writer_->file()->Sync();
      if (!ios.ok()) {
        s = IOToStatus(ios);
      }
    }

    if (s.ok() && options_.max_wal_file_size > 0 && log_writer_ &&
        log_writer_->file()->GetFileSize() > options_.max_wal_file_size) {
      Status rs = RotateLogFile();
      if (!rs.ok()) s = rs;
    }
  }

  write_thread_->ExitAsBatchGroupLeader(group, s);
  return s;
}

Status DBWal::Close() {
  if (closed_) return Status::OK();

  // Stop async write coalescer before draining group commit
  if (write_coalescer_) write_coalescer_->Stop();

  // A1: signal write thread shutdown first
  write_thread_->DrainAndClose();

  // C5: stop background purge thread
  shutdown_.store(true, std::memory_order_release);
  bg_purge_cv_.notify_all();
  if (bg_purge_thread_.joinable()) {
    bg_purge_thread_.join();
  }

  // A1: acquire writer_mu_ to safely close the log writer
  std::lock_guard<std::mutex> lock(writer_mu_);
  closed_ = true;

  Status s;
  if (log_writer_) {
    IOStatus ios = log_writer_->WriteBuffer();
    if (!ios.ok()) s = IOToStatus(ios);
    ios = log_writer_->Close();
    if (!ios.ok() && s.ok()) s = IOToStatus(ios);
    log_writer_.reset();
  }

  // A4: release directory lock
  if (dir_lock_) {
    env_->UnlockFile(dir_lock_.release());
  }

  return s;
}

Status DBWal::FlushWAL(bool sync) {
  std::lock_guard<std::mutex> lock(writer_mu_);
  if (!log_writer_) return Status::OK();

  IOStatus ios = log_writer_->WriteBuffer();
  if (!ios.ok()) return IOToStatus(ios);
  if (sync) {
    ios = log_writer_->file()->Sync();
    if (!ios.ok()) return IOToStatus(ios);
  }
  return Status::OK();
}

Status DBWal::SyncWAL() { return FlushWAL(true); }

SequenceNumber DBWal::GetLatestSequenceNumber() const {
  return last_sequence_.load(std::memory_order_acquire);
}

uint64_t DBWal::GetCurrentLogNumber() const {
  return log_number_.load(std::memory_order_acquire);
}

// --- B1 ---
void DBWal::SetMinLogNumberToKeep(uint64_t n) {
  min_log_to_keep_.store(n, std::memory_order_release);
}

// --- B2 ---
Status DBWal::PurgeObsoleteFiles() {
  return wal_manager_->PurgeObsoleteWALFiles();
}

// --- B3 ---
Status DBWal::GetLiveWalFiles(std::vector<WalFileInfo>* files) {
  return wal_manager_->GetSortedWalFiles(files);
}

// --- B4 ---
Status DBWal::NewWalIterator(SequenceNumber start_seq,
                             std::unique_ptr<WalIterator>* result) {
  std::vector<WalFileInfo> files;
  Status s = wal_manager_->GetSortedWalFiles(&files);
  if (!s.ok()) return s;
  result->reset(
      new WalIterator(env_, options_, std::move(files), start_seq));
  return Status::OK();
}

Status DBWal::WriteCoalescedBatches(std::vector<WriteBatch*>& batches) {
  std::lock_guard<std::mutex> lock(writer_mu_);
  if (closed_) return Status::Aborted("WAL is closed");

  SequenceNumber first_seq =
      last_sequence_.load(std::memory_order_relaxed) + 1;
  uint32_t total_count = 0;

  WriteBatch merged;
  Slice record_data;

  if (batches.size() == 1) {
    WriteBatch* b = batches[0];
    total_count = static_cast<uint32_t>(b->Count());
    b->SetSequence(first_seq);
    record_data = Slice(b->Data());
  } else {
    for (auto* b : batches) {
      if (total_count == 0) {
        WriteBatchInternal::SetContents(&merged,
                                        WriteBatchInternal::Contents(b));
      } else {
        WriteBatchInternal::Append(&merged, b);
      }
      total_count += static_cast<uint32_t>(b->Count());
    }
    if (total_count > 0) {
      merged.SetSequence(first_seq);
      record_data = Slice(merged.Data());
    }
  }

  if (total_count == 0) return Status::OK();

  last_sequence_.fetch_add(total_count, std::memory_order_relaxed);

  std::string compressed;
  Status cs =
      WalCompressor::Compress(options_.wal_compression, record_data, &compressed);
  if (cs.ok()) {
    record_data = Slice(compressed);
  }

  IOStatus ios = log_writer_->AddRecord(record_data);
  if (!ios.ok()) return IOToStatus(ios);

  // Assign individual sequences back to each batch
  if (batches.size() > 1) {
    SequenceNumber seq = first_seq;
    for (auto* b : batches) {
      b->SetSequence(seq);
      seq += static_cast<SequenceNumber>(b->Count());
    }
  }

  if (options_.max_wal_file_size > 0 && log_writer_ &&
      log_writer_->file()->GetFileSize() > options_.max_wal_file_size) {
    Status rs = RotateLogFile();
    if (!rs.ok()) return rs;
  }

  return Status::OK();
}

Status DBWal::Recover(
    std::function<Status(SequenceNumber seq, WriteBatch* batch)> callback) {
  std::vector<WalFileInfo> files;
  Status s = wal_manager_->GetSortedWalFiles(&files);
  if (!s.ok()) return s;

  SequenceNumber max_seq = 0;

  for (const auto& f : files) {
    std::unique_ptr<SequentialFile> file;
    s = env_->NewSequentialFile(f.path, &file, EnvOptions());
    if (!s.ok()) return s;

    auto file_reader =
        std::make_unique<SequentialFileReader>(std::move(file), f.path);

    RecoveryReporter reporter;
    log::Reader reader(std::move(file_reader), &reporter, true, f.log_number);

    Slice record;
    std::string scratch;
    size_t records_from_file = 0;
    while (reader.ReadRecord(&record, &scratch, options_.wal_recovery_mode)) {
      records_from_file++;
      std::string decompressed;
      Slice payload;
      s = WalDecompressor::Decompress(record, &decompressed, &payload);
      if (!s.ok()) return s;

      WriteBatch batch;
      WriteBatchInternal::SetContents(&batch, payload);

      SequenceNumber seq = batch.Sequence();
      uint32_t count = batch.Count();
      SequenceNumber batch_end = (count > 0) ? seq + count - 1 : seq;
      if (batch_end > max_seq) {
        max_seq = batch_end;
      }

      if (callback) {
        s = callback(seq, &batch);
        if (!s.ok()) return s;
      }
    }

    if (!reporter.first_corruption.ok() &&
        options_.wal_recovery_mode == WALRecoveryMode::kAbsoluteConsistency) {
      return reporter.first_corruption;
    }

    // Non-empty file with zero records and no corruption report: treat as
    // truncated/corrupt for kAbsoluteConsistency.
    if (f.size_bytes > 0 && records_from_file == 0 &&
        reporter.first_corruption.ok() &&
        options_.wal_recovery_mode == WALRecoveryMode::kAbsoluteConsistency) {
      return Status::Corruption("truncated or corrupt log file");
    }
  }

  last_sequence_.store(max_seq, std::memory_order_release);

  if (!files.empty()) {
    log_number_.store(files.back().log_number, std::memory_order_release);
  }
  if (log_writer_) {
    log_writer_->Close();
    log_writer_.reset();
  }
  return NewLogFile();
}

}  // namespace mwal
