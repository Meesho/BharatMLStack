# Layout1 vs Layout2 Compression — Catalog Use Case

## Executive Summary

✅ **Layout2 is better than or equal to Layout1** in **21/66** catalog scenarios (31.8%).

## Test Results by Data Type

### DataTypeInt32Vector

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/vector_int32 50% defaults | 1 | 50.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32 80% defaults | 1 | 80.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32_lifetime 50% def... | 1 | 50.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32_lifetime 80% def... | 1 | 80.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32_lifetime_v2 50% ... | 1 | 50.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32_lifetime_v2 80% ... | 1 | 80.0% | -6.25% | -6.25% ⚠️ |

### DataTypeFP16Vector

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/embeddings_v2_fp16 50% defaults | 3 | 50.0% | 29.17% | 29.17% ✅ |
| catalog/embeddings_v2_fp16 80% defaults | 3 | 80.0% | 62.50% | 62.50% ✅ |
| catalog/embedding_stcg_fp16 50% defaults | 3 | 50.0% | 29.17% | 29.17% ✅ |
| catalog/embedding_stcg_fp16 80% defaults | 3 | 80.0% | 62.50% | 62.50% ✅ |
| catalog/merlin_embeddings_fp16 50% de... | 2 | 50.0% | 43.75% | 43.75% ✅ |
| catalog/merlin_embeddings_fp16 80% de... | 2 | 80.0% | 43.75% | 43.75% ✅ |
| catalog/embeddings_fp16 50% defaults | 1 | 50.0% | -12.50% | -12.50% ⚠️ |
| catalog/embeddings_fp16 80% defaults | 1 | 80.0% | -12.50% | -12.50% ⚠️ |

### DataTypeFP16

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/raw_fp16_7d_1d_1am 50% defaults | 1 | 50.0% | -50.00% | -50.00% ⚠️ |
| catalog/raw_fp16_7d_1d_1am 80% defaults | 1 | 80.0% | -50.00% | -50.00% ⚠️ |
| catalog/derived_fp16 50% defaults | 4 | 50.0% | 37.50% | 37.50% ✅ |
| catalog/derived_fp16 80% defaults | 4 | 80.0% | 62.50% | 62.50% ✅ |
| catalog/raw_fp16_1d_30m_12am 50% defa... | 1 | 50.0% | -50.00% | -50.00% ⚠️ |
| catalog/raw_fp16_1d_30m_12am 80% defa... | 1 | 80.0% | -50.00% | -50.00% ⚠️ |

### DataTypeFP32

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/rt_raw_ads_demand_attributes_... | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ads_demand_attributes_... | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_3_fp32 50% defaults | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_3_fp32 80% defaults | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_4_fp32 50% defaults | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_4_fp32 80% defaults | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_attributes_v1_fp32 ... | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_attributes_v1_fp32 ... | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_ads_fp32 50% defaults | 3 | 50.0% | 25.00% | 25.00% ✅ |
| catalog/derived_ads_fp32 80% defaults | 3 | 80.0% | 58.33% | 58.33% ✅ |
| catalog/organic__derived_fp32 50% def... | 11 | 50.0% | 40.91% | 40.91% ✅ |
| catalog/organic__derived_fp32 80% def... | 11 | 80.0% | 68.18% | 57.58% ✅ |
| catalog/derived_fp32 50% defaults | 46 | 50.0% | 46.74% | 27.41% ✅ |
| catalog/derived_fp32 80% defaults | 46 | 80.0% | 75.00% | 44.58% ✅ |
| catalog/derived_2_fp32 50% defaults | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_2_fp32 80% defaults | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_cpc_value_fp32 50% ... | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_cpc_value_fp32 80% ... | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_batch_attributes_fp... | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_batch_attributes_fp... | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_gmv_max_attributes_... | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_gmv_max_attributes_... | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_fp32 0% defaults (all... | 46 | 0.0% | -3.26% | -3.26% ⚠️ |
| catalog/derived_fp32 100% defaults | 46 | 100.0% | 96.74% | 57.14% ✅ |

### DataTypeString

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/properties_string 50% defaults | 1 | 50.0% | -14.29% | -14.29% ⚠️ |
| catalog/properties_string 80% defaults | 1 | 80.0% | -14.29% | -14.29% ⚠️ |
| catalog/derived_string 50% defaults | 4 | 50.0% | 17.65% | 17.65% ✅ |
| catalog/derived_string 80% defaults | 4 | 80.0% | 38.46% | 38.46% ✅ |
| catalog/properties_2_string 50% defaults | 1 | 50.0% | -14.29% | -14.29% ⚠️ |
| catalog/properties_2_string 80% defaults | 1 | 80.0% | -14.29% | -14.29% ⚠️ |
| catalog/rt_raw_is_live_on_ad_string 5... | 1 | 50.0% | -14.29% | -14.29% ⚠️ |
| catalog/rt_raw_is_live_on_ad_string 8... | 1 | 80.0% | -14.29% | -14.29% ⚠️ |
| catalog/realtime_string 50% defaults | 1 | 50.0% | -14.29% | -14.29% ⚠️ |
| catalog/realtime_string 80% defaults | 1 | 80.0% | -14.29% | -14.29% ⚠️ |

### DataTypeInt64

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/realtime_int64_1 50% defaults | 1 | 50.0% | -12.50% | -12.50% ⚠️ |
| catalog/realtime_int64_1 80% defaults | 1 | 80.0% | -12.50% | -12.50% ⚠️ |
| catalog/realtime_int64 50% defaults | 1 | 50.0% | -12.50% | -12.50% ⚠️ |
| catalog/realtime_int64 80% defaults | 1 | 80.0% | -12.50% | -12.50% ⚠️ |

### DataTypeFP32Vector

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/embedding_ca_fp32 50% defaults | 1 | 50.0% | -6.25% | -6.25% ⚠️ |
| catalog/embedding_ca_fp32 80% defaults | 1 | 80.0% | -6.25% | -6.25% ⚠️ |

### DataTypeInt32

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/rt_raw_ad_attributes_int32 50... | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_attributes_int32 80... | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_int32 50% defaults | 14 | 50.0% | 46.43% | 46.43% ✅ |
| catalog/derived_int32 80% defaults | 14 | 80.0% | 75.00% | 66.67% ✅ |

### DataTypeUint64

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/raw_uint64 50% defaults | 3 | 50.0% | 29.17% | 29.17% ✅ |
| catalog/raw_uint64 80% defaults | 3 | 80.0% | 62.50% | 62.50% ✅ |

## All Results Summary (Catalog Use Case)

| Test Name | Data Type | Features | Defaults | Original Δ | Compressed Δ |
|-----------|-----------|----------|-----------|------------|-------------|
| catalog/vector_int32 50% defaults | DataTypeInt32Vector | 1 | 50.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32 80% defaults | DataTypeInt32Vector | 1 | 80.0% | -6.25% | -6.25% ⚠️ |
| catalog/embeddings_v2_fp16 50% defaults | DataTypeFP16Vector | 3 | 50.0% | 29.17% | 29.17% ✅ |
| catalog/embeddings_v2_fp16 80% defaults | DataTypeFP16Vector | 3 | 80.0% | 62.50% | 62.50% ✅ |
| catalog/embedding_stcg_fp16 50% defaults | DataTypeFP16Vector | 3 | 50.0% | 29.17% | 29.17% ✅ |
| catalog/embedding_stcg_fp16 80% defaults | DataTypeFP16Vector | 3 | 80.0% | 62.50% | 62.50% ✅ |
| catalog/raw_fp16_7d_1d_1am 50% defaults | DataTypeFP16 | 1 | 50.0% | -50.00% | -50.00% ⚠️ |
| catalog/raw_fp16_7d_1d_1am 80% defaults | DataTypeFP16 | 1 | 80.0% | -50.00% | -50.00% ⚠️ |
| catalog/rt_raw_ads_demand_attributes_fp32 ... | DataTypeFP32 | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ads_demand_attributes_fp32 ... | DataTypeFP32 | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_3_fp32 50% defaults | DataTypeFP32 | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_3_fp32 80% defaults | DataTypeFP32 | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_fp16 50% defaults | DataTypeFP16 | 4 | 50.0% | 37.50% | 37.50% ✅ |
| catalog/derived_fp16 80% defaults | DataTypeFP16 | 4 | 80.0% | 62.50% | 62.50% ✅ |
| catalog/properties_string 50% defaults | DataTypeString | 1 | 50.0% | -14.29% | -14.29% ⚠️ |
| catalog/properties_string 80% defaults | DataTypeString | 1 | 80.0% | -14.29% | -14.29% ⚠️ |
| catalog/realtime_int64_1 50% defaults | DataTypeInt64 | 1 | 50.0% | -12.50% | -12.50% ⚠️ |
| catalog/realtime_int64_1 80% defaults | DataTypeInt64 | 1 | 80.0% | -12.50% | -12.50% ⚠️ |
| catalog/derived_4_fp32 50% defaults | DataTypeFP32 | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_4_fp32 80% defaults | DataTypeFP32 | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_attributes_v1_fp32 50% d... | DataTypeFP32 | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_attributes_v1_fp32 80% d... | DataTypeFP32 | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_ads_fp32 50% defaults | DataTypeFP32 | 3 | 50.0% | 25.00% | 25.00% ✅ |
| catalog/derived_ads_fp32 80% defaults | DataTypeFP32 | 3 | 80.0% | 58.33% | 58.33% ✅ |
| catalog/embedding_ca_fp32 50% defaults | DataTypeFP32Vector | 1 | 50.0% | -6.25% | -6.25% ⚠️ |
| catalog/embedding_ca_fp32 80% defaults | DataTypeFP32Vector | 1 | 80.0% | -6.25% | -6.25% ⚠️ |
| catalog/organic__derived_fp32 50% defaults | DataTypeFP32 | 11 | 50.0% | 40.91% | 40.91% ✅ |
| catalog/organic__derived_fp32 80% defaults | DataTypeFP32 | 11 | 80.0% | 68.18% | 57.58% ✅ |
| catalog/derived_fp32 50% defaults | DataTypeFP32 | 46 | 50.0% | 46.74% | 27.41% ✅ |
| catalog/derived_fp32 80% defaults | DataTypeFP32 | 46 | 80.0% | 75.00% | 44.58% ✅ |
| catalog/raw_fp16_1d_30m_12am 50% defaults | DataTypeFP16 | 1 | 50.0% | -50.00% | -50.00% ⚠️ |
| catalog/raw_fp16_1d_30m_12am 80% defaults | DataTypeFP16 | 1 | 80.0% | -50.00% | -50.00% ⚠️ |
| catalog/derived_string 50% defaults | DataTypeString | 4 | 50.0% | 17.65% | 17.65% ✅ |
| catalog/derived_string 80% defaults | DataTypeString | 4 | 80.0% | 38.46% | 38.46% ✅ |
| catalog/properties_2_string 50% defaults | DataTypeString | 1 | 50.0% | -14.29% | -14.29% ⚠️ |
| catalog/properties_2_string 80% defaults | DataTypeString | 1 | 80.0% | -14.29% | -14.29% ⚠️ |
| catalog/derived_2_fp32 50% defaults | DataTypeFP32 | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_2_fp32 80% defaults | DataTypeFP32 | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/realtime_int64 50% defaults | DataTypeInt64 | 1 | 50.0% | -12.50% | -12.50% ⚠️ |
| catalog/realtime_int64 80% defaults | DataTypeInt64 | 1 | 80.0% | -12.50% | -12.50% ⚠️ |
| catalog/merlin_embeddings_fp16 50% defaults | DataTypeFP16Vector | 2 | 50.0% | 43.75% | 43.75% ✅ |
| catalog/merlin_embeddings_fp16 80% defaults | DataTypeFP16Vector | 2 | 80.0% | 43.75% | 43.75% ✅ |
| catalog/rt_raw_ad_attributes_int32 50% def... | DataTypeInt32 | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_attributes_int32 80% def... | DataTypeInt32 | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_cpc_value_fp32 50% defaults | DataTypeFP32 | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_cpc_value_fp32 80% defaults | DataTypeFP32 | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/raw_uint64 50% defaults | DataTypeUint64 | 3 | 50.0% | 29.17% | 29.17% ✅ |
| catalog/raw_uint64 80% defaults | DataTypeUint64 | 3 | 80.0% | 62.50% | 62.50% ✅ |
| catalog/rt_raw_ad_batch_attributes_fp32 50... | DataTypeFP32 | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_batch_attributes_fp32 80... | DataTypeFP32 | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/embeddings_fp16 50% defaults | DataTypeFP16Vector | 1 | 50.0% | -12.50% | -12.50% ⚠️ |
| catalog/embeddings_fp16 80% defaults | DataTypeFP16Vector | 1 | 80.0% | -12.50% | -12.50% ⚠️ |
| catalog/vector_int32_lifetime 50% defaults | DataTypeInt32Vector | 1 | 50.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32_lifetime 80% defaults | DataTypeInt32Vector | 1 | 80.0% | -6.25% | -6.25% ⚠️ |
| catalog/derived_int32 50% defaults | DataTypeInt32 | 14 | 50.0% | 46.43% | 46.43% ✅ |
| catalog/derived_int32 80% defaults | DataTypeInt32 | 14 | 80.0% | 75.00% | 66.67% ✅ |
| catalog/vector_int32_lifetime_v2 50% defaults | DataTypeInt32Vector | 1 | 50.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32_lifetime_v2 80% defaults | DataTypeInt32Vector | 1 | 80.0% | -6.25% | -6.25% ⚠️ |
| catalog/rt_raw_is_live_on_ad_string 50% de... | DataTypeString | 1 | 50.0% | -14.29% | -14.29% ⚠️ |
| catalog/rt_raw_is_live_on_ad_string 80% de... | DataTypeString | 1 | 80.0% | -14.29% | -14.29% ⚠️ |
| catalog/rt_raw_ad_gmv_max_attributes_fp32 ... | DataTypeFP32 | 1 | 50.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_gmv_max_attributes_fp32 ... | DataTypeFP32 | 1 | 80.0% | -25.00% | -25.00% ⚠️ |
| catalog/realtime_string 50% defaults | DataTypeString | 1 | 50.0% | -14.29% | -14.29% ⚠️ |
| catalog/realtime_string 80% defaults | DataTypeString | 1 | 80.0% | -14.29% | -14.29% ⚠️ |
| catalog/derived_fp32 0% defaults (all non-... | DataTypeFP32 | 46 | 0.0% | -3.26% | -3.26% ⚠️ |
| catalog/derived_fp32 100% defaults | DataTypeFP32 | 46 | 100.0% | 96.74% | 57.14% ✅ |

## Key Findings (Catalog Use Case)

- **Use case:** entityLabel=catalog with the defined feature groups (scalars and vectors).
- Layout2 uses bitmap-based storage; bitmap present is the 72nd bit (10th byte bit 0). Bool scalar (derived_bool) is layout-1 only and excluded from layout-2 comparison.
- With 0% defaults, Layout2 has small bitmap overhead; with 50%/80%/100% defaults, Layout2 reduces size.

## Test Implementation

Tests: `online-feature-store/internal/data/blocks/layout_comparison_test.go`

```bash
go test ./internal/data/blocks -run TestLayout1VsLayout2Compression -v
go test ./internal/data/blocks -run TestLayout2BitmapOptimization -v
```

**Generated:** 2026-02-25 14:32:23
