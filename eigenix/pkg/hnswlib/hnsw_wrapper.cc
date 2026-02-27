#include "hnsw_wrapper.h"
#include "hnswlib.h"
#include <memory>
#include <stdexcept>

struct HNSWIndexImpl {
    hnswlib::SpaceInterface<float>* space;
    hnswlib::HierarchicalNSW<float>* alg_hnsw;
    std::string space_name;
    int dimension;
    
    HNSWIndexImpl(const std::string& space_name, int dim) 
        : space_name(space_name), dimension(dim), alg_hnsw(nullptr) {
        
        if (space_name == "l2") {
            space = new hnswlib::L2Space(dim);
        } else if (space_name == "ip") {
            space = new hnswlib::InnerProductSpace(dim);
        } else if (space_name == "cosine") {
            space = new hnswlib::InnerProductSpace(dim);
        } else {
            throw std::runtime_error("Unknown space name");
        }
    }
    
    ~HNSWIndexImpl() {
        if (alg_hnsw) delete alg_hnsw;
        if (space) delete space;
    }
};

extern "C" {

HNSWIndex hnsw_create_index(const char* space_name, int dim, int max_elements, int M, int ef_construction, int random_seed, int allow_replace_deleted) {
    try {
        auto* impl = new HNSWIndexImpl(space_name, dim);
        bool allow_replace = (allow_replace_deleted != 0);
        impl->alg_hnsw = new hnswlib::HierarchicalNSW<float>(impl->space, max_elements, M, ef_construction, random_seed, allow_replace);
        return static_cast<HNSWIndex>(impl);
    } catch (...) {
        return nullptr;
    }
}

void hnsw_delete_index(HNSWIndex index) {
    if (index) {
        delete static_cast<HNSWIndexImpl*>(index);
    }
}

int hnsw_add_point(HNSWIndex index, const float* data, unsigned long long label) {
    if (!index || !data) return -1;
    
    try {
        auto* impl = static_cast<HNSWIndexImpl*>(index);
        if (!impl->alg_hnsw) return -1;
        
        // For cosine similarity, we need to normalize the vector
        if (impl->space_name == "cosine") {
            std::vector<float> normalized_data(impl->dimension);
            float norm = 0.0f;
            for (int i = 0; i < impl->dimension; i++) {
                norm += data[i] * data[i];
            }
            norm = std::sqrt(norm);
            
            if (norm > 0) {
                for (int i = 0; i < impl->dimension; i++) {
                    normalized_data[i] = data[i] / norm;
                }
                impl->alg_hnsw->addPoint(normalized_data.data(), label);
            } else {
                return -1; // Cannot normalize zero vector
            }
        } else {
            impl->alg_hnsw->addPoint(data, label);
        }
        return 0;
    } catch (...) {
        return -1;
    }
}

int hnsw_search_knn(HNSWIndex index, const float* query_data, int k, unsigned long long* labels, float* distances) {
    if (!index || !query_data || !labels || !distances) return -1;
    
    try {
        auto* impl = static_cast<HNSWIndexImpl*>(index);
        if (!impl->alg_hnsw) return -1;
        
        std::vector<float> query_vector(impl->dimension);
        
        // For cosine similarity, normalize the query vector
        if (impl->space_name == "cosine") {
            float norm = 0.0f;
            for (int i = 0; i < impl->dimension; i++) {
                norm += query_data[i] * query_data[i];
            }
            norm = std::sqrt(norm);
            
            if (norm > 0) {
                for (int i = 0; i < impl->dimension; i++) {
                    query_vector[i] = query_data[i] / norm;
                }
            } else {
                return -1; // Cannot normalize zero vector
            }
        } else {
            for (int i = 0; i < impl->dimension; i++) {
                query_vector[i] = query_data[i];
            }
        }
        
        auto result = impl->alg_hnsw->searchKnn(query_vector.data(), k);
        
        int count = 0;
        while (!result.empty() && count < k) {
            distances[k - 1 - count] = result.top().first;
            labels[k - 1 - count] = result.top().second;
            result.pop();
            count++;
        }
        
        return count;
    } catch (...) {
        return -1;
    }
}

void hnsw_set_ef(HNSWIndex index, int ef) {
    if (!index) return;
    
    auto* impl = static_cast<HNSWIndexImpl*>(index);
    if (impl->alg_hnsw) {
        impl->alg_hnsw->ef_ = ef;
    }
}

int hnsw_get_current_count(HNSWIndex index) {
    if (!index) return -1;
    
    auto* impl = static_cast<HNSWIndexImpl*>(index);
    if (impl->alg_hnsw) {
        return impl->alg_hnsw->cur_element_count;
    }
    return -1;
}

int hnsw_save_index(HNSWIndex index, const char* path) {
    if (!index || !path) return -1;
    
    try {
        auto* impl = static_cast<HNSWIndexImpl*>(index);
        if (impl->alg_hnsw) {
            impl->alg_hnsw->saveIndex(path);
            return 0;
        }
        return -1;
    } catch (...) {
        return -1;
    }
}

HNSWIndex hnsw_load_index(const char* space_name, int dim, const char* path, int max_elements) {
    if (!space_name || !path) return nullptr;
    
    try {
        auto* impl = new HNSWIndexImpl(space_name, dim);
        impl->alg_hnsw = new hnswlib::HierarchicalNSW<float>(impl->space, path, false, max_elements);
        return static_cast<HNSWIndex>(impl);
    } catch (...) {
        return nullptr;
    }
}

void hnsw_clear(HNSWIndex index) {
    if (!index) return;
    
    auto* impl = static_cast<HNSWIndexImpl*>(index);
    if (impl->alg_hnsw) {
        impl->alg_hnsw->clear();
    }
}

int hnsw_resize_index(HNSWIndex index, int new_max_elements) {
    if (!index) return -1;
    
    try {
        auto* impl = static_cast<HNSWIndexImpl*>(index);
        if (impl->alg_hnsw) {
            impl->alg_hnsw->resizeIndex(new_max_elements);
            return 0;
        }
        return -1;
    } catch (...) {
        return -1;
    }
}

int hnsw_update_point(HNSWIndex index, const float* data, unsigned long long label) {
    if (!index || !data) return -1;
    
    try {
        auto* impl = static_cast<HNSWIndexImpl*>(index);
        if (!impl->alg_hnsw) return -1;
        
        // Find the internal ID for the label
        hnswlib::tableint internal_id;
        {
            std::unique_lock<std::mutex> lock(impl->alg_hnsw->label_lookup_lock);
            auto search = impl->alg_hnsw->label_lookup_.find(label);
            if (search == impl->alg_hnsw->label_lookup_.end()) {
                return -2; // Label not found
            }
            internal_id = search->second;
        }
        
        // Normalize for cosine similarity
        if (impl->space_name == "cosine") {
            std::vector<float> normalized_data(impl->dimension);
            float norm = 0.0f;
            for (int i = 0; i < impl->dimension; i++) {
                norm += data[i] * data[i];
            }
            norm = std::sqrt(norm);
            
            if (norm > 0) {
                for (int i = 0; i < impl->dimension; i++) {
                    normalized_data[i] = data[i] / norm;
                }
                impl->alg_hnsw->updatePoint(normalized_data.data(), internal_id, 1.0);
            } else {
                return -1; // Cannot normalize zero vector
            }
        } else {
            impl->alg_hnsw->updatePoint(data, internal_id, 1.0);
        }
        return 0;
    } catch (...) {
        return -1;
    }
}

int hnsw_mark_deleted(HNSWIndex index, unsigned long long label) {
    if (!index) return -1;
    
    try {
        auto* impl = static_cast<HNSWIndexImpl*>(index);
        if (impl->alg_hnsw) {
            impl->alg_hnsw->markDelete(label);
            return 0;
        }
        return -1;
    } catch (...) {
        return -1;
    }
}

int hnsw_unmark_deleted(HNSWIndex index, unsigned long long label) {
    if (!index) return -1;
    
    try {
        auto* impl = static_cast<HNSWIndexImpl*>(index);
        if (impl->alg_hnsw) {
            impl->alg_hnsw->unmarkDelete(label);
            return 0;
        }
        return -1;
    } catch (...) {
        return -1;
    }
}

int hnsw_is_marked_deleted(HNSWIndex index, unsigned long long label) {
    if (!index) return -1;
    
    try {
        auto* impl = static_cast<HNSWIndexImpl*>(index);
        if (!impl->alg_hnsw) return -1;
        
        // Find the internal ID for the label
        std::unique_lock<std::mutex> lock(impl->alg_hnsw->label_lookup_lock);
        auto search = impl->alg_hnsw->label_lookup_.find(label);
        if (search == impl->alg_hnsw->label_lookup_.end()) {
            return -2; // Label not found
        }
        hnswlib::tableint internal_id = search->second;
        lock.unlock();
        
        return impl->alg_hnsw->isMarkedDeleted(internal_id) ? 1 : 0;
    } catch (...) {
        return -1;
    }
}

int hnsw_get_deleted_count(HNSWIndex index) {
    if (!index) return -1;
    
    auto* impl = static_cast<HNSWIndexImpl*>(index);
    if (impl->alg_hnsw) {
        return impl->alg_hnsw->num_deleted_;
    }
    return -1;
}

int hnsw_get_max_elements(HNSWIndex index) {
    if (!index) return -1;
    
    auto* impl = static_cast<HNSWIndexImpl*>(index);
    if (impl->alg_hnsw) {
        return impl->alg_hnsw->max_elements_;
    }
    return -1;
}

} // extern "C"
