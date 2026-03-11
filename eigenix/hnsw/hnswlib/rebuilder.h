#pragma once

#include "collection.h"

#include <thread>
#include <queue>
#include <mutex>
#include <condition_variable>
#include <functional>
#include <atomic>
#include <vector>
#include <chrono>

enum class RebuildPriority {
    NORMAL,  // degradation threshold hit (20% tombstones)
    URGENT   // buffer > 90% full
};

struct RebuildTask {
    Collection* collection;
    RebuildPriority priority;

    bool operator<(const RebuildTask& other) const {
        return priority < other.priority; // URGENT > NORMAL in priority_queue
    }
};

class Rebuilder {
public:
    explicit Rebuilder(int num_workers);
    ~Rebuilder();

    void submit(Collection* collection, RebuildPriority priority);
    void stop();

private:
    void worker(std::stop_token stoken);
    void executeRebuild(const RebuildTask& task);

    static void collectLive(
        const std::shared_ptr<Segment>& seg,
        const std::unordered_set<unsigned long long>& tombstones,
        std::vector<std::pair<unsigned long long, std::vector<float>>>& out,
        int dimension);

    static void parallelInsert(
        HNSWIndex index,
        const std::vector<std::pair<unsigned long long, std::vector<float>>>& vectors,
        int num_threads);

    static void prewarm(
        HNSWIndex index,
        const std::vector<std::pair<unsigned long long, std::vector<float>>>& vectors,
        int dimension, int num_queries, int k);

    std::priority_queue<RebuildTask> queue_;
    std::mutex queue_mu_;
    std::condition_variable_any queue_cv_;
    std::vector<std::jthread> workers_;
};
