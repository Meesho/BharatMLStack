// Dedicated async write coalescer for mwal.
// Routes async (sync=false) writes through an MPSC queue drained by a single
// writer thread, eliminating WriteThread::mu_ contention for async paths.

#pragma once

#include <chrono>
#include <condition_variable>
#include <deque>
#include <functional>
#include <mutex>
#include <thread>
#include <vector>

#include "mwal/status.h"
#include "mwal/write_batch.h"

namespace mwal {

class WriteCoalescer {
 public:
  struct Request {
    WriteBatch* batch;
    Status status;
    bool done = false;
    std::condition_variable cv;

    Request() : batch(nullptr) {}
    Request(const Request&) = delete;
    Request& operator=(const Request&) = delete;
  };

  explicit WriteCoalescer(
      size_t max_queue_depth,
      std::chrono::milliseconds max_flush_interval = std::chrono::milliseconds(0));
  ~WriteCoalescer();

  WriteCoalescer(const WriteCoalescer&) = delete;
  WriteCoalescer& operator=(const WriteCoalescer&) = delete;

  // Producer: enqueue batch, block until written. Thread-safe for multiple
  // producers. Blocks when queue is at max depth (back-pressure).
  Status Submit(WriteBatch* batch);

  // Start the background writer thread. The callback receives a vector of
  // batches to write as a single merged record (merge + AddRecord + rotate).
  void Start(std::function<Status(std::vector<WriteBatch*>& batches)> write_fn);

  // Drain remaining items and stop the writer thread.
  void Stop();

 private:
  void WriterLoop();

  std::mutex mu_;
  std::condition_variable producer_cv_;
  std::condition_variable writer_cv_;
  std::deque<Request*> queue_;
  size_t max_depth_;
  std::chrono::milliseconds max_flush_interval_;
  bool stopped_ = false;
  std::thread writer_thread_;
  std::function<Status(std::vector<WriteBatch*>&)> write_fn_;
};

}  // namespace mwal
