// Copyright (c) 2011-present, Facebook, Inc.  All rights reserved.
// Derived from RocksDB — adapted for mwal standalone WAL library.

#pragma once

#include <algorithm>
#include <cassert>
#include <cstdint>
#include <cstring>
#include <memory>

namespace mwal {

inline size_t TruncateToPageBoundary(size_t page_size, size_t s) {
  s -= (s & (page_size - 1));
  assert((s % page_size) == 0);
  return s;
}

inline size_t Roundup(size_t x, size_t y) { return ((x + y - 1) / y) * y; }

inline size_t Rounddown(size_t x, size_t y) { return (x / y) * y; }

class AlignedBuffer {
  size_t alignment_;
  std::unique_ptr<char[]> buf_;
  size_t capacity_;
  size_t cursize_;
  char* bufstart_;

 public:
  AlignedBuffer()
      : alignment_(1), capacity_(0), cursize_(0), bufstart_(nullptr) {}

  AlignedBuffer(AlignedBuffer&& o) noexcept { *this = std::move(o); }

  AlignedBuffer& operator=(AlignedBuffer&& o) noexcept {
    alignment_ = o.alignment_;
    buf_ = std::move(o.buf_);
    capacity_ = o.capacity_;
    cursize_ = o.cursize_;
    bufstart_ = o.bufstart_;
    o.capacity_ = 0;
    o.cursize_ = 0;
    o.bufstart_ = nullptr;
    return *this;
  }

  AlignedBuffer(const AlignedBuffer&) = delete;
  AlignedBuffer& operator=(const AlignedBuffer&) = delete;

  static bool isAligned(const void* ptr, size_t alignment) {
    return reinterpret_cast<uintptr_t>(ptr) % alignment == 0;
  }

  size_t Alignment() const { return alignment_; }
  size_t Capacity() const { return capacity_; }
  size_t CurrentSize() const { return cursize_; }
  const char* BufferStart() const { return bufstart_; }
  char* BufferStart() { return bufstart_; }

  void Clear() { cursize_ = 0; }

  void Alignment(size_t alignment) {
    assert(alignment > 0);
    assert((alignment & (alignment - 1)) == 0);
    alignment_ = alignment;
  }

  void AllocateNewBuffer(size_t requested_capacity, bool copy_data = false,
                         uint64_t copy_offset = 0, size_t copy_len = 0) {
    assert(alignment_ > 0);
    assert((alignment_ & (alignment_ - 1)) == 0);

    copy_len = copy_len > 0 ? copy_len : cursize_;
    if (copy_data && requested_capacity < copy_len) {
      return;
    }

    size_t new_capacity = Roundup(requested_capacity, alignment_);
    char* new_buf = new char[new_capacity + alignment_];
    char* new_bufstart = reinterpret_cast<char*>(
        (reinterpret_cast<uintptr_t>(new_buf) + (alignment_ - 1)) &
        ~static_cast<uintptr_t>(alignment_ - 1));

    if (copy_data && bufstart_ != nullptr) {
      assert(copy_offset + copy_len <= cursize_);
      memcpy(new_bufstart, bufstart_ + copy_offset, copy_len);
      cursize_ = copy_len;
    } else {
      cursize_ = 0;
    }

    bufstart_ = new_bufstart;
    capacity_ = new_capacity;
    buf_.reset(new_buf);
  }

  size_t Append(const char* src, size_t append_size) {
    size_t buffer_remaining = capacity_ - cursize_;
    size_t to_copy = std::min(append_size, buffer_remaining);
    if (to_copy > 0) {
      memcpy(bufstart_ + cursize_, src, to_copy);
      cursize_ += to_copy;
    }
    return to_copy;
  }

  size_t Read(char* dest, size_t offset, size_t read_size) const {
    assert(offset < cursize_);
    size_t to_read = 0;
    if (offset < cursize_) {
      to_read = std::min(cursize_ - offset, read_size);
    }
    if (to_read > 0) {
      memcpy(dest, bufstart_ + offset, to_read);
    }
    return to_read;
  }

  void PadToAlignmentWith(int padding) {
    size_t total_size = Roundup(cursize_, alignment_);
    size_t pad_size = total_size - cursize_;
    if (pad_size > 0) {
      assert((pad_size + cursize_) <= capacity_);
      memset(bufstart_ + cursize_, padding, pad_size);
      cursize_ += pad_size;
    }
  }

  void RefitTail(size_t tail_offset, size_t tail_size) {
    if (tail_size > 0) {
      memmove(bufstart_, bufstart_ + tail_offset, tail_size);
    }
    cursize_ = tail_size;
  }

  char* Destination() { return bufstart_ + cursize_; }
  void Size(size_t cursize) { cursize_ = cursize; }
};

}  // namespace mwal
