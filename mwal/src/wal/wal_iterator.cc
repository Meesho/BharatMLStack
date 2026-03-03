// Pull-based WAL record reader for mwal.

#include "mwal/wal_iterator.h"

#include <cassert>
#include <memory>
#include <string>
#include <vector>

#include "file/sequential_file_reader.h"
#include "log/log_reader.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "wal/wal_compressor.h"
#include "wal/wal_manager.h"

namespace mwal {

namespace {
class SilentReporter : public log::Reader::Reporter {
 public:
  Status first_error;
  void Corruption(size_t /*bytes*/, const Status& s) override {
    if (first_error.ok()) first_error = s;
  }
};
}  // namespace

struct WalIterator::Impl {
  Env* env;
  WALOptions options;
  std::vector<WalFileInfo> files;
  SequenceNumber start_seq;

  size_t file_idx = 0;
  std::unique_ptr<log::Reader> reader;
  std::unique_ptr<SilentReporter> reporter;

  bool valid = false;
  Status status_;
  WriteBatch current_batch;
  SequenceNumber current_seq = 0;

  std::string scratch;
};

WalIterator::WalIterator(Env* env, const WALOptions& options,
                         std::vector<WalFileInfo> files,
                         SequenceNumber start_seq) {
  impl_ = std::make_unique<Impl>();
  impl_->env = env;
  impl_->options = options;
  impl_->files = std::move(files);
  impl_->start_seq = start_seq;

  AdvanceToStartSeq();
}

WalIterator::~WalIterator() = default;

bool WalIterator::Valid() const { return impl_->valid; }

void WalIterator::Next() {
  if (!impl_->valid) return;
  impl_->valid = ReadNextRecord();
  while (impl_->valid && impl_->current_seq < impl_->start_seq) {
    impl_->valid = ReadNextRecord();
  }
}

const WriteBatch& WalIterator::GetBatch() const { return impl_->current_batch; }

SequenceNumber WalIterator::GetSequenceNumber() const {
  return impl_->current_seq;
}

Status WalIterator::status() const { return impl_->status_; }

void WalIterator::AdvanceToStartSeq() {
  impl_->valid = ReadNextRecord();
  while (impl_->valid && impl_->current_seq < impl_->start_seq) {
    impl_->valid = ReadNextRecord();
  }
}

bool WalIterator::OpenNextFile() {
  while (impl_->file_idx < impl_->files.size()) {
    const auto& f = impl_->files[impl_->file_idx];
    impl_->file_idx++;

    std::unique_ptr<SequentialFile> file;
    Status s = impl_->env->NewSequentialFile(f.path, &file, EnvOptions());
    if (!s.ok()) {
      impl_->status_ = s;
      return false;
    }

    auto file_reader =
        std::make_unique<SequentialFileReader>(std::move(file), f.path);
    impl_->reporter = std::make_unique<SilentReporter>();
    impl_->reader = std::make_unique<log::Reader>(
        std::move(file_reader), impl_->reporter.get(), true, f.log_number);
    return true;
  }
  return false;
}

bool WalIterator::ReadNextRecord() {
  for (;;) {
    if (!impl_->reader) {
      if (!OpenNextFile()) return false;
    }

    Slice record;
    bool ok = impl_->reader->ReadRecord(&record, &impl_->scratch,
                                        impl_->options.wal_recovery_mode);
    if (!ok) {
      if (!impl_->reporter->first_error.ok() &&
          impl_->options.wal_recovery_mode ==
              WALRecoveryMode::kAbsoluteConsistency) {
        impl_->status_ = impl_->reporter->first_error;
        return false;
      }
      impl_->reader.reset();
      impl_->reporter.reset();
      if (!OpenNextFile()) {
        if (impl_->options.wal_recovery_mode ==
            WALRecoveryMode::kAbsoluteConsistency) {
          impl_->status_ = Status::Corruption("truncated or corrupt log file");
        }
        return false;
      }
      continue;
    }

    std::string decompressed;
    Slice payload;
    Status s = WalDecompressor::Decompress(record, &decompressed, &payload);
    if (!s.ok()) {
      impl_->status_ = s;
      return false;
    }

    WriteBatchInternal::SetContents(&impl_->current_batch, payload);
    impl_->current_seq = impl_->current_batch.Sequence();
    return true;
  }
}

}  // namespace mwal
