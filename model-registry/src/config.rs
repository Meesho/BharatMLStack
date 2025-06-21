use config::ConfigError;
use serde::Deserialize;

#[derive(Debug, Deserialize, Clone)]
pub struct Config {
    pub storage: StorageConfig,
    pub server: ServerConfig,
    pub logger: LoggerConfig,
}

#[derive(Debug, Deserialize, Clone)]
pub struct StorageConfig {
    pub bucket_url: String,

    //For S3
    pub region: Option<String>,
    pub access_key: Option<String>,
    pub secret_key: Option<String>,

    //For GCS
    pub project_id: Option<String>,
    pub credentials_path: Option<String>,
}

#[derive(Debug, Deserialize, Clone)]
pub struct ServerConfig {
    pub port: u16,
    pub host: String,
}

#[derive(Debug, Deserialize, Clone)]
pub struct LoggerConfig {
    pub level: String,
}

impl Config {
    pub fn from_env() -> Result<Self, ConfigError> {
        // Check APP_ENV to determine if we should load from .env file
        let app_env = std::env::var("APP_ENV").unwrap_or_else(|_| "development".to_string());
        
        // Load from .env file if not in production
        if app_env != "prod" && app_env != "production" {
            dotenvy::dotenv().ok();
        }

        let cfg = config::Config::builder()
            .add_source(config::Environment::default().separator("__"))
            .build()?;

        cfg.try_deserialize()
    }
}


#[cfg(test)]
mod tests {
    use super::*;
    use std::env;

    #[test]
    fn test_config_from_env() {
        // Setup environment variables
        unsafe {
            env::set_var("STORAGE__BUCKET_URL", "s3://my-models-bucket");
            env::set_var("STORAGE__REGION", "us-west-2");
            env::set_var("STORAGE__ACCESS_KEY", "test-access-key");
            env::set_var("STORAGE__SECRET_KEY", "test-secret-key");

            env::set_var("SERVER__PORT", "8080");
            env::set_var("SERVER__HOST", "127.0.0.1");

            env::set_var("LOGGER__LEVEL", "debug");
        }

        // Load config
        let config = Config::from_env().expect("Failed to load config");

        // Assertions
        assert_eq!(config.storage.bucket_url, "s3://my-models-bucket");
        assert_eq!(config.storage.region.as_deref(), Some("us-west-2"));
        assert_eq!(config.server.port, 8080);
        assert_eq!(config.logger.level, "debug");
    }
}
