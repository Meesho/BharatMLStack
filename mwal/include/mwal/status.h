// Copyright (c) 2011-present, Facebook, Inc.  All rights reserved.
// Derived from RocksDB — adapted for mwal standalone WAL library.

#pragma once

#include <memory>
#include <string>

#include "mwal/slice.h"

namespace mwal {

class Status {
 public:
  Status()
      : code_(kOk),
        subcode_(kNone),
        sev_(kNoError),
        retryable_(false),
        data_loss_(false),
        scope_(0),
        state_(nullptr) {}
  ~Status() = default;

  Status(const Status& s);
  Status& operator=(const Status& s);
  Status(Status&& s) noexcept;
  Status& operator=(Status&& s) noexcept;
  bool operator==(const Status& rhs) const;
  bool operator!=(const Status& rhs) const;

  enum Code : unsigned char {
    kOk = 0,
    kNotFound = 1,
    kCorruption = 2,
    kNotSupported = 3,
    kInvalidArgument = 4,
    kIOError = 5,
    kIncomplete = 7,
    kShutdownInProgress = 8,
    kTimedOut = 9,
    kAborted = 10,
    kBusy = 11,
    kTryAgain = 13,
    kMaxCode
  };

  Code code() const { return code_; }

  enum SubCode : unsigned char {
    kNone = 0,
    kNoSpace = 4,
    kMemoryLimit = 7,
    kPathNotFound = 9,
    kIOFenced = 14,
    kMaxSubCode
  };

  SubCode subcode() const { return subcode_; }

  enum Severity : unsigned char {
    kNoError = 0,
    kSoftError = 1,
    kHardError = 2,
    kFatalError = 3,
    kUnrecoverableError = 4,
    kMaxSeverity
  };

  Severity severity() const { return sev_; }

  const char* getState() const { return state_.get(); }

  static Status OK() { return Status(); }

  static Status NotFound(const Slice& msg, const Slice& msg2 = Slice()) {
    return Status(kNotFound, msg, msg2);
  }
  static Status NotFound(SubCode msg = kNone) { return Status(kNotFound, msg); }

  static Status Corruption(const Slice& msg, const Slice& msg2 = Slice()) {
    return Status(kCorruption, msg, msg2);
  }
  static Status Corruption(SubCode msg = kNone) {
    return Status(kCorruption, msg);
  }

  static Status NotSupported(const Slice& msg, const Slice& msg2 = Slice()) {
    return Status(kNotSupported, msg, msg2);
  }
  static Status NotSupported(SubCode msg = kNone) {
    return Status(kNotSupported, msg);
  }

  static Status InvalidArgument(const Slice& msg, const Slice& msg2 = Slice()) {
    return Status(kInvalidArgument, msg, msg2);
  }
  static Status InvalidArgument(SubCode msg = kNone) {
    return Status(kInvalidArgument, msg);
  }

  static Status IOError(const Slice& msg, const Slice& msg2 = Slice()) {
    return Status(kIOError, msg, msg2);
  }
  static Status IOError(SubCode msg = kNone) { return Status(kIOError, msg); }

  static Status Incomplete(const Slice& msg, const Slice& msg2 = Slice()) {
    return Status(kIncomplete, msg, msg2);
  }

  static Status Aborted(const Slice& msg, const Slice& msg2 = Slice()) {
    return Status(kAborted, msg, msg2);
  }

  static Status Busy(const Slice& msg, const Slice& msg2 = Slice()) {
    return Status(kBusy, msg, msg2);
  }

  static Status TimedOut(const Slice& msg, const Slice& msg2 = Slice()) {
    return Status(kTimedOut, msg, msg2);
  }

  static Status TryAgain(const Slice& msg, const Slice& msg2 = Slice()) {
    return Status(kTryAgain, msg, msg2);
  }

  static Status NoSpace() { return Status(kIOError, kNoSpace); }
  static Status NoSpace(const Slice& msg, const Slice& msg2 = Slice()) {
    return Status(kIOError, kNoSpace, msg, msg2);
  }

  static Status PathNotFound() { return Status(kIOError, kPathNotFound); }
  static Status PathNotFound(const Slice& msg, const Slice& msg2 = Slice()) {
    return Status(kIOError, kPathNotFound, msg, msg2);
  }

  bool ok() const { return code_ == kOk; }
  bool IsNotFound() const { return code_ == kNotFound; }
  bool IsCorruption() const { return code_ == kCorruption; }
  bool IsNotSupported() const { return code_ == kNotSupported; }
  bool IsInvalidArgument() const { return code_ == kInvalidArgument; }
  bool IsIOError() const { return code_ == kIOError; }
  bool IsIncomplete() const { return code_ == kIncomplete; }
  bool IsShutdownInProgress() const { return code_ == kShutdownInProgress; }
  bool IsTimedOut() const { return code_ == kTimedOut; }
  bool IsAborted() const { return code_ == kAborted; }
  bool IsBusy() const { return code_ == kBusy; }
  bool IsTryAgain() const { return code_ == kTryAgain; }
  bool IsNoSpace() const { return (code_ == kIOError) && (subcode_ == kNoSpace); }
  bool IsPathNotFound() const { return (code_ == kIOError) && (subcode_ == kPathNotFound); }

  std::string ToString() const;

 protected:
  Code code_;
  SubCode subcode_;
  Severity sev_;
  bool retryable_;
  bool data_loss_;
  unsigned char scope_;
  std::unique_ptr<const char[]> state_;

  explicit Status(Code _code, SubCode _subcode = kNone)
      : code_(_code),
        subcode_(_subcode),
        sev_(kNoError),
        retryable_(false),
        data_loss_(false),
        scope_(0) {}

  explicit Status(Code _code, SubCode _subcode, bool retryable, bool data_loss,
                  unsigned char scope)
      : code_(_code),
        subcode_(_subcode),
        sev_(kNoError),
        retryable_(retryable),
        data_loss_(data_loss),
        scope_(scope) {}

  Status(Code _code, SubCode _subcode, const Slice& msg, const Slice& msg2,
         Severity sev = kNoError);
  Status(Code _code, const Slice& msg, const Slice& msg2)
      : Status(_code, kNone, msg, msg2) {}

  static std::unique_ptr<const char[]> CopyState(const char* s);
};

inline Status::Status(const Status& s)
    : code_(s.code_),
      subcode_(s.subcode_),
      sev_(s.sev_),
      retryable_(s.retryable_),
      data_loss_(s.data_loss_),
      scope_(s.scope_) {
  state_ = (s.state_ == nullptr) ? nullptr : CopyState(s.state_.get());
}

inline Status& Status::operator=(const Status& s) {
  if (this != &s) {
    code_ = s.code_;
    subcode_ = s.subcode_;
    sev_ = s.sev_;
    retryable_ = s.retryable_;
    data_loss_ = s.data_loss_;
    scope_ = s.scope_;
    state_ = (s.state_ == nullptr) ? nullptr : CopyState(s.state_.get());
  }
  return *this;
}

inline Status::Status(Status&& s) noexcept : Status() {
  *this = std::move(s);
}

inline Status& Status::operator=(Status&& s) noexcept {
  if (this != &s) {
    code_ = s.code_;
    s.code_ = kOk;
    subcode_ = s.subcode_;
    s.subcode_ = kNone;
    sev_ = s.sev_;
    s.sev_ = kNoError;
    retryable_ = s.retryable_;
    s.retryable_ = false;
    data_loss_ = s.data_loss_;
    s.data_loss_ = false;
    scope_ = s.scope_;
    s.scope_ = 0;
    state_ = std::move(s.state_);
  }
  return *this;
}

inline bool Status::operator==(const Status& rhs) const {
  return (code_ == rhs.code_);
}

inline bool Status::operator!=(const Status& rhs) const {
  return !(*this == rhs);
}

}  // namespace mwal
