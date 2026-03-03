// Copyright (c) 2011-present, Facebook, Inc.  All rights reserved.
// Derived from RocksDB — adapted for mwal standalone WAL library.

#pragma once

#include <cstdint>
#include <functional>
#include <string>

#include "mwal/compression_type.h"
#include "mwal/status.h"

namespace mwal {

class WriteBatch;
using SequenceNumber = uint64_t;

enum class WALRecoveryMode : char {
  kTolerateCorruptedTailRecords = 0x00,
  kAbsoluteConsistency = 0x01,
  kPointInTimeRecovery = 0x02,
  kSkipAnyCorruptedRecords = 0x03,
};

struct WALOptions {
  std::string wal_dir;
  uint64_t WAL_ttl_seconds = 0;
  uint64_t WAL_size_limit_MB = 0;
  WALRecoveryMode wal_recovery_mode = WALRecoveryMode::kPointInTimeRecovery;
  bool manual_wal_flush = false;
  CompressionType wal_compression = kNoCompression;
  uint64_t max_wal_file_size = 256 * 1024 * 1024;  // 0 = no limit

  // If set, Open() automatically replays existing WAL files via this callback
  // before accepting new writes.
  std::function<Status(SequenceNumber, WriteBatch*)> recovery_callback;

  // Background purge interval in seconds. 0 = disabled (manual purge only).
  uint64_t purge_interval_seconds = 0;

  // Max writers batched into a single group commit. 0 = no limit.
  size_t max_write_group_size = 0;

  // Max total WAL bytes on disk. Write() returns Busy when exceeded.
  // 0 = no limit.
  uint64_t max_total_wal_size = 0;

  // Max pending async writes in the coalescer queue. Async writes (sync=false)
  // are routed to a dedicated writer thread for contention-free coalescing.
  // When the queue is full, Write() blocks until space is available
  // (back-pressure). 0 = disabled (all writes use group commit). Default: 10000.
  size_t max_async_queue_depth = 10000;

  // Max time in ms before pending async batches are flushed. 0 = disabled
  // (flush only on new Submit or Stop).
  uint64_t max_async_flush_interval_ms = 0;
};

struct WriteOptions {
  bool sync = false;
  bool disableWAL = false;
};

}  // namespace mwal
