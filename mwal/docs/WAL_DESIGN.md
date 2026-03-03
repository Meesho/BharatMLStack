# mwal Write-Ahead Log (WAL) System — Design Document

This document describes the architecture and internal mechanics of the **mwal** WAL library. It is structured as a High-Level Design (HLD) first, then progressively dives deeper using a breadth-first approach so an engineer can understand the system from top-level architecture down to implementation details.

**Document map**

| Section | Content |
|---------|---------|
| **1. HLD** | Purpose, top-level architecture, write/recovery flows, invariants |
| **2. Public API** | DBWal interface, WALOptions, WriteOptions, WalFileInfo, WalIterator |
| **3. Concurrency** | WriteThread leader–follower, merge + single record, Close() ordering |
| **4. WalManager** | File lifecycle, purge (TTL, size, min_log), naming, discovery |
| **5. Record format** | Block layout, log::Writer/Reader, buffering |
| **6. Compression** | 1-byte prefix, WriteBatch format, merge |
| **7. Env** | File I/O, time, directory lock (flock) |
| **8. Recovery & iterator** | Recover(), auto-recovery on Open, WalIterator snapshot |
| **9. Back-pressure & bg purge** | max_total_wal_size stall, background purge thread |
| **10. Directory layout** | LOCK, \*.log naming |
| **11. Corruption handling** | Where corruption is detected, recovery modes, Reporter, how each path proceeds |
| **12. Summary table** | Component responsibility matrix |

---

## 1. High-Level Design (HLD)

### 1.1 Purpose and Scope

**mwal** is a standalone C++ library that provides a **Write-Ahead Log (WAL)** for durability and crash recovery. It does **not** implement a full database; it only handles:

- **Append-only logging** of atomic write batches to disk
- **Group commit** to batch concurrent writes for throughput
- **Log rotation** and **purge** of obsolete WAL files
- **Recovery** by replaying WAL records in order
- **Sequence numbers** for ordering and consistency

A database or storage engine typically uses mwal to persist mutations before applying them to in-memory or on-disk structures. After a crash, the engine replays the WAL to reconstruct state.

### 1.2 Top-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Application / DB Engine                        │
└─────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                              DBWal (Public API)                          │
│  Open / Write / Recover / FlushWAL / SyncWAL / Close                     │
│  SetMinLogNumberToKeep / PurgeObsoleteFiles / GetLiveWalFiles            │
│  NewWalIterator                                                          │
└─────────────────────────────────────────────────────────────────────────┘
         │                    │                    │                    │
         ▼                    ▼                    ▼                    ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ WriteThread  │    │ WalManager   │    │ log::Writer  │    │ WalIterator  │
│ (group       │    │ (file life-  │    │ (record      │    │ (pull-based  │
│  commit)     │    │  cycle)      │    │  format)     │    │  read)       │
└──────────────┘    └──────────────┘    └──────────────┘    └──────────────┘
         │                    │                    │                    │
         └────────────────────┴────────────────────┴────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  WalCompressor / WritableFileWriter / Env (File I/O, Lock, Time)         │
└─────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  WAL directory: LOCK, 000001.log, 000002.log, ...                        │
└─────────────────────────────────────────────────────────────────────────┘
```

- **DBWal**: Single entry point. Owns all WAL state, sequence numbers, and coordination.
- **WriteThread**: Leader–follower concurrency; batches concurrent writers into one group commit.
- **WalManager**: Discovers, sorts, and purges WAL files (TTL, size limit, min log to keep).
- **log::Writer**: Formats records into blocks with checksums and fragmentation (RocksDB-style format).
- **WalIterator**: Pull-based reader over a snapshot of WAL files (e.g. from a given sequence number).
- **Env**: Abstraction for file creation, directory lock, time; POSIX implementation uses `flock` and `system_clock`.

### 1.3 Core Data Flow (Write Path)

1. Application calls **DBWal::Write(WriteOptions, WriteBatch\*)**.
2. If `disableWAL`: sequence is advanced and return; no I/O.
3. Otherwise the thread **joins** the WriteThread; it becomes either **leader** or **follower**.
4. **Leader** collects pending writers (up to `max_write_group_size`), merges their batches into one **WriteBatch**, assigns a single contiguous sequence range, optionally **compresses** the merged payload, then writes **one** record via **log::Writer::AddRecord** under **writer_mu_**.
5. If any writer had `sync=true`, the leader syncs the file after the write.
6. If current WAL file size exceeds **max_wal_file_size**, the leader **rotates** to a new log file.
7. Leader signals all **followers** with the same status; they wake and return.

### 1.4 Core Data Flow (Recovery Path)

1. Application calls **DBWal::Recover(callback)** or sets **recovery_callback** in WALOptions so **Open()** replays automatically.
2. **WalManager::GetSortedWalFiles** returns WAL files in log-number order.
3. For each file: open via Env, wrap in **SequentialFileReader**, create **log::Reader**, then **ReadRecord** in a loop.
4. Each record is **decompressed** (1-byte prefix: 0x00 = none, 0x07 = zstd), then interpreted as a **WriteBatch**.
5. Callback is invoked with (sequence, WriteBatch\*). Engine applies the batch (e.g. to memtable).
6. After all files, **last_sequence_** is set to the max sequence seen; a **new** log file is created for subsequent writes.

### 1.5 Key Invariants

- **Single writer to current log**: Only the leader writes; all writes go through **writer_mu_** and the single **log_writer_**.
- **Monotonic sequence numbers**: Assigned in order; recovery replays in order.
- **Single process per directory**: **LOCK** file (e.g. `wal_dir/LOCK`) is held for the lifetime of DBWal; second Open() on same dir fails.
- **Log numbers never decrease**: New log file gets `log_number_ + 1`; existing files are discovered at Open so new files never overwrite.

---

## 2. Layer 2: Public API and Configuration

### 2.1 DBWal Public Interface

| Method | Purpose |
|--------|--------|
| **Open(options, env, result)** | Create and open a WAL. Acquires dir lock, discovers existing WALs, optionally runs recovery_callback, creates first/next log file. May start background purge thread. |
| **Write(options, batch)** | Append batch to WAL (or advance sequence only if disableWAL). Blocking; returns when persisted (and synced if requested). |
| **Recover(callback)** | Replay all existing WAL files in order; callback(seq, batch) for each batch. Creates new log file after replay. |
| **FlushWAL(sync)** | Flush in-memory buffer; if sync, fsync. |
| **SyncWAL()** | FlushWAL(true). |
| **Close()** | Drain writers (DrainAndClose), stop bg purge, flush and close log writer, release dir lock. |
| **GetLatestSequenceNumber()** | Last assigned sequence (atomic). |
| **GetCurrentLogNumber()** | Current active WAL file number (atomic). |
| **SetMinLogNumberToKeep(n)** | Set min log number; purge may delete files with log_number < n when other conditions allow. |
| **PurgeObsoleteFiles()** | Delegate to WalManager to delete obsolete WAL files. |
| **GetLiveWalFiles(files)** | Delegate to WalManager; return sorted list of current WAL files (WalFileInfo). |
| **NewWalIterator(start_seq, result)** | Create a pull-based iterator over WAL from **start_seq through the current end** of the snapshot. Uses a snapshot of live WAL files at creation time; you can iterate (Valid / Next / GetBatch / GetSequenceNumber) to read every record with sequence ≥ start_seq until the last record in that snapshot. "Current" is fixed at iterator creation—new writes after that are not seen. |

### 2.2 WALOptions (Configuration)

| Option | Default | Meaning |
|--------|---------|--------|
| **wal_dir** | (required) | Directory for LOCK and \*.log files. |
| **WAL_ttl_seconds** | 0 | Delete WAL files older than this (wall-clock). 0 = disable TTL purge. |
| **WAL_size_limit_MB** | 0 | Purge oldest files when total WAL size exceeds this. 0 = no limit. |
| **wal_recovery_mode** | kPointInTimeRecovery | How to handle corruption (tolerate tail, absolute consistency, skip bad, etc.). |
| **manual_wal_flush** | false | If true, log::Writer does not auto-flush after each record; app must FlushWAL. |
| **wal_compression** | kNoCompression | None or kZSTD (if built with zstd). |
| **max_wal_file_size** | 256 MB | Rotate to new file when current file exceeds this. 0 = no rotation. |
| **recovery_callback** | (empty) | If set, Open() runs Recover(recovery_callback) when existing WALs exist. |
| **purge_interval_seconds** | 0 | Background thread calls PurgeObsoleteFiles() every N seconds. 0 = disabled. |
| **max_write_group_size** | 0 | Cap number of writers per group commit. 0 = no cap. |
| **max_total_wal_size** | 0 | Write() returns Busy when total WAL size exceeds this. 0 = no back-pressure. |
| **max_async_queue_depth** | 10000 | Max pending async writes in coalescer queue. 0 = disable coalescer. |
| **max_async_flush_interval_ms** | 0 | Max time in ms before pending async batches are flushed. 0 = disabled (flush only on new Submit or Stop). |

### 2.3 WriteOptions

| Option | Default | Meaning |
|--------|---------|--------|
| **sync** | false | If true, fsync after this write (and coalesced group). |
| **disableWAL** | false | If true, do not write to WAL; only advance sequence. |

### 2.4 WalFileInfo and WalIterator

- **WalFileInfo**: `log_number`, `size_bytes`, `path`. Used by GetLiveWalFiles and by WalIterator.
- **WalIterator**: **Valid()**, **Next()**, **GetBatch()**, **GetSequenceNumber()**, **status()**. Built from a snapshot of WAL files at creation time; does not hold a lock on the WAL directory.

---

## 3. Layer 3: Concurrency and Write Path (WriteThread + DBWal)

### 3.1 Leader–Follower Model

- **WriteThread** maintains a **deque** of pending **Writer** structs and a **leader_active_** flag.
- **JoinBatchGroup(w)**:
  - If **DrainAndClose()** was called: set w->status = Aborted, w->done = true, return false.
  - Else push w onto **pending_**.
  - If no leader: set **leader_active_ = true**, return true (this thread is leader).
  - Else: wait on **cv_** until **w->done** or **w->is_leader** or **closed_**. If closed and not leader, set Aborted and return false. If **w->is_leader**, return true (new leader); else return false (follower, w->status already set).
- **EnterAsBatchGroupLeader(leader, group, max_group_size)**:
  - Under **mu_**, drain from **pending_** into **group** (leader + linked list via **link_newer**), up to **max_group_size** if > 0. Set **need_sync** if any writer has sync=true.
- **ExitAsBatchGroupLeader(group, status)**:
  - Set status and done on all followers in the group; if **pending_** is non-empty, set **pending_.front()->is_leader = true**; else **leader_active_ = false**. **notify_all**.
- **DrainAndClose()**: Set **closed_ = true**, mark all pending writers Aborted and done, clear pending_, leader_active_ = false, notify_all. Called from **Close()** before closing the log writer so in-flight writers wake up with Aborted.

### 3.2 Merge and Single-Record Write (DBWal::Write)

- Leader calls **EnterAsBatchGroupLeader** (with **options_.max_write_group_size**).
- Under **writer_mu_**:
  - Build a single **WriteBatch** by **SetContents** for the first batch and **Append** for the rest (WriteBatchInternal).
  - Assign **first_seq = last_sequence_ + 1** and **total_count** = sum of counts; set **last_sequence_ += total_count**.
  - **WalCompressor::Compress** the merged batch data (always adds 1-byte prefix: 0x00 or 0x07 for zstd).
  - **log_writer_->AddRecord(compressed_slice)** once.
  - Then set each original batch’s sequence (first_seq, first_seq + count1, …) so callers see correct seq.
  - If **group.need_sync**: **log_writer_->file()->Sync()**.
  - If **max_wal_file_size** exceeded: **RotateLogFile()** (flush/close current, **NewLogFile()**).
- **ExitAsBatchGroupLeader(group, s)** so all followers get the same status.

### 3.3 Close() Ordering and Safety

1. **WriteThread::DrainAndClose()** so new joins get Aborted and waiters wake.
2. **shutdown_ = true**, **bg_purge_cv_.notify_all()**, **bg_purge_thread_.join()** if running.
3. **writer_mu_** lock, **closed_ = true**, flush and close **log_writer_**.
4. **env_->UnlockFile(dir_lock_.release())**.

---

## 4. Layer 4: WAL File Lifecycle (WalManager)

### 4.1 Roles

- **GetSortedWalFiles(files)**: Scan **wal_dir** for `\d{6}\.log` files, parse log number, get size; sort by log_number. Used by recovery, iterator, and purge.
- **PurgeObsoleteWALFiles()**: Under **mu_**, get **min_log_to_keep** from DBWal’s lambda, **now_seconds** from Env, and **total_wal_size**. For each file in sorted order, if **ShouldPurgeWal** returns true, delete file and subtract from total_size.
- **ShouldPurgeWal**: Do not purge if **log_number >= min_log_to_keep**. Else: if TTL set and (now_seconds - mtime) > WAL_ttl_seconds, purge; if size limit set and total_wal_size > limit, purge; else purge (obsolete by log number).

### 4.2 File Naming and Discovery

- Active log files: **000001.log**, **000002.log**, … (6-digit zero-padded log number).
- At **Open()**, DBWal calls **GetSortedWalFiles**; if non-empty, **log_number_** is set to the **max** existing log number so **NewLogFile()** creates the next number (no overwrite).
- **min_log_to_keep** is provided by the engine (e.g. after memtable flush) via **SetMinLogNumberToKeep**; purge respects it so files still needed for recovery are not deleted.

---

## 5. Layer 5: Record Format and I/O (log::Writer / log::Reader)

### 5.1 Block and Record Layout (log_format.h)

- **Block size**: 32 KiB (**kBlockSize**).
- **Record types**: kFullType (single block), kFirstType / kMiddleType / kLastType for multi-block records; recyclable variants add log number in header (mwal uses **recycle=false**).
- **Header** (non-recyclable): 4B CRC, 2B length, 1B type (**kHeaderSize = 7**). Recyclable adds 4B log number.

### 5.2 Writer (log_writer.cc)

- **AddRecord(slice)**: Split payload into **fragment_length** chunks that fit in (block - block_offset - header). For each fragment, choose type (Full / First / Middle / Last), then **EmitPhysicalRecord(type, ptr, len)**. Pad to block boundary if needed (zero padding). **block_offset_** tracks position within current block. If not **manual_flush**, **Flush** after each record.
- **EmitPhysicalRecord**: Encode length and type, compute CRC32C over type (and log number if recyclable), then over payload; append header + payload to **WritableFileWriter**.

### 5.3 Reader (log_reader)

- **ReadRecord(record, scratch, wal_recovery_mode)**: Read physical records (possibly fragmented); reassemble into one logical record; verify checksum. On corruption, behavior depends on **wal_recovery_mode** (e.g. skip tail, report, or stop).
- Uses **SequentialFileReader** (wraps Env’s SequentialFile) and a **Reporter** for corruption callbacks.

### 5.4 Buffering (WritableFileWriter / SequentialFileReader)

- **WritableFileWriter**: In-memory buffer (default 64 KiB); **Append** accumulates; **Flush** writes buffer to **WritableFile**; **Sync** flushes and calls **WritableFile::Sync()**.
- **SequentialFileReader**: Wraps **SequentialFile** for sequential read; used by log::Reader and recovery/iterator paths.

---

## 6. Layer 6: Compression and Payload Format

### 6.1 Compression Prefix (wal_compressor)

- Every record written by DBWal has a **1-byte prefix** before the WriteBatch bytes:
  - **0x00**: uncompressed (payload is raw WriteBatch).
  - **0x07**: zstd-compressed (payload is zstd-compressed WriteBatch).
- **WalCompressor::Compress**: For kNoCompression, output = prefix 0x00 + input. For kZSTD, prefix 0x07 + ZSTD_compress(level 1).
- **WalDecompressor::Decompress**: If first byte 0x00, strip byte and return rest; if 0x07, zstd decompress and return; else treat as legacy (no prefix) and return whole input.

### 6.2 WriteBatch Format (write_batch.h / write_batch.cc)

- **In-memory**: **rep_** = 12-byte header (8B sequence, 4B count) + sequence of operations. Each op: 1B type (Put/Delete/…) + varint key len + key + varint value len + value (for Put).
- **SetContents** / **Append** (WriteBatchInternal): Used to merge batches and to set payload after decompression during recovery/iteration.

---

## 7. Layer 7: Environment and Platform (Env)

### 7.1 Env Abstraction (env.h)

- **File I/O**: NewWritableFile, NewSequentialFile, DeleteFile, RenameFile, GetChildren, FileExists, GetFileSize, **GetFileModificationTime** (for TTL).
- **Time**: **NowMicros()** (POSIX uses **system_clock** so TTL matches file mtime).
- **Lock**: **LockFile(path, \*lock)** / **UnlockFile(lock)**. POSIX: **flock(LOCK_EX | LOCK_NB)** on **wal_dir/LOCK**; Unlock releases and removes LOCK file.

### 7.2 DBWal Use of Env

- **Open**: CreateDirIfMissing(wal_dir), LockFile(wal_dir + "/LOCK"), GetChildren for WalManager, NewWritableFile for new log, NewSequentialFile for recovery/iterator.
- **Close**: UnlockFile(dir_lock_).

---

## 8. Layer 8: Recovery and Iterator Details

### 8.1 Recover() and Auto-Recovery

- **Recover(callback)**: GetSortedWalFiles; for each file, NewSequentialFile → SequentialFileReader → log::Reader; loop ReadRecord → Decompress → SetContents(batch) → callback(seq, &batch). Track max sequence; set **last_sequence_**; close current log_writer_ if any; **NewLogFile()** for next writes.
- **Open()** with **recovery_callback** set and existing WALs: after discovering files, call **Recover(recovery_callback)** and skip the usual **NewLogFile()** until after Recover (Recover creates the new log at the end).

### 8.2 WalIterator

- **NewWalIterator(start_seq)**: GetSortedWalFiles, then construct **WalIterator(env, options, files, start_seq)**. Iterator holds a **snapshot** of the file list. This gives you logs from **start_seq to the current end** of the WAL at creation time (i.e. from a given sequence number up to whatever was latest when you called NewWalIterator).
- **AdvanceToStartSeq**: Read records until **current_seq >= start_seq** (skip earlier batches).
- **ReadNextRecord**: Open next file if needed (NewSequentialFile → log::Reader), ReadRecord, Decompress, SetContents into **current_batch**, set **current_seq** from batch. **Valid()** is true when a batch is loaded; **Next()** advances to next record (and skips records with seq < start_seq if used mid-iteration). Iteration continues until all snapshot files are read; then Valid() becomes false—so you naturally get every record from start_seq through the end of the snapshot.

---

## 9. Layer 9: Back-Pressure and Background Purge

### 9.1 Write Stall (max_total_wal_size)

- Before joining the write group, **Write()** checks **TotalWalSize()** (sum of sizes from GetSortedWalFiles). If **max_total_wal_size > 0** and total > limit, return **Status::Busy("write stall: total WAL size exceeded")**. Engine can retry after purge or compaction.

### 9.2 Background Purge Thread

- If **purge_interval_seconds > 0**, **Open()** starts a thread that: wait **purge_interval_seconds** (or until **shutdown_**), then call **PurgeObsoleteFiles()**; repeat until shutdown. **Close()** sets **shutdown_** and notifies so the thread exits and is joined.

---

## 10. File and Directory Layout (Reference)

```
wal_dir/
  LOCK              # Held by open DBWal (flock); prevents concurrent open
  000001.log        # First WAL file (block-aligned, checksummed records)
  000002.log        # After rotation
  ...
```

---

## 11. Corruption Handling

This section describes **where** corruption can be detected, **how** it is reported, and **how each path proceeds** (recovery, iterator, decompression, WriteBatch parsing). The write path does not detect corruption (it only produces data); all handling is on **read** paths.

### 11.1 Where Corruption Is Detected

| Layer | Location | What is checked | Failure reported as |
|-------|----------|-----------------|---------------------|
| **log::Reader** (physical WAL) | `log_reader.cc` | Block header (truncation), record type, checksum (CRC32C), record length, fragment sequence (First/Middle/Last), EOF in middle of record | `Reporter::Corruption(bytes, Status)`; `ReadRecord` returns `false` or skips and continues depending on mode |
| **WalDecompressor** | `wal_compressor.cc` | ZSTD frame (decompressed size unknown/error, decompress failure) | `Status::Corruption(...)` returned to caller |
| **WriteBatch::Iterate** | `write_batch.cc` | Header size, count, tag bytes, varint/key/value bounds | `Status::Corruption(...)` returned from callback / Iterate |

- **Reporter**: The log::Reader is given a **Reporter***. When it detects a physical-layer problem (bad header, checksum, fragment, etc.), it calls **Reporter::Corruption(size_t bytes, const Status&)**. The caller supplies the Reporter and can store the first corruption (e.g. **RecoveryReporter.first_corruption** in Recover, **SilentReporter.first_error** in WalIterator).

### 11.2 WALRecoveryMode Semantics (log::Reader)

Recovery mode controls **whether** to report corruption and **whether** to stop reading the current file or skip and continue. It is passed into **ReadRecord(..., wal_recovery_mode)**.

| Mode | Behavior summary |
|------|------------------|
| **kTolerateCorruptedTailRecords** | Prefer to **avoid** reporting tail corruptions: on checksum/length error in recyclable (tail) context, clear state and **return false** without calling Reporter. Other corruptions are still reported; Reader may **break** (skip one physical record and continue) or **return false** depending on case. |
| **kAbsoluteConsistency** | Treat any error as fatal for this file: **ReportCorruption** when applicable, then **return false** (e.g. truncated header, EOF in middle of record, truncated record at EOF, old record with in_fragmented_record, bad record length at EOF). Bad checksum / bad record: report and **break** (skip one block, continue). |
| **kPointInTimeRecovery** | Same as kAbsoluteConsistency for when to **report** and when to **return false**: report truncation/EOF-in-middle/truncated body and stop reading this file. Checksum/bad-record: report and **break** (skip, continue). |
| **kSkipAnyCorruptedRecords** | For **kOldRecord**: do **not** report, do **not** return false; **fallthrough** and treat like a skippable bad record (break). Other cases same as above (report + break or return false). |

So in practice:

- **TolerateCorruptedTailRecords**: Tail of file can be corrupt; Reader may stop without reporting. Used when you accept losing the last incomplete/corrupt record.
- **AbsoluteConsistency** / **PointInTimeRecovery**: Any truncation or EOF-in-middle is reported and stops the file read; checksum/bad-record skips one physical record and continues.
- **SkipAnyCorruptedRecords**: In addition, “old record” (wrong log number in recyclable header) is skipped without reporting and without stopping.

### 11.3 Recovery Path (DBWal::Recover)

- For each WAL file, Recover creates a **RecoveryReporter** (stores **first_corruption**), then runs **ReadRecord(..., options_.wal_recovery_mode)** in a loop.
- **When ReadRecord returns true**: Record is decompressed; if **WalDecompressor::Decompress** returns non-OK, Recover **returns that status** immediately (corruption from decompression). Otherwise WriteBatch is applied and **callback(seq, batch)** is called; if the callback returns non-OK (e.g. WriteBatch::Iterate found corruption), Recover **returns that status**.
- **When ReadRecord returns false**: Loop exits for **this file**. No decompression or callback for the rest of this file. Then:
  - If **reporter.first_corruption** is not OK **and** **wal_recovery_mode == kAbsoluteConsistency**, Recover **returns reporter.first_corruption** (whole Recover fails).
  - Otherwise, Recover **continues to the next WAL file** (and later creates a new log file and returns OK). So with **PointInTimeRecovery** or **TolerateCorruptedTailRecords**, one bad file does not fail the entire Recover; only **AbsoluteConsistency** turns the first reported corruption into a hard failure.

### 11.4 Iterator Path (WalIterator)

- Each file is read with a **SilentReporter** that stores **first_error** (first time **Corruption(...)** is called).
- **When ReadRecord returns false**:
  - If **first_error** is set **and** **wal_recovery_mode == kAbsoluteConsistency**, iterator sets **impl_->status_ = first_error** and **Valid()** becomes false; **status()** returns that corruption.
  - Otherwise, iterator **resets the reader** and **continues** (tries next file). So with **PointInTimeRecovery** or **TolerateCorruptedTailRecords**, one bad file is skipped and the iterator moves on.
- **When WalDecompressor::Decompress** returns non-OK: iterator sets **impl_->status_** to that status and returns false (stops iteration; **status()** returns the decompression corruption).

### 11.5 Decompression and WriteBatch Corruption

- **Decompression** (ZSTD): Failures (e.g. “Cannot determine decompressed size”, “ZSTD decompression failed”) are **Status::Corruption**. In **Recover** they abort the whole Recover. In **WalIterator** they set **status()** and stop the iterator.
- **WriteBatch**: If the recovery **callback** calls **batch->Iterate(handler)** and the batch is malformed, **Iterate** can return **Status::Corruption** (e.g. “malformed WriteBatch”, “bad WriteBatch Put”). That is returned by the callback; **Recover** then returns that status and stops. The iterator does not parse the batch for corruption itself; it only surfaces decompression and, if mode is AbsoluteConsistency, log::Reader reporter errors.

### 11.6 Summary: What Happens When Corruption Is Found

| Path | Where | If corruption / error | What happens next |
|------|--------|------------------------|-------------------|
| **Recover** | log::Reader (ReadRecord returns false) | first_corruption set; mode == **AbsoluteConsistency** | **Return** first_corruption; Recover fails. |
| **Recover** | log::Reader (ReadRecord returns false) | first_corruption set; mode != AbsoluteConsistency | **Continue** to next WAL file; Recover can still succeed. |
| **Recover** | WalDecompressor::Decompress | Returns Corruption | **Return** that status; Recover fails. |
| **Recover** | callback (e.g. batch->Iterate) | Returns Corruption | **Return** that status; Recover fails. |
| **WalIterator** | log::Reader (ReadRecord returns false) | first_error set; mode == **AbsoluteConsistency** | Set **status()**, **Valid()** = false; iteration stops. |
| **WalIterator** | log::Reader (ReadRecord returns false) | first_error set; mode != AbsoluteConsistency | **Continue** to next file. |
| **WalIterator** | WalDecompressor::Decompress | Returns Corruption | Set **status()**, **Valid()** = false; iteration stops. |

---

## 12. Summary Table: Component Responsibilities

| Component | Responsibility |
|-----------|----------------|
| **DBWal** | Public API, sequence numbers, dir lock, log rotation, merge + compress + single AddRecord, recovery orchestration, bg purge, write stall. |
| **WriteThread** | Leader election, group formation (with size cap), follower wait/wake, DrainAndClose on shutdown. |
| **WalManager** | List and sort WAL files, purge by TTL/size/min_log_to_keep. |
| **log::Writer** | Block format, fragmentation, CRC, write to WritableFileWriter. |
| **log::Reader** | Read and reassemble records, checksum, recovery mode. |
| **WalCompressor / WalDecompressor** | 1-byte prefix + optional zstd on record payload. |
| **WalIterator** | Snapshot of files, sequential read, decompress, skip to start_seq. |
| **Env / PosixEnv** | File and directory ops, time (system_clock), LockFile/UnlockFile (flock). |
| **WritableFileWriter** | Buffered append and flush/sync. |
| **WriteBatch / WriteBatchInternal** | Serialization format, merge (Append), sequence/count in header. |

This design document, together with the source under **mwal/** (include and src), should give an engineer a complete path from high-level architecture down to the internal mechanics of the WAL system.
