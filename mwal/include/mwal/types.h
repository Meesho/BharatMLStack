// Copyright (c) 2011-present, Facebook, Inc.  All rights reserved.
// Derived from RocksDB — adapted for mwal standalone WAL library.

#pragma once

#include <cstdint>

namespace mwal {

using SequenceNumber = uint64_t;

const SequenceNumber kMinUnCommittedSeq = 1;

enum FileType {
  kWalFile,
  kTableFile,
  kDescriptorFile,
  kCurrentFile,
  kTempFile,
  kInfoLogFile,
  kIdentityFile,
  kOptionsFile,
};

}  // namespace mwal
