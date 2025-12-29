use hyper::server::conn::http1;
use hyper::service::service_fn;
use hyper::{body::Incoming as IncomingBody, Request, Response, StatusCode};
use hyper_util::rt::TokioIo;
use http_body_util::{BodyExt, Full};
use hyper::body::Bytes;
use serde::Deserialize;
use smallvec::SmallVec;
use serde::Deserialize;
use std::borrow::Cow;
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

#[derive(Deserialize)]
struct RetrieveFeaturesRequest<'a> {
    #[serde(rename = "entity_label")]
    entity_label: &'a str, 
    
    #[serde(borrow)]
    feature_groups: SmallVec<[FeatureGroupRequest<'a>; 8]>,
    
    #[serde(rename = "keys_schema")]
    keys_schema: Vec<&'a str>,
    
    #[serde(borrow)]
    keys: Vec<KeysRequest<'a>>,
}

#[derive(Deserialize)]
struct FeatureGroupRequest<'a> {
    label: &'a str,
    #[serde(rename = "feature_labels")]
    feature_labels: Vec<&'a str>,
}

#[derive(Deserialize)]
struct KeysRequest<'a> {
    cols: Vec<&'a str>,
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
    
    let body_bytes: Bytes = req.into_body().collect().await?.to_bytes();

    let request_body: RetrieveFeaturesRequest = serde_json::from_slice(&body_bytes)?;

    let grpc_request = Query {
        entity_label: request_body.entity_label.clone(), // Zero CPU copy
        feature_groups: request_body.feature_groups.into_iter().map(|fg| {
            FeatureGroup {
                label: fg.label.clone(),
                feature_labels: fg.feature_labels.clone(),
            }
        }).collect(),
        keys_schema: request_body.keys_schema.clone(),
        keys: request_body.keys.into_iter().map(|k| Keys { cols: k.cols }).collect(),
    };


    let metadata = grpc_request.metadata_mut();
    metadata.insert("online-feature-store-auth-token", state.auth_token.clone());
    metadata.insert("online-feature-store-caller-id", state.caller_id.clone());

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

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let channel = Channel::from_static("http://online-feature-store-api.int.meesho.int:80")
        .http2_keep_alive_interval(Duration::from_secs(30))
        .keep_alive_timeout(Duration::from_secs(10))
        .keep_alive_while_idle(true)
        .connect()
        .await?;

    let state = Arc::new(AppState {
        client: FeatureServiceClient::new(channel),
        auth_token: AsciiMetadataValue::from_static("atishay"),
        caller_id: AsciiMetadataValue::from_static("test-3"),
    });

    let listener = TcpListener::bind("0.0.0.0:8080").await?;

    loop {
        let (stream, _) = listener.accept().await?;
        let io = TokioIo::new(stream);
        let state_clone = state.clone();

        tokio::task::spawn(async move {
            let service = service_fn({
                let state = state_clone.clone();
                move |req| {
                    let state = state.clone();
                    handler(req, state)
                }
            });

            let mut builder = http1::Builder::new();
            builder.keep_alive(true);
            let _ = builder.serve_connection(io, service).await;
        });
    }
}
