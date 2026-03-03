// Copyright (c) 2011-present, Facebook, Inc.  All rights reserved.
// Derived from RocksDB — adapted for mwal standalone WAL library.

#include "mwal/slice.h"

namespace mwal {

std::string Slice::ToString(bool hex) const {
  std::string result;
  if (hex) {
    result.reserve(2 * size_);
    for (size_t i = 0; i < size_; i++) {
      char buf[3];
      snprintf(buf, sizeof(buf), "%02x",
               static_cast<unsigned char>(data_[i]));
      result.append(buf, 2);
    }
    return result;
  } else {
    result.assign(data_, size_);
    return result;
  }
}

}  // namespace mwal
