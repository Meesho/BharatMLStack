// Copyright (c) 2011-present, Facebook, Inc.  All rights reserved.
// Derived from RocksDB — adapted for mwal standalone WAL library.

#include "mwal/status.h"

#include <cstring>

namespace mwal {

std::unique_ptr<const char[]> Status::CopyState(const char* s) {
  const size_t cch = std::strlen(s) + 1;
  char* rv = new char[cch];
  std::memcpy(rv, s, cch);
  return std::unique_ptr<const char[]>(rv);
}

Status::Status(Code _code, SubCode _subcode, const Slice& msg,
               const Slice& msg2, Severity sev)
    : code_(_code),
      subcode_(_subcode),
      sev_(sev),
      retryable_(false),
      data_loss_(false),
      scope_(0) {
  assert(code_ != kOk);
  const size_t len1 = msg.size();
  const size_t len2 = msg2.size();
  const size_t size = len1 + (len2 ? (2 + len2) : 0);
  char* const result = new char[size + 1];
  memcpy(result, msg.data(), len1);
  if (len2) {
    result[len1] = ':';
    result[len1 + 1] = ' ';
    memcpy(result + len1 + 2, msg2.data(), len2);
  }
  result[size] = '\0';
  state_.reset(result);
}

std::string Status::ToString() const {
  const char* type = nullptr;
  switch (code_) {
    case kOk:
      return "OK";
    case kNotFound:
      type = "NotFound: ";
      break;
    case kCorruption:
      type = "Corruption: ";
      break;
    case kNotSupported:
      type = "Not implemented: ";
      break;
    case kInvalidArgument:
      type = "Invalid argument: ";
      break;
    case kIOError:
      type = "IO error: ";
      break;
    case kIncomplete:
      type = "Result incomplete: ";
      break;
    case kShutdownInProgress:
      type = "Shutdown in progress: ";
      break;
    case kTimedOut:
      type = "Operation timed out: ";
      break;
    case kAborted:
      type = "Operation aborted: ";
      break;
    case kBusy:
      type = "Resource busy: ";
      break;
    case kTryAgain:
      type = "Operation failed, try again: ";
      break;
    default:
      type = "Unknown code: ";
      break;
  }
  std::string result(type);
  if (state_) {
    result.append(state_.get());
  }
  return result;
}

}  // namespace mwal
