// Copyright (c) 2011-present, Facebook, Inc.  All rights reserved.
// Derived from RocksDB — WAL record reader for mwal.

#pragma once

#include <cstdint>
#include <memory>
#include <string>

#include "file/sequential_file_reader.h"
#include "log/log_format.h"
#include "mwal/options.h"
#include "mwal/slice.h"
#include "mwal/status.h"

namespace mwal {
namespace log {

class Reader {
 public:
  class Reporter {
   public:
    virtual ~Reporter();
    virtual void Corruption(size_t bytes, const Status& status) = 0;
  };

  Reader(std::unique_ptr<SequentialFileReader>&& file, Reporter* reporter,
         bool checksum, uint64_t log_num);

  Reader(const Reader&) = delete;
  void operator=(const Reader&) = delete;

  virtual ~Reader();

  virtual bool ReadRecord(Slice* record, std::string* scratch,
                          WALRecoveryMode wal_recovery_mode =
                              WALRecoveryMode::kTolerateCorruptedTailRecords);

  uint64_t LastRecordOffset() const;
  uint64_t LastRecordEnd() const;

  bool IsEOF() const { return eof_; }
  bool hasReadError() const { return read_error_; }
  virtual void UnmarkEOF();

  SequentialFileReader* file() { return file_.get(); }
  uint64_t GetLogNumber() const { return log_number_; }

 protected:
  const std::unique_ptr<SequentialFileReader> file_;
  Reporter* const reporter_;
  bool const checksum_;
  char* const backing_store_;

  Slice buffer_;
  bool eof_;
  bool read_error_;
  size_t eof_offset_;

  uint64_t last_record_offset_;
  uint64_t end_of_buffer_offset_;
  uint64_t const log_number_;

  bool recycled_;
  bool first_record_read_;

  enum : uint8_t {
    kEof = kMaxRecordType + 1,
    kBadRecord = kMaxRecordType + 2,
    kBadHeader = kMaxRecordType + 3,
    kOldRecord = kMaxRecordType + 4,
    kBadRecordLen = kMaxRecordType + 5,
    kBadRecordChecksum = kMaxRecordType + 6,
  };

  uint8_t ReadPhysicalRecord(Slice* result, size_t* drop_size);
  bool ReadMore(size_t* drop_size, uint8_t* error);
  void UnmarkEOFInternal();

  void ReportCorruption(size_t bytes, const char* reason);
  void ReportDrop(size_t bytes, const Status& reason);
};

class FragmentBufferedReader : public Reader {
 public:
  FragmentBufferedReader(std::unique_ptr<SequentialFileReader>&& file,
                         Reporter* reporter, bool checksum, uint64_t log_num)
      : Reader(std::move(file), reporter, checksum, log_num),
        fragments_(),
        in_fragmented_record_(false) {}

  ~FragmentBufferedReader() override = default;

  bool ReadRecord(Slice* record, std::string* scratch,
                  WALRecoveryMode wal_recovery_mode =
                      WALRecoveryMode::kTolerateCorruptedTailRecords) override;
  void UnmarkEOF() override;

 private:
  std::string fragments_;
  bool in_fragmented_record_;

  bool TryReadFragment(Slice* result, size_t* drop_size,
                       uint8_t* fragment_type_or_err);
  bool TryReadMore(size_t* drop_size, uint8_t* error);

  FragmentBufferedReader(const FragmentBufferedReader&) = delete;
  void operator=(const FragmentBufferedReader&) = delete;
};

}  // namespace log
}  // namespace mwal
