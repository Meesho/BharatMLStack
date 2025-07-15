use rust_sdk::{BharatMLClient, retrieve::{Query, FeatureGroup, Keys}};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("Hello, feature-store-rust!");

    let mut client = BharatMLClient::new("127.0.0.1", 8081).await?;

    let request = Query {
        entity_label: "some_entity".to_string(),
        feature_groups: vec![
            FeatureGroup {
                label: "fg_1".to_string(),
                feature_labels: vec!["f1".to_string(), "f2".to_string()],
            },
        ],
        keys_schema: vec!["key1".to_string()],
        keys: vec![
            Keys {
                cols: vec!["k1".to_string()],
            },
        ],
    };

    let response = client.retrieve_features(request).await?;

    println!("RESPONSE={:?}", response);

    Ok(())
} 