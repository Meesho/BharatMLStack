#include <benchmark/benchmark.h>

#include <algorithm>
#include <atomic>
#include <barrier>
#include <chrono>
#include <cmath>
#include <cstdlib>
#include <filesystem>
#include <memory>
#include <numeric>
#include <string>
#include <thread>
#include <unistd.h>
#include <vector>

#include "mwal/db_wal.h"
#include "mwal/env.h"
#include "mwal/options.h"
#include "mwal/wal_iterator.h"
#include "mwal/write_batch.h"
#include "vector_db_payload.h"

namespace mwal {

namespace {

class BenchTmpDir {
 public:
  BenchTmpDir() {
    char tmpl[] = "/tmp/mwal_bench_XXXXXX";
    char* dir = mkdtemp(tmpl);
    path_ = dir;
  }
  ~BenchTmpDir() { std::filesystem::remove_all(path_); }
  const std::string& path() const { return path_; }
 private:
  std::string path_;
};

}  // namespace

// Single-threaded write throughput with varying batch sizes.
static void BM_DBWalWrite(benchmark::State& state) {
  const int ops_per_batch = state.range(0);

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    state.ResumeTiming();

    for (int i = 0; i < 1000; i++) {
      WriteBatch batch;
      for (int j = 0; j < ops_per_batch; j++) {
        std::string k = "key_" + std::to_string(i * ops_per_batch + j);
        std::string v = "val_" + std::to_string(j);
        batch.Put(k, v);
      }
      Status ws = wal->Write(WriteOptions(), &batch);
      benchmark::DoNotOptimize(ws);
      if (!ws.ok()) {
        state.SkipWithError(ws.ToString().c_str());
        return;
      }
    }

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * 1000 * ops_per_batch);
}
BENCHMARK(BM_DBWalWrite)->Arg(1)->Arg(10)->Arg(100);

// VectorDB: Single-threaded write with catalog_id + embedding.
static void BM_DBWalWrite_VectorDB(benchmark::State& state) {
  const int ops_per_batch = state.range(0);
  const int dim = state.range(1);
  const bool use_fp64 = (state.range(2) != 0);

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    state.ResumeTiming();

    for (int i = 0; i < 1000; i++) {
      WriteBatch batch;
      for (int j = 0; j < ops_per_batch; j++) {
        MakeVectorPutBatch(static_cast<uint64_t>(i * ops_per_batch + j), dim,
                          use_fp64, &batch);
      }
      Status ws = wal->Write(WriteOptions(), &batch);
      benchmark::DoNotOptimize(ws);
      if (!ws.ok()) {
        state.SkipWithError(ws.ToString().c_str());
        return;
      }
    }

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * 1000 * ops_per_batch);
}
BENCHMARK(BM_DBWalWrite_VectorDB)
    ->ArgsProduct({{1, 10, 100}, {32, 128, 512, 768}, {0, 1}});

// Write with sync=true vs sync=false.
static void BM_DBWalWriteSync(benchmark::State& state) {
  const bool do_sync = state.range(0) != 0;

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    WriteOptions wo;
    wo.sync = do_sync;
    state.ResumeTiming();

    constexpr int kWritesPerIteration = 1000;
    for (int i = 0; i < kWritesPerIteration; i++) {
      WriteBatch batch;
      batch.Put("key", "value");
      Status ws = wal->Write(wo, &batch);
      if (!ws.ok()) {
        state.SkipWithError(ws.ToString().c_str());
        return;
      }
    }

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * 1000);
}
BENCHMARK(BM_DBWalWriteSync)->Arg(0)->Arg(1);

// VectorDB: Write with sync on/off.
static void BM_DBWalWriteSync_VectorDB(benchmark::State& state) {
  const bool do_sync = state.range(0) != 0;
  const int dim = state.range(1);
  const bool use_fp64 = (state.range(2) != 0);

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    WriteOptions wo;
    wo.sync = do_sync;
    state.ResumeTiming();

    constexpr int kWritesPerIteration = 1000;
    for (int i = 0; i < kWritesPerIteration; i++) {
      WriteBatch batch;
      MakeVectorPutBatch(static_cast<uint64_t>(i), dim, use_fp64, &batch);
      Status ws = wal->Write(wo, &batch);
      if (!ws.ok()) {
        state.SkipWithError(ws.ToString().c_str());
        return;
      }
    }

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * 1000);
}
BENCHMARK(BM_DBWalWriteSync_VectorDB)->ArgsProduct({{0, 1}, {32, 128, 512, 768}, {0, 1}});

// Concurrent write throughput: measures group commit effectiveness.
static void BM_DBWalConcurrentWrite(benchmark::State& state) {
  const int num_threads = state.range(0);

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    state.ResumeTiming();

    constexpr int kWritesPerThread = 200;
    std::barrier sync_barrier(num_threads);
    std::vector<std::thread> threads;
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back([&wal, &sync_barrier, t] {
        sync_barrier.arrive_and_wait();  // All threads start Write() simultaneously
        for (int i = 0; i < kWritesPerThread; i++) {
          WriteBatch batch;
          batch.Put("t" + std::to_string(t) + "_k" + std::to_string(i), "v");
          Status ws = wal->Write(WriteOptions(), &batch);
          if (!ws.ok()) return;
        }
      });
    }
    for (auto& t : threads) t.join();

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * num_threads * 200);
}
BENCHMARK(BM_DBWalConcurrentWrite)->Arg(1)->Arg(2)->Arg(4)->Arg(8)->Arg(16);

// VectorDB: Concurrent write throughput.
static void BM_DBWalConcurrentWrite_VectorDB(benchmark::State& state) {
  const int num_threads = state.range(0);
  const int dim = state.range(1);
  const bool use_fp64 = (state.range(2) != 0);

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    state.ResumeTiming();

    constexpr int kWritesPerThread = 200;
    std::barrier sync_barrier(num_threads);
    std::vector<std::thread> threads;
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back([&wal, &sync_barrier, t, dim, use_fp64] {
        sync_barrier.arrive_and_wait();
        for (int i = 0; i < kWritesPerThread; i++) {
          WriteBatch batch;
          MakeVectorPutBatch(static_cast<uint64_t>(t) * 1000000 + i, dim,
                            use_fp64, &batch);
          Status ws = wal->Write(WriteOptions(), &batch);
          if (!ws.ok()) return;
        }
      });
    }
    for (auto& t : threads) t.join();

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * num_threads * 200);
}
BENCHMARK(BM_DBWalConcurrentWrite_VectorDB)
    ->ArgsProduct({{1, 2, 4, 8, 16}, {32, 128, 512, 768}, {0, 1}});

// Per-write latency (p50, p95, p99) under concurrent throughput.
// Measures user-visible latency: lock acquisition + queue wait + write + sync.
// Batches are pre-built outside the timing window so only Write() is measured.
// 10k samples/thread for statistically meaningful p99.
// Pinned to Iterations(1) since each iteration is a full experiment.
static void BM_DBWalWriteLatencyUnderThroughput(benchmark::State& state) {
  const int num_threads = static_cast<int>(state.range(0));
  constexpr int kWritesPerThread = 10000;

  auto percentile = [](const std::vector<int64_t>& sorted, double p) -> double {
    if (sorted.empty()) return 0.0;
    double rank = (p / 100.0) * static_cast<double>(sorted.size() - 1);
    size_t lo = static_cast<size_t>(rank);
    size_t hi = std::min(lo + 1, sorted.size() - 1);
    double frac = rank - static_cast<double>(lo);
    return (1.0 - frac) * static_cast<double>(sorted[lo]) +
           frac * static_cast<double>(sorted[hi]);
  };

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    std::vector<std::vector<WriteBatch>> all_batches(
        static_cast<size_t>(num_threads));
    for (int t = 0; t < num_threads; t++) {
      all_batches[static_cast<size_t>(t)].reserve(
          static_cast<size_t>(kWritesPerThread));
      for (int i = 0; i < kWritesPerThread; i++) {
        WriteBatch b;
        b.Put("t" + std::to_string(t) + "_k" + std::to_string(i), "v");
        all_batches[static_cast<size_t>(t)].push_back(std::move(b));
      }
    }
    state.ResumeTiming();

    std::barrier sync_barrier(num_threads);
    std::vector<std::vector<int64_t>> latencies_per_thread(
        static_cast<size_t>(num_threads));
    std::vector<std::thread> threads;
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back(
          [&wal, &latencies_per_thread, &all_batches, &sync_barrier, t]() {
            auto& lat = latencies_per_thread[static_cast<size_t>(t)];
            auto& batches = all_batches[static_cast<size_t>(t)];
            lat.reserve(static_cast<size_t>(kWritesPerThread));
            sync_barrier.arrive_and_wait();
            for (int i = 0; i < kWritesPerThread; i++) {
              auto t0 = std::chrono::high_resolution_clock::now();
              Status ws = wal->Write(WriteOptions(), &batches[static_cast<size_t>(i)]);
              auto t1 = std::chrono::high_resolution_clock::now();
              if (!ws.ok()) return;
              lat.push_back(static_cast<int64_t>(
                  std::chrono::duration_cast<std::chrono::nanoseconds>(t1 - t0)
                      .count()));
            }
          });
    }
    for (auto& t : threads) t.join();

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }

    std::vector<int64_t> all;
    all.reserve(static_cast<size_t>(num_threads * kWritesPerThread));
    for (const auto& lat : latencies_per_thread) {
      all.insert(all.end(), lat.begin(), lat.end());
    }
    if (!all.empty()) {
      std::sort(all.begin(), all.end());
      double sum = std::accumulate(all.begin(), all.end(), 0.0);
      double mean = sum / static_cast<double>(all.size());

      state.counters["min_ns"] =
          benchmark::Counter(static_cast<double>(all.front()));
      state.counters["mean_ns"] = benchmark::Counter(mean);
      state.counters["p50_ns"] = benchmark::Counter(percentile(all, 50.0));
      state.counters["p95_ns"] = benchmark::Counter(percentile(all, 95.0));
      state.counters["p99_ns"] = benchmark::Counter(percentile(all, 99.0));
      state.counters["max_ns"] =
          benchmark::Counter(static_cast<double>(all.back()));
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * num_threads * kWritesPerThread);
}
BENCHMARK(BM_DBWalWriteLatencyUnderThroughput)
    ->Iterations(1)
    ->Arg(1)
    ->Arg(2)
    ->Arg(4)
    ->Arg(8)
    ->Arg(16);

// VectorDB: Per-write latency under throughput.
static void BM_DBWalWriteLatencyUnderThroughput_VectorDB(benchmark::State& state) {
  const int num_threads = static_cast<int>(state.range(0));
  const int dim = static_cast<int>(state.range(1));
  const bool use_fp64 = (state.range(2) != 0);
  constexpr int kWritesPerThread = 10000;

  auto percentile = [](const std::vector<int64_t>& sorted, double p) -> double {
    if (sorted.empty()) return 0.0;
    double rank = (p / 100.0) * static_cast<double>(sorted.size() - 1);
    size_t lo = static_cast<size_t>(rank);
    size_t hi = std::min(lo + 1, sorted.size() - 1);
    double frac = rank - static_cast<double>(lo);
    return (1.0 - frac) * static_cast<double>(sorted[lo]) +
           frac * static_cast<double>(sorted[hi]);
  };

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    std::vector<std::vector<WriteBatch>> all_batches(
        static_cast<size_t>(num_threads));
    for (int t = 0; t < num_threads; t++) {
      all_batches[static_cast<size_t>(t)].reserve(
          static_cast<size_t>(kWritesPerThread));
      for (int i = 0; i < kWritesPerThread; i++) {
        WriteBatch b;
        MakeVectorPutBatch(static_cast<uint64_t>(t) * 1000000 + i, dim, use_fp64,
                          &b);
        all_batches[static_cast<size_t>(t)].push_back(std::move(b));
      }
    }
    state.ResumeTiming();

    std::barrier sync_barrier(num_threads);
    std::vector<std::vector<int64_t>> latencies_per_thread(
        static_cast<size_t>(num_threads));
    std::vector<std::thread> threads;
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back(
          [&wal, &latencies_per_thread, &all_batches, &sync_barrier, t]() {
            auto& lat = latencies_per_thread[static_cast<size_t>(t)];
            auto& batches = all_batches[static_cast<size_t>(t)];
            lat.reserve(static_cast<size_t>(kWritesPerThread));
            sync_barrier.arrive_and_wait();
            for (int i = 0; i < kWritesPerThread; i++) {
              auto t0 = std::chrono::high_resolution_clock::now();
              Status ws = wal->Write(WriteOptions(), &batches[static_cast<size_t>(i)]);
              auto t1 = std::chrono::high_resolution_clock::now();
              if (!ws.ok()) return;
              lat.push_back(static_cast<int64_t>(
                  std::chrono::duration_cast<std::chrono::nanoseconds>(t1 - t0)
                      .count()));
            }
          });
    }
    for (auto& t : threads) t.join();

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }

    std::vector<int64_t> all;
    all.reserve(static_cast<size_t>(num_threads * kWritesPerThread));
    for (const auto& lat : latencies_per_thread) {
      all.insert(all.end(), lat.begin(), lat.end());
    }
    if (!all.empty()) {
      std::sort(all.begin(), all.end());
      double sum = std::accumulate(all.begin(), all.end(), 0.0);
      double mean = sum / static_cast<double>(all.size());

      state.counters["min_ns"] =
          benchmark::Counter(static_cast<double>(all.front()));
      state.counters["mean_ns"] = benchmark::Counter(mean);
      state.counters["p50_ns"] = benchmark::Counter(percentile(all, 50.0));
      state.counters["p95_ns"] = benchmark::Counter(percentile(all, 95.0));
      state.counters["p99_ns"] = benchmark::Counter(percentile(all, 99.0));
      state.counters["max_ns"] =
          benchmark::Counter(static_cast<double>(all.back()));
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * num_threads * kWritesPerThread);
}
BENCHMARK(BM_DBWalWriteLatencyUnderThroughput_VectorDB)
    ->Iterations(1)
    ->ArgsProduct({{1, 2, 4, 8, 16}, {32, 128, 512, 768}, {0, 1}});

// Recovery time.
static void BM_DBWalRecovery(benchmark::State& state) {
  const int num_records = state.range(0);

  BenchTmpDir dir;
  WALOptions opts;
  opts.wal_dir = dir.path();

  // Populate WAL once.
  {
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    for (int i = 0; i < num_records; i++) {
      WriteBatch batch;
      batch.Put("key_" + std::to_string(i), "value_" + std::to_string(i));
      Status ws = wal->Write(WriteOptions(), &batch);
      if (!ws.ok()) {
        state.SkipWithError(ws.ToString().c_str());
        return;
      }
    }
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
  }

  for (auto _ : state) {
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    int count = 0;
    s = wal->Recover([&](SequenceNumber, WriteBatch*) {
      count++;
      return Status::OK();
    });
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    benchmark::DoNotOptimize(count);
    // Don't close — we don't want to measure cleanup or the new log file.
  }
  state.SetItemsProcessed(state.iterations() * num_records);
}
BENCHMARK(BM_DBWalRecovery)->Arg(1000)->Arg(10000)->Arg(100000);

// VectorDB: Recovery time.
static void BM_DBWalRecovery_VectorDB(benchmark::State& state) {
  const int num_records = state.range(0);
  const int dim = state.range(1);
  const bool use_fp64 = (state.range(2) != 0);

  BenchTmpDir dir;
  WALOptions opts;
  opts.wal_dir = dir.path();

  {
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    for (int i = 0; i < num_records; i++) {
      WriteBatch batch;
      MakeVectorPutBatch(static_cast<uint64_t>(i), dim, use_fp64, &batch);
      Status ws = wal->Write(WriteOptions(), &batch);
      if (!ws.ok()) {
        state.SkipWithError(ws.ToString().c_str());
        return;
      }
    }
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
  }

  for (auto _ : state) {
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    int count = 0;
    s = wal->Recover([&](SequenceNumber, WriteBatch*) {
      count++;
      return Status::OK();
    });
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    benchmark::DoNotOptimize(count);
  }
  state.SetItemsProcessed(state.iterations() * num_records);
}
BENCHMARK(BM_DBWalRecovery_VectorDB)
    ->ArgsProduct({{1000, 10000, 100000}, {32, 128, 512, 768}, {0, 1}});

// Write throughput with log rotation (small max_wal_file_size).
static void BM_DBWalRotation(benchmark::State& state) {
  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.max_wal_file_size = 4096;  // Force frequent rotation.
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    state.ResumeTiming();

    std::string big_val(256, 'X');
    for (int i = 0; i < 500; i++) {
      WriteBatch batch;
      batch.Put("key_" + std::to_string(i), big_val);
      Status ws = wal->Write(WriteOptions(), &batch);
      if (!ws.ok()) {
        state.SkipWithError(ws.ToString().c_str());
        return;
      }
    }

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * 500);
}
BENCHMARK(BM_DBWalRotation);

// VectorDB: Write throughput with log rotation.
static void BM_DBWalRotation_VectorDB(benchmark::State& state) {
  const int dim = state.range(0);
  const bool use_fp64 = (state.range(1) != 0);

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.max_wal_file_size = 4096;
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    state.ResumeTiming();

    for (int i = 0; i < 500; i++) {
      WriteBatch batch;
      MakeVectorPutBatch(static_cast<uint64_t>(i), dim, use_fp64, &batch);
      Status ws = wal->Write(WriteOptions(), &batch);
      if (!ws.ok()) {
        state.SkipWithError(ws.ToString().c_str());
        return;
      }
    }

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * 500);
}
BENCHMARK(BM_DBWalRotation_VectorDB)->ArgsProduct({{32, 128, 512, 768}, {0, 1}});

// Group commit scaling: sync write throughput vs thread count (group commit coalescing).
// Reports wall-time throughput since CPU time is misleading for sync I/O.
static void BM_GroupCommitScaling(benchmark::State& state) {
  const int num_threads = state.range(0);
  constexpr int kTotal = 2000;
  double total_wall_ns = 0;

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    int per_thread = kTotal / num_threads;
    std::barrier sync_barrier(num_threads);
    std::vector<std::thread> threads;
    WriteOptions wo;
    wo.sync = true;
    state.ResumeTiming();

    auto wall_start = std::chrono::high_resolution_clock::now();
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back([&wal, &wo, &sync_barrier, t, per_thread] {
        sync_barrier.arrive_and_wait();
        for (int i = 0; i < per_thread; i++) {
          WriteBatch batch;
          batch.Put("t" + std::to_string(t) + "_" + std::to_string(i), "v");
          Status ws = wal->Write(wo, &batch);
          if (!ws.ok()) return;
        }
      });
    }
    for (auto& t : threads) t.join();
    auto wall_end = std::chrono::high_resolution_clock::now();
    total_wall_ns += static_cast<double>(
        std::chrono::duration_cast<std::chrono::nanoseconds>(wall_end - wall_start)
            .count());

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * kTotal);
  if (total_wall_ns > 0) {
    double total_ops = static_cast<double>(state.iterations() * kTotal);
    state.counters["wall_ops_per_sec"] =
        benchmark::Counter(total_ops / (total_wall_ns / 1e9));
  }
}
BENCHMARK(BM_GroupCommitScaling)->Arg(1)->Arg(2)->Arg(4)->Arg(8);

// VectorDB: Group commit scaling with sync writes.
static void BM_GroupCommitScaling_VectorDB(benchmark::State& state) {
  const int num_threads = state.range(0);
  const int dim = state.range(1);
  const bool use_fp64 = (state.range(2) != 0);
  constexpr int kTotal = 2000;
  double total_wall_ns = 0;

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    int per_thread = kTotal / num_threads;
    std::barrier sync_barrier(num_threads);
    std::vector<std::thread> threads;
    WriteOptions wo;
    wo.sync = true;
    state.ResumeTiming();

    auto wall_start = std::chrono::high_resolution_clock::now();
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back([&wal, &wo, &sync_barrier, t, per_thread, dim, use_fp64] {
        sync_barrier.arrive_and_wait();
        for (int i = 0; i < per_thread; i++) {
          WriteBatch batch;
          MakeVectorPutBatch(static_cast<uint64_t>(t) * 10000 + i, dim, use_fp64,
                            &batch);
          Status ws = wal->Write(wo, &batch);
          if (!ws.ok()) return;
        }
      });
    }
    for (auto& t : threads) t.join();
    auto wall_end = std::chrono::high_resolution_clock::now();
    total_wall_ns += static_cast<double>(
        std::chrono::duration_cast<std::chrono::nanoseconds>(wall_end - wall_start)
            .count());

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * kTotal);
  if (total_wall_ns > 0) {
    double total_ops = static_cast<double>(state.iterations() * kTotal);
    state.counters["wall_ops_per_sec"] =
        benchmark::Counter(total_ops / (total_wall_ns / 1e9));
  }
}
BENCHMARK(BM_GroupCommitScaling_VectorDB)
    ->ArgsProduct({{1, 2, 4, 8}, {32, 128, 512, 768}, {0, 1}});

// Merged group commit throughput: async writes, many threads, batches merged.
// Reports wall-time throughput alongside CPU-time items_per_second.
static void BM_MergedGroupCommit(benchmark::State& state) {
  const int num_threads = state.range(0);
  constexpr int kWritesPerThread = 500;
  double total_wall_ns = 0;

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    std::barrier sync_barrier(num_threads);
    std::vector<std::thread> threads;
    state.ResumeTiming();

    auto wall_start = std::chrono::high_resolution_clock::now();
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back([&wal, &sync_barrier, t] {
        sync_barrier.arrive_and_wait();
        for (int i = 0; i < kWritesPerThread; i++) {
          WriteBatch batch;
          batch.Put("t" + std::to_string(t) + "_k" + std::to_string(i), "v");
          Status ws = wal->Write(WriteOptions(), &batch);
          if (!ws.ok()) return;
        }
      });
    }
    for (auto& t : threads) t.join();
    auto wall_end = std::chrono::high_resolution_clock::now();
    total_wall_ns += static_cast<double>(
        std::chrono::duration_cast<std::chrono::nanoseconds>(wall_end - wall_start)
            .count());

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  int64_t total_ops = state.iterations() * num_threads * kWritesPerThread;
  state.SetItemsProcessed(total_ops);
  if (total_wall_ns > 0) {
    state.counters["wall_ops_per_sec"] =
        benchmark::Counter(static_cast<double>(total_ops) / (total_wall_ns / 1e9));
  }
}
BENCHMARK(BM_MergedGroupCommit)->Arg(1)->Arg(2)->Arg(4)->Arg(8)->Arg(16);

// VectorDB: Merged group commit throughput.
static void BM_MergedGroupCommit_VectorDB(benchmark::State& state) {
  const int num_threads = state.range(0);
  const int dim = state.range(1);
  const bool use_fp64 = (state.range(2) != 0);
  constexpr int kWritesPerThread = 500;
  double total_wall_ns = 0;

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    std::barrier sync_barrier(num_threads);
    std::vector<std::thread> threads;
    state.ResumeTiming();

    auto wall_start = std::chrono::high_resolution_clock::now();
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back([&wal, &sync_barrier, t, dim, use_fp64] {
        sync_barrier.arrive_and_wait();
        for (int i = 0; i < kWritesPerThread; i++) {
          WriteBatch batch;
          MakeVectorPutBatch(static_cast<uint64_t>(t) * 100000 + i, dim, use_fp64,
                            &batch);
          Status ws = wal->Write(WriteOptions(), &batch);
          if (!ws.ok()) return;
        }
      });
    }
    for (auto& t : threads) t.join();
    auto wall_end = std::chrono::high_resolution_clock::now();
    total_wall_ns += static_cast<double>(
        std::chrono::duration_cast<std::chrono::nanoseconds>(wall_end - wall_start)
            .count());

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  int64_t total_ops = state.iterations() * num_threads * kWritesPerThread;
  state.SetItemsProcessed(total_ops);
  if (total_wall_ns > 0) {
    state.counters["wall_ops_per_sec"] =
        benchmark::Counter(static_cast<double>(total_ops) / (total_wall_ns / 1e9));
  }
}
BENCHMARK(BM_MergedGroupCommit_VectorDB)
    ->ArgsProduct({{1, 2, 4, 8, 16}, {32, 128, 512, 768}, {0, 1}});

// Coalesced async write throughput: same workload as BM_MergedGroupCommit but
// with the dedicated writer thread (max_async_queue_depth enabled).
static void BM_CoalescedAsyncWrite(benchmark::State& state) {
  const int num_threads = state.range(0);
  constexpr int kWritesPerThread = 500;
  double total_wall_ns = 0;

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.max_async_queue_depth = 10000;
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    std::barrier sync_barrier(num_threads);
    std::vector<std::thread> threads;
    state.ResumeTiming();

    auto wall_start = std::chrono::high_resolution_clock::now();
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back([&wal, &sync_barrier, t] {
        sync_barrier.arrive_and_wait();
        for (int i = 0; i < kWritesPerThread; i++) {
          WriteBatch batch;
          batch.Put("t" + std::to_string(t) + "_k" + std::to_string(i), "v");
          Status ws = wal->Write(WriteOptions(), &batch);
          if (!ws.ok()) return;
        }
      });
    }
    for (auto& t : threads) t.join();
    auto wall_end = std::chrono::high_resolution_clock::now();
    total_wall_ns += static_cast<double>(
        std::chrono::duration_cast<std::chrono::nanoseconds>(wall_end - wall_start)
            .count());

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  int64_t total_ops = state.iterations() * num_threads * kWritesPerThread;
  state.SetItemsProcessed(total_ops);
  if (total_wall_ns > 0) {
    state.counters["wall_ops_per_sec"] =
        benchmark::Counter(static_cast<double>(total_ops) / (total_wall_ns / 1e9));
  }
}
BENCHMARK(BM_CoalescedAsyncWrite)->Arg(1)->Arg(2)->Arg(4)->Arg(8)->Arg(16);

// VectorDB: Coalesced async write throughput.
static void BM_CoalescedAsyncWrite_VectorDB(benchmark::State& state) {
  const int num_threads = state.range(0);
  const int dim = state.range(1);
  const bool use_fp64 = (state.range(2) != 0);
  constexpr int kWritesPerThread = 500;
  double total_wall_ns = 0;

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.max_async_queue_depth = 10000;
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    std::barrier sync_barrier(num_threads);
    std::vector<std::thread> threads;
    state.ResumeTiming();

    auto wall_start = std::chrono::high_resolution_clock::now();
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back([&wal, &sync_barrier, t, dim, use_fp64] {
        sync_barrier.arrive_and_wait();
        for (int i = 0; i < kWritesPerThread; i++) {
          WriteBatch batch;
          MakeVectorPutBatch(static_cast<uint64_t>(t) * 100000 + i, dim, use_fp64,
                            &batch);
          Status ws = wal->Write(WriteOptions(), &batch);
          if (!ws.ok()) return;
        }
      });
    }
    for (auto& t : threads) t.join();
    auto wall_end = std::chrono::high_resolution_clock::now();
    total_wall_ns += static_cast<double>(
        std::chrono::duration_cast<std::chrono::nanoseconds>(wall_end - wall_start)
            .count());

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  int64_t total_ops = state.iterations() * num_threads * kWritesPerThread;
  state.SetItemsProcessed(total_ops);
  if (total_wall_ns > 0) {
    state.counters["wall_ops_per_sec"] =
        benchmark::Counter(static_cast<double>(total_ops) / (total_wall_ns / 1e9));
  }
}
BENCHMARK(BM_CoalescedAsyncWrite_VectorDB)
    ->ArgsProduct({{1, 2, 4, 8, 16}, {32, 128, 512, 768}, {0, 1}});

// Time-based flush: throughput vs flush interval (async-only).
static void BM_TimeBasedFlushThroughput_VectorDB(benchmark::State& state) {
  const int interval_ms = static_cast<int>(state.range(0));
  const int num_threads = static_cast<int>(state.range(1));
  const int dim = static_cast<int>(state.range(2));
  const bool use_fp64 = (state.range(3) != 0);
  constexpr int kWritesPerThread = 500;
  double total_wall_ns = 0;

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.max_async_queue_depth = 10000;
    opts.max_async_flush_interval_ms = static_cast<uint64_t>(interval_ms);
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    std::barrier sync_barrier(num_threads);
    state.ResumeTiming();

    auto wall_start = std::chrono::high_resolution_clock::now();
    std::vector<std::thread> threads;
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back([&wal, &sync_barrier, t, dim, use_fp64] {
        sync_barrier.arrive_and_wait();
        for (int i = 0; i < kWritesPerThread; i++) {
          WriteBatch batch;
          MakeVectorPutBatch(static_cast<uint64_t>(t) * 100000 + i, dim,
                            use_fp64, &batch);
          Status ws = wal->Write(WriteOptions(), &batch);
          if (!ws.ok()) return;
        }
      });
    }
    for (auto& t : threads) t.join();
    auto wall_end = std::chrono::high_resolution_clock::now();
    total_wall_ns += static_cast<double>(
        std::chrono::duration_cast<std::chrono::nanoseconds>(wall_end - wall_start)
            .count());

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  int64_t total_ops = state.iterations() * num_threads * kWritesPerThread;
  state.SetItemsProcessed(total_ops);
  if (total_wall_ns > 0) {
    state.counters["wall_ops_per_sec"] =
        benchmark::Counter(static_cast<double>(total_ops) / (total_wall_ns / 1e9));
  }
}
BENCHMARK(BM_TimeBasedFlushThroughput_VectorDB)
    ->ArgsProduct({{0, 10, 50, 100, 200}, {4, 8}, {32, 128, 512, 768}, {0, 1}});

// Time-based flush: single-thread sequential write latency vs interval.
// Measures per-write latency when one thread does back-to-back async writes
// (not strict "idle" — one write then silence). Interval affects when the
// writer wakes (timeout vs notify), so p50/p99 capture that variance.
static void BM_TimeBasedFlushIdleLatency_VectorDB(benchmark::State& state) {
  const int interval_ms = static_cast<int>(state.range(0));
  const int dim = static_cast<int>(state.range(1));
  const bool use_fp64 = (state.range(2) != 0);
  constexpr int kWritesPerIteration = 100;

  auto percentile = [](const std::vector<int64_t>& sorted, double p) -> double {
    if (sorted.empty()) return 0.0;
    double rank = (p / 100.0) * static_cast<double>(sorted.size() - 1);
    size_t lo = static_cast<size_t>(rank);
    size_t hi = std::min(lo + 1, sorted.size() - 1);
    double frac = rank - static_cast<double>(lo);
    return (1.0 - frac) * static_cast<double>(sorted[lo]) +
           frac * static_cast<double>(sorted[hi]);
  };

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.max_async_queue_depth = 10000;
    opts.max_async_flush_interval_ms = static_cast<uint64_t>(interval_ms);
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    std::vector<WriteBatch> batches(static_cast<size_t>(kWritesPerIteration));
    for (int i = 0; i < kWritesPerIteration; i++) {
      MakeVectorPutBatch(static_cast<uint64_t>(i), dim, use_fp64,
                        &batches[static_cast<size_t>(i)]);
    }
    state.ResumeTiming();

    std::vector<int64_t> latencies;
    latencies.reserve(static_cast<size_t>(kWritesPerIteration));
    for (int i = 0; i < kWritesPerIteration; i++) {
      auto t0 = std::chrono::high_resolution_clock::now();
      Status ws = wal->Write(WriteOptions(), &batches[static_cast<size_t>(i)]);
      auto t1 = std::chrono::high_resolution_clock::now();
      if (!ws.ok()) return;
      latencies.push_back(static_cast<int64_t>(
          std::chrono::duration_cast<std::chrono::nanoseconds>(t1 - t0).count()));
    }

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    if (!latencies.empty()) {
      std::sort(latencies.begin(), latencies.end());
      state.counters["p50_ns"] = benchmark::Counter(percentile(latencies, 50.0));
      state.counters["p99_ns"] = benchmark::Counter(percentile(latencies, 99.0));
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * kWritesPerIteration);
}
BENCHMARK(BM_TimeBasedFlushIdleLatency_VectorDB)
    ->ArgsProduct({{0, 25, 50, 100}, {32, 128, 512, 768}, {0, 1}});

// Time-based flush: burst of K writes then stop; total completion time.
static void BM_TimeBasedFlushBurstLatency_VectorDB(benchmark::State& state) {
  const int interval_ms = static_cast<int>(state.range(0));
  const int burst_size = static_cast<int>(state.range(1));
  const int dim = static_cast<int>(state.range(2));
  const bool use_fp64 = (state.range(3) != 0);

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.max_async_queue_depth = 10000;
    opts.max_async_flush_interval_ms = static_cast<uint64_t>(interval_ms);
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    std::vector<WriteBatch> batches(static_cast<size_t>(burst_size));
    for (int i = 0; i < burst_size; i++) {
      MakeVectorPutBatch(static_cast<uint64_t>(i), dim, use_fp64,
                        &batches[static_cast<size_t>(i)]);
    }
    state.ResumeTiming();

    auto wall_start = std::chrono::high_resolution_clock::now();
    for (int i = 0; i < burst_size; i++) {
      Status ws = wal->Write(WriteOptions(), &batches[static_cast<size_t>(i)]);
      if (!ws.ok()) return;
    }
    auto wall_end = std::chrono::high_resolution_clock::now();
    int64_t burst_ns = static_cast<int64_t>(
        std::chrono::duration_cast<std::chrono::nanoseconds>(wall_end - wall_start)
            .count());

    state.PauseTiming();
    state.counters["burst_total_ns"] = benchmark::Counter(static_cast<double>(burst_ns));
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * burst_size);
}
BENCHMARK(BM_TimeBasedFlushBurstLatency_VectorDB)
    ->ArgsProduct({{0, 50, 100}, {5, 10}, {32, 128, 512, 768}, {0, 1}});

// Time-based flush: mixed sync/async with and without timer.
static void BM_TimeBasedFlushMixedSyncAsync_VectorDB(benchmark::State& state) {
  const int num_threads = static_cast<int>(state.range(0));
  const int sync_pct = static_cast<int>(state.range(1));
  const int interval_ms = static_cast<int>(state.range(2));
  const int dim = static_cast<int>(state.range(3));
  const bool use_fp64 = (state.range(4) != 0);
  constexpr int kTotal = 2000;
  const int per_thread = kTotal / num_threads;
  const int num_sync =
      std::max(1, (num_threads * sync_pct + 99) / 100);

  auto percentile = [](const std::vector<int64_t>& sorted, double p) -> double {
    if (sorted.empty()) return 0.0;
    double rank = (p / 100.0) * static_cast<double>(sorted.size() - 1);
    size_t lo = static_cast<size_t>(rank);
    size_t hi = std::min(lo + 1, sorted.size() - 1);
    double frac = rank - static_cast<double>(lo);
    return (1.0 - frac) * static_cast<double>(sorted[lo]) +
           frac * static_cast<double>(sorted[hi]);
  };

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.max_async_queue_depth = 10000;
    opts.max_async_flush_interval_ms = static_cast<uint64_t>(interval_ms);
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    std::vector<std::vector<WriteBatch>> all_batches(
        static_cast<size_t>(num_threads));
    std::vector<bool> use_sync(static_cast<size_t>(num_threads));
    for (int t = 0; t < num_threads; t++) {
      use_sync[static_cast<size_t>(t)] = (t < num_sync);
      all_batches[static_cast<size_t>(t)].reserve(
          static_cast<size_t>(per_thread));
      for (int i = 0; i < per_thread; i++) {
        WriteBatch b;
        MakeVectorPutBatch(static_cast<uint64_t>(t) * 10000 + i, dim, use_fp64,
                          &b);
        all_batches[static_cast<size_t>(t)].push_back(std::move(b));
      }
    }
    state.ResumeTiming();

    std::barrier sync_barrier(num_threads);
    std::vector<std::vector<int64_t>> latencies_per_thread(
        static_cast<size_t>(num_threads));
    std::vector<std::thread> threads;

    auto wall_start = std::chrono::high_resolution_clock::now();
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back(
          [&wal, &latencies_per_thread, &all_batches, &use_sync, &sync_barrier,
           t, per_thread]() {
            auto& lat = latencies_per_thread[static_cast<size_t>(t)];
            auto& batches = all_batches[static_cast<size_t>(t)];
            lat.reserve(static_cast<size_t>(per_thread));
            WriteOptions wo;
            wo.sync = use_sync[static_cast<size_t>(t)];
            sync_barrier.arrive_and_wait();
            for (int i = 0; i < per_thread; i++) {
              auto t0 = std::chrono::high_resolution_clock::now();
              Status ws = wal->Write(wo, &batches[static_cast<size_t>(i)]);
              auto t1 = std::chrono::high_resolution_clock::now();
              if (!ws.ok()) return;
              lat.push_back(static_cast<int64_t>(
                  std::chrono::duration_cast<std::chrono::nanoseconds>(t1 - t0)
                      .count()));
            }
          });
    }
    for (auto& t : threads) t.join();
    auto wall_end = std::chrono::high_resolution_clock::now();

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }

    std::vector<int64_t> async_lat, sync_lat;
    for (int t = 0; t < num_threads; t++) {
      const auto& lat = latencies_per_thread[static_cast<size_t>(t)];
      if (use_sync[static_cast<size_t>(t)]) {
        sync_lat.insert(sync_lat.end(), lat.begin(), lat.end());
      } else {
        async_lat.insert(async_lat.end(), lat.begin(), lat.end());
      }
    }
    if (!async_lat.empty()) {
      std::sort(async_lat.begin(), async_lat.end());
      state.counters["async_p99_ns"] =
          benchmark::Counter(percentile(async_lat, 99.0));
    }
    if (!sync_lat.empty()) {
      std::sort(sync_lat.begin(), sync_lat.end());
      state.counters["sync_p99_ns"] =
          benchmark::Counter(percentile(sync_lat, 99.0));
    }
    double wall_s =
        std::chrono::duration_cast<std::chrono::nanoseconds>(wall_end - wall_start)
            .count() /
        1e9;
    if (wall_s > 0) {
      state.counters["wall_ops_per_sec"] =
          benchmark::Counter(static_cast<double>(kTotal) / wall_s);
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * kTotal);
}
BENCHMARK(BM_TimeBasedFlushMixedSyncAsync_VectorDB)
    ->Iterations(1)
    ->ArgsProduct({{8, 16}, {10, 25}, {0, 50}, {32, 128, 512, 768}, {0, 1}});

// Time-based flush: low write rate (delay between writes).
static void BM_TimeBasedFlushLowRate_VectorDB(benchmark::State& state) {
  const int interval_ms = static_cast<int>(state.range(0));
  const int num_threads = static_cast<int>(state.range(1));
  const int dim = static_cast<int>(state.range(2));
  const bool use_fp64 = (state.range(3) != 0);
  constexpr int kWritesPerThread = 100;
  constexpr int kDelayMs = 10;
  double total_wall_ns = 0;

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.max_async_queue_depth = 10000;
    opts.max_async_flush_interval_ms = static_cast<uint64_t>(interval_ms);
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    std::barrier sync_barrier(num_threads);
    state.ResumeTiming();

    auto wall_start = std::chrono::high_resolution_clock::now();
    std::vector<std::thread> threads;
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back([&wal, &sync_barrier, t, dim, use_fp64, kDelayMs] {
        sync_barrier.arrive_and_wait();
        for (int i = 0; i < kWritesPerThread; i++) {
          WriteBatch batch;
          MakeVectorPutBatch(static_cast<uint64_t>(t) * 1000 + i, dim,
                            use_fp64, &batch);
          Status ws = wal->Write(WriteOptions(), &batch);
          if (!ws.ok()) return;
          std::this_thread::sleep_for(std::chrono::milliseconds(kDelayMs));
        }
      });
    }
    for (auto& t : threads) t.join();
    auto wall_end = std::chrono::high_resolution_clock::now();
    total_wall_ns += static_cast<double>(
        std::chrono::duration_cast<std::chrono::nanoseconds>(wall_end - wall_start)
            .count());

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  int64_t total_ops = state.iterations() * num_threads * kWritesPerThread;
  state.SetItemsProcessed(total_ops);
  if (total_wall_ns > 0) {
    state.counters["wall_ops_per_sec"] =
        benchmark::Counter(static_cast<double>(total_ops) / (total_wall_ns / 1e9));
  }
}
BENCHMARK(BM_TimeBasedFlushLowRate_VectorDB)
    ->ArgsProduct({{0, 50}, {1, 2}, {32, 128, 512, 768}, {0, 1}});

// Time-based flush: throughput scaling (threads x interval).
static void BM_TimeBasedFlushScaling_VectorDB(benchmark::State& state) {
  const int num_threads = static_cast<int>(state.range(0));
  const int interval_ms = static_cast<int>(state.range(1));
  const int dim = static_cast<int>(state.range(2));
  const bool use_fp64 = (state.range(3) != 0);
  constexpr int kWritesPerThread = 500;
  double total_wall_ns = 0;

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.max_async_queue_depth = 10000;
    opts.max_async_flush_interval_ms = static_cast<uint64_t>(interval_ms);
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    std::barrier sync_barrier(num_threads);
    state.ResumeTiming();

    auto wall_start = std::chrono::high_resolution_clock::now();
    std::vector<std::thread> threads;
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back([&wal, &sync_barrier, t, dim, use_fp64] {
        sync_barrier.arrive_and_wait();
        for (int i = 0; i < kWritesPerThread; i++) {
          WriteBatch batch;
          MakeVectorPutBatch(static_cast<uint64_t>(t) * 100000 + i, dim,
                            use_fp64, &batch);
          Status ws = wal->Write(WriteOptions(), &batch);
          if (!ws.ok()) return;
        }
      });
    }
    for (auto& t : threads) t.join();
    auto wall_end = std::chrono::high_resolution_clock::now();
    total_wall_ns += static_cast<double>(
        std::chrono::duration_cast<std::chrono::nanoseconds>(wall_end - wall_start)
            .count());

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  int64_t total_ops = state.iterations() * num_threads * kWritesPerThread;
  state.SetItemsProcessed(total_ops);
  if (total_wall_ns > 0) {
    state.counters["wall_ops_per_sec"] =
        benchmark::Counter(static_cast<double>(total_ops) / (total_wall_ns / 1e9));
  }
}
BENCHMARK(BM_TimeBasedFlushScaling_VectorDB)
    ->ArgsProduct({{1, 4, 8, 16}, {0, 50, 100}, {32, 128, 512, 768}, {0, 1}});

// Production pattern: 90% async, 10% sync threads. Measures async/sync p99 latency
// and throughput. Async writers get coalesced into sync groups; async p99 should
// increase with sync_ratio if group commit coalescing works.
static void BM_MixedSyncAsync(benchmark::State& state) {
  const int num_threads = static_cast<int>(state.range(0));
  const int sync_pct = static_cast<int>(state.range(1));
  constexpr int kTotal = 2000;
  const int per_thread = kTotal / num_threads;
  const int num_sync =
      std::max(1, (num_threads * sync_pct + 99) / 100);

  auto percentile = [](const std::vector<int64_t>& sorted, double p) -> double {
    if (sorted.empty()) return 0.0;
    double rank = (p / 100.0) * static_cast<double>(sorted.size() - 1);
    size_t lo = static_cast<size_t>(rank);
    size_t hi = std::min(lo + 1, sorted.size() - 1);
    double frac = rank - static_cast<double>(lo);
    return (1.0 - frac) * static_cast<double>(sorted[lo]) +
           frac * static_cast<double>(sorted[hi]);
  };

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    std::vector<std::vector<WriteBatch>> all_batches(
        static_cast<size_t>(num_threads));
    std::vector<bool> use_sync(static_cast<size_t>(num_threads));
    for (int t = 0; t < num_threads; t++) {
      use_sync[static_cast<size_t>(t)] = (t < num_sync);
      all_batches[static_cast<size_t>(t)].reserve(
          static_cast<size_t>(per_thread));
      for (int i = 0; i < per_thread; i++) {
        WriteBatch b;
        b.Put("t" + std::to_string(t) + "_k" + std::to_string(i), "v");
        all_batches[static_cast<size_t>(t)].push_back(std::move(b));
      }
    }
    state.ResumeTiming();

    std::barrier sync_barrier(num_threads);
    std::vector<std::vector<int64_t>> latencies_per_thread(
        static_cast<size_t>(num_threads));
    std::vector<std::thread> threads;

    auto wall_start = std::chrono::high_resolution_clock::now();
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back(
          [&wal, &latencies_per_thread, &all_batches, &use_sync, &sync_barrier,
           t, per_thread]() {
            auto& lat = latencies_per_thread[static_cast<size_t>(t)];
            auto& batches = all_batches[static_cast<size_t>(t)];
            lat.reserve(static_cast<size_t>(per_thread));
            WriteOptions wo;
            wo.sync = use_sync[static_cast<size_t>(t)];
            sync_barrier.arrive_and_wait();
            for (int i = 0; i < per_thread; i++) {
              auto t0 = std::chrono::high_resolution_clock::now();
              Status ws = wal->Write(wo, &batches[static_cast<size_t>(i)]);
              auto t1 = std::chrono::high_resolution_clock::now();
              if (!ws.ok()) return;
              lat.push_back(static_cast<int64_t>(
                  std::chrono::duration_cast<std::chrono::nanoseconds>(t1 - t0)
                      .count()));
            }
          });
    }
    for (auto& t : threads) t.join();
    auto wall_end = std::chrono::high_resolution_clock::now();

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }

    std::vector<int64_t> async_lat, sync_lat;
    for (int t = 0; t < num_threads; t++) {
      const auto& lat = latencies_per_thread[static_cast<size_t>(t)];
      if (use_sync[static_cast<size_t>(t)]) {
        sync_lat.insert(sync_lat.end(), lat.begin(), lat.end());
      } else {
        async_lat.insert(async_lat.end(), lat.begin(), lat.end());
      }
    }
    if (!async_lat.empty()) {
      std::sort(async_lat.begin(), async_lat.end());
      state.counters["async_p99_ns"] =
          benchmark::Counter(percentile(async_lat, 99.0));
    }
    if (!sync_lat.empty()) {
      std::sort(sync_lat.begin(), sync_lat.end());
      state.counters["sync_p99_ns"] =
          benchmark::Counter(percentile(sync_lat, 99.0));
    }
    double wall_s =
        std::chrono::duration_cast<std::chrono::nanoseconds>(wall_end - wall_start)
            .count() /
        1e9;
    if (wall_s > 0) {
      state.counters["wall_ops_per_sec"] =
          benchmark::Counter(static_cast<double>(kTotal) / wall_s);
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * kTotal);
}
BENCHMARK(BM_MixedSyncAsync)
    ->Iterations(1)
    ->ArgsProduct({{8, 16, 32}, {10, 25, 50}});

// VectorDB: Mixed sync/async pattern.
static void BM_MixedSyncAsync_VectorDB(benchmark::State& state) {
  const int num_threads = static_cast<int>(state.range(0));
  const int sync_pct = static_cast<int>(state.range(1));
  const int dim = static_cast<int>(state.range(2));
  const bool use_fp64 = (state.range(3) != 0);
  constexpr int kTotal = 2000;
  const int per_thread = kTotal / num_threads;
  const int num_sync =
      std::max(1, (num_threads * sync_pct + 99) / 100);

  auto percentile = [](const std::vector<int64_t>& sorted, double p) -> double {
    if (sorted.empty()) return 0.0;
    double rank = (p / 100.0) * static_cast<double>(sorted.size() - 1);
    size_t lo = static_cast<size_t>(rank);
    size_t hi = std::min(lo + 1, sorted.size() - 1);
    double frac = rank - static_cast<double>(lo);
    return (1.0 - frac) * static_cast<double>(sorted[lo]) +
           frac * static_cast<double>(sorted[hi]);
  };

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    std::vector<std::vector<WriteBatch>> all_batches(
        static_cast<size_t>(num_threads));
    std::vector<bool> use_sync(static_cast<size_t>(num_threads));
    for (int t = 0; t < num_threads; t++) {
      use_sync[static_cast<size_t>(t)] = (t < num_sync);
      all_batches[static_cast<size_t>(t)].reserve(
          static_cast<size_t>(per_thread));
      for (int i = 0; i < per_thread; i++) {
        WriteBatch b;
        MakeVectorPutBatch(static_cast<uint64_t>(t) * 10000 + i, dim, use_fp64,
                          &b);
        all_batches[static_cast<size_t>(t)].push_back(std::move(b));
      }
    }
    state.ResumeTiming();

    std::barrier sync_barrier(num_threads);
    std::vector<std::vector<int64_t>> latencies_per_thread(
        static_cast<size_t>(num_threads));
    std::vector<std::thread> threads;

    auto wall_start = std::chrono::high_resolution_clock::now();
    for (int t = 0; t < num_threads; t++) {
      threads.emplace_back(
          [&wal, &latencies_per_thread, &all_batches, &use_sync, &sync_barrier,
           t, per_thread]() {
            auto& lat = latencies_per_thread[static_cast<size_t>(t)];
            auto& batches = all_batches[static_cast<size_t>(t)];
            lat.reserve(static_cast<size_t>(per_thread));
            WriteOptions wo;
            wo.sync = use_sync[static_cast<size_t>(t)];
            sync_barrier.arrive_and_wait();
            for (int i = 0; i < per_thread; i++) {
              auto t0 = std::chrono::high_resolution_clock::now();
              Status ws = wal->Write(wo, &batches[static_cast<size_t>(i)]);
              auto t1 = std::chrono::high_resolution_clock::now();
              if (!ws.ok()) return;
              lat.push_back(static_cast<int64_t>(
                  std::chrono::duration_cast<std::chrono::nanoseconds>(t1 - t0)
                      .count()));
            }
          });
    }
    for (auto& t : threads) t.join();
    auto wall_end = std::chrono::high_resolution_clock::now();

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }

    std::vector<int64_t> async_lat, sync_lat;
    for (int t = 0; t < num_threads; t++) {
      const auto& lat = latencies_per_thread[static_cast<size_t>(t)];
      if (use_sync[static_cast<size_t>(t)]) {
        sync_lat.insert(sync_lat.end(), lat.begin(), lat.end());
      } else {
        async_lat.insert(async_lat.end(), lat.begin(), lat.end());
      }
    }
    if (!async_lat.empty()) {
      std::sort(async_lat.begin(), async_lat.end());
      state.counters["async_p99_ns"] =
          benchmark::Counter(percentile(async_lat, 99.0));
    }
    if (!sync_lat.empty()) {
      std::sort(sync_lat.begin(), sync_lat.end());
      state.counters["sync_p99_ns"] =
          benchmark::Counter(percentile(sync_lat, 99.0));
    }
    double wall_s =
        std::chrono::duration_cast<std::chrono::nanoseconds>(wall_end - wall_start)
            .count() /
        1e9;
    if (wall_s > 0) {
      state.counters["wall_ops_per_sec"] =
          benchmark::Counter(static_cast<double>(kTotal) / wall_s);
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * kTotal);
}
BENCHMARK(BM_MixedSyncAsync_VectorDB)
    ->Iterations(1)
    ->ArgsProduct({{8, 16, 32}, {10, 25, 50}, {32, 128, 512, 768}, {0, 1}});

// Compare 1×1MB vs 1000×1KB (same 100 MB total). Scenario B should be slower per byte
// due to per-record header and CRC overhead.
static void BM_SingleLargeRecordVsManySmall(benchmark::State& state) {
  const bool many_small = (state.range(0) == 1);
  constexpr int64_t kTotalBytes = 100LL * 1024 * 1024;  // 100 MB

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    state.ResumeTiming();

    if (!many_small) {
      std::string big(1024 * 1024, 'X');
      WriteBatch batch;
      for (int i = 0; i < 100; i++) {
        batch.Clear();
        batch.Put("k", big);
        Status ws = wal->Write(WriteOptions(), &batch);
        if (!ws.ok()) {
          state.SkipWithError(ws.ToString().c_str());
          return;
        }
      }
    } else {
      std::string small(1024, 'x');
      for (int i = 0; i < 100; i++) {
        WriteBatch batch;
        for (int j = 0; j < 1000; j++) {
          batch.Put("k" + std::to_string(i * 1000 + j), small);
        }
        Status ws = wal->Write(WriteOptions(), &batch);
        if (!ws.ok()) {
          state.SkipWithError(ws.ToString().c_str());
          return;
        }
      }
    }

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetBytesProcessed(state.iterations() * kTotalBytes);
}
BENCHMARK(BM_SingleLargeRecordVsManySmall)->Arg(0)->Arg(1);

// VectorDB: 1 large (768 fp64) vs many small (32 fp32) records.
static void BM_SingleLargeRecordVsManySmall_VectorDB(benchmark::State& state) {
  const bool many_small = (state.range(0) == 1);
  const int dim = state.range(1);
  const bool use_fp64 = (state.range(2) != 0);

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    state.ResumeTiming();

    if (!many_small) {
      for (int i = 0; i < 100; i++) {
        WriteBatch batch;
        MakeVectorPutBatch(static_cast<uint64_t>(i), dim, use_fp64, &batch);
        Status ws = wal->Write(WriteOptions(), &batch);
        if (!ws.ok()) {
          state.SkipWithError(ws.ToString().c_str());
          return;
        }
      }
    } else {
      for (int i = 0; i < 100; i++) {
        WriteBatch batch;
        for (int j = 0; j < 1000; j++) {
          MakeVectorPutBatch(static_cast<uint64_t>(i * 1000 + j), dim, use_fp64,
                            &batch);
        }
        Status ws = wal->Write(WriteOptions(), &batch);
        if (!ws.ok()) {
          state.SkipWithError(ws.ToString().c_str());
          return;
        }
      }
    }

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  int64_t bytes = many_small ? 100 * 1000 * GetVectorPayloadSize(dim, use_fp64)
                             : 100 * GetVectorPayloadSize(dim, use_fp64);
  state.SetBytesProcessed(state.iterations() * bytes);
}
BENCHMARK(BM_SingleLargeRecordVsManySmall_VectorDB)
    ->ArgsProduct({{0, 1}, {32, 128, 512, 768}, {0, 1}});

// Time from PurgeObsoleteFiles() to first successful Write() after stall.
// Measures how long a stalled writer is blocked before retry succeeds.
static void BM_WriteStallRecoveryTime(benchmark::State& state) {
  constexpr uint64_t kLimit = 512 * 1024;
  constexpr uint64_t kFileSize = 64 * 1024;
  std::string val(2048, 'x');

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.max_total_wal_size = kLimit;
    opts.max_wal_file_size = kFileSize;

    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    while (true) {
      WriteBatch batch;
      batch.Put("fill", val);
      s = wal->Write(WriteOptions(), &batch);
      if (s.IsBusy()) break;
      if (!s.ok()) {
        state.SkipWithError(s.ToString().c_str());
        return;
      }
    }

    wal->SetMinLogNumberToKeep(wal->GetCurrentLogNumber());

    state.ResumeTiming();
    auto t0 = std::chrono::high_resolution_clock::now();
    s = wal->PurgeObsoleteFiles();
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    WriteBatch retry_batch;
    retry_batch.Put("after_purge", "v");
    while (true) {
      s = wal->Write(WriteOptions(), &retry_batch);
      if (s.ok()) break;
      if (!s.IsBusy()) {
        state.SkipWithError(s.ToString().c_str());
        return;
      }
      std::this_thread::sleep_for(std::chrono::microseconds(10));
    }
    auto t1 = std::chrono::high_resolution_clock::now();
    state.PauseTiming();

    double stall_us =
        std::chrono::duration_cast<std::chrono::microseconds>(t1 - t0).count();
    state.counters["stall_recovery_us"] = benchmark::Counter(stall_us);

    wal->Close();
  }
}
BENCHMARK(BM_WriteStallRecoveryTime)->Iterations(20);

// VectorDB: Write stall recovery time.
static void BM_WriteStallRecoveryTime_VectorDB(benchmark::State& state) {
  const int dim = state.range(0);
  const bool use_fp64 = (state.range(1) != 0);
  constexpr uint64_t kLimit = 512 * 1024;
  constexpr uint64_t kFileSize = 64 * 1024;

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.max_total_wal_size = kLimit;
    opts.max_wal_file_size = kFileSize;

    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    WriteBatch fill_batch;
    MakeVectorPutBatch(0, dim, use_fp64, &fill_batch);
    while (true) {
      s = wal->Write(WriteOptions(), &fill_batch);
      if (s.IsBusy()) break;
      if (!s.ok()) {
        state.SkipWithError(s.ToString().c_str());
        return;
      }
    }

    wal->SetMinLogNumberToKeep(wal->GetCurrentLogNumber());

    state.ResumeTiming();
    auto t0 = std::chrono::high_resolution_clock::now();
    s = wal->PurgeObsoleteFiles();
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    WriteBatch retry_batch;
    MakeVectorPutBatch(999999, dim, use_fp64, &retry_batch);
    while (true) {
      s = wal->Write(WriteOptions(), &retry_batch);
      if (s.ok()) break;
      if (!s.IsBusy()) {
        state.SkipWithError(s.ToString().c_str());
        return;
      }
      std::this_thread::sleep_for(std::chrono::microseconds(10));
    }
    auto t1 = std::chrono::high_resolution_clock::now();
    state.PauseTiming();

    double stall_us =
        std::chrono::duration_cast<std::chrono::microseconds>(t1 - t0).count();
    state.counters["stall_recovery_us"] = benchmark::Counter(stall_us);

    wal->Close();
  }
}
BENCHMARK(BM_WriteStallRecoveryTime_VectorDB)
    ->Iterations(20)
    ->ArgsProduct({{32, 128, 512, 768}, {0, 1}});

// WalIterator scan speed for replication/change capture. Pre-populate WAL, then
// NewWalIterator(0) and consume all records.
static void BM_WalIteratorScanSpeed(benchmark::State& state) {
  const int num_records = static_cast<int>(state.range(0));
  const int record_size = static_cast<int>(state.range(1));

  BenchTmpDir dir;
  WALOptions opts;
  opts.wal_dir = dir.path();

  {
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    std::string val(static_cast<size_t>(record_size), 'v');
    for (int i = 0; i < num_records; i++) {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), val);
      s = wal->Write(WriteOptions(), &batch);
      if (!s.ok()) {
        state.SkipWithError(s.ToString().c_str());
        return;
      }
    }
    s = wal->Close();
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
  }

  for (auto _ : state) {
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    std::unique_ptr<WalIterator> iter;
    s = wal->NewWalIterator(0, &iter);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    int count = 0;
    while (iter->Valid()) {
      count++;
      iter->Next();
    }
    benchmark::DoNotOptimize(count);
  }
  state.SetItemsProcessed(state.iterations() * num_records);
  state.SetBytesProcessed(state.iterations() * static_cast<int64_t>(num_records) *
                          (record_size + 16));
}
BENCHMARK(BM_WalIteratorScanSpeed)
    ->ArgsProduct({{10000, 100000, 1000000}, {256, 4096}});

// VectorDB: WalIterator scan speed.
static void BM_WalIteratorScanSpeed_VectorDB(benchmark::State& state) {
  const int num_records = static_cast<int>(state.range(0));
  const int dim = static_cast<int>(state.range(1));
  const bool use_fp64 = (state.range(2) != 0);

  BenchTmpDir dir;
  WALOptions opts;
  opts.wal_dir = dir.path();

  {
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    for (int i = 0; i < num_records; i++) {
      WriteBatch batch;
      MakeVectorPutBatch(static_cast<uint64_t>(i), dim, use_fp64, &batch);
      s = wal->Write(WriteOptions(), &batch);
      if (!s.ok()) {
        state.SkipWithError(s.ToString().c_str());
        return;
      }
    }
    s = wal->Close();
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
  }

  for (auto _ : state) {
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    std::unique_ptr<WalIterator> iter;
    s = wal->NewWalIterator(0, &iter);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    int count = 0;
    while (iter->Valid()) {
      count++;
      iter->Next();
    }
    benchmark::DoNotOptimize(count);
  }
  state.SetItemsProcessed(state.iterations() * num_records);
  int64_t payload_size = static_cast<int64_t>(GetVectorPayloadSize(dim, use_fp64));
  state.SetBytesProcessed(state.iterations() * static_cast<int64_t>(num_records) *
                          (payload_size + 16));
}
BENCHMARK(BM_WalIteratorScanSpeed_VectorDB)
    ->ArgsProduct({{10000, 100000, 1000000}, {32, 128, 512, 768}, {0, 1}});

// Mixed read/write: realistic load — 20k inserts/s (1 per batch), readers do
// 5k-record reads; read_pct 10–90 (thread mix); dim 128 fp32; 5 s phase.
// Reports write/read throughput, write and read latency (p50/p95/p99), and
// optional records_per_read_mean, read_MB_per_sec.
static void BM_MixedReadWrite_VectorDB(benchmark::State& state) {
  const int read_pct = static_cast<int>(state.range(0));
  const int num_readers = read_pct / 10;
  const int num_writers = 10 - num_readers;
  constexpr double kPhaseSec = 5.0;
  constexpr int kTargetWriteRate = 20000;
  constexpr int kRecordsPerRead = 5000;
  constexpr int kPrePopulateBatches = 1000;
  constexpr int kDim = 128;
  constexpr bool kUseFp64 = false;

  auto percentile = [](const std::vector<int64_t>& sorted, double p) -> double {
    if (sorted.empty()) return 0.0;
    double rank = (p / 100.0) * static_cast<double>(sorted.size() - 1);
    size_t lo = static_cast<size_t>(rank);
    size_t hi = std::min(lo + 1, sorted.size() - 1);
    double frac = rank - static_cast<double>(lo);
    return (1.0 - frac) * static_cast<double>(sorted[lo]) +
           frac * static_cast<double>(sorted[hi]);
  };

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    for (int i = 0; i < kPrePopulateBatches; i++) {
      WriteBatch batch;
      MakeVectorPutBatch(static_cast<uint64_t>(i), kDim, kUseFp64, &batch);
      s = wal->Write(WriteOptions(), &batch);
      if (!s.ok()) {
        state.SkipWithError(s.ToString().c_str());
        return;
      }
    }
    state.ResumeTiming();

    std::atomic<uint64_t> write_count{0};
    std::atomic<uint64_t> next_id{static_cast<uint64_t>(kPrePopulateBatches)};
    std::atomic<uint64_t> phase_start_ns{0};
    std::barrier sync_barrier(10);

    std::vector<std::vector<int64_t>> write_latencies_per_thread(
        static_cast<size_t>(num_writers));
    std::vector<std::vector<int64_t>> read_latencies_per_thread(
        static_cast<size_t>(num_readers));
    std::vector<std::atomic<uint64_t>> read_5k_ops(
        static_cast<size_t>(num_readers));
    std::vector<std::atomic<uint64_t>> records_read(
        static_cast<size_t>(num_readers));
    for (int r = 0; r < num_readers; r++) {
      read_5k_ops[static_cast<size_t>(r)].store(0);
      records_read[static_cast<size_t>(r)].store(0);
    }

    std::vector<std::thread> threads;

    for (int t = 0; t < num_writers; t++) {
      const size_t tid = static_cast<size_t>(t);
      threads.emplace_back(
          [&wal, &write_count, &next_id, &phase_start_ns, &sync_barrier,
           &write_latencies_per_thread, tid]() {
            auto& lat = write_latencies_per_thread[tid];
            sync_barrier.arrive_and_wait();
            if (tid == 0) {
              phase_start_ns.store(
                  std::chrono::steady_clock::now().time_since_epoch().count());
            }
            while (phase_start_ns.load() == 0) {
              std::this_thread::yield();
            }
            uint64_t start_ns = phase_start_ns.load();
            const int64_t phase_ns = static_cast<int64_t>(kPhaseSec * 1e9);
            WriteOptions wo;
            wo.sync = false;
            while (true) {
              int64_t now_ns =
                  static_cast<int64_t>(std::chrono::steady_clock::now()
                                           .time_since_epoch()
                                           .count());
              if (now_ns - static_cast<int64_t>(start_ns) >= phase_ns) break;
              double elapsed_sec =
                  (now_ns - static_cast<int64_t>(start_ns)) / 1e9;
              uint64_t target =
                  static_cast<uint64_t>(kTargetWriteRate * elapsed_sec);
              if (write_count.load() >= target) {
                std::this_thread::sleep_for(std::chrono::milliseconds(1));
                continue;
              }
              uint64_t id = next_id.fetch_add(1);
              WriteBatch batch;
              MakeVectorPutBatch(id, kDim, kUseFp64, &batch);
              auto t0 = std::chrono::high_resolution_clock::now();
              Status ws = wal->Write(wo, &batch);
              auto t1 = std::chrono::high_resolution_clock::now();
              if (!ws.ok()) return;
              write_count++;
              lat.push_back(static_cast<int64_t>(
                  std::chrono::duration_cast<std::chrono::nanoseconds>(t1 - t0)
                      .count()));
            }
          });
    }

    for (int t = 0; t < num_readers; t++) {
      const size_t rid = static_cast<size_t>(t);
      threads.emplace_back(
          [&wal, &phase_start_ns, &sync_barrier,
           &read_latencies_per_thread, &read_5k_ops, &records_read, rid,
           num_writers]() {
            auto& lat = read_latencies_per_thread[rid];
            sync_barrier.arrive_and_wait();
            if (num_writers == 0 && rid == 0) {
              phase_start_ns.store(
                  std::chrono::steady_clock::now().time_since_epoch().count());
            }
            while (phase_start_ns.load() == 0) {
              std::this_thread::yield();
            }
            uint64_t start_ns = phase_start_ns.load();
            const int64_t phase_ns = static_cast<int64_t>(kPhaseSec * 1e9);
            SequenceNumber start_seq = 0;
            while (true) {
              int64_t now_ns =
                  static_cast<int64_t>(std::chrono::steady_clock::now()
                                           .time_since_epoch()
                                           .count());
              if (now_ns - static_cast<int64_t>(start_ns) >= phase_ns) break;
              auto t0 = std::chrono::high_resolution_clock::now();
              std::unique_ptr<WalIterator> iter;
              Status rs = wal->NewWalIterator(start_seq, &iter);
              if (!rs.ok()) break;
              int count = 0;
              SequenceNumber last_seq = start_seq;
              while (count < kRecordsPerRead && iter->Valid()) {
                iter->GetBatch();
                last_seq = iter->GetSequenceNumber();
                iter->Next();
                count++;
              }
              auto t1 = std::chrono::high_resolution_clock::now();
              lat.push_back(static_cast<int64_t>(
                  std::chrono::duration_cast<std::chrono::nanoseconds>(t1 - t0)
                      .count()));
              read_5k_ops[rid]++;
              records_read[rid] += static_cast<uint64_t>(count);
              if (count > 0) {
                start_seq = last_seq + 1;
              }
              if (count < kRecordsPerRead) {
                start_seq = 0;
              }
            }
          });
    }

    for (auto& t : threads) t.join();

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }

    uint64_t total_writes = write_count.load();
    uint64_t total_5k_reads = 0;
    uint64_t total_records_read = 0;
    for (int r = 0; r < num_readers; r++) {
      total_5k_reads += read_5k_ops[static_cast<size_t>(r)].load();
      total_records_read += records_read[static_cast<size_t>(r)].load();
    }

    if (kPhaseSec > 0) {
      state.counters["write_ops_per_sec"] = benchmark::Counter(
          static_cast<double>(total_writes) / kPhaseSec);
      state.counters["read_5k_ops_per_sec"] = benchmark::Counter(
          static_cast<double>(total_5k_reads) / kPhaseSec);
      state.counters["records_read_per_sec"] = benchmark::Counter(
          static_cast<double>(total_records_read) / kPhaseSec);
    }

    std::vector<int64_t> all_write_lat;
    for (const auto& l : write_latencies_per_thread) {
      all_write_lat.insert(all_write_lat.end(), l.begin(), l.end());
    }
    if (!all_write_lat.empty()) {
      std::sort(all_write_lat.begin(), all_write_lat.end());
      state.counters["write_p50_ns"] =
          benchmark::Counter(percentile(all_write_lat, 50.0));
      state.counters["write_p95_ns"] =
          benchmark::Counter(percentile(all_write_lat, 95.0));
      state.counters["write_p99_ns"] =
          benchmark::Counter(percentile(all_write_lat, 99.0));
    }

    std::vector<int64_t> all_read_lat;
    for (const auto& l : read_latencies_per_thread) {
      all_read_lat.insert(all_read_lat.end(), l.begin(), l.end());
    }
    if (!all_read_lat.empty()) {
      std::sort(all_read_lat.begin(), all_read_lat.end());
      state.counters["read_p50_ns"] =
          benchmark::Counter(percentile(all_read_lat, 50.0));
      state.counters["read_p95_ns"] =
          benchmark::Counter(percentile(all_read_lat, 95.0));
      state.counters["read_p99_ns"] =
          benchmark::Counter(percentile(all_read_lat, 99.0));
    }

    if (total_5k_reads > 0) {
      state.counters["records_per_read_mean"] = benchmark::Counter(
          static_cast<double>(total_records_read) /
          static_cast<double>(total_5k_reads));
      int64_t record_bytes =
          static_cast<int64_t>(8 + GetVectorPayloadSize(kDim, kUseFp64));
      state.counters["read_MB_per_sec"] = benchmark::Counter(
          static_cast<double>(total_records_read) * record_bytes / kPhaseSec /
          1e6);
    }

    state.SetItemsProcessed(state.iterations() *
                            (static_cast<int64_t>(total_writes) +
                             static_cast<int64_t>(total_5k_reads)));
    state.ResumeTiming();
  }
}
BENCHMARK(BM_MixedReadWrite_VectorDB)
    ->Iterations(1)
    ->Arg(10)
    ->Arg(20)
    ->Arg(30)
    ->Arg(40)
    ->Arg(50)
    ->Arg(60)
    ->Arg(70)
    ->Arg(80)
    ->Arg(90);

// Write throughput and latency with vs without concurrent PurgeObsoleteFiles.
// Each phase uses a separate directory and a warmup pass to normalize
// filesystem cache state. Only the purge thread differs between phases.
static void BM_PurgeUnderWriteLoad(benchmark::State& state) {
  const int write_threads = static_cast<int>(state.range(0));
  const int purge_interval_ms = static_cast<int>(state.range(1));
  constexpr int kWritesPerPhase = 2000;
  constexpr int kWarmupWrites = 500;
  constexpr int kMaxWalFileSize = 64 * 1024;
  const int per_thread = kWritesPerPhase / write_threads;
  const int warmup_per_thread = kWarmupWrites / write_threads;

  auto percentile = [](const std::vector<int64_t>& sorted, double p) -> double {
    if (sorted.empty()) return 0.0;
    double rank = (p / 100.0) * static_cast<double>(sorted.size() - 1);
    size_t lo = static_cast<size_t>(rank);
    size_t hi = std::min(lo + 1, sorted.size() - 1);
    double frac = rank - static_cast<double>(lo);
    return (1.0 - frac) * static_cast<double>(sorted[lo]) +
           frac * static_cast<double>(sorted[hi]);
  };

  auto run_phase = [&](bool enable_purge, double* throughput_out,
                        double* p99_out) {
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.max_wal_file_size = kMaxWalFileSize;

    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    // Warmup: populate WAL files so both phases start with similar state.
    {
      std::barrier warmup_barrier(write_threads);
      std::vector<std::thread> warmup_threads;
      for (int t = 0; t < write_threads; t++) {
        warmup_threads.emplace_back(
            [&wal, &warmup_barrier, t, warmup_per_thread]() {
              warmup_barrier.arrive_and_wait();
              for (int i = 0; i < warmup_per_thread; i++) {
                WriteBatch batch;
                batch.Put(
                    "w" + std::to_string(t) + "_k" + std::to_string(i), "v");
                wal->Write(WriteOptions(), &batch);
              }
            });
      }
      for (auto& t : warmup_threads) t.join();
    }

    std::atomic<bool> purge_stop{false};
    std::thread purge_thread;
    if (enable_purge) {
      purge_thread = std::thread([&wal, purge_interval_ms, &purge_stop]() {
        while (!purge_stop.load()) {
          std::this_thread::sleep_for(
              std::chrono::milliseconds(purge_interval_ms));
          if (purge_stop.load()) break;
          wal->SetMinLogNumberToKeep(wal->GetCurrentLogNumber());
          wal->PurgeObsoleteFiles();
        }
      });
    }

    std::barrier sync_barrier(write_threads);
    std::vector<std::vector<int64_t>> latencies(
        static_cast<size_t>(write_threads));

    auto wall_start = std::chrono::high_resolution_clock::now();
    std::vector<std::thread> threads;
    for (int t = 0; t < write_threads; t++) {
      threads.emplace_back(
          [&wal, &sync_barrier, &latencies, t, per_thread]() {
            auto& lat = latencies[static_cast<size_t>(t)];
            lat.reserve(static_cast<size_t>(per_thread));
            sync_barrier.arrive_and_wait();
            for (int i = 0; i < per_thread; i++) {
              WriteBatch batch;
              batch.Put(
                  "t" + std::to_string(t) + "_k" + std::to_string(i), "v");
              auto t0 = std::chrono::high_resolution_clock::now();
              Status ws = wal->Write(WriteOptions(), &batch);
              auto t1 = std::chrono::high_resolution_clock::now();
              if (!ws.ok()) return;
              lat.push_back(
                  std::chrono::duration_cast<std::chrono::nanoseconds>(t1 - t0)
                      .count());
            }
          });
    }
    for (auto& t : threads) t.join();
    auto wall_end = std::chrono::high_resolution_clock::now();

    if (enable_purge) {
      purge_stop.store(true);
      purge_thread.join();
    }

    std::vector<int64_t> all;
    for (const auto& l : latencies) {
      all.insert(all.end(), l.begin(), l.end());
    }
    double wall_s =
        std::chrono::duration_cast<std::chrono::nanoseconds>(wall_end -
                                                            wall_start)
            .count() /
        1e9;
    *throughput_out = (wall_s > 0) ? (kWritesPerPhase / wall_s) : 0;
    std::sort(all.begin(), all.end());
    *p99_out = all.empty() ? 0 : percentile(all, 99.0);

    wal->Close();
  };

  for (auto _ : state) {
    state.PauseTiming();
    double throughput_no_purge = 0, p99_no_purge = 0;
    double throughput_with_purge = 0, p99_with_purge = 0;
    state.ResumeTiming();

    run_phase(false, &throughput_no_purge, &p99_no_purge);
    run_phase(true, &throughput_with_purge, &p99_with_purge);

    state.PauseTiming();
    state.counters["throughput_no_purge"] =
        benchmark::Counter(throughput_no_purge);
    state.counters["throughput_with_purge"] =
        benchmark::Counter(throughput_with_purge);
    state.counters["p99_ns_no_purge"] = benchmark::Counter(p99_no_purge);
    state.counters["p99_ns_with_purge"] = benchmark::Counter(p99_with_purge);
    if (throughput_no_purge > 0) {
      state.counters["throughput_drop_pct"] = benchmark::Counter(
          (1.0 - throughput_with_purge / throughput_no_purge) * 100.0);
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * 2 * kWritesPerPhase);
}
BENCHMARK(BM_PurgeUnderWriteLoad)
    ->Iterations(3)
    ->ArgsProduct({{4, 8}, {100, 500, 1000}});

// VectorDB: Purge under write load.
static void BM_PurgeUnderWriteLoad_VectorDB(benchmark::State& state) {
  const int write_threads = static_cast<int>(state.range(0));
  const int purge_interval_ms = static_cast<int>(state.range(1));
  const int dim = static_cast<int>(state.range(2));
  const bool use_fp64 = (state.range(3) != 0);
  constexpr int kWritesPerPhase = 2000;
  constexpr int kWarmupWrites = 500;
  constexpr int kMaxWalFileSize = 64 * 1024;
  const int per_thread = kWritesPerPhase / write_threads;
  const int warmup_per_thread = kWarmupWrites / write_threads;

  auto percentile = [](const std::vector<int64_t>& sorted, double p) -> double {
    if (sorted.empty()) return 0.0;
    double rank = (p / 100.0) * static_cast<double>(sorted.size() - 1);
    size_t lo = static_cast<size_t>(rank);
    size_t hi = std::min(lo + 1, sorted.size() - 1);
    double frac = rank - static_cast<double>(lo);
    return (1.0 - frac) * static_cast<double>(sorted[lo]) +
           frac * static_cast<double>(sorted[hi]);
  };

  auto run_phase = [&](bool enable_purge, double* throughput_out,
                        double* p99_out) {
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.max_wal_file_size = kMaxWalFileSize;

    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }

    {
      std::barrier warmup_barrier(write_threads);
      std::vector<std::thread> warmup_threads;
      for (int t = 0; t < write_threads; t++) {
        warmup_threads.emplace_back(
            [&wal, &warmup_barrier, t, warmup_per_thread, dim, use_fp64]() {
              warmup_barrier.arrive_and_wait();
              for (int i = 0; i < warmup_per_thread; i++) {
                WriteBatch batch;
                MakeVectorPutBatch(static_cast<uint64_t>(t) * 10000 + i, dim,
                                   use_fp64, &batch);
                wal->Write(WriteOptions(), &batch);
              }
            });
      }
      for (auto& t : warmup_threads) t.join();
    }

    std::atomic<bool> purge_stop{false};
    std::thread purge_thread;
    if (enable_purge) {
      purge_thread = std::thread([&wal, purge_interval_ms, &purge_stop]() {
        while (!purge_stop.load()) {
          std::this_thread::sleep_for(
              std::chrono::milliseconds(purge_interval_ms));
          if (purge_stop.load()) break;
          wal->SetMinLogNumberToKeep(wal->GetCurrentLogNumber());
          wal->PurgeObsoleteFiles();
        }
      });
    }

    std::barrier sync_barrier(write_threads);
    std::vector<std::vector<int64_t>> latencies(
        static_cast<size_t>(write_threads));

    auto wall_start = std::chrono::high_resolution_clock::now();
    std::vector<std::thread> threads;
    for (int t = 0; t < write_threads; t++) {
      threads.emplace_back(
          [&wal, &sync_barrier, &latencies, t, per_thread, dim, use_fp64]() {
            auto& lat = latencies[static_cast<size_t>(t)];
            lat.reserve(static_cast<size_t>(per_thread));
            sync_barrier.arrive_and_wait();
            for (int i = 0; i < per_thread; i++) {
              WriteBatch batch;
              MakeVectorPutBatch(static_cast<uint64_t>(t) * 100000 + i, dim,
                                 use_fp64, &batch);
              auto t0 = std::chrono::high_resolution_clock::now();
              Status ws = wal->Write(WriteOptions(), &batch);
              auto t1 = std::chrono::high_resolution_clock::now();
              if (!ws.ok()) return;
              lat.push_back(
                  std::chrono::duration_cast<std::chrono::nanoseconds>(t1 - t0)
                      .count());
            }
          });
    }
    for (auto& t : threads) t.join();
    auto wall_end = std::chrono::high_resolution_clock::now();

    if (enable_purge) {
      purge_stop.store(true);
      purge_thread.join();
    }

    std::vector<int64_t> all;
    for (const auto& l : latencies) {
      all.insert(all.end(), l.begin(), l.end());
    }
    double wall_s =
        std::chrono::duration_cast<std::chrono::nanoseconds>(wall_end -
                                                            wall_start)
            .count() /
        1e9;
    *throughput_out = (wall_s > 0) ? (kWritesPerPhase / wall_s) : 0;
    std::sort(all.begin(), all.end());
    *p99_out = all.empty() ? 0 : percentile(all, 99.0);

    wal->Close();
  };

  for (auto _ : state) {
    state.PauseTiming();
    double throughput_no_purge = 0, p99_no_purge = 0;
    double throughput_with_purge = 0, p99_with_purge = 0;
    state.ResumeTiming();

    run_phase(false, &throughput_no_purge, &p99_no_purge);
    run_phase(true, &throughput_with_purge, &p99_with_purge);

    state.PauseTiming();
    state.counters["throughput_no_purge"] =
        benchmark::Counter(throughput_no_purge);
    state.counters["throughput_with_purge"] =
        benchmark::Counter(throughput_with_purge);
    state.counters["p99_ns_no_purge"] = benchmark::Counter(p99_no_purge);
    state.counters["p99_ns_with_purge"] = benchmark::Counter(p99_with_purge);
    if (throughput_no_purge > 0) {
      state.counters["throughput_drop_pct"] = benchmark::Counter(
          (1.0 - throughput_with_purge / throughput_no_purge) * 100.0);
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * 2 * kWritesPerPhase);
}
BENCHMARK(BM_PurgeUnderWriteLoad_VectorDB)
    ->Iterations(3)
    ->ArgsProduct({{4, 8}, {100, 500, 1000}, {32, 128, 512, 768}, {0, 1}});

// Compression prefix overhead: measure write throughput with no-compression prefix path.
static void BM_CompressionPrefixOverhead(benchmark::State& state) {
  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.wal_compression = kNoCompression;
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    state.ResumeTiming();

    std::string val(512, 'X');
    for (int i = 0; i < 2000; i++) {
      WriteBatch batch;
      batch.Put("key_" + std::to_string(i), val);
      Status ws = wal->Write(WriteOptions(), &batch);
      if (!ws.ok()) {
        state.SkipWithError(ws.ToString().c_str());
        return;
      }
    }

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * 2000);
}
BENCHMARK(BM_CompressionPrefixOverhead);

// VectorDB: Compression prefix overhead.
static void BM_CompressionPrefixOverhead_VectorDB(benchmark::State& state) {
  const int dim = state.range(0);
  const bool use_fp64 = (state.range(1) != 0);

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    opts.wal_compression = kNoCompression;
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    state.ResumeTiming();

    for (int i = 0; i < 2000; i++) {
      WriteBatch batch;
      MakeVectorPutBatch(static_cast<uint64_t>(i), dim, use_fp64, &batch);
      Status ws = wal->Write(WriteOptions(), &batch);
      if (!ws.ok()) {
        state.SkipWithError(ws.ToString().c_str());
        return;
      }
    }

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * 2000);
}
BENCHMARK(BM_CompressionPrefixOverhead_VectorDB)->ArgsProduct({{32, 128, 512, 768}, {0, 1}});

#ifdef MWAL_HAVE_ZSTD

// Recovery with zstd compression. Measures decompression cost during recovery.
static void BM_RecoveryWithCompression(benchmark::State& state) {
  const int num_records = static_cast<int>(state.range(0));
  const int payload_size = static_cast<int>(state.range(1));
  const bool compress = (state.range(2) != 0);

  BenchTmpDir dir;
  WALOptions opts;
  opts.wal_dir = dir.path();
  opts.wal_compression = compress ? kZSTD : kNoCompression;

  {
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    std::string val(static_cast<size_t>(payload_size), 'x');
    for (int i = 0; i < num_records; i++) {
      WriteBatch batch;
      batch.Put("k" + std::to_string(i), val);
      s = wal->Write(WriteOptions(), &batch);
      if (!s.ok()) {
        state.SkipWithError(s.ToString().c_str());
        return;
      }
    }
    s = wal->Close();
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
  }

  size_t total_decompressed = 0;
  for (auto _ : state) {
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    size_t count = 0;
    size_t bytes_this_iter = 0;
    s = wal->Recover([&](SequenceNumber, WriteBatch* batch) {
      count++;
      bytes_this_iter += batch->GetDataSize();
      return Status::OK();
    });
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    total_decompressed += bytes_this_iter;
    benchmark::DoNotOptimize(count);
  }
  state.SetItemsProcessed(state.iterations() * num_records);
  state.SetBytesProcessed(static_cast<int64_t>(total_decompressed));
}
BENCHMARK(BM_RecoveryWithCompression)
    ->ArgsProduct({{1000, 10000, 100000}, {512, 4096}, {0, 1}});

// VectorDB: Recovery with compression.
static void BM_RecoveryWithCompression_VectorDB(benchmark::State& state) {
  const int num_records = static_cast<int>(state.range(0));
  const bool compress = (state.range(1) != 0);
  const int dim = static_cast<int>(state.range(2));
  const bool use_fp64 = (state.range(3) != 0);

  BenchTmpDir dir;
  WALOptions opts;
  opts.wal_dir = dir.path();
  opts.wal_compression = compress ? kZSTD : kNoCompression;

  {
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    for (int i = 0; i < num_records; i++) {
      WriteBatch batch;
      MakeVectorPutBatch(static_cast<uint64_t>(i), dim, use_fp64, &batch);
      s = wal->Write(WriteOptions(), &batch);
      if (!s.ok()) {
        state.SkipWithError(s.ToString().c_str());
        return;
      }
    }
    s = wal->Close();
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
  }

  size_t total_decompressed = 0;
  for (auto _ : state) {
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    size_t count = 0;
    size_t bytes_this_iter = 0;
    s = wal->Recover([&](SequenceNumber, WriteBatch* batch) {
      count++;
      bytes_this_iter += batch->GetDataSize();
      return Status::OK();
    });
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    total_decompressed += bytes_this_iter;
    benchmark::DoNotOptimize(count);
  }
  state.SetItemsProcessed(state.iterations() * num_records);
  state.SetBytesProcessed(static_cast<int64_t>(total_decompressed));
}
BENCHMARK(BM_RecoveryWithCompression_VectorDB)
    ->ArgsProduct({{1000, 10000, 100000}, {0, 1}, {32, 128, 512, 768}, {0, 1}});

// Write throughput with compression on vs off.
static void BM_DBWalCompression(benchmark::State& state) {
  const bool compress = state.range(0) != 0;

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    if (compress) opts.wal_compression = kZSTD;
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    state.ResumeTiming();

    std::string val(1024, 'A');
    for (int i = 0; i < 1000; i++) {
      WriteBatch batch;
      batch.Put("key_" + std::to_string(i), val);
      Status ws = wal->Write(WriteOptions(), &batch);
      if (!ws.ok()) {
        state.SkipWithError(ws.ToString().c_str());
        return;
      }
    }

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * 1000);
}
BENCHMARK(BM_DBWalCompression)->Arg(0)->Arg(1);

// VectorDB: Write throughput with compression on vs off.
static void BM_DBWalCompression_VectorDB(benchmark::State& state) {
  const bool compress = state.range(0) != 0;
  const int dim = state.range(1);
  const bool use_fp64 = (state.range(2) != 0);

  for (auto _ : state) {
    state.PauseTiming();
    BenchTmpDir dir;
    WALOptions opts;
    opts.wal_dir = dir.path();
    if (compress) opts.wal_compression = kZSTD;
    std::unique_ptr<DBWal> wal;
    Status s = DBWal::Open(opts, Env::Default(), &wal);
    if (!s.ok()) {
      state.SkipWithError(s.ToString().c_str());
      return;
    }
    state.ResumeTiming();

    for (int i = 0; i < 1000; i++) {
      WriteBatch batch;
      MakeVectorPutBatch(static_cast<uint64_t>(i), dim, use_fp64, &batch);
      Status ws = wal->Write(WriteOptions(), &batch);
      if (!ws.ok()) {
        state.SkipWithError(ws.ToString().c_str());
        return;
      }
    }

    state.PauseTiming();
    Status cs = wal->Close();
    if (!cs.ok()) {
      state.SkipWithError(cs.ToString().c_str());
      return;
    }
    state.ResumeTiming();
  }
  state.SetItemsProcessed(state.iterations() * 1000);
}
BENCHMARK(BM_DBWalCompression_VectorDB)->ArgsProduct({{0, 1}, {32, 128, 512, 768}, {0, 1}});

#endif

}  // namespace mwal
