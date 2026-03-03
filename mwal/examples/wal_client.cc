// Simple client that exercises all mwal WAL scenarios.
// Build: from mwal/build, run: ./examples/wal_client
// Uses a temporary directory under /tmp; no gtest dependency.

#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <filesystem>
#include <functional>
#include <string>
#include <thread>
#include <unistd.h>
#include <vector>

#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/wal_file_info.h"
#include "mwal/wal_iterator.h"
#include "mwal/write_batch.h"

namespace {

std::string MakeTempDir() {
  char tmpl[] = "/tmp/mwal_client_XXXXXX";
  char* d = mkdtemp(tmpl);
  if (!d) return "";
  return std::string(d);
}

int passed = 0;
int failed = 0;

void run(const char* name, std::function<bool()> fn) {
  bool ok = false;
  try {
    ok = fn();
  } catch (const std::exception& e) {
    printf("  EXCEPTION: %s\n", e.what());
  }
  if (ok) {
    printf("  [PASS] %s\n", name);
    ++passed;
  } else {
    printf("  [FAIL] %s\n", name);
    ++failed;
  }
}

}  // namespace

int main() {
  std::string tmpdir = MakeTempDir();
  if (tmpdir.empty()) {
    fprintf(stderr, "Failed to create temp dir\n");
    return 1;
  }
  printf("WAL client temp dir: %s\n\n", tmpdir.c_str());

  mwal::Env* env = mwal::Env::Default();

  // ---- 1. Open / Close ----
  run("Open and Close", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario1";
    mwal::Status s = env->CreateDirIfMissing(opts.wal_dir);
    if (!s.ok()) return false;
    std::unique_ptr<mwal::DBWal> wal;
    s = mwal::DBWal::Open(opts, env, &wal);
    if (!s.ok()) return false;
    return wal->Close().ok();
  });

  // ---- 2. Single write + recover ----
  run("Single write and recover", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario2";
    env->CreateDirIfMissing(opts.wal_dir);
    {
      std::unique_ptr<mwal::DBWal> wal;
      if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
      mwal::WriteBatch b;
      b.Put("hello", "world");
      if (!wal->Write(mwal::WriteOptions(), &b).ok()) return false;
      if (wal->GetLatestSequenceNumber() != 1) return false;
      wal->Close();
    }
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    int count = 0;
    mwal::Status s = wal->Recover([&](mwal::SequenceNumber, mwal::WriteBatch*) {
      ++count;
      return mwal::Status::OK();
    });
    if (!s.ok() || count != 1) return false;
    wal->Close();
    return true;
  });

  // ---- 3. Multiple writes + recover ----
  run("Multiple writes and recover", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario3";
    env->CreateDirIfMissing(opts.wal_dir);
    const int N = 50;
    {
      std::unique_ptr<mwal::DBWal> wal;
      if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
      for (int i = 0; i < N; i++) {
        mwal::WriteBatch b;
        b.Put("k" + std::to_string(i), "v" + std::to_string(i));
        if (!wal->Write(mwal::WriteOptions(), &b).ok()) return false;
      }
      if (wal->GetLatestSequenceNumber() != static_cast<mwal::SequenceNumber>(N))
        return false;
      wal->Close();
    }
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    int count = 0;
    wal->Recover([&](mwal::SequenceNumber, mwal::WriteBatch*) {
      ++count;
      return mwal::Status::OK();
    });
    wal->Close();
    return count == N;
  });

  // ---- 4. disableWAL: only WAL writes are recovered ----
  run("disableWAL (recover sees only WAL writes)", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario4";
    env->CreateDirIfMissing(opts.wal_dir);
    {
      std::unique_ptr<mwal::DBWal> wal;
      if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
      mwal::WriteBatch b1;
      b1.Put("in_wal", "yes");
      if (!wal->Write(mwal::WriteOptions(), &b1).ok()) return false;
      mwal::WriteOptions wo;
      wo.disableWAL = true;
      mwal::WriteBatch b2;
      b2.Put("not_in_wal", "no");
      if (!wal->Write(wo, &b2).ok()) return false;
      wal->Close();
    }
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    int count = 0;
    wal->Recover([&](mwal::SequenceNumber, mwal::WriteBatch*) {
      ++count;
      return mwal::Status::OK();
    });
    wal->Close();
    return count == 1;  // only the WAL write is recovered; disableWAL write is not
  });

  // ---- 5. Log rotation ----
  run("Log rotation (small max_wal_file_size)", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario5";
    opts.max_wal_file_size = 1024;
    env->CreateDirIfMissing(opts.wal_dir);
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    std::string big(200, 'X');
    for (int i = 0; i < 20; i++) {
      mwal::WriteBatch b;
      b.Put("k" + std::to_string(i), big);
      if (!wal->Write(mwal::WriteOptions(), &b).ok()) return false;
    }
    if (wal->GetCurrentLogNumber() <= 1) return false;
    wal->Close();
    std::unique_ptr<mwal::DBWal> wal2;
    if (!mwal::DBWal::Open(opts, env, &wal2).ok()) return false;
    int count = 0;
    wal2->Recover([&](mwal::SequenceNumber, mwal::WriteBatch*) {
      ++count;
      return mwal::Status::OK();
    });
    wal2->Close();
    return count == 20;
  });

  // ---- 6. Concurrent writes (group commit) ----
  run("Concurrent writes and recover all", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario6";
    env->CreateDirIfMissing(opts.wal_dir);
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    const int threads = 4;
    const int per = 25;
    std::vector<std::thread> th;
    for (int t = 0; t < threads; t++) {
      th.emplace_back([&wal, t]() {
        for (int i = 0; i < per; i++) {
          mwal::WriteBatch b;
          b.Put("t" + std::to_string(t) + "_k" + std::to_string(i), "v");
          wal->Write(mwal::WriteOptions(), &b);
        }
      });
    }
    for (auto& t : th) t.join();
    if (wal->GetLatestSequenceNumber() != static_cast<mwal::SequenceNumber>(threads * per))
      return false;
    wal->Close();
    std::unique_ptr<mwal::DBWal> wal2;
    if (!mwal::DBWal::Open(opts, env, &wal2).ok()) return false;
    int count = 0;
    wal2->Recover([&](mwal::SequenceNumber, mwal::WriteBatch* batch) {
      count += batch->Count();
      return mwal::Status::OK();
    });
    wal2->Close();
    return count == threads * per;
  });

  // ---- 7. FlushWAL / SyncWAL ----
  run("FlushWAL and SyncWAL", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario7";
    opts.manual_wal_flush = true;
    env->CreateDirIfMissing(opts.wal_dir);
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    mwal::WriteBatch b;
    b.Put("flushed", "data");
    if (!wal->Write(mwal::WriteOptions(), &b).ok()) return false;
    if (!wal->FlushWAL(false).ok()) return false;
    if (!wal->SyncWAL().ok()) return false;
    return wal->Close().ok();
  });

  // ---- 8. SetMinLogNumberToKeep, PurgeObsoleteFiles, GetLiveWalFiles ----
  run("SetMinLogToKeep, Purge, GetLiveWalFiles", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario8";
    opts.max_wal_file_size = 512;
    env->CreateDirIfMissing(opts.wal_dir);
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    std::string v(150, 'A');
    for (int i = 0; i < 15; i++) {
      mwal::WriteBatch b;
      b.Put("k" + std::to_string(i), v);
      if (!wal->Write(mwal::WriteOptions(), &b).ok()) return false;
    }
    uint64_t cur = wal->GetCurrentLogNumber();
    if (cur <= 1) return false;
    wal->SetMinLogNumberToKeep(cur);
    std::vector<mwal::WalFileInfo> before;
    if (!wal->GetLiveWalFiles(&before).ok()) return false;
    if (!wal->PurgeObsoleteFiles().ok()) return false;
    std::vector<mwal::WalFileInfo> after;
    if (!wal->GetLiveWalFiles(&after).ok()) return false;
    for (const auto& f : after)
      if (f.log_number < cur) return false;
    wal->Close();
    return after.size() <= before.size();
  });

  // ---- 9. WalIterator from 0 and from middle ----
  run("WalIterator from start and from middle", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario9";
    env->CreateDirIfMissing(opts.wal_dir);
    {
      std::unique_ptr<mwal::DBWal> wal;
      if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
      for (int i = 0; i < 20; i++) {
        mwal::WriteBatch b;
        b.Put("k" + std::to_string(i), "v");
        wal->Write(mwal::WriteOptions(), &b);
      }
      wal->Close();
    }
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    std::unique_ptr<mwal::WalIterator> it;
    if (!wal->NewWalIterator(0, &it).ok()) return false;
    int from_zero = 0;
    while (it->Valid()) {
      ++from_zero;
      it->Next();
    }
    if (from_zero != 20 || !it->status().ok()) return false;
    std::unique_ptr<mwal::WalIterator> it2;
    if (!wal->NewWalIterator(11, &it2).ok()) return false;
    int from_mid = 0;
    while (it2->Valid()) {
      if (it2->GetSequenceNumber() < 11) return false;
      ++from_mid;
      it2->Next();
    }
    wal->Close();
    return from_mid == 10;
  });

  // ---- 10. Read from particular sequence number ----
  run("Read from particular sequence number", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario10_seq";
    env->CreateDirIfMissing(opts.wal_dir);
    const int total = 25;
    const mwal::SequenceNumber start_seq = 8;  // read from seq 8 onward

    // Input: write data and log what we write
    printf("    Input: writing %d records (seq 1..%d)\n", total, total);
    {
      std::unique_ptr<mwal::DBWal> wal;
      if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
      for (int i = 0; i < total; i++) {
        mwal::WriteBatch b;
        std::string key = "k" + std::to_string(i);
        std::string val = "v";
        b.Put(key, val);
        wal->Write(mwal::WriteOptions(), &b);
        printf("      Put(%s, %s) -> seq %u\n", key.c_str(), val.c_str(),
               static_cast<unsigned>(b.Sequence()));
      }
      wal->Close();
    }

    // Output: read from start_seq and log what we get
    printf("    Output: reading from start_seq=%llu\n",
           static_cast<unsigned long long>(start_seq));
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    std::unique_ptr<mwal::WalIterator> it;
    if (!wal->NewWalIterator(start_seq, &it).ok()) return false;
    int record_count = 0;
    int op_count = 0;
    mwal::SequenceNumber first_seq = 0;
    struct PrintHandler : mwal::WriteBatch::Handler {
      mwal::Status Put(const mwal::Slice& key, const mwal::Slice& value) override {
        printf("        Put(%s, %s)\n", key.ToString().c_str(),
               value.ToString().c_str());
        return mwal::Status::OK();
      }
      mwal::Status Delete(const mwal::Slice& key) override {
        printf("        Delete(%s)\n", key.ToString().c_str());
        return mwal::Status::OK();
      }
    } print_handler;
    while (it->Valid()) {
      mwal::SequenceNumber seq = it->GetSequenceNumber();
      if (seq < start_seq) return false;
      if (record_count == 0) first_seq = seq;
      int n = it->GetBatch().Count();
      op_count += n;
      printf("      log record #%d: seq=%llu ops=%d\n", record_count + 1,
             static_cast<unsigned long long>(seq), n);
      it->GetBatch().Iterate(&print_handler);
      ++record_count;
      it->Next();
    }
    printf("    Output: %d log records, %d ops total (expected %d ops from seq %llu..%d)\n",
           record_count, op_count, total - static_cast<int>(start_seq) + 1,
           static_cast<unsigned long long>(start_seq), total);
    if (!it->status().ok()) return false;
    wal->Close();
    const int expected_ops = total - static_cast<int>(start_seq) + 1;
    return first_seq == start_seq && op_count == expected_ops;
  });

  // ---- 11. Auto-recovery on Open (recovery_callback) ----
  run("Auto-recovery on Open (recovery_callback)", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario11";
    env->CreateDirIfMissing(opts.wal_dir);
    {
      std::unique_ptr<mwal::DBWal> wal;
      if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
      for (int i = 0; i < 5; i++) {
        mwal::WriteBatch b;
        b.Put("k" + std::to_string(i), "v");
        wal->Write(mwal::WriteOptions(), &b);
      }
      wal->Close();
    }
    int recovered = 0;
    opts.recovery_callback = [&](mwal::SequenceNumber, mwal::WriteBatch*) {
      ++recovered;
      return mwal::Status::OK();
    };
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    if (recovered != 5) return false;
    if (wal->GetLatestSequenceNumber() != 5) return false;
    mwal::WriteBatch b;
    b.Put("after_auto", "ok");
    if (!wal->Write(mwal::WriteOptions(), &b).ok()) return false;
    if (wal->GetLatestSequenceNumber() != 6) return false;
    wal->Close();
    return true;
  });

  // ---- 12. Directory lock (second Open fails) ----
  run("Directory lock (second Open fails)", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario12";
    env->CreateDirIfMissing(opts.wal_dir);
    std::unique_ptr<mwal::DBWal> wal1;
    if (!mwal::DBWal::Open(opts, env, &wal1).ok()) return false;
    std::unique_ptr<mwal::DBWal> wal2;
    mwal::Status s = mwal::DBWal::Open(opts, env, &wal2);
    if (s.ok()) return false;  // must fail
    if (!s.IsIOError()) return false;
    wal1->Close();
    s = mwal::DBWal::Open(opts, env, &wal2);
    if (!s.ok()) return false;
    wal2->Close();
    return true;
  });

  // ---- 13. Write after Close returns Aborted ----
  run("Write after Close returns Aborted", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario13";
    env->CreateDirIfMissing(opts.wal_dir);
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    wal->Close();
    mwal::WriteBatch b;
    b.Put("k", "v");
    mwal::Status s = wal->Write(mwal::WriteOptions(), &b);
    return s.IsAborted();
  });

  // ---- 14. Null batch returns InvalidArgument ----
  run("Null batch returns InvalidArgument", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario14";
    env->CreateDirIfMissing(opts.wal_dir);
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    mwal::Status s = wal->Write(mwal::WriteOptions(), nullptr);
    wal->Close();
    return s.IsInvalidArgument();
  });

  // ---- 15. GetLatestSequenceNumber / GetCurrentLogNumber ----
  run("GetLatestSequenceNumber / GetCurrentLogNumber", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario15";
    env->CreateDirIfMissing(opts.wal_dir);
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    if (wal->GetLatestSequenceNumber() != 0) return false;
    if (wal->GetCurrentLogNumber() != 1) return false;
    mwal::WriteBatch b;
    b.Put("a", "1");
    b.Put("b", "2");
    if (!wal->Write(mwal::WriteOptions(), &b).ok()) return false;
    if (wal->GetLatestSequenceNumber() != 2) return false;
    if (b.Sequence() != 1) return false;
    wal->Close();
    return true;
  });

  // ---- 16. Write stall (max_total_wal_size) ----
  run("Write stall when max_total_wal_size exceeded", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario16";
    opts.max_total_wal_size = 1;  // 1 byte -> stall after first write grows file
    env->CreateDirIfMissing(opts.wal_dir);
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    mwal::WriteBatch b1, b2;
    b1.Put("k1", "v1");
    b2.Put("k2", "v2");
    mwal::Status s1 = wal->Write(mwal::WriteOptions(), &b1);
    mwal::Status s2 = wal->Write(mwal::WriteOptions(), &b2);
    wal->Close();
    return s2.IsBusy() || (s1.ok() && s2.IsBusy());
  });

  // ---- 17. max_write_group_size cap ----
  run("max_write_group_size cap (all writes still complete)", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario17";
    opts.max_write_group_size = 2;
    env->CreateDirIfMissing(opts.wal_dir);
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    const int total = 20;
    std::vector<std::thread> th;
    for (int i = 0; i < total; i++) {
      th.emplace_back([&wal, i]() {
        mwal::WriteBatch b;
        b.Put("k" + std::to_string(i), "v");
        wal->Write(mwal::WriteOptions(), &b);
      });
    }
    for (auto& t : th) t.join();
    if (wal->GetLatestSequenceNumber() != static_cast<mwal::SequenceNumber>(total))
      return false;
    wal->Close();
    return true;
  });

#ifdef MWAL_HAVE_ZSTD
  // ---- 18. Compression roundtrip ----
  run("Compression (zstd) roundtrip", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario18";
    opts.wal_compression = mwal::kZSTD;
    env->CreateDirIfMissing(opts.wal_dir);
    {
      std::unique_ptr<mwal::DBWal> wal;
      if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
      for (int i = 0; i < 20; i++) {
        mwal::WriteBatch b;
        b.Put("key_" + std::to_string(i), std::string(100, 'a'));
        wal->Write(mwal::WriteOptions(), &b);
      }
      wal->Close();
    }
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    int count = 0;
    wal->Recover([&](mwal::SequenceNumber, mwal::WriteBatch*) {
      ++count;
      return mwal::Status::OK();
    });
    wal->Close();
    return count == 20;
  });
#endif

  // ---- 19. Recover empty (no WAL files) ----
  run("Recover empty (no WAL files)", [&]() {
    mwal::WALOptions opts;
    opts.wal_dir = tmpdir + "/scenario19";
    env->CreateDirIfMissing(opts.wal_dir);
    std::unique_ptr<mwal::DBWal> wal;
    if (!mwal::DBWal::Open(opts, env, &wal).ok()) return false;
    int count = 0;
    mwal::Status s = wal->Recover([&](mwal::SequenceNumber, mwal::WriteBatch*) {
      ++count;
      return mwal::Status::OK();
    });
    if (!s.ok() || count != 0) return false;
    wal->Close();
    return true;
  });

  // Cleanup temp dir (optional; OS will clean /tmp)
  std::filesystem::remove_all(tmpdir);

  printf("\n--- Result: %d passed, %d failed ---\n", passed, failed);
  return failed == 0 ? 0 : 1;
}
