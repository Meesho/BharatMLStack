#pragma once

#ifdef __cplusplus
extern "C" {
#endif

#include <stddef.h>
#include <stdint.h>

typedef struct FlashRingC FlashRingC;

enum {
    FLASHRINGC_OK = 0,
    FLASHRINGC_NOT_FOUND = 1,
    FLASHRINGC_ERROR = 2,
};

FlashRingC* flashringc_open(const char* path, uint64_t capacity_bytes, int num_shards);
void flashringc_close(FlashRingC* c);

int flashringc_get(FlashRingC* c,
                   const char* key, int key_len,
                   char* val_out, int val_cap, int* val_len);

int flashringc_put(FlashRingC* c,
                   const char* key, int key_len,
                   const char* val, int val_len);

int flashringc_delete(FlashRingC* c,
                      const char* key, int key_len);

int flashringc_batch_get(FlashRingC* c,
                         const char** keys, const int* key_lens, int n,
                         char** vals_out, const int* val_caps, int* val_lens,
                         int* statuses);

int flashringc_batch_put(FlashRingC* c,
                         const char** keys, const int* key_lens,
                         const char** vals, const int* val_lens, int n,
                         int* statuses);

#ifdef __cplusplus
}
#endif
