// Vector DB benchmark payload utilities.
// Simulates catalog_id + embedding for add/update, catalog_id only for delete.

#pragma once

#include <cstdint>
#include <string>

#include "mwal/slice.h"
#include "mwal/write_batch.h"

namespace mwal {

// Returns embedding value size in bytes: dim * 4 (fp32) or dim * 8 (fp64).
inline size_t GetVectorPayloadSize(int dim, bool use_fp64) {
  return use_fp64 ? static_cast<size_t>(dim) * 8 : static_cast<size_t>(dim) * 4;
}

// Encodes catalog_id as 8-byte little-endian key for WriteBatch.
inline std::string CatalogIdToKey(uint64_t catalog_id) {
  std::string key(8, '\0');
  for (int i = 0; i < 8; i++) {
    key[static_cast<size_t>(i)] = static_cast<char>(catalog_id & 0xff);
    catalog_id >>= 8;
  }
  return key;
}

// Builds WriteBatch with one Put: key=catalog_id (8B), value=embedding.
// Embedding: deterministic fill (float[i] = i / dim) for reproducibility.
inline void MakeVectorPutBatch(uint64_t catalog_id, int dim, bool use_fp64,
                              WriteBatch* batch) {
  std::string key = CatalogIdToKey(catalog_id);
  std::string value;
  value.resize(GetVectorPayloadSize(dim, use_fp64));

  if (use_fp64) {
    double* ptr = reinterpret_cast<double*>(&value[0]);
    for (int i = 0; i < dim; i++) {
      ptr[i] = static_cast<double>(i) / dim;
    }
  } else {
    float* ptr = reinterpret_cast<float*>(&value[0]);
    for (int i = 0; i < dim; i++) {
      ptr[i] = static_cast<float>(i) / dim;
    }
  }

  batch->Put(key, value);
}

// Builds WriteBatch with one Delete for catalog_id.
inline void MakeVectorDeleteBatch(uint64_t catalog_id, WriteBatch* batch) {
  std::string key = CatalogIdToKey(catalog_id);
  batch->Delete(key);
}

// Raw bytes for log::Writer (wal_write_bench, wal_read_bench).
// Mimics WAL logical record: 1B compression prefix + 12B WriteBatch header +
// 1B Put + varint(8) + 8B key + varint(embed_size) + embedding.
// Simplified: just produce raw float array bytes of the right size.
inline std::string MakeVectorRecordBytes(int dim, bool use_fp64) {
  std::string value;
  value.resize(GetVectorPayloadSize(dim, use_fp64));

  if (use_fp64) {
    double* ptr = reinterpret_cast<double*>(&value[0]);
    for (int i = 0; i < dim; i++) {
      ptr[i] = static_cast<double>(i) / dim;
    }
  } else {
    float* ptr = reinterpret_cast<float*>(&value[0]);
    for (int i = 0; i < dim; i++) {
      ptr[i] = static_cast<float>(i) / dim;
    }
  }

  return value;
}

}  // namespace mwal
