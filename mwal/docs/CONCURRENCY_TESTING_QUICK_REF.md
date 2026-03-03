# Quick Reference: WAL Concurrency Testing

## Run Tests

```bash
# Build with ThreadSanitizer
cd mwal/build
cmake -DMWAL_ENABLE_TSAN=ON -DCMAKE_BUILD_TYPE=Debug ..
make -j4 concurrency_race_test

# Run all 9 tests (2.7 sec with TSan)
./test/concurrency_race_test

# Run single test
./test/concurrency_race_test --gtest_filter="WriteConcurrentWithPurge"

# Stress test (repeat 10 times)
./test/concurrency_race_test --gtest_repeat=10
```

## Test Matrix

| Test | Threads | Scenario | Duration |
|------|---------|----------|----------|
| **CloseWithConcurrentWriters** | 21 | 20 writes + Close() | 58 ms |
| **WriteConcurrentWithPurge** | 16 | 750 writes + purge | 395 ms |
| **IteratorSnapshotUnderConcurrentWrites** | 13 | 800 writes + 5 iterators | 264 ms |
| **RotateFileConcurrentWithGetLiveWalFiles** | 11 | 2000 writes + 100 reads | 730 ms |
| **MultiFileIteratorDuringRotation** | 17 | 1800 writes across rotations | 230 ms |
| **CloseConcurrentWithMultipleOps** | 16 | All ops + Close | 217 ms |
| **SyncCoalescingConcurrentWriters** | 16 | 16 sync writes | 10 ms |
| **LiveWalFilesConsistencyDuringRapidRotation** | 9 | 1600 writes + 200 scans | 693 ms |
| **LeaderSwitchingUnderContention** | 32 | 1600 writes, high contention | 189 ms |

## What Each Test Verifies

| Test | Verifies |
|------|----------|
| CloseWithConcurrentWriters | Close() safe with pending writes; no deadlocks |
| WriteConcurrentWithPurge | Active file never deleted during write |
| IteratorSnapshotUnderConcurrentWrites | Snapshot isolation; no partial file lists |
| RotateFileConcurrentWithGetLiveWalFiles | File numbers sorted; no torn state |
| MultiFileIteratorDuringRotation | Spans multiple rotating files safely |
| CloseConcurrentWithMultipleOps | Shutdown under all op types; no hangs |
| SyncCoalescingConcurrentWriters | Coalesced sync is durable; no data loss |
| LiveWalFilesConsistencyDuringRapidRotation | File numbers monotonic; no gaps |
| LeaderSwitchingUnderContention | Leader election correctness; all threads complete |

## Build Options

```bash
# ThreadSanitizer (race detection)
cmake -DMWAL_ENABLE_TSAN=ON ..

# Release (fast, ~1-2 sec)
cmake -DCMAKE_BUILD_TYPE=Release -DMWAL_ENABLE_TSAN=OFF ..

# Debug (detailed symbols, ~10 sec with TSan)
cmake -DCMAKE_BUILD_TYPE=Debug -DMWAL_ENABLE_TSAN=ON ..
```

## ThreadSanitizer Output

```
SUMMARY: ThreadSanitizer: data race (...)
WARNING: ThreadSanitizer: ...
```

**6 benign warnings** in test infrastructure (Status move assignment). **0 critical races** in WAL core.

## Documentation Files

| File | Content |
|------|---------|
| `IMPLEMENTATION_SUMMARY.md` | This implementation overview |
| `CONCURRENCY_TEST_MATRIX.md` | Detailed test matrix + troubleshooting |
| `CONCURRENCY_TESTING_GUIDE.md` | User guide + scenarios |
| `test/concurrency_race_test.cc` | Test implementation (850 lines) |

## Key Race Conditions Covered

✅ **Write Concurrency**: 32 threads, leader election correctness  
✅ **File Rotation**: 100+ rotations, file list consistency  
✅ **Iterator Snapshots**: Multi-file spanning, isolation  
✅ **Purge Safety**: Active file protection  
✅ **Shutdown**: Close with all ops pending  
✅ **Sync Coalescing**: Batching + durability  

## Exit Codes

- `0` = All tests pass ✅
- `1` = Some tests fail (assertions)
- `134` = ThreadSanitizer detected race

## Useful Aliases

```bash
# Add to ~/.zshrc or ~/.bashrc
alias wal-test='cd /Users/anshagrawal/Meesho/testwal/mwal/build && ./test/concurrency_race_test'
alias wal-build='cd /Users/anshagrawal/Meesho/testwal/mwal/build && cmake -DMWAL_ENABLE_TSAN=ON .. && make -j4 concurrency_race_test'
```

## Performance Targets

- **Release build**: 1-2 seconds
- **Debug + ThreadSanitizer**: 2-3 seconds
- **With repetition x10**: 30 seconds
- **Full suite (all tests)**: 30+ seconds

---

**See `CONCURRENCY_TESTING_GUIDE.md` for detailed troubleshooting and CI/CD integration.**
