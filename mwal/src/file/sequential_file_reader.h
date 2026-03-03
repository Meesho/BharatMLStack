// Derived from RocksDB — buffered SequentialFileReader for mwal.

#pragma once

#include <cstdint>
#include <memory>
#include <string>

#include "mwal/env.h"
#include "mwal/io_status.h"
#include "mwal/slice.h"

namespace mwal {

class SequentialFileReader {
 public:
  explicit SequentialFileReader(std::unique_ptr<SequentialFile>&& file,
                                const std::string& fname)
      : file_(std::move(file)), fname_(fname), offset_(0) {}

  SequentialFileReader(const SequentialFileReader&) = delete;
  SequentialFileReader& operator=(const SequentialFileReader&) = delete;

  IOStatus Read(size_t n, Slice* result, char* scratch);
  IOStatus Skip(uint64_t n);

  const std::string& file_name() const { return fname_; }
  SequentialFile* file() const { return file_.get(); }

 private:
  std::unique_ptr<SequentialFile> file_;
  std::string fname_;
  uint64_t offset_;
};

}  // namespace mwal
