#include <benchmark/benchmark.h>

#include <barrier>
#include <cstdlib>
#include <filesystem>
#include <mutex>
#include <unistd.h>
#include <memory>
#include <string>
#include <thread>
#include <vector>

#include "file/writable_file_writer.h"
#include "log/log_writer.h"
#include "mwal/env.h"
#include "vector_db_payload.h"

namespace mwal {

static void BM_FsyncLatency(benchmark::State& state) {
  char tmpl[] = "/tmp/mwal_bench_XXXXXX";
  char* dir = mkdtemp(tmpl);
  std::string path = std::string(dir) + "/fsync_bench.log";

  std::unique_ptr<WritableFile> file;
  Status s = Env::Default()->NewWritableFile(path, &file, EnvOptions());
  if (!s.ok()) {
    state.SkipWithError(s.ToString().c_str());
    return;
  }
  std::string data(1024, 'z');
  for (auto _ : state) {
    IOStatus as = file->Append(Slice(data));
    if (!as.ok()) {
      state.SkipWithError("Append failed");
      return;
    }
    IOStatus ss = file->Sync();
    if (!ss.ok()) {
      state.SkipWithError("Sync failed");
      return;
    }
  }
  IOStatus cs = file->Close();
  if (!cs.ok()) {
    state.SkipWithError("Close failed");
    return;
  }
  std::filesystem::remove_all(dir);
}
BENCHMARK(BM_FsyncLatency);

// VectorDB: Fsync latency with 768-dim fp32 vector-sized append.
static void BM_FsyncLatency_VectorDB(benchmark::State& state) {
  const int dim = state.range(0);
  const bool use_fp64 = (state.range(1) != 0);
  std::string data = MakeVectorRecordBytes(dim, use_fp64);

  char tmpl[] = "/tmp/mwal_bench_XXXXXX";
  char* dir = mkdtemp(tmpl);
  std::string path = std::string(dir) + "/fsync_vdb_bench.log";

  std::unique_ptr<WritableFile> file;
  Status s = Env::Default()->NewWritableFile(path, &file, EnvOptions());
  if (!s.ok()) {
    state.SkipWithError(s.ToString().c_str());
    return;
  }
  for (auto _ : state) {
    IOStatus as = file->Append(Slice(data));
    if (!as.ok()) {
      state.SkipWithError("Append failed");
      return;
    }
    IOStatus ss = file->Sync();
    if (!ss.ok()) {
      state.SkipWithError("Sync failed");
      return;
    }
  }
  IOStatus cs = file->Close();
  if (!cs.ok()) {
    state.SkipWithError("Close failed");
    return;
  }
  std::filesystem::remove_all(dir);
}
BENCHMARK(BM_FsyncLatency_VectorDB)->ArgsProduct({{32, 128, 512, 768}, {0, 1}});

// log::Writer::AddRecord() is NOT thread-safe, so a mutex serializes writes.
// This measures the cost of N threads producing records through a single writer.
static void BM_GroupCommit(benchmark::State& state) {
  const int num_threads = state.range(0);
  constexpr int kRecordsPerThread = 100;
  char tmpl[] = "/tmp/mwal_bench_XXXXXX";
  char* dir = mkdtemp(tmpl);
  std::string path = std::string(dir) + "/group_bench.log";
  std::string data(1024, 'g');

  for (auto _ : state) {
    state.PauseTiming();
    std::unique_ptr<WritableFile> file;
    Status s = Env::Default()->NewWritableFile(path, &file, EnvOptions());
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    auto wf = std::make_unique<WritableFileWriter>(std::move(file), path);
    auto writer = std::make_shared<log::Writer>(std::move(wf), 1, false, true);
    std::mutex writer_mu;
    state.ResumeTiming();

    std::barrier sync_barrier(num_threads);
    std::vector<std::thread> threads;
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back([&writer, &writer_mu, &sync_barrier, &data]() {
        sync_barrier.arrive_and_wait();
        for (int i = 0; i < kRecordsPerThread; i++) {
          std::lock_guard<std::mutex> lock(writer_mu);
          IOStatus ws = writer->AddRecord(Slice(data));
          if (!ws.ok()) return;
        }
      });
    }
    for (auto& t : threads) {
      t.join();
    }
    IOStatus fs = writer->WriteBuffer();
    if (!fs.ok()) {
      state.SkipWithError("WriteBuffer failed");
      return;
    }
    IOStatus cs = writer->Close();
    if (!cs.ok()) {
      state.SkipWithError("Close failed");
      return;
    }
  }
  state.SetItemsProcessed(state.iterations() * num_threads * kRecordsPerThread);
  std::filesystem::remove_all(dir);
}
BENCHMARK(BM_GroupCommit)->Arg(1)->Arg(2)->Arg(4)->Arg(8);

// VectorDB: Group commit with vector-sized records.
static void BM_GroupCommit_VectorDB(benchmark::State& state) {
  const int num_threads = state.range(0);
  const int dim = state.range(1);
  const bool use_fp64 = (state.range(2) != 0);
  constexpr int kRecordsPerThread = 100;
  std::string data = MakeVectorRecordBytes(dim, use_fp64);

  char tmpl[] = "/tmp/mwal_bench_XXXXXX";
  char* dir = mkdtemp(tmpl);
  std::string path = std::string(dir) + "/group_vdb_bench.log";

  for (auto _ : state) {
    state.PauseTiming();
    std::unique_ptr<WritableFile> file;
    Status s = Env::Default()->NewWritableFile(path, &file, EnvOptions());
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    auto wf = std::make_unique<WritableFileWriter>(std::move(file), path);
    auto writer = std::make_shared<log::Writer>(std::move(wf), 1, false, true);
    std::mutex writer_mu;
    state.ResumeTiming();

    std::barrier sync_barrier(num_threads);
    std::vector<std::thread> threads;
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back([&writer, &writer_mu, &sync_barrier, &data]() {
        sync_barrier.arrive_and_wait();
        for (int i = 0; i < kRecordsPerThread; i++) {
          std::lock_guard<std::mutex> lock(writer_mu);
          IOStatus ws = writer->AddRecord(Slice(data));
          if (!ws.ok()) return;
        }
      });
    }
    for (auto& t : threads) {
      t.join();
    }
    IOStatus fs = writer->WriteBuffer();
    if (!fs.ok()) {
      state.SkipWithError("WriteBuffer failed");
      return;
    }
    IOStatus cs = writer->Close();
    if (!cs.ok()) {
      state.SkipWithError("Close failed");
      return;
    }
  }
  state.SetItemsProcessed(state.iterations() * num_threads * kRecordsPerThread);
  std::filesystem::remove_all(dir);
}
BENCHMARK(BM_GroupCommit_VectorDB)
    ->ArgsProduct({{1, 2, 4, 8}, {32, 128, 512, 768}, {0, 1}});

}  // namespace mwal
