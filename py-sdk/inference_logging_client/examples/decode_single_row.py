"""
Standalone script that decodes inference-log rows from a local CSV using
`decode_mplog_proto_dataframe` with the full 256-feature schema for
`clp-organic-l2-ranker-v1-0` (version=1, PROTO format).

Run locally:

    pip install inference-logging-client==0.3.4 zstandard pyspark
    python decode_single_row.py [path/to/logs.csv]

If no path is given, defaults to /Users/dheerajchouhan/Downloads/test_new.csv.

Output:
  - prints schema, row counts, and a sample of score columns
  - writes parquet to /tmp/decoded_single_row/
"""

import sys

from pyspark.sql import SparkSession
from pyspark.sql.types import LongType, StringType, StructField, StructType

from inference_logging_client import decode_mplog_proto_dataframe


# ---------------------------------------------------------------------------
# 1. Input CSV path
# ---------------------------------------------------------------------------

DEFAULT_CSV_PATH = "/Users/dheerajchouhan/Downloads/test_new.csv"


# ---------------------------------------------------------------------------
# 2. Full feature schema (256 features, version=1 for this mp_config_id)
# ---------------------------------------------------------------------------

SCHEMA = {
    "data": [
        {"feature_name": "user_scat:derived_fp32:orders_by_clicks_laplace_56day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_scat:derived_fp32:clicks_by_views_laplace_3day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_scat:derived_fp32:clicks_by_views_laplace_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_scat:derived_fp32:log_clicks_14day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_scat:derived_fp32:log_clicks_56day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_scat:derived_fp32:log_clicks_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_scat:derived_fp32:log_views_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_clp:derived_fp32:clicks_by_views_3_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_clp:derived_fp32:clicks_by_views_7_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_clp:derived_fp32:orders_by_views_28_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_clp:derived_fp32:orders_by_clicks_28_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_clp:derived_fp32:orders_by_views_7_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_clp:derived_fp32:clicks_3_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_clp:derived_fp32:orders_28_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_clp:derived_fp32:orders_3_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_clp:derived_fp32:orders_7_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:ads_interactions_timeseries_transforms_orders_7day_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:catalog__nqp_90_days", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:clicks_by_views_28_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:low_risk_user_orders_percentage", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:od_more_than_75p_user_api_user_orders_percentage_90day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:vrs_boosting_factor", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:ads_interactions_timeseries_transforms_clicks_28day_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:ads_interactions_timeseries_transforms_orders_3day_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:clicks_28_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:clicks_3_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:clicks_7_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:log_reviews", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:low_risk_user_orders_percentage_90day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:ads_interactions_timeseries_transforms_clicks_3day_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:catalog__per_qr_return", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:orders_by_clicks_28_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:price_discount_percent", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:views_7_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:ads_interactions_timeseries_transforms_views_28day_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:avg_rating", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:catalog__per_return", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:orders_7_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:orders_by_clicks_7_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:nqd_boosting_factor_gbm_model_v0", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:nqp_by_nqd", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:od_20_plus_order_users_orders_percentage_90day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:price_decrease_percent_decay", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:ads_interactions_timeseries_transforms_views_3day_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:catalog__ads_orders_by_clicks_28_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:loyalty_boosting_factor", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:nqp", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:orders_28_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:views_3_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:ads_interactions_timeseries_transforms_orders_28day_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:catalog__ads_clicks_by_views_7_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:catalog__ads_orders_by_clicks_7_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:clicks_by_views_7_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:net_order_by_gross_order_smoothened", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:nqp_by_nqd_90day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:views_28_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:ads_interactions_timeseries_transforms_clicks_7day_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:ads_interactions_timeseries_transforms_views_7day_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:catalog__ads_clicks_by_views_28_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:catalog__mean_price_90_days", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_fp32:orders_3_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog:derived_int32:catalog__is_non_gst", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "catalog:derived_int32:catalog__price_arp", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "catalog:derived_int32:fds_attributes_raw_age", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "catalog:derived_int32:is_mall", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "catalog:derived_int32:od_less_than_25p_user_aov_user_orders_90day", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "catalog:derived_int32:price_shipping", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "catalog:derived_int32:top_supplier_id", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "catalog:derived_int32:arp", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "catalog:derived_string:portfolio_name", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "catalog:derived_string:sscat_level_attribute_value_2", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "catalog:derived_string:super_portfolio_name", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "catalog:derived_string:attribute_value_1", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "catalog:derived_string:attribute_value_2", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "catalog:derived_string:attribute_value_3", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "catalog:embeddings_fp16:search__flava_embedding_str", "feature_type": "DataTypeFP16Vector", "feature_size": 1},
        {"feature_name": "catalog:realtime_int64:cat_id", "feature_type": "DataTypeInt64", "feature_size": 1},
        {"feature_name": "catalog:realtime_int64:scat_id", "feature_type": "DataTypeInt64", "feature_size": 1},
        {"feature_name": "catalog:realtime_int64:sscat_id", "feature_type": "DataTypeInt64", "feature_size": 1},
        {"feature_name": "clp:derived_int32:unique_sscats", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "clp:derived_string:name", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqd_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqp_90_days_gender_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqd_90_days_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqp_90_days_od_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqp_by_nqd_division_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqp_by_nqd_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqp_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqp_by_nqd_90_days_od_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqp_by_nqd_90_days_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqp_by_nqd_first_order_age_bin_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__avg_rating_90_days_od_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqp_90_days_division_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqp_90_days_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqp_by_nqd_90_days_division_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqp_by_nqd_90_days_gender_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__avg_rating_90_days_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqp_by_nqd_90_days_first_order_age_bin_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqp_by_nqd_gender_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_user_quality_segment:rollup:supplier__nqp_by_nqd_od_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_cat:derived_fp32:clicks_by_views_laplace_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_cat:derived_fp32:log_clicks_28day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_cat:derived_fp32:log_clicks_56day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_cat:derived_fp32:log_views_3day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_cat:derived_fp32:log_views_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_cat:derived_fp32:orders_by_clicks_laplace_56day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "cat:derived_string:name", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:clicks_by_views_laplace_3day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:clicks_by_views_laplace_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:orders_by_clicks_laplace_28day_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:orders_by_clicks_laplace_3day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:orders_by_clicks_laplace_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:views_3_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:orders_by_views_laplace_28day_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:price_ratio", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:orders_by_views_laplace_28day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:clicks_3_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:clicks_by_views_laplace_28day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:clicks_by_views_laplace_3day_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:orders_3_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:orders_7_days_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:orders_by_clicks_laplace_28day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:clicks_by_views_laplace_28day_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:clicks_by_views_laplace_7day_percentile", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_fp32:orders_by_views_laplace_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_catalog:derived_int32:price_diff", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "supplier:derived_2_fp32:supplier__nqp_28_days", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier:derived_2_fp32:supplier__nqp_By_nqd", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier:derived_2_fp32:supplier__nqd", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier:derived_2_fp32:supplier__nqd_28_days", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier:derived_2_fp32:supplier__nqp_By_nqd_90_days", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier:derived_2_fp32:supplier__per_qr_return", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier:derived_2_fp32:supplier__per_return", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier:derived_2_fp32:supplier__per_wfr_return", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier:derived_2_fp32:supplier__pq_return_per", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier:derived_2_fp32:supplier__nqp", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier:derived_2_fp32:supplier__nqp_90_days", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_sscat:derived_fp32:net_orders_by_gross_orders_90day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_sscat:derived_fp32:num_video_review_by_num_review_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_sscat:derived_fp32:supplier_sscat__nqd_28_days", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_sscat:derived_fp32:supplier_sscat__nqp_28_days", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "supplier_sscat:derived_fp32:supplier_sscat__num_nps_response_By_num_user_with_nps_response_28_days", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqp_90_days_gender_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqp_90_days_division_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__per_qr_return_division_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__per_return_first_order_age_bin_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__per_wfr_return_division_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__avg_rating_90_days_od_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__avg_rating_90_days_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqd_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqp_90_days_od_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqp_by_nqd_90_days_first_order_age_bin_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqp_by_nqd_90_days_gender_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqp_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__per_wfr_return_gender_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__pq_return_per_first_order_age_bin_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqp_by_nqd_90_days_od_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqp_by_nqd_division_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqp_by_nqd_first_order_age_bin_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__per_return_od_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqp_by_nqd_gender_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__wfr_return_per_gender_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqp_by_nqd_90_days_division_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqp_by_nqd_od_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__per_return_division_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__per_return_gender_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqd_90_days_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__per_return_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__wfr_return_per_first_order_age_bin_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqp_90_days_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqp_by_nqd_90_days_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__nqp_by_nqd_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__per_qr_return_first_order_age_bin_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__per_qr_return_gender_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__per_qr_return_review_media_engagement_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "catalog_user_quality_segment:rollup:catalog__per_wfr_return_first_order_age_bin_cohort", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_sscat:derived_fp32:view_contribution_3_days", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_sscat:derived_fp32:clp_sscat__clicks_28_days__log", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_sscat:derived_fp32:clp_sscat__clicks_7_days__log", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_sscat:derived_fp32:clp_sscat__orders_28_days__log", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_sscat:derived_fp32:clp_sscat__orders_7_days__log", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_sscat:derived_fp32:clp_sscat__views_28_days__log", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_sscat:derived_fp32:clp_sscat__views_7_days__log", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_sscat:derived_fp32:order_contribution_30_days", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_sscat:derived_int32:clicks_28day", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "clp_sscat:derived_int32:clicks_7day", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "clp_sscat:derived_int32:orders_28day", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "clp_sscat:derived_int32:orders_7day", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "clp_sscat:derived_int32:views_7day", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "clp_sscat:derived_int64:views_28day", "feature_type": "DataTypeInt64", "feature_size": 1},
        {"feature_name": "sscat:derived_fp32:sscat__mean_price_90_days", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "sscat:derived_string:name", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_int32:user__app_count", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "user:derived_2_int32:voice_search_count", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "user:derived_2_int32:percentile_clicks_bin_28day", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "user:derived_2_int32:text_search_count", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "user:derived_2_int32:total_click_gold_catalog_count_30day", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "user:derived_2_int32:total_click_rated_catalog_count_30day", "feature_type": "DataTypeInt32", "feature_size": 1},
        {"feature_name": "user:derived_2_string:aov_bin_90day", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:app_language", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:net_orders_bin_30day", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:price_index_bin_90day", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:order_stage_bin", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:order_stage_bin_90day", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:pincode", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:aov_bin", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:first_order_bin", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:gross_orders_bin_30day", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:last_month_r_segment", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:occupation", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:region", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:user_quality_segment", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:gross_orders_bin", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:install_source", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:net_orders_bin", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:price_index_bin", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:user__division", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:age_bin", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_string:gender", "feature_type": "DataTypeString", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:user__avg_rating", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:avg_order_catalog_nqd_90day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:gross_orders", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:net_orders", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:avg_click_price_30day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:retention_90_days", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:user__nqp", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:avg_click_catalog_nqd_30day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:log_clicks_56day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:percentage_click_rated_catalog_count_30day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:orders_by_clicks_laplace_28day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:browse_time_last_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:clicks_by_views_laplace_28day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:clicks_by_views_laplace_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:net_order_by_gross_order", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:orders_by_clicks_laplace_56day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:percentage_click_female_catalog_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:user__nqd", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:user__nqp_by_nqd", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:log_clicks_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:log_views_56day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:mean_price_index", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:clicks_by_views_laplace_56day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:log_orders_lifetime", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:orders_by_clicks_laplace_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:log_clicks_14day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:clicks_by_views_laplace_14day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user:derived_2_fp32:engagement_click_percent", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_sscat:derived_fp32:clicks_by_views_laplace_3day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_sscat:derived_fp32:log_clicks_56day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_sscat:derived_fp32:orders_by_clicks_laplace_56day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_sscat:derived_fp32:log_views_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_sscat:derived_fp32:nqd", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_sscat:derived_fp32:nqp", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_sscat:derived_fp32:clicks_by_views_laplace_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_sscat:derived_fp32:log_clicks_14day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_sscat:derived_fp32:log_clicks_28day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "user_sscat:derived_fp32:log_clicks_7day", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_query:embeddings_fp16:embedding_str", "feature_type": "DataTypeFP16Vector", "feature_size": 1},
        {"feature_name": "score", "feature_type": "DataTypeFP32", "feature_size": 1},
        {"feature_name": "clp_query_val", "feature_type": "BYTES", "feature_size": 1},
        {"feature_name": "pctr_pre:portfolio_name", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:super_portfolio_name", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__net_orders_bin", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:sscat_level_attribute_value_2", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__region", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__first_order_bin", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__aov_bin", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__aov_bin_90_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__order_stage_bin", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__order_stage_bin_90_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__price_index_bin", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__price_index_bin_90_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__age_bin_on_meesho_in_days__30_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__gross_orders_bin", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__net_orders_bin__30_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__gross_orders_bin__30_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:cat_id", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__app_language", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__occupation", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:user__last_month_r_segment", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:is_mall", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:catalog_level_attribute_value_1", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:catalog_level_attribute_value_2", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:catalog_level_attribute_value_3", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:catalog__is_non_gst", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:predicted_nqd", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:clp_sscat__obyc_7_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:clp_sscat__obyc_28_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:clp_sscat__cbyv_7_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_pre:clp_sscat__cbyv_28_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:portfolio_name", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__region", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:super_portfolio_name", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__first_order_bin", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__app_language", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__order_stage_bin_90_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__age_bin_on_meesho_in_days__30_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__gender", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__aov_bin", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__aov_bin_90_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__order_stage_bin", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__price_index_bin", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__price_index_bin_90_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__net_orders_bin", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__gross_orders_bin", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__net_orders_bin__30_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__gross_orders_bin__30_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:cat_id", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:user__install_source", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:is_mall", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:catalog_level_attribute_value_1", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:catalog_level_attribute_value_2", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:catalog_level_attribute_value_3", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:catalog__is_non_gst", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:predicted_nqd", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:clp_sscat__obyc_7_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:clp_sscat__obyc_28_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:clp_sscat__cbyv_7_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_pre:clp_sscat__cbyv_28_days", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pctr_score", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "pcvr_score", "feature_type": "FP16", "feature_size": 1},
        {"feature_name": "p_nqd_score", "feature_type": "FP32", "feature_size": 1},
    ]
}


# ---------------------------------------------------------------------------
# 3. Build the DataFrame from CSV and decode
# ---------------------------------------------------------------------------

CSV_SCHEMA = StructType([
    StructField("prism_ingested_at", LongType(), True),
    StructField("prism_extracted_at", LongType(), True),
    StructField("created_at", StringType(), True),
    StructField("entities", StringType(), True),
    StructField("features", StringType(), True),
    StructField("metadata", StringType(), True),
    StructField("mp_config_id", StringType(), True),
    StructField("parent_entity", StringType(), True),
    StructField("tracking_id", StringType(), True),
    StructField("user_id", StringType(), True),
    StructField("year", StringType(), True),
    StructField("month", StringType(), True),
    StructField("day", StringType(), True),
    StructField("hour", StringType(), True),
])


def main():
    csv_path = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_CSV_PATH

    spark = (
        SparkSession.builder
        .appName("decode_single_row")
        .config("spark.sql.execution.arrow.pyspark.enabled", "true")
        .getOrCreate()
    )

    # multiLine=true because the JSON cells contain embedded commas/quotes.
    df = (
        spark.read
        .option("header", "true")
        .option("multiLine", "true")
        .option("escape", '"')
        .schema(CSV_SCHEMA)
        .csv(csv_path)
    )

    n_in = df.count()
    print(f"input csv: {csv_path}")
    print(f"input rows: {n_in}")
    print(f"schema features: {len(SCHEMA['data'])}")

    if n_in == 0:
        print("no rows in csv, exiting")
        return

    decoded = decode_mplog_proto_dataframe(df, spark, schema=SCHEMA)

    print("decoded schema:")
    decoded.printSchema()
    print(f"decoded rows: {decoded.count()}")

    quick_cols = [
        c for c in [
            "entity_id",
            "score",
            "pctr_score",
            "pcvr_score",
            "p_nqd_score",
            "catalog:realtime_int64:cat_id",
            "catalog:realtime_int64:scat_id",
            "catalog:realtime_int64:sscat_id",
            "catalog:derived_string:portfolio_name",
        ]
        if c in decoded.columns
    ]
    if quick_cols:
        decoded.select(*quick_cols).show(20, truncate=False)

    out_path = "/tmp/decoded_single_row"
    decoded.write.mode("overwrite").parquet(out_path)
    print(f"wrote: {out_path}")


if __name__ == "__main__":
    main()
