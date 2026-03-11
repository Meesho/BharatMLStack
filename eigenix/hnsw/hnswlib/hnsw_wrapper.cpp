#include "hnsw_wrapper.h"
#include "hnswlib.h"
#include "collection.h"
#include "rebuilder.h"

#include <cstring>
#include <cmath>
#include <cstdio>
#include <cstdlib>
#include <string>
#include <vector>
#include <unordered_set>
#include <algorithm>
#include <memory>

struct HNSWIndexImpl {
    hnswlib::SpaceInterface<float>* space;
    hnswlib::HierarchicalNSW<float>* alg_hnsw;
    int dimension;
    std::string space_name;
    bool normalize;
};

// hnswlib's BaseFilterFunctor that excludes tombstoned labels
class TombstoneFilter : public hnswlib::BaseFilterFunctor {
    const std::unordered_set<hnswlib::labeltype>& blocked_;
public:
    explicit TombstoneFilter(const std::unordered_set<hnswlib::labeltype>& blocked)
        : blocked_(blocked) {}

    bool operator()(hnswlib::labeltype id) override {
        return blocked_.find(id) == blocked_.end();
    }
};

static hnswlib::SpaceInterface<float>* create_space(const char* space_name, int dim) {
    std::string sname(space_name);
    if (sname == "l2") {
        return new hnswlib::L2Space(dim);
    } else if (sname == "ip" || sname == "cosine") {
        return new hnswlib::InnerProductSpace(dim);
    }
    return nullptr;
}

static bool needs_normalize(const char* space_name) {
    return std::string(space_name) == "cosine";
}

static std::vector<float> normalize_vector(const float* data, int dim) {
    std::vector<float> norm(data, data + dim);
    float mag = 0.0f;
    for (int i = 0; i < dim; i++) {
        mag += norm[i] * norm[i];
    }
    mag = std::sqrt(mag);
    if (mag > 0.0f) {
        for (int i = 0; i < dim; i++) {
            norm[i] /= mag;
        }
    }
    return norm;
}

extern "C" {

HNSWIndex hnsw_create_index(
    const char* space_name,
    int dim,
    int max_elements,
    int M,
    int ef_construction,
    int random_seed)
{
    auto* impl = new HNSWIndexImpl();
    impl->space = create_space(space_name, dim);
    if (!impl->space) {
        delete impl;
        return nullptr;
    }
    impl->dimension = dim;
    impl->space_name = space_name;
    impl->normalize = needs_normalize(space_name);

    try {
        impl->alg_hnsw = new hnswlib::HierarchicalNSW<float>(
            impl->space,
            static_cast<size_t>(max_elements),
            static_cast<size_t>(M),
            static_cast<size_t>(ef_construction),
            static_cast<size_t>(random_seed));
    } catch (...) {
        delete impl->space;
        delete impl;
        return nullptr;
    }
    return static_cast<HNSWIndex>(impl);
}


void hnsw_delete_index(HNSWIndex index) {
    if (!index) return;
    auto* impl = static_cast<HNSWIndexImpl*>(index);
    delete impl->alg_hnsw;
    delete impl->space;
    delete impl;
}


int hnsw_add_point(HNSWIndex index, const float* data, unsigned long long label) {
    if (!index) return -1;
    auto* impl = static_cast<HNSWIndexImpl*>(index);
    try {
        if (impl->normalize) {
            auto norm = normalize_vector(data, impl->dimension);
            impl->alg_hnsw->addPoint(norm.data(), static_cast<hnswlib::labeltype>(label));
        } else {
            impl->alg_hnsw->addPoint(data, static_cast<hnswlib::labeltype>(label));
        }
        return 0;
    } catch (...) {
        return -1;
    }
}


int hnsw_search_knn(
    HNSWIndex index,
    const float* query_data,
    int k,
    unsigned long long* labels,
    float* distances)
{
    if (!index) return -1;
    auto* impl = static_cast<HNSWIndexImpl*>(index);

    try {
        const float* query = query_data;
        std::vector<float> norm;
        if (impl->normalize) {
            norm = normalize_vector(query_data, impl->dimension);
            query = norm.data();
        }

        auto result = impl->alg_hnsw->searchKnn(query, static_cast<size_t>(k));

        int count = static_cast<int>(result.size());
        for (int i = count - 1; i >= 0; i--) {
            auto& top = result.top();
            distances[i] = top.first;
            labels[i] = static_cast<unsigned long long>(top.second);
            result.pop();
        }
        return count;
    } catch (...) {
        return -1;
    }
}


int hnsw_search_knn_filtered(
    HNSWIndex index,
    const float* query_data,
    int k,
    unsigned long long* labels,
    float* distances,
    const unsigned long long* tombstone_ids,
    int tombstone_count)
{
    if (!index) return -1;
    auto* impl = static_cast<HNSWIndexImpl*>(index);

    try {
        const float* query = query_data;
        std::vector<float> norm;
        if (impl->normalize) {
            norm = normalize_vector(query_data, impl->dimension);
            query = norm.data();
        }

        std::priority_queue<std::pair<float, hnswlib::labeltype>> result;

        if (tombstone_count == 0) {
            result = impl->alg_hnsw->searchKnn(query, static_cast<size_t>(k));
        } else {
            std::unordered_set<hnswlib::labeltype> blocked;
            blocked.reserve(tombstone_count);
            for (int i = 0; i < tombstone_count; i++) {
                blocked.insert(static_cast<hnswlib::labeltype>(tombstone_ids[i]));
            }
            TombstoneFilter filter(blocked);
            result = impl->alg_hnsw->searchKnn(query, static_cast<size_t>(k), &filter);
        }

        int count = static_cast<int>(result.size());
        for (int i = count - 1; i >= 0; i--) {
            auto& top = result.top();
            distances[i] = top.first;
            labels[i] = static_cast<unsigned long long>(top.second);
            result.pop();
        }
        return count;
    } catch (...) {
        return -1;
    }
}


void hnsw_set_ef(HNSWIndex index, int ef) {
    if (!index) return;
    auto* impl = static_cast<HNSWIndexImpl*>(index);
    impl->alg_hnsw->setEf(static_cast<size_t>(ef));
}


int hnsw_get_current_count(HNSWIndex index) {
    if (!index) return 0;
    auto* impl = static_cast<HNSWIndexImpl*>(index);
    return static_cast<int>(impl->alg_hnsw->getCurrentElementCount());
}


int hnsw_get_max_elements(HNSWIndex index) {
    if (!index) return 0;
    auto* impl = static_cast<HNSWIndexImpl*>(index);
    return static_cast<int>(impl->alg_hnsw->getMaxElements());
}


int hnsw_get_dimension(HNSWIndex index) {
    if (!index) return 0;
    auto* impl = static_cast<HNSWIndexImpl*>(index);
    return impl->dimension;
}


int hnsw_save_index(HNSWIndex index, const char* path) {
    if (!index) return -1;
    auto* impl = static_cast<HNSWIndexImpl*>(index);
    try {
        impl->alg_hnsw->saveIndex(std::string(path));
        return 0;
    } catch (...) {
        return -1;
    }
}


HNSWIndex hnsw_load_index(
    const char* path,
    const char* space_name,
    int dim,
    int max_elements)
{
    auto* impl = new HNSWIndexImpl();
    impl->space = create_space(space_name, dim);
    if (!impl->space) {
        delete impl;
        return nullptr;
    }
    impl->dimension = dim;
    impl->space_name = space_name;
    impl->normalize = needs_normalize(space_name);

    try {
        impl->alg_hnsw = new hnswlib::HierarchicalNSW<float>(
            impl->space,
            std::string(path),
            false,
            static_cast<size_t>(max_elements));
    } catch (...) {
        delete impl->space;
        delete impl;
        return nullptr;
    }
    return static_cast<HNSWIndex>(impl);
}


int hnsw_get_data_by_label(HNSWIndex index, unsigned long long label, float* out_data) {
    if (!index || !out_data) return -1;
    auto* impl = static_cast<HNSWIndexImpl*>(index);

    try {
        std::unique_lock<std::mutex> lock_table(impl->alg_hnsw->label_lookup_lock);
        auto search = impl->alg_hnsw->label_lookup_.find(
            static_cast<hnswlib::labeltype>(label));
        if (search == impl->alg_hnsw->label_lookup_.end()) {
            return -2;
        }
        hnswlib::tableint internal_id = search->second;
        if (impl->alg_hnsw->isMarkedDeleted(internal_id)) {
            return -3;  // label is mark-deleted, do not return data
        }
        lock_table.unlock();

        char* data_ptr = impl->alg_hnsw->getDataByInternalId(internal_id);
        std::memcpy(out_data, data_ptr, impl->dimension * sizeof(float));
        return 0;
    } catch (...) {
        return -1;
    }
}


int hnsw_get_all_labels(HNSWIndex index, unsigned long long* out_labels, int max_count) {
    if (!index || !out_labels) return -1;
    auto* impl = static_cast<HNSWIndexImpl*>(index);

    try {
        std::unique_lock<std::mutex> lock_table(impl->alg_hnsw->label_lookup_lock);
        int count = 0;
        for (auto& [label, internal_id] : impl->alg_hnsw->label_lookup_) {
            if (count >= max_count) break;
            out_labels[count++] = static_cast<unsigned long long>(label);
        }
        return count;
    } catch (...) {
        return -1;
    }
}

int hnsw_mark_deleted(HNSWIndex index, unsigned long long label) {
    if (!index) return -1;
    auto* impl = static_cast<HNSWIndexImpl*>(index);

    try {
        std::unique_lock<std::mutex> lock_table(impl->alg_hnsw->label_lookup_lock);
        auto search = impl->alg_hnsw->label_lookup_.find(
            static_cast<hnswlib::labeltype>(label));
        if (search == impl->alg_hnsw->label_lookup_.end()) {
            return -1;
        }
        hnswlib::tableint internal_id = search->second;
        if (impl->alg_hnsw->isMarkedDeleted(internal_id)) {
            return 1;  // already deleted: idempotent, caller must not decrement counts
        }
        lock_table.unlock();

        impl->alg_hnsw->markDelete(static_cast<hnswlib::labeltype>(label));
        return 0;  // just marked: caller should decrement counts
    } catch (...) {
        return -1;
    }
}


int hnsw_is_label_deleted(HNSWIndex index, unsigned long long label) {
    if (!index) return -1;
    auto* impl = static_cast<HNSWIndexImpl*>(index);

    try {
        std::unique_lock<std::mutex> lock_table(impl->alg_hnsw->label_lookup_lock);
        auto search = impl->alg_hnsw->label_lookup_.find(
            static_cast<hnswlib::labeltype>(label));
        if (search == impl->alg_hnsw->label_lookup_.end()) {
            return -1;
        }
        hnswlib::tableint internal_id = search->second;
        return impl->alg_hnsw->isMarkedDeleted(internal_id) ? 1 : 0;
    } catch (...) {
        return -1;
    }
}


// --- Collection Manager ---

static std::unique_ptr<Rebuilder> g_rebuilder;

HNSWCollection collection_create(
    const char* name,
    const char* space_name,
    int dim,
    int M,
    int ef_construction,
    int ef_search,
    long long initial_sealed_capacity)
{
    try {
        CollectionConfig cfg;
        cfg.name = name ? name : "";
        cfg.space_name = space_name;
        cfg.dimension = dim;
        cfg.M = M;
        cfg.ef_construction = ef_construction;
        cfg.ef_search = ef_search;
        cfg.initial_sealed_capacity = static_cast<int64_t>(initial_sealed_capacity);

        auto* col = new Collection(cfg);
        return static_cast<HNSWCollection>(col);
    } catch (...) {
        return nullptr;
    }
}


void collection_destroy(HNSWCollection col) {
    if (!col) return;
    delete static_cast<Collection*>(col);
}


int collection_add_point(HNSWCollection col, const float* data, unsigned long long label) {
    if (!col) return -1;
    auto* c = static_cast<Collection*>(col);
    int rc = c->addPoint(data, label);

    // Check rebuild trigger after successful add
    if (rc == 0 && g_rebuilder && c->needsRebuild()) {
        c->metrics().rebuild_in_progress.store(true, std::memory_order_relaxed);
        RebuildPriority prio = (c->bufferFillRatio() >= 0.90)
            ? RebuildPriority::URGENT
            : RebuildPriority::NORMAL;
        g_rebuilder->submit(c, prio);
    }
    return rc;
}


int collection_delete_point(HNSWCollection col, unsigned long long label) {
    if (!col) return -1;
    auto* c = static_cast<Collection*>(col);
    int rc = c->deletePoint(label);

    // Check rebuild trigger after delete
    if (rc == 0 && g_rebuilder && c->needsRebuild()) {
        c->metrics().rebuild_in_progress.store(true, std::memory_order_relaxed);
        g_rebuilder->submit(c, RebuildPriority::NORMAL);
    }
    return rc;
}


int collection_update_point(HNSWCollection col, const float* data, unsigned long long label) {
    if (!col) return -1;
    return static_cast<Collection*>(col)->updatePoint(data, label);
}


int collection_search(
    HNSWCollection col,
    const float* query,
    int k,
    unsigned long long* out_labels,
    float* out_distances)
{
    if (!col) return -1;
    return static_cast<Collection*>(col)->search(query, k, out_labels, out_distances);
}


char* collection_get_stats(HNSWCollection col) {
    if (!col) return nullptr;
    auto* c = static_cast<Collection*>(col);
    auto& m = c->metrics();

    char buf[512];
    std::snprintf(buf, sizeof(buf),
        "{\"sealed_count\":%lld,"
        "\"buffer_count\":%lld,"
        "\"tombstone_count\":%lld,"
        "\"degradation_pct\":%.2f,"
        "\"buffer_fill_pct\":%.2f,"
        "\"rebuild_count\":%lld,"
        "\"last_rebuild_ms\":%lld,"
        "\"is_rebuilding\":%s}",
        static_cast<long long>(m.sealed_count.load(std::memory_order_relaxed)),
        static_cast<long long>(m.buffer_count.load(std::memory_order_relaxed)),
        static_cast<long long>(m.tombstone_count.load(std::memory_order_relaxed)),
        c->degradationRatio() * 100.0,
        c->bufferFillRatio() * 100.0,
        static_cast<long long>(m.rebuild_count.load(std::memory_order_relaxed)),
        static_cast<long long>(m.last_rebuild_ms.load(std::memory_order_relaxed)),
        m.rebuild_in_progress.load(std::memory_order_relaxed) ? "true" : "false"
    );

    size_t len = std::strlen(buf) + 1;
    char* result = static_cast<char*>(std::malloc(len));
    if (result) {
        std::memcpy(result, buf, len);
    }
    return result;
}


void rebuilder_init(int num_workers) {
    if (!g_rebuilder) {
        int n = num_workers;
        if (n <= 0) {
            n = std::max(2, static_cast<int>(std::thread::hardware_concurrency()) / 4);
        }
        g_rebuilder = std::make_unique<Rebuilder>(n);
    }
}


void rebuilder_stop(void) {
    if (g_rebuilder) {
        g_rebuilder->stop();
        g_rebuilder.reset();
    }
}

} // extern "C"
