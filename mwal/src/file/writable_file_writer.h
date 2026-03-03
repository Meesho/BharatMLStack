// Derived from RocksDB — buffered WritableFileWriter for mwal.

#pragma once

#include <cstdint>
#include <memory>
#include <string>

#include "mwal/env.h"
#include "mwal/io_status.h"
#include "mwal/slice.h"

namespace mwal {

class WritableFileWriter {
 public:
  WritableFileWriter(std::unique_ptr<WritableFile>&& file,
                     const std::string& fname,
                     size_t buf_size = 65536)
      : writable_file_(std::move(file)),
        fname_(fname),
        buf_(new char[buf_size]),
        buf_size_(buf_size),
        pos_(0),
        filesize_(0),
        synced_(false) {}

  ~WritableFileWriter() {
    if (!synced_) {
      Flush();
    }
    delete[] buf_;
  }

  WritableFileWriter(const WritableFileWriter&) = delete;
  WritableFileWriter& operator=(const WritableFileWriter&) = delete;

  IOStatus Append(const Slice& data);
  IOStatus Flush();
  IOStatus Sync(bool use_fsync = false);
  IOStatus Close();

  uint64_t GetFileSize() const { return filesize_ + pos_; }
  WritableFile* writable_file() const { return writable_file_.get(); }
  const std::string& file_name() const { return fname_; }

 private:
  IOStatus WriteBuffered();

  std::unique_ptr<WritableFile> writable_file_;
  std::string fname_;
  char* buf_;
  size_t buf_size_;
  size_t pos_;
  uint64_t filesize_;
  bool synced_;
};

}  // namespace mwal
