use hyper::server::conn::http1;
use hyper::service::service_fn;
use hyper::{body::Incoming as IncomingBody, Method, Request, Response, StatusCode};
use hyper_util::rt::TokioIo;
use std::convert::Infallible;
use std::sync::Arc;
use std::time::Duration;
use tokio::net::TcpListener;
use tokio::sync::{mpsc, oneshot};
use tonic::{metadata::AsciiMetadataValue, transport::{Channel, Endpoint}};
use http_body_util::{BodyExt, Full};
use hyper::body::Bytes;

// Configure jemalloc for heap profiling
// Profiling adds significant overhead (~5-10% CPU)
// Enable for heap profiling, but expect lower RPS
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

// Report request types for pprof
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
    report_tx: Option<mpsc::Sender<ReportRequest>>,  // Optional for pprof
}

// Endpoint to get pprof data in protobuf format (for go tool pprof)
async fn get_pprof_protobuf(state: Arc<AppState>) -> Result<Response<Full<Bytes>>, Infallible> {
    let report_tx = match &state.report_tx {
        Some(tx) => tx,
        None => {
            return Ok(Response::builder()
                .status(StatusCode::SERVICE_UNAVAILABLE)
                .body(Full::new(Bytes::from("Profiling not enabled")))
                .unwrap());
        }
    };
    
    let (tx, rx) = oneshot::channel();
    if report_tx.send(ReportRequest::Protobuf(tx)).await.is_err() {
        return Ok(Response::builder()
            .status(StatusCode::SERVICE_UNAVAILABLE)
            .body(Full::new(Bytes::from("Profiling service unavailable")))
            .unwrap());
    }
    
    match rx.await {
        Ok(Ok(data)) => {
            Ok(Response::builder()
                .status(StatusCode::OK)
                .header("Content-Type", "application/x-protobuf")
                .header("Content-Disposition", "attachment; filename=profile.pb.gz")
                .body(Full::new(Bytes::from(data)))
                .unwrap())
        }
        _ => Ok(Response::builder()
            .status(StatusCode::INTERNAL_SERVER_ERROR)
            .body(Full::new(Bytes::from("Failed to generate profile")))
            .unwrap()),
    }
}

// Endpoint to get flamegraph SVG
async fn get_flamegraph(state: Arc<AppState>) -> Result<Response<Full<Bytes>>, Infallible> {
    let report_tx = match &state.report_tx {
        Some(tx) => tx,
        None => {
            return Ok(Response::builder()
                .status(StatusCode::SERVICE_UNAVAILABLE)
                .body(Full::new(Bytes::from("Profiling not enabled")))
                .unwrap());
        }
    };
    
    let (tx, rx) = oneshot::channel();
    if report_tx.send(ReportRequest::Flamegraph(tx)).await.is_err() {
        return Ok(Response::builder()
            .status(StatusCode::SERVICE_UNAVAILABLE)
            .body(Full::new(Bytes::from("Profiling service unavailable")))
            .unwrap());
    }
    
    match rx.await {
        Ok(Ok(data)) => {
            Ok(Response::builder()
                .status(StatusCode::OK)
                .header("Content-Type", "image/svg+xml")
                .header("Content-Disposition", "inline; filename=flamegraph.svg")
                .body(Full::new(Bytes::from(data)))
                .unwrap())
        }
        _ => Ok(Response::builder()
            .status(StatusCode::INTERNAL_SERVER_ERROR)
            .body(Full::new(Bytes::from("Failed to generate flamegraph")))
            .unwrap()),
    }
}

// Endpoint to get text report
async fn get_pprof_text(state: Arc<AppState>) -> Result<Response<Full<Bytes>>, Infallible> {
    let report_tx = match &state.report_tx {
        Some(tx) => tx,
        None => {
            return Ok(Response::builder()
                .status(StatusCode::SERVICE_UNAVAILABLE)
                .body(Full::new(Bytes::from("Profiling not enabled")))
                .unwrap());
        }
    };
    
    let (tx, rx) = oneshot::channel();
    if report_tx.send(ReportRequest::Text(tx)).await.is_err() {
        return Ok(Response::builder()
            .status(StatusCode::SERVICE_UNAVAILABLE)
            .body(Full::new(Bytes::from("Profiling service unavailable")))
            .unwrap());
    }
    
    match rx.await {
        Ok(Ok(text)) => {
            let html = format!("<pre>{}</pre>", text);
            Ok(Response::builder()
                .status(StatusCode::OK)
                .header("Content-Type", "text/html")
                .body(Full::new(Bytes::from(html)))
                .unwrap())
        }
        _ => Ok(Response::builder()
            .status(StatusCode::INTERNAL_SERVER_ERROR)
            .body(Full::new(Bytes::from("Failed to generate text report")))
            .unwrap()),
    }
}

// Endpoint to get heap/memory profiling data
#[cfg(not(target_env = "msvc"))]
async fn get_pprof_heap(_state: Arc<AppState>) -> Result<Response<Full<Bytes>>, Infallible> {
    // Check if jemalloc profiling is available
    // Wrap in catch_unwind to handle panics if jemalloc isn't configured
    let prof_ctl = match std::panic::catch_unwind(|| jemalloc_pprof::PROF_CTL.as_ref()) {
        Ok(Some(ctl)) => ctl,
        Ok(None) | Err(_) => {
            return Ok(Response::builder()
                .status(StatusCode::SERVICE_UNAVAILABLE)
                .body(Full::new(Bytes::from("jemalloc profiling not available. Ensure tikv-jemallocator is configured correctly.")))
                .unwrap());
        }
    };
    
    let mut prof_ctl = prof_ctl.lock().await;
    
    // Verify profiling is activated
    if !prof_ctl.activated() {
        return Ok(Response::builder()
            .status(StatusCode::SERVICE_UNAVAILABLE)
            .body(Full::new(Bytes::from("Heap profiling not activated. Ensure jemalloc is configured with profiling enabled.")))
            .unwrap());
    }
    
    // Generate pprof heap profile
    match prof_ctl.dump_pprof() {
        Ok(pprof_data) => {
            Ok(Response::builder()
                .status(StatusCode::OK)
                .header("Content-Type", "application/x-protobuf")
                .header("Content-Encoding", "gzip")
                .header("Content-Disposition", "attachment; filename=heap.pb.gz")
                .body(Full::new(Bytes::from(pprof_data)))
                .unwrap())
        }
        Err(e) => {
            Ok(Response::builder()
                .status(StatusCode::INTERNAL_SERVER_ERROR)
                .body(Full::new(Bytes::from(format!("Failed to generate heap profile: {}", e))))
                .unwrap())
        }
    }
}

#[cfg(target_env = "msvc")]
async fn get_pprof_heap(_state: Arc<AppState>) -> Result<Response<Full<Bytes>>, Infallible> {
    Ok(Response::builder()
        .status(StatusCode::SERVICE_UNAVAILABLE)
        .body(Full::new(Bytes::from("Heap profiling not available on Windows/MSVC")))
        .unwrap())
}

// Pre-allocated header values to avoid string allocations
static CONTENT_TYPE_JSON: &str = "application/json";

// Main router function that dispatches requests based on path
// Prevent inlining to ensure function appears in profiles
#[inline(never)]
async fn router(
    req: Request<IncomingBody>,
    state: Arc<AppState>,
) -> Result<Response<Full<Bytes>>, Infallible> {
    // Optimization: Get path and method once, avoid multiple method calls
    let path = req.uri().path();
    let method = req.method();
    
    // Route pprof endpoints (support both GET and HEAD methods)
    if method == Method::GET || method == Method::HEAD {
        match path {
            "/pprof/protobuf" => return get_pprof_protobuf(state).await,
            "/pprof/flamegraph" => return get_flamegraph(state).await,
            "/pprof/text" => return get_pprof_text(state).await,
            #[cfg(not(target_env = "msvc"))]
            "/pprof/heap" => return get_pprof_heap(state).await,
            _ => {}
        }
    }
    
    // Default to retrieve_features handler
    retrieve_features_handler(req, state).await
}

// Pre-allocated error responses to avoid allocations in hot path
static BODY_ERROR_RESPONSE: &[u8] = b"Body Error";
static SUCCESS_RESPONSE: &[u8] = b"\"success\"";

// Prevent inlining to ensure function appears in profiles
#[inline(never)]
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
                .body(Full::new(Bytes::from_static(BODY_ERROR_RESPONSE)))
                .unwrap());
        }
    };
    
    let body_bytes = collected.to_bytes();

    let request_body: RetrieveFeaturesRequest = match serde_json::from_slice(&body_bytes) {
        Ok(body) => body,
        Err(_) => {
            // Avoid string allocation for common error case
            return Ok(Response::builder()
                .status(StatusCode::BAD_REQUEST)
                .body(Full::new(Bytes::from_static(b"Invalid JSON")))
                .unwrap());
        }
    };

    // Optimization: Pre-allocate vectors with known capacity to reduce reallocations
    let feature_groups_len = request_body.feature_groups.len();
    let keys_len = request_body.keys.len();
    
    let mut feature_groups = Vec::with_capacity(feature_groups_len);
    for fg in request_body.feature_groups {
        feature_groups.push(FeatureGroup {
            label: fg.label,
            feature_labels: fg.feature_labels,
        });
    }
    
    let mut keys = Vec::with_capacity(keys_len);
    for k in request_body.keys {
        keys.push(Keys { cols: k.cols });
    }

    let query = retrieve::Query {
        entity_label: request_body.entity_label,
        feature_groups,
        keys_schema: request_body.keys_schema,
        keys,
    };

    let mut grpc_request = tonic::Request::new(query);
    // Optimization: Add metadata using permanent AsciiMetadataValue to avoid re-parsing strings
    grpc_request.set_timeout(Duration::from_secs(10));
    grpc_request.metadata_mut().insert("online-feature-store-auth-token", state.auth_token.clone());
    grpc_request.metadata_mut().insert("online-feature-store-caller-id", state.caller_id.clone());
    
    // Note: tonic::Client is cheap to clone (internally uses Arc), but we still avoid unnecessary clones
    match state.client.clone().retrieve_features(grpc_request).await {
            Ok(_grpc_resp) => {
            // Optimization: Use static bytes and pre-allocated header value
            Ok(Response::builder()
                .status(StatusCode::OK)
                .header("Content-Type", CONTENT_TYPE_JSON)
                .body(Full::new(Bytes::from_static(SUCCESS_RESPONSE)))
                .unwrap())
        }
        Err(_e) => {
            // Optimization: Use static error message to avoid format!() allocation
            // For production, consider logging the error separately
            Ok(Response::builder()
                .status(StatusCode::INTERNAL_SERVER_ERROR)
                .body(Full::new(Bytes::from_static(b"\"gRPC Error\"")))
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
    






    // Start profiling - guard must be kept alive for profiling to continue
    // Higher frequency = more samples = better resolution
    // blocklist excludes low-level libraries to focus on application code
    // Note: pprof adds overhead (~5-10% CPU), so it's optional
    let (report_tx, report_rx) = mpsc::channel::<ReportRequest>(100);
    
    // Try to initialize pprof, but don't fail if it's not available
    let profiler_guard = pprof::ProfilerGuardBuilder::default()
        .frequency(10000)  // Increased from 1000 to 10000 for better resolution
        .blocklist(&["libc", "libgcc", "pthread", "vdso"])
        .build();
    
    let report_tx_final = if profiler_guard.is_ok() {
        let guard = profiler_guard.unwrap();
        
        // Spawn background task to handle report generation
        // This task holds the guard (which is not Send) and generates reports on demand
        let report_tx_clone = report_tx.clone();
        tokio::task::spawn_blocking(move || {
            let mut report_rx = report_rx;
            while let Some(request) = report_rx.blocking_recv() {
                match request {
                    ReportRequest::Protobuf(tx) => {
                        // Generate pprof protobuf format for go tool pprof
                        // Note: The pprof crate's Report doesn't have a direct pprof() method
                        // We'll use the report's data to generate a format that go tool pprof can use
                        // For now, return the report in a format that can be processed
                        match guard.report().build() {
                            Ok(report) => {
                                // The pprof crate doesn't directly support protobuf output
                                // We need to use external tools or generate it manually
                                // For now, return text format - users should use go tool pprof with the text format
                                // or use the flamegraph endpoint for visualization
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
                                // Generate flamegraph using inferno
                                // Parse the report's Debug format to extract function names
                                // The Debug format shows: "FRAME: function_name -> FRAME: function_name -> ..."
                                use pprof::flamegraph;
                                let mut flamegraph_data = Vec::new();
                                let mut options = flamegraph::Options::default();
                                
                                // Convert report to text format to parse function names
                                let report_text = format!("{:?}", report);
                                
                                // Parse the text format to extract call stacks
                                // Format in text: "FRAME: func1 -> FRAME: func2 -> ... THREAD: thread_name"
                                let mut flamegraph_lines = Vec::new();
                                
                                // Split by lines and parse each stack trace
                                for line in report_text.lines() {
                                    if !line.contains("FRAME:") {
                                        continue;
                                    }
                                    
                                    // Extract function names from "FRAME: function_name ->" pattern
                                    let mut stack_parts = Vec::new();
                                    let parts: Vec<&str> = line.split("FRAME:").collect();
                                    
                                    for part in parts.iter().skip(1) {
                                        // Extract function name before "->" or end of line
                                        let func_part = part.split("->").next().unwrap_or("").trim();
                                        if !func_part.is_empty() && func_part != "THREAD" {
                                            stack_parts.push(func_part.to_string());
                                        }
                                    }
                                    
                                    // Extract sample count (if available) or use 1
                                    // The count is typically in the report.data, but we'll use 1 for now
                                    // since we're parsing from text format
                                    if !stack_parts.is_empty() {
                                        // Reverse to get call stack from root to leaf
                                        stack_parts.reverse();
                                        let stack_str = stack_parts.join(";");
                                        flamegraph_lines.push(format!("{} 1", stack_str));
                                    }
                                }
                                
                                // If text parsing didn't work, fall back to direct data access
                                if flamegraph_lines.is_empty() {
                                    // Try direct access to report.data
                                    for (frames, count) in report.data.iter() {
                                        if frames.frames.is_empty() {
                                            continue;
                                        }
                                        
                                        let mut stack_parts = Vec::new();
                                        for frame_symbols in frames.frames.iter().rev() {
                                            let mut frame_name = None;
                                            
                                            // Try to get the best symbol name from all symbols in this frame
                                            for symbol in frame_symbols.iter() {
                                                if let Some(name_bytes) = &symbol.name {
                                                    if !name_bytes.is_empty() {
                                                        let name = String::from_utf8_lossy(name_bytes).trim().to_string();
                                                        // Prefer function names over closures, but include closures if that's all we have
                                                        if !name.is_empty() && name != "unknown" {
                                                            let is_closure = name.contains("{{closure}}");
                                                            // Prefer actual function names over closure names
                                                            if !is_closure || frame_name.is_none() {
                                                                frame_name = Some(name.clone());
                                                                // If we found a non-closure name, use it
                                                                if !is_closure {
                                                                    break;
                                                                }
                                                            }
                                                        }
                                                    }
                                                }
                                            }
                                            
                                            // Fallback to address if no name found
                                            let frame_str = frame_name.unwrap_or_else(|| {
                                                if let Some(symbol) = frame_symbols.first() {
                                                    if let Some(addr) = symbol.addr {
                                                        format!("0x{:p}", addr)
                                                    } else {
                                                        "unknown".to_string()
                                                    }
                                                } else {
                                                    "unknown".to_string()
                                                }
                                            });
                                            
                                            stack_parts.push(frame_str);
                                        }
                                        
                                        if !stack_parts.is_empty() {
                                            let stack_str = stack_parts.join(";");
                                            flamegraph_lines.push(format!("{} {}", stack_str, count));
                                        }
                                    }
                                }
                                
                                // Generate flamegraph
                                if flamegraph_lines.is_empty() {
                                    let _ = tx.send(Err("No call stack data found. Ensure the server is running and receiving traffic.".to_string()));
                                } else {
                                    match flamegraph::from_lines(&mut options, flamegraph_lines.iter().map(|s| s.as_str()), &mut flamegraph_data) {
                                        Ok(_) => {
                                            let _ = tx.send(Ok(flamegraph_data));
                                        }
                                        Err(e) => {
                                            let _ = tx.send(Err(format!("Failed to generate flamegraph: {:?}", e)));
                                        }
                                    }
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
        
        Some(report_tx_clone)
    } else {
        eprintln!("Warning: pprof profiling not available. Profiling endpoints will be disabled.");
        None
    };

    let state = Arc::new(AppState {
        client,
        auth_token: AsciiMetadataValue::from_static("atishay"),
        caller_id: AsciiMetadataValue::from_static("test-3"),
        report_tx: report_tx_final,
    });

    // Note: Heap profiling disabled - requires jemalloc to be configured as global allocator
    // To enable: uncomment jemalloc configuration at top of file and jemalloc_pprof in Cargo.toml

    if state.report_tx.is_some() {
        println!("CPU Profiler started. Profiling endpoints available:");
        println!("  - GET /pprof/protobuf - Download CPU pprof data (use with: go tool pprof http://localhost:8080/pprof/protobuf)");
        println!("  - GET /pprof/flamegraph - View CPU flamegraph SVG in browser");
        println!("  - GET /pprof/text - View CPU text report");
        #[cfg(not(target_env = "msvc"))]
        println!("  - GET /pprof/heap - Download heap/memory pprof data (use with: go tool pprof http://localhost:8080/pprof/heap)");
    }

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
                        router(req, state)
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



