use axum::{extract::State, http::StatusCode, response::Json, routing::post, Router};
use std::sync::Arc;
// use std::sync::mpsc;  // Commented out - pprof related
use std::time::Duration;
// use tokio::sync::oneshot;  // Commented out - pprof related
use tonic::{metadata::AsciiMetadataValue, transport::{Channel, Endpoint}};
use std::sync::atomic::{AtomicUsize, Ordering};

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
use serde::Deserialize;

// Constants to avoid repeated string allocations
const SUCCESS_RESPONSE: &str = "success";

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

// Report request types - COMMENTED OUT (pprof related)
// enum ReportRequest {
//     Protobuf(oneshot::Sender<Result<Vec<u8>, String>>),
//     Flamegraph(oneshot::Sender<Result<Vec<u8>, String>>),
//     Text(oneshot::Sender<Result<String, String>>),
// }

// Connection pool for gRPC clients to handle high concurrency
// HTTP/2 has stream limits per connection, so multiple connections are needed for 3k+ RPS
struct ClientPool {
    clients: Vec<RetrieveClient<Channel>>,
    counter: AtomicUsize,
}

impl ClientPool {
    fn get_client(&self) -> RetrieveClient<Channel> {
        // Round-robin selection to distribute load across connections
        let idx = self.counter.fetch_add(1, Ordering::Relaxed) % self.clients.len();
        self.clients[idx].clone()
    }
}

#[derive(Clone)]
struct AppState {
    client_pool: Arc<ClientPool>,
    auth_token: AsciiMetadataValue,
    caller_id: AsciiMetadataValue,
    // report_tx: mpsc::Sender<ReportRequest>,  // Commented out - pprof related
}

// Endpoint to get pprof data in protobuf format (for go tool pprof) - COMMENTED OUT
/* async fn get_pprof_protobuf(State(state): State<Arc<AppState>>) -> Result<Response<Body>, StatusCode> {
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
} */

// Endpoint to get flamegraph SVG - COMMENTED OUT
/* async fn get_flamegraph(State(state): State<Arc<AppState>>) -> Result<Response<Body>, StatusCode> {
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
} */

// Endpoint to get text report - COMMENTED OUT
/* async fn get_pprof_text(State(state): State<Arc<AppState>>) -> Result<Html<String>, StatusCode> {
    let (tx, rx) = oneshot::channel();
    state.report_tx.send(ReportRequest::Text(tx))
        .map_err(|_| StatusCode::SERVICE_UNAVAILABLE)?;
    
    match rx.await {
        Ok(Ok(text)) => Ok(Html(format!("<pre>{}</pre>", text))),
        _ => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
} */

// Endpoint to get heap/memory profiling data - COMMENTED OUT
/* #[cfg(not(target_env = "msvc"))]
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
} */

async fn retrieve_features(
    State(state): State<Arc<AppState>>,
    Json(request_body): Json<RetrieveFeaturesRequest>,
) -> Result<Json<String>, StatusCode> {
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

    // OPTIMIZATION: Use connection pool to distribute load across multiple HTTP/2 connections
    // This prevents hitting HTTP/2 stream limits on a single connection
    let client = state.client_pool.get_client();

    // OPTIMIZATION: Drop response immediately after checking success to reduce cleanup overhead
    // Based on flamegraph analysis: ~13-15% CPU was spent on drop_in_place for unused protobuf objects
    // (drop_in_place<Result>, drop_in_place<Response>, drop_in_place<Vec<Row>>, etc.)
    // By dropping explicitly in a smaller scope, we reduce the cleanup cost and memory pressure
    let result = client.clone().retrieve_features(request).await;
    
    match result {
        Ok(response) => {
            // OPTIMIZATION: Drop response immediately - don't wait for end of function
            // This reduces the time expensive drop operations hold resources
            // The response contains large protobuf structures (Vec<Row>, Feature, etc.) that are expensive to clean up
            // Since we don't use the response, dropping it immediately reduces memory pressure
            drop(response);
            Ok(Json(SUCCESS_RESPONSE.to_string()))
        }
        Err(_) => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
}

#[tokio::main(flavor = "multi_thread")]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Configure Tokio runtime for high performance
    // worker_threads = 0 means use all available CPU cores
    // This allows better CPU utilization for high RPS scenarios
    
    println!("Connecting to feature store version 4...");

    // PERFORMANCE FIX: Create multiple gRPC channels for connection pooling
    // HTTP/2 has stream limits (~100 concurrent streams per connection)
    // For 3k+ RPS, we need multiple connections to avoid hitting these limits
    // Using 10-20 connections should handle 3k-6k RPS comfortably
    const CONNECTION_POOL_SIZE: usize = 16;
    
    let mut clients = Vec::with_capacity(CONNECTION_POOL_SIZE);
    
    for i in 0..CONNECTION_POOL_SIZE {
        let channel = Endpoint::from_static("http://online-feature-store-api.int.meesho.int:80")
            .timeout(Duration::from_secs(10))
            // Optimized HTTP/2 settings for high concurrency
            .http2_keep_alive_interval(Duration::from_secs(30))
            .keep_alive_timeout(Duration::from_secs(10))
            .keep_alive_while_idle(true)
            // Increase initial window size for better throughput
            .initial_stream_window_size(Some(1024 * 1024 * 2)) // 2MB
            .initial_connection_window_size(Some(1024 * 1024 * 4)) // 4MB
            .connect()
            .await?;
        
        clients.push(RetrieveClient::new(channel));
        
        if (i + 1) % 4 == 0 {
            println!("Created {} gRPC connections...", i + 1);
        }
    }
    
    println!("Created {} gRPC connections for connection pooling", CONNECTION_POOL_SIZE);
    
    let client_pool = Arc::new(ClientPool {
        clients,
        counter: AtomicUsize::new(0),
    });
    
    // Start profiling - guard must be kept alive for profiling to continue - COMMENTED OUT
    // Higher frequency = more samples = better resolution
    // blocklist excludes low-level libraries to focus on application code
    /* let guard = pprof::ProfilerGuardBuilder::default()
        .frequency(10000)  // Increased from 1000 to 10000 for better resolution
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
    }); */

    let state = Arc::new(AppState {
        client_pool,
        auth_token: AsciiMetadataValue::from_static("atishay"),
        caller_id: AsciiMetadataValue::from_static("test-3"),
        // report_tx,  // Commented out - pprof related
    });

    // Check jemalloc heap profiling availability - COMMENTED OUT
    /* #[cfg(not(target_env = "msvc"))]
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
    println!("  - GET /pprof/heap - Download heap/memory pprof data (use with: go tool pprof http://localhost:8080/pprof/heap)"); */

    let app = Router::new()
        .route("/retrieve-features", post(retrieve_features));
        // Pprof routes commented out
        // .route("/pprof/protobuf", get(get_pprof_protobuf))
        // .route("/pprof/flamegraph", get(get_flamegraph))
        // .route("/pprof/text", get(get_pprof_text));
    
    // #[cfg(not(target_env = "msvc"))]
    // {
    //     app = app.route("/pprof/heap", get(get_pprof_heap));
    // }
    
    let app = app.with_state(state);

    // PERFORMANCE FIX: Configure TCP listener for high concurrency
    // The main bottleneck fix is connection pooling (done above)
    // Axum/Tokio handle TCP settings efficiently by default
    let listener = tokio::net::TcpListener::bind("0.0.0.0:8080").await?;
    
    println!("Server listening on 0.0.0.0:8080");
    println!("Configured for high performance:");
    println!("  - {} gRPC connection pool (main bottleneck fix)", CONNECTION_POOL_SIZE);
    println!("  - Tokio runtime using all CPU cores");
    println!("  - HTTP/2 window sizes optimized for throughput");
    
    // Profiling continues while server runs - COMMENTED OUT
    // When server exits, guard is dropped and profiling stops
    // Axum 0.7 uses axum::serve which handles high concurrency efficiently
    axum::serve(listener, app).await?;

    Ok(())
}



