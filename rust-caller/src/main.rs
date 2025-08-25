use axum::{
    http::StatusCode,
    response::Json,
    routing::post,
    Router,
};
use rust_sdk::retrieve::{
    feature_service_client::FeatureServiceClient as RetrieveClient, FeatureGroup, Keys, Query,
};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::str::FromStr;
use tonic::metadata::MetadataValue;
use tonic::transport::Channel;
use tower::ServiceBuilder;
use tower_http::cors::CorsLayer;

#[derive(Serialize, Deserialize)]
struct ApiResponse {
    success: bool,
    data: Option<String>,
    error: Option<String>,
    message: String,
}

async fn retrieve_features() -> Result<Json<ApiResponse>, StatusCode> {
    match retrieve_features_internal().await {
        Ok(result) => Ok(Json(ApiResponse {
            success: true,
            data: Some(format!("{:?}", result)),
            error: None,
            message: "Features retrieved successfully".to_string(),
        })),
        Err(e) => Ok(Json(ApiResponse {
            success: false,
            data: None,
            error: Some(e.to_string()),
            message: "Failed to retrieve features".to_string(),
        })),
    }
}

async fn retrieve_features_internal() -> Result<rust_sdk::retrieve::Result, Box<dyn std::error::Error>> {
    println!("Attempting to connect to the feature store...");

    let mut metadata = HashMap::new();
    metadata.insert(
        "online-feature-store-auth-token",
        "test".to_string(),
    );
    metadata.insert(
        "online-feature-store-caller-id",
        "model-proxy-service-experiment".to_string(),
    );

    let channel =
        Channel::from_static("http://online-feature-store-api-mp.prd.meesho.int:80")
            .connect()
            .await?;

    let service = ServiceBuilder::new()
        .layer(tonic::service::interceptor(
            move |mut req: tonic::Request<()>| {
                for (key, value) in &metadata {
                    req.metadata_mut()
                        .insert(*key, MetadataValue::from_str(value).unwrap());
                }
                Ok(req)
            },
        ))
        .service(channel);

    let mut client = RetrieveClient::new(service);

    println!("Connection successful. Retrieving features...");

    let request = tonic::Request::new(Query {
        entity_label: "catalog".to_string(),
        feature_groups: vec![
            FeatureGroup {
                label: "derived_2_fp32".to_string(),
                feature_labels: vec!["sbid_value".to_string()],
            },
            FeatureGroup {
                label: "derived_fp16".to_string(),
                feature_labels: vec![
                    "search__organic_clicks_by_views_3_days_percentile".to_string(),
                    "search__organic_clicks_by_views_5_days_percentile".to_string(),
                ],
            },
        ],
        keys_schema: vec!["catalog_id".to_string()],
        keys: vec![
            Keys {
                cols: vec!["176".to_string()],
            },
            Keys {
                cols: vec!["179".to_string()],
            },
        ],
        metadata: HashMap::new(),
    });

    let response = client.retrieve_features(request).await?;
    Ok(response.into_inner())
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Create the application with routes
    let app = Router::new()
        .route("/retrieve-features", post(retrieve_features))
        .layer(CorsLayer::permissive());

    // Run the server
    println!("Starting Rust Feature Store API server on http://0.0.0.0:8080");
    let listener = tokio::net::TcpListener::bind("0.0.0.0:8080").await?;
    axum::serve(listener, app).await?;

    Ok(())
}
