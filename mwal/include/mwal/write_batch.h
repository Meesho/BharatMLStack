// Copyright (c) 2011-present, Facebook, Inc.  All rights reserved.
// Derived from RocksDB — WriteBatch core serialization for mwal.
//
// Binary format:
//   [sequence_number: 8B][count: 4B][record...]
//   Each record: [type: 1B][key_len: varint][key][value_len: varint][value]

#pragma once

#include <cstdint>
#include <string>

#include "mwal/slice.h"
#include "mwal/status.h"
#include "mwal/types.h"

namespace mwal {

class WriteBatch {
 public:
  enum ValueType : uint8_t {
    kTypeValue = 0x1,
    kTypeDeletion = 0x0,
    kTypeMerge = 0x2,
    kTypeLogData = 0x3,
    kTypeSingleDeletion = 0x7,
  };

  class Handler {
   public:
    virtual ~Handler() = default;
    virtual Status PutCF(uint32_t /*cf*/, const Slice& key,
                         const Slice& value) {
      return Put(key, value);
    }
    virtual Status Put(const Slice& key, const Slice& value) = 0;
    virtual Status Delete(const Slice& key) = 0;
    virtual Status Merge(const Slice& /*key*/, const Slice& /*value*/) {
      return Status::NotSupported("Merge");
    }
    virtual Status SingleDelete(const Slice& key) { return Delete(key); }
    virtual void LogData(const Slice& /*blob*/) {}
    virtual bool Continue() { return true; }
  };

  WriteBatch();
  ~WriteBatch() = default;

  WriteBatch(const WriteBatch&) = default;
  WriteBatch& operator=(const WriteBatch&) = default;
  WriteBatch(WriteBatch&&) = default;
  WriteBatch& operator=(WriteBatch&&) = default;

  Status Put(const Slice& key, const Slice& value);
  Status Delete(const Slice& key);
  Status SingleDelete(const Slice& key);
  Status Merge(const Slice& key, const Slice& value);
  Status PutLogData(const Slice& blob);

  Status Iterate(Handler* handler) const;

  void Clear();

  int Count() const;

  void SetSequence(SequenceNumber seq);
  SequenceNumber Sequence() const;

  const std::string& Data() const { return rep_; }
  size_t GetDataSize() const { return rep_.size(); }
  bool HasPut() const { return has_put_; }

 private:
  friend class WriteBatchInternal;
  std::string rep_;
  bool has_put_ = false;

  static constexpr size_t kHeader = 12;  // 8-byte seq + 4-byte count
};

class WriteBatchInternal {
 public:
  static void SetCount(WriteBatch* batch, int n);
  static int Count(const WriteBatch* batch);
  static void SetSequence(WriteBatch* batch, SequenceNumber seq);
  static SequenceNumber Sequence(const WriteBatch* batch);
  static Slice Contents(const WriteBatch* batch) { return Slice(batch->rep_); }
  static void SetContents(WriteBatch* batch, const Slice& contents);
  static void Append(WriteBatch* dst, const WriteBatch* src);
};

}  // namespace mwal
