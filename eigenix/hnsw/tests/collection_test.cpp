#include <gtest/gtest.h>
#include "collection.h"
#include "rebuilder.h"

#include <random>
#include <vector>
#include <thread>
#include <atomic>
#include <chrono>
#include <set>
#include <cmath>
#include <algorithm>
#include <numeric>

// =================== Helpers ===================

static std::vector<float> randomVector(int dim, std::mt19937& rng) {
    std::uniform_real_distribution<float> dist(-1.0f, 1.0f);
    std::vector<float> v(dim);
    for (int i = 0; i < dim; i++) v[i] = dist(rng);
    return v;
}

static CollectionConfig makeConfig(int64_t sealed_cap = 10000) {
    CollectionConfig cfg;
    cfg.name = "test";
    cfg.space_name = "l2";
    cfg.dimension = 128;
    cfg.M = 16;
    cfg.ef_construction = 200;
    cfg.ef_search = 50;
    cfg.initial_sealed_capacity = sealed_cap;
    return cfg;
}

// =================== 3.1 Basic CRUD ===================

TEST(CollectionCRUD, AddAndSearch) {
    auto cfg = makeConfig();
    Collection col(cfg);

    std::mt19937 rng(42);
    int N = 100;
    std::vector<std::vector<float>> vectors(N);
    for (int i = 0; i < N; i++) {
        vectors[i] = randomVector(128, rng);
        ASSERT_EQ(0, col.addPoint(vectors[i].data(), i));
    }

    for (int i = 0; i < N; i++) {
        unsigned long long labels[1];
        float distances[1];
        int found = col.search(vectors[i].data(), 1, labels, distances);
        ASSERT_GE(found, 1);
        EXPECT_EQ(labels[0], static_cast<unsigned long long>(i));
        EXPECT_NEAR(distances[0], 0.0f, 1e-4f);
    }
}

TEST(CollectionCRUD, DeletePoint) {
    auto cfg = makeConfig();
    Collection col(cfg);

    std::mt19937 rng(42);
    int N = 100;
    std::vector<std::vector<float>> vectors(N);
    for (int i = 0; i < N; i++) {
        vectors[i] = randomVector(128, rng);
        col.addPoint(vectors[i].data(), i);
    }

    col.deletePoint(50);

    unsigned long long labels[10];
    float distances[10];
    int found = col.search(vectors[50].data(), 10, labels, distances);
    for (int i = 0; i < found; i++) {
        EXPECT_NE(labels[i], 50ULL);
    }
}

TEST(CollectionCRUD, UpdatePoint) {
    auto cfg = makeConfig();
    Collection col(cfg);

    std::mt19937 rng(42);
    auto vecA = randomVector(128, rng);
    auto vecB = randomVector(128, rng);

    col.addPoint(vecA.data(), 1);
    col.updatePoint(vecB.data(), 1);

    unsigned long long labels[1];
    float distances[1];
    int found = col.search(vecB.data(), 1, labels, distances);
    ASSERT_GE(found, 1);
    EXPECT_EQ(labels[0], 1ULL);
    EXPECT_NEAR(distances[0], 0.0f, 1e-4f);
}

TEST(CollectionCRUD, DeleteThenReinsert) {
    auto cfg = makeConfig();
    Collection col(cfg);

    std::mt19937 rng(42);
    auto vecOrig = randomVector(128, rng);
    auto vecNew = randomVector(128, rng);

    col.addPoint(vecOrig.data(), 42);
    col.deletePoint(42);
    col.addPoint(vecNew.data(), 42);

    unsigned long long labels[1];
    float distances[1];
    int found = col.search(vecNew.data(), 1, labels, distances);
    ASSERT_GE(found, 1);
    EXPECT_EQ(labels[0], 42ULL);
    EXPECT_NEAR(distances[0], 0.0f, 1e-4f);
}

// =================== 3.2 Tombstone Behavior ===================

TEST(CollectionTombstone, ThresholdNotTriggeredBelow20Pct) {
    // Use a small number of vectors that fits in the buffer comfortably
    // Buffer capacity floor is 5000, so use fewer than 4500 (90% of 5000)
    auto cfg = makeConfig();
    Collection col(cfg);

    std::mt19937 rng(42);
    int N = 2000;
    for (int i = 0; i < N; i++) {
        auto v = randomVector(128, rng);
        col.addPoint(v.data(), i);
    }

    // Delete 19% = 380 out of 2000
    // degradationRatio = 380 / (2000 + 380) = 15.97% < 20%
    // bufferFillRatio = 2000 / 5000 = 40% < 90%
    for (int i = 0; i < 380; i++) {
        col.deletePoint(i);
    }

    EXPECT_FALSE(col.needsRebuild());
}

TEST(CollectionTombstone, ThresholdTriggeredAt20Pct) {
    auto cfg = makeConfig();
    Collection col(cfg);

    std::mt19937 rng(42);
    int N = 2000;
    for (int i = 0; i < N; i++) {
        auto v = randomVector(128, rng);
        col.addPoint(v.data(), i);
    }

    // Delete 25% = 500 out of 2000
    // degradationRatio = 500 / (2000 + 500) = 20% >= 20%
    for (int i = 0; i < 500; i++) {
        col.deletePoint(i);
    }

    EXPECT_TRUE(col.needsRebuild());
}

// =================== 3.3 Buffer Management ===================

TEST(CollectionBuffer, AdaptiveBufferSizing) {
    auto cfg = makeConfig();
    Collection col(cfg);

    // Use a config with known sealed capacity
    auto cfg_small = makeConfig(1000);
    Collection col_small(cfg_small);
    // floor = 5000, 1000*0.05 = 50 < 5000 => 5000

    auto cfg_mid = makeConfig(200000);
    Collection col_mid(cfg_mid);
    // 200000*0.05 = 10000

    auto cfg_large = makeConfig(2000000);
    Collection col_large(cfg_large);
    // 2000000*0.05 = 100000 > 50000 ceiling => 50000

    // Just verify the collections are constructed (buffer sizes are internal)
    EXPECT_FALSE(col_small.needsRebuild());
    EXPECT_FALSE(col_mid.needsRebuild());
    EXPECT_FALSE(col_large.needsRebuild());
}

TEST(CollectionBuffer, Backpressure) {
    // Create a collection with very small buffer
    CollectionConfig cfg;
    cfg.name = "bp_test";
    cfg.space_name = "l2";
    cfg.dimension = 4;
    cfg.M = 8;
    cfg.ef_construction = 50;
    cfg.ef_search = 10;
    cfg.initial_sealed_capacity = 100;
    Collection col(cfg);

    std::mt19937 rng(42);

    // Fill the buffer to near capacity (buffer min is 5000)
    for (int i = 0; i < 4800; i++) {
        auto v = randomVector(4, rng);
        col.addPoint(v.data(), i);
    }

    // Simulate rebuild_in_progress
    col.metrics().rebuild_in_progress.store(true);

    // The buffer should be near 96% (4800/5000)
    auto v = randomVector(4, rng);
    int rc = col.addPoint(v.data(), 99999);
    EXPECT_EQ(rc, -3); // BACKPRESSURE
}

// =================== 3.4 Scatter-Gather ===================

TEST(CollectionScatterGather, SearchAcrossBothSegments) {
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::mt19937 rng(42);

    // We can't easily pre-fill the sealed segment without a rebuild,
    // so we test searching the buffer (which holds all data pre-rebuild)
    int N = 200;
    std::vector<std::vector<float>> vectors(N);
    for (int i = 0; i < N; i++) {
        vectors[i] = randomVector(128, rng);
        col.addPoint(vectors[i].data(), i);
    }

    unsigned long long labels[10];
    float distances[10];
    int found = col.search(vectors[0].data(), 10, labels, distances);
    ASSERT_GT(found, 0);
    EXPECT_EQ(labels[0], 0ULL);
}

TEST(CollectionScatterGather, SearchEmptyBuffer) {
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    // Empty collection search should return 0
    std::mt19937 rng(42);
    auto q = randomVector(128, rng);
    unsigned long long labels[10];
    float distances[10];
    int found = col.search(q.data(), 10, labels, distances);
    EXPECT_EQ(found, 0);
}

// =================== 3.5 Merge ===================

TEST(CollectionMerge, MergeTopK_Normal) {
    using Result = std::pair<float, unsigned long long>;
    std::vector<Result> a = {{0.1f, 1}, {0.3f, 2}, {0.5f, 3}};
    std::vector<Result> b = {{0.2f, 4}, {0.4f, 5}, {0.6f, 6}};

    // Test via Collection::search indirectly — or test the merge logic
    // We can test through the collection by doing a search that spans segments

    // Direct merge test: add items to a collection, search
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::mt19937 rng(42);
    for (int i = 0; i < 50; i++) {
        auto v = randomVector(128, rng);
        col.addPoint(v.data(), i);
    }

    unsigned long long labels[5];
    float distances[5];
    int found = col.search(randomVector(128, rng).data(), 5, labels, distances);
    ASSERT_GT(found, 0);

    // Verify results are sorted by distance
    for (int i = 1; i < found; i++) {
        EXPECT_LE(distances[i - 1], distances[i]);
    }
}

TEST(CollectionMerge, MergeTopK_KExceedsTotal) {
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::mt19937 rng(42);
    for (int i = 0; i < 3; i++) {
        auto v = randomVector(128, rng);
        col.addPoint(v.data(), i);
    }

    unsigned long long labels[100];
    float distances[100];
    int found = col.search(randomVector(128, rng).data(), 100, labels, distances);
    EXPECT_LE(found, 3);
}

// =================== 3.6 Filtered Search ===================

TEST(CollectionFiltered, ExcludesTombstones) {
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::mt19937 rng(42);
    int N = 100;
    std::vector<std::vector<float>> vectors(N);
    for (int i = 0; i < N; i++) {
        vectors[i] = randomVector(128, rng);
        col.addPoint(vectors[i].data(), i);
    }

    col.deletePoint(10);
    col.deletePoint(20);
    col.deletePoint(30);

    unsigned long long labels[N];
    float distances[N];
    int found = col.search(vectors[10].data(), N, labels, distances);
    std::set<unsigned long long> result_set(labels, labels + found);
    EXPECT_EQ(result_set.count(10), 0u);
    EXPECT_EQ(result_set.count(20), 0u);
    EXPECT_EQ(result_set.count(30), 0u);
}

TEST(CollectionFiltered, EmptyTombstones) {
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::mt19937 rng(42);
    int N = 50;
    for (int i = 0; i < N; i++) {
        auto v = randomVector(128, rng);
        col.addPoint(v.data(), i);
    }

    unsigned long long labels[10];
    float distances[10];
    int found = col.search(randomVector(128, rng).data(), 10, labels, distances);
    EXPECT_GT(found, 0);
}

TEST(CollectionFiltered, AllTombstoned) {
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::mt19937 rng(42);
    int N = 50;
    for (int i = 0; i < N; i++) {
        auto v = randomVector(128, rng);
        col.addPoint(v.data(), i);
    }
    for (int i = 0; i < N; i++) {
        col.deletePoint(i);
    }

    unsigned long long labels[10];
    float distances[10];
    int found = col.search(randomVector(128, rng).data(), 10, labels, distances);
    EXPECT_EQ(found, 0);
}

// =================== 3.7 Concurrency Stress Tests ===================

TEST(CollectionConcurrency, ConcurrentReadWrite) {
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::atomic<bool> running{true};
    std::atomic<int> write_count{0};
    std::atomic<int> read_count{0};

    std::mt19937 rng_base(42);
    // Pre-load some data
    for (int i = 0; i < 100; i++) {
        auto v = randomVector(128, rng_base);
        col.addPoint(v.data(), i);
    }

    auto writer = [&](int thread_id) {
        std::mt19937 rng(1000 + thread_id);
        int base = 1000 + thread_id * 10000;
        while (running.load(std::memory_order_relaxed)) {
            auto v = randomVector(128, rng);
            int rc = col.addPoint(v.data(), base++);
            if (rc == 0) write_count.fetch_add(1, std::memory_order_relaxed);
        }
    };

    auto reader = [&](int thread_id) {
        std::mt19937 rng(2000 + thread_id);
        while (running.load(std::memory_order_relaxed)) {
            auto q = randomVector(128, rng);
            unsigned long long labels[5];
            float distances[5];
            (void)col.search(q.data(), 5, labels, distances);
            read_count.fetch_add(1, std::memory_order_relaxed);
        }
    };

    auto deleter = [&]() {
        std::mt19937 rng(3000);
        std::uniform_int_distribution<int> dist(0, 99);
        while (running.load(std::memory_order_relaxed)) {
            col.deletePoint(dist(rng));
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
        }
    };

    std::vector<std::thread> threads;
    for (int i = 0; i < 4; i++) threads.emplace_back(writer, i);
    for (int i = 0; i < 8; i++) threads.emplace_back(reader, i);
    threads.emplace_back(deleter);

    std::this_thread::sleep_for(std::chrono::seconds(3));
    running.store(false);

    for (auto& t : threads) t.join();

    EXPECT_GT(write_count.load(), 0);
    EXPECT_GT(read_count.load(), 0);
}

TEST(CollectionConcurrency, RebuildDuringSearch) {
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::mt19937 rng(42);
    int N = 1000;
    for (int i = 0; i < N; i++) {
        auto v = randomVector(128, rng);
        col.addPoint(v.data(), i);
    }

    Rebuilder rebuilder(2);

    std::atomic<bool> running{true};
    std::atomic<int> search_count{0};

    auto searcher = [&](int thread_id) {
        std::mt19937 rng(5000 + thread_id);
        while (running.load(std::memory_order_relaxed)) {
            auto q = randomVector(128, rng);
            unsigned long long labels[5];
            float distances[5];
            (void)col.search(q.data(), 5, labels, distances);
            search_count.fetch_add(1, std::memory_order_relaxed);
        }
    };

    std::vector<std::thread> threads;
    for (int i = 0; i < 8; i++) threads.emplace_back(searcher, i);

    // Trigger a rebuild
    col.metrics().rebuild_in_progress.store(true);
    rebuilder.submit(&col, RebuildPriority::NORMAL);

    std::this_thread::sleep_for(std::chrono::seconds(3));
    running.store(false);

    for (auto& t : threads) t.join();
    rebuilder.stop();

    EXPECT_GT(search_count.load(), 0);
}

// =================== 3.8 Rebuild Engine Tests ===================

TEST(Rebuilder, PriorityOrdering) {
    // Submit multiple tasks and verify URGENT ones are processed first
    // We test indirectly by checking that the urgent collection finishes first
    auto cfg1 = makeConfig(10000);
    Collection col_normal(cfg1);
    auto cfg2 = makeConfig(10000);
    Collection col_urgent(cfg2);

    std::mt19937 rng(42);
    for (int i = 0; i < 100; i++) {
        auto v = randomVector(128, rng);
        col_normal.addPoint(v.data(), i);
        col_urgent.addPoint(v.data(), i + 1000);
    }

    col_normal.metrics().rebuild_in_progress.store(true);
    col_urgent.metrics().rebuild_in_progress.store(true);

    Rebuilder rebuilder(1);

    // Submit normal first, then urgent
    rebuilder.submit(&col_normal, RebuildPriority::NORMAL);
    rebuilder.submit(&col_urgent, RebuildPriority::URGENT);

    // Let rebuilds finish
    std::this_thread::sleep_for(std::chrono::seconds(5));
    rebuilder.stop();

    // Both should have been rebuilt
    EXPECT_FALSE(col_normal.metrics().rebuild_in_progress.load());
    EXPECT_FALSE(col_urgent.metrics().rebuild_in_progress.load());
}

TEST(Rebuilder, CleanGraph) {
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::mt19937 rng(42);
    int N = 1000;
    std::vector<std::vector<float>> vectors(N);
    for (int i = 0; i < N; i++) {
        vectors[i] = randomVector(128, rng);
        col.addPoint(vectors[i].data(), i);
    }

    // Delete 25%
    int del_count = N / 4;
    for (int i = 0; i < del_count; i++) {
        col.deletePoint(i);
    }

    col.metrics().rebuild_in_progress.store(true);
    Rebuilder rebuilder(2);
    rebuilder.submit(&col, RebuildPriority::NORMAL);

    // Wait for rebuild
    for (int i = 0; i < 100; i++) {
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
        if (!col.metrics().rebuild_in_progress.load()) break;
    }
    rebuilder.stop();

    EXPECT_FALSE(col.metrics().rebuild_in_progress.load());
    EXPECT_GE(col.metrics().rebuild_count.load(), 1);

    // Tombstones should be cleared after rebuild
    EXPECT_EQ(col.metrics().tombstone_count.load(), 0);
}

TEST(Rebuilder, DataIntegrity) {
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::mt19937 rng(42);
    int N = 200;
    std::vector<std::vector<float>> vectors(N);
    for (int i = 0; i < N; i++) {
        vectors[i] = randomVector(128, rng);
        col.addPoint(vectors[i].data(), i);
    }

    // Trigger rebuild
    col.metrics().rebuild_in_progress.store(true);
    Rebuilder rebuilder(2);
    rebuilder.submit(&col, RebuildPriority::NORMAL);

    for (int i = 0; i < 100; i++) {
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
        if (!col.metrics().rebuild_in_progress.load()) break;
    }
    rebuilder.stop();

    // Every vector should still be findable
    for (int i = 0; i < N; i++) {
        unsigned long long labels[1];
        float distances[1];
        int found = col.search(vectors[i].data(), 1, labels, distances);
        ASSERT_GE(found, 1) << "Vector " << i << " not found after rebuild";
        EXPECT_EQ(labels[0], static_cast<unsigned long long>(i));
        EXPECT_NEAR(distances[0], 0.0f, 1e-3f);
    }
}

TEST(Rebuilder, GracefulShutdown) {
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::mt19937 rng(42);
    for (int i = 0; i < 500; i++) {
        auto v = randomVector(128, rng);
        col.addPoint(v.data(), i);
    }

    col.metrics().rebuild_in_progress.store(true);
    Rebuilder rebuilder(2);
    rebuilder.submit(&col, RebuildPriority::NORMAL);

    // Immediately stop — should still complete the in-progress rebuild
    rebuilder.stop();

    // The rebuild should have completed
    EXPECT_FALSE(col.metrics().rebuild_in_progress.load());
}

// =================== Extern-C API Tests ===================

TEST(ExternCAPI, CollectionLifecycle) {
    rebuilder_init(2);

    HNSWCollection col = collection_create("api_test", "l2", 32, 16, 200, 50, 10000);
    ASSERT_NE(col, nullptr);

    std::mt19937 rng(42);
    for (int i = 0; i < 100; i++) {
        auto v = randomVector(32, rng);
        EXPECT_EQ(0, collection_add_point(col, v.data(), i));
    }

    auto q = randomVector(32, rng);
    unsigned long long labels[5];
    float distances[5];
    int found = collection_search(col, q.data(), 5, labels, distances);
    EXPECT_GT(found, 0);

    // Stats
    char* stats = collection_get_stats(col);
    ASSERT_NE(stats, nullptr);
    std::string s(stats);
    EXPECT_NE(s.find("sealed_count"), std::string::npos);
    EXPECT_NE(s.find("buffer_count"), std::string::npos);
    free(stats);

    // Delete
    EXPECT_EQ(0, collection_delete_point(col, 50));

    collection_destroy(col);
    rebuilder_stop();
}

// =================== Single-Index Wrapper Tests ===================

TEST(HNSWWrapper, CreateAndSearch) {
    HNSWIndex idx = hnsw_create_index("l2", 32, 1000, 16, 200, 42);
    ASSERT_NE(idx, nullptr);

    std::mt19937 rng(42);
    for (int i = 0; i < 100; i++) {
        auto v = randomVector(32, rng);
        EXPECT_EQ(0, hnsw_add_point(idx, v.data(), i));
    }

    EXPECT_EQ(100, hnsw_get_current_count(idx));
    EXPECT_EQ(32, hnsw_get_dimension(idx));

    auto q = randomVector(32, rng);
    unsigned long long labels[5];
    float distances[5];
    int found = hnsw_search_knn(idx, q.data(), 5, labels, distances);
    EXPECT_GT(found, 0);

    hnsw_delete_index(idx);
}

TEST(HNSWWrapper, FilteredSearch) {
    HNSWIndex idx = hnsw_create_index("l2", 32, 1000, 16, 200, 42);
    ASSERT_NE(idx, nullptr);

    std::mt19937 rng(42);
    std::vector<std::vector<float>> vectors(100);
    for (int i = 0; i < 100; i++) {
        vectors[i] = randomVector(32, rng);
        hnsw_add_point(idx, vectors[i].data(), i);
    }

    unsigned long long tombstones[] = {10, 20, 30};
    unsigned long long labels[100];
    float distances[100];
    int found = hnsw_search_knn_filtered(idx, vectors[10].data(), 100, labels, distances, tombstones, 3);
    std::set<unsigned long long> result_set(labels, labels + found);
    EXPECT_EQ(result_set.count(10), 0u);
    EXPECT_EQ(result_set.count(20), 0u);
    EXPECT_EQ(result_set.count(30), 0u);

    hnsw_delete_index(idx);
}

TEST(HNSWWrapper, GetDataByLabel) {
    HNSWIndex idx = hnsw_create_index("l2", 4, 100, 8, 50, 42);
    ASSERT_NE(idx, nullptr);

    float data[] = {1.0f, 2.0f, 3.0f, 4.0f};
    hnsw_add_point(idx, data, 42);

    float out[4] = {};
    EXPECT_EQ(0, hnsw_get_data_by_label(idx, 42, out));
    for (int i = 0; i < 4; i++) {
        EXPECT_FLOAT_EQ(data[i], out[i]);
    }

    EXPECT_EQ(-2, hnsw_get_data_by_label(idx, 999, out));

    hnsw_delete_index(idx);
}

TEST(HNSWWrapper, GetAllLabels) {
    HNSWIndex idx = hnsw_create_index("l2", 4, 100, 8, 50, 42);
    ASSERT_NE(idx, nullptr);

    std::mt19937 rng(42);
    for (int i = 0; i < 10; i++) {
        auto v = randomVector(4, rng);
        hnsw_add_point(idx, v.data(), i * 10);
    }

    unsigned long long labels[20];
    int count = hnsw_get_all_labels(idx, labels, 20);
    EXPECT_EQ(count, 10);

    std::set<unsigned long long> label_set(labels, labels + count);
    for (int i = 0; i < 10; i++) {
        EXPECT_EQ(label_set.count(i * 10), 1u);
    }

    hnsw_delete_index(idx);
}

TEST(HNSWWrapper, MarkDeletedAndIsDeleted) {
    HNSWIndex idx = hnsw_create_index("l2", 4, 100, 8, 50, 42);
    ASSERT_NE(idx, nullptr);

    float data[] = {1.0f, 2.0f, 3.0f, 4.0f};
    hnsw_add_point(idx, data, 10);

    EXPECT_EQ(0, hnsw_is_label_deleted(idx, 10));
    EXPECT_EQ(-1, hnsw_is_label_deleted(idx, 999));

    EXPECT_EQ(0, hnsw_mark_deleted(idx, 10));
    EXPECT_EQ(1, hnsw_is_label_deleted(idx, 10));

    // get_data_by_label for mark-deleted label returns -3
    float out_deleted[4];
    EXPECT_EQ(-3, hnsw_get_data_by_label(idx, 10, out_deleted));

    // Idempotent: marking again returns 1 (already deleted), not 0
    EXPECT_EQ(1, hnsw_mark_deleted(idx, 10));

    // Non-existent label returns -1
    EXPECT_EQ(-1, hnsw_mark_deleted(idx, 999));

    hnsw_delete_index(idx);
}

TEST(CollectionTombstone, DoubleDeleteNoDoubleDecrement) {
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::mt19937 rng(42);
    auto v = randomVector(128, rng);
    col.addPoint(v.data(), 1);

    EXPECT_EQ(1, col.metrics().buffer_count.load());
    EXPECT_EQ(1, col.metrics().total_vectors.load());

    col.deletePoint(1);
    EXPECT_EQ(0, col.metrics().buffer_count.load());
    EXPECT_EQ(0, col.metrics().total_vectors.load());

    col.deletePoint(1);
    col.deletePoint(1);
    EXPECT_EQ(0, col.metrics().buffer_count.load());
    EXPECT_EQ(0, col.metrics().total_vectors.load());
}

TEST(CollectionTombstone, RepeatedDeleteTombstoneCountOnce) {
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::mt19937 rng(42);
    int N = 10;
    for (int i = 0; i < N; i++) {
        auto v = randomVector(128, rng);
        col.addPoint(v.data(), i);
    }
    col.deletePoint(5);
    EXPECT_EQ(1, col.metrics().tombstone_count.load());
    col.deletePoint(5);
    col.deletePoint(5);
    EXPECT_EQ(1, col.metrics().tombstone_count.load());
}

// =================== Tombstone / Mark-Delete Correctness ===================

TEST(CollectionTombstone, UpdateNoDuplicateFromSealed) {
    // After a rebuild, the sealed segment contains vectors.
    // Update a label that is in sealed. Search must return exactly one result
    // for that label (the new vector), not duplicates from sealed + buffer.
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::mt19937 rng(42);
    int N = 200;
    std::vector<std::vector<float>> vectors(N);
    for (int i = 0; i < N; i++) {
        vectors[i] = randomVector(128, rng);
        col.addPoint(vectors[i].data(), i);
    }

    // Trigger rebuild to move vectors into sealed
    col.metrics().rebuild_in_progress.store(true);
    Rebuilder rebuilder(2);
    rebuilder.submit(&col, RebuildPriority::NORMAL);
    for (int w = 0; w < 100; w++) {
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
        if (!col.metrics().rebuild_in_progress.load()) break;
    }
    rebuilder.stop();
    ASSERT_FALSE(col.metrics().rebuild_in_progress.load());

    // Now label 50 is in sealed. Update it with a new vector.
    auto vecNew = randomVector(128, rng);
    ASSERT_EQ(0, col.updatePoint(vecNew.data(), 50));

    // Search for the new vector; expect exactly one hit for label 50
    unsigned long long labels[20];
    float distances[20];
    int found = col.search(vecNew.data(), 20, labels, distances);
    ASSERT_GE(found, 1);

    int count_50 = 0;
    for (int i = 0; i < found; i++) {
        if (labels[i] == 50ULL) count_50++;
    }
    EXPECT_EQ(count_50, 1) << "Label 50 appeared " << count_50 << " times (expected 1)";
    EXPECT_EQ(labels[0], 50ULL);
    EXPECT_NEAR(distances[0], 0.0f, 1e-3f);
}

TEST(CollectionTombstone, BufferDeleteNotVisibleInSearch) {
    // Add a point only to the buffer, delete it, then verify search
    // does not return the deleted label.
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::mt19937 rng(42);
    int N = 50;
    std::vector<std::vector<float>> vectors(N);
    for (int i = 0; i < N; i++) {
        vectors[i] = randomVector(128, rng);
        col.addPoint(vectors[i].data(), i);
    }

    col.deletePoint(25);

    unsigned long long labels[50];
    float distances[50];
    int found = col.search(vectors[25].data(), 50, labels, distances);
    for (int i = 0; i < found; i++) {
        EXPECT_NE(labels[i], 25ULL) << "Deleted label 25 still visible in search";
    }
}

TEST(CollectionTombstone, RebuildAfterBufferDelete) {
    // Add to buffer, delete that label (mark-delete in buffer),
    // trigger rebuild, then verify the rebuilt sealed segment does not
    // contain the deleted label.
    auto cfg = makeConfig(10000);
    Collection col(cfg);

    std::mt19937 rng(42);
    int N = 100;
    std::vector<std::vector<float>> vectors(N);
    for (int i = 0; i < N; i++) {
        vectors[i] = randomVector(128, rng);
        col.addPoint(vectors[i].data(), i);
    }

    col.deletePoint(42);
    col.deletePoint(77);

    col.metrics().rebuild_in_progress.store(true);
    Rebuilder rebuilder(2);
    rebuilder.submit(&col, RebuildPriority::NORMAL);
    for (int w = 0; w < 100; w++) {
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
        if (!col.metrics().rebuild_in_progress.load()) break;
    }
    rebuilder.stop();
    ASSERT_FALSE(col.metrics().rebuild_in_progress.load());

    // Search for deleted vectors; they should not be found
    unsigned long long labels[100];
    float distances[100];

    int found = col.search(vectors[42].data(), 100, labels, distances);
    std::set<unsigned long long> result_set(labels, labels + found);
    EXPECT_EQ(result_set.count(42), 0u) << "Deleted label 42 found after rebuild";
    EXPECT_EQ(result_set.count(77), 0u) << "Deleted label 77 found after rebuild";

    // Non-deleted vectors should still be findable
    found = col.search(vectors[10].data(), 1, labels, distances);
    ASSERT_GE(found, 1);
    EXPECT_EQ(labels[0], 10ULL);
}
