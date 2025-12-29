use hyper::server::conn::http1;
use hyper::service::service_fn;
use hyper::{body::Incoming as IncomingBody, Request, Response, StatusCode};
use hyper_util::rt::TokioIo;
use http_body_util::{BodyExt, Full};
use hyper::body::Bytes;
use serde::Deserialize;
use std::convert::Infallible;
use std::sync::Arc;
use std::time::Duration;
use tokio::net::TcpListener;
use tonic::{metadata::AsciiMetadataValue, transport::Channel, Request as TonicRequest};

pub mod retrieve {
    tonic::include_proto!("retrieve");
}

use retrieve::feature_service_client::FeatureServiceClient;
use retrieve::{FeatureGroup, Keys, Query};

// RetrieveFeaturesRequest matches Query structure exactly
// This allows direct assignment without any conversion/copying
#[derive(Deserialize)]
struct RetrieveFeaturesRequest {
    #[serde(rename = "entity_label")]
    entity_label: String, 
    
    #[serde(rename = "feature_groups")]
    feature_groups: Vec<FeatureGroupRequest>,
    
    #[serde(rename = "keys_schema")]
    keys_schema: Vec<String>,
    
    keys: Vec<KeysRequest>,
}

#[derive(Deserialize)]
struct FeatureGroupRequest {
    label: String,
    #[serde(rename = "feature_labels")]
    feature_labels: Vec<String>,
}

#[derive(Deserialize)]
struct KeysRequest {
    cols: Vec<String>,
}

struct AppState {
    client: FeatureServiceClient<Channel>,
    auth_token: AsciiMetadataValue,
    caller_id: AsciiMetadataValue,
}

static SUCCESS: &[u8] = b"\"success\"";
static ERROR_TIMEOUT: &[u8] = b"{\"error\":\"Request timeout\"}";
static ERROR_BAD_REQUEST: &[u8] = b"Bad request";
static ERROR_INVALID_JSON: &[u8] = b"Invalid JSON";
static CONTENT_JSON: &str = "application/json";

async fn handler(
    req: Request<IncomingBody>,
    state: Arc<AppState>,
) -> Result<Response<Full<Bytes>>, Infallible> {
    // Collect body bytes - to_bytes() efficiently combines chunks
    let body_bytes = match req.into_body().collect().await {
        Ok(collected) => collected.to_bytes(),
        Err(_) => {
            return Ok(Response::builder()
                .status(StatusCode::BAD_REQUEST)
                .header("Content-Type", CONTENT_JSON)
                .body(Full::new(Bytes::from_static(ERROR_BAD_REQUEST)))
                .unwrap());
        }
    };

    // Parse JSON directly into RetrieveFeaturesRequest (matches Query structure)
    let request_body: RetrieveFeaturesRequest = match serde_json::from_slice(&body_bytes) {
        Ok(body) => body,
        Err(_) => {
            return Ok(Response::builder()
                .status(StatusCode::BAD_REQUEST)
                .header("Content-Type", CONTENT_JSON)
                .body(Full::new(Bytes::from_static(ERROR_INVALID_JSON)))
                .unwrap());
        }
    };

    // Direct assignment - ZERO COPY: moves ownership from request_body to Query
    // All fields match exactly, so we can move without any conversion
    let query = Query {
        entity_label: request_body.entity_label, // MOVE: zero-copy transfer
        feature_groups: request_body.feature_groups
            .into_iter() // Consumes Vec, moves elements (zero-copy)
            .map(|fg| FeatureGroup {
                label: fg.label, // MOVE: zero-copy transfer
                feature_labels: fg.feature_labels, // MOVE: zero-copy transfer
            })
            .collect(), // Only allocates Vec container, data is moved
        keys_schema: request_body.keys_schema, // MOVE: zero-copy transfer
        keys: request_body.keys
            .into_iter() // Consumes Vec, moves elements (zero-copy)
            .map(|k| Keys { 
                cols: k.cols // MOVE: zero-copy transfer
            })
            .collect(), // Only allocates Vec container, data is moved
    };

    // Create Tonic request with metadata - reuse metadata from state (no clone needed)
    let mut grpc_request = TonicRequest::new(query);
    grpc_request.metadata_mut().insert(
        "online-feature-store-auth-token",
        state.auth_token.clone(), // AsciiMetadataValue clone is cheap (Arc internally)
    );
    grpc_request.metadata_mut().insert(
        "online-feature-store-caller-id",
        state.caller_id.clone(), // AsciiMetadataValue clone is cheap (Arc internally)
    );

    // Clone client once - FeatureServiceClient::clone() is cheap (Arc internally)
    // We need clone because retrieve_features takes &mut self
    let mut client = state.client.clone();
    match tokio::time::timeout(Duration::from_secs(5), client.retrieve_features(grpc_request)).await {
        Ok(Ok(_)) => Ok(Response::builder()
            .status(StatusCode::OK)
            .header("Content-Type", CONTENT_JSON)
            .body(Full::new(Bytes::from_static(SUCCESS)))
            .unwrap()),
        Ok(Err(_)) => Ok(Response::builder()
            .status(StatusCode::INTERNAL_SERVER_ERROR)
            .header("Content-Type", CONTENT_JSON)
            .body(Full::new(Bytes::from_static(b"{\"error\":\"gRPC error\"}")))
            .unwrap()),
        Err(_) => Ok(Response::builder()
            .status(StatusCode::INTERNAL_SERVER_ERROR)
            .header("Content-Type", CONTENT_JSON)
            .body(Full::new(Bytes::from_static(ERROR_TIMEOUT)))
            .unwrap()),
    }
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Configure Tokio runtime to use all CPU cores for maximum performance
    // Default worker_threads() uses number of CPU cores automatically
    let rt = tokio::runtime::Builder::new_multi_thread()
        .enable_all()
        .build()?;
    
    rt.block_on(async_main())
}

async fn async_main() -> Result<(), Box<dyn std::error::Error>> {
    // Configure gRPC channel with HTTP/2 optimizations
    let channel = Channel::from_static("http://online-feature-store-api.int.meesho.int:80")
        .http2_keep_alive_interval(Duration::from_secs(30))
        .keep_alive_timeout(Duration::from_secs(10))
        .keep_alive_while_idle(true)
        .initial_stream_window_size(2 * 1024 * 1024) // 2MB stream window
        .initial_connection_window_size(4 * 1024 * 1024) // 4MB connection window
        .concurrency_limit(4000) // Allow up to 4000 concurrent requests
        .connect()
        .await?;

    let state = Arc::new(AppState {
        client: FeatureServiceClient::new(channel),
        auth_token: AsciiMetadataValue::from_static("atishay"),
        caller_id: AsciiMetadataValue::from_static("test-3"),
    });

    let listener = TcpListener::bind("0.0.0.0:8080").await?;

    // Create service once and reuse for all connections
    let service = service_fn({
        let state = state.clone();
        move |req| {
            let state = state.clone();
            handler(req, state)
        }
    });

    loop {
        let (stream, _) = listener.accept().await?;
        let io = TokioIo::new(stream);
        let service = service.clone();

        tokio::task::spawn(async move {
            // Use HTTP/1.1 with keep-alive for better connection reuse
            let mut builder = http1::Builder::new();
            builder.keep_alive(true);
            let _ = builder.serve_connection(io, service).await;
        });
    }
}
