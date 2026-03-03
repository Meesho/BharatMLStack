// Copyright (c) 2011-present, Facebook, Inc.  All rights reserved.
// Derived from RocksDB — WAL record writer for mwal.

#include "log/log_writer.h"

#include <cassert>
#include <cstdint>

#include "util/coding.h"
#include "util/crc32c.h"

namespace mwal {
namespace log {

Writer::Writer(std::unique_ptr<WritableFileWriter>&& dest, uint64_t log_number,
               bool recycle_log_files, bool manual_flush)
    : dest_(std::move(dest)),
      block_offset_(0),
      log_number_(log_number),
      recycle_log_files_(recycle_log_files),
      header_size_(recycle_log_files ? kRecyclableHeaderSize : kHeaderSize),
      manual_flush_(manual_flush) {
  for (uint8_t i = 0; i <= kMaxRecordType; i++) {
    char t = static_cast<char>(i);
    type_crc_[i] = crc32c::Value(&t, 1);
  }
}

Writer::~Writer() {
  if (dest_) {
    WriteBuffer();
  }
}

IOStatus Writer::WriteBuffer() {
  if (dest_) {
    return dest_->Flush();
  }
  return IOStatus::OK();
}

IOStatus Writer::Close() {
  IOStatus s;
  if (dest_) {
    s = dest_->Close();
    dest_.reset();
  }
  return s;
}

IOStatus Writer::AddRecord(const Slice& slice) {
  const char* ptr = slice.data();
  size_t left = slice.size();

  bool begin = true;
  IOStatus s;
  do {
    const int64_t leftover = kBlockSize - block_offset_;
    assert(leftover >= 0);
    if (leftover < header_size_) {
      if (leftover > 0) {
        assert(header_size_ <= 11);
        s = dest_->Append(
            Slice("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
                  static_cast<size_t>(leftover)));
        if (!s.ok()) break;
      }
      block_offset_ = 0;
    }

    assert(static_cast<int64_t>(kBlockSize - block_offset_) >= header_size_);

    const size_t avail = kBlockSize - block_offset_ - header_size_;
    const size_t fragment_length = (left < avail) ? left : avail;

    RecordType type;
    const bool end = (left == fragment_length);
    if (begin && end) {
      type = recycle_log_files_ ? kRecyclableFullType : kFullType;
    } else if (begin) {
      type = recycle_log_files_ ? kRecyclableFirstType : kFirstType;
    } else if (end) {
      type = recycle_log_files_ ? kRecyclableLastType : kLastType;
    } else {
      type = recycle_log_files_ ? kRecyclableMiddleType : kMiddleType;
    }

    s = EmitPhysicalRecord(type, ptr, fragment_length);
    ptr += fragment_length;
    left -= fragment_length;
    begin = false;
  } while (s.ok() && left > 0);

  if (s.ok() && !manual_flush_) {
    s = dest_->Flush();
  }
  return s;
}

IOStatus Writer::EmitPhysicalRecord(RecordType t, const char* ptr, size_t n) {
  assert(n <= 0xffff);

  size_t header_size;
  char buf[kRecyclableHeaderSize];

  buf[4] = static_cast<char>(n & 0xff);
  buf[5] = static_cast<char>(n >> 8);
  buf[6] = static_cast<char>(t);

  uint32_t crc = type_crc_[t];
  if (t < kRecyclableFullType) {
    assert(block_offset_ + kHeaderSize + n <= kBlockSize);
    header_size = kHeaderSize;
  } else {
    assert(block_offset_ + kRecyclableHeaderSize + n <= kBlockSize);
    header_size = kRecyclableHeaderSize;
    EncodeFixed32(buf + 7, static_cast<uint32_t>(log_number_));
    crc = crc32c::Extend(crc, buf + 7, 4);
  }

  crc = crc32c::Extend(crc, ptr, n);
  crc = crc32c::Mask(crc);
  EncodeFixed32(buf, crc);

  IOStatus s = dest_->Append(Slice(buf, header_size));
  if (s.ok()) {
    s = dest_->Append(Slice(ptr, n));
  }
  block_offset_ += header_size + n;
  return s;
}

}  // namespace log
}  // namespace mwal
