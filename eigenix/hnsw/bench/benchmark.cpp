#include <benchmark/benchmark.h>
#include "collection.h"
#include "rebuilder.h"
#include "hnswlib.h"

#include <random>
#include <vector>
#include <thread>
#include <atomic>
#include <algorithm>
#include <cmath>
#include <numeric>

// =================== Helpers ===================

static std::vector<float> randomVec(int dim, std::mt19937& rng) {
    std::uniform_real_distribution<float> dist(-1.0f, 1.0f);
    std::vector<float> v(dim);
    for (int i = 0; i < dim; i++) v[i] = dist(rng);
    return v;
}

static CollectionConfig benchConfig(int64_t sealed_cap = 200000) {
    CollectionConfig cfg;
    cfg.name = "bench";
    cfg.space_name = "l2";
    cfg.dimension = 128;
    cfg.M = 16;
    cfg.ef_construction = 200;
    cfg.ef_search = 50;
    cfg.initial_sealed_capacity = sealed_cap;
    return cfg;
}

// Pre-built dataset holder (avoids re-generating for every benchmark iteration)
struct Dataset {
    int dim;
    int count;
    std::vector<std::vector<float>> vectors;

    Dataset(int dim, int count, int seed = 42) : dim(dim), count(count) {
        std::mt19937 rng(seed);
        vectors.resize(count);
        for (int i = 0; i < count; i++) {
            vectors[i] = randomVec(dim, rng);
        }
    }
};

// =================== Baseline Single-Index Benchmarks ===================

static void BM_Baseline_Search(benchmark::State& state) {
    int N = state.range(0);
    int dim = 128;
    Dataset data(dim, N, 42);

    HNSWIndex idx = hnsw_create_index("l2", dim, N, 16, 200, 42);
    hnsw_set_ef(idx, 50);
    for (int i = 0; i < N; i++) {
        hnsw_add_point(idx, data.vectors[i].data(), i);
    }

    std::mt19937 rng(100);
    std::vector<unsigned long long> labels(10);
    std::vector<float> distances(10);

    for (auto _ : state) {
        auto q = randomVec(dim, rng);
        benchmark::DoNotOptimize(
            hnsw_search_knn(idx, q.data(), 10, labels.data(), distances.data()));
    }

    state.SetItemsProcessed(state.iterations());
    hnsw_delete_index(idx);
}

BENCHMARK(BM_Baseline_Search)->Arg(1000)->Arg(10000)->Arg(100000)->Unit(benchmark::kMicrosecond);

static void BM_Baseline_AddPoint(benchmark::State& state) {
    int dim = 128;
    int max_el = 200000;
    HNSWIndex idx = hnsw_create_index("l2", dim, max_el, 16, 200, 42);

    std::mt19937 rng(42);
    unsigned long long label = 0;

    for (auto _ : state) {
        auto v = randomVec(dim, rng);
        benchmark::DoNotOptimize(hnsw_add_point(idx, v.data(), label++));
    }

    state.SetItemsProcessed(state.iterations());
    hnsw_delete_index(idx);
}

BENCHMARK(BM_Baseline_AddPoint)->Unit(benchmark::kMicrosecond);

// =================== Collection Benchmarks ===================

static void BM_Collection_Search(benchmark::State& state) {
    int N = state.range(0);
    int dim = 128;
    Dataset data(dim, N, 42);

    auto cfg = benchConfig(N + 10000);
    Collection col(cfg);
    for (int i = 0; i < N; i++) {
        col.addPoint(data.vectors[i].data(), i);
    }

    std::mt19937 rng(100);
    std::vector<unsigned long long> labels(10);
    std::vector<float> distances(10);

    for (auto _ : state) {
        auto q = randomVec(dim, rng);
        benchmark::DoNotOptimize(
            col.search(q.data(), 10, labels.data(), distances.data()));
    }

    state.SetItemsProcessed(state.iterations());
}

BENCHMARK(BM_Collection_Search)->Arg(1000)->Arg(10000)->Unit(benchmark::kMicrosecond);

static void BM_Collection_AddPoint(benchmark::State& state) {
    int dim = 128;
    auto cfg = benchConfig(200000);
    Collection col(cfg);

    std::mt19937 rng(42);
    unsigned long long label = 0;

    for (auto _ : state) {
        auto v = randomVec(dim, rng);
        benchmark::DoNotOptimize(col.addPoint(v.data(), label++));
    }

    state.SetItemsProcessed(state.iterations());
}

BENCHMARK(BM_Collection_AddPoint)->Unit(benchmark::kMicrosecond);

// =================== Filtered Search Benchmarks ===================

static void BM_FilteredSearch(benchmark::State& state) {
    int tombstone_pct = state.range(0);
    int N = 10000;
    int dim = 128;
    Dataset data(dim, N, 42);

    auto cfg = benchConfig(N + 10000);
    Collection col(cfg);
    for (int i = 0; i < N; i++) {
        col.addPoint(data.vectors[i].data(), i);
    }

    int num_tombstones = N * tombstone_pct / 100;
    for (int i = 0; i < num_tombstones; i++) {
        col.deletePoint(i);
    }

    std::mt19937 rng(100);
    std::vector<unsigned long long> labels(10);
    std::vector<float> distances(10);

    for (auto _ : state) {
        auto q = randomVec(dim, rng);
        benchmark::DoNotOptimize(
            col.search(q.data(), 10, labels.data(), distances.data()));
    }

    state.SetItemsProcessed(state.iterations());
    state.SetLabel(std::to_string(tombstone_pct) + "% tombstones");
}

BENCHMARK(BM_FilteredSearch)->Arg(0)->Arg(10)->Arg(20)->Unit(benchmark::kMicrosecond);

// =================== Rebuild Benchmark ===================

static void BM_Rebuild(benchmark::State& state) {
    int N = state.range(0);
    int dim = 128;
    Dataset data(dim, N, 42);

    for (auto _ : state) {
        state.PauseTiming();
        auto cfg = benchConfig(N + 10000);
        Collection col(cfg);
        for (int i = 0; i < N; i++) {
            col.addPoint(data.vectors[i].data(), i);
        }
        // Delete 25%
        for (int i = 0; i < N / 4; i++) {
            col.deletePoint(i);
        }
        col.metrics().rebuild_in_progress.store(true);
        state.ResumeTiming();

        Rebuilder rebuilder(2);
        rebuilder.submit(&col, RebuildPriority::NORMAL);

        // Wait for completion
        while (col.metrics().rebuild_in_progress.load()) {
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
        }
        rebuilder.stop();
    }
}

BENCHMARK(BM_Rebuild)->Arg(1000)->Arg(10000)->Unit(benchmark::kMillisecond);

// =================== Concurrent Throughput ===================

static void BM_Concurrent(benchmark::State& state) {
    int num_writers = state.range(0);
    int num_readers = state.range(1);
    int dim = 128;
    int N = 5000;

    auto cfg = benchConfig(N + 50000);
    Collection col(cfg);

    std::mt19937 rng_init(42);
    for (int i = 0; i < N; i++) {
        auto v = randomVec(dim, rng_init);
        col.addPoint(v.data(), i);
    }

    for (auto _ : state) {
        std::atomic<bool> running{true};
        std::atomic<int64_t> total_search_ops{0};
        std::atomic<int64_t> total_write_ops{0};

        std::vector<std::thread> threads;

        for (int w = 0; w < num_writers; w++) {
            threads.emplace_back([&, w]() {
                std::mt19937 rng(1000 + w);
                unsigned long long label = 100000ULL + w * 100000;
                while (running.load(std::memory_order_relaxed)) {
                    auto v = randomVec(dim, rng);
                    if (col.addPoint(v.data(), label++) == 0)
                        total_write_ops.fetch_add(1, std::memory_order_relaxed);
                }
            });
        }

        for (int r = 0; r < num_readers; r++) {
            threads.emplace_back([&, r]() {
                std::mt19937 rng(2000 + r);
                unsigned long long labels[10];
                float distances[10];
                while (running.load(std::memory_order_relaxed)) {
                    auto q = randomVec(dim, rng);
                    (void)col.search(q.data(), 10, labels, distances);
                    total_search_ops.fetch_add(1, std::memory_order_relaxed);
                }
            });
        }

        std::this_thread::sleep_for(std::chrono::seconds(5));
        running.store(false);

        for (auto& t : threads) t.join();

        state.counters["search_ops/s"] = benchmark::Counter(
            total_search_ops.load(), benchmark::Counter::kIsRate);
        state.counters["write_ops/s"] = benchmark::Counter(
            total_write_ops.load(), benchmark::Counter::kIsRate);
    }
}

BENCHMARK(BM_Concurrent)
    ->Args({0, 8})   // pure read
    ->Args({1, 7})   // 12.5% write
    ->Args({4, 4})   // 50% write
    ->Unit(benchmark::kSecond)
    ->Iterations(1);

// =================== Recall Benchmark ===================

static void BM_Recall(benchmark::State& state) {
    int tombstone_pct = state.range(0);
    int N = 5000;
    int dim = 128;
    int k = 10;
    int num_queries = 200;

    Dataset data(dim, N, 42);
    Dataset queries(dim, num_queries, 100);

    // Build brute-force ground truth
    hnswlib::L2Space space(dim);
    hnswlib::BruteforceSearch<float> bf(&space, N);
    for (int i = 0; i < N; i++) {
        bf.addPoint(data.vectors[i].data(), i);
    }

    auto cfg = benchConfig(N + 10000);

    for (auto _ : state) {
        state.PauseTiming();
        Collection col(cfg);
        for (int i = 0; i < N; i++) {
            col.addPoint(data.vectors[i].data(), i);
        }

        int num_tombstones = N * tombstone_pct / 100;
        for (int i = 0; i < num_tombstones; i++) {
            col.deletePoint(i);
        }

        // Build ground truth without tombstoned IDs
        std::unordered_set<hnswlib::labeltype> tombstone_set;
        for (int i = 0; i < num_tombstones; i++) {
            tombstone_set.insert(static_cast<hnswlib::labeltype>(i));
        }

        state.ResumeTiming();

        double total_recall = 0.0;
        for (int q = 0; q < num_queries; q++) {
            // Ground truth (brute force, filtered)
            auto gt_results = bf.searchKnn(queries.vectors[q].data(), k + num_tombstones);
            std::set<unsigned long long> gt_set;
            while (!gt_results.empty()) {
                auto [dist, label] = gt_results.top();
                gt_results.pop();
                if (tombstone_set.count(label) == 0 && static_cast<int>(gt_set.size()) < k) {
                    gt_set.insert(label);
                }
            }

            // Collection search
            std::vector<unsigned long long> labels(k);
            std::vector<float> distances(k);
            int found = col.search(queries.vectors[q].data(), k, labels.data(), distances.data());

            int hits = 0;
            for (int i = 0; i < found; i++) {
                if (gt_set.count(labels[i])) hits++;
            }

            if (!gt_set.empty()) {
                total_recall += static_cast<double>(hits) / static_cast<double>(gt_set.size());
            }
        }

        double avg_recall = total_recall / num_queries;
        state.counters["recall@10"] = avg_recall;
    }
}

BENCHMARK(BM_Recall)
    ->Arg(0)    // clean
    ->Arg(10)   // 10% tombstoned
    ->Arg(19)   // 19% tombstoned (worst case before rebuild)
    ->Unit(benchmark::kMillisecond)
    ->Iterations(1);

BENCHMARK_MAIN();
