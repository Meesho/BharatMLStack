#include <gtest/gtest.h>

#include <chrono>
#include <fstream>
#include <map>
#include <memory>
#include <string>

#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/write_batch.h"
#include "test_util.h"
#include "wal/wal_manager.h"

namespace mwal {

namespace {

// Env wrapper for TC-WALMGR-04: override NowMicros and GetFileModificationTime.
class MockTimeEnv : public Env {
 public:
  explicit MockTimeEnv(Env* base) : base_(base) {}

  void SetNowSeconds(uint64_t s) { now_seconds_ = s; }
  void SetFileMtime(const std::string& path, uint64_t mtime) {
    mtime_map_[path] = mtime;
  }

  uint64_t NowMicros() override { return now_seconds_ * 1000000ULL; }
  Status GetFileModificationTime(const std::string& fname,
                                 uint64_t* file_mtime) override {
    auto it = mtime_map_.find(fname);
    if (it != mtime_map_.end()) {
      *file_mtime = it->second;
      return Status::OK();
    }
    return base_->GetFileModificationTime(fname, file_mtime);
  }

  Status NewWritableFile(const std::string& fname,
                        std::unique_ptr<WritableFile>* result,
                        const EnvOptions& options) override {
    return base_->NewWritableFile(fname, result, options);
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
  Status LockFile(const std::string& fname, FileLock** lock) override {
    return base_->LockFile(fname, lock);
  }
  Status UnlockFile(FileLock* lock) override {
    return base_->UnlockFile(lock);
  }

 private:
  Env* base_;
  uint64_t now_seconds_ = 0;
  std::map<std::string, uint64_t> mtime_map_;
};

}  // namespace

class WalManagerTest : public ::testing::Test {
 protected:
  test::TempDir tmpdir_;
  Env* env_ = Env::Default();

  void CreateWalFile(uint64_t log_num, size_t size = 100) {
    std::string path = test::TempFileName(tmpdir_.path(), log_num);
    std::unique_ptr<WritableFile> wf;
    ASSERT_TRUE(env_->NewWritableFile(path, &wf, EnvOptions()).ok());
    std::string data(size, 'x');
    wf->Append(Slice(data));
    wf->Close();
  }
};

TEST_F(WalManagerTest, GetSortedWalFiles) {
  CreateWalFile(3);
  CreateWalFile(1);
  CreateWalFile(7);

  WALOptions opts;
  opts.wal_dir = tmpdir_.path();
  WalManager mgr(env_, opts, []() -> uint64_t { return 0; });

  std::vector<WalFileInfo> files;
  ASSERT_TRUE(mgr.GetSortedWalFiles(&files).ok());
  ASSERT_EQ(files.size(), 3u);
  EXPECT_EQ(files[0].log_number, 1u);
  EXPECT_EQ(files[1].log_number, 3u);
  EXPECT_EQ(files[2].log_number, 7u);
}

TEST_F(WalManagerTest, PurgeObsolete) {
  CreateWalFile(1);
  CreateWalFile(2);
  CreateWalFile(3);
  CreateWalFile(5);

  WALOptions opts;
  opts.wal_dir = tmpdir_.path();
  uint64_t min_to_keep = 3;
  WalManager mgr(env_, opts,
                  [&min_to_keep]() -> uint64_t { return min_to_keep; });

  ASSERT_TRUE(mgr.PurgeObsoleteWALFiles().ok());

  std::vector<WalFileInfo> files;
  ASSERT_TRUE(mgr.GetSortedWalFiles(&files).ok());
  ASSERT_EQ(files.size(), 2u);
  EXPECT_EQ(files[0].log_number, 3u);
  EXPECT_EQ(files[1].log_number, 5u);
}

TEST_F(WalManagerTest, EmptyDir) {
  WALOptions opts;
  opts.wal_dir = tmpdir_.path();
  WalManager mgr(env_, opts, []() -> uint64_t { return 0; });

  std::vector<WalFileInfo> files;
  ASSERT_TRUE(mgr.GetSortedWalFiles(&files).ok());
  EXPECT_TRUE(files.empty());
}

TEST_F(WalManagerTest, ZeroByteFile) {
  CreateWalFile(1, 0);
  CreateWalFile(2, 100);

  WALOptions opts;
  opts.wal_dir = tmpdir_.path();
  WalManager mgr(env_, opts, []() -> uint64_t { return 0; });

  std::vector<WalFileInfo> files;
  ASSERT_TRUE(mgr.GetSortedWalFiles(&files).ok());
  ASSERT_EQ(files.size(), 2u);
  EXPECT_EQ(files[0].size_bytes, 0u);
  EXPECT_EQ(files[1].size_bytes, 100u);
}

TEST_F(WalManagerTest, DeleteWalFile) {
  CreateWalFile(10);
  std::string path = test::TempFileName(tmpdir_.path(), 10);

  WALOptions opts;
  opts.wal_dir = tmpdir_.path();
  WalManager mgr(env_, opts, []() -> uint64_t { return 0; });

  ASSERT_TRUE(env_->FileExists(path).ok());
  ASSERT_TRUE(mgr.DeleteWalFile(path).ok());
  EXPECT_FALSE(env_->FileExists(path).ok());
}

TEST_F(WalManagerTest, PurgeSizeLimit) {
  CreateWalFile(1, 500);
  CreateWalFile(2, 500);
  CreateWalFile(3, 500);

  WALOptions opts;
  opts.wal_dir = tmpdir_.path();

  // min_log_to_keep=4 means all files (1,2,3) qualify for purge.
  uint64_t min_keep = 4;
  WalManager mgr(env_, opts, [&min_keep]() -> uint64_t { return min_keep; });

  ASSERT_TRUE(mgr.PurgeObsoleteWALFiles().ok());
  std::vector<WalFileInfo> files;
  ASSERT_TRUE(mgr.GetSortedWalFiles(&files).ok());
  EXPECT_EQ(files.size(), 0u);
}

TEST_F(WalManagerTest, PurgeRespectsMinLog) {
  CreateWalFile(1, 100);
  CreateWalFile(5, 100);
  CreateWalFile(10, 100);

  WALOptions opts;
  opts.wal_dir = tmpdir_.path();
  uint64_t min_keep = 5;
  WalManager mgr(env_, opts, [&min_keep]() -> uint64_t { return min_keep; });

  ASSERT_TRUE(mgr.PurgeObsoleteWALFiles().ok());
  std::vector<WalFileInfo> files;
  ASSERT_TRUE(mgr.GetSortedWalFiles(&files).ok());
  ASSERT_EQ(files.size(), 2u);
  EXPECT_EQ(files[0].log_number, 5u);
  EXPECT_EQ(files[1].log_number, 10u);
}

// --- Section 8: WalManager Edge Cases ---

// TC-WALMGR-01: Non-log files in directory are ignored.
TEST_F(WalManagerTest, TC_WALMGR_01_NonLogFilesIgnored) {
  ASSERT_TRUE(env_->CreateDirIfMissing(tmpdir_.path()).ok());

  CreateWalFile(3, 100);

  std::string lock_path = tmpdir_.path() + "/LOCK";
  {
    std::unique_ptr<WritableFile> wf;
    ASSERT_TRUE(env_->NewWritableFile(lock_path, &wf, EnvOptions()).ok());
    wf->Append(Slice("x"));
    wf->Close();
  }
  std::string manifest_path = tmpdir_.path() + "/manifest.tmp";
  {
    std::unique_ptr<WritableFile> wf;
    ASSERT_TRUE(env_->NewWritableFile(manifest_path, &wf, EnvOptions()).ok());
    wf->Close();
  }
  std::string notes_path = tmpdir_.path() + "/notes.txt";
  {
    std::unique_ptr<WritableFile> wf;
    ASSERT_TRUE(env_->NewWritableFile(notes_path, &wf, EnvOptions()).ok());
    wf->Close();
  }
  std::string subdir_path = tmpdir_.path() + "/subdir";
  ASSERT_TRUE(env_->CreateDir(subdir_path).ok());

  WALOptions opts;
  opts.wal_dir = tmpdir_.path();
  WalManager mgr(env_, opts, []() -> uint64_t { return 0; });

  std::vector<WalFileInfo> files;
  Status s = mgr.GetSortedWalFiles(&files);
  ASSERT_TRUE(s.ok()) << s.ToString();
  ASSERT_EQ(files.size(), 1u);
  EXPECT_EQ(files[0].log_number, 3u);
  EXPECT_EQ(files[0].path, tmpdir_.path() + "/000003.log");
}

// TC-WALMGR-02: Log number gaps sorted correctly; PurgeObsoleteFiles with min_log_to_keep=5.
TEST_F(WalManagerTest, TC_WALMGR_02_LogNumberGapsSortedCorrectly) {
  CreateWalFile(1, 50);
  CreateWalFile(5, 50);
  CreateWalFile(9, 50);

  WALOptions opts;
  opts.wal_dir = tmpdir_.path();
  WalManager mgr(env_, opts, []() -> uint64_t { return 0; });

  std::vector<WalFileInfo> files;
  ASSERT_TRUE(mgr.GetSortedWalFiles(&files).ok());
  ASSERT_EQ(files.size(), 3u);
  EXPECT_EQ(files[0].log_number, 1u);
  EXPECT_EQ(files[1].log_number, 5u);
  EXPECT_EQ(files[2].log_number, 9u);

  uint64_t min_keep = 5;
  WalManager mgr2(env_, opts, [&min_keep]() -> uint64_t { return min_keep; });
  ASSERT_TRUE(mgr2.PurgeObsoleteWALFiles().ok());

  std::vector<WalFileInfo> after;
  ASSERT_TRUE(mgr2.GetSortedWalFiles(&after).ok());
  ASSERT_EQ(after.size(), 2u);
  EXPECT_EQ(after[0].log_number, 5u);
  EXPECT_EQ(after[1].log_number, 9u);

  EXPECT_FALSE(env_->FileExists(test::TempFileName(tmpdir_.path(), 1)).ok());
  EXPECT_TRUE(env_->FileExists(test::TempFileName(tmpdir_.path(), 5)).ok());
  EXPECT_TRUE(env_->FileExists(test::TempFileName(tmpdir_.path(), 9)).ok());
}

// TC-WALMGR-03: Purge does not delete the active log file.
TEST_F(WalManagerTest, TC_WALMGR_03_PurgeDoesNotDeleteActiveLog) {
  WALOptions opts;
  opts.wal_dir = tmpdir_.path();
  opts.max_wal_file_size = 512;

  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

  std::string val(200, 'x');
  for (int i = 0; i < 15; i++) {
    WriteBatch batch;
    batch.Put("k" + std::to_string(i), val);
    ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
  }

  uint64_t current_log = wal->GetCurrentLogNumber();
  ASSERT_GE(current_log, 3u) << "Need at least 3 log files";

  wal->SetMinLogNumberToKeep(current_log);
  ASSERT_TRUE(wal->PurgeObsoleteFiles().ok());

  WriteBatch batch;
  batch.Put("after_purge", "ok");
  ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());

  std::vector<WalFileInfo> files;
  ASSERT_TRUE(wal->GetLiveWalFiles(&files).ok());
  ASSERT_EQ(files.size(), 1u);
  EXPECT_EQ(files[0].log_number, current_log);

  wal->Close();
}

// TC-WALMGR-04: TTL and size limit both set — only file 1 (TTL expired) deleted.
TEST_F(WalManagerTest, TC_WALMGR_04_TTLAndSizeLimitBothApply) {
  MockTimeEnv mock_env(env_);
  uint64_t now_sec = 4000;
  mock_env.SetNowSeconds(now_sec);

  std::string p1 = tmpdir_.path() + "/000001.log";
  std::string p2 = tmpdir_.path() + "/000002.log";
  std::string p3 = tmpdir_.path() + "/000003.log";

  {
    std::unique_ptr<WritableFile> wf;
    ASSERT_TRUE(mock_env.NewWritableFile(p1, &wf, EnvOptions()).ok());
    ASSERT_TRUE(wf->Append(Slice(std::string(400 * 1024, 'x'))).ok());
    wf->Close();
  }
  {
    std::unique_ptr<WritableFile> wf;
    ASSERT_TRUE(mock_env.NewWritableFile(p2, &wf, EnvOptions()).ok());
    ASSERT_TRUE(wf->Append(Slice(std::string(400 * 1024, 'x'))).ok());
    wf->Close();
  }
  {
    std::unique_ptr<WritableFile> wf;
    ASSERT_TRUE(mock_env.NewWritableFile(p3, &wf, EnvOptions()).ok());
    ASSERT_TRUE(wf->Append(Slice(std::string(100 * 1024, 'x'))).ok());
    wf->Close();
  }

  mock_env.SetFileMtime(p1, now_sec - 4000);
  mock_env.SetFileMtime(p2, now_sec - 100);
  mock_env.SetFileMtime(p3, now_sec - 100);

  WALOptions opts;
  opts.wal_dir = tmpdir_.path();
  opts.WAL_ttl_seconds = 3600;
  opts.WAL_size_limit_MB = 1;

  uint64_t min_keep = 2;
  WalManager mgr(&mock_env, opts, [&min_keep]() -> uint64_t { return min_keep; });
  ASSERT_TRUE(mgr.PurgeObsoleteWALFiles().ok());

  std::vector<WalFileInfo> files;
  ASSERT_TRUE(mgr.GetSortedWalFiles(&files).ok());
  ASSERT_EQ(files.size(), 2u);
  EXPECT_EQ(files[0].log_number, 2u);
  EXPECT_EQ(files[1].log_number, 3u);

  EXPECT_FALSE(mock_env.FileExists(p1).ok());
  EXPECT_TRUE(mock_env.FileExists(p2).ok());
  EXPECT_TRUE(mock_env.FileExists(p3).ok());
}

// TC-WALMGR-05: Directory scan with large number of files.
TEST_F(WalManagerTest, TC_WALMGR_05_LargeNumberOfFiles) {
  for (uint64_t i = 1; i <= 500; i++) {
    CreateWalFile(i, 0);
  }

  WALOptions opts;
  opts.wal_dir = tmpdir_.path();
  WalManager mgr(env_, opts, []() -> uint64_t { return 0; });

  std::vector<WalFileInfo> files;
  ASSERT_TRUE(mgr.GetSortedWalFiles(&files).ok());
  ASSERT_EQ(files.size(), 500u);

  for (size_t i = 0; i < files.size(); i++) {
    EXPECT_EQ(files[i].log_number, static_cast<uint64_t>(i + 1));
  }

  auto start = std::chrono::steady_clock::now();
  for (int i = 0; i < 100; i++) {
    std::vector<WalFileInfo> tmp;
    ASSERT_TRUE(mgr.GetSortedWalFiles(&tmp).ok());
  }
  auto end = std::chrono::steady_clock::now();
  auto elapsed_ms =
      std::chrono::duration_cast<std::chrono::milliseconds>(end - start)
          .count();
  double avg_ms = elapsed_ms / 100.0;
  if (avg_ms > 50) {
    GTEST_LOG_(INFO) << "TC-WALMGR-05: GetSortedWalFiles avg " << avg_ms
                     << " ms over 100 runs (500 files); baseline for O(n) scan";
  }
}

}  // namespace mwal
