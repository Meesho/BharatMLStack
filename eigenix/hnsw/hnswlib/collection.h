#pragma once

#include "hnsw_wrapper.h"

#include <atomic>
#include <memory>
#include <mutex>
#include <shared_mutex>
#include <unordered_set>
#include <string>
#include <vector>
#include <algorithm>
#include <functional>
#include <iostream>
#include <limits>
#include <cstdint>

struct CollectionConfig {
    std::string name;
    std::string space_name;
    int dimension;
    int M;
    int ef_construction;
    int ef_search;
    int64_t initial_sealed_capacity;
};

struct CollectionMetrics {
    std::atomic<int64_t> total_vectors{0};
    std::atomic<int64_t> tombstone_count{0};
    std::atomic<int64_t> buffer_count{0};
    std::atomic<int64_t> sealed_count{0};
    std::atomic<int64_t> rebuild_count{0};
    std::atomic<int64_t> last_rebuild_ms{0};
    std::atomic<bool>    rebuild_in_progress{false};
};

struct Segment {
    HNSWIndex index{nullptr};
    std::atomic<int64_t> count{0};
    bool sealed{false};

    ~Segment() {
        if (index) {
            hnsw_delete_index(index);
            index = nullptr;
        }
    }

    Segment() = default;
    Segment(const Segment&) = delete;
    Segment& operator=(const Segment&) = delete;
};

// Thin wrapper around std::shared_ptr with atomic load/store semantics.
// Uses the C++11 std::atomic_load/store free functions from <memory>,
// which work across all compilers (GCC 11+, Clang 11+, MSVC).
// std::atomic<std::shared_ptr<T>> (the class specialization) requires GCC 12+.
template <typename T>
class AtomicSharedPtr {
    mutable std::shared_ptr<T> ptr_;
    mutable std::shared_mutex mu_;
public:
    AtomicSharedPtr() = default;
    explicit AtomicSharedPtr(std::shared_ptr<T> p) : ptr_(std::move(p)) {}

    std::shared_ptr<T> load() const {
        std::shared_lock<std::shared_mutex> lock(mu_);
        return ptr_;
    }

    void store(std::shared_ptr<T> desired) {
        std::unique_lock<std::shared_mutex> lock(mu_);
        ptr_ = std::move(desired);
    }
};

class Collection {
public:
    explicit Collection(const CollectionConfig& config);
    ~Collection();

    // --- Write path ---
    int addPoint(const float* data, unsigned long long label);
    int deletePoint(unsigned long long label);
    int updatePoint(const float* data, unsigned long long label);

    // --- Read path ---
    [[nodiscard]]
    int search(const float* query, int k,
               unsigned long long* out_labels, float* out_distances);

    // --- Stats ---
    CollectionMetrics& metrics() { return metrics_; }
    const CollectionConfig& config() const { return config_; }

    double degradationRatio() const;
    double bufferFillRatio() const;
    bool needsRebuild() const;

    // --- Rebuild interface (called by Rebuilder) ---
    struct RebuildSnapshot {
        std::shared_ptr<Segment> old_sealed;
        std::shared_ptr<Segment> frozen_buffer;
        std::unordered_set<unsigned long long> tombstone_snapshot;
    };

    RebuildSnapshot prepareRebuild();
    void installRebuiltSegment(std::shared_ptr<Segment> new_sealed);

    // Segment creation exposed for Rebuilder use
    std::shared_ptr<Segment> createSegment(int64_t capacity, bool is_sealed);

private:
    CollectionConfig config_;
    CollectionMetrics metrics_;

    // Sealed and frozen segments use AtomicSharedPtr for concurrent reads.
    // Search threads load these via shared_lock (reader-side).
    // prepareRebuild/installRebuiltSegment store via unique_lock (writer-side, rare).
    AtomicSharedPtr<Segment> sealed_;
    AtomicSharedPtr<Segment> frozen_buffer_;

    // Mutable buffer needs a shared_mutex because writers mutate it.
    std::shared_ptr<Segment> buffer_;
    mutable std::shared_mutex buffer_mu_;

    // Coordination mutex for prepareRebuild / installRebuiltSegment
    mutable std::shared_mutex segments_mu_;

    // Tombstone set
    std::unordered_set<unsigned long long> tombstones_;
    mutable std::shared_mutex tombstones_mu_;

    int64_t max_buffer_size_;

    // Internal helpers
    int64_t computeBufferCapacity(int64_t sealed_size) const;
    std::vector<unsigned long long> snapshotTombstoneIDs() const;

    using Result = std::pair<float, unsigned long long>;

    void searchSegment(
        std::shared_ptr<Segment>& seg,
        const float* query, int k,
        const std::vector<unsigned long long>& tombstone_ids,
        std::vector<Result>& results);

    static std::vector<Result> mergeKSorted(
        const std::vector<std::vector<Result>*>& lists, int k);
};
