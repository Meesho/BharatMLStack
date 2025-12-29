use hyper::server::conn::http1;
use hyper::service::service_fn;
use hyper::{body::Incoming as IncomingBody, Request, Response, StatusCode};
use hyper_util::rt::TokioIo;
use std::convert::Infallible;
use std::sync::Arc;
use std::time::Duration;
use tokio::net::TcpListener;
use tonic::{metadata::AsciiMetadataValue, transport::{Channel, Endpoint}};
use http_body_util::{BodyExt, Full};
use hyper::body::Bytes;

// Configure jemalloc - DISABLED profiling for production performance
// Profiling adds significant overhead (~5-10% CPU)
// Uncomment below for profiling, but expect lower RPS
// #[cfg(not(target_env = "msvc"))]
// #[global_allocator]
// static ALLOC: tikv_jemallocator::Jemalloc = tikv_jemallocator::Jemalloc;

// #[allow(non_upper_case_globals)]
// #[export_name = "malloc_conf"]
// pub static malloc_conf: &[u8] = b"prof:true,prof_active:true,lg_prof_sample:19\0";

pub mod retrieve {
    tonic::include_proto!("retrieve");
}

use retrieve::feature_service_client::FeatureServiceClient as RetrieveClient;
use retrieve::{FeatureGroup, Keys};
use serde::Deserialize;


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

#[derive(Clone)]
struct AppState {
    client: RetrieveClient<Channel>,
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

async fn retrieve_features_handler(
    req: Request<IncomingBody>,
    state: Arc<AppState>,
) -> Result<Response<Full<Bytes>>, Infallible> {

    let body = req.into_body();
    let collected = match body.collect().await {
        Ok(c) => c,
        Err(_) => {
            return Ok(Response::builder()
                .status(StatusCode::BAD_REQUEST)
                .body(Full::new(Bytes::from("Body Error")))
                .unwrap());
        }
    };
    
    let body_bytes = collected.to_bytes();

    let request_body: RetrieveFeaturesRequest = match serde_json::from_slice(&body_bytes) {
        Ok(body) => body,
        Err(e) => return Ok(Response::builder().status(StatusCode::BAD_REQUEST).body(Full::new(Bytes::from(e.to_string()))).unwrap()),
    };

    let query = retrieve::Query {
        entity_label: request_body.entity_label,
        feature_groups: request_body.feature_groups
            .into_iter()
            .map(|fg| FeatureGroup {
                label: fg.label,
                feature_labels: fg.feature_labels,
            })
            .collect(),
        keys_schema: request_body.keys_schema,
        keys: request_body.keys
            .into_iter()
            .map(|k| Keys { cols: k.cols })
            .collect(),
    };

    let mut grpc_request = tonic::Request::new(query);
    // Optimization: Add metadata using permanent AsciiMetadataValue to avoid re-parsing strings
    grpc_request.set_timeout(Duration::from_secs(10));
    grpc_request.metadata_mut().insert("online-feature-store-auth-token", state.auth_token.clone());
    grpc_request.metadata_mut().insert("online-feature-store-caller-id", state.caller_id.clone());
    
    match state.client.clone().retrieve_features(grpc_request).await {
        Ok(grpc_resp) => {
            let _inner_data = grpc_resp.into_inner();
            Ok(Response::builder()
                .status(StatusCode::OK)
                .header("Content-Type", "application/json")
                .body(Full::new(Bytes::from_static(b"\"success\"")))
                .unwrap())
        }
        Err(e) => {
            Ok(Response::builder()
                .status(StatusCode::INTERNAL_SERVER_ERROR)
                .body(Full::new(Bytes::from(format!("\"gRPC Error: {}\"", e))))
                .unwrap())
        }
    }
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Configure Tokio runtime explicitly for high performance
    // This ensures optimal CPU utilization
    // Note: Not setting worker_threads uses all available CPU cores by default
    let rt = tokio::runtime::Builder::new_multi_thread()
        .enable_all()
        .build()?;
    
    rt.block_on(async_main())
}

async fn async_main() -> Result<(), Box<dyn std::error::Error>> {
    println!("Connecting to feature store version 4...");

    // Single gRPC connection with optimizations enabled
    // NOTE: HTTP/2 has a hard limit of ~100 concurrent streams per connection
    // This limits throughput to ~1,000-1,500 RPS depending on latency
    // For higher throughput, consider using connection pooling
    let channel = Endpoint::from_static("http://online-feature-store-api.int.meesho.int:80")
        .timeout(Duration::from_secs(10))
        // Keep-alive settings (standard optimization)
        .http2_keep_alive_interval(Duration::from_secs(30))
        .keep_alive_timeout(Duration::from_secs(10))
        .keep_alive_while_idle(true)
        // Window sizes optimization for better throughput - ENABLED
        .initial_stream_window_size(1024 * 1024 * 2) // 2MB (default: 65,535 bytes)
        .initial_connection_window_size(1024 * 1024 * 4) // 4MB (default: 65,535 bytes)
        .concurrency_limit(4000) // Allow up to 4000 concurrent requests
        .connect()
        .await?;
    
    let client = RetrieveClient::new(channel);
    






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
        client,
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

    // Configure TCP listener
    // Hyper/Tokio handle TCP settings efficiently by default
    let listener = TcpListener::bind("0.0.0.0:8080").await?;
    
    println!("Server listening on 0.0.0.0:8080");
    println!("Configured for high performance:");
    println!("  - Single gRPC connection (limited to ~100 concurrent streams)");
    println!("  - Tokio runtime using all CPU cores");
    println!("  - HTTP/2 window sizes optimized: 2MB stream, 4MB connection");
    println!("  - concurrency_limit: 4000 (client-side protection)");
    println!("  - Using raw Hyper for maximum performance");
    
    // Accept connections in a loop
    loop {
        match listener.accept().await {
            Ok((stream, _)) => {
                let io = TokioIo::new(stream);
                let state_clone = state.clone();
                
                // Spawn a task to handle each connection
                tokio::task::spawn(async move {
                    let service = service_fn(move |req| {
                        let state = state_clone.clone();
                        retrieve_features_handler(req, state)
                    });
                    
                    // Use HTTP/1.1 connection
                    if let Err(err) = http1::Builder::new()
                        .serve_connection(io, service)
                        .await
                    {
                        eprintln!("Error serving connection: {:?}", err);
                    }
                });
            }
            Err(e) => {
                eprintln!("Error accepting connection: {:?}", e);
            }
        }
    }
}



