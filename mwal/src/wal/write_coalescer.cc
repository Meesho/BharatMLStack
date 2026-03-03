// Dedicated async write coalescer implementation for mwal.

#include "wal/write_coalescer.h"

#include <chrono>
#include <cassert>

namespace mwal {

WriteCoalescer::WriteCoalescer(
    size_t max_queue_depth,
    std::chrono::milliseconds max_flush_interval)
    : max_depth_(max_queue_depth), max_flush_interval_(max_flush_interval) {}

WriteCoalescer::~WriteCoalescer() { Stop(); }

Status WriteCoalescer::Submit(WriteBatch* batch) {
  Request req;
  req.batch = batch;

  {
    std::unique_lock<std::mutex> lock(mu_);
    // Back-pressure: block when queue is full
    producer_cv_.wait(lock, [&] {
      return queue_.size() < max_depth_ || stopped_;
    });
    if (stopped_) return Status::Aborted("WriteCoalescer is stopped");
    queue_.push_back(&req);
  }
  writer_cv_.notify_one();

  // Wait for the writer thread to process our request
  {
    std::unique_lock<std::mutex> lock(mu_);
    req.cv.wait(lock, [&] { return req.done; });
  }
  return req.status;
}

void WriteCoalescer::Start(
    std::function<Status(std::vector<WriteBatch*>& batches)> write_fn) {
  write_fn_ = std::move(write_fn);
  stopped_ = false;
  writer_thread_ = std::thread(&WriteCoalescer::WriterLoop, this);
}

void WriteCoalescer::Stop() {
  {
    std::lock_guard<std::mutex> lock(mu_);
    if (stopped_) return;
    stopped_ = true;
  }
  writer_cv_.notify_one();
  producer_cv_.notify_all();
  if (writer_thread_.joinable()) {
    writer_thread_.join();
  }
}

void WriteCoalescer::WriterLoop() {
  while (true) {
    std::vector<Request*> batch;
    {
      std::unique_lock<std::mutex> lock(mu_);
      if (max_flush_interval_.count() > 0) {
        writer_cv_.wait_for(
            lock, max_flush_interval_,
            [&] { return !queue_.empty() || stopped_; });
      } else {
        writer_cv_.wait(lock,
                       [&] { return !queue_.empty() || stopped_; });
      }
      if (stopped_ && queue_.empty()) break;
      batch.assign(queue_.begin(), queue_.end());
      queue_.clear();
    }
    // Timeout may have returned with empty queue; nothing to write
    if (batch.empty()) continue;
    // Release back-pressure: wake blocked producers
    producer_cv_.notify_all();

    std::vector<WriteBatch*> batches;
    batches.reserve(batch.size());
    for (auto* req : batch) batches.push_back(req->batch);

    Status s = write_fn_(batches);

    // Notify all producers in this coalesced batch
    {
      std::lock_guard<std::mutex> lock(mu_);
      for (auto* req : batch) {
        req->status = s;
        req->done = true;
        req->cv.notify_one();
      }
    }
  }

  // Drain any remaining requests on shutdown
  std::vector<Request*> remaining;
  {
    std::lock_guard<std::mutex> lock(mu_);
    remaining.assign(queue_.begin(), queue_.end());
    queue_.clear();
  }

  if (!remaining.empty()) {
    std::vector<WriteBatch*> batches;
    batches.reserve(remaining.size());
    for (auto* req : remaining) batches.push_back(req->batch);

    Status s = write_fn_(batches);

    std::lock_guard<std::mutex> lock(mu_);
    for (auto* req : remaining) {
      req->status = s;
      req->done = true;
      req->cv.notify_one();
    }
  }
}

}  // namespace mwal
