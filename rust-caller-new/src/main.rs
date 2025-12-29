use axum::{
    extract::State,
    http::StatusCode,
    response::Json,
    routing::post,
    Router,
};
use serde::{Deserialize, Serialize};
use std::net::SocketAddr;
use std::sync::Arc;
use std::time::Duration;
use tower_http::cors::CorsLayer;
use tower_http::trace::TraceLayer;
use tracing::{info, Level};
use tonic::{metadata::AsciiMetadataValue, transport::Channel, Request};

// Include the generated proto code
pub mod retrieve {
    tonic::include_proto!("retrieve");
}

use retrieve::feature_service_client::FeatureServiceClient;
use retrieve::{FeatureGroup, Keys, Query};

// Request body structures for retrieve_features endpoint
#[derive(Debug, Deserialize, Serialize)]
struct RetrieveFeaturesRequest {
    #[serde(rename = "entity_label")]
    entity_label: String,
    #[serde(rename = "feature_groups")]
    feature_groups: Vec<FeatureGroupRequest>,
    #[serde(rename = "keys_schema")]
    keys_schema: Vec<String>,
    keys: Vec<KeysRequest>,
}

#[derive(Debug, Deserialize, Serialize)]
struct FeatureGroupRequest {
    label: String,
    #[serde(rename = "feature_labels")]
    feature_labels: Vec<String>,
}

#[derive(Debug, Deserialize, Serialize)]
struct KeysRequest {
    cols: Vec<String>,
}

// AppState stores gRPC client and pre-parsed metadata for zero-copy
#[derive(Clone)]
struct AppState {
    client: Arc<FeatureServiceClient<Channel>>,
    auth_token: AsciiMetadataValue,
    caller_id: AsciiMetadataValue,
}

// Pre-allocated static error responses to avoid allocations
static ERROR_TIMEOUT: &str = "Request timeout";

impl AppState {
    async fn handler(
        State(state): State<AppState>,
        Json(request_body): Json<RetrieveFeaturesRequest>,
    ) -> Result<Json<serde_json::Value>, (StatusCode, Json<serde_json::Value>)> {
        // Zero-copy optimization: Pre-allocate vectors with known capacity
        let feature_groups_len = request_body.feature_groups.len();
        let keys_len = request_body.keys.len();
        
        // Move ownership efficiently - pre-allocate to avoid reallocations
        let mut feature_groups = Vec::with_capacity(feature_groups_len);
        feature_groups.extend(
            request_body.feature_groups.into_iter().map(|fg| FeatureGroup {
                label: fg.label,
                feature_labels: fg.feature_labels,
            })
        );
        
        let mut keys = Vec::with_capacity(keys_len);
        keys.extend(
            request_body.keys.into_iter().map(|k| Keys { cols: k.cols })
        );

        // Create request with moved data (zero-copy move semantics)
        let mut request = Request::new(Query {
            entity_label: request_body.entity_label,
            feature_groups,
            keys_schema: request_body.keys_schema,
            keys,
        });

        // Zero-copy: Use pre-parsed metadata values (no string parsing per request)
        let metadata = request.metadata_mut();
        metadata.insert("online-feature-store-auth-token", state.auth_token.clone());
        metadata.insert("online-feature-store-caller-id", state.caller_id.clone());

        // Call gRPC service with timeout
        // Note: tonic::Client is cheap to clone (internally uses Arc), but we avoid explicit clone
        match tokio::time::timeout(
            Duration::from_secs(5),
            state.client.retrieve_features(request)
        ).await {
            Ok(Ok(_response)) => Ok(Json(serde_json::json!("success"))),
            Ok(Err(e)) => {
                // Only allocate error string when needed
                let error_response = serde_json::json!({
                    "error": e.to_string()
                });
                Err((StatusCode::INTERNAL_SERVER_ERROR, Json(error_response)))
            }
            Err(_) => {
                // Use static string reference to avoid allocation
                let error_response = serde_json::json!({
                    "error": ERROR_TIMEOUT
                });
                Err((StatusCode::INTERNAL_SERVER_ERROR, Json(error_response)))
            }
        }
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("Starting rust-caller with 4 threads version 4");
    
    // Initialize tracing
    tracing_subscriber::fmt()
        .with_target(false)
        .with_max_level(Level::INFO)
        .init();

    // Create gRPC channel with keepalive settings matching Go configuration
    // Time: 30 seconds, Timeout: 10 seconds, PermitWithoutStream: true
    let channel = Channel::from_static("http://online-feature-store-api.int.meesho.int:80")
        .http2_keep_alive_interval(Duration::from_secs(30))
        .keep_alive_timeout(Duration::from_secs(10))
        .keep_alive_while_idle(true)
        .connect()
        .await?;

    let client = Arc::new(FeatureServiceClient::new(channel));
    
    // Pre-parse metadata values once (zero-copy optimization)
    // This avoids parsing strings on every request
    let auth_token = AsciiMetadataValue::from_static("atishay");
    let caller_id = AsciiMetadataValue::from_static("test-3");
    
    let app_state = AppState {
        client,
        auth_token,
        caller_id,
    };

    // Build the application router
    let app = Router::new()
        .route("/retrieve-features", post(AppState::handler))
        .layer(CorsLayer::permissive())
        .layer(TraceLayer::new_for_http())
        .with_state(app_state);

    // Start the server
    let addr = SocketAddr::from(([0, 0, 0, 0], 8080));
    info!("🚀 Rust gRPC Client running on http://0.0.0.0:8080");

    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}
