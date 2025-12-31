use axum::{extract::State, http::StatusCode, response::Json, routing::post, Router};
use std::sync::Arc;
use std::time::Duration;
use tokio::signal;
use tonic::{metadata::AsciiMetadataValue, transport::{Channel, Endpoint}};

pub mod retrieve {
    tonic::include_proto!("retrieve");
}

use retrieve::feature_service_client::FeatureServiceClient as RetrieveClient;
use retrieve::{FeatureGroup, Keys, Result as RetrieveResult};
use serde::{Deserialize, Serialize};

// Response structure matching Go's response format
#[derive(Serialize)]
struct RetrieveFeaturesResponse {
    status: String,
    response: RetrieveResult,
}

// Request body structure for retrieve_features endpoint
#[derive(Debug, Deserialize)]
struct RetrieveFeaturesRequest {
    entity_label: String,
    feature_groups: Vec<FeatureGroupRequest>,
    keys_schema: Vec<String>,
    keys: Vec<KeysRequest>,
}
#[derive(Debug, Deserialize)]
struct FeatureGroupRequest {
    label: String,
    feature_labels: Vec<String>,
}
#[derive(Debug, Deserialize)]
struct KeysRequest {
    cols: Vec<String>,
}

#[derive(Clone)]
struct AppState {
    client: RetrieveClient<Channel>,
    auth_token: AsciiMetadataValue,
    caller_id: AsciiMetadataValue,
}
async fn retrieve_features(
    State(state): State<Arc<AppState>>,
    Json(request_body): Json<RetrieveFeaturesRequest>,
) -> Result<Json<RetrieveFeaturesResponse>, StatusCode> {
    // Convert request body to protobuf Query
    let mut feature_groups = Vec::new();
    for fg in request_body.feature_groups {
        feature_groups.push(FeatureGroup {
            label: fg.label,
            feature_labels: fg.feature_labels,
        });
    }
    
    let mut keys = Vec::new();
    for k in request_body.keys {
        keys.push(Keys { cols: k.cols });
    }
    
    let query = retrieve::Query {
        entity_label: request_body.entity_label,
        feature_groups,
        keys_schema: request_body.keys_schema,
        keys,
    };

    let mut request = tonic::Request::new(query);
    // Increased timeout to 10s to handle high load scenarios without premature timeouts
    request.set_timeout(Duration::from_secs(10));
    request.metadata_mut().insert("online-feature-store-auth-token", state.auth_token.clone());
    request.metadata_mut().insert("online-feature-store-caller-id", state.caller_id.clone());

    let result = state.client.clone().retrieve_features(request).await;

    match result {
        Ok(response) => {
            let result = response.into_inner();
            Ok(Json(RetrieveFeaturesResponse {
                status: "success".to_string(),
                response: result,
            }))
        }
        Err(_) => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
}

#[tokio::main(flavor = "multi_thread")]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("Starting rust-caller version 8...");

    // Create a single gRPC channel
    let channel = Endpoint::from_static("http://online-feature-store-api.int.meesho.int:80")
        .http2_keep_alive_interval(Duration::from_secs(30))
        .keep_alive_timeout(Duration::from_secs(10))
        .keep_alive_while_idle(true)
        .connect()
        .await?;

    let client = RetrieveClient::new(channel);
    println!("Created gRPC connection");

    let state = Arc::new(AppState {
        client,
        auth_token: AsciiMetadataValue::from_static("atishay"),
        caller_id: AsciiMetadataValue::from_static("test-3"),
    });

    let app = Router::new()
        .route("/retrieve-features", post(retrieve_features));

    let app = app.with_state(state);

    // Configure TCP listener for high concurrency
    let listener = tokio::net::TcpListener::bind("0.0.0.0:8080").await?;

    println!("🚀 Rust gRPC Client running on http://0.0.0.0:8080");
    println!("Configured for high performance:");
    println!("  - Single gRPC connection");
    println!("  - Tokio runtime using all CPU cores");
    println!("  - HTTP/2 window sizes optimized for throughput");

    // Create shutdown signal handler
    let shutdown_signal = async {
        let ctrl_c = async {
            signal::ctrl_c()
                .await
                .expect("failed to install Ctrl+C handler");
        };

        #[cfg(unix)]
        let terminate = async {
            signal::unix::signal(signal::unix::SignalKind::terminate())
                .expect("failed to install signal handler")
                .recv()
                .await;
        };

        #[cfg(not(unix))]
        let terminate = std::future::pending::<()>();

        tokio::select! {
            _ = ctrl_c => {},
            _ = terminate => {},
        }

        println!("\nShutting down server...");
    };

    // Start server in background task
    let server_handle = tokio::spawn(async move {
        if let Err(e) = axum::serve(listener, app).await {
            eprintln!("Server error: {}", e);
        }
    });

    // Wait for shutdown signal
    shutdown_signal.await;

    // Graceful shutdown with timeout (similar to go-caller's 5 second timeout)
    println!("Waiting for in-flight requests to complete...");
    let shutdown_timeout = tokio::time::sleep(Duration::from_secs(5));
    let abort_handle = server_handle.abort_handle();
    
    tokio::select! {
        result = server_handle => {
            match result {
                Ok(_) => println!("Server stopped gracefully"),
                Err(e) => eprintln!("Server task error: {}", e),
            }
        }
        _ = shutdown_timeout => {
            println!("Shutdown timeout reached, forcing shutdown");
            abort_handle.abort();
            // Wait a bit for abort to complete
            tokio::time::sleep(Duration::from_millis(100)).await;
        }
    }
    
    // Channel will be closed when dropped
    println!("Server exited");

    Ok(())
}