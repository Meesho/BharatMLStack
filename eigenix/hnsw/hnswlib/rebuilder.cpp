#include "rebuilder.h"

#include <algorithm>
#include <cstring>
#include <random>
#include <iostream>

Rebuilder::Rebuilder(int num_workers) {
    int n = std::max(1, num_workers);
    for (int i = 0; i < n; i++) {
        workers_.emplace_back([this](std::stop_token stoken) {
            worker(stoken);
        });
    }
}


Rebuilder::~Rebuilder() {
    stop();
}


void Rebuilder::submit(Collection* collection, RebuildPriority priority) {
    {
        std::lock_guard<std::mutex> lock(queue_mu_);
        queue_.push(RebuildTask{collection, priority});
    }
    queue_cv_.notify_one();
}


void Rebuilder::stop() {
    // Request stop on all jthreads — their stop_tokens become triggered
    for (auto& t : workers_) {
        t.request_stop();
    }
    queue_cv_.notify_all();
    // jthread destructor joins automatically
    workers_.clear();
}


void Rebuilder::worker(std::stop_token stoken) {
    while (!stoken.stop_requested()) {
        RebuildTask task;
        {
            std::unique_lock<std::mutex> lock(queue_mu_);
            queue_cv_.wait(lock, stoken, [this] {
                return !queue_.empty();
            });

            if (stoken.stop_requested() && queue_.empty()) {
                return;
            }
            if (queue_.empty()) continue;

            task = queue_.top();
            queue_.pop();
        }

        try {
            executeRebuild(task);
        } catch (const std::exception& e) {
            std::cerr << "[eigenix] Rebuild failed: " << e.what() << "\n";
            task.collection->metrics().rebuild_in_progress.store(false, std::memory_order_relaxed);
        }
    }

    // Drain remaining tasks before exit
    while (true) {
        RebuildTask task;
        {
            std::lock_guard<std::mutex> lock(queue_mu_);
            if (queue_.empty()) break;
            task = queue_.top();
            queue_.pop();
        }
        try {
            executeRebuild(task);
        } catch (const std::exception& e) {
            std::cerr << "[eigenix] Rebuild failed during drain: " << e.what() << "\n";
            task.collection->metrics().rebuild_in_progress.store(false, std::memory_order_relaxed);
        }
    }
}


void Rebuilder::executeRebuild(const RebuildTask& task) {
    Collection* c = task.collection;
    auto start = std::chrono::steady_clock::now();

    // 1. Rotate buffer, snapshot tombstones
    auto snap = c->prepareRebuild();

    // 2. Collect all live vectors from old_sealed + frozen_buffer
    int dim = c->config().dimension;
    std::vector<std::pair<unsigned long long, std::vector<float>>> live_vectors;

    collectLive(snap.old_sealed, snap.tombstone_snapshot, live_vectors, dim);
    collectLive(snap.frozen_buffer, snap.tombstone_snapshot, live_vectors, dim);

    // 3. Build new HNSW index
    int64_t new_capacity = std::max(
        static_cast<int64_t>(live_vectors.size() * 1.3),
        static_cast<int64_t>(5000)
    );
    auto new_seg = c->createSegment(new_capacity, true);

    // Parallel insertion
    int hw = static_cast<int>(std::thread::hardware_concurrency());
    int num_threads = std::max(1, hw / 4);
    parallelInsert(new_seg->index, live_vectors, num_threads);
    new_seg->count.store(static_cast<int64_t>(live_vectors.size()), std::memory_order_relaxed);

    // 4. Pre-warm: run synthetic searches to load upper graph layers into CPU cache
    if (!live_vectors.empty()) {
        prewarm(new_seg->index, live_vectors, dim, 20, 10);
    }

    // 5. Atomic install
    c->installRebuiltSegment(new_seg);

    // 6. Metrics
    auto elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::steady_clock::now() - start);
    c->metrics().last_rebuild_ms.store(elapsed.count(), std::memory_order_relaxed);
    c->metrics().rebuild_count.fetch_add(1, std::memory_order_relaxed);
    c->metrics().rebuild_in_progress.store(false, std::memory_order_release);
}


void Rebuilder::collectLive(
    const std::shared_ptr<Segment>& seg,
    const std::unordered_set<unsigned long long>& tombstones,
    std::vector<std::pair<unsigned long long, std::vector<float>>>& out,
    int dimension)
{
    if (!seg || !seg->index) return;

    int max_count = hnsw_get_current_count(seg->index);
    if (max_count <= 0) return;

    std::vector<unsigned long long> labels(max_count);
    int label_count = hnsw_get_all_labels(seg->index, labels.data(), max_count);
    if (label_count <= 0) return;

    std::vector<float> vec_buf(dimension);
    for (int i = 0; i < label_count; i++) {
        if (tombstones.count(labels[i])) continue;
        if (hnsw_is_label_deleted(seg->index, labels[i]) == 1) continue;

        int rc = hnsw_get_data_by_label(seg->index, labels[i], vec_buf.data());
        if (rc == 0) {
            out.emplace_back(labels[i],
                std::vector<float>(vec_buf.begin(), vec_buf.end()));
        }
    }
}


void Rebuilder::parallelInsert(
    HNSWIndex index,
    const std::vector<std::pair<unsigned long long, std::vector<float>>>& vectors,
    int num_threads)
{
    if (vectors.empty()) return;

    int n = static_cast<int>(vectors.size());
    num_threads = std::min(num_threads, n);

    if (num_threads <= 1) {
        for (auto& [label, data] : vectors) {
            hnsw_add_point(index, data.data(), label);
        }
        return;
    }

    // hnswlib requires the first point to be inserted single-threaded
    // (initializes entry point)
    hnsw_add_point(index, vectors[0].second.data(), vectors[0].first);

    int remaining = n - 1;
    int chunk_size = (remaining + num_threads - 1) / num_threads;

    std::vector<std::thread> threads;
    threads.reserve(num_threads);

    for (int t = 0; t < num_threads; t++) {
        int start = 1 + t * chunk_size;
        int end = std::min(start + chunk_size, n);
        if (start >= n) break;

        threads.emplace_back([&vectors, index, start, end]() {
            for (int i = start; i < end; i++) {
                hnsw_add_point(index, vectors[i].second.data(), vectors[i].first);
            }
        });
    }

    for (auto& t : threads) {
        t.join();
    }
}


void Rebuilder::prewarm(
    HNSWIndex index,
    const std::vector<std::pair<unsigned long long, std::vector<float>>>& vectors,
    int /*dimension*/, int num_queries, int k)
{
    if (vectors.empty()) return;

    std::mt19937 rng(42);
    std::uniform_int_distribution<size_t> dist(0, vectors.size() - 1);

    std::vector<unsigned long long> labels(k);
    std::vector<float> distances(k);

    int queries = std::min(num_queries, static_cast<int>(vectors.size()));
    for (int i = 0; i < queries; i++) {
        size_t idx = dist(rng);
        hnsw_search_knn(index, vectors[idx].second.data(), k, labels.data(), distances.data());
    }
}
