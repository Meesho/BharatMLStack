#include "collection.h"

#include <cstdio>
#include <cstring>
#include <chrono>


Collection::Collection(const CollectionConfig& config)
    : config_(config)
{
    auto sealed = createSegment(config.initial_sealed_capacity, true);
    sealed_.store(sealed);

    max_buffer_size_ = computeBufferCapacity(config.initial_sealed_capacity);
    auto buf = createSegment(max_buffer_size_, false);
    buffer_ = buf;

    frozen_buffer_.store(std::shared_ptr<Segment>(nullptr));
}


Collection::~Collection() {
    // shared_ptrs clean up automatically
}


std::shared_ptr<Segment> Collection::createSegment(int64_t capacity, bool is_sealed) {
    auto seg = std::make_shared<Segment>();
    seg->sealed = is_sealed;
    seg->index = hnsw_create_index(
        config_.space_name.c_str(),
        config_.dimension,
        static_cast<int>(capacity),
        config_.M,
        config_.ef_construction,
        100 // random_seed
    );
    if (seg->index) {
        hnsw_set_ef(seg->index, config_.ef_search);
    }
    return seg;
}


int64_t Collection::computeBufferCapacity(int64_t sealed_size) const {
    int64_t adaptive = static_cast<int64_t>(sealed_size * 0.05);
    const int64_t floor_val = 5000, ceiling_val = 50000;
    return std::max(floor_val, std::min(adaptive, ceiling_val));
}


// --- Write Path ---

int Collection::addPoint(const float* data, unsigned long long label) {
    if (bufferFillRatio() > 0.95 && metrics_.rebuild_in_progress.load(std::memory_order_relaxed)) {
        return -3; // BACKPRESSURE
    }

    int rc;
    {
        std::unique_lock<std::shared_mutex> write_lock(buffer_mu_);
        rc = hnsw_add_point(buffer_->index, data, label);
        if (rc == 0) {
            buffer_->count.fetch_add(1, std::memory_order_relaxed);
            metrics_.buffer_count.fetch_add(1, std::memory_order_relaxed);
            metrics_.total_vectors.fetch_add(1, std::memory_order_relaxed);
        }
    }
    return rc;
}


int Collection::deletePoint(unsigned long long label) {
    bool inserted;
    {
        std::unique_lock<std::shared_mutex> lock(tombstones_mu_);
        inserted = tombstones_.insert(label).second;
    }
    if (inserted) {
        metrics_.tombstone_count.fetch_add(1, std::memory_order_relaxed);
    }

    {
        std::unique_lock<std::shared_mutex> buf_lock(buffer_mu_);
        int md_rc = hnsw_mark_deleted(buffer_->index, label);
        if (md_rc == 0) {
            buffer_->count.fetch_sub(1, std::memory_order_relaxed);
            metrics_.buffer_count.fetch_sub(1, std::memory_order_relaxed);
            metrics_.total_vectors.fetch_sub(1, std::memory_order_relaxed);
        }
    }
    return 0;
}


int Collection::updatePoint(const float* data, unsigned long long label) {
    deletePoint(label);
    return addPoint(data, label);
}


// --- Read Path ---

std::vector<unsigned long long> Collection::snapshotTombstoneIDs() const {
    std::shared_lock<std::shared_mutex> lock(tombstones_mu_);
    return std::vector<unsigned long long>(tombstones_.begin(), tombstones_.end());
}


void Collection::searchSegment(
    std::shared_ptr<Segment>& seg,
    const float* query, int k,
    const std::vector<unsigned long long>& tombstone_ids,
    std::vector<Result>& results)
{
    std::vector<unsigned long long> labels(k);
    std::vector<float> distances(k);

    int found;
    if (tombstone_ids.empty()) {
        found = hnsw_search_knn(seg->index, query, k, labels.data(), distances.data());
    } else {
        found = hnsw_search_knn_filtered(
            seg->index, query, k, labels.data(), distances.data(),
            tombstone_ids.data(), static_cast<int>(tombstone_ids.size()));
    }

    if (found > 0) {
        results.reserve(found);
        for (int i = 0; i < found; i++) {
            results.emplace_back(distances[i], labels[i]);
        }
    }
}


int Collection::search(
    const float* query, int k,
    unsigned long long* out_labels, float* out_distances)
{
    // 1. Load segment shared_ptrs (reader lock for sealed/frozen, shared_lock for buffer)
    auto s_sealed = sealed_.load();
    auto s_frozen = frozen_buffer_.load();

    std::shared_ptr<Segment> s_buffer;
    {
        std::shared_lock<std::shared_mutex> lock(buffer_mu_);
        s_buffer = buffer_;
    }

    // 2. Snapshot tombstone IDs
    auto tombstone_ids = snapshotTombstoneIDs();

    // 3. Search each non-empty segment
    std::vector<Result> sealed_res, frozen_res, buffer_res;

    if (s_sealed && s_sealed->count.load(std::memory_order_relaxed) > 0) {
        searchSegment(s_sealed, query, k, tombstone_ids, sealed_res);
    }

    if (s_frozen && s_frozen->count.load(std::memory_order_relaxed) > 0) {
        searchSegment(s_frozen, query, k, tombstone_ids, frozen_res);
    }

    if (s_buffer && s_buffer->count.load(std::memory_order_relaxed) > 0) {
        // Tiered ef: use lower ef for the small buffer
        int buffer_ef = std::min(config_.ef_search,
            static_cast<int>(s_buffer->count.load(std::memory_order_relaxed) / 2));
        buffer_ef = std::max(buffer_ef, k);
        hnsw_set_ef(s_buffer->index, buffer_ef);

        // Buffer uses hnswlib mark-delete for visibility; no tombstone filtering needed
        std::vector<unsigned long long> no_tombstones;
        std::shared_lock<std::shared_mutex> lock(buffer_mu_);
        searchSegment(s_buffer, query, k, no_tombstones, buffer_res);
    }

    // 4. N-way merge
    std::vector<std::vector<Result>*> lists;
    if (!sealed_res.empty())  lists.push_back(&sealed_res);
    if (!frozen_res.empty())  lists.push_back(&frozen_res);
    if (!buffer_res.empty())  lists.push_back(&buffer_res);

    auto merged = mergeKSorted(lists, k);

    int count = static_cast<int>(merged.size());
    for (int i = 0; i < count; i++) {
        out_distances[i] = merged[i].first;
        out_labels[i]    = merged[i].second;
    }
    return count;
}


std::vector<Collection::Result> Collection::mergeKSorted(
    const std::vector<std::vector<Result>*>& lists, int k)
{
    std::vector<int> idx(lists.size(), 0);
    std::vector<Result> out;
    out.reserve(k);

    while (static_cast<int>(out.size()) < k) {
        int best = -1;
        float best_dist = std::numeric_limits<float>::max();
        for (int s = 0; s < static_cast<int>(lists.size()); s++) {
            if (idx[s] < static_cast<int>(lists[s]->size()) &&
                (*lists[s])[idx[s]].first < best_dist) {
                best_dist = (*lists[s])[idx[s]].first;
                best = s;
            }
        }
        if (best == -1) break;
        out.push_back((*lists[best])[idx[best]]);
        idx[best]++;
    }
    return out;
}


// --- Stats ---

double Collection::degradationRatio() const {
    int64_t total = metrics_.total_vectors.load(std::memory_order_relaxed);
    int64_t tombstones = metrics_.tombstone_count.load(std::memory_order_relaxed);
    if (total == 0) return 0.0;
    return static_cast<double>(tombstones) / static_cast<double>(total + tombstones);
}


double Collection::bufferFillRatio() const {
    if (max_buffer_size_ <= 0) return 0.0;
    int64_t buf = metrics_.buffer_count.load(std::memory_order_relaxed);
    return static_cast<double>(buf) / static_cast<double>(max_buffer_size_);
}


bool Collection::needsRebuild() const {
    if (metrics_.rebuild_in_progress.load(std::memory_order_relaxed)) return false;
    return bufferFillRatio() >= 0.90 || degradationRatio() >= 0.20;
}


// --- Rebuild Interface ---

Collection::RebuildSnapshot Collection::prepareRebuild() {
    std::unique_lock<std::shared_mutex> seg_lock(segments_mu_);
    std::unique_lock<std::shared_mutex> buf_lock(buffer_mu_);

    RebuildSnapshot snap;
    snap.old_sealed = sealed_.load();
    snap.frozen_buffer = buffer_;

    // Snapshot and clear tombstones
    {
        std::unique_lock<std::shared_mutex> ts_lock(tombstones_mu_);
        snap.tombstone_snapshot = tombstones_;
        tombstones_.clear();
    }
    metrics_.tombstone_count.store(0, std::memory_order_relaxed);

    // Frozen buffer is now read-only; searches can access it via reader lock
    frozen_buffer_.store(snap.frozen_buffer);

    // Create a fresh appendable buffer
    int64_t sealed_count = snap.old_sealed ? snap.old_sealed->count.load(std::memory_order_relaxed) : 0;
    int64_t new_cap = computeBufferCapacity(sealed_count);
    max_buffer_size_ = new_cap;
    buffer_ = createSegment(new_cap, false);
    metrics_.buffer_count.store(0, std::memory_order_relaxed);

    return snap;
}


void Collection::installRebuiltSegment(std::shared_ptr<Segment> new_sealed) {
    std::unique_lock<std::shared_mutex> lock(segments_mu_);

    sealed_.store(new_sealed);
    frozen_buffer_.store(std::shared_ptr<Segment>(nullptr));

    int64_t sealed_count = new_sealed->count.load(std::memory_order_relaxed);
    int64_t buf_count = metrics_.buffer_count.load(std::memory_order_relaxed);
    metrics_.sealed_count.store(sealed_count, std::memory_order_relaxed);
    metrics_.total_vectors.store(sealed_count + buf_count, std::memory_order_relaxed);
    max_buffer_size_ = computeBufferCapacity(sealed_count);
}
