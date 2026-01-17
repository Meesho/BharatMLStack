#ifndef HNSW_WRAPPER_H
#define HNSW_WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

// Opaque pointer to the C++ HNSW index
typedef void* HNSWIndex;

// Create a new HNSW index
HNSWIndex hnsw_create_index(const char* space_name, int dim, int max_elements, int M, int ef_construction, int random_seed, int allow_replace_deleted);

// Delete the HNSW index
void hnsw_delete_index(HNSWIndex index);

// Add a point to the index
int hnsw_add_point(HNSWIndex index, const float* data, unsigned long long label);

// Search for k nearest neighbors
int hnsw_search_knn(HNSWIndex index, const float* query_data, int k, unsigned long long* labels, float* distances);

// Set ef parameter for search
void hnsw_set_ef(HNSWIndex index, int ef);

// Get current element count
int hnsw_get_current_count(HNSWIndex index);

// Save index to file
int hnsw_save_index(HNSWIndex index, const char* path);

// Load index from file
HNSWIndex hnsw_load_index(const char* space_name, int dim, const char* path, int max_elements);

// Clear all data from the index
void hnsw_clear(HNSWIndex index);

// Resize the index to new maximum capacity
int hnsw_resize_index(HNSWIndex index, int new_max_elements);

// Update an existing point's vector
int hnsw_update_point(HNSWIndex index, const float* data, unsigned long long label);

// Mark a point as deleted (soft delete)
int hnsw_mark_deleted(HNSWIndex index, unsigned long long label);

// Unmark a deleted point (restore)
int hnsw_unmark_deleted(HNSWIndex index, unsigned long long label);

// Check if a point is marked as deleted
int hnsw_is_marked_deleted(HNSWIndex index, unsigned long long label);

// Get number of deleted elements
int hnsw_get_deleted_count(HNSWIndex index);

// Get maximum capacity
int hnsw_get_max_elements(HNSWIndex index);

#ifdef __cplusplus
}
#endif

#endif // HNSW_WRAPPER_H
