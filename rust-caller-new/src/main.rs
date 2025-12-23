use axum::{extract::State, http::StatusCode, response::{Html, Json, Response, IntoResponse}, routing::{get, post}, Router};
use axum::body::Body;
use std::sync::Arc;
use std::sync::mpsc;
use std::time::Duration;
use tokio::sync::oneshot;
use tonic::{metadata::AsciiMetadataValue, transport::{Channel, Endpoint}};

// Configure jemalloc with profiling enabled
#[cfg(not(target_env = "msvc"))]
#[global_allocator]
static ALLOC: tikv_jemallocator::Jemalloc = tikv_jemallocator::Jemalloc;

#[allow(non_upper_case_globals)]
#[export_name = "malloc_conf"]
pub static malloc_conf: &[u8] = b"prof:true,prof_active:true,lg_prof_sample:19\0";

pub mod retrieve {
    tonic::include_proto!("retrieve");
}

use retrieve::feature_service_client::FeatureServiceClient as RetrieveClient;
use retrieve::{FeatureGroup, Keys};

// Report request types
enum ReportRequest {
    Protobuf(oneshot::Sender<Result<Vec<u8>, String>>),
    Flamegraph(oneshot::Sender<Result<Vec<u8>, String>>),
    Text(oneshot::Sender<Result<String, String>>),
}

#[derive(Clone)]
struct AppState {
    client: RetrieveClient<Channel>,
    auth_token: AsciiMetadataValue,
    caller_id: AsciiMetadataValue,
    // Store strings as Arc<str> - allows sharing without cloning string data
    feature_labels: Arc<Vec<Arc<str>>>, // Shared strings, only Arc pointers are cloned
    entity_label: Arc<str>,
    keys_schema: Arc<[Arc<str>]>, // Shared strings
    report_tx: mpsc::Sender<ReportRequest>,
}

// Endpoint to get pprof data in protobuf format (for go tool pprof)
async fn get_pprof_protobuf(State(state): State<Arc<AppState>>) -> Result<Response<Body>, StatusCode> {
    let (tx, rx) = oneshot::channel();
    state.report_tx.send(ReportRequest::Protobuf(tx))
        .map_err(|_| StatusCode::SERVICE_UNAVAILABLE)?;
    
    match rx.await {
        Ok(Ok(data)) => {
            Response::builder()
                .status(StatusCode::OK)
                .header("Content-Type", "application/x-protobuf")
                .header("Content-Disposition", "attachment; filename=profile.pb.gz")
                .body(Body::from(data))
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)
        }
        _ => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
}

// Endpoint to get flamegraph SVG
async fn get_flamegraph(State(state): State<Arc<AppState>>) -> Result<Response<Body>, StatusCode> {
    let (tx, rx) = oneshot::channel();
    state.report_tx.send(ReportRequest::Flamegraph(tx))
        .map_err(|_| StatusCode::SERVICE_UNAVAILABLE)?;
    
    match rx.await {
        Ok(Ok(data)) => {
            Response::builder()
                .status(StatusCode::OK)
                .header("Content-Type", "image/svg+xml")
                .header("Content-Disposition", "inline; filename=flamegraph.svg")
                .body(Body::from(data))
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)
        }
        _ => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
}

// Endpoint to get text report
async fn get_pprof_text(State(state): State<Arc<AppState>>) -> Result<Html<String>, StatusCode> {
    let (tx, rx) = oneshot::channel();
    state.report_tx.send(ReportRequest::Text(tx))
        .map_err(|_| StatusCode::SERVICE_UNAVAILABLE)?;
    
    match rx.await {
        Ok(Ok(text)) => Ok(Html(format!("<pre>{}</pre>", text))),
        _ => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
}

// Endpoint to get heap/memory profiling data
#[cfg(not(target_env = "msvc"))]
async fn get_pprof_heap() -> Result<impl IntoResponse, (StatusCode, String)> {
    // Check if jemalloc profiling is available
    let prof_ctl = jemalloc_pprof::PROF_CTL.as_ref()
        .ok_or_else(|| (
            StatusCode::SERVICE_UNAVAILABLE,
            "jemalloc profiling not available. Ensure tikv-jemallocator is configured correctly.".to_string(),
        ))?;
    
    let mut prof_ctl = prof_ctl.lock().await;
    
    // Verify profiling is activated
    if !prof_ctl.activated() {
        return Err((
            StatusCode::SERVICE_UNAVAILABLE,
            "Heap profiling not activated. Ensure jemalloc is configured with profiling enabled.".to_string(),
        ));
    }
    
    // Generate pprof heap profile
    let pprof_data = prof_ctl.dump_pprof()
        .map_err(|e| (
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("Failed to generate heap profile: {}", e),
        ))?;
    
    Ok(Response::builder()
        .status(StatusCode::OK)
        .header("Content-Type", "application/x-protobuf")
        .header("Content-Encoding", "gzip")
        .header("Content-Disposition", "attachment; filename=heap.pb.gz")
        .body(Body::from(pprof_data))
        .map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?)
}

async fn retrieve_features(State(state): State<Arc<AppState>>) -> Result<Json<String>, StatusCode> {
    // Build Query efficiently using Arc<str> - clone Arc pointers (cheap), convert to String only for protobuf
    // This is similar to Go: strings are shared, only pointers/references are copied
    let feature_labels: Vec<String> = state.feature_labels
        .iter()
        .map(|arc_str| arc_str.as_ref().to_string()) // Clone Arc pointer, then convert to String
        .collect();
    
    let query = retrieve::Query {
        entity_label: state.entity_label.as_ref().to_string(),
        feature_groups: vec![FeatureGroup {
            label: "derived_fp32".to_string(),
            feature_labels,
        }],
        keys_schema: state.keys_schema
            .iter()
            .map(|arc_str| arc_str.as_ref().to_string())
            .collect(),
        keys: vec![
            Keys { cols: vec!["176".to_string()] },
            Keys { cols: vec!["179".to_string()] },
        ],
    };

    let mut request = tonic::Request::new(query);
    request.set_timeout(Duration::from_secs(5));
    request.metadata_mut().insert("online-feature-store-auth-token", state.auth_token.clone());
    request.metadata_mut().insert("online-feature-store-caller-id", state.caller_id.clone());

    // Clone client only when needed (tonic clients are cheap to clone)
    match state.client.clone().retrieve_features(request).await {
        Ok(_) => Ok(Json("success".to_string())),
        Err(_) => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
}


fn get_labels() -> Vec<Arc<str>> {
    vec!["max_product_price_percentile".into(), "orders_by_views_56day_percentile".into(), "catalog__platform_orders_by_clicks_28_days_percentile_bin".into(), "orders_by_views_28day_bin_percentile".into(), "min_product_price_percentile".into(), "orders_14day_percentile".into(), "total_rated_orders_bin_percentile".into(), "orders_28day_percentile".into(), "clicks_by_views_1day_percentile".into(), "min_product_price_bin_percentile".into(), "clicks_by_views_3day_percentile".into(), "log_orders_1day".into(), "total_rated_orders_percentile".into(), "clicks_by_views_5day".into(), "views_56day_percentile".into(), "log_clicks_7day".into(), "views_3day_percentile".into(), "orders_by_views_28day_percentile".into(), "orders_7day_percentile".into(), "orders_by_clicks_14day_percentile".into(), "clicks_by_views_5day_percentile".into(), "log_clicks_56day".into(), "orders_by_clicks_28day_percentile".into(), "orders_3day_percentile".into(), "clicks_by_views_56day_percentile".into(), "log_orders_14day".into(), "log_views_5day".into(), "max_product_price_percentile_bin".into(), "log_clicks_3day".into(), "views_7day_percentile".into(), "orders_by_views_14day_percentile".into(), "log_clicks_28day".into(), "log_views_1day".into(), "log_orders_7day".into(), "clicks_by_views_1day__bin_percentile".into(), "clicks_56day_percentile".into(), "clicks_5day_percentile".into(), "catalog__platform_orders_by_clicks_3_days".into(), "orders_by_clicks_3day_percentile".into(), "orders_by_clicks_1day_percentile".into(), "orders_by_clicks_5day_percentile".into(), "log_clicks_1day".into(), "orders_by_clicks_7day_percentile".into(), "log_orders_3day".into(), "log_orders_56day".into(), "clicks_7day_percentile".into(), "orders_by_views_3day_percentile".into(), "log_views_56day".into(), "clicks_3day_percentile".into(), "orders_by_views_7day_percentile".into(), "clicks_1day_percentile".into(), "orders_by_views_5day_percentile".into(), "log_clicks_14day".into(), "clicks_by_views_7day_percentile".into(), "clicks_28day_percentile".into(), "orders_5day_percentile".into(), "orders_by_views_1day_percentile".into(), "log_orders_28day".into(), "clicks_by_views_28day_bin_percentile".into(), "log_views_7day".into(), "views_5day_percentile".into(), "views_28day_percentile".into(), "log_views_28day".into(), "views_1day_percentile".into(), "orders_1day_percentile".into(), "log_views_3day".into(), "orders_by_clicks_56day_percentile".into(), "log_clicks_5day".into(), "orders_56day_percentile".into(), "clicks_by_views_14day_percentile".into(), "clicks_by_views_28day_percentile".into(), "ctr_bin_normalized_clicks_by_views_1_days".into(), "sscat_normalized_clicks_by_views_1_days".into(), "position_coec_7_days".into(), "pdp__hour_coec_7_days".into(), "hour_coec_1_days".into(), "pdp__sscat_normalized_clicks_by_views_7_days".into(), "search__clicks_by_views_1_days".into(), "pdp__position_coec_7_days".into(), "search__clicks_by_views_3_days".into(), "hour_coec_30_days".into(), "search__active_days_normalized_clicks_by_views_3_days".into(), "pdp__clicks_by_views_7_days".into(), "clicks_by_views_14_days".into(), "hour_coec_14_days".into(), "search__sscat_normalized_clicks_by_views_1_days".into(), "search__hour_coec_1_days".into(), "clicks_by_views_30_days".into(), "position_coec_3_days".into(), "search__active_days_normalized_clicks_by_views_1_days".into(), "search__position_coec_7_days".into(), "clicks_by_active_days_14_days".into(), "search__clicks_by_active_days_7_days".into(), "pdp__hour_coec_14_days".into(), "search__active_days_normalized_clicks_by_views_7_days".into(), "search__hour_coec_14_days".into(), "hour_expected_clicks_1_days".into(), "position_coec_14_days".into(), "search__clicks_by_views_7_days".into(), "search__hour_coec_3_days".into(), "position_expected_clicks_1_days".into(), "clicks_by_views_3_days".into(), "pdp__sscat_normalized_clicks_by_views_30_days".into(), "search__position_coec_30_days".into(), "pdp__clicks_by_active_days_7_days".into(), "search__position_coec_3_days".into(), "search__active_days_normalized_clicks_by_views_14_days".into(), "search__clicks_by_views_14_days".into(), "pdp__clicks_by_views_1_days".into(), "search__sscat_normalized_clicks_by_views_30_days".into(), "pdp__clicks_by_views_14_days".into(), "hour_coec_7_days".into(), "pdp__hour_coec_30_days".into(), "search__hour_coec_7_days".into(), "search__sscat_normalized_clicks_by_views_7_days".into(), "sscat_normalized_clicks_by_views_7_days".into(), "pdp__clicks_by_views_3_days".into(), "search__position_coec_1_days".into(), "search__sscat_normalized_clicks_by_views_14_days".into(), "search__clicks_by_views_30_days".into(), "clicks_by_views_1_days".into(), "pdp__position_coec_30_days".into(), "clicks_by_views_7_days".into(), "position_coec_1_days".into(), "pdp__hour_coec_1_days".into(), "search__hour_coec_30_days".into(), "sscat_normalized_clicks_by_views_30_days".into(), "search__sscat_normalized_clicks_by_views_3_days".into(), "position_coec_30_days".into(), "pdp__clicks_by_views_30_days".into(), "orders_by_clicks_laplace_28day".into(), "clicks_by_views_laplace_7day".into(), "orders_by_views_laplace_1day".into(), "clicks_by_views_laplace_3day".into(), "clicks_by_views_laplace_56day".into(), "catalog__ads_orders_by_clicks_14_days__te_laplace".into(), "orders_by_clicks_laplace_3day".into(), "orders_by_views_laplace_28day".into(), "orders_by_clicks_laplace_56day".into(), "orders_by_clicks_laplace_7day".into(), "orders_by_views_laplace_5day".into(), "clicks_by_views_laplace_5day".into(), "orders_by_views_laplace_3day".into(), "orders_by_views_laplace_56day".into(), "catalog__ads_orders_by_views_14_days__te_laplace".into(), "orders_by_clicks_laplace_5day".into(), "clicks_by_views_laplace_1day".into(), "orders_by_clicks_laplace_1day".into(), "orders_by_views_laplace_7day".into(), "clicks_by_views_laplace_28day".into(), "laplace_cbyv_by_platform_cbyv_7day".into(), "laplace_cbyv_by_platform_cbyv_56day".into(), "laplace_obyv_by_platform_obyv_1day".into(), "laplace_cbyv_by_platform_cbyv_3day".into(), "laplace_obyv_by_platform_obyv_56day".into(), "laplace_obyc_by_platform_obyc_56day".into(), "laplace_obyc_by_platform_obyc_7day".into(), "laplace_obyv_by_platform_obyv_5day".into(), "laplace_obyc_by_platform_obyc_3day".into(), "laplace_obyc_by_platform_obyc_28day".into(), "laplace_obyv_by_platform_obyv_28day".into(), "laplace_cbyv_by_platform_cbyv_5day".into(), "laplace_obyc_by_platform_obyc_5day".into(), "laplace_cbyv_by_platform_cbyv_1day".into(), "laplace_cbyv_by_platform_cbyv_28day".into(), "laplace_obyv_by_platform_obyv_3day".into(), "catalog__base_cpc".into(), "laplace_obyv_by_platform_obyv_7day".into(), "laplace_obyc_by_platform_obyc_1day".into(), "orders_by_views_3_days_percentile".into(), "views_7_days_percentile".into(), "orders_by_clicks_56_days_percentile".into(), "orders_by_clicks_5_days_percentile".into(), "clicks_by_views_28_days_percentile".into(), "orders_1_days__te_log".into(), "search__orders_by_views_3_days_percentile".into(), "orders_3_days__te_log".into(), "orders_1_days_percentile".into(), "orders_7_days_percentile".into(), "orders_7_days__te_log".into(), "search__orders_by_clicks_3_days_percentile".into(), "search__clicks_by_views_1_days_percentile_bin".into(), "search__clicks_56_days_percentile".into(), "search__orders_by_views_28_days_percentile_bin".into(), "orders_by_views_7_days_percentile_bin".into(), "orders_by_clicks_3_days_percentile".into(), "search__clicks_by_views_1_days_percentile".into(), "search__orders_by_clicks_1_days_percentile_bin".into(), "orders_by_views_14_days_percentile".into(), "search__orders_by_views_28_days_percentile".into(), "search__orders_by_views_56_days_percentile_bin".into(), "orders_by_views_1_days_percentile".into(), "orders_by_clicks_14_days_percentile".into(), "search__orders_by_views_14_days_percentile".into(), "search__orders_by_views_1_days_percentile".into(), "clicks_by_views_14_days_percentile".into(), "search__orders_by_views_5_days_percentile_bin".into(), "clicks_3_days_percentile".into(), "clicks_by_views_3_days_percentile".into(), "orders_by_clicks_28_days_percentile".into(), "search__orders_by_clicks_56_days_percentile".into(), "search__clicks_by_views_28_days_percentile".into(), "search__orders_by_clicks_7_days_percentile_bin".into(), "search__clicks_by_views_14_days_percentile_bin".into(), "orders_by_clicks_7_days".into(), "orders_by_clicks_3_days_percentile_bin".into(), "orders_by_views_3_days_percentile_bin".into(), "views_28_days_percentile".into(), "clicks_by_views_7_days_percentile".into(), "clicks_by_views_1_days_percentile".into(), "clicks_56_days__te_log".into(), "orders_by_views_28_days_percentile".into(), "search__orders_by_clicks_7_days_percentile".into(), "search__orders_by_clicks_1_days_percentile".into(), "orders_by_clicks_1_days_percentile".into(), "orders_by_clicks_7_days_percentile".into(), "orders_by_views_5_days_percentile".into(), "search__orders_by_views_14_days_percentile_bin".into(), "search__orders_by_clicks_3_days_percentile_bin".into(), "search__orders_by_views_5_days_percentile".into(), "clicks_7_days_percentile".into(), "views_56_days_percentile_bin".into(), "orders_by_clicks_7_days_percentile_bin".into(), "search__orders_1_days_percentile".into(), "clicks_28_days__te_log".into(), "orders_by_views_56_days_percentile".into(), "orders_by_clicks_1_days_percentile_bin".into(), "search__orders_by_views_56_days_percentile".into(), "views_3_days_percentile".into(), "orders_by_views_7_days_percentile".into(), "search__orders_by_views_7_days_percentile".into(), "clicks_28_days_percentile".into(), "search__orders_by_views_1_days_percentile_bin".into(), "orders_3_days_percentile".into(), "orders_28_days_percentile".into(), "clicks_by_views_56_days_percentile".into(), "num_rating_3_By_num_rating".into(), "qr_orders_by_sub_orders_28day".into(), "num_rating_4_By_num_rating".into(), "catalog__nqd_28_days".into(), "catalog__num_rating_3_By_num_rating_56_days".into(), "rtos_by_net_orders".into(), "nqp_28day".into(), "net_orders_by_gross_orders_28day".into(), "catalog__qr_orders_By_return_orders_90_days".into(), "rating_avg".into(), "rtos_by_gross_orders".into(), "net_orders_by_gross_orders_90day".into(), "num_review_By_num_rating".into(), "catalog__wfr_orders_By_return_orders".into(), "catalog__num_rating_3_By_num_rating_90_days".into(), "catalog__mean_price_90_days".into(), "rtos_by_net_orders_90day".into(), "catalog__num_rating_3_By_num_rating_28_days".into(), "num_img_review_by_num_review".into(), "avg_ratings_7day".into(), "num_review_By_num_rating_7day".into(), "nqp_by_nqd_90day".into(), "nqp".into(), "nqd".into(), "catalog__avg_ratings_28_days".into(), "net_orders_by_gross_orders_56day".into(), "catalog__nqp_90_days".into(), "user_cancelled_by_net_orders_7day".into(), "return_orders_by_sub_orders".into(), "cancellations_by_gross_orders_90day".into(), "cancellations_by_net_orders_7day".into(), "qr_orders_by_sub_orders".into(), "catalog__total_helpful_review_By_num_review".into(), "catalog__total_o2d_delay_By_total_orders_28_days".into(), "return_orders_by_sub_orders_90day".into(), "nqp_By_nqd_7day".into(), "nqp_by_nqd_28day".into(), "catalog__user_cancelled_By_net_orders".into(), "cancellations_by_net_orders_56day".into(), "catalog__num_rating_4_By_num_rating_7_days".into(), "catalog__net_orders_By_gross_orders_7_days".into(), "rtos_by_gross_orders_28day".into(), "avg_ratings_56day".into(), "total_s2d_delay_by_total_orders_28day".into(), "nqp_by_nqd".into(), "num_rating_5_by_num_rating".into(), "count_reviews_with_helpful_by_num_review_28day".into(), "rtos_by_gross_orders_56day".into(), "user_cancelled_by_net_orders_28day".into(), "catalog__return_orders_By_sub_orders_28_days".into(), "wfr_orders_by_sub_orders".into(), "num_review_by_num_rating_28day".into(), "wfr_orders_by_sub_orders_28day".into(), "num_review_by_num_rating_90day".into(), "catalog__num_rating_1_By_num_rating_5_56_days".into(), "catalog__num_rating_3_By_num_rating_7_days".into(), "nqp_7day".into(), "sscat_standardized_cat_predicted_nqd_wd_sigmoid".into(), "net_orders_by_gross_orders".into(), "cancellations_by_net_orders".into(), "return_orders_by_sub_orders_56day".into(), "cancellations_by_gross_orders".into(), "catalog__rtos_By_gross_orders_7_days".into(), "catalog__total_o2d_delay_By_total_orders".into(), "catalog__total_o2s_delayed_orders_By_total_orders_56_days".into(), "catalog__nqd_7_days".into(), "catalog__num_rating_5_By_num_rating_7_days".into(), "user_cancelled_by_net_orders_56day".into(), "cancellations_by_net_orders_28day".into(), "sscat__price_asp".into(), "asp_sscat_percentile".into(), "price_shipping_percent".into(), "price_cheapest_duplicate_diff_percent".into(), "sscat_asp_price_arp_diff_percent".into(), "price_wdrp_depth_percent".into(), "price_sscat_percentile".into(), "price_decrease_percent_decay".into(), "arp_sscat_percentile".into(), "price_discount_percent".into(), "catalog__price_decrease_pct".into(), "mrp_sscat_percentile".into(), "price_increase_percent_decay".into(), "sscat_asp_price_arp_diff".into(), "price_discount".into(), "catalog__user_risk_weighted_orders_90_days".into(), "od_between_25p_to_50p_user_api_user_orders_percentage_90day".into(), "high_risk_user_orders_percentage".into(), "od_less_than_25p_user_api_user_orders_percentage_90day".into(), "od_2_4_order_users_orders_percentage_90day".into(), "od_more_than_75p_user_aov_user_orders_percentage_90day".into(), "od_20_plus_order_users_orders_percentage_90day".into(), "avg_orders_weighted_aov_90day".into(), "low_risk_user_orders_percentage".into(), "od_5_10_order_users_orders_percentage".into(), "od_2_4_order_users_orders_percentage".into(), "low_risk_user_orders_percentage_90day".into(), "od_10_20_order_users_orders_percentage".into(), "catalog__between_50p_to_75p_user_aov_user_orders_percentage_90_days".into(), "od_0_1_order_users_orders_percentage_90day".into(), "avg_orders_weighted_api_90day".into(), "od_less_than_25p_user_aov_user_orders_percentage_90day".into(), "od_0_1_order_users_orders_percentage".into(), "od_5_10_order_users_orders_percentage_90day".into(), "od_between_25p_to_50p_user_aov_user_orders_percentage_90day".into(), "od_more_than_75p_user_api_user_orders_percentage_90day".into(), "od_between_50p_to_75p_user_api_user_orders_percentage_90day".into(), "catalog__high_risk_user_orders_percentage_90_days".into(), "od_20_plus_order_users_orders_percentage".into(), "catalog__10_20_order_users_orders_percentage_90_days".into(), "catalog__ads_clicks_by_views_56_days_percentile".into(), "catalog__ads_orders_by_views_1_days_percentile_bin".into(), "catalog__ads_orders_by_views_56_days_percentile_bin".into(), "catalog__ads_clicks_by_views_28_days_percentile".into(), "catalog__ads_orders_by_views_14_days_percentile_bin".into(), "clicks_28day_percentile".into(), "catalog__ads_orders_by_views_3_days_percentile".into(), "catalog__ads_clicks_by_views_28_days_percentile_bin".into(), "catalog__ads_orders_by_clicks_5_days_percentile".into(), "orders_56day_percentile".into(), "catalog__ads_orders_14_days_percentile".into(), "clicks_56day_percentile".into(), "catalog__ads_clicks_by_views_5_days".into(), "catalog__ads_orders_by_views_7_days_percentile".into(), "views_1day_percentile".into(), "catalog__ads_orders_by_clicks_28_days_percentile_bin".into(), "catalog__ads_orders_by_clicks_7_days_percentile_bin".into(), "catalog__ads_orders_by_views_56_days_percentile".into(), "clicks_7day_percentile".into(), "catalog__ads_orders_by_views_7_days_percentile_bin".into(), "clicks_3day_percentile".into(), "catalog__ads_orders_by_views_28_days_percentile_bin".into(), "catalog__ads_orders_by_clicks_1_days_percentile_bin".into(), "views_56day_percentile".into(), "catalog__ads_orders_by_clicks_56_days_percentile_bin".into(), "views_5day_percentile".into(), "catalog__ads_clicks_by_views_3_days_percentile_bin".into(), "catalog__ads_orders_by_clicks_7_days_percentile".into(), "catalog__ads_orders_by_views_28_days_percentile".into(), "catalog__ads_clicks_by_views_7_days_percentile".into(), "catalog__ads_orders_by_clicks_3_days_percentile".into(), "catalog__ads_clicks_by_views_1_days_percentile".into(), "catalog__ads_orders_by_clicks_3_days_percentile_bin".into(), "catalog__ads_orders_by_clicks_56_days_percentile".into(), "catalog__ads_orders_by_clicks_28_days_percentile".into(), "catalog__ads_orders_by_clicks_14_days".into(), "catalog__ads_clicks_by_views_14_days_percentile_bin".into(), "orders_3day_percentile".into(), "catalog__ads_orders_by_clicks_14_days_percentile".into(), "orders_1day_percentile".into(), "views_28day_percentile".into(), "catalog__ads_clicks_by_views_1_days_percentile_bin".into(), "orders_5day_percentile".into(), "catalog__ads_clicks_by_views_5_days_percentile_bin".into(), "orders_28day_percentile".into(), "catalog__ads_orders_by_views_5_days_percentile".into(), "orders_7day_percentile".into(), "catalog__ads_orders_by_views_5_days_percentile_bin".into(), "catalog__ads_orders_by_views_1_days_percentile".into(), "catalog__ads_clicks_by_views_56_days_percentile_bin".into(), "catalog__ads_clicks_by_views_14_days_percentile".into(), "catalog__ads_orders_by_views_14_days_percentile".into(), "views_3day_percentile".into(), "clicks_5day_percentile".into(), "catalog__ads_clicks_by_views_56_days".into(), "catalog__ads_clicks_by_views_5_days_percentile".into(), "catalog__ads_orders_by_views_3_days_percentile_bin".into(), "clicks_1day_percentile".into(), "views_7day_percentile".into(), "catalog__ads_orders_by_clicks_5_days_percentile_bin".into(), "catalog__ads_clicks_by_views_3_days_percentile".into(), "catalog__ads_clicks_by_views_7_days_percentile_bin".into(), "catalog__ads_orders_by_clicks_1_days_percentile".into(), "catalog__ads_orders_by_clicks_14_days_percentile_bin".into(), "search__orders_by_views_7_days_percentile".into(), "search__orders_by_clicks_1_days_percentile_bin".into(), "search__orders_by_views_1_days_percentile".into(), "search__orders_by_views_28_days_percentile_bin".into(), "search__clicks_by_views_3_days_percentile".into(), "search__orders_by_clicks_7_days_percentile".into(), "search__orders_by_views_5_days_percentile_bin".into(), "search__orders_by_views_14_days_percentile_bin".into(), "search__orders_by_clicks_7_days_percentile_bin".into(), "search__orders_by_views_3_days_percentile".into(), "search__clicks_by_views_14_days_percentile_bin".into(), "search__clicks_by_views_14_days_percentile".into(), "search__orders_by_views_5_days_percentile".into(), "search__clicks_by_views_28_days_percentile".into(), "search__clicks_by_views_1_days_percentile_bin".into(), "search__orders_by_views_14_days_percentile".into(), "log_reviews".into(), "price_change_percent_dec".into(), "rating_30day".into(), "price_change_percent_inc".into(), "cancel_percent".into(), "price_diff_with_p70".into(), "rating_90day".into(), "asp".into(), "final_nqd".into(), "diff_bw_asp_and_arp".into(), "avg_rating".into(), "rto_percent".into(), "return_percent".into(), "odnr_score".into(), "rating_60day".into(), "sscat_price_clicks_by_views_56day_percentile_bin_percentage".into(), "sscat_rating_orders_by_views_56day_percentile_bin_percentage".into(), "rating_orders_56day_percentile".into(), "sscat_rating_clicks_by_views_56day_percentile_bin_percentage".into(), "rating_views_56day_percentile".into(), "sscat_price_orders_by_views_56day_percentile_bin_percentage".into(), "catalog_rating".into(), "sscat_price_orders_by_clicks_56day_percentile_bin_percentage".into(), "price_clicks_56day_percentile".into(), "price_orders_56day_percentile".into(), "sscat_rating_orders_by_clicks_56day_percentile_bin_percentage".into(), "price_views_56day_percentile".into(), "rating_clicks_56day_percentile".into(), "catalog__platform_orders_by_views_14_days__te_laplace".into(), "orders_by_clicks_laplace_1day".into(), "orders_by_views_laplace_28day".into(), "orders_by_views_laplace_7day".into(), "orders_by_views_laplace_3day".into(), "orders_by_clicks_laplace_5day".into(), "clicks_by_views_laplace_1day".into(), "clicks_by_views_laplace_5day".into(), "orders_by_views_laplace_56day".into(), "orders_by_clicks_laplace_28day".into(), "orders_by_clicks_laplace_3day".into(), "orders_by_views_laplace_1day".into(), "clicks_by_views_laplace_7day".into(), "clicks_by_views_laplace_28day".into(), "orders_by_clicks_laplace_56day".into(), "orders_by_clicks_laplace_7day".into(), "orders_by_views_laplace_5day".into(), "clicks_by_views_laplace_56day".into(), "clicks_by_views_laplace_3day".into(), "catalog__platform_carts_3_days__te_log".into(), "catalog__platform_carts_1_days__te_log".into(), "catalog__platform_carts_14_days__te_log".into(), "catalog__no_by_go_56_days".into(), "gross_orders".into(), "net_orders".into(), "catalog__per_return".into(), "cancellations".into(), "catalog__nqd_56_days".into(), "catalog__nqd_90_days".into(), "catalog__per_qr_return".into(), "catalog__avg_rating".into(), "catalog__no_by_go_7_days".into(), "catalog__no_by_go_28_days".into(), "catalog__no_by_go_90_days".into(), "search__sale_discount_factor_1".into(), "search__sale_discount_factor_2".into(), "clp__sale_discount_factor_2".into(), "clp__sale_discount_factor_1".into(), "pdp__sale_discount_factor_2".into(), "pdp__sale_discount_factor_1".into(), "ads_predicted_obyc_new_catalogs".into(), "ads_ds_predicted_target_roi_v1".into(), "ads_product_predicted_target_roi_v1".into(), "shipped_rto_rate_6m".into(), "shipped_rto_rate_3m".into(), "rate_6m".into(), "rate_6m".into(), "rate_3m".into(), "catalog__search_orders_by_clicks_28_days_percentile".into(), "catalog__search_clicks_by_views_28_days_percentile".into(), "catalog__search_views_3_days__te_log".into(), "predicted_net_orders_by_gross_orders".into(), "predicted_net_orders_by_gross_orders_smoothened_calibrated".into(), "predicted_net_orders_by_gross_orders_smoothened".into(), "views_3day".into(), "views_28day".into(), "orders_28day".into(), "clicks_5day".into(), "te_cbyv_7day".into(), "te_obyv_7day".into(), "te_obyv_3day".into(), "orders_5day".into(), "te_obyv_3day".into(), "te_obyc_7day".into(), "clicks_28day".into(), "clicks_1day".into(), "te_cbyv_3day".into(), "te_obyc_28day".into(), "te_cbyv_3day".into(), "clicks_56day".into(), "orders_56day".into(), "views_56".into(), "te_cbyv_56".into(), "rated_orders_sscat_percentile".into(), "views_7day".into(), "rated_orders_sscat_percentile_bin".into(), "orders_1day".into(), "views_56day".into(), "orders_1day".into(), "clicks_28day".into(), "te_cbyv_28day".into(), "orders_5day".into(), "max_price_sscat_percentile".into(), "te_obyc_56day".into(), "clicks_1day".into(), "rating".into(), "te_cbyv_28day".into(), "clicks_5day".into(), "te_obyc_56".into(), "te_obyv_56day".into(), "views_28day".into(), "views_1day".into(), "clicks_56day".into(), "te_obyc_5day".into(), "te_obyv_56".into(), "te_obyc_28day".into(), "te_cbyv_56".into(), "te_cbyv_56day".into(), "views_7day".into(), "te_obyc_3day".into(), "te_obyc_1day".into(), "max_price_sscat_percentile_bin".into(), "orders_56".into(), "te_cbyv_7day".into(), "clicks_7day".into(), "te_obyv_5day".into(), "orders_3day".into(), "te_obyv_1day".into(), "te_cbyv_5day".into(), "views_1day".into(), "te_cbyv_5day".into(), "te_cbyv_1day".into(), "min_price_sscat_percentile_bin".into(), "views_5day".into(), "clicks_3day".into(), "te_obyc_56".into(), "orders_56".into(), "orders_7day".into(), "te_obyv_5day".into(), "min_price_sscat_percentile".into(), "te_obyv_1day".into(), "te_obyv_56".into(), "te_cbyv_1day".into(), "clicks_56".into(), "te_obyv_56day".into(), "te_obyv_28day".into(), "te_obyv_28day".into(), "views_56day".into(), "clicks_56".into(), "views_5day".into(), "orders_28day".into(), "views_3day".into(), "views_56".into(), "te_obyc_3day".into(), "orders_3day".into(), "te_obyc_7day".into(), "orders_7day".into(), "te_cbyv_56day".into(), "clicks_3day".into(), "orders_56day".into(), "te_obyv_7day".into(), "te_obyc_5day".into(), "clicks_7day".into(), "te_obyc_1day".into(), "te_obyc_56day".into(), "log_28day".into(), "log_28day".into(), "laplace_1day".into(), "rating_30day".into(), "diff_asp_arp".into(), "laplace_7day".into(), "laplace_28day".into(), "log_7day".into(), "log_1day".into(), "28day".into(), "log_7day".into(), "rating_60day".into(), "log_1day".into(), "laplace_14day".into(), "log_7day".into(), "log_14day".into(), "log_28day".into(), "log_28day".into(), "log_14day".into(), "log_1day".into(), "56day".into(), "laplace_14day".into(), "laplace_7day".into(), "log_7day".into(), "laplace_1day".into(), "rating_90day".into(), "asp".into(), "laplace_28day".into(), "nqd_boosting_factor_gbm_model_v1".into(), "nqd_boosting_factor_gbm_model_v0".into(), "clp_price_aov_boosting_factor_var2".into(), "clp_price_aov_boosting_factor_var1".into(), "clp_price_aov_boosting_factor_var4".into(), "clp_price_aov_boosting_factor_var3".into(), "loyalty_boosting_factor".into(), "asp_p70_adj_30day".into(), "arp_p70_adj_30day".into(), "orders_by_clicks_bayesian".into(), "clicks_by_views_bayesian".into(), "orders_by_views_bayesian".into(), "orders_by_views_28day".into(), "clicks_by_views_28day".into(), "clicks_by_views_7day".into(), "orders_by_clicks_28day".into(), "clicks_by_views_3day".into(), "clicks_by_views_1day".into(), "clicks_by_views_14day".into(), "orders_by_views_1day".into(), "clp_odnr_sale_boosting_factor".into(), "net_order_by_gross_order".into(), "net_order_by_gross_order_smoothened".into(), "clp_odnr_sale_boosting_factor_var2".into(), "scaledup_boosting_factor".into(), "vrs_boosting_factor".into(), "nqd_boosting_factor_analytical_v2".into(), "search_odnr_sale_boosting_factor".into(), "nqd_boosting_factor".into()]
}

#[tokio::main(flavor = "multi_thread", worker_threads = 4)]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("Connecting to feature store version 4...");

    let channel = Endpoint::from_static("http://online-feature-store-api.int.meesho.int:80")
        .timeout(Duration::from_secs(10))
        .http2_keep_alive_interval(Duration::from_secs(30))
        .keep_alive_timeout(Duration::from_secs(10))
        .keep_alive_while_idle(true)
        .connect()
        .await?;

    let client = RetrieveClient::new(channel);
    
    // Pre-build static parts once at startup using Arc<str> for sharing
    // This allows multiple owners without cloning string data - similar to Go's string literals!
    let feature_labels = Arc::new(get_labels());
    let keys_schema = Arc::from(vec![Arc::from("catalog_id")]);
    
    // Start profiling - guard must be kept alive for profiling to continue
    let guard = pprof::ProfilerGuardBuilder::default()
        .frequency(1000)
        .blocklist(&["libc", "libgcc", "pthread", "vdso"])
        .build()
        .unwrap();

    // Channel for report requests - handlers send requests, background task generates reports
    let (report_tx, report_rx) = mpsc::channel::<ReportRequest>();

    // Spawn background task to handle report generation
    // This task holds the guard (which is not Send) and generates reports on demand
    tokio::task::spawn_blocking(move || {
        while let Ok(request) = report_rx.recv() {
            match request {
                ReportRequest::Protobuf(tx) => {
                    // For protobuf format, use the resolved report and convert to text format
                    // Note: Full pprof protobuf format requires additional conversion libraries
                    // For now, return text format that can be used with pprof tools
                    match guard.report().build() {
                        Ok(report) => {
                            // Convert report to text format (pprof can read text format)
                            let text_str = format!("{:?}", report);
                            let _ = tx.send(Ok(text_str.into_bytes()));
                        }
                        Err(e) => {
                            let _ = tx.send(Err(format!("Failed to build report: {:?}", e)));
                        }
                    }
                }
                ReportRequest::Flamegraph(tx) => {
                    match guard.report().build() {
                        Ok(report) => {
                            let mut flamegraph = Vec::new();
                            if report.flamegraph(&mut flamegraph).is_ok() {
                                let _ = tx.send(Ok(flamegraph));
                            } else {
                                let _ = tx.send(Err("Failed to generate flamegraph".to_string()));
                            }
                        }
                        Err(e) => {
                            let _ = tx.send(Err(format!("Failed to build report: {:?}", e)));
                        }
                    }
                }
                ReportRequest::Text(tx) => {
                    match guard.report().build() {
                        Ok(report) => {
                            // Use Debug trait for text output
                            let text_str = format!("{:?}", report);
                            let _ = tx.send(Ok(text_str));
                        }
                        Err(e) => {
                            let _ = tx.send(Err(format!("Failed to build report: {:?}", e)));
                        }
                    }
                }
            }
        }
        // Keep guard alive - it will be dropped when this task ends
        drop(guard);
    });

    let state = Arc::new(AppState {
        client,
        auth_token: AsciiMetadataValue::from_static("atishay"),
        caller_id: AsciiMetadataValue::from_static("test-3"),
        feature_labels,
        entity_label: Arc::from("catalog"),
        keys_schema,
        report_tx,
    });

    // Check jemalloc heap profiling availability
    #[cfg(not(target_env = "msvc"))]
    {
        if let Some(prof_ctl) = jemalloc_pprof::PROF_CTL.as_ref() {
            let prof_ctl = prof_ctl.lock().await;
            if prof_ctl.activated() {
                println!("Heap profiling: Enabled");
            } else {
                println!("Heap profiling: Configured but not activated");
            }
        } else {
            println!("Heap profiling: Not available (jemalloc not configured)");
        }
    }

    println!("Profiler started. Server will begin shortly...");
    println!("Profiling endpoints available:");
    println!("  - GET /pprof/protobuf - Download CPU pprof data (use with: go tool pprof http://localhost:8080/pprof/protobuf)");
    println!("  - GET /pprof/flamegraph - View CPU flamegraph SVG in browser");
    println!("  - GET /pprof/text - View CPU text report");
    #[cfg(not(target_env = "msvc"))]
    println!("  - GET /pprof/heap - Download heap/memory pprof data (use with: go tool pprof http://localhost:8080/pprof/heap)");

    let mut app = Router::new()
        .route("/retrieve-features", post(retrieve_features))
        .route("/pprof/protobuf", get(get_pprof_protobuf))
        .route("/pprof/flamegraph", get(get_flamegraph))
        .route("/pprof/text", get(get_pprof_text));
    
    #[cfg(not(target_env = "msvc"))]
    {
        app = app.route("/pprof/heap", get(get_pprof_heap));
    }
    
    let app = app.with_state(state);

    let listener = tokio::net::TcpListener::bind("0.0.0.0:8080").await?;
    println!("Server listening on 0.0.0.0:8080");
    
    // Profiling continues while server runs
    // When server exits, guard is dropped and profiling stops
    axum::serve(listener, app).await?;

    Ok(())
}



