#include "flashringc/cache.h"
#include "flashringc/flashringc.h"

#include <algorithm>
#include <cstring>
#include <thread>
#include <vector>

struct FlashRingC {
    Cache cache;
};

extern "C" {

FlashRingC* flashringc_open(const char* path, uint64_t capacity_bytes, int num_shards) {
    CacheConfig cfg;
    cfg.device_path = path;
    cfg.ring_capacity = capacity_bytes;
    cfg.num_shards = (num_shards > 0) ? static_cast<uint32_t>(num_shards)
                                      : static_cast<uint32_t>(std::max(1, static_cast<int>(std::thread::hardware_concurrency())));
    try {
        Cache cache = Cache::open(cfg);
        return new FlashRingC{std::move(cache)};
    } catch (...) {
        return nullptr;
    }
}

void flashringc_close(FlashRingC* c) {
    delete c;
}

int flashringc_get(FlashRingC* c,
                   const char* key, int key_len,
                   char* val_out, int val_cap, int* val_len) {
    if (!c || !val_len) return FLASHRINGC_ERROR;
    Result result = c->cache.get(std::string_view(key, static_cast<size_t>(key_len)));
    if (result.status == Status::NotFound) {
        *val_len = 0;
        return FLASHRINGC_NOT_FOUND;
    }
    if (result.status != Status::Ok) {
        *val_len = 0;
        return FLASHRINGC_ERROR;
    }
    *val_len = std::min(static_cast<int>(result.value.size()), val_cap);
    if (*val_len > 0 && val_out)
        std::memcpy(val_out, result.value.data(), static_cast<size_t>(*val_len));
    return FLASHRINGC_OK;
}

int flashringc_put(FlashRingC* c,
                   const char* key, int key_len,
                   const char* val, int val_len,
                   uint16_t ttl_seconds) {
    if (!c) return FLASHRINGC_ERROR;
    Result result = c->cache.put(
        std::string_view(key, static_cast<size_t>(key_len)),
        std::string_view(val, static_cast<size_t>(val_len)),
        static_cast<uint32_t>(ttl_seconds));
    return (result.status == Status::Ok) ? FLASHRINGC_OK : FLASHRINGC_ERROR;
}

int flashringc_delete(FlashRingC* c, const char* key, int key_len) {
    if (!c) return FLASHRINGC_ERROR;
    Result result = c->cache.del(std::string_view(key, static_cast<size_t>(key_len)));
    if (result.status == Status::NotFound) return FLASHRINGC_NOT_FOUND;
    return (result.status == Status::Ok) ? FLASHRINGC_OK : FLASHRINGC_ERROR;
}

int flashringc_batch_get(FlashRingC* c,
                         const char** keys, const int* key_lens, int n,
                         char** vals_out, const int* val_caps, int* val_lens,
                         int* statuses) {
    if (!c || n <= 0 || !keys || !key_lens || !vals_out || !val_caps || !val_lens || !statuses)
        return 0;
    std::vector<std::string_view> key_views(static_cast<size_t>(n));
    for (int i = 0; i < n; ++i)
        key_views[static_cast<size_t>(i)] = std::string_view(keys[i], static_cast<size_t>(key_lens[i]));

    std::vector<Result> results = c->cache.batch_get(key_views);
    int ok_count = 0;
    for (int i = 0; i < n; ++i) {
        size_t idx = static_cast<size_t>(i);
        statuses[i] = (results[idx].status == Status::Ok) ? FLASHRINGC_OK
                     : (results[idx].status == Status::NotFound) ? FLASHRINGC_NOT_FOUND
                     : FLASHRINGC_ERROR;
        if (results[idx].status == Status::Ok) {
            val_lens[i] = std::min(static_cast<int>(results[idx].value.size()), val_caps[i]);
            if (val_lens[i] > 0 && vals_out[i])
                std::memcpy(vals_out[i], results[idx].value.data(), static_cast<size_t>(val_lens[i]));
            ok_count++;
        } else {
            val_lens[i] = 0;
        }
    }
    return ok_count;
}

int flashringc_batch_put(FlashRingC* c,
                         const char** keys, const int* key_lens,
                         const char** vals, const int* val_lens, int n,
                         int* statuses) {
    if (!c || n <= 0 || !keys || !key_lens || !vals || !val_lens || !statuses)
        return 0;
    std::vector<KVPair> pairs(static_cast<size_t>(n));
    for (int i = 0; i < n; ++i) {
        size_t idx = static_cast<size_t>(i);
        pairs[idx].key = std::string_view(keys[i], static_cast<size_t>(key_lens[i]));
        pairs[idx].value = std::string_view(vals[i], static_cast<size_t>(val_lens[i]));
    }
    std::vector<Result> results = c->cache.batch_put(pairs);
    int ok_count = 0;
    for (int i = 0; i < n; ++i) {
        size_t idx = static_cast<size_t>(i);
        statuses[i] = (results[idx].status == Status::Ok) ? FLASHRINGC_OK : FLASHRINGC_ERROR;
        if (results[idx].status == Status::Ok) ok_count++;
    }
    return ok_count;
}

}
