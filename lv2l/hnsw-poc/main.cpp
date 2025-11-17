#include <iostream>
#include <random>
#include <vector>
#include <chrono>
#include "hnswlib/hnswlib.h"

using Clock = std::chrono::high_resolution_clock;

static std::vector<fs::path> find_all_snappy_parquet_in(const fs::path& data_dir) {
    std::vector<fs::path> snappy_files;
    
    if (!fs::exists(data_dir) || !fs::is_directory(data_dir)) return snappy_files;

    for (auto& p : fs::directory_iterator(data_dir)) {
        if (!p.is_regular_file()) continue;
        if (p.path().extension() == ".parquet") {
            const auto fname = p.path().filename().string();
            // Only collect *.snappy.parquet files
            if (fname.size() >= 16 && fname.rfind(".snappy.parquet") == fname.size() - 16) {
                snappy_files.push_back(p.path());
            }
        }
    }
    
    return snappy_files;
}

int main(int argc, char** argv) {
    fs::path data_dir = (argc > 1) ? fs::path(argv[1]) : fs::path("data");

    auto snappy_files = find_all_snappy_parquet_in(data_dir);
    if (snappy_files.empty()) {
        std::cerr << "No .snappy.parquet files found in: " << data_dir << "\n";
        return 1;
    }

    std::cout << "Found " << snappy_files.size() << " .snappy.parquet file(s):\n";

    
    const size_t dim = 768;         // vector dimensionality
    const size_t num_points = 1000000; // dataset size
    const size_t M = 32;            // graph degree (tradeoff: mem/search)
    const size_t ef_construction = 200;
    const size_t ef_search = 128;    // higher = better recall, slower
    const size_t k = 1000;            // top-k neighbors
    const bool use_cosine = false;  // false => L2, true => Cosine

    // --- RNG for synthetic data ---
    std::mt19937 rng(42);
    std::normal_distribution<float> dist(0.0f, 1.0f);

    // --- Create/own the space and index ---
    // Space options: hnswlib::L2Space or hnswlib::InnerProductSpace (for cosine, normalize vectors)
    std::unique_ptr<hnswlib::SpaceInterface<float>> space;
    if (use_cosine) {
        // For cosine, either normalize vectors and use InnerProduct,
        // or use CosineSpace if present in your checkout.
        space.reset(new hnswlib::InnerProductSpace(dim));
    } else {
        space.reset(new hnswlib::L2Space(dim));
    }

    // Index that stores int labels
    hnswlib::HierarchicalNSW<float> index(space.get(), num_points, M, ef_construction);

    // Optional: set number of threads used internally (0 = auto)
    //index.setNumThreads(std::thread::hardware_concurrency());

    // --- Build dataset and add to index ---
    std::vector<float> vec(dim);
    auto t0 = Clock::now();
    for (size_t i = 0; i < num_points; ++i) {
        for (size_t d = 0; d < dim; ++d) vec[d] = dist(rng);

        if (use_cosine) {
            // L2-norm normalize for cosine via inner product
            float norm = 0.f;
            for (float v : vec) norm += v * v;
            norm = std::sqrt(norm) + 1e-12f;
            for (float &v : vec) v /= norm;
        }

        index.addPoint(vec.data(), static_cast<hnswlib::labeltype>(i));
    }
    auto t1 = Clock::now();
    double build_ms = std::chrono::duration<double, std::milli>(t1 - t0).count();
    std::cout << "Built index for " << num_points << " points in " << build_ms << " ms\n";

    // --- Search a few queries ---
    index.setEf(ef_search);

    size_t num_queries = 5;
    for (size_t qi = 0; qi < num_queries; ++qi) {
        for (size_t d = 0; d < dim; ++d) vec[d] = dist(rng);
        if (use_cosine) {
            float norm = 0.f;
            for (float v : vec) norm += v * v;
            norm = std::sqrt(norm) + 1e-12f;
            for (float &v : vec) v /= norm;
        }

        auto q0 = Clock::now();
        auto result = index.searchKnn(vec.data(), k);
        auto q1 = Clock::now();
        double q_ms = std::chrono::duration<double, std::micro>(q1 - q0).count();

        std::vector<std::pair<hnswlib::labeltype, float>> neighbors;
        neighbors.reserve(k);
        while (!result.empty()) {
            auto &top = result.top();
            neighbors.emplace_back(top.first, top.second);
            result.pop();
        }
        std::reverse(neighbors.begin(), neighbors.end()); // closest first

        std::cout << "Query " << qi << " (" << k << "-NN, ~" << q_ms << " us): ";
        for (auto &p : neighbors) std::cout << "(" << p.first << ", " << p.second << ") ";
        std::cout << "\n";
    }

    // --- Save & Load roundtrip ---
    const std::string path = "hnsw.index";
    index.saveIndex(path);
    std::cout << "Saved index to " << path << "\n";

    // Construct a new index object and load
    hnswlib::HierarchicalNSW<float> loaded(space.get(), path);
    loaded.setEf(ef_search);

    // quick check: search again with loaded index
    for (size_t d = 0; d < dim; ++d) vec[d] = dist(rng);
    if (use_cosine) {
        float norm = 0.f;
        for (float v : vec) norm += v * v;
        norm = std::sqrt(norm) + 1e-12f;
        for (float &v : vec) v /= norm;
    }
    auto res2 = loaded.searchKnn(vec.data(), k);
    std::cout << "Loaded index search top-1 label: " << (res2.empty() ? -1 : res2.top().first) << "\n";

    return 0;
}
