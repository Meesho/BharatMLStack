#pragma once

#ifdef __cplusplus
extern "C" {
#endif

typedef void* HNSWIndex;
typedef void* HNSWCollection;

// --- Single-Index API ---

HNSWIndex hnsw_create_index(
    const char* space_name,
    int dim,
    int max_elements,
    int M,
    int ef_construction,
    int random_seed
);

void hnsw_delete_index(HNSWIndex index);

int hnsw_add_point(HNSWIndex index, const float* data, unsigned long long label);

int hnsw_search_knn(
    HNSWIndex index,
    const float* query_data,
    int k,
    unsigned long long* labels,
    float* distances
);

int hnsw_search_knn_filtered(
    HNSWIndex index,
    const float* query_data,
    int k,
    unsigned long long* labels,
    float* distances,
    const unsigned long long* tombstone_ids,
    int tombstone_count
);

void hnsw_set_ef(HNSWIndex index, int ef);

int hnsw_get_current_count(HNSWIndex index);

int hnsw_get_max_elements(HNSWIndex index);

int hnsw_get_dimension(HNSWIndex index);

int hnsw_save_index(HNSWIndex index, const char* path);

HNSWIndex hnsw_load_index(
    const char* path,
    const char* space_name,
    int dim,
    int max_elements
);

int hnsw_get_data_by_label(HNSWIndex index, unsigned long long label, float* out_data);
// Returns 0 on success, -1 on error, -2 if label not in index, -3 if label is mark-deleted.

int hnsw_get_all_labels(HNSWIndex index, unsigned long long* out_labels, int max_count);

// Returns 0 if label was present and is now mark-deleted (caller should decrement counts);
// 1 if label was already mark-deleted (idempotent, do not decrement);
// -1 if label not in index.
int hnsw_mark_deleted(HNSWIndex index, unsigned long long label);

// Returns 1 if label is mark-deleted, 0 if present and not deleted, -1 if not in index.
int hnsw_is_label_deleted(HNSWIndex index, unsigned long long label);

// --- Collection API ---

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
void rebuilder_stop(void);

#ifdef __cplusplus
}
#endif
