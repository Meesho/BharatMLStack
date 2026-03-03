// Section 11: Operational / Tooling Tests
// TC-OPS-01 through TC-OPS-06.

#include <gtest/gtest.h>

#include <csignal>
#include <cstdlib>
#include <cstring>
#include <random>
#include <string>
#include <unistd.h>

#include "mwal/compression_type.h"
#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/write_batch.h"
#include "test_util.h"

namespace mwal {

class OperationalTest : public ::testing::Test {
 protected:
  void SetUp() override { env_ = Env::Default(); }

  WALOptions MakeOptions() {
    WALOptions opts;
    opts.wal_dir = tmpdir_.path();
    return opts;
  }

  test::TempDir tmpdir_;
  Env* env_ = nullptr;
};

#ifdef MWAL_HAVE_ZSTD
// TC-OPS-01: Open WAL written with compression using no-compression options
TEST_F(OperationalTest, TC_OPS_01_OpenCompressedWithNoCompressionOptions) {
  {
    WALOptions opts = MakeOptions();
    opts.wal_compression = kZSTD;

    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(opts, env_, &wal).ok());

    for (int i = 0; i < 10; i++) {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), "v" + std::to_string(i));
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    wal->Close();
  }

  WALOptions opts = MakeOptions();
  opts.wal_compression = kNoCompression;
  int count = 0;
  opts.recovery_callback = [&count](SequenceNumber, WriteBatch* batch) {
    count++;
    struct NoOpHandler : public WriteBatch::Handler {
      Status Put(const Slice&, const Slice&) override { return Status::OK(); }
      Status Delete(const Slice&) override { return Status::OK(); }
    } handler;
    return batch->Iterate(&handler);
  };

  std::unique_ptr<DBWal> wal;
  Status s = DBWal::Open(opts, env_, &wal);
  ASSERT_TRUE(s.ok()) << s.ToString();
  EXPECT_EQ(count, 10);
  wal->Close();
}
#endif

// TC-OPS-02: Open non-WAL file as WAL
TEST_F(OperationalTest, TC_OPS_02_OpenNonWalFileAsWal) {
  std::string log_path = tmpdir_.path() + "/000001.log";
  {
    std::unique_ptr<WritableFile> wf;
    ASSERT_TRUE(env_->NewWritableFile(log_path, &wf, EnvOptions()).ok());
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_int_distribution<> dis(0, 255);
    std::string random_data(4096, '\0');
    for (char& c : random_data) c = static_cast<char>(dis(gen));
    wf->Append(Slice(random_data));
    wf->Close();
  }

  WALOptions opts = MakeOptions();
  opts.wal_recovery_mode = WALRecoveryMode::kAbsoluteConsistency;
  int callback_count = 0;
  opts.recovery_callback = [&callback_count](SequenceNumber, WriteBatch*) {
    callback_count++;
    return Status::OK();
  };

  std::unique_ptr<DBWal> wal;
  Status s = DBWal::Open(opts, env_, &wal);
  if (s.ok()) {
    EXPECT_EQ(callback_count, 0);
  } else {
    EXPECT_TRUE(s.IsCorruption()) << s.ToString();
  }
}

// TC-OPS-03: Stale LOCK file from crashed process
TEST_F(OperationalTest, TC_OPS_03_StaleLockFromCrashedProcess) {
  std::string wal_dir = tmpdir_.path();
  int pipefd[2];
  ASSERT_EQ(pipe(pipefd), 0);

  pid_t pid = fork();
  if (pid == 0) {
    close(pipefd[0]);
    WALOptions opts;
    opts.wal_dir = wal_dir;

    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, env_, &wal);
    if (!s.ok()) _exit(1);

    WriteBatch batch;
    batch.Put("before_crash", "data");
    if (!wal->Write(WriteOptions(), &batch).ok()) _exit(2);

    write(pipefd[1], "x", 1);
    close(pipefd[1]);
    pause();
    _exit(0);
  }

  close(pipefd[1]);
  char c;
  ASSERT_EQ(read(pipefd[0], &c, 1), 1);
  close(pipefd[0]);

  ASSERT_EQ(kill(pid, SIGKILL), 0);
  int status;
  ASSERT_EQ(waitpid(pid, &status, 0), pid);
  ASSERT_TRUE(WIFSIGNALED(status) && WTERMSIG(status) == SIGKILL);

  WALOptions opts;
  opts.wal_dir = wal_dir;
  opts.recovery_callback = [](SequenceNumber, WriteBatch*) {
    return Status::OK();
  };

  std::unique_ptr<DBWal> wal;
  Status s = DBWal::Open(opts, env_, &wal);
  ASSERT_TRUE(s.ok()) << s.ToString();

  int count = 0;
  Status rs = wal->Recover([&count](SequenceNumber, WriteBatch*) {
    count++;
    return Status::OK();
  });
  ASSERT_TRUE(rs.ok()) << rs.ToString();
  EXPECT_GE(count, 1);

  wal->Close();
}

// TC-OPS-04: Two processes race to Open()
TEST_F(OperationalTest, TC_OPS_04_TwoProcessesRaceToOpen) {
  std::string wal_dir = tmpdir_.path();
  int pipefd[2];
  ASSERT_EQ(pipe(pipefd), 0);

  pid_t p1 = fork();
  if (p1 == 0) {
    close(pipefd[0]);
    char sync;
    read(pipefd[1], &sync, 1);
    close(pipefd[1]);

    WALOptions opts;
    opts.wal_dir = wal_dir;
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, env_, &wal);
    _exit(s.ok() ? 0 : 1);
  }

  pid_t p2 = fork();
  if (p2 == 0) {
    close(pipefd[0]);
    char sync;
    read(pipefd[1], &sync, 1);
    close(pipefd[1]);

    WALOptions opts;
    opts.wal_dir = wal_dir;
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, env_, &wal);
    _exit(s.ok() ? 0 : 1);
  }

  close(pipefd[0]);
  write(pipefd[1], "xx", 2);
  close(pipefd[1]);

  int s1, s2;
  waitpid(p1, &s1, 0);
  waitpid(p2, &s2, 0);

  int ok_count = 0;
  if (WIFEXITED(s1) && WEXITSTATUS(s1) == 0) ok_count++;
  if (WIFEXITED(s2) && WEXITSTATUS(s2) == 0) ok_count++;
  EXPECT_EQ(ok_count, 1) << "Exactly one process should succeed";
}

// TC-OPS-05: Write after Close() returns a clear error
TEST_F(OperationalTest, TC_OPS_05_WriteAfterCloseReturnsError) {
  std::unique_ptr<DBWal> wal;
  ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());

  WriteBatch batch;
  batch.Put("k", "v");
  ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());

  ASSERT_TRUE(wal->Close().ok());

  WriteBatch batch2;
  batch2.Put("after_close", "v");
  Status s = wal->Write(WriteOptions(), &batch2);

  ASSERT_FALSE(s.ok());
  EXPECT_TRUE(s.IsAborted()) << s.ToString();
  EXPECT_NE(s.ToString().find("closed"), std::string::npos)
      << "Error should mention 'closed': " << s.ToString();
}

// TC-OPS-06: Recovery callback error propagates and stops recovery
TEST_F(OperationalTest, TC_OPS_06_RecoveryCallbackErrorPropagates) {
  {
    std::unique_ptr<DBWal> wal;
    ASSERT_TRUE(DBWal::Open(MakeOptions(), env_, &wal).ok());

    for (int i = 0; i < 10; i++) {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), "v" + std::to_string(i));
      ASSERT_TRUE(wal->Write(WriteOptions(), &batch).ok());
    }
    wal->Close();
  }

  int callback_count = 0;
  WALOptions opts = MakeOptions();
  opts.recovery_callback = [&callback_count](SequenceNumber, WriteBatch*) {
    callback_count++;
    if (callback_count == 5) {
      return Status::Corruption("intentional");
    }
    return Status::OK();
  };

  std::unique_ptr<DBWal> wal;
  Status s = DBWal::Open(opts, env_, &wal);

  EXPECT_TRUE(s.IsCorruption()) << s.ToString();
  EXPECT_EQ(callback_count, 5);

  if (s.ok()) wal->Close();
}

}  // namespace mwal
