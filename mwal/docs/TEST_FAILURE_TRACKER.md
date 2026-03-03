# WAL Test Failure Tracker

Purpose: track strict test failures where test intent (from `WAL_DESIGN.md`) does not currently match implementation behavior.

**Guardrails (do not violate):**
- Keep tests strict. Do not weaken assertions to make tests pass.
- Never lower guards on any test case.
- Add one entry per failing test.
- Mark an entry resolved by switching `[ ]` to `[x]` and adding a short resolution note.

Latest strict run: `ctest --output-on-failure` (full suite)  
Current unresolved strict failures: `0`

---

## Resolved Entries

Move resolved items here and mark them `[x]` with a one-line note about the implementation fix and test rerun result.

- [x] **Test**: `BitFlipCorruptionTest.TC_CORRUPT_01_BitFlipInCrcField`  
  **Fix**: `log_reader.cc` — For `kBadRecordChecksum`, return false (stop) instead of break for `kTolerateCorruptedTailRecords` and `kPointInTimeRecovery`; keep break for `kAbsoluteConsistency` and `kSkipAnyCorruptedRecords`. All modes now match spec.

- [x] **Test**: `CrashRecoveryTest.MidRotationNewFilePartialFirstRecord`  
  **Fix**: Test setup — Use `TruncateWalFileAt` on a valid 000002.log instead of creating a new file with 5 bytes (NewWritableFile+Append was not persisting). `DBWal::Recover` fallback (non-empty file, zero records, kAbsoluteConsistency) also added.

- [x] **Test**: `CrashRecoveryTest.RecoveryModeMatrix_MultiFileCorruptionInMiddle`  
  **Fix**: Same truncation-based setup as above; create valid 000002 via DBWal, then truncate to 5 bytes.

- [x] **Test**: `WalIteratorTest.IteratorMultiFileCorruptionInSecond`  
  **Fix**: `wal_iterator.cc` — When `ReadRecord` returns false and no more files, set `status_ = Status::Corruption("truncated or corrupt log file")` for `kAbsoluteConsistency` if `first_error` is OK (fallback when reader does not report).

- [x] **Test**: `CrashRecoveryTest.RecoverAcrossThreeFilesMiddleCorrupt`  
  **Fix**: Test setup — Use `WriteRawWalRecord` for 000001, 000002, 000003, then `TruncateWalFileAt(000002, 5)` to simulate corrupt middle file. `DBWal::Recover` fallback handles non-empty zero-record files.
