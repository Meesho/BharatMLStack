// Copyright (c) 2011-present, Facebook, Inc.  All rights reserved.
// Derived from RocksDB — WriteBatch core serialization for mwal.

#include "mwal/write_batch.h"

#include <cassert>
#include <cstring>

#include "util/coding.h"

namespace mwal {

WriteBatch::WriteBatch() : rep_() {
  rep_.resize(kHeader);
  memset(rep_.data(), 0, kHeader);
}

void WriteBatch::Clear() {
  rep_.clear();
  rep_.resize(kHeader);
  memset(rep_.data(), 0, kHeader);
  has_put_ = false;
}

int WriteBatch::Count() const { return WriteBatchInternal::Count(this); }

void WriteBatch::SetSequence(SequenceNumber seq) {
  WriteBatchInternal::SetSequence(this, seq);
}

SequenceNumber WriteBatch::Sequence() const {
  return WriteBatchInternal::Sequence(this);
}

Status WriteBatch::Put(const Slice& key, const Slice& value) {
  WriteBatchInternal::SetCount(this, WriteBatchInternal::Count(this) + 1);
  rep_.push_back(static_cast<char>(kTypeValue));
  PutLengthPrefixedSlice(&rep_, key);
  PutLengthPrefixedSlice(&rep_, value);
  has_put_ = true;
  return Status::OK();
}

Status WriteBatch::Delete(const Slice& key) {
  WriteBatchInternal::SetCount(this, WriteBatchInternal::Count(this) + 1);
  rep_.push_back(static_cast<char>(kTypeDeletion));
  PutLengthPrefixedSlice(&rep_, key);
  return Status::OK();
}

Status WriteBatch::SingleDelete(const Slice& key) {
  WriteBatchInternal::SetCount(this, WriteBatchInternal::Count(this) + 1);
  rep_.push_back(static_cast<char>(kTypeSingleDeletion));
  PutLengthPrefixedSlice(&rep_, key);
  return Status::OK();
}

Status WriteBatch::Merge(const Slice& key, const Slice& value) {
  WriteBatchInternal::SetCount(this, WriteBatchInternal::Count(this) + 1);
  rep_.push_back(static_cast<char>(kTypeMerge));
  PutLengthPrefixedSlice(&rep_, key);
  PutLengthPrefixedSlice(&rep_, value);
  return Status::OK();
}

Status WriteBatch::PutLogData(const Slice& blob) {
  rep_.push_back(static_cast<char>(kTypeLogData));
  PutLengthPrefixedSlice(&rep_, blob);
  return Status::OK();
}

Status WriteBatch::Iterate(Handler* handler) const {
  if (rep_.size() < kHeader) {
    return Status::Corruption("malformed WriteBatch (too small)");
  }
  Slice input(rep_.data() + kHeader, rep_.size() - kHeader);
  Slice key, value;
  int found = 0;

  while (!input.empty() && handler->Continue()) {
    char tag = input[0];
    input.remove_prefix(1);

    switch (static_cast<ValueType>(tag)) {
      case kTypeValue:
        if (GetLengthPrefixedSlice(&input, &key) &&
            GetLengthPrefixedSlice(&input, &value)) {
          Status s = handler->Put(key, value);
          if (!s.ok()) return s;
        } else {
          return Status::Corruption("bad WriteBatch Put");
        }
        found++;
        break;

      case kTypeDeletion:
        if (GetLengthPrefixedSlice(&input, &key)) {
          Status s = handler->Delete(key);
          if (!s.ok()) return s;
        } else {
          return Status::Corruption("bad WriteBatch Delete");
        }
        found++;
        break;

      case kTypeSingleDeletion:
        if (GetLengthPrefixedSlice(&input, &key)) {
          Status s = handler->SingleDelete(key);
          if (!s.ok()) return s;
        } else {
          return Status::Corruption("bad WriteBatch SingleDelete");
        }
        found++;
        break;

      case kTypeMerge:
        if (GetLengthPrefixedSlice(&input, &key) &&
            GetLengthPrefixedSlice(&input, &value)) {
          Status s = handler->Merge(key, value);
          if (!s.ok()) return s;
        } else {
          return Status::Corruption("bad WriteBatch Merge");
        }
        found++;
        break;

      case kTypeLogData:
        if (GetLengthPrefixedSlice(&input, &value)) {
          handler->LogData(value);
        } else {
          return Status::Corruption("bad WriteBatch LogData");
        }
        break;

      default:
        return Status::Corruption("unknown WriteBatch tag");
    }
  }

  if (found != WriteBatchInternal::Count(this)) {
    return Status::Corruption("WriteBatch has wrong count");
  }
  return Status::OK();
}

// --- WriteBatchInternal ---

void WriteBatchInternal::SetCount(WriteBatch* batch, int n) {
  EncodeFixed32(batch->rep_.data() + 8, n);
}

int WriteBatchInternal::Count(const WriteBatch* batch) {
  return DecodeFixed32(batch->rep_.data() + 8);
}

void WriteBatchInternal::SetSequence(WriteBatch* batch, SequenceNumber seq) {
  EncodeFixed64(batch->rep_.data(), seq);
}

SequenceNumber WriteBatchInternal::Sequence(const WriteBatch* batch) {
  return DecodeFixed64(batch->rep_.data());
}

void WriteBatchInternal::SetContents(WriteBatch* batch, const Slice& contents) {
  assert(contents.size() >= WriteBatch::kHeader);
  batch->rep_.assign(contents.data(), contents.size());
}

void WriteBatchInternal::Append(WriteBatch* dst, const WriteBatch* src) {
  SetCount(dst, Count(dst) + Count(src));
  assert(src->rep_.size() >= WriteBatch::kHeader);
  dst->rep_.append(src->rep_.data() + WriteBatch::kHeader,
                   src->rep_.size() - WriteBatch::kHeader);
}

}  // namespace mwal
