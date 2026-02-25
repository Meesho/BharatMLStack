# Layout1 vs Layout2 Compression — Catalog Use Case

## Executive Summary

✅ **Layout2 is better than or equal to Layout1** in **42/65** catalog scenarios (64.6%).

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
| catalog/embeddings_v2_fp16 1/3 defaul... | 3 | 33.3% | 29.17% | 29.17% ✅ |
| catalog/embeddings_v2_fp16 2/3 defaul... | 3 | 66.7% | 62.50% | 62.50% ✅ |
| catalog/embedding_stcg_fp16 1/3 defau... | 3 | 33.3% | 29.17% | 29.17% ✅ |
| catalog/embedding_stcg_fp16 2/3 defau... | 3 | 66.7% | 62.50% | 62.50% ✅ |
| catalog/merlin_embeddings_fp16 1/2 de... | 2 | 50.0% | 43.75% | 43.75% ✅ |
| catalog/embeddings_fp16 0/1 defaults ... | 1 | 0.0% | -12.50% | -12.50% ⚠️ |
| catalog/embeddings_fp16 1/1 defaults ... | 1 | 100.0% | 87.50% | 87.50% ✅ |

### DataTypeFP16

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/raw_fp16_7d_1d_1am 0/1 defaul... | 1 | 0.0% | -50.00% | -50.00% ⚠️ |
| catalog/raw_fp16_7d_1d_1am 1/1 defaul... | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/derived_fp16 2/4 defaults (50%) | 4 | 50.0% | 37.50% | 37.50% ✅ |
| catalog/derived_fp16 3/4 defaults (75%) | 4 | 75.0% | 62.50% | 62.50% ✅ |
| catalog/raw_fp16_1d_30m_12am 0/1 defa... | 1 | 0.0% | -50.00% | -50.00% ⚠️ |
| catalog/raw_fp16_1d_30m_12am 1/1 defa... | 1 | 100.0% | 50.00% | 50.00% ✅ |

### DataTypeFP32

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/rt_raw_ads_demand_attributes_... | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ads_demand_attributes_... | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/derived_3_fp32 0/1 defaults (0%) | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_3_fp32 1/1 defaults (... | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/derived_4_fp32 0/1 defaults (0%) | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_4_fp32 1/1 defaults (... | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/rt_raw_ad_attributes_v1_fp32 ... | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_attributes_v1_fp32 ... | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/derived_ads_fp32 1/3 defaults... | 3 | 33.3% | 25.00% | 25.00% ✅ |
| catalog/derived_ads_fp32 2/3 defaults... | 3 | 66.7% | 58.33% | 58.33% ✅ |
| catalog/organic__derived_fp32 5/11 de... | 11 | 45.5% | 40.91% | 40.91% ✅ |
| catalog/organic__derived_fp32 8/11 de... | 11 | 72.7% | 68.18% | 65.85% ✅ |
| catalog/derived_fp32 23/46 defaults (... | 46 | 50.0% | 46.74% | 34.67% ✅ |
| catalog/derived_fp32 36/46 defaults (... | 46 | 78.3% | 75.00% | 46.51% ✅ |
| catalog/derived_2_fp32 0/1 defaults (0%) | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_2_fp32 1/1 defaults (... | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/rt_raw_ad_cpc_value_fp32 0/1 ... | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_cpc_value_fp32 1/1 ... | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/rt_raw_ad_batch_attributes_fp... | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_batch_attributes_fp... | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/rt_raw_ad_gmv_max_attributes_... | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_gmv_max_attributes_... | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/derived_fp32 0% defaults (all... | 46 | 0.0% | -3.26% | -3.26% ⚠️ |
| catalog/derived_fp32 100% defaults | 46 | 100.0% | 96.74% | 57.14% ✅ |

### DataTypeString

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/properties_string 0/1 default... | 1 | 0.0% | -14.29% | -14.29% ⚠️ |
| catalog/properties_string 1/1 default... | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/derived_string 2/4 defaults (... | 4 | 50.0% | 16.67% | 16.67% ✅ |
| catalog/derived_string 3/4 defaults (... | 4 | 75.0% | 38.46% | 38.46% ✅ |
| catalog/properties_2_string 0/1 defau... | 1 | 0.0% | -14.29% | -14.29% ⚠️ |
| catalog/properties_2_string 1/1 defau... | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/rt_raw_is_live_on_ad_string 0... | 1 | 0.0% | -14.29% | -14.29% ⚠️ |
| catalog/rt_raw_is_live_on_ad_string 1... | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/realtime_string 0/1 defaults ... | 1 | 0.0% | -16.67% | -16.67% ⚠️ |
| catalog/realtime_string 1/1 defaults ... | 1 | 100.0% | 50.00% | 50.00% ✅ |

### DataTypeInt64

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/realtime_int64_1 0/1 defaults... | 1 | 0.0% | -12.50% | -12.50% ⚠️ |
| catalog/realtime_int64_1 1/1 defaults... | 1 | 100.0% | 87.50% | 87.50% ✅ |
| catalog/realtime_int64 0/1 defaults (0%) | 1 | 0.0% | -12.50% | -12.50% ⚠️ |
| catalog/realtime_int64 1/1 defaults (... | 1 | 100.0% | 87.50% | 87.50% ✅ |

### DataTypeFP32Vector

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/embedding_ca_fp32 0/1 default... | 1 | 0.0% | -6.25% | -6.25% ⚠️ |
| catalog/embedding_ca_fp32 1/1 default... | 1 | 100.0% | 93.75% | 92.86% ✅ |

### DataTypeInt32

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/rt_raw_ad_attributes_int32 0/... | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_attributes_int32 1/... | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/derived_int32 7/14 defaults (... | 14 | 50.0% | 46.43% | 46.43% ✅ |
| catalog/derived_int32 11/14 defaults ... | 14 | 78.6% | 75.00% | 68.18% ✅ |

### DataTypeUint64

| Scenario | Features | Defaults | Original Δ | Compressed Δ |
|----------|----------|-----------|------------|-------------|
| catalog/raw_uint64 1/3 defaults (33%) | 3 | 33.3% | 29.17% | 29.17% ✅ |
| catalog/raw_uint64 2/3 defaults (67%) | 3 | 66.7% | 62.50% | 62.50% ✅ |

## All Results Summary (Catalog Use Case)

| Test Name | Data Type | Features | Defaults | Original Δ | Compressed Δ |
|-----------|-----------|----------|-----------|------------|-------------|
| catalog/vector_int32 0/1 defaults (0%) | DataTypeInt32Vector | 1 | 0.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32 1/1 defaults (100%) | DataTypeInt32Vector | 1 | 100.0% | 93.75% | 92.86% ✅ |
| catalog/embeddings_v2_fp16 1/3 defaults (33%) | DataTypeFP16Vector | 3 | 33.3% | 29.17% | 29.17% ✅ |
| catalog/embeddings_v2_fp16 2/3 defaults (67%) | DataTypeFP16Vector | 3 | 66.7% | 62.50% | 62.50% ✅ |
| catalog/embedding_stcg_fp16 1/3 defaults (... | DataTypeFP16Vector | 3 | 33.3% | 29.17% | 29.17% ✅ |
| catalog/embedding_stcg_fp16 2/3 defaults (... | DataTypeFP16Vector | 3 | 66.7% | 62.50% | 62.50% ✅ |
| catalog/raw_fp16_7d_1d_1am 0/1 defaults (0%) | DataTypeFP16 | 1 | 0.0% | -50.00% | -50.00% ⚠️ |
| catalog/raw_fp16_7d_1d_1am 1/1 defaults (1... | DataTypeFP16 | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/rt_raw_ads_demand_attributes_fp32 ... | DataTypeFP32 | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ads_demand_attributes_fp32 ... | DataTypeFP32 | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/derived_3_fp32 0/1 defaults (0%) | DataTypeFP32 | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_3_fp32 1/1 defaults (100%) | DataTypeFP32 | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/derived_fp16 2/4 defaults (50%) | DataTypeFP16 | 4 | 50.0% | 37.50% | 37.50% ✅ |
| catalog/derived_fp16 3/4 defaults (75%) | DataTypeFP16 | 4 | 75.0% | 62.50% | 62.50% ✅ |
| catalog/properties_string 0/1 defaults (0%) | DataTypeString | 1 | 0.0% | -14.29% | -14.29% ⚠️ |
| catalog/properties_string 1/1 defaults (100%) | DataTypeString | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/realtime_int64_1 0/1 defaults (0%) | DataTypeInt64 | 1 | 0.0% | -12.50% | -12.50% ⚠️ |
| catalog/realtime_int64_1 1/1 defaults (100%) | DataTypeInt64 | 1 | 100.0% | 87.50% | 87.50% ✅ |
| catalog/derived_4_fp32 0/1 defaults (0%) | DataTypeFP32 | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_4_fp32 1/1 defaults (100%) | DataTypeFP32 | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/rt_raw_ad_attributes_v1_fp32 0/1 d... | DataTypeFP32 | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_attributes_v1_fp32 1/1 d... | DataTypeFP32 | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/derived_ads_fp32 1/3 defaults (33%) | DataTypeFP32 | 3 | 33.3% | 25.00% | 25.00% ✅ |
| catalog/derived_ads_fp32 2/3 defaults (67%) | DataTypeFP32 | 3 | 66.7% | 58.33% | 58.33% ✅ |
| catalog/embedding_ca_fp32 0/1 defaults (0%) | DataTypeFP32Vector | 1 | 0.0% | -6.25% | -6.25% ⚠️ |
| catalog/embedding_ca_fp32 1/1 defaults (100%) | DataTypeFP32Vector | 1 | 100.0% | 93.75% | 92.86% ✅ |
| catalog/organic__derived_fp32 5/11 default... | DataTypeFP32 | 11 | 45.5% | 40.91% | 40.91% ✅ |
| catalog/organic__derived_fp32 8/11 default... | DataTypeFP32 | 11 | 72.7% | 68.18% | 65.85% ✅ |
| catalog/derived_fp32 23/46 defaults (50%) | DataTypeFP32 | 46 | 50.0% | 46.74% | 34.67% ✅ |
| catalog/derived_fp32 36/46 defaults (78%) | DataTypeFP32 | 46 | 78.3% | 75.00% | 46.51% ✅ |
| catalog/raw_fp16_1d_30m_12am 0/1 defaults ... | DataTypeFP16 | 1 | 0.0% | -50.00% | -50.00% ⚠️ |
| catalog/raw_fp16_1d_30m_12am 1/1 defaults ... | DataTypeFP16 | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/derived_string 2/4 defaults (50%) | DataTypeString | 4 | 50.0% | 16.67% | 16.67% ✅ |
| catalog/derived_string 3/4 defaults (75%) | DataTypeString | 4 | 75.0% | 38.46% | 38.46% ✅ |
| catalog/properties_2_string 0/1 defaults (0%) | DataTypeString | 1 | 0.0% | -14.29% | -14.29% ⚠️ |
| catalog/properties_2_string 1/1 defaults (... | DataTypeString | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/derived_2_fp32 0/1 defaults (0%) | DataTypeFP32 | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/derived_2_fp32 1/1 defaults (100%) | DataTypeFP32 | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/realtime_int64 0/1 defaults (0%) | DataTypeInt64 | 1 | 0.0% | -12.50% | -12.50% ⚠️ |
| catalog/realtime_int64 1/1 defaults (100%) | DataTypeInt64 | 1 | 100.0% | 87.50% | 87.50% ✅ |
| catalog/merlin_embeddings_fp16 1/2 default... | DataTypeFP16Vector | 2 | 50.0% | 43.75% | 43.75% ✅ |
| catalog/rt_raw_ad_attributes_int32 0/1 def... | DataTypeInt32 | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_attributes_int32 1/1 def... | DataTypeInt32 | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/rt_raw_ad_cpc_value_fp32 0/1 defau... | DataTypeFP32 | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_cpc_value_fp32 1/1 defau... | DataTypeFP32 | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/raw_uint64 1/3 defaults (33%) | DataTypeUint64 | 3 | 33.3% | 29.17% | 29.17% ✅ |
| catalog/raw_uint64 2/3 defaults (67%) | DataTypeUint64 | 3 | 66.7% | 62.50% | 62.50% ✅ |
| catalog/rt_raw_ad_batch_attributes_fp32 0/... | DataTypeFP32 | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_batch_attributes_fp32 1/... | DataTypeFP32 | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/embeddings_fp16 0/1 defaults (0%) | DataTypeFP16Vector | 1 | 0.0% | -12.50% | -12.50% ⚠️ |
| catalog/embeddings_fp16 1/1 defaults (100%) | DataTypeFP16Vector | 1 | 100.0% | 87.50% | 87.50% ✅ |
| catalog/vector_int32_lifetime 0/1 defaults... | DataTypeInt32Vector | 1 | 0.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32_lifetime 1/1 defaults... | DataTypeInt32Vector | 1 | 100.0% | 93.75% | 92.86% ✅ |
| catalog/derived_int32 7/14 defaults (50%) | DataTypeInt32 | 14 | 50.0% | 46.43% | 46.43% ✅ |
| catalog/derived_int32 11/14 defaults (79%) | DataTypeInt32 | 14 | 78.6% | 75.00% | 68.18% ✅ |
| catalog/vector_int32_lifetime_v2 0/1 defau... | DataTypeInt32Vector | 1 | 0.0% | -6.25% | -6.25% ⚠️ |
| catalog/vector_int32_lifetime_v2 1/1 defau... | DataTypeInt32Vector | 1 | 100.0% | 93.75% | 92.86% ✅ |
| catalog/rt_raw_is_live_on_ad_string 0/1 de... | DataTypeString | 1 | 0.0% | -14.29% | -14.29% ⚠️ |
| catalog/rt_raw_is_live_on_ad_string 1/1 de... | DataTypeString | 1 | 100.0% | 50.00% | 50.00% ✅ |
| catalog/rt_raw_ad_gmv_max_attributes_fp32 ... | DataTypeFP32 | 1 | 0.0% | -25.00% | -25.00% ⚠️ |
| catalog/rt_raw_ad_gmv_max_attributes_fp32 ... | DataTypeFP32 | 1 | 100.0% | 75.00% | 75.00% ✅ |
| catalog/realtime_string 0/1 defaults (0%) | DataTypeString | 1 | 0.0% | -16.67% | -16.67% ⚠️ |
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

**Generated:** 2026-02-25 17:39:01
