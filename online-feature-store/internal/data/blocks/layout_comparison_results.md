# Layout1 vs Layout2 Compression — Catalog Use Case

## Executive Summary

✅ **Layout2 is better than or equal to Layout1** in **44/63** catalog scenarios (69.8%).

## Test Results by Data Type

### DataTypeInt32Vector

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/vector_int32 0/1 defaults (0%) | 1 | 0.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32 1/1 defaults (100%) | 1 | 100.0% | 93.75% | 92.86% ✅ |
| catalog/vector_int32_lifetime 0/1 def... | 1 | 0.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32_lifetime 1/1 def... | 1 | 100.0% | 93.75% | 92.86% ✅ |
| catalog/vector_int32_lifetime_v2 0/1 ... | 1 | 0.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32_lifetime_v2 1/1 ... | 1 | 100.0% | 93.75% | 92.86% ✅ |

### DataTypeFP16Vector

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/embeddings_v2_fp16 1/2 defaul... | 2 | 50.0% | 43.75% | 43.75% ✅ |
| catalog/embedding_stcg_fp16 0/1 defau... | 1 | 0.0% | -12.50% | -12.50% ⚠️ |
| catalog/embedding_stcg_fp16 1/1 defau... | 1 | 100.0% | 87.50% | 87.50% ✅ |
| catalog/merlin_embeddings_fp16 0/1 de... | 1 | 0.0% | -12.50% | -12.50% ⚠️ |
| catalog/merlin_embeddings_fp16 1/1 de... | 1 | 100.0% | 87.50% | 87.50% ✅ |
| catalog/embeddings_fp16 0/1 defaults ... | 1 | 0.0% | -12.50% | -12.50% ⚠️ |
| catalog/embeddings_fp16 1/1 defaults ... | 1 | 100.0% | 87.50% | 87.50% ✅ |

### DataTypeFP16

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/raw_fp16_7d_1d_1am 0/1 defaul... | 1 | 0.0% | -50.00% | -50.00% ⚠️ |
| catalog/raw_fp16_7d_1d_1am 1/1 defaul... | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/derived_fp16 5/10 defaults (50%) | 10 | 50.0% | 40.00% | 40.00% ✅ |
| catalog/derived_fp16 8/10 defaults (80%) | 10 | 80.0% | 70.00% | 70.00% ✅ |
| catalog/raw_fp16_1d_30m_12am 0/1 defa... | 1 | 0.0% | -50.00% | -50.00% ⚠️ |
| catalog/raw_fp16_1d_30m_12am 1/1 defa... | 1 | 100.0% | 50.00% | 50.00% ✅ |

### DataTypeFP32

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/rt_raw_ads_demand_attributes_... | 2 | 50.0% | 37.50% | 37.50% ✅ |
| catalog/derived_3_fp32 0/1 defaults (0%) | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_3_fp32 1/1 defaults (... | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/derived_4_fp32 2/4 defaults (... | 4 | 50.0% | 43.75% | 43.75% ✅ |
| catalog/derived_4_fp32 3/4 defaults (... | 4 | 75.0% | 68.75% | 68.75% ✅ |
| catalog/rt_raw_ad_attributes_v1_fp32 ... | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_attributes_v1_fp32 ... | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/derived_ads_fp32 6/12 default... | 12 | 50.0% | 45.83% | 45.83% ✅ |
| catalog/derived_ads_fp32 9/12 default... | 12 | 75.0% | 70.83% | 69.57% ✅ |
| catalog/organic__derived_fp32 60/121 ... | 121 | 49.6% | 46.28% | 17.72% ✅ |
| catalog/organic__derived_fp32 96/121 ... | 121 | 79.3% | 76.03% | 35.56% ✅ |
| catalog/derived_fp32 457/914 defaults... | 914 | 50.0% | 46.85% | 17.74% ✅ |
| catalog/derived_fp32 731/914 defaults... | 914 | 80.0% | 76.83% | 19.94% ✅ |
| catalog/derived_2_fp32 0/1 defaults (0%) | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_2_fp32 1/1 defaults (... | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/rt_raw_ad_cpc_value_fp32 0/1 ... | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_cpc_value_fp32 1/1 ... | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/rt_raw_ad_batch_attributes_fp... | 3 | 33.3% | 25.00% | 25.00% ✅ |
| catalog/rt_raw_ad_batch_attributes_fp... | 3 | 66.7% | 58.33% | 58.33% ✅ |
| catalog/rt_raw_ad_gmv_max_attributes_... | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_gmv_max_attributes_... | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/derived_fp32 0% defaults (all... | 46 | 0.0% | -3.26% | -3.26% ⚠️ |
| catalog/derived_fp32 100% defaults | 46 | 100.0% | 96.74% | 57.14% ✅ |

### DataTypeString

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/properties_string 0/1 default... | 1 | 0.0% | -14.29% | -14.29% ⚠️ |
| catalog/properties_string 1/1 default... | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/derived_string 9/19 defaults ... | 19 | 47.4% | 17.44% | 2.82% ✅ |
| catalog/derived_string 15/19 defaults... | 19 | 78.9% | 46.55% | 34.04% ✅ |
| catalog/properties_2_string 1/2 defau... | 2 | 50.0% | 11.11% | 11.11% ✅ |
| catalog/rt_raw_is_live_on_ad_string 0... | 1 | 0.0% | -14.29% | -14.29% ⚠️ |
| catalog/rt_raw_is_live_on_ad_string 1... | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/realtime_string 0/1 defaults ... | 1 | 0.0% | -14.29% | -14.29% ⚠️ |
| catalog/realtime_string 1/1 defaults ... | 1 | 100.0% | 50.00% | 50.00% ✅ |

### DataTypeInt64

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/realtime_int64_1 0/1 defaults... | 1 | 0.0% | -12.50% | -12.50% ⚠️ |
| catalog/realtime_int64_1 1/1 defaults... | 1 | 100.0% | 87.50% | 87.50% ✅ |
| catalog/realtime_int64 2/4 defaults (... | 4 | 50.0% | 46.88% | 46.88% ✅ |
| catalog/realtime_int64 3/4 defaults (... | 4 | 75.0% | 71.88% | 71.88% ✅ |

### DataTypeFP32Vector

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/embedding_ca_fp32 0/1 default... | 1 | 0.0% | -6.25% | -6.25% ⚠️ |
| catalog/embedding_ca_fp32 1/1 default... | 1 | 100.0% | 93.75% | 92.86% ✅ |

### DataTypeInt32

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/rt_raw_ad_attributes_int32 2/... | 4 | 50.0% | 43.75% | 43.75% ✅ |
| catalog/rt_raw_ad_attributes_int32 3/... | 4 | 75.0% | 68.75% | 68.75% ✅ |
| catalog/derived_int32 41/83 defaults ... | 83 | 49.4% | 46.08% | 32.71% ✅ |
| catalog/derived_int32 66/83 defaults ... | 83 | 79.5% | 76.20% | 46.62% ✅ |

### DataTypeUint64

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/raw_uint64 3/6 defaults (50%) | 6 | 50.0% | 47.92% | 47.92% ✅ |
| catalog/raw_uint64 4/6 defaults (67%) | 6 | 66.7% | 64.58% | 56.41% ✅ |

## All Results Summary (Catalog Use Case)

| Test Name | Data Type | Features | Defaults | Original Δ | Compressed Δ |
|-----------|-----------|----------|-----------|------------|-------------|
| catalog/vector_int32 0/1 defaults (0%) | DataTypeInt32Vector | 1 | 0.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32 1/1 defaults (100%) | DataTypeInt32Vector | 1 | 100.0% | 93.75% | 92.86% ✅ |
| catalog/embeddings_v2_fp16 1/2 defaults (50%) | DataTypeFP16Vector | 2 | 50.0% | 43.75% | 43.75% ✅ |
| catalog/embedding_stcg_fp16 0/1 defaults (0%) | DataTypeFP16Vector | 1 | 0.0% | -12.50% | -12.50% ⚠️ |
| catalog/embedding_stcg_fp16 1/1 defaults (... | DataTypeFP16Vector | 1 | 100.0% | 87.50% | 87.50% ✅ |
| catalog/raw_fp16_7d_1d_1am 0/1 defaults (0%) | DataTypeFP16 | 1 | 0.0% | -50.00% | -50.00% ⚠️ |
| catalog/raw_fp16_7d_1d_1am 1/1 defaults (1... | DataTypeFP16 | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/rt_raw_ads_demand_attributes_fp32 ... | DataTypeFP32 | 2 | 50.0% | 37.50% | 37.50% ✅ |
| catalog/derived_3_fp32 0/1 defaults (0%) | DataTypeFP32 | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_3_fp32 1/1 defaults (100%) | DataTypeFP32 | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/derived_fp16 5/10 defaults (50%) | DataTypeFP16 | 10 | 50.0% | 40.00% | 40.00% ✅ |
| catalog/derived_fp16 8/10 defaults (80%) | DataTypeFP16 | 10 | 80.0% | 70.00% | 70.00% ✅ |
| catalog/properties_string 0/1 defaults (0%) | DataTypeString | 1 | 0.0% | -14.29% | -14.29% ⚠️ |
| catalog/properties_string 1/1 defaults (100%) | DataTypeString | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/realtime_int64_1 0/1 defaults (0%) | DataTypeInt64 | 1 | 0.0% | -12.50% | -12.50% ⚠️ |
| catalog/realtime_int64_1 1/1 defaults (100%) | DataTypeInt64 | 1 | 100.0% | 87.50% | 87.50% ✅ |
| catalog/derived_4_fp32 2/4 defaults (50%) | DataTypeFP32 | 4 | 50.0% | 43.75% | 43.75% ✅ |
| catalog/derived_4_fp32 3/4 defaults (75%) | DataTypeFP32 | 4 | 75.0% | 68.75% | 68.75% ✅ |
| catalog/rt_raw_ad_attributes_v1_fp32 0/1 d... | DataTypeFP32 | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_attributes_v1_fp32 1/1 d... | DataTypeFP32 | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/derived_ads_fp32 6/12 defaults (50%) | DataTypeFP32 | 12 | 50.0% | 45.83% | 45.83% ✅ |
| catalog/derived_ads_fp32 9/12 defaults (75%) | DataTypeFP32 | 12 | 75.0% | 70.83% | 69.57% ✅ |
| catalog/embedding_ca_fp32 0/1 defaults (0%) | DataTypeFP32Vector | 1 | 0.0% | -6.25% | -6.25% ⚠️ |
| catalog/embedding_ca_fp32 1/1 defaults (100%) | DataTypeFP32Vector | 1 | 100.0% | 93.75% | 92.86% ✅ |
| catalog/organic__derived_fp32 60/121 defau... | DataTypeFP32 | 121 | 49.6% | 46.28% | 17.72% ✅ |
| catalog/organic__derived_fp32 96/121 defau... | DataTypeFP32 | 121 | 79.3% | 76.03% | 35.56% ✅ |
| catalog/derived_fp32 457/914 defaults (50%) | DataTypeFP32 | 914 | 50.0% | 46.85% | 17.74% ✅ |
| catalog/derived_fp32 731/914 defaults (80%) | DataTypeFP32 | 914 | 80.0% | 76.83% | 19.94% ✅ |
| catalog/raw_fp16_1d_30m_12am 0/1 defaults ... | DataTypeFP16 | 1 | 0.0% | -50.00% | -50.00% ⚠️ |
| catalog/raw_fp16_1d_30m_12am 1/1 defaults ... | DataTypeFP16 | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/derived_string 9/19 defaults (47%) | DataTypeString | 19 | 47.4% | 17.44% | 2.82% ✅ |
| catalog/derived_string 15/19 defaults (79%) | DataTypeString | 19 | 78.9% | 46.55% | 34.04% ✅ |
| catalog/properties_2_string 1/2 defaults (... | DataTypeString | 2 | 50.0% | 11.11% | 11.11% ✅ |
| catalog/derived_2_fp32 0/1 defaults (0%) | DataTypeFP32 | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_2_fp32 1/1 defaults (100%) | DataTypeFP32 | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/realtime_int64 2/4 defaults (50%) | DataTypeInt64 | 4 | 50.0% | 46.88% | 46.88% ✅ |
| catalog/realtime_int64 3/4 defaults (75%) | DataTypeInt64 | 4 | 75.0% | 71.88% | 71.88% ✅ |
| catalog/merlin_embeddings_fp16 0/1 default... | DataTypeFP16Vector | 1 | 0.0% | -12.50% | -12.50% ⚠️ |
| catalog/merlin_embeddings_fp16 1/1 default... | DataTypeFP16Vector | 1 | 100.0% | 87.50% | 87.50% ✅ |
| catalog/rt_raw_ad_attributes_int32 2/4 def... | DataTypeInt32 | 4 | 50.0% | 43.75% | 43.75% ✅ |
| catalog/rt_raw_ad_attributes_int32 3/4 def... | DataTypeInt32 | 4 | 75.0% | 68.75% | 68.75% ✅ |
| catalog/rt_raw_ad_cpc_value_fp32 0/1 defau... | DataTypeFP32 | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_cpc_value_fp32 1/1 defau... | DataTypeFP32 | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/raw_uint64 3/6 defaults (50%) | DataTypeUint64 | 6 | 50.0% | 47.92% | 47.92% ✅ |
| catalog/raw_uint64 4/6 defaults (67%) | DataTypeUint64 | 6 | 66.7% | 64.58% | 56.41% ✅ |
| catalog/rt_raw_ad_batch_attributes_fp32 1/... | DataTypeFP32 | 3 | 33.3% | 25.00% | 25.00% ✅ |
| catalog/rt_raw_ad_batch_attributes_fp32 2/... | DataTypeFP32 | 3 | 66.7% | 58.33% | 58.33% ✅ |
| catalog/embeddings_fp16 0/1 defaults (0%) | DataTypeFP16Vector | 1 | 0.0% | -12.50% | -12.50% ⚠️ |
| catalog/embeddings_fp16 1/1 defaults (100%) | DataTypeFP16Vector | 1 | 100.0% | 87.50% | 87.50% ✅ |
| catalog/vector_int32_lifetime 0/1 defaults... | DataTypeInt32Vector | 1 | 0.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32_lifetime 1/1 defaults... | DataTypeInt32Vector | 1 | 100.0% | 93.75% | 92.86% ✅ |
| catalog/derived_int32 41/83 defaults (49%) | DataTypeInt32 | 83 | 49.4% | 46.08% | 32.71% ✅ |
| catalog/derived_int32 66/83 defaults (80%) | DataTypeInt32 | 83 | 79.5% | 76.20% | 46.62% ✅ |
| catalog/vector_int32_lifetime_v2 0/1 defau... | DataTypeInt32Vector | 1 | 0.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32_lifetime_v2 1/1 defau... | DataTypeInt32Vector | 1 | 100.0% | 93.75% | 92.86% ✅ |
| catalog/rt_raw_is_live_on_ad_string 0/1 de... | DataTypeString | 1 | 0.0% | -14.29% | -14.29% ⚠️ |
| catalog/rt_raw_is_live_on_ad_string 1/1 de... | DataTypeString | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/rt_raw_ad_gmv_max_attributes_fp32 ... | DataTypeFP32 | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_gmv_max_attributes_fp32 ... | DataTypeFP32 | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/realtime_string 0/1 defaults (0%) | DataTypeString | 1 | 0.0% | -14.29% | -14.29% ⚠️ |
| catalog/realtime_string 1/1 defaults (100%) | DataTypeString | 1 | 100.0% | 50.00% | 50.00% ✅ |
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

**Generated:** 2026-02-26 10:03:16
