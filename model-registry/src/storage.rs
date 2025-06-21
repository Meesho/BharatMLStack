use crate::config::StorageConfig;
use anyhow::{anyhow, Result};
use aws_sdk_s3::Client as S3Client;
use url::Url;
use tracing::info;

pub struct StorageManager {
    config: StorageConfig,
}

impl StorageManager {
    pub fn new(config: StorageConfig) -> Self {
        Self { config }
    }

    pub async fn ensure_bucket_exists(&self) -> Result<()> {
        let bucket_url = Url::parse(&self.config.bucket_url)?;
        
        match bucket_url.scheme() {
            "s3" => self.ensure_s3_bucket_exists().await,
            "gs" => self.ensure_gcs_bucket_exists().await,
            _ => Err(anyhow!("Unsupported storage scheme: {}", bucket_url.scheme())),
        }
    }

    async fn ensure_s3_bucket_exists(&self) -> Result<()> {
        let bucket_url = Url::parse(&self.config.bucket_url)?;
        let bucket_name = bucket_url.host_str()
            .ok_or_else(|| anyhow!("Invalid S3 bucket URL"))?;

        // Configure AWS SDK
        let mut config_builder = aws_config::defaults(aws_config::BehaviorVersion::latest());
        
        // Add region if specified
        if let Some(region) = &self.config.region {
            config_builder = config_builder.region(aws_config::Region::new(region.clone()));
        }

        // Use explicit credentials if provided
        if let (Some(access_key), Some(secret_key)) = (&self.config.access_key, &self.config.secret_key) {
            use aws_sdk_s3::config::Credentials;
            let credentials = Credentials::new(
                access_key,
                secret_key,
                None, // No session token
                None, // No expiration
                "explicit_config"
            );
            config_builder = config_builder.credentials_provider(credentials);
            info!("Using explicit S3 credentials");
        } else {
            info!("Using default S3 credential chain (IAM roles, environment variables, etc.)");
        }

        let aws_config = config_builder.load().await;
        let s3_client = S3Client::new(&aws_config);

        // Check if bucket exists
        match s3_client.head_bucket().bucket(bucket_name).send().await {
            Ok(_) => {
                info!("S3 bucket '{}' already exists", bucket_name);
                Ok(())
            }
            Err(_) => {
                // Bucket doesn't exist, create it
                info!("Creating S3 bucket '{}'", bucket_name);
                
                let mut create_bucket_request = s3_client.create_bucket().bucket(bucket_name);
                
                // Add region configuration for bucket creation if specified
                if let Some(region) = &self.config.region {
                    if region != "us-east-1" {
                        use aws_sdk_s3::types::{CreateBucketConfiguration, BucketLocationConstraint};
                        let location_constraint = BucketLocationConstraint::from(region.as_str());
                        let bucket_config = CreateBucketConfiguration::builder()
                            .location_constraint(location_constraint)
                            .build();
                        create_bucket_request = create_bucket_request.create_bucket_configuration(bucket_config);
                    }
                }

                create_bucket_request.send().await
                    .map_err(|e| anyhow!("Failed to create S3 bucket: {}", e))?;
                
                info!("Successfully created S3 bucket '{}'", bucket_name);
                Ok(())
            }
        }
    }

    async fn ensure_gcs_bucket_exists(&self) -> Result<()> {
        let bucket_url = Url::parse(&self.config.bucket_url)?;
        let bucket_name = bucket_url.host_str()
            .ok_or_else(|| anyhow!("Invalid GCS bucket URL"))?;

        let _project_id = self.config.project_id.as_ref()
            .ok_or_else(|| anyhow!("GCS project_id is required"))?;

        // Create GCS client with explicit credentials if provided
        let client = if let Some(credentials_path) = &self.config.credentials_path {
            // Temporarily set the environment variable for this client
            // This is the standard way the cloud-storage crate reads credentials
            let original_var = std::env::var("GOOGLE_APPLICATION_CREDENTIALS").ok();
            
            unsafe {
                std::env::set_var("GOOGLE_APPLICATION_CREDENTIALS", credentials_path);
            }
            
            info!("Using explicit GCS credentials from: {}", credentials_path);
            let client = cloud_storage::Client::default();
            
            // Restore original environment variable if it existed
            unsafe {
                match original_var {
                    Some(original) => std::env::set_var("GOOGLE_APPLICATION_CREDENTIALS", original),
                    None => std::env::remove_var("GOOGLE_APPLICATION_CREDENTIALS"),
                }
            }
            
            client
        } else {
            info!("Using default GCS credential chain (application default credentials, environment variables, etc.)");
            cloud_storage::Client::default()
        };

        // Check if bucket exists
        match client.bucket().read(bucket_name).await {
            Ok(_) => {
                info!("GCS bucket '{}' already exists", bucket_name);
                Ok(())
            }
            Err(_) => {
                // Bucket doesn't exist, create it
                info!("Creating GCS bucket '{}'", bucket_name);
                
                let new_bucket = cloud_storage::NewBucket {
                    name: bucket_name.to_string(),
                    ..Default::default()
                };

                match client.bucket().create(&new_bucket).await {
                    Ok(_) => {
                        info!("Successfully created GCS bucket '{}'", bucket_name);
                        Ok(())
                    }
                    Err(e) => Err(anyhow!("Failed to create GCS bucket: {}", e))
                }
            }
        }
    }
} 