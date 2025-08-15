use rust_sdk::retrieve::{
    feature_service_client::FeatureServiceClient as RetrieveClient, FeatureGroup, Keys, Query,
};
use std::collections::HashMap;
use std::str::FromStr;
use tonic::metadata::MetadataValue;
use tonic::transport::Channel;
use tower::ServiceBuilder;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
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

    let response = match client.retrieve_features(request).await {
        Ok(response) => response,
        Err(e) => {
            eprintln!("Failed to retrieve features: {}", e);
            return Err(e.into());
        }
    };

    println!(
        "Successfully retrieved features: {:?}",
        response.into_inner()
    );

    Ok(())
}
