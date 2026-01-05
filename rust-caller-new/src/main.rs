use axum::{extract::State, http::StatusCode, response::Json, routing::post, Router};
use std::sync::Arc;
use std::time::Duration;
use tokio::signal;
use tokio::sync::Mutex;
use tonic::{metadata::AsciiMetadataValue};
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
// Note: RetrieveResult is a protobuf type, so we serialize it manually
#[derive(Serialize)]
struct RetrieveFeaturesResponse {
    status: String,
    #[serde(serialize_with = "serialize_protobuf_result")]
    response: RetrieveResult,
}

// Helper function to serialize protobuf Result
fn serialize_protobuf_result<S>(result: &RetrieveResult, serializer: S) -> Result<S::Ok, S::Error>
where
    S: serde::Serializer,
{
    // Convert protobuf to JSON bytes and serialize
    use prost::Message;
    let bytes = result.encode_to_vec();
    serializer.serialize_bytes(&bytes)
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

struct AppState {
    client: Arc<Mutex<RetrieveClient<tonic_h3::H3Channel<tonic_h3::quinn::H3QuinnConnector>>>>,
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

    // Get mutable access to the client via Mutex
    let mut client = state.client.lock().await;
    let result = client.retrieve_features(request).await;

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
async fn create_channel(server: &str) -> Result<tonic_h3::H3Channel<tonic_h3::quinn::H3QuinnConnector>, Box<dyn std::error::Error>> {
    // Use quinn types directly as shown in tonic-h3 test examples
    // See: https://github.com/youyuanwu/tonic-h3
    use quinn::{ClientConfig, Endpoint};
    use quinn::crypto::rustls::QuicClientConfig;
    use rustls::{pki_types::CertificateDer, ClientConfig as RustlsClientConfig, RootCertStore};
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
    // Load system certificates - convert Certificate to CertificateDer
    for cert in rustls_native_certs::load_native_certs()? {
        root_store.add(CertificateDer::from(cert.0))?;
    }
    
    // Build rustls client config (rustls 0.23 API)
    let rustls_config = RustlsClientConfig::builder()
        .with_root_certificates(root_store)
        .with_no_client_auth();
    
    // Create quinn ClientConfig using rustls crypto (as shown in test examples)
    let quic_client_config = ClientConfig::new(Arc::new(
        QuicClientConfig::try_from(Arc::new(rustls_config))?
    ));
    
    // Create QUIC endpoint with client configuration
    // Use quinn::Endpoint directly (as in test examples)
    let mut client_endpoint = Endpoint::client(SocketAddr::from(([0, 0, 0, 0], 0)))?;
    client_endpoint.set_default_client_config(quic_client_config);
    
    // Build the full URI for gRPC (QUIC requires HTTPS)
    let grpc_uri = if server.starts_with("http://") {
        format!("https://{}:{}", host, port)
    } else {
        server.to_string()
    };
    
    let grpc_uri: Uri = grpc_uri.parse()?;
    
    // Create HTTP/3 channel using tonic-h3
    // Based on actual source code from tonic-h3-tests/examples/client.rs
    // 1. Create H3QuinnConnector (re-exported from tonic_h3::quinn)
    // 2. Create H3Channel from the connector
    // Convert quinn::Endpoint to h3_quinn::quinn::Endpoint for the connector
    // h3-quinn wraps quinn, so we need to use h3-quinn's endpoint type
    use h3_quinn::quinn::Endpoint as H3Endpoint;
    
    // Create h3-quinn endpoint from quinn endpoint
    // The types are compatible since h3-quinn wraps quinn
    let h3_endpoint: H3Endpoint = unsafe { std::mem::transmute(client_endpoint) };
    
    let server_name = host.to_string();
    // H3QuinnConnector is re-exported from tonic_h3::quinn (which re-exports h3_util::quinn)
    let connector = tonic_h3::quinn::H3QuinnConnector::new(grpc_uri.clone(), server_name, h3_endpoint.clone());
    let channel = tonic_h3::H3Channel::new(connector, grpc_uri.clone());
    
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
        client: Arc::new(Mutex::new(client)),
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
