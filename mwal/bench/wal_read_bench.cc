#include <benchmark/benchmark.h>

#include <cstdlib>
#include <filesystem>
#include <unistd.h>
#include <memory>
#include <string>

#include "file/sequential_file_reader.h"
#include "file/writable_file_writer.h"
#include "log/log_reader.h"
#include "log/log_writer.h"
#include "mwal/env.h"
#include "vector_db_payload.h"

namespace mwal {

static void BM_WALRead(benchmark::State& state) {
  const size_t record_size = state.range(0);
  const int num_records = 1000;
  std::string data(record_size, 'r');
  char tmpl[] = "/tmp/mwal_bench_XXXXXX";
  char* dir = mkdtemp(tmpl);
  std::string path = std::string(dir) + "/read_bench.log";

  {
    std::unique_ptr<WritableFile> file;
    Status s = Env::Default()->NewWritableFile(path, &file, EnvOptions());
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    auto wf = std::make_unique<WritableFileWriter>(std::move(file), path);
    log::Writer writer(std::move(wf), 1, false);
    for (int i = 0; i < num_records; i++) {
      IOStatus ws = writer.AddRecord(Slice(data));
      if (!ws.ok()) {
        state.SkipWithError("AddRecord failed");
        return;
      }
    }
    IOStatus cs = writer.Close();
    if (!cs.ok()) {
      state.SkipWithError("Close failed");
      return;
    }
  }

  for (auto _ : state) {
    std::unique_ptr<SequentialFile> file;
    Status s = Env::Default()->NewSequentialFile(path, &file, EnvOptions());
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    auto sf = std::make_unique<SequentialFileReader>(std::move(file), path);
    log::Reader reader(std::move(sf), nullptr, true, 1);

    Slice record;
    std::string scratch;
    int count = 0;
    while (reader.ReadRecord(&record, &scratch)) {
      count++;
    }
    benchmark::DoNotOptimize(count);
  }
  state.SetBytesProcessed(static_cast<int64_t>(state.iterations()) *
                          num_records * record_size);
  std::filesystem::remove_all(dir);
}
BENCHMARK(BM_WALRead)->Arg(64)->Arg(256)->Arg(1024)->Arg(4096)->Arg(32768);

// VectorDB: Raw log read with vector-sized records.
static void BM_WALRead_VectorDB(benchmark::State& state) {
  const int dim = state.range(0);
  const bool use_fp64 = (state.range(1) != 0);
  std::string data = MakeVectorRecordBytes(dim, use_fp64);
  const size_t record_size = data.size();
  const int num_records = 1000;
  char tmpl[] = "/tmp/mwal_bench_XXXXXX";
  char* dir = mkdtemp(tmpl);
  std::string path = std::string(dir) + "/read_vdb_bench.log";

  {
    std::unique_ptr<WritableFile> file;
    Status s = Env::Default()->NewWritableFile(path, &file, EnvOptions());
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    auto wf = std::make_unique<WritableFileWriter>(std::move(file), path);
    log::Writer writer(std::move(wf), 1, false);
    for (int i = 0; i < num_records; i++) {
      IOStatus ws = writer.AddRecord(Slice(data));
      if (!ws.ok()) {
        state.SkipWithError("AddRecord failed");
        return;
      }
    }
    IOStatus cs = writer.Close();
    if (!cs.ok()) {
      state.SkipWithError("Close failed");
      return;
    }
  }

  for (auto _ : state) {
    std::unique_ptr<SequentialFile> file;
    Status s = Env::Default()->NewSequentialFile(path, &file, EnvOptions());
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    auto sf = std::make_unique<SequentialFileReader>(std::move(file), path);
    log::Reader reader(std::move(sf), nullptr, true, 1);

    Slice record;
    std::string scratch;
    int count = 0;
    while (reader.ReadRecord(&record, &scratch)) {
      count++;
    }
    benchmark::DoNotOptimize(count);
  }
  state.SetBytesProcessed(static_cast<int64_t>(state.iterations()) *
                          num_records * record_size);
  std::filesystem::remove_all(dir);
}
BENCHMARK(BM_WALRead_VectorDB)->ArgsProduct({{32, 128, 512, 768}, {0, 1}});

static void BM_WALRecovery(benchmark::State& state) {
  const int num_records = state.range(0);
  std::string data(256, 'v');
  char tmpl[] = "/tmp/mwal_bench_XXXXXX";
  char* dir = mkdtemp(tmpl);
  std::string path = std::string(dir) + "/recovery_bench.log";

  {
    std::unique_ptr<WritableFile> file;
    Status s = Env::Default()->NewWritableFile(path, &file, EnvOptions());
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    auto wf = std::make_unique<WritableFileWriter>(std::move(file), path);
    log::Writer writer(std::move(wf), 1, false);
    for (int i = 0; i < num_records; i++) {
      IOStatus ws = writer.AddRecord(Slice(data));
      if (!ws.ok()) {
        state.SkipWithError("AddRecord failed");
        return;
      }
    }
    IOStatus cs = writer.Close();
    if (!cs.ok()) {
      state.SkipWithError("Close failed");
      return;
    }
  }

  for (auto _ : state) {
    std::unique_ptr<SequentialFile> file;
    Status s = Env::Default()->NewSequentialFile(path, &file, EnvOptions());
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    auto sf = std::make_unique<SequentialFileReader>(std::move(file), path);
    log::Reader reader(std::move(sf), nullptr, true, 1);

    Slice record;
    std::string scratch;
    int count = 0;
    while (reader.ReadRecord(&record, &scratch)) {
      count++;
    }
    benchmark::DoNotOptimize(count);
  }
  state.SetItemsProcessed(state.iterations() * num_records);
  std::filesystem::remove_all(dir);
}
BENCHMARK(BM_WALRecovery)->Arg(1000)->Arg(10000)->Arg(100000);

// VectorDB: Raw log recovery with vector-sized records.
static void BM_WALRecovery_VectorDB(benchmark::State& state) {
  const int num_records = state.range(0);
  const int dim = state.range(1);
  const bool use_fp64 = (state.range(2) != 0);
  std::string data = MakeVectorRecordBytes(dim, use_fp64);
  char tmpl[] = "/tmp/mwal_bench_XXXXXX";
  char* dir = mkdtemp(tmpl);
  std::string path = std::string(dir) + "/recovery_vdb_bench.log";

  {
    std::unique_ptr<WritableFile> file;
    Status s = Env::Default()->NewWritableFile(path, &file, EnvOptions());
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    auto wf = std::make_unique<WritableFileWriter>(std::move(file), path);
    log::Writer writer(std::move(wf), 1, false);
    for (int i = 0; i < num_records; i++) {
      IOStatus ws = writer.AddRecord(Slice(data));
      if (!ws.ok()) {
        state.SkipWithError("AddRecord failed");
        return;
      }
    }
    IOStatus cs = writer.Close();
    if (!cs.ok()) {
      state.SkipWithError("Close failed");
      return;
    }
  }

  for (auto _ : state) {
    std::unique_ptr<SequentialFile> file;
    Status s = Env::Default()->NewSequentialFile(path, &file, EnvOptions());
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    auto sf = std::make_unique<SequentialFileReader>(std::move(file), path);
    log::Reader reader(std::move(sf), nullptr, true, 1);

    Slice record;
    std::string scratch;
    int count = 0;
    while (reader.ReadRecord(&record, &scratch)) {
      count++;
    }
    benchmark::DoNotOptimize(count);
  }
  state.SetItemsProcessed(state.iterations() * num_records);
  std::filesystem::remove_all(dir);
}
BENCHMARK(BM_WALRecovery_VectorDB)
    ->ArgsProduct({{1000, 10000, 100000}, {32, 128, 512, 768}, {0, 1}});

}  // namespace mwal
