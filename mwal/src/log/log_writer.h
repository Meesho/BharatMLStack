// Copyright (c) 2011-present, Facebook, Inc.  All rights reserved.
// Derived from RocksDB — WAL record writer for mwal.

#pragma once

#include <cstdint>
#include <memory>

#include "file/writable_file_writer.h"
#include "log/log_format.h"
#include "mwal/io_status.h"
#include "mwal/slice.h"

namespace mwal {
namespace log {

class Writer {
 public:
  explicit Writer(std::unique_ptr<WritableFileWriter>&& dest,
                  uint64_t log_number, bool recycle_log_files,
                  bool manual_flush = false);

  Writer(const Writer&) = delete;
  void operator=(const Writer&) = delete;

  ~Writer();

  IOStatus AddRecord(const Slice& slice);
  IOStatus WriteBuffer();
  IOStatus Close();

  WritableFileWriter* file() { return dest_.get(); }
  const WritableFileWriter* file() const { return dest_.get(); }
  uint64_t get_log_number() const { return log_number_; }
  bool BufferIsEmpty() const { return dest_ == nullptr; }

  size_t TEST_block_offset() const { return block_offset_; }

 private:
  IOStatus EmitPhysicalRecord(RecordType type, const char* ptr, size_t length);

  std::unique_ptr<WritableFileWriter> dest_;
  size_t block_offset_;
  uint64_t log_number_;
  bool recycle_log_files_;
  int header_size_;
  bool manual_flush_;

  uint32_t type_crc_[kMaxRecordType + 1];
};

}  // namespace log
}  // namespace mwal
