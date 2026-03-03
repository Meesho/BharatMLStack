// Copyright (c) 2011-present, Facebook, Inc.  All rights reserved.
// Derived from RocksDB — WAL record reader for mwal.

#include "log/log_reader.h"

#include <cstdio>
#include <cstring>

#include "util/coding.h"
#include "util/crc32c.h"

namespace mwal {
namespace log {

Reader::Reporter::~Reporter() = default;

Reader::Reader(std::unique_ptr<SequentialFileReader>&& file, Reporter* reporter,
               bool checksum, uint64_t log_num)
    : file_(std::move(file)),
      reporter_(reporter),
      checksum_(checksum),
      backing_store_(new char[kBlockSize]),
      buffer_(),
      eof_(false),
      read_error_(false),
      eof_offset_(0),
      last_record_offset_(0),
      end_of_buffer_offset_(0),
      log_number_(log_num),
      recycled_(false),
      first_record_read_(false) {}

Reader::~Reader() { delete[] backing_store_; }

bool Reader::ReadRecord(Slice* record, std::string* scratch,
                        WALRecoveryMode wal_recovery_mode) {
  scratch->clear();
  record->clear();
  bool in_fragmented_record = false;
  uint64_t prospective_record_offset = 0;

  Slice fragment;
  for (;;) {
    uint64_t physical_record_offset = end_of_buffer_offset_ - buffer_.size();
    size_t drop_size = 0;
    const uint8_t record_type = ReadPhysicalRecord(&fragment, &drop_size);

    switch (record_type) {
      case kFullType:
      case kRecyclableFullType:
        if (in_fragmented_record && !scratch->empty()) {
          ReportCorruption(scratch->size(), "partial record without end(1)");
        }
        prospective_record_offset = physical_record_offset;
        scratch->clear();
        *record = fragment;
        last_record_offset_ = prospective_record_offset;
        first_record_read_ = true;
        return true;

      case kFirstType:
      case kRecyclableFirstType:
        if (in_fragmented_record && !scratch->empty()) {
          ReportCorruption(scratch->size(), "partial record without end(2)");
        }
        prospective_record_offset = physical_record_offset;
        scratch->assign(fragment.data(), fragment.size());
        in_fragmented_record = true;
        break;

      case kMiddleType:
      case kRecyclableMiddleType:
        if (!in_fragmented_record) {
          ReportCorruption(fragment.size(),
                           "missing start of fragmented record(1)");
        } else {
          scratch->append(fragment.data(), fragment.size());
        }
        break;

      case kLastType:
      case kRecyclableLastType:
        if (!in_fragmented_record) {
          ReportCorruption(fragment.size(),
                           "missing start of fragmented record(2)");
        } else {
          scratch->append(fragment.data(), fragment.size());
          *record = Slice(*scratch);
          last_record_offset_ = prospective_record_offset;
          first_record_read_ = true;
          return true;
        }
        break;

      case kBadHeader:
        if (wal_recovery_mode == WALRecoveryMode::kAbsoluteConsistency ||
            wal_recovery_mode == WALRecoveryMode::kPointInTimeRecovery) {
          ReportCorruption(drop_size, "truncated header");
        }
        [[fallthrough]];

      case kEof:
        if (in_fragmented_record) {
          if (wal_recovery_mode == WALRecoveryMode::kAbsoluteConsistency ||
              wal_recovery_mode == WALRecoveryMode::kPointInTimeRecovery) {
            ReportCorruption(scratch->size(),
                             "error reading trailing data (EOF)");
          }
          scratch->clear();
        } else if (!first_record_read_ &&
                   (wal_recovery_mode == WALRecoveryMode::kAbsoluteConsistency ||
                    wal_recovery_mode == WALRecoveryMode::kPointInTimeRecovery) &&
                   end_of_buffer_offset_ > 0) {
          ReportCorruption(drop_size > 0 ? drop_size : 1,
                           "truncated header or incomplete data at EOF");
        }
        return false;

      case kOldRecord:
        if (wal_recovery_mode !=
            WALRecoveryMode::kSkipAnyCorruptedRecords) {
          if (in_fragmented_record) {
            if (wal_recovery_mode == WALRecoveryMode::kAbsoluteConsistency ||
                wal_recovery_mode == WALRecoveryMode::kPointInTimeRecovery) {
              ReportCorruption(scratch->size(),
                               "error reading trailing data (old record)");
            }
            scratch->clear();
          }
          return false;
        }
        [[fallthrough]];

      case kBadRecord:
        if (in_fragmented_record) {
          ReportCorruption(scratch->size(), "error in middle of record");
          in_fragmented_record = false;
          scratch->clear();
        }
        break;

      case kBadRecordLen:
        if (eof_) {
          if (wal_recovery_mode == WALRecoveryMode::kAbsoluteConsistency ||
              wal_recovery_mode == WALRecoveryMode::kPointInTimeRecovery) {
            ReportCorruption(drop_size, "truncated record body");
          }
          return false;
        }
        [[fallthrough]];

      case kBadRecordChecksum:
        if (recycled_ && wal_recovery_mode ==
                             WALRecoveryMode::kTolerateCorruptedTailRecords) {
          scratch->clear();
          return false;
        }
        if (record_type == kBadRecordLen) {
          ReportCorruption(drop_size, "bad record length");
        } else {
          ReportCorruption(drop_size, "checksum mismatch");
        }
        if (in_fragmented_record) {
          ReportCorruption(scratch->size(), "error in middle of record");
          in_fragmented_record = false;
          scratch->clear();
        }
        if (wal_recovery_mode == WALRecoveryMode::kTolerateCorruptedTailRecords ||
            wal_recovery_mode == WALRecoveryMode::kPointInTimeRecovery) {
          return false;
        }
        break;

      default: {
        std::string reason = "unknown record type " + std::to_string(record_type);
        ReportCorruption(
            (fragment.size() + (in_fragmented_record ? scratch->size() : 0)),
            reason.c_str());
        in_fragmented_record = false;
        scratch->clear();
        break;
      }
    }
  }
}

uint64_t Reader::LastRecordOffset() const { return last_record_offset_; }

uint64_t Reader::LastRecordEnd() const {
  return end_of_buffer_offset_ - buffer_.size();
}

void Reader::UnmarkEOF() {
  if (read_error_) return;
  eof_ = false;
  if (eof_offset_ == 0) return;
  UnmarkEOFInternal();
}

void Reader::UnmarkEOFInternal() {
  size_t consumed_bytes = eof_offset_ - buffer_.size();
  size_t remaining = kBlockSize - eof_offset_;

  if (buffer_.data() != backing_store_ + consumed_bytes) {
    memmove(backing_store_ + consumed_bytes, buffer_.data(), buffer_.size());
  }

  Slice read_buffer;
  IOStatus status =
      file_->Read(remaining, &read_buffer, backing_store_ + eof_offset_);

  size_t added = read_buffer.size();
  end_of_buffer_offset_ += added;

  if (!status.ok()) {
    if (added > 0) {
      ReportDrop(added, Status::IOError("read error"));
    }
    read_error_ = true;
    return;
  }

  if (read_buffer.data() != backing_store_ + eof_offset_) {
    memmove(backing_store_ + eof_offset_, read_buffer.data(),
            read_buffer.size());
  }

  buffer_ = Slice(backing_store_ + consumed_bytes,
                  eof_offset_ + added - consumed_bytes);

  if (added < remaining) {
    eof_ = true;
    eof_offset_ += added;
  } else {
    eof_offset_ = 0;
  }
}

void Reader::ReportCorruption(size_t bytes, const char* reason) {
  ReportDrop(bytes, Status::Corruption(reason));
}

void Reader::ReportDrop(size_t bytes, const Status& reason) {
  if (reporter_ != nullptr) {
    reporter_->Corruption(bytes, reason);
  }
}

bool Reader::ReadMore(size_t* drop_size, uint8_t* error) {
  if (!eof_ && !read_error_) {
    buffer_.clear();
    IOStatus status = file_->Read(kBlockSize, &buffer_, backing_store_);
    end_of_buffer_offset_ += buffer_.size();
    if (!status.ok()) {
      buffer_.clear();
      ReportDrop(kBlockSize, Status::IOError("read error"));
      read_error_ = true;
      *error = kEof;
      return false;
    } else if (buffer_.size() < static_cast<size_t>(kBlockSize)) {
      eof_ = true;
      eof_offset_ = buffer_.size();
    }
    return true;
  } else {
    if (buffer_.size()) {
      *drop_size = buffer_.size();
      buffer_.clear();
      *error = kBadHeader;
      return false;
    }
    buffer_.clear();
    *error = kEof;
    return false;
  }
}

uint8_t Reader::ReadPhysicalRecord(Slice* result, size_t* drop_size) {
  while (true) {
    if (buffer_.size() < static_cast<size_t>(kHeaderSize)) {
      uint8_t r = kEof;
      if (!ReadMore(drop_size, &r)) {
        return r;
      }
      continue;
    }

    const char* header = buffer_.data();
    const uint32_t a = static_cast<uint32_t>(header[4]) & 0xff;
    const uint32_t b = static_cast<uint32_t>(header[5]) & 0xff;
    const uint8_t type = static_cast<uint8_t>(header[6]);
    const uint32_t length = a | (b << 8);
    int header_size = kHeaderSize;

    const bool is_recyclable_type =
        (type >= kRecyclableFullType && type <= kRecyclableLastType);
    if (is_recyclable_type) {
      header_size = kRecyclableHeaderSize;
      if (first_record_read_ && !recycled_) {
        return kBadRecord;
      }
      recycled_ = true;
      if (buffer_.size() < static_cast<size_t>(kRecyclableHeaderSize)) {
        uint8_t r = kEof;
        if (!ReadMore(drop_size, &r)) {
          return r;
        }
        continue;
      }
    }

    if (static_cast<size_t>(header_size) + length > buffer_.size()) {
      assert(buffer_.size() >= static_cast<size_t>(header_size));
      *drop_size = buffer_.size();
      buffer_.clear();
      return kBadRecordLen;
    }

    if (is_recyclable_type) {
      const uint32_t log_num = DecodeFixed32(header + 7);
      if (log_num != log_number_) {
        buffer_.remove_prefix(header_size + length);
        return kOldRecord;
      }
    }

    if (type == kZeroType && length == 0) {
      buffer_.clear();
      return kBadRecord;
    }

    if (checksum_) {
      uint32_t expected_crc = crc32c::Unmask(DecodeFixed32(header));
      uint32_t actual_crc =
          crc32c::Value(header + 6, length + header_size - 6);
      if (actual_crc != expected_crc) {
        *drop_size = buffer_.size();
        buffer_.clear();
        return kBadRecordChecksum;
      }
    }

    buffer_.remove_prefix(header_size + length);
    *result = Slice(header + header_size, length);
    return type;
  }
}

// --- FragmentBufferedReader ---

bool FragmentBufferedReader::ReadRecord(Slice* record, std::string* scratch,
                                        WALRecoveryMode /*wal_recovery_mode*/) {
  assert(record != nullptr);
  assert(scratch != nullptr);
  record->clear();
  scratch->clear();

  uint64_t prospective_record_offset = 0;
  uint64_t physical_record_offset = end_of_buffer_offset_ - buffer_.size();
  size_t drop_size = 0;
  uint8_t fragment_type_or_err = 0;
  Slice fragment;

  while (TryReadFragment(&fragment, &drop_size, &fragment_type_or_err)) {
    switch (fragment_type_or_err) {
      case kFullType:
      case kRecyclableFullType:
        if (in_fragmented_record_ && !fragments_.empty()) {
          ReportCorruption(fragments_.size(), "partial record without end(1)");
        }
        fragments_.clear();
        *record = fragment;
        prospective_record_offset = physical_record_offset;
        last_record_offset_ = prospective_record_offset;
        first_record_read_ = true;
        in_fragmented_record_ = false;
        return true;

      case kFirstType:
      case kRecyclableFirstType:
        if (in_fragmented_record_ || !fragments_.empty()) {
          ReportCorruption(fragments_.size(), "partial record without end(2)");
        }
        prospective_record_offset = physical_record_offset;
        fragments_.assign(fragment.data(), fragment.size());
        in_fragmented_record_ = true;
        break;

      case kMiddleType:
      case kRecyclableMiddleType:
        if (!in_fragmented_record_) {
          ReportCorruption(fragment.size(),
                           "missing start of fragmented record(1)");
        } else {
          fragments_.append(fragment.data(), fragment.size());
        }
        break;

      case kLastType:
      case kRecyclableLastType:
        if (!in_fragmented_record_) {
          ReportCorruption(fragment.size(),
                           "missing start of fragmented record(2)");
        } else {
          fragments_.append(fragment.data(), fragment.size());
          scratch->assign(fragments_.data(), fragments_.size());
          fragments_.clear();
          *record = Slice(*scratch);
          last_record_offset_ = prospective_record_offset;
          first_record_read_ = true;
          in_fragmented_record_ = false;
          return true;
        }
        break;

      case kBadHeader:
      case kBadRecord:
      case kEof:
      case kOldRecord:
        if (in_fragmented_record_) {
          ReportCorruption(fragments_.size(), "error in middle of record");
          in_fragmented_record_ = false;
          fragments_.clear();
        }
        break;

      case kBadRecordChecksum:
        if (recycled_) {
          fragments_.clear();
          return false;
        }
        ReportCorruption(drop_size, "checksum mismatch");
        if (in_fragmented_record_) {
          ReportCorruption(fragments_.size(), "error in middle of record");
          in_fragmented_record_ = false;
          fragments_.clear();
        }
        break;

      default: {
        std::string reason =
            "unknown record type " + std::to_string(fragment_type_or_err);
        ReportCorruption(
            fragment.size() + (in_fragmented_record_ ? fragments_.size() : 0),
            reason.c_str());
        in_fragmented_record_ = false;
        fragments_.clear();
        break;
      }
    }
  }
  return false;
}

void FragmentBufferedReader::UnmarkEOF() {
  if (read_error_) return;
  eof_ = false;
  UnmarkEOFInternal();
}

bool FragmentBufferedReader::TryReadMore(size_t* drop_size, uint8_t* error) {
  if (!eof_ && !read_error_) {
    buffer_.clear();
    IOStatus status = file_->Read(kBlockSize, &buffer_, backing_store_);
    end_of_buffer_offset_ += buffer_.size();
    if (!status.ok()) {
      buffer_.clear();
      ReportDrop(kBlockSize, Status::IOError("read error"));
      read_error_ = true;
      *error = kEof;
      return false;
    } else if (buffer_.size() < static_cast<size_t>(kBlockSize)) {
      eof_ = true;
      eof_offset_ = buffer_.size();
    }
    return true;
  } else if (!read_error_) {
    UnmarkEOF();
  }
  if (!read_error_) {
    return true;
  }
  *error = kEof;
  *drop_size = buffer_.size();
  if (buffer_.size() > 0) {
    *error = kBadHeader;
  }
  buffer_.clear();
  return false;
}

bool FragmentBufferedReader::TryReadFragment(Slice* fragment,
                                              size_t* drop_size,
                                              uint8_t* fragment_type_or_err) {
  while (buffer_.size() < static_cast<size_t>(kHeaderSize)) {
    size_t old_size = buffer_.size();
    uint8_t error = kEof;
    if (!TryReadMore(drop_size, &error)) {
      *fragment_type_or_err = error;
      return false;
    } else if (old_size == buffer_.size()) {
      return false;
    }
  }

  const char* header = buffer_.data();
  const uint32_t a = static_cast<uint32_t>(header[4]) & 0xff;
  const uint32_t b = static_cast<uint32_t>(header[5]) & 0xff;
  const uint8_t type = static_cast<uint8_t>(header[6]);
  const uint32_t length = a | (b << 8);
  int header_size = kHeaderSize;

  if (type >= kRecyclableFullType && type <= kRecyclableLastType) {
    if (first_record_read_ && !recycled_) {
      *fragment_type_or_err = kBadRecord;
      return true;
    }
    recycled_ = true;
    header_size = kRecyclableHeaderSize;
    while (buffer_.size() < static_cast<size_t>(kRecyclableHeaderSize)) {
      size_t old_size = buffer_.size();
      uint8_t error = kEof;
      if (!TryReadMore(drop_size, &error)) {
        *fragment_type_or_err = error;
        return false;
      } else if (old_size == buffer_.size()) {
        return false;
      }
    }
    const uint32_t log_num = DecodeFixed32(header + 7);
    if (log_num != log_number_) {
      *fragment_type_or_err = kOldRecord;
      return true;
    }
  }

  while (static_cast<size_t>(header_size) + length > buffer_.size()) {
    size_t old_size = buffer_.size();
    uint8_t error = kEof;
    if (!TryReadMore(drop_size, &error)) {
      *fragment_type_or_err = error;
      return false;
    } else if (old_size == buffer_.size()) {
      return false;
    }
  }

  if (type == kZeroType && length == 0) {
    buffer_.clear();
    *fragment_type_or_err = kBadRecord;
    return true;
  }

  if (checksum_) {
    uint32_t expected_crc = crc32c::Unmask(DecodeFixed32(header));
    uint32_t actual_crc = crc32c::Value(header + 6, length + header_size - 6);
    if (actual_crc != expected_crc) {
      *drop_size = buffer_.size();
      buffer_.clear();
      *fragment_type_or_err = kBadRecordChecksum;
      return true;
    }
  }

  buffer_.remove_prefix(header_size + length);
  *fragment = Slice(header + header_size, length);
  *fragment_type_or_err = type;
  return true;
}

}  // namespace log
}  // namespace mwal
