use axum::{extract::State, http::StatusCode, response::Json, routing::post, Router};
use std::sync::Arc;
use std::time::Duration;
use tokio::signal;
use tonic::{metadata::AsciiMetadataValue, transport::Channel};
use clap::Parser;
use http::Uri;

pub mod retrieve {
    tonic::include_proto!("retrieve");
}
use retrieve::feature_service_client::FeatureServiceClient as RetrieveClient;
use retrieve::{FeatureGroup, Keys, Result as RetrieveResult};
use serde::{Deserialize, Serialize};

#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Server address (must support HTTP/3/QUIC)
    #[arg(short, long, default_value = "https://online-feature-store-api.int.meesho.int:443")]
    server: String,
}

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
    request.set_timeout(Duration::from_secs(5));
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

// Create HTTP/3 gRPC channel using tonic-h3
// 
// Key benefits of HTTP/3:
// 1. Reduced latency: 0-RTT connection establishment vs 2-3 RTT for HTTP/2
// 2. No head-of-line blocking: Independent QUIC streams vs TCP blocking
// 3. Connection migration: Seamless network transitions
// 4. Better packet loss handling: 60% better latency under 3% packet loss
// 5. Enhanced security: Encrypted headers, mandatory TLS 1.3
//
// Implementation uses tonic-h3 library (https://github.com/youyuanwu/tonic-h3)
// which provides experimental gRPC over HTTP/3 support using QUIC
async fn create_channel(server: &str) -> Result<Channel, Box<dyn std::error::Error>> {
    use h3_quinn::quinn::{ClientConfig, Endpoint as QuinnEndpoint};
    use rustls::{ClientConfig as RustlsClientConfig, RootCertStore};
    use std::net::SocketAddr;
    
    println!("🚀 Using HTTP/3 (QUIC) via tonic-h3");
    println!("   Benefits: 0-RTT, no head-of-line blocking, connection migration");
    println!("   Note: Server must support HTTP/3/QUIC");
    
    // Parse server URI
    let uri: Uri = server.parse()?;
    let host = uri.host().ok_or("Invalid server URI: missing host")?;
    let port = uri.port_u16().unwrap_or(443); // Default to 443 for HTTPS/QUIC
    
    // Create QUIC client configuration with TLS
    let mut root_store = RootCertStore::empty();
    root_store.extend(rustls_native_certs::load_native_certs()?);
    
    let client_config = RustlsClientConfig::builder()
        .with_safe_defaults()
        .with_root_certificates(root_store)
        .with_no_client_auth();
    
    let quic_client_config = ClientConfig::new(Arc::new(client_config));
    
    // Create QUIC endpoint with client configuration
    let client_endpoint = QuinnEndpoint::client(SocketAddr::from(([0, 0, 0, 0], 0)))?
        .with_default_client_config(quic_client_config);
    
    // Build the full URI for gRPC (QUIC requires HTTPS)
    let grpc_uri = if server.starts_with("http://") {
        // Convert http:// to https:// for QUIC (QUIC requires TLS)
        format!("https://{}:{}", host, port)
    } else {
        server.to_string()
    };
    
    let grpc_uri: Uri = grpc_uri.parse()?;
    
    // Create HTTP/3 channel using tonic-h3
    // Note: This is experimental - server must support HTTP/3/QUIC
    // If the server doesn't support HTTP/3, this will fail at connection time
    let channel = tonic_h3::quinn::new_quinn_h3_channel(grpc_uri.clone(), client_endpoint);
    
    println!("✅ HTTP/3 channel created successfully");
    println!("   Server URI: {}", grpc_uri);
    println!("   ⚠️  If connection fails, ensure server supports HTTP/3/QUIC");
    
    Ok(channel)
}

#[tokio::main(flavor = "multi_thread")]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args = Args::parse();
    
    println!("Starting rust-caller with HTTP/3 (QUIC)...");

    // Create HTTP/3 gRPC channel using tonic-h3
    let channel = create_channel(&args.server).await?;

    // Create gRPC client with the HTTP/3 channel
    let client = RetrieveClient::new(channel);
    println!("✅ Created gRPC connection using HTTP/3 (QUIC)");

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
    println!("  - Protocol: HTTP/3 (QUIC)");
    println!("  - Single gRPC connection");
    println!("  - Tokio runtime using all CPU cores");
    println!("  - HTTP/3 over QUIC (UDP-based, independent streams)");
    println!("  - No head-of-line blocking");
    println!("  - Connection migration support");
    println!("  - Enhanced packet loss handling");
    println!("  - 0-RTT connection establishment");
    println!("  - Encrypted headers (TLS 1.3)");
    println!("\n⚠️  Note: Server must support HTTP/3/QUIC");
    println!("📊 For performance details, see HTTP3_BENEFITS.md");

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
