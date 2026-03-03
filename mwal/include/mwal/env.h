// Copyright (c) 2011-present, Facebook, Inc.  All rights reserved.
// Derived from RocksDB — minimal Env abstraction for mwal standalone WAL library.

#pragma once

#include <cstdint>
#include <memory>
#include <string>
#include <vector>

#include "mwal/io_status.h"
#include "mwal/slice.h"
#include "mwal/status.h"

namespace mwal {

class FileLock {
 public:
  virtual ~FileLock() = default;
  FileLock() = default;
  FileLock(const FileLock&) = delete;
  FileLock& operator=(const FileLock&) = delete;
};

class WritableFile {
 public:
  virtual ~WritableFile() = default;
  virtual IOStatus Append(const Slice& data) = 0;
  virtual IOStatus PositionedAppend(const Slice& /*data*/, uint64_t /*offset*/) {
    return IOStatus::NotSupported("PositionedAppend");
  }
  virtual IOStatus Close() = 0;
  virtual IOStatus Flush() = 0;
  virtual IOStatus Sync() = 0;
  virtual IOStatus Fsync() { return Sync(); }
  virtual uint64_t GetFileSize() const = 0;
  virtual IOStatus Truncate(uint64_t /*size*/) { return IOStatus::OK(); }
  virtual bool use_direct_io() const { return false; }
};

class SequentialFile {
 public:
  virtual ~SequentialFile() = default;
  virtual IOStatus Read(size_t n, Slice* result, char* scratch) = 0;
  virtual IOStatus Skip(uint64_t n) = 0;
  virtual bool use_direct_io() const { return false; }
};

struct EnvOptions {
  bool use_mmap_reads = false;
  bool use_mmap_writes = false;
  bool use_direct_reads = false;
  bool use_direct_writes = false;
  bool allow_fallocate = true;
  size_t writable_file_max_buffer_size = 1024 * 1024;
  bool fallocate_with_keep_size = true;
};

class Env {
 public:
  virtual ~Env() = default;

  static Env* Default();

  virtual Status NewWritableFile(const std::string& fname,
                                 std::unique_ptr<WritableFile>* result,
                                 const EnvOptions& options) = 0;
  virtual Status NewSequentialFile(const std::string& fname,
                                   std::unique_ptr<SequentialFile>* result,
                                   const EnvOptions& options) = 0;
  virtual Status DeleteFile(const std::string& fname) = 0;
  virtual Status RenameFile(const std::string& src,
                            const std::string& target) = 0;
  virtual Status GetChildren(const std::string& dir,
                             std::vector<std::string>* result) = 0;
  virtual Status FileExists(const std::string& fname) = 0;
  virtual Status GetFileSize(const std::string& fname, uint64_t* size) = 0;
  virtual Status GetFileModificationTime(const std::string& fname,
                                         uint64_t* file_mtime) = 0;
  virtual Status CreateDir(const std::string& dirname) = 0;
  virtual Status CreateDirIfMissing(const std::string& dirname) = 0;
  virtual uint64_t NowMicros() = 0;
  virtual uint64_t NowNanos() { return NowMicros() * 1000; }

  virtual Status LockFile(const std::string& fname, FileLock** lock) = 0;
  virtual Status UnlockFile(FileLock* lock) = 0;
};

}  // namespace mwal
