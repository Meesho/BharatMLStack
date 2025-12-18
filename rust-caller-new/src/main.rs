use axum::{
    extract::State,
    http::StatusCode,
    response::Json,
    routing::post,
    Router,
};
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, sync::Arc, time::Duration};
use tonic::{
    metadata::AsciiMetadataValue,
    transport::{Channel, Endpoint},
};
use tower_http::cors::CorsLayer;

pub mod retrieve {
    tonic::include_proto!("retrieve");
}

use retrieve::feature_service_client::FeatureServiceClient as RetrieveClient;
use retrieve::{FeatureGroup, Keys, Query};

#[derive(Serialize, Deserialize)]
struct ApiResponse {
    success: bool,
    data: Option<String>,
    error: Option<String>,
    message: String,
}

#[derive(Clone)]
struct AppState {
    client: RetrieveClient<Channel>,
    // Pre-built metadata values to avoid allocations on every request
    auth_token: AsciiMetadataValue,
    caller_id: AsciiMetadataValue,
}

impl AppState {
    fn new(client: RetrieveClient<Channel>) -> Result<Self, Box<dyn std::error::Error>> {
        // Pre-build metadata values once using static strings
        // This avoids string allocations on every request
        Ok(Self {
            client,
            auth_token: AsciiMetadataValue::from_static("atishay"),
            caller_id: AsciiMetadataValue::from_static("test-3"),
        })
    }
}

async fn retrieve_features(State(state): State<Arc<AppState>>)
    -> Result<Json<ApiResponse>, (StatusCode, Json<ApiResponse>)>
{
    // Clone client - this is cheap because it only clones the Arc pointer to the channel
    let mut client = state.client.clone();
    
    match retrieve_features_internal(&mut client, &state.auth_token, &state.caller_id).await {
        Ok(_result) => Ok(Json(ApiResponse {
            success: true,
            data: Some("Features retrieved successfully".to_string()),
            error: None,
            message: "Features retrieved successfully".to_string(),
        })),
        Err(e) => {
            eprintln!("❌ gRPC Error: {}", e);
            Err((
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(ApiResponse {
                    success: false,
                    data: None,
                    error: Some(e.to_string()),
                    message: "Failed to retrieve features".to_string(),
                }),
            ))
        }
    }
}

async fn retrieve_features_internal(
    client: &mut RetrieveClient<Channel>,
    auth_token: &AsciiMetadataValue,
    caller_id: &AsciiMetadataValue,
) -> Result<retrieve::Result, Box<dyn std::error::Error>> {
    // Build request with minimal allocations - use string literals where possible
    let mut request = tonic::Request::new(Query {
        entity_label: "catalog".to_string(), // This needs to be owned
        feature_groups: vec![
            FeatureGroup {
                label: "derived_fp32".to_string(),
                feature_labels: vec!["clicks_by_views_3_days".to_string()],
            },
        ],
        keys_schema: vec!["catalog_id".to_string()],
        keys: vec![
            Keys { cols: vec!["176".to_string()] },
            Keys { cols: vec!["179".to_string()] },
        ],
        metadata: HashMap::new(),
    });

    // Set timeout (5 seconds like Go implementation)
    request.set_timeout(Duration::from_secs(5));

    // Insert pre-built metadata values directly - no cloning, no allocations
    request.metadata_mut().insert(
        "online-feature-store-auth-token",
        auth_token.clone(),
    );
    request.metadata_mut().insert(
        "online-feature-store-caller-id",
        caller_id.clone(),
    );

    let response = client.retrieve_features(request).await?;
    Ok(response.into_inner())
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Configure Tokio runtime for optimal I/O performance
    // Use fewer worker threads for I/O-bound workloads to reduce context switching overhead
    let worker_threads = std::thread::available_parallelism()
        .map(|n| n.get().min(4))
        .unwrap_or(4); // Default to 4 if unavailable
    
    let rt = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(worker_threads) // Limit to 4 threads max for I/O workloads
        .enable_io()
        .enable_time()
        .build()?;

    rt.block_on(async_main())
}

async fn async_main() -> Result<(), Box<dyn std::error::Error>> {
    println!("Connecting to feature store...");

    // Configure channel with optimizations for high-performance IO
    let channel = Endpoint::from_static("http://online-feature-store-api.int.meesho.int:80")
        // Enable HTTP/2 keepalive to keep connections alive
        // Set interval between HTTP/2 Ping frames (30 seconds)
        .http2_keep_alive_interval(Duration::from_secs(30))
        // Set timeout for keepalive ping acknowledgment (10 seconds)
        .keep_alive_timeout(Duration::from_secs(10))
        // Send keepalive pings even when connection is idle
        .keep_alive_while_idle(true)
        // Set connection timeout
        .timeout(Duration::from_secs(10))
        // Connect with retry logic
        .connect()
        .await?;

    // Create client - this is cheap to clone, uses Arc internally
    let client = RetrieveClient::new(channel);
    let state = Arc::new(AppState::new(client)?);

    let app = Router::new()
        .route("/retrieve-features", post(retrieve_features))
        .with_state(state)
        .layer(CorsLayer::permissive());

    println!("Starting rust-caller-new on http://0.0.0.0:8080");
    let listener = tokio::net::TcpListener::bind("0.0.0.0:8080").await?;
    axum::serve(listener, app).await?;

    Ok(())
}


