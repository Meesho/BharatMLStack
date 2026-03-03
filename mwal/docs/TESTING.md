# mwal Testing Documentation

This document describes all unit tests and benchmarks in the mwal (Write-Ahead Log) library.

---

## Testing Guardrails (Non-Negotiable)

- **Never lower guards on any test case.** Do not weaken assertions, relax expectations, or change `EXPECT_EQ` to `EXPECT_GE`/`EXPECT_LE` just to make tests pass.
- **Test intent, not implementation.** Assertions must reflect the spec (e.g. WAL_DESIGN.md, test case descriptions), not current implementation behavior.
- **Track failures, do not hide them.** If a strict assertion fails, add an entry to `TEST_FAILURE_TRACKER.md` with intended vs actual behavior. Fix the implementation or resolve the spec/implementation mismatch—do not relax the test.

---

## How to Run Tests

### Build and run unit tests

```bash
cd mwal
mkdir -p build && cd build
cmake .. -DCMAKE_BUILD_TYPE=Release
make
ctest
```

Or run a specific test:

```bash
./test/db_wal_test
./test/log_test
./test/crash_recovery_test
./test/wal_iterator_test
./test/sequence_number_test
./test/env_error_injection_test
./test/back_pressure_test
./test/operational_test
./test/concurrency_race_test
# etc.
ctest -R db_wal_test
ctest -R crash_recovery_test
ctest -R wal_iterator_test
```

If strict intent-vs-implementation tests fail, track them in `docs/TEST_FAILURE_TRACKER.md` and resolve items by fixing implementation (not by weakening assertions).
Benchmarks in `bench/` are intent-audited for setup/IO error handling; this pass does not enforce performance thresholds.

### Build and run benchmarks

Benchmarks require [Google Benchmark](https://github.com/google/benchmark). Install with `brew install google-benchmark`, then:

```bash
cd mwal/build
cmake .. -DCMAKE_BUILD_TYPE=Release
make
./bench/db_wal_bench
./bench/wal_write_bench
./bench/wal_read_bench
./bench/wal_sync_bench
```

CMake options:

- `MWAL_BUILD_TESTS=ON` (default) — build unit tests (Google Test).
- `MWAL_BUILD_BENCH=ON` (default) — build benchmarks if Google Benchmark is found.

---

## Unit Tests (Google Test)

Tests live under `mwal/test/`. Each executable is registered with CTest via `add_test()`.

### 1. `db_wal_test` — DBWal integration

**File:** `test/db_wal_test.cc`  
**Fixture:** `DBWalTest` (temp dir, `Env::Default()`).

| Test | Description |
|------|-------------|
| `OpenClose` | Open and close DBWal; sequence number is 0. |
| `SingleWrite` | Single `Write` then recover; data matches. |
| `MultipleWrites` | Multiple writes; recovery returns all entries in order. |
| `SequenceNumberMonotonicity` | Sequence numbers increase across writes. |
| `DisableWAL` | With WAL disabled, no log file is created. |
| `LogRotation` | WAL rotates when `max_wal_file_size` is exceeded. |
| `RecoverEmpty` | Recover on empty WAL dir succeeds, no entries. |
| `ConcurrentWrites` | Multiple threads writing; all batches recovered. |
| `WriteAfterRecovery` | Write after Open (recovery); sequence continues correctly. |
| `LargeBatch` | Large write batch is written and recovered. |
| `FlushWAL` | `FlushWAL()` flushes without sync. |
| `SyncWrite` | `WriteOptions::sync = true` persists. |
| `NullBatchReturnsError` | Writing null batch returns error. |
| `WriteAfterClose` | Write after close returns error. |
| `CompressionRoundtrip` | Zstd compression (if enabled): write and recover. |
| `CloseRaceWithWriters` | Close while writers are active; no crash. |
| `SetMinLogToKeep` | `SetMinLogNumberToKeep()` affects purge. |
| `DirectoryLock` | Second open of same dir fails (lock). |
| `PurgeObsoleteExposed` | Purge removes obsolete WAL files. |
| `GetLiveWalFiles` | `GetLiveWalFiles()` returns current log files. |
| `WalIteratorBasic` | WAL iterator reads all records. |
| `WalIteratorFromMiddle` | Iterator from non-zero start sequence. |
| `AutoRecoveryOnOpen` | Open triggers recovery. |
| `MergedGroupCommit` | Multiple writers form one group commit. |
| `BackgroundPurge` | Background purge of old WAL files. |
| `WriteStallOnGroupSize` | Stall when group size limit hit. |
| `WriteStallOnTotalSize` | Stall when total WAL size limit hit. |
| `TTLPurgeWorks` | TTL-based purge of old logs. |

### 2. `log_test` — Log format (writer/reader)

**File:** `test/log_test.cc`  
**Fixture:** `LogTest` (temp dir, writer/reader helpers).

| Test | Description |
|------|-------------|
| `EmptyRecord` | Write and read empty record. |
| `SmallRecord` | Single small record roundtrip. |
| `MultipleRecords` | Several records in one log. |
| `LargeRecord` | Record larger than block size. |
| `ExactBlockSize` | Record exactly one block. |
| `RandomRecords` | Random-sized records roundtrip. |
| `RecyclableRecord` | Recyclable log (reuse) path. |
| `ChecksumMismatch` | Corrupt data; reader reports corruption. |
| `MarginalTrailer` | Trailer at block boundary. |

### 3. `wal_edit_test` — WAL manifest edits

**File:** `test/wal_edit_test.cc`

| Test | Description |
|------|-------------|
| `AdditionRoundtrip` | `WalAddition` encode/decode. |
| `DeletionRoundtrip` | `WalDeletion` encode/decode. |
| `WalSetAddAndDelete` | `WalSet`: add WALs, delete before N, min log. |
| `WalSetEmpty` | Empty WalSet; `GetMinLogNumber`, `CheckWals`. |
| `DebugString` | `WalAddition::DebugString` contains expected fields. |

### 4. `write_batch_test` — WriteBatch

**File:** `test/write_batch_test.cc`

| Test | Description |
|------|-------------|
| `Empty` | Empty batch: count 0, minimal data size. |
| `SinglePut` | One Put; iterate yields one Put. |
| `SingleDelete` | One Delete; iterate yields one Delete. |
| `MultipleOps` | Put, Delete, Merge; iteration order. |
| `SequenceNumber` | Sequence number in batch. |
| `PutLogData` | LogData in batch; handler sees it. |
| `Clear` | Clear() resets batch. |
| `LargeBatch` | Large batch encode. |
| `Append` | Append(batch) merges contents. |
| `SetContents` | SetContents from slice. |

### 5. `wal_manager_test` — WalManager

**File:** `test/wal_manager_test.cc`  
**Fixture:** `WalManagerTest` (temp dir, `CreateWalFile` helper).

| Test | Description |
|------|-------------|
| `GetSortedWalFiles` | Files returned sorted by log number. |
| `PurgeObsolete` | Purge removes files below min log to keep. |
| `EmptyDir` | Empty WAL dir returns no files. |
| `ZeroByteFile` | Zero-byte WAL file handling. |
| `DeleteWalFile` | DeleteWalFile removes file. |
| `PurgeSizeLimit` | Purge respects size limit. |
| `PurgeRespectsMinLog` | Purge does not delete files >= min log. |

**Section 8: WalManager edge cases (TC-WALMGR-01 through TC-WALMGR-05)**

| Test | Description |
|------|-------------|
| `TC_WALMGR_01_NonLogFilesIgnored` | LOCK, manifest.tmp, notes.txt, subdir/ in WAL dir; GetSortedWalFiles returns only 000003.log. |
| `TC_WALMGR_02_LogNumberGapsSortedCorrectly` | 000001, 000005, 000009.log; sorted 1,5,9; PurgeObsoleteFiles min_log=5 deletes only 000001. |
| `TC_WALMGR_03_PurgeDoesNotDeleteActiveLog` | DBWal rotated to file 3; SetMinLogNumberToKeep(3); Purge deletes 1 and 2, keeps 3; Write() succeeds; GetLiveWalFiles returns file 3 only. |
| `TC_WALMGR_04_TTLAndSizeLimitBothApply` | TTL=3600, size_limit=1MB; file 1 old (TTL expired), files 2+3 new; only file 1 deleted. |
| `TC_WALMGR_05_LargeNumberOfFiles` | 500 .log files; GetSortedWalFiles correct; 100-run latency baseline (logs if avg > 50ms). |

### 6. `write_thread_test` — WriteThread (group commit)

**File:** `test/write_thread_test.cc`

| Test | Description |
|------|-------------|
| `SingleWriter` | One writer is leader; group size 1. |
| `TwoWriters` | Two writers; one leader, one follower; both get OK. |
| `ManyWriters` | Multiple writers join group; all complete. |
| `LeaderSyncCoalescing` | Leader sync coalescing (sync flag). |
| `ErrorPropagation` | Leader error propagates to followers. |

### 7. `wal_compressor_test` — WAL compression

**File:** `test/wal_compressor_test.cc`

| Test | Description |
|------|-------------|
| `RoundtripUncompressed` | No compression: prefix + passthrough. |
| `RoundtripEmpty` | Empty input compress/decompress. |
| `DecompressEmpty` | Decompress empty slice. |
| `LegacyUncompressedPassthrough` | Legacy (non-tag) data passed through. |
| `ZstdRoundtripSmall` | Zstd small payload roundtrip (if Zstd). |
| `ZstdRoundtripLarge` | Zstd large payload roundtrip (if Zstd). |
| `ZstdCompressionRatio` | Zstd reduces size (if Zstd). |
| `ZstdCorruptedInput` | Corrupt Zstd data returns error (if Zstd). |
| `IsCompressed` | `IsCompressed()` for compressed/uncompressed. |
| `ZstdNotAvailable` | Behavior when Zstd not linked. |
| `UnsupportedType` | Unsupported compression type error. |

### 8. `crash_recovery_test` — Crash / truncation recovery

**File:** `test/crash_recovery_test.cc`  
**Fixture:** `CrashRecoveryTest` (temp dir, `Env::Default()`).

Simulates crash by truncating WAL files at specific byte offsets, then runs recovery under all four `WALRecoveryMode` values and asserts recovered count and Open status per WAL_DESIGN.md Section 11.2 and 11.3.

| Test | Description |
|------|-------------|
| `CleanShutdown` | Normal `Close()`; full recovery, zero data loss; all 4 modes recover N batches. |
| `TruncateLastBytes` | Truncate so last record is partial (crash after write before flush); Tolerate/PointInTime/SkipAny recover N−1, AbsoluteConsistency fails. |
| `TruncateMidHeader` | Truncate inside 7-byte record header; Tolerate/PointInTime/SkipAny recover 1 batch, AbsoluteConsistency fails. |
| `TruncateMidPayload` | Header complete, payload cut short; same assertions as TruncateMidHeader. |
| `TruncateBetweenFirstAndLast` | Multi-block record truncated after first fragment; tail tolerance skips incomplete record; AbsoluteConsistency fails. |
| `TruncateAtRecordBoundary` | Truncate exactly after last full record; all 4 modes recover all records. |
| `TruncateZeroBytes` | Truncate file to 0; no records recovered; all modes OK. |
| `MidRotationNewFileZeroBytes` | 000001.log full, 000002.log zero bytes; recovery reads file 1 fine; all modes OK. |
| `MidRotationNewFilePartialFirstRecord` | 000001.log full, 000002.log partial first record; Tolerate/PointInTime/SkipAny recover from file 1. |
| `DirectoryLock` | Second `Open()` on same WAL dir must fail (flock). |
| `RecoveryModeMatrix_FourModesTailCorruption` | Same corrupt file (truncated tail), four modes: Tolerate/PointInTime/SkipAny recover N−1, AbsoluteConsistency fails. |
| `RecoveryModeMatrix_ChecksumCorruptMidFile` | Corrupt checksum of mid-file record under TolerateCorruptedTailRecords; skip bad record, continue (1–2 records recovered). |
| `RecoveryModeMatrix_MultiFileCorruptionInMiddle` | Three files, middle file partially corrupt; AbsoluteConsistency fails, others skip middle file and recover from file 1 + 3. |

**Helpers:** `GetWalRecordOffsets` (parse WAL for record start/end), `TruncateWalFileAt` (POSIX truncate by path), `OverwriteByteAt` (corrupt one byte for checksum tests), `RunRecoveryWithMode` (Open with given mode and count recovered batches).

### 9. `wal_iterator_test` — WalIterator corruption and snapshot behavior

**File:** `test/wal_iterator_test.cc`  
**Fixture:** `WalIteratorTest` (temp dir, `Env::Default()`).

Covers iterator error paths and snapshot semantics per WAL_DESIGN.md Section 11.4.

| Test | Description |
|------|-------------|
| `IteratorCorruptionAbsoluteConsistency` | One file truncated after first record; Open (no recovery) + AbsoluteConsistency, NewWalIterator; first record valid, Next() hits corruption → Valid() false, status() Corruption. |
| `IteratorCorruptionPointInTimeSkipsFile` | File 1 full, file 2 partial; PointInTimeRecovery; iterator gets all records from file 1, then Valid() false, status() OK. |
| `IteratorStartSeqBeyondLast` | 10 records, NewWalIterator(100, &iter); Valid() immediately false, status() OK. |
| `IteratorSnapshotSemantics` | Open, write 5, NewWalIterator(0,&iter), write 5 more; iterator sees only 5 records (snapshot at creation). |
| `IteratorDecompressionFails` | (ZSTD) One compressed record, corrupt payload byte; Valid() false, status() Corruption when set (optional). |
| `IteratorMultiFileCorruptionInSecond` | File 1 full, file 2 partial; AbsoluteConsistency; iterator gets 2 records from file 1, then Valid() false. |

### 10. `sequence_number_test` — Sequence number correctness

**File:** `test/sequence_number_test.cc`  
**Fixture:** `SequenceNumberTest` (temp dir, `Env::Default()`).

Covers sequence number semantics per Section 6 (TC-SEQ-01 through TC-SEQ-04).

| Test | Description |
|------|-------------|
| `SequenceContinuityAcrossLogRotation` | TC-SEQ-01: Sequences contiguous across file boundaries; no gaps/duplicates; GetCurrentLogNumber() increments. |
| `SequenceContinuityAfterRecovery` | TC-SEQ-02: New write after recovery starts at last_recovered + last_batch_count; GetLatestSequenceNumber() correct. |
| `ConcurrentWriteSequenceCorrectness` | TC-SEQ-03: 32×100 batches, 1 op each; sequences 1–3200 exactly once, no gaps. |
| `DisableWALStillAdvancesSequence` | TC-SEQ-04: disableWAL advances sequence to 10; no WAL I/O; recovery replays only 5; GetLatestSequenceNumber() correct. |

### 11. `bit_flip_corruption_test` — Bit-flip / data corruption (Section 7)

**File:** `test/bit_flip_corruption_test.cc`  
**Fixture:** `BitFlipCorruptionTest` (temp dir, `Env::Default()`).

Covers bit-flip and data corruption scenarios per WAL_DESIGN.md Section 11 (TC-CORRUPT-01 through TC-CORRUPT-06).

| Test | Description |
|------|-------------|
| `TC_CORRUPT_01_BitFlipInCrcField` | XOR 1 byte in CRC of record 2; run under all 4 modes. Tolerate: record 1 recovered, no error. Absolute: Recover() returns Corruption. PointInTime: record 1 recovered, OK. SkipAny: records 1 and 3 recovered, OK. |
| `TC_CORRUPT_02_BitFlipInLengthField` | XOR 1 byte in length field of record 2; kPointInTimeRecovery: record 1 recovered, 2 and 3 not; Recover() OK. |
| `TC_CORRUPT_03_BitFlipInPayload` | XOR 1 byte in payload of record 2; kPointInTimeRecovery: CRC fails on record 2; record 1 recovered. |
| `TC_CORRUPT_04_MiddleFragmentTypeToFullType` | 100 KiB record (3 blocks); corrupt Middle fragment type to kFullType; kPointInTimeRecovery: fragment sequence error; recovery stops. |
| `TC_CORRUPT_05_UnknownCompressionPrefix` | Change compression prefix 0x00 to 0x05; run under all modes; document actual behavior (Corruption vs silent garbage). |
| `TC_CORRUPT_06_ValidCorruptValidSkipMode` | Corrupt record 2 CRC; kSkipAnyCorruptedRecords: records 1 and 3 recovered; callback exactly twice; Recover() OK. |

**Helpers:** `GetWalRecordOffsets`, `OverwriteByteAt`, `XorByteAt`, `RunRecoveryWithMode` (duplicated from crash_recovery_test).

### 12. `env_error_injection_test` — File System / Env Error Injection (Section 9)

**File:** `test/env_error_injection_test.cc`  
**Fixture:** `EnvErrorInjectionTest` (temp dir, `FaultInjectionEnv` wrapping `Env::Default()`).

Uses `FaultInjectionEnv` to fail specific operations after N successful calls. Covers env/IO error handling per spec TC-ENV-01 through TC-ENV-07.

| Test | Description |
|------|-------------|
| `TC_ENV_01_NewWritableFileFailsAtOpen` | NewWritableFile fails at Open(); Open() returns IOError; no LOCK or log file left behind; second Open with healthy env succeeds. |
| `TC_ENV_02_NewWritableFileFailsDuringRotation` | Write that triggers rotation fails when NewWritableFile fails; WAL in known error state; old log file intact and recoverable. |
| `TC_ENV_03_AppendFailsMidWrite` | Append fails mid-write; Write() returns IOError; partial record skipped on recovery (kTolerateCorruptedTailRecords); WAL usable after. |
| `TC_ENV_04_SyncFailsOnSyncWAL` | Sync fails on SyncWAL(); SyncWAL returns IOError; data in buffer not corrupted; subsequent Sync (after removing fault) succeeds. |
| `TC_ENV_05_GetFileModificationTimeFailsDuringPurge` | GetFileModificationTime fails for one log file; PurgeObsoleteFiles does not crash; file with unreadable mtime is kept (conservative). |
| `TC_ENV_06_DirectoryDoesNotExistAtOpen` | wal_dir does not exist; CreateDirIfMissing creates it; Open, Write, Close succeed. |
| `TC_ENV_07_ReadOnlyDirectoryAtOpen` | Directory chmod 444; Open returns non-OK (IOError or Permission denied); no crash; no partial files. |

### 13. `back_pressure_test` — Back-Pressure Behavior (Section 10)

**File:** `test/back_pressure_test.cc`  
**Fixture:** `BackPressureTest` (temp dir, `Env::Default()`).

Covers `max_total_wal_size` back-pressure and purge-unblock behavior.

| Test | Description |
|------|-------------|
| `TC_BP_01_WriteSucceedsAfterPurgeClearsSize` | Write until Busy; purge to reduce size; Write() after purge returns OK; post-purge write visible in recovery. |
| `TC_BP_02_MultipleThreadsUnblockAfterPurge` | 10 threads hit Busy; main thread purges; at least one thread succeeds on retry; no hang. |
| `TC_BP_03_ZeroDisablesBackPressure` | max_total_wal_size=0; 5000 writes; never returns Busy. |

### 14. `operational_test` — Operational / Tooling (Section 11)

**File:** `test/operational_test.cc`  
**Fixture:** `OperationalTest` (temp dir, `Env::Default()`).

Covers operational scenarios: compression compatibility, non-WAL files, stale LOCK, process races, WriteAfterClose, recovery callback error propagation.

| Test | Description |
|------|-------------|
| `TC_OPS_01_OpenCompressedWithNoCompressionOptions` | Write with kZSTD; Open with kNoCompression; recovery succeeds; all 10 batches recovered (per-record prefix drives decompression). |
| `TC_OPS_02_OpenNonWalFileAsWal` | 000001.log filled with random bytes; Open returns Corruption (AbsoluteConsistency) or OK with 0 records (Tolerate). |
| `TC_OPS_03_StaleLockFromCrashedProcess` | Child opens, killed with SIGKILL; parent Open succeeds (kernel releases locks on process death). |
| `TC_OPS_04_TwoProcessesRaceToOpen` | Two children Open simultaneously; exactly one succeeds. |
| `TC_OPS_05_WriteAfterCloseReturnsError` | Write after Close returns non-OK, descriptive error (e.g. "closed"). |
| `TC_OPS_06_RecoveryCallbackErrorPropagates` | Callback returns Corruption on 5th batch; Open returns Corruption; callback invoked exactly 5 times. |

### 15. `concurrency_race_test` — Concurrency and race conditions

**File:** `test/concurrency_race_test.cc`  
**Fixture:** `ConcurrencyRaceTest` (temp dir, `Env::Default()`).

Comprehensive concurrency and race-condition tests. **Run with ThreadSanitizer**: `cmake -DMWAL_ENABLE_TSAN=ON` then `./test/concurrency_race_test`. See `CONCURRENCY_TESTING_GUIDE.md` and `CONCURRENCY_TEST_MATRIX.md` for details.

| Test | Description |
|------|-------------|
| `CloseWithConcurrentWriters` | 20 writers concurrent with Close(); no crash; writers get OK or Aborted. |
| `WriteConcurrentWithPurge` | 15 writers + purge thread; active log never deleted; data integrity preserved. |
| `IteratorSnapshotUnderConcurrentWrites` | 8 writers + 5 iterators; iterator snapshot isolation; sequences monotonic. |
| `RotateFileConcurrentWithGetLiveWalFiles` | Rapid writes/rotations + GetLiveWalFiles(); file list consistent; no torn state. |
| `MultiFileIteratorDuringRotation` | 12 writers causing rotations + 5 iterators; snapshot consistent across rotations. |
| `CloseConcurrentWithMultipleOps` | Close while writes/purges/iterators pending; safe shutdown; no deadlock. |
| `SyncCoalescingConcurrentWriters` | 16 threads with sync=true; coalescing correct; no data loss on recovery. |
| `LiveWalFilesConsistencyDuringRapidRotation` | Rapid rotations + repeated GetLiveWalFiles; file numbers monotonic. |
| `LeaderSwitchingUnderContention` | 32 writers; leader election correctness; all complete successfully. |

---

## Benchmarks (Google Benchmark)

Benchmarks live under `mwal/bench/`. They use a temporary directory (e.g. `/tmp/mwal_bench_*`) and are not registered as CTest tests. See [BENCHMARK_RESULTS.md](BENCHMARK_RESULTS.md) for run instructions and recorded results.

### 1. `db_wal_bench` — DBWal performance

**File:** `bench/db_wal_bench.cc`

| Benchmark | Parameters | Description |
|-----------|------------|-------------|
| `BM_DBWalWrite` | ops_per_batch: 1, 10, 100 | Single-thread write throughput (1000 batches per iteration). |
| `BM_DBWalWriteSync` | do_sync: 0, 1 | Write with sync=false vs sync=true (100 writes). |
| `BM_DBWalConcurrentWrite` | num_threads: 1,2,4,8,16 | Concurrent write throughput (200 writes/thread). |
| `BM_DBWalWriteLatencyUnderThroughput` | num_threads: 1,2,4,8,16 | p50/p95/p99 write latency under load. |
| `BM_DBWalRecovery` | num_records: 1k, 10k, 100k | Time to open and recover WAL. |
| `BM_DBWalRotation` | — | Write throughput with small `max_wal_file_size` (frequent rotation). |
| `BM_GroupCommitScaling` | num_threads: 1,2,4,8 | Sync write throughput vs thread count (group commit). |
| `BM_MergedGroupCommit` | num_threads: 1,2,4,8,16 | Throughput with many threads (merged group commit). |
| `BM_CompressionPrefixOverhead` | — | Write throughput with no compression (prefix overhead only). |
| `BM_DBWalCompression` | compress: 0, 1 | Write throughput with compression off vs on (Zstd; if `MWAL_HAVE_ZSTD`). |

### 2. `wal_write_bench` — Log writer and WriteBatch encoding

**File:** `bench/wal_write_bench.cc`

| Benchmark | Parameters | Description |
|-----------|------------|-------------|
| `BM_WALWrite` | record_size: 64, 256, 1K, 4K, 32K, 128K | Raw log writer: 1000 records per iteration. |
| `BM_WALWriteSync` | per_record_sync: 0, 1 | Log write with optional per-record flush. |
| `BM_WriteBatchEncode` | num_ops: 1, 10, 100, 1000 | WriteBatch encode (Put only) and `GetDataSize()`. |

### 3. `wal_read_bench` — Log reader and recovery

**File:** `bench/wal_read_bench.cc`

| Benchmark | Parameters | Description |
|-----------|------------|-------------|
| `BM_WALRead` | record_size: 64, 256, 1K, 4K, 32K | Read 1000 records; reports bytes processed. |
| `BM_WALRecovery` | num_records: 1k, 10k, 100k | Full read of log (recovery-style). |

### 4. `wal_sync_bench` — Sync and group commit

**File:** `bench/wal_sync_bench.cc`

| Benchmark | Parameters | Description |
|-----------|------------|-------------|
| `BM_FsyncLatency` | — | Latency of `WritableFile::Sync()` (repeated fsync). |
| `BM_GroupCommit` | num_threads: 1, 2, 4, 8 | Multiple threads appending to one log writer then single flush. |

---

## Test Utilities

- **`test_util.h`** — Used by tests (e.g. `db_wal_test`, `log_test`, `wal_manager_test`). Provides helpers such as `test::TempDir` and `test::TempFileName` for temporary directories and WAL file paths.

---

## Summary

| Category | Count |
|----------|--------|
| Unit test executables | 15 |
| Unit test cases (approx.) | 124+ |
| Benchmark executables | 4 |
| Benchmark functions (approx.) | 16+ |

All unit tests use Google Test and are run via CTest. Benchmarks use Google Benchmark and are run by executing the benchmark binaries directly.
