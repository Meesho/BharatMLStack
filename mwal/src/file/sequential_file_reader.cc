// Derived from RocksDB — buffered SequentialFileReader for mwal.

#include "file/sequential_file_reader.h"

namespace mwal {

IOStatus SequentialFileReader::Read(size_t n, Slice* result, char* scratch) {
  IOStatus s = file_->Read(n, result, scratch);
  if (s.ok()) {
    offset_ += result->size();
  }
  return s;
}

IOStatus SequentialFileReader::Skip(uint64_t n) {
  IOStatus s = file_->Skip(n);
  if (s.ok()) {
    offset_ += n;
  }
  return s;
}

}  // namespace mwal
