"""
Standalone script that decodes ONE inference-log row using
`decode_mplog_proto_dataframe` with the full 256-feature schema.

Run on Databricks (or any pyspark environment with the package installed):

    pip install inference-logging-client==0.3.4 zstandard pyspark
    python decode_single_row.py

The script:
  1. Builds a one-row Spark DataFrame with the exact `entities`, `features`,
     and `metadata` strings provided.
  2. Calls `decode_mplog_proto_dataframe(df, spark, schema=SCHEMA)`.
  3. Prints the decoded DataFrame (one row per entity) and writes parquet
     to `/tmp/decoded_single_row/`.

If the `features` JSON or `entities` JSON is very long, paste the full
strings into the two placeholders below (FEATURES_JSON and ENTITIES_JSON).
"""

from pyspark.sql import Row, SparkSession

from inference_logging_client import decode_mplog_proto_dataframe


# ---------------------------------------------------------------------------
# 1. Row data
# ---------------------------------------------------------------------------

PRISM_INGESTED_AT = 1778093134932
PRISM_EXTRACTED_AT = 1778093172000
CREATED_AT = "2026-05-07T00:15:34.000+05:30"

# Fill in the actual mp_config_id for this row (pass-through column).
MP_CONFIG_ID = "clp-organic-l2-ranker-v1-0"

YEAR, MONTH, DAY, HOUR = "2026", "05", "07", "00"

# Metadata byte = 0x04 -> compression=False, version=1, format=PROTO
METADATA_JSON = '["BA=="]'

# Paste the full entities JSON-array string from the row here (300 entity ids).
ENTITIES_JSON = (
    '["129858133","172683451","40159726","163432064","157307877","127153263",'
    '"150494430","91634625","105262423","143362971","152885639","141723254",'
    '"67612028","90190259","90799634","37254681","69442240","12704667",'
    '"182883452","101545685","92571270","109083227","122270821","189312092",'
    '"147408625","88849258","105146149","157307878","178517767","68884198",'
    '"124324363","183029599","101509512","187296531","185250355","105290811",'
    '"94130043","110902288","162834270","46188585","193161","187058552",'
    '"143662941","182577489","178489609","78387570","165354806","23923038",'
    '"124383285","102178839","187855725","67999990","112389083","53622608",'
    '"157307876","40322544","193681","87261452","91976737","58163190",'
    '"182561787","159505400","174790442","162708871","125787271","189130322",'
    '"133152244","154338081","78656868","18307540","185845733","165178985",'
    '"105045470","115278211","168328734","127711153","47093013","173028789",'
    '"152891706","150483687","119108032","148530689","137907125","189735602",'
    '"140606894","55894553","113416713","134191955","137961683","108117958",'
    '"126322517","81413078","88758009","76830400","188805968","162409042",'
    '"101610920","129403638","14712806","114253030","188269192","15729965",'
    '"77084232","123680130","105910520","133951061","116614716","182882195",'
    '"81376971","150810651","148564784","138414869","11645744","32447000",'
    '"114509225","110751353","60133034","95408000","100316387","125457803",'
    '"105186243","138102163","87461454","110808182","187527955","174901426",'
    '"112985436","158026050","176144041","125544584","90201083","9105405",'
    '"57338961","23055087","90191915","178738841","125250712","151895413",'
    '"112335263","163340375","161219586","182447297","124901483","182847433",'
    '"149289846","108108708","96851092","116066124","148109044","92858017",'
    '"81310699","80426512","123293294","105146150","130108227","166968286",'
    '"189302536","120632248","181328675","162162514","55835366","120336101",'
    '"164118551","169249718","96818909","119434054","113454362","156102338",'
    '"124943412","184408554","99651509","120571620","124904974","161748042",'
    '"92858620","40159729","14770797","136077639","67658808","90035215",'
    '"72921911","118273836","133708453","137496876","106377352","81906055",'
    '"158064774","22371023","131621161","132315814","160010257","9640992",'
    '"75121762","55205272","105838887","127970505","96546979","184455344",'
    '"178084048","42691306","117911911","154907429","101610459","120345620",'
    '"187298515","90519128","182727359","101497968","13157465","80803175",'
    '"188408895","67656425","91403106","155610407","142739602","185836448",'
    '"159831050","138611069","103394964","134702173","144832430","645632",'
    '"166130025","174766204","123236695","18745107","62959962","127979490",'
    '"101508311","103660271","177931974","67610484","181645149","137370294",'
    '"40159730","129572066","127117824","127212003","43596020","70468712",'
    '"129395854","78664565","161719437","134874694","180101337","113065467",'
    '"117541611","67655129","99498602","536757","45502433","114285885",'
    '"77234578","40964091","183387920","90519125","73522956","139548067",'
    '"115084443","108526129","40322281","172987169","138291358","125458478",'
    '"129140995","175694187","40988876","187451321","41704353","90930750",'
    '"117304498","112026524","120715082","181587439","129737618","120534472",'
    '"123654440","156204061","95892707","186627409","89870249","156174988",'
    '"158606418","73774749","184531000","41005967","151464858","123351724",'
    '"129180516","134941738","50414200","12805166","156839738","151884108",'
    '"92552214","118409718","49832509","161418189","108108982","97081458"]'
)

# Paste the full features JSON-array string here. This is the cell content
# from the `features` column, exactly as stored. Truncated here for brevity;
# replace with the full value when running.
FEATURES_JSON = (
    '[{"encoded_features":"<PASTE_FULL_BASE64_FOR_ENTITY_0_HERE>"},'
    ' {"encoded_features":"<PASTE_FULL_BASE64_FOR_ENTITY_1_HERE>"},'
    ' ... 300 entries total ...'
    ']'
)


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
# 3. Build the DataFrame and decode
# ---------------------------------------------------------------------------

def main():
    spark = (
        SparkSession.builder
        .appName("decode_single_row")
        .getOrCreate()
    )

    row = Row(
        prism_ingested_at=PRISM_INGESTED_AT,
        prism_extracted_at=PRISM_EXTRACTED_AT,
        created_at=CREATED_AT,
        entities=ENTITIES_JSON,
        features=FEATURES_JSON,
        metadata=METADATA_JSON,
        mp_config_id=MP_CONFIG_ID,
        parent_entity=None,
        tracking_id=None,
        user_id=None,
        year=YEAR,
        month=MONTH,
        day=DAY,
        hour=HOUR,
    )
    df = spark.createDataFrame([row])

    print(f"input rows: {df.count()}")
    print(f"schema features: {len(SCHEMA['data'])}")

    decoded = decode_mplog_proto_dataframe(df, spark, schema=SCHEMA)

    print("decoded schema:")
    decoded.printSchema()
    print(f"decoded rows: {decoded.count()}")

    # Quick look at the most interesting score columns
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
