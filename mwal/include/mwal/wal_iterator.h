// Pull-based WAL record reader for mwal.

#pragma once

#include <memory>
#include <string>
#include <vector>

#include "mwal/status.h"
#include "mwal/types.h"
#include "mwal/wal_file_info.h"
#include "mwal/write_batch.h"

namespace mwal {

class Env;
struct WALOptions;

class WalIterator {
 public:
  ~WalIterator();

  bool Valid() const;
  void Next();

  const WriteBatch& GetBatch() const;
  SequenceNumber GetSequenceNumber() const;
  Status status() const;

 private:
  friend class DBWal;
  WalIterator(Env* env, const WALOptions& options,
              std::vector<WalFileInfo> files, SequenceNumber start_seq);

  void AdvanceToStartSeq();
  bool ReadNextRecord();
  bool OpenNextFile();

  struct Impl;
  std::unique_ptr<Impl> impl_;
};

}  // namespace mwal
