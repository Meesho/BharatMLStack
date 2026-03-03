#include <benchmark/benchmark.h>

#include <cstdlib>
#include <filesystem>
#include <unistd.h>
#include <memory>
#include <string>

#include "file/writable_file_writer.h"
#include "log/log_writer.h"
#include "mwal/env.h"
#include "mwal/write_batch.h"
#include "vector_db_payload.h"

namespace mwal {

static void BM_WALWrite(benchmark::State& state) {
  const size_t record_size = state.range(0);
  std::string data(record_size, 'x');
  char tmpl[] = "/tmp/mwal_bench_XXXXXX";
  char* dir = mkdtemp(tmpl);
  std::string path = std::string(dir) + "/bench.log";

  for (auto _ : state) {
    state.PauseTiming();
    std::unique_ptr<WritableFile> file;
    Status s = Env::Default()->NewWritableFile(path, &file, EnvOptions());
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    auto writer_file = std::make_unique<WritableFileWriter>(std::move(file), path);
    log::Writer writer(std::move(writer_file), 1, false);
    state.ResumeTiming();

    for (int i = 0; i < 1000; i++) {
      IOStatus ws = writer.AddRecord(Slice(data));
      if (!ws.ok()) {
        state.SkipWithError("AddRecord failed");
        return;
      }
    }

    state.PauseTiming();
    IOStatus cs = writer.Close();
    if (!cs.ok()) {
      state.SkipWithError("Close failed");
      return;
    }
    state.ResumeTiming();
  }
  state.SetBytesProcessed(static_cast<int64_t>(state.iterations()) * 1000 *
                          record_size);
  std::filesystem::remove_all(dir);
}
BENCHMARK(BM_WALWrite)->Arg(64)->Arg(256)->Arg(1024)->Arg(4096)->Arg(32768)->Arg(131072);

// VectorDB: Raw log write with vector-sized records.
static void BM_WALWrite_VectorDB(benchmark::State& state) {
  const int dim = state.range(0);
  const bool use_fp64 = (state.range(1) != 0);
  std::string data = MakeVectorRecordBytes(dim, use_fp64);
  const size_t record_size = data.size();
  char tmpl[] = "/tmp/mwal_bench_XXXXXX";
  char* dir = mkdtemp(tmpl);
  std::string path = std::string(dir) + "/bench_vdb.log";

  for (auto _ : state) {
    state.PauseTiming();
    std::unique_ptr<WritableFile> file;
    Status s = Env::Default()->NewWritableFile(path, &file, EnvOptions());
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    auto writer_file = std::make_unique<WritableFileWriter>(std::move(file), path);
    log::Writer writer(std::move(writer_file), 1, false);
    state.ResumeTiming();

    for (int i = 0; i < 1000; i++) {
      IOStatus ws = writer.AddRecord(Slice(data));
      if (!ws.ok()) {
        state.SkipWithError("AddRecord failed");
        return;
      }
    }

    state.PauseTiming();
    IOStatus cs = writer.Close();
    if (!cs.ok()) {
      state.SkipWithError("Close failed");
      return;
    }
    state.ResumeTiming();
  }
  state.SetBytesProcessed(static_cast<int64_t>(state.iterations()) * 1000 *
                          static_cast<int64_t>(record_size));
  std::filesystem::remove_all(dir);
}
BENCHMARK(BM_WALWrite_VectorDB)->ArgsProduct({{32, 128, 512, 768}, {0, 1}});

static void BM_WALWriteSync(benchmark::State& state) {
  const bool per_record_sync = (state.range(0) == 1);
  std::string data(1024, 'y');
  char tmpl[] = "/tmp/mwal_bench_XXXXXX";
  char* dir = mkdtemp(tmpl);
  std::string path = std::string(dir) + "/sync_bench.log";

  for (auto _ : state) {
    std::unique_ptr<WritableFile> file;
    Status s = Env::Default()->NewWritableFile(path, &file, EnvOptions());
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    auto writer_file = std::make_unique<WritableFileWriter>(std::move(file), path);
    log::Writer writer(std::move(writer_file), 1, false, !per_record_sync);

    for (int i = 0; i < 100; i++) {
      IOStatus ws = writer.AddRecord(Slice(data));
      if (!ws.ok()) {
        state.SkipWithError("AddRecord failed");
        return;
      }
      if (per_record_sync) {
        IOStatus fs = writer.WriteBuffer();
        if (!fs.ok()) {
          state.SkipWithError("WriteBuffer failed");
          return;
        }
      }
    }
    if (!per_record_sync) {
      IOStatus fs = writer.WriteBuffer();
      if (!fs.ok()) {
        state.SkipWithError("WriteBuffer failed");
        return;
      }
    }
    IOStatus cs = writer.Close();
    if (!cs.ok()) {
      state.SkipWithError("Close failed");
      return;
    }
  }
  constexpr int kRecords = 100;
  state.SetItemsProcessed(state.iterations() * kRecords);
  state.SetBytesProcessed(state.iterations() * kRecords * 1024);
  std::filesystem::remove_all(dir);
}
BENCHMARK(BM_WALWriteSync)->Arg(0)->Arg(1);

// VectorDB: Raw log write with per-record sync, 768 fp32 records.
static void BM_WALWriteSync_VectorDB(benchmark::State& state) {
  const bool per_record_sync = (state.range(0) == 1);
  const int dim = 768;
  const bool use_fp64 = false;
  std::string data = MakeVectorRecordBytes(dim, use_fp64);
  char tmpl[] = "/tmp/mwal_bench_XXXXXX";
  char* dir = mkdtemp(tmpl);
  std::string path = std::string(dir) + "/sync_vdb_bench.log";

  for (auto _ : state) {
    std::unique_ptr<WritableFile> file;
    Status s = Env::Default()->NewWritableFile(path, &file, EnvOptions());
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    auto writer_file = std::make_unique<WritableFileWriter>(std::move(file), path);
    log::Writer writer(std::move(writer_file), 1, false, !per_record_sync);

    for (int i = 0; i < 100; i++) {
      IOStatus ws = writer.AddRecord(Slice(data));
      if (!ws.ok()) {
        state.SkipWithError("AddRecord failed");
        return;
      }
      if (per_record_sync) {
        IOStatus fs = writer.WriteBuffer();
        if (!fs.ok()) {
          state.SkipWithError("WriteBuffer failed");
          return;
        }
      }
    }
    if (!per_record_sync) {
      IOStatus fs = writer.WriteBuffer();
      if (!fs.ok()) {
        state.SkipWithError("WriteBuffer failed");
        return;
      }
    }
    IOStatus cs = writer.Close();
    if (!cs.ok()) {
      state.SkipWithError("Close failed");
      return;
    }
  }
  constexpr int kRecords = 100;
  state.SetItemsProcessed(state.iterations() * kRecords);
  state.SetBytesProcessed(state.iterations() * kRecords *
                          static_cast<int64_t>(data.size()));
  std::filesystem::remove_all(dir);
}
BENCHMARK(BM_WALWriteSync_VectorDB)->Arg(0)->Arg(1);

static void BM_WriteBatchEncode(benchmark::State& state) {
  const int num_ops = state.range(0);
  for (auto _ : state) {
    WriteBatch batch;
    for (int i = 0; i < num_ops; i++) {
      std::string k = "key_" + std::to_string(i);
      std::string v = "val_" + std::to_string(i);
      batch.Put(k, v);
    }
    benchmark::DoNotOptimize(batch.GetDataSize());
  }
}
BENCHMARK(BM_WriteBatchEncode)->Arg(1)->Arg(10)->Arg(100)->Arg(1000);

// VectorDB: WriteBatch encode with vector puts.
static void BM_WriteBatchEncode_VectorDB(benchmark::State& state) {
  const int num_ops = state.range(0);
  const int dim = state.range(1);
  const bool use_fp64 = (state.range(2) != 0);

  for (auto _ : state) {
    WriteBatch batch;
    for (int i = 0; i < num_ops; i++) {
      MakeVectorPutBatch(static_cast<uint64_t>(i), dim, use_fp64, &batch);
    }
    benchmark::DoNotOptimize(batch.GetDataSize());
  }
}
BENCHMARK(BM_WriteBatchEncode_VectorDB)
    ->ArgsProduct({{1, 10, 100, 1000}, {32, 128, 512, 768}, {0, 1}});

}  // namespace mwal
