// Derived from RocksDB — buffered WritableFileWriter for mwal.

#include "file/writable_file_writer.h"

#include <algorithm>
#include <cstring>

namespace mwal {

IOStatus WritableFileWriter::Append(const Slice& data) {
  const char* src = data.data();
  size_t left = data.size();

  while (left > 0) {
    size_t avail = buf_size_ - pos_;
    size_t copy = std::min(left, avail);

    if (copy > 0) {
      memcpy(buf_ + pos_, src, copy);
      pos_ += copy;
      src += copy;
      left -= copy;
    }

    if (pos_ == buf_size_) {
      IOStatus s = WriteBuffered();
      if (!s.ok()) return s;
    }
  }
  return IOStatus::OK();
}

IOStatus WritableFileWriter::Flush() {
  if (pos_ > 0) {
    IOStatus s = WriteBuffered();
    if (!s.ok()) return s;
  }
  return writable_file_->Flush();
}

IOStatus WritableFileWriter::Sync(bool use_fsync) {
  IOStatus s = Flush();
  if (!s.ok()) return s;

  if (use_fsync) {
    s = writable_file_->Fsync();
  } else {
    s = writable_file_->Sync();
  }
  synced_ = s.ok();
  return s;
}

IOStatus WritableFileWriter::Close() {
  IOStatus s = Flush();
  if (!s.ok()) return s;
  synced_ = true;
  return writable_file_->Close();
}

IOStatus WritableFileWriter::WriteBuffered() {
  if (pos_ == 0) return IOStatus::OK();

  IOStatus s = writable_file_->Append(Slice(buf_, pos_));
  if (s.ok()) {
    filesize_ += pos_;
    pos_ = 0;
  }
  return s;
}

}  // namespace mwal
