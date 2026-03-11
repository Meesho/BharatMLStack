# High-Concurrency Dual-Segment Architecture for `pkg/hnswlib`

> Scope: Everything lives inside `pkg/hnswlib/`. Pure C++17. No Go, no external
> orchestration. WAL is handled externally and is out of scope.

## Table of Contents

- [1. What and Why](#1-what-and-why)
  - [1.5 C++17 Upgrade Rationale](#15-c-standard-upgrading-from-c11-to-c17)
- [2. How — Implementation Phases](#2-how--implementation-phases)
  - [Phase 1: Primitives — Filtered Search and Data Extraction](#phase-1-primitives--filtered-search-and-data-extraction)
  - [Phase 2: The Collection Class](#phase-2-the-collection-class)
  - [Phase 3: The Rebuild Engine](#phase-3-the-rebuild-engine)
  - [Phase 4: Collection Manager (Top-Level API)](#phase-4-collection-manager-top-level-api)
- [3. Test Module](#3-test-module)
- [4. Benchmarking Suite](#4-benchmarking-suite)
- [5. Files Summary](#6-files-summary)

---

## 1. What and Why

### 1.1 The Problem

The current `HNSWIndexImpl` wraps a single `hnswlib::HierarchicalNSW` per index. This is
crash-prone under concurrent read/write workloads:

1. **`resizeIndex` moves memory under live readers.**
   `resizeIndex` (`hnswalg.h:633`) calls `realloc` on `data_level0_memory_` and `linkLists_`.
   Concurrent `searchBaseLayerST` threads traverse those exact pointers via `_mm_prefetch` —
   the CPU dereferences addresses that `realloc` has already freed, causing segfaults.

2. **Search is lock-free but writes mutate shared memory.**
   `addPoint` acquires `label_lookup_lock` and per-element `link_list_locks_`, but
   `searchBaseLayerST` (`hnswalg.h:310`) acquires **no locks** — it reads
   `data_level0_memory_` and `linkLists_` raw. Safe only if the underlying memory never
   moves, which `resizeIndex` violates.

3. **In-place `updatePoint` breaks routing geometry.**
   When a vector is updated to a distant position in metric space, the greedy routing
   through its old neighborhood becomes sub-optimal, progressively degrading recall.

### 1.2 The Solution

Treat each `hnswlib::HierarchicalNSW` as either **append-only** (small, mutex-protected
buffer) or **immutable** (large, lock-free sealed segment). A `Collection` class manages
exactly two of these, plus a concurrent tombstone set:

```
┌──────────────────────────────────────────────────┐
│              Collection "products"                │
│                                                   │
│  ┌────────────────────────┐                       │
│  │   Sealed Segment       │ ← large, immutable    │
│  │   (shared_ptr)         │   zero-lock reads      │
│  │   95% of data          │                       │
│  └────────────────────────┘                       │
│                                                   │
│  ┌────────────────────────┐                       │
│  │   Appendable Buffer    │ ← small, RW-locked    │
│  │   (shared_ptr)         │   microsecond writes   │
│  │   5% of data           │                       │
│  └────────────────────────┘                       │
│                                                   │
│  ┌────────────────────────┐                       │
│  │   Tombstone Bitset     │ ← concurrent hashset  │
│  └────────────────────────┘                       │
│                                                   │
│  ┌────────────────────────┐                       │
│  │   Metrics (atomics)    │ ← counters, gauges    │
│  └────────────────────────┘                       │
└──────────────────────────────────────────────────┘
```

Reads scatter-gather across both segments, filter tombstones via hnswlib's native
`BaseFilterFunctor`, and merge Top-K. A background **Rebuild Engine** periodically compacts
everything into a fresh sealed segment.

### 1.3 Why This Design

| Property | RCU/Lock-Free (Plan 1) | Multi-Segment (Plan 2) | **Dual-Segment (Plan 3)** |
|---|---|---|---|
| Modifies hnswlib internals | Yes (deep fork) | No | **No** |
| C++ complexity | Very high (EBR, arenas) | Medium | **Medium** |
| Query overhead | Lowest (1 index) | O(N×k) segments | **O(k) — always 2 segments** |
| Bug surface area | Enormous | Large | **Small** |

### 1.4 Guarantees

- **No segfaults:** `resizeIndex` is never called on a segment being queried. The buffer is
  pre-allocated to its max capacity. The sealed segment is never mutated.
- **SIMD prefetch preserved:** Each segment's `data_level0_memory_` is contiguous and stable,
  so `_mm_prefetch` in `searchBaseLayerST` (`hnswalg.h:370-384`) always hits valid memory.
- **Bounded query overhead:** Always 2 segments (briefly 3 during rebuild). Merge is O(k).
- **hnswlib is untouched:** All `.h` files under `pkg/hnswlib/` (hnswalg.h, hnswlib.h,
  space_l2.h, space_ip.h, etc.) remain unmodified.

### 1.5 C++ Standard: Upgrading from C++11 to C++17

The existing codebase compiles with `-std=c++11`. This plan requires upgrading to **C++17**.

**Why C++17 is required:**

| Feature needed | Standard | Used for |
|---|---|---|
| `std::shared_mutex` | C++17 | Read-write locks on segments, tombstones, buffer — the core concurrency model |
| `std::shared_lock<>` | C++17 | RAII shared (reader) lock wrapper |
| `std::optional` | C++17 | Cleaner error returns from internal helpers |
| `std::string_view` | C++17 | Zero-copy string parameters (space names, etc.) |
| `[[nodiscard]]` | C++17 | Compile-time enforcement that search/add return values are checked |
| Structured bindings | C++17 | Cleaner iteration over `label_lookup_` and result pairs |

**Why this is safe:**

- **hnswlib compatibility:** hnswlib headers are pure C++11. C++17 is a strict superset —
  all existing code compiles identically without any modifications.
- **Compiler support:** GCC 7+, Clang 5+, MSVC 2017+ all fully support C++17. Any modern
  build environment (2020+) has this.
- **Migration effort:** Single flag change: `-std=c++11` → `-std=c++17`. Zero code changes
  to existing files.

**The alternative (staying on C++11)** would require implementing a custom `SharedMutex` class
(~40 lines of hand-rolled concurrency code) — a fragile, unnecessary reimplementation of a
standard library primitive. This is strictly worse in every dimension: correctness risk,
maintenance burden, and readability.

### 1.6 What Changes

| File | Action | Description |
|---|---|---|
| `hnswalg.h`, `hnswlib.h`, `space_*.h`, etc. | **No change** | hnswlib core stays untouched |
| `hnsw_wrapper.h` | **Modify** | Add filtered search, data extraction, Collection API |
| `hnsw_wrapper.cpp` | **Modify** | Implement TombstoneFilter, data extraction functions |
| `collection.h` | **New** | Collection class (segments, tombstones, scatter-gather) |
| `collection.cpp` | **New** | Collection implementation |
| `rebuilder.h` | **New** | Rebuild engine (thread pool, buffer rotation, compaction) |
| `rebuilder.cpp` | **New** | Rebuild engine implementation |
| `collection_test.cpp` | **New** | Unit + concurrency tests |
| `benchmark.cpp` | **New** | Benchmarking suite |
| `Makefile` (or CMakeLists) | **New/Modify** | Build targets for lib, tests, benchmarks |

---

## 2. How — Implementation Phases

### Phase 1: Primitives — Filtered Search and Data Extraction

**Goal:** Add C++ functions that the Collection class will use internally. These also remain
available as extern-C for any FFI caller.

#### Step 1.1: Enable SIMD in the Build

hnswlib has `#ifdef USE_SSE` and `#ifdef USE_AVX` guards in `searchBaseLayerST`, `space_l2.h`,
and `space_ip.h`. The build must define these:

```
CXXFLAGS += -std=c++17 -O2 -DUSE_SSE -DUSE_AVX -msse4.2 -mavx
```

**`-std=c++17`** is required for `std::shared_mutex`, `std::shared_lock`, `std::optional`,
`std::string_view`, structured bindings, and `[[nodiscard]]` — all used extensively in the
Collection and Rebuilder classes (see §1.5 for full rationale).

Without the SIMD flags the scalar fallback runs — potentially 4-8x slower distance
computation and no `_mm_prefetch` in the search hot-loop.

#### Step 1.2: Tombstone Filter Functor

**File:** `hnsw_wrapper.cpp`

```cpp
#include <unordered_set>

class TombstoneFilter : public hnswlib::BaseFilterFunctor {
    const std::unordered_set<hnswlib::labeltype>& blocked_;
public:
    explicit TombstoneFilter(const std::unordered_set<hnswlib::labeltype>& blocked)
        : blocked_(blocked) {}

    bool operator()(hnswlib::labeltype id) override {
        return blocked_.find(id) == blocked_.end();
    }
};
```

**Why:** `searchBaseLayerST` (`hnswalg.h:407`) already evaluates `isIdAllowed` — tombstoned
nodes are excluded from the result queue but their **edges are still traversed** for greedy
routing. This gives us "connected tombstones" for free without touching hnswlib.

When the tombstone set is empty, `searchKnn` is called without a filter so hnswlib takes the
faster `bare_bone_search` path (`hnswalg.h:1306`).

#### Step 1.3: Filtered Search Function

**File:** `hnsw_wrapper.h` — add to the `extern "C"` block:

```cpp
int hnsw_search_knn_filtered(
    HNSWIndex index,
    const float* query_data,
    int k,
    unsigned long long* labels,
    float* distances,
    const unsigned long long* tombstone_ids,
    int tombstone_count
);
```

**File:** `hnsw_wrapper.cpp` — implementation:

Same flow as the existing `hnsw_search_knn` (including cosine normalization), but:
1. Build an `std::unordered_set<hnswlib::labeltype>` from the `tombstone_ids` array.
2. Construct a `TombstoneFilter` wrapping that set.
3. Call `impl->alg_hnsw->searchKnn(query_vector.data(), k, &filter)`.
4. When `tombstone_count == 0`, call `searchKnn` without a filter.

#### Step 1.4: Vector Data Extraction

For the rebuild engine to copy live vectors out of an old segment:

**File:** `hnsw_wrapper.h`:

```cpp
// Copy the vector data for a given label into out_data.
// Returns 0 on success, -1 on error, -2 if label not found.
int hnsw_get_data_by_label(HNSWIndex index, unsigned long long label, float* out_data);

// Write all labels currently in the index into out_labels.
// Returns the number of labels written, or -1 on error.
int hnsw_get_all_labels(HNSWIndex index, unsigned long long* out_labels, int max_count);
```

**File:** `hnsw_wrapper.cpp`:

- `hnsw_get_data_by_label`: Lock `label_lookup_lock`, find `internal_id`, unlock, then
  `memcpy` from `getDataByInternalId(internal_id)` — copying `dimension * sizeof(float)`
  bytes into `out_data`.
- `hnsw_get_all_labels`: Lock `label_lookup_lock`, iterate `label_lookup_`, write each
  label to `out_labels`, unlock, return count.

---

### Phase 2: The Collection Class

**Goal:** A C++ class that owns two hnswlib segments, a tombstone set, and implements
thread-safe add/delete/update/search with scatter-gather.

#### Step 2.1: Core Data Structures

**New file:** `collection.h`

```cpp
#pragma once

#include "hnswlib.h"
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

struct CollectionConfig {
    std::string space_name;
    int dimension;
    int M;
    int ef_construction;
    int ef_search;
    int64_t initial_sealed_capacity;  // max_elements for the first sealed segment
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
    HNSWIndex index;           // opaque pointer to HNSWIndexImpl
    std::atomic<int64_t> count{0};
    bool sealed;               // true = immutable, false = appendable

    ~Segment();                // calls hnsw_delete_index
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
    int search(const float* query, int k,
               unsigned long long* out_labels, float* out_distances);

    // --- Stats ---
    CollectionMetrics& metrics();
    double degradationRatio() const;
    double bufferFillRatio() const;
    bool needsRebuild() const;

    // --- Rebuild interface (called by Rebuilder) ---
    // Rotates the current buffer into a frozen state, creates a fresh buffer.
    // Returns shared_ptrs to the old sealed + frozen buffer for the rebuild
    // worker to read from.
    struct RebuildSnapshot {
        std::shared_ptr<Segment> old_sealed;
        std::shared_ptr<Segment> frozen_buffer;
        std::unordered_set<unsigned long long> tombstone_snapshot;
    };
    RebuildSnapshot prepareRebuild();
    void installRebuiltSegment(std::shared_ptr<Segment> new_sealed);

private:
    CollectionConfig config_;
    CollectionMetrics metrics_;

    // Segments are held via shared_ptr for safe handoff during rebuild.
    // Active search threads hold a local copy of the shared_ptr, preventing
    // the destructor from running until they finish — the C++ equivalent of
    // epoch-based reclamation.
    std::shared_ptr<Segment> sealed_;
    std::shared_ptr<Segment> buffer_;
    std::shared_ptr<Segment> frozen_buffer_;  // non-null only during rebuild

    mutable std::shared_mutex segments_mu_;   // protects pointer swaps
    mutable std::shared_mutex buffer_rw_mu_;  // protects writes to buffer

    // Tombstone set: concurrent reads are lock-free via shared_lock,
    // writes take unique_lock.
    std::unordered_set<unsigned long long> tombstones_;
    mutable std::shared_mutex tombstones_mu_;

    int64_t max_buffer_size_;

    // Internal helpers
    int64_t computeBufferCapacity(int64_t sealed_size) const;
    std::shared_ptr<Segment> createSegment(int64_t capacity, bool sealed);
    std::vector<unsigned long long> snapshotTombstoneIDs() const;

    void searchSegment(
        std::shared_ptr<Segment> seg,
        const float* query, int k,
        const std::vector<unsigned long long>& tombstone_ids,
        std::vector<std::pair<float, unsigned long long>>& results);
};
```

**Key design choices:**

- **`std::shared_ptr<Segment>`:** When the rebuild swaps in a new sealed segment, any
  in-flight search thread still holds a `shared_ptr` copy to the old one. The old
  segment's `HNSWIndex` memory stays alive until the last reader finishes — exactly
  the reference-counted safe-reclamation from Plan 3. No manual grace periods.

- **`std::shared_mutex` (C++17):** Read-heavy access pattern. Searches take
  `shared_lock`, writes take `unique_lock`. The sealed segment needs no lock at all.

- **Tombstones as `std::unordered_set` with `shared_mutex`:** Simple and correct.
  Reads (every search) take `shared_lock`. Writes (deletes) take `unique_lock`.
  Under high read concurrency this is near-zero contention.

#### Step 2.2: Adaptive Buffer Sizing

```cpp
int64_t Collection::computeBufferCapacity(int64_t sealed_size) const {
    int64_t adaptive = static_cast<int64_t>(sealed_size * 0.05);
    const int64_t floor = 5000, ceiling = 50000;
    return std::max(floor, std::min(adaptive, ceiling));
}
```

**Why:** A fixed 20K threshold is too large for small collections (20% overhead) and too small
for large ones (rebuilds too frequently). 5% of sealed size with floor/ceiling adapts to any
scale.

#### Step 2.3: The Write Path — `addPoint`

```cpp
int Collection::addPoint(const float* data, unsigned long long label) {
    // Backpressure: reject if buffer nearly full and rebuild already pending
    if (bufferFillRatio() > 0.95 && metrics_.rebuild_in_progress.load()) {
        return -3;  // BACKPRESSURE — caller should retry
    }

    std::unique_lock<std::shared_mutex> lock(buffer_rw_mu_);
    auto buf = buffer_;  // local shared_ptr copy
    lock.unlock();       // release segments_mu_ early, only need buffer_rw_mu_

    // Actual write under buffer write lock
    std::unique_lock<std::shared_mutex> write_lock(buffer_rw_mu_);
    int rc = hnsw_add_point(buf->index, data, label);
    if (rc == 0) {
        buf->count.fetch_add(1);
        metrics_.buffer_count.fetch_add(1);
        metrics_.total_vectors.fetch_add(1);
    }
    return rc;
}
```

**Backpressure** (borrowed from Plan 1): If the buffer is >95% full and a rebuild is already
running, return a specific error code. The caller can retry after a short delay. Without this,
`addPoint` would throw when `cur_element_count >= max_elements_`.

#### Step 2.4: The Delete and Update Paths

```cpp
int Collection::deletePoint(unsigned long long label) {
    {
        std::unique_lock<std::shared_mutex> lock(tombstones_mu_);
        tombstones_.insert(label);
    }
    metrics_.tombstone_count.fetch_add(1);
    return 0;
}

int Collection::updatePoint(const float* data, unsigned long long label) {
    deletePoint(label);             // tombstone the old position
    return addPoint(data, label);   // insert fresh into buffer
}
```

**Why delete-then-insert:** hnswlib's native `updatePoint` does in-place vector replacement.
Under concurrent search this mutates data that readers are traversing. The tombstone approach
keeps old node edges intact for routing while logically removing it.

#### Step 2.5: The Read Path — Scatter-Gather Search

```cpp
int Collection::search(
    const float* query, int k,
    unsigned long long* out_labels, float* out_distances)
{
    // 1. Grab shared_ptr copies of all active segments (lock-free after this)
    std::shared_ptr<Segment> s_sealed, s_buffer, s_frozen;
    {
        std::shared_lock<std::shared_mutex> lock(segments_mu_);
        s_sealed = sealed_;
        s_buffer = buffer_;
        s_frozen = frozen_buffer_;
    }

    // 2. Snapshot tombstone IDs for the filter
    auto tombstone_ids = snapshotTombstoneIDs();

    // 3. Search each non-empty segment
    //    sealed = lock-free (immutable)
    //    frozen = lock-free (immutable, only exists during rebuild)
    //    buffer = shared_lock on buffer_rw_mu_
    using Result = std::pair<float, unsigned long long>;
    std::vector<Result> sealed_res, frozen_res, buffer_res;

    if (s_sealed && s_sealed->count.load() > 0) {
        searchSegment(s_sealed, query, k, tombstone_ids, sealed_res);
    }

    if (s_frozen && s_frozen->count.load() > 0) {
        searchSegment(s_frozen, query, k, tombstone_ids, frozen_res);
    }

    if (s_buffer && s_buffer->count.load() > 0) {
        std::shared_lock<std::shared_mutex> lock(buffer_rw_mu_);
        searchSegment(s_buffer, query, k, tombstone_ids, buffer_res);
    }

    // 4. N-way merge (N is 2 or 3, so simple pointer-chase merge in O(k))
    //    Each segment's results are sorted by distance ascending.
    std::vector<Result> merged;
    merged.reserve(sealed_res.size() + frozen_res.size() + buffer_res.size());
    merged.insert(merged.end(), sealed_res.begin(), sealed_res.end());
    merged.insert(merged.end(), frozen_res.begin(), frozen_res.end());
    merged.insert(merged.end(), buffer_res.begin(), buffer_res.end());
    std::sort(merged.begin(), merged.end());  // sort by distance (first element)

    int count = std::min(static_cast<int>(merged.size()), k);
    for (int i = 0; i < count; i++) {
        out_distances[i] = merged[i].first;
        out_labels[i]    = merged[i].second;
    }
    return count;
}
```

**`searchSegment` calls `hnsw_search_knn_filtered`** internally, passing the tombstone IDs.
hnswlib's native filter excludes tombstoned nodes from results while still traversing their
edges.

**Tiered ef optimization:** Before searching the buffer, set a lower `ef`:
```cpp
int buffer_ef = std::min(config_.ef_search, (int)(s_buffer->count.load() / 2));
hnsw_set_ef(s_buffer->index, buffer_ef);
```
The buffer is small — using the full `ef` wastes distance computations when most neighbors
are reachable in a few hops.

**Skip-empty fast path:** If the buffer count is 0, it is skipped entirely — no lock
acquisition, no search call, no merge entries.

#### Step 2.6: Efficient N-Way Merge (Optimization Detail)

For production, replace the sort-based merge with a pointer-chase merge since inputs are
already sorted:

```cpp
static std::vector<Result> mergeKSorted(
    const std::vector<std::vector<Result>*>& lists, int k)
{
    // With 2-3 lists of size k, a simple index-based merge is O(k).
    std::vector<int> idx(lists.size(), 0);
    std::vector<Result> out;
    out.reserve(k);

    while ((int)out.size() < k) {
        int best = -1;
        float best_dist = std::numeric_limits<float>::max();
        for (int s = 0; s < (int)lists.size(); s++) {
            if (idx[s] < (int)lists[s]->size() &&
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
```

---

### Phase 3: The Rebuild Engine

**Goal:** A global thread pool that compacts collections in the background without blocking
reads or writes.

#### Step 3.1: Rebuilder Class

**New file:** `rebuilder.h`

```cpp
#pragma once

#include "collection.h"
#include <thread>
#include <queue>
#include <mutex>
#include <condition_variable>
#include <functional>
#include <atomic>

enum class RebuildPriority {
    NORMAL,  // degradation threshold hit (20% tombstones)
    URGENT   // buffer > 90% full
};

struct RebuildTask {
    Collection* collection;
    RebuildPriority priority;

    bool operator<(const RebuildTask& other) const {
        return priority < other.priority;  // URGENT > NORMAL
    }
};

class Rebuilder {
public:
    explicit Rebuilder(int num_workers);
    ~Rebuilder();

    void submit(Collection* collection, RebuildPriority priority);
    void stop();  // drains in-progress work, then joins all threads

private:
    void worker();
    void executeRebuild(const RebuildTask& task);

    std::priority_queue<RebuildTask> queue_;
    std::mutex queue_mu_;
    std::condition_variable queue_cv_;
    std::vector<std::thread> workers_;
    std::atomic<bool> stopped_{false};
};
```

**Why a global pool:** If 20 collections each spawn their own rebuild thread, a burst of
rebuilds can starve search threads of CPU. A fixed pool (e.g. `std::thread::hardware_concurrency() / 4` workers, min 2) caps rebuild concurrency.

**Why a priority queue:** A buffer at 95% capacity is urgent — if it fills before the rebuild
completes, writes start returning BACKPRESSURE errors. Normal degradation-triggered rebuilds
can wait.

#### Step 3.2: Rebuild Trigger Logic

Inside `Collection::addPoint` and `Collection::deletePoint`, after the operation:

```cpp
void Collection::checkRebuildTrigger(Rebuilder& rebuilder) {
    if (metrics_.rebuild_in_progress.load()) return;

    if (bufferFillRatio() >= 0.90) {
        metrics_.rebuild_in_progress.store(true);
        rebuilder.submit(this, RebuildPriority::URGENT);
    } else if (degradationRatio() >= 0.20) {
        metrics_.rebuild_in_progress.store(true);
        rebuilder.submit(this, RebuildPriority::NORMAL);
    }
}
```

Thresholds:
- **Buffer 90% full** → urgent rebuild
- **Tombstone ratio >= 20%** → normal rebuild

#### Step 3.3: Buffer Rotation (Temporary Third Segment)

```cpp
Collection::RebuildSnapshot Collection::prepareRebuild() {
    std::unique_lock<std::shared_mutex> seg_lock(segments_mu_);
    std::unique_lock<std::shared_mutex> buf_lock(buffer_rw_mu_);

    RebuildSnapshot snap;
    snap.old_sealed = sealed_;
    snap.frozen_buffer = buffer_;

    // Snapshot and clear tombstones
    {
        std::unique_lock<std::shared_mutex> ts_lock(tombstones_mu_);
        snap.tombstone_snapshot = tombstones_;
        tombstones_.clear();
    }
    metrics_.tombstone_count.store(0);

    // The frozen buffer is now read-only — search threads can access it lock-free
    frozen_buffer_ = snap.frozen_buffer;

    // Create a fresh appendable buffer
    int64_t new_cap = computeBufferCapacity(sealed_->count.load());
    buffer_ = createSegment(new_cap, /*sealed=*/false);
    metrics_.buffer_count.store(0);

    return snap;
}
```

During the rebuild, the search path hits 3 segments: `sealed_` + `frozen_buffer_` +
`buffer_`. The `frozen_buffer_` is immutable, so it needs no lock.

#### Step 3.4: Out-of-Place Graph Construction

```cpp
void Rebuilder::executeRebuild(const RebuildTask& task) {
    Collection* c = task.collection;
    auto start = std::chrono::steady_clock::now();

    // 1. Rotate buffer, snapshot tombstones
    auto snap = c->prepareRebuild();

    // 2. Collect all live vectors from old_sealed + frozen_buffer
    std::vector<std::pair<unsigned long long, std::vector<float>>> live_vectors;
    collectLive(snap.old_sealed, snap.tombstone_snapshot, live_vectors, c->config().dimension);
    collectLive(snap.frozen_buffer, snap.tombstone_snapshot, live_vectors, c->config().dimension);

    // 3. Build new HNSW index
    int64_t new_capacity = std::max(
        (int64_t)(live_vectors.size() * 1.3),  // 30% headroom
        (int64_t)5000
    );
    auto new_seg = c->createSegment(new_capacity, /*sealed=*/true);

    // Parallel insertion — hnswlib's addPoint is thread-safe via label_op_locks_
    int num_threads = std::max(1, (int)std::thread::hardware_concurrency() / 4);
    parallelInsert(new_seg->index, live_vectors, num_threads);
    new_seg->count.store(live_vectors.size());

    // 4. Pre-warm: run synthetic searches to load upper graph layers into CPU cache
    prewarm(new_seg->index, live_vectors, c->config().dimension, /*num_queries=*/20, /*k=*/10);

    // 5. Atomic install
    c->installRebuiltSegment(std::make_shared<Segment>(*new_seg));

    // 6. Metrics
    auto elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::steady_clock::now() - start);
    c->metrics().last_rebuild_ms.store(elapsed.count());
    c->metrics().rebuild_count.fetch_add(1);
    c->metrics().rebuild_in_progress.store(false);

    // old_sealed and frozen_buffer shared_ptrs go out of scope here.
    // If any search thread still holds a copy, the Segment destructor is
    // deferred until that thread finishes — zero segfaults, guaranteed.
}
```

**`collectLive` helper:** Uses `hnsw_get_all_labels` + `hnsw_get_data_by_label` to iterate
a segment, skipping any label in the tombstone snapshot.

**`parallelInsert` helper:** Splits the `live_vectors` into chunks, spawns N `std::thread`s,
each calls `hnsw_add_point` on its chunk. hnswlib's per-label mutexes handle the
synchronization internally.

**Pre-warming:** Run ~20 random searches on the new segment before installing it. This pulls
the entry-point node and upper-level links into the CPU L1/L2 cache, preventing a latency
spike on the first real queries after the swap.

#### Step 3.5: Atomic Install

```cpp
void Collection::installRebuiltSegment(std::shared_ptr<Segment> new_sealed) {
    std::unique_lock<std::shared_mutex> lock(segments_mu_);

    sealed_ = new_sealed;
    frozen_buffer_.reset();  // clear the third segment

    metrics_.sealed_count.store(new_sealed->count.load());
    max_buffer_size_ = computeBufferCapacity(new_sealed->count.load());
}
```

After `installRebuiltSegment`, the collection is back to exactly 2 segments. The old
`sealed_` and `frozen_buffer_` `shared_ptr`s are released — if no search thread holds a
copy, the `Segment` destructor fires and frees the hnswlib index. If a search thread does
hold a copy, the destructor is deferred until that thread's local `shared_ptr` goes out of
scope.

#### Step 3.6: Graceful Shutdown

```cpp
Rebuilder::~Rebuilder() { stop(); }

void Rebuilder::stop() {
    stopped_.store(true);
    queue_cv_.notify_all();
    for (auto& t : workers_) {
        if (t.joinable()) t.join();
    }
}
```

Called before process exit. In-progress rebuilds run to completion.

---

### Phase 4: Collection Manager (Top-Level API)

**Goal:** A thin management layer that the database application (your main C++ DB) uses to
create/destroy/lookup collections and route operations. Exposed as `extern "C"` functions
in `hnsw_wrapper.h` for FFI compatibility.

#### Step 4.1: Extern-C Collection API

Add to `hnsw_wrapper.h`:

```cpp
// --- Collection API ---
typedef void* HNSWCollection;

HNSWCollection collection_create(
    const char* name,
    const char* space_name,
    int dim,
    int M,
    int ef_construction,
    int ef_search,
    long long initial_sealed_capacity
);

void collection_destroy(HNSWCollection col);

int collection_add_point(HNSWCollection col, const float* data, unsigned long long label);
int collection_delete_point(HNSWCollection col, unsigned long long label);
int collection_update_point(HNSWCollection col, const float* data, unsigned long long label);

int collection_search(
    HNSWCollection col,
    const float* query,
    int k,
    unsigned long long* out_labels,
    float* out_distances
);

// Returns a JSON string with metrics. Caller must free() the returned pointer.
char* collection_get_stats(HNSWCollection col);

// --- Global Rebuilder ---
void rebuilder_init(int num_workers);
void rebuilder_stop();
```

These `extern "C"` functions are thin wrappers that cast the opaque pointer and delegate
to the `Collection` class. The existing `hnsw_*` functions (single-index API) remain
unchanged for backward compatibility.

#### Step 4.2: Stats Structure

The `collection_get_stats` function returns a JSON-encoded string with:

```json
{
  "sealed_count": 95000,
  "buffer_count": 3200,
  "tombstone_count": 1800,
  "degradation_pct": 1.86,
  "buffer_fill_pct": 64.0,
  "rebuild_count": 7,
  "last_rebuild_ms": 2340,
  "is_rebuilding": false
}
```

The caller (your main DB) can expose this via HTTP/gRPC health endpoints, dashboards, etc.

---

## 3. Test Module

**File:** `collection_test.cpp`

Build with any test framework available in your environment (Google Test, Catch2, or plain
`assert` + `main`). The tests below assume Google Test syntax but the logic is framework-agnostic.

### 3.1 Unit Tests — Basic CRUD

| Test | What it does | Assertion |
|---|---|---|
| `AddAndSearch` | Add 100 random 128-d vectors, search for each | Each vector's nearest neighbor is itself (distance ≈ 0) |
| `DeletePoint` | Add 100 vectors, delete ID=50, search for ID=50's vector | ID=50 not in results; its geometric neighbors still found |
| `UpdatePoint` | Add ID=1 at pos A, update to pos B, search near B | ID=1 found near B, not near A |
| `DeleteThenReinsert` | Add, delete, re-add same ID with different vector | Latest vector is the one returned |

### 3.2 Unit Tests — Tombstone Behavior

| Test | What it does | Assertion |
|---|---|---|
| `TombstonePreservesRouting` | Delete 100 high-degree nodes from 1000-vector index, search remaining | recall@10 >= 0.95 |
| `ThresholdNotTriggeredBelow20Pct` | Delete 1999 out of 10000 | `needsRebuild() == false` |
| `ThresholdTriggeredAt20Pct` | Delete 2000 out of 10000 | `needsRebuild() == true` |

### 3.3 Unit Tests — Buffer Management

| Test | What it does | Assertion |
|---|---|---|
| `BufferRotation` | Fill buffer past 90%, verify rebuild triggers, add more during rebuild | New vectors in new buffer; all findable after rebuild completes |
| `AdaptiveBufferSizing` | Call `computeBufferCapacity` with varying sealed sizes | sealed=1K → 5000 (floor); sealed=200K → 10000; sealed=2M → 50000 (ceiling) |
| `Backpressure` | Fill buffer to 96%, set `rebuild_in_progress=true`, call `addPoint` | Returns `-3` (BACKPRESSURE) |

### 3.4 Unit Tests — Scatter-Gather

| Test | What it does | Assertion |
|---|---|---|
| `SearchAcrossBothSegments` | Pre-load sealed with 1000 vectors, add 50 to buffer, search | Results from both segments, correctly merged by distance |
| `SearchEmptyBuffer` | Sealed=1000, buffer=0 | Correct results, no errors (skip-empty fast path) |
| `SearchDuringThreeSegmentWindow` | Trigger rebuild, search mid-rebuild | Results from sealed + frozen + new buffer, all correct |

### 3.5 Unit Tests — Merge

| Test | What it does | Assertion |
|---|---|---|
| `MergeTopK_Normal` | Two sorted lists, k < total | Top-k by distance, correctly interleaved |
| `MergeTopK_OneEmpty` | One list empty | Returns other list up to k |
| `MergeTopK_Duplicates` | Lists with duplicate distances | Stable, includes all up to k |
| `MergeTopK_KExceedsTotal` | k > total available | Returns all available |

### 3.6 Unit Tests — Filtered Search

| Test | What it does | Assertion |
|---|---|---|
| `FilteredSearch_ExcludesTombstones` | Mark IDs {10,20,30} tombstoned, search | None in results |
| `FilteredSearch_EmptyTombstones` | No tombstones, filtered vs unfiltered | Identical results |
| `FilteredSearch_AllTombstoned` | Tombstone every vector | 0 results returned |

### 3.7 Concurrency Stress Tests

| Test | What it does | Duration | Assertion |
|---|---|---|---|
| `ConcurrentReadWrite` | 4 writer threads + 8 reader threads + 1 deleter thread, all hammering the same collection | 10s | No crashes, no ASAN violations, all searches return valid results |
| `RebuildDuringSearch` | Pre-load 10K vectors, trigger rebuild, 16 search threads | 30s | All searches succeed; post-rebuild all non-tombstoned vectors findable |
| `RebuildDuringWrite` | Trigger rebuild, continuously add vectors in parallel | 30s | New vectors in new buffer; all found after rebuild |
| `MultipleCollections` | 5 collections, simultaneous writes, trigger 3 concurrent rebuilds | 30s | Global pool limits concurrent rebuilds; no cross-collection interference |
| `SearchDuringAtomicSwap` | Start a long search, trigger swap mid-flight | 10s | Pre-swap search completes with correct results using old segment memory |

### 3.8 Rebuild Engine Tests

| Test | What it does | Assertion |
|---|---|---|
| `PriorityOrdering` | Submit 3 NORMAL + 1 URGENT tasks | URGENT processed first |
| `ConcurrencyLimit` | 2-worker pool, submit 5 tasks | At most 2 run concurrently (track with atomic counter) |
| `CleanGraph` | 10K vectors, delete 2500 (25%), rebuild | Sealed = 7500, tombstones = 0, recall@10 >= 0.98 |
| `DataIntegrity` | Add known vectors, rebuild, search each | Vectors bit-for-bit identical post-rebuild |
| `GracefulShutdown` | Start rebuild, call `stop()` | In-progress rebuild completes before return; no thread leaks |

### 3.9 Build & Run

```makefile
test: collection_test
    ./collection_test

collection_test: collection_test.cpp collection.cpp rebuilder.cpp hnsw_wrapper.cpp
    $(CXX) -std=c++17 -DUSE_SSE -DUSE_AVX -msse4.2 -mavx \
        -I. -lgtest -lgtest_main -lpthread \
        -fsanitize=address,thread \
        -o $@ $^
```

**`-fsanitize=address,thread`** is critical — it catches use-after-free (the exact bug we're
fixing) and data races that might not manifest as segfaults but could cause silent corruption.

> Note: ASAN and TSAN cannot run simultaneously. Run the test binary twice — once with
> `-fsanitize=address` and once with `-fsanitize=thread`.

---

## 4. Benchmarking Suite

**File:** `benchmark.cpp`

Uses Google Benchmark or a simple timer harness. Reports ops/sec and latency.

### 4.1 Micro-Benchmarks

```
BM_Baseline_Search_1K              // current single HNSWIndexImpl, 1K vectors
BM_Baseline_Search_100K            // current single HNSWIndexImpl, 100K vectors
BM_Baseline_Search_1M              // current single HNSWIndexImpl, 1M vectors
BM_Baseline_AddPoint               // single addPoint latency

BM_Collection_Search_1K            // new Collection, 1K vectors
BM_Collection_Search_100K          // new Collection, 100K vectors
BM_Collection_Search_1M            // new Collection, 1M vectors
BM_Collection_AddPoint             // addPoint through Collection (buffer write lock)

BM_MergeTopK_k10                   // merge overhead, k=10
BM_MergeTopK_k100                  // merge overhead, k=100

BM_FilteredSearch_0pctTombstones   // 0% tombstones (bare_bone_search path)
BM_FilteredSearch_10pctTombstones  // 10% tombstones
BM_FilteredSearch_20pctTombstones  // 20% tombstones (worst case before rebuild)

BM_Rebuild_10K                     // full compaction, 10K vectors
BM_Rebuild_100K                    // full compaction, 100K vectors
BM_Rebuild_ParallelInsert_100K     // parallel rebuild vs single-threaded
```

### 4.2 Concurrent Throughput Benchmark

Measures total ops/sec and tail latency under mixed read/write load:

```
BM_Concurrent_W0_R16               // pure read (16 threads)
BM_Concurrent_W1_R15               // 6% write
BM_Concurrent_W4_R12               // 25% write
BM_Concurrent_W8_R8                // 50% write (stress)
```

Each configuration runs for 30 seconds and reports:
- Total search ops/sec
- Total write ops/sec
- p50, p95, p99 search latency
- p50, p95, p99 write latency

### 4.3 Recall Benchmark

Validates that the dual-segment architecture doesn't degrade search quality:

```
BM_Recall_CleanSingleSegment       // baseline — all data in one index
BM_Recall_CleanDualSegment          // 95% sealed, 5% buffer, 0% tombstones
BM_Recall_10pctTombstoned           // 10% deleted
BM_Recall_19pctTombstoned           // worst case before rebuild trigger
BM_Recall_PostRebuild               // after compaction completes
```

Each scenario:
1. Generates 10K random 128-d vectors.
2. Computes ground truth via brute-force exact k-NN (using `bruteforce.h`).
3. Runs 1000 random queries through the Collection.
4. Reports recall@1, recall@10, recall@100.

### 4.4 Target Metrics

| Metric | Target |
|---|---|
| Search QPS (pure read) | >= 95% of baseline single-index |
| Search QPS (mixed R/W) | >= 80% of pure-read, **0 segfaults** |
| Search p99 latency | < 2x baseline normal; < 5x during rebuild |
| Write throughput | >= 90% of baseline single-index addPoint |
| Rebuild time (100K vectors) | < 30s with parallel insert |
| Recall@10 | >= 0.95 at all times; >= 0.98 post-rebuild |
| Memory overhead | < 1.5x during rebuild (old + new coexist) |
| Segfault / ASAN violations | **0** |

### 4.5 Build & Run

```makefile
benchmark: benchmark.cpp collection.cpp rebuilder.cpp hnsw_wrapper.cpp
    $(CXX) -std=c++17 -O2 -DNDEBUG -DUSE_SSE -DUSE_AVX -msse4.2 -mavx \
        -I. -lbenchmark -lpthread \
        -o $@ $^
    ./benchmark --benchmark_format=console --benchmark_counters_tabular=true
```

> Note: Benchmarks are compiled with `-O2 -DNDEBUG` (no asserts, full optimization) to
> reflect production performance.

---

## 5. Files Summary

```
pkg/hnswlib/
├── hnswlib.h                   # UNCHANGED — hnswlib core
├── hnswalg.h                   # UNCHANGED — hnswlib HNSW algorithm
├── space_l2.h                  # UNCHANGED — L2 distance (SIMD)
├── space_ip.h                  # UNCHANGED — inner product distance (SIMD)
├── visited_list_pool.h         # UNCHANGED — visited list pool
├── stop_condition.h            # UNCHANGED — search stop conditions
├── bruteforce.h                # UNCHANGED — brute-force (used in recall benchmarks)
│
├── hnsw_wrapper.h              # MODIFIED  — add filtered search, data extraction, collection API
├── hnsw_wrapper.cpp            # MODIFIED  — TombstoneFilter, new functions
├── hnsw_wrapper.cc             # EXISTING  — (duplicate, may need consolidation)
│
├── collection.h                # NEW — Collection class declaration
├── collection.cpp              # NEW — Collection implementation
├── rebuilder.h                 # NEW — Rebuild engine declaration
├── rebuilder.cpp               # NEW — Rebuild engine implementation
│
├── collection_test.cpp         # NEW — Unit + concurrency tests
├── benchmark.cpp               # NEW — Benchmarking suite
└── Makefile                    # NEW — Build targets for lib, tests, benchmarks
```
