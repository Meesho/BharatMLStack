// Section 9: File System / Env Error Injection
// FaultInjectionEnv wrapper and TC-ENV-01 through TC-ENV-07.

#include <gtest/gtest.h>

#include <sys/stat.h>
#include <atomic>
#include <memory>
#include <string>

#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/write_batch.h"
#include "test_util.h"

namespace mwal {

namespace {

// WritableFile wrapper that can fail Append or Sync on the Nth call.
class FaultInjectionWritableFile : public WritableFile {
 public:
  FaultInjectionWritableFile(std::unique_ptr<WritableFile>&& real,
                             std::atomic<int>* append_count,
                             std::atomic<int>* sync_count, int fail_append_on,
                             int fail_sync_on, const Status& fail_status)
      : real_(std::move(real)),
        append_count_(append_count),
        sync_count_(sync_count),
        fail_append_on_(fail_append_on),
        fail_sync_on_(fail_sync_on),
        fail_status_(fail_status) {}

  IOStatus Append(const Slice& data) override {
    int n = append_count_->fetch_add(1) + 1;
    if (fail_append_on_ > 0 && n == fail_append_on_) {
      return IOStatus::IOError(fail_status_.ToString());
    }
    return real_->Append(data);
  }

  IOStatus Close() override { return real_->Close(); }
  IOStatus Flush() override { return real_->Flush(); }
  IOStatus Sync() override {
    int n = sync_count_->fetch_add(1) + 1;
    if (fail_sync_on_ > 0 && n == fail_sync_on_) {
      return IOStatus::IOError(fail_status_.ToString());
    }
    return real_->Sync();
  }
  uint64_t GetFileSize() const override { return real_->GetFileSize(); }

 private:
  std::unique_ptr<WritableFile> real_;
  std::atomic<int>* append_count_;
  std::atomic<int>* sync_count_;
  int fail_append_on_;
  int fail_sync_on_;
  Status fail_status_;
};

// Env wrapper that fails specific operations after N successful calls.
class FaultInjectionEnv : public Env {
 public:
  explicit FaultInjectionEnv(Env* base) : base_(base) {}

  void SetFailNewWritableFileOnCall(int n) { fail_nwf_on_call_ = n; }
  void SetFailGetFileModificationTimeOnCall(int n) {
    fail_getmtime_on_call_ = n;
  }
  void SetFailGetFileModificationTimeForPath(const std::string& path) {
    fail_getmtime_path_ = path;
  }
  void SetFailAppendOnCall(int n) { fail_append_on_call_ = n; }
  void SetFailSyncOnCall(int n) { fail_sync_on_call_ = n; }
  void ResetCounts() {
    nwf_count_.store(0);
    getmtime_count_.store(0);
    append_count_.store(0);
    sync_count_.store(0);
  }
  int GetNewWritableFileCount() const { return nwf_count_.load(); }

  Status NewWritableFile(const std::string& fname,
                        std::unique_ptr<WritableFile>* result,
                        const EnvOptions& options) override {
    int n = ++nwf_count_;
    if (fail_nwf_on_call_ > 0 && n == fail_nwf_on_call_) {
      return Status::IOError("disk full");
    }
    Status s = base_->NewWritableFile(fname, result, options);
    if (!s.ok()) return s;
    if (fail_append_on_call_ > 0 || fail_sync_on_call_ > 0) {
      result->reset(new FaultInjectionWritableFile(
          std::move(*result), &append_count_, &sync_count_,
          fail_append_on_call_, fail_sync_on_call_,
          Status::IOError("append/sync failed")));
    }
    return Status::OK();
  }

  Status GetFileModificationTime(const std::string& fname,
                                 uint64_t* file_mtime) override {
    if (!fail_getmtime_path_.empty() && fname == fail_getmtime_path_) {
      return Status::IOError("getmtime failed");
    }
    int n = ++getmtime_count_;
    if (fail_getmtime_on_call_ > 0 && n == fail_getmtime_on_call_) {
      return Status::IOError("getmtime failed");
    }
    return base_->GetFileModificationTime(fname, file_mtime);
  }

  Status NewSequentialFile(const std::string& fname,
                           std::unique_ptr<SequentialFile>* result,
                           const EnvOptions& options) override {
    return base_->NewSequentialFile(fname, result, options);
  }
  Status DeleteFile(const std::string& fname) override {
    return base_->DeleteFile(fname);
  }
  Status RenameFile(const std::string& src, const std::string& target) override {
    return base_->RenameFile(src, target);
  }
  Status GetChildren(const std::string& dir,
                     std::vector<std::string>* result) override {
    return base_->GetChildren(dir, result);
  }
  Status FileExists(const std::string& fname) override {
    return base_->FileExists(fname);
  }
  Status GetFileSize(const std::string& fname, uint64_t* size) override {
    return base_->GetFileSize(fname, size);
  }
  Status CreateDir(const std::string& dirname) override {
    return base_->CreateDir(dirname);
  }
  Status CreateDirIfMissing(const std::string& dirname) override {
    return base_->CreateDirIfMissing(dirname);
  }
  uint64_t NowMicros() override { return base_->NowMicros(); }
  Status LockFile(const std::string& fname, FileLock** lock) override {
    return base_->LockFile(fname, lock);
  }
  Status UnlockFile(FileLock* lock) override {
    return base_->UnlockFile(lock);
  }

 private:
  Env* base_;
  int fail_nwf_on_call_ = 0;
  int fail_getmtime_on_call_ = 0;
  std::string fail_getmtime_path_;
  int fail_append_on_call_ = 0;
  int fail_sync_on_call_ = 0;
  mutable std::atomic<int> nwf_count_{0};
  std::atomic<int> getmtime_count_{0};
  std::atomic<int> append_count_{0};
  std::atomic<int> sync_count_{0};
};

}  // namespace

class EnvErrorInjectionTest : public ::testing::Test {
 protected:
  void SetUp() override {
    base_env_ = Env::Default();
    fault_env_ = std::make_unique<FaultInjectionEnv>(base_env_);
    env_ = fault_env_.get();
  }

  WALOptions MakeOptions() {
    WALOptions opts;
    opts.wal_dir = tmpdir_.path();
    return opts;
  }

  test::TempDir tmpdir_;
  Env* base_env_ = nullptr;
  std::unique_ptr<FaultInjectionEnv> fault_env_;
  Env* env_ = nullptr;
};

// TC-ENV-01: NewWritableFile fails at Open()
TEST_F(EnvErrorInjectionTest, TC_ENV_01_NewWritableFileFailsAtOpen) {
  fault_env_->SetFailNewWritableFileOnCall(1);

  std::unique_ptr<DBWal> wal;
  Status s = DBWal::Open(MakeOptions(), env_, &wal);
  ASSERT_FALSE(s.ok());
  EXPECT_TRUE(s.IsIOError()) << s.ToString();

  std::string lock_path = tmpdir_.path() + "/LOCK";
  EXPECT_FALSE(base_env_->FileExists(lock_path).ok());

  std::string log_path = tmpdir_.path() + "/000001.log";
  EXPECT_FALSE(base_env_->FileExists(log_path).ok());

  fault_env_->SetFailNewWritableFileOnCall(0);
  ASSERT_TRUE(DBWal::Open(MakeOptions(), base_env_, &wal).ok());
  wal->Close();
}

// TC-ENV-02: NewWritableFile fails during RotateLogFile()
TEST_F(EnvErrorInjectionTest, TC_ENV_02_NewWritableFileFailsDuringRotation) {
  WALOptions opts = MakeOptions();
  opts.max_wal_file_size = 128;

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  WriteBatch batch1;
  batch1.Put("k1", std::string(80, 'x'));
  ASSERT_TRUE(wal->Write(WriteOptions(), &batch1).ok());

  fault_env_->SetFailNewWritableFileOnCall(
      fault_env_->GetNewWritableFileCount() + 1);

  WriteBatch batch;
  batch.Put("trigger_rotation", std::string(80, 'y'));
  Status s = wal->Write(WriteOptions(), &batch);
  ASSERT_FALSE(s.ok()) << s.ToString();

  wal->Close();

  opts.wal_recovery_mode = WALRecoveryMode::kPointInTimeRecovery;
  int count = 0;
  opts.recovery_callback = [&count](SequenceNumber, WriteBatch*) {
    count++;
    return Status::OK();
  };
  std::unique_ptr<DBWal> wal2;
  ASSERT_TRUE(DBWal::Open(opts, base_env_, &wal2).ok());
  EXPECT_GE(count, 1u);
  wal2->Close();
}

// TC-ENV-03: Append fails mid-write
TEST_F(EnvErrorInjectionTest, TC_ENV_03_AppendFailsMidWrite) {
  fault_env_->SetFailAppendOnCall(2);

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());

  WriteBatch batch1;
  batch1.Put("k1", "v1");
  ASSERT_TRUE(wal->Write(WriteOptions(), &batch1).ok());

  WriteBatch batch2;
  batch2.Put("k2", std::string(70000, 'v'));
  Status s = wal->Write(WriteOptions(), &batch2);
  ASSERT_FALSE(s.ok());
  EXPECT_TRUE(s.IsIOError()) << s.ToString();

  wal->Close();

  WALOptions opts = MakeOptions();
  opts.wal_recovery_mode = WALRecoveryMode::kTolerateCorruptedTailRecords;
  int count = 0;
  opts.recovery_callback = [&count](SequenceNumber, WriteBatch*) {
    count++;
    return Status::OK();
  };
  std::unique_ptr<DBWal> wal2;
  ASSERT_TRUE(DBWal::Open(opts, base_env_, &wal2).ok());
  EXPECT_EQ(count, 1);

  WriteBatch batch3;
  batch3.Put("k3", "v3");
  ASSERT_TRUE(wal2->Write(WriteOptions(), &batch3).ok());
  wal2->Close();
}

// TC-ENV-04: Sync fails on SyncWAL()
TEST_F(EnvErrorInjectionTest, TC_ENV_04_SyncFailsOnSyncWAL) {
  fault_env_->SetFailSyncOnCall(1);

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());

  WriteBatch batch;
  batch.Put("k", "v");
  ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());

  Status s = wal->SyncWAL();
  ASSERT_FALSE(s.ok());
  EXPECT_TRUE(s.IsIOError()) << s.ToString();

  fault_env_->SetFailSyncOnCall(0);
  s = wal->SyncWAL();
  ASSERT_TRUE(s.ok()) << s.ToString();

  wal->Close();
}

// TC-ENV-05: GetFileModificationTime fails during TTL purge
TEST_F(EnvErrorInjectionTest, TC_ENV_05_GetFileModificationTimeFailsDuringPurge) {
  std::string p1 = tmpdir_.path() + "/000001.log";
  std::string p2 = tmpdir_.path() + "/000002.log";
  {
    std::unique_ptr<WritableFile> wf;
    ASSERT_TRUE(base_env_->NewWritableFile(p1, &wf, EnvOptions()).ok());
    wf->Append(Slice("x"));
    wf->Close();
  }
  {
    std::unique_ptr<WritableFile> wf;
    ASSERT_TRUE(base_env_->NewWritableFile(p2, &wf, EnvOptions()).ok());
    wf->Append(Slice("y"));
    wf->Close();
  }

  fault_env_->SetFailGetFileModificationTimeForPath(p1);

  WALOptions opts;
  opts.wal_dir = tmpdir_.path();
  opts.WAL_ttl_seconds = 1;
  opts.WAL_size_limit_MB = 0;

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());
  wal->SetMinLogNumberToKeep(3);

  Status s = wal->PurgeObsoleteFiles();

  EXPECT_TRUE(base_env_->FileExists(p1).ok())
      << "Spec: file with unreadable mtime should be kept (conservative).";
  wal->Close();
}

// TC-ENV-06: Directory does not exist at Open()
TEST_F(EnvErrorInjectionTest, TC_ENV_06_DirectoryDoesNotExistAtOpen) {
  std::string wal_dir = tmpdir_.path() + "/nonexistent_subdir";
  EXPECT_FALSE(base_env_->FileExists(wal_dir).ok());

  WALOptions opts;
  opts.wal_dir = wal_dir;

  std::unique_ptr<DBWal> wal;
  Status s = DBWal::Open(opts, base_env_, &wal);
  ASSERT_TRUE(s.ok()) << s.ToString();

  WriteBatch batch;
  batch.Put("k", "v");
  ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
  ASSERT_TRUE(wal->Close().ok());

  EXPECT_TRUE(base_env_->FileExists(wal_dir).ok());
}

// TC-ENV-07: Read-only directory at Open()
TEST_F(EnvErrorInjectionTest, TC_ENV_07_ReadOnlyDirectoryAtOpen) {
  std::string ro_dir = tmpdir_.path() + "/readonly";
  ASSERT_TRUE(base_env_->CreateDir(ro_dir).ok());
  ASSERT_EQ(chmod(ro_dir.c_str(), 0444), 0);

  WALOptions opts;
  opts.wal_dir = ro_dir;

  std::unique_ptr<DBWal> wal;
  Status s = DBWal::Open(opts, base_env_, &wal);
  ASSERT_FALSE(s.ok()) << s.ToString();
  EXPECT_TRUE(s.IsIOError() || s.ToString().find("Permission") != std::string::npos ||
              s.ToString().find("permission") != std::string::npos)
      << s.ToString();

  chmod(ro_dir.c_str(), 0755);
}

}  // namespace mwal
