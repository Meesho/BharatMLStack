mod config;
mod storage;

use config::Config;
use storage::StorageManager;
use tracing::{info, warn, debug};
use tracing_subscriber::{EnvFilter, fmt, prelude::*};

fn get_log_level_from_env() -> String {
    // Try to get log level from environment variables first
    // This allows us to set up logging before loading the full config
    std::env::var("LOGGER__LEVEL")
        .or_else(|_| std::env::var("LOG_LEVEL"))
        .unwrap_or_else(|_| "info".to_string())
}

fn init_logger(level: &str) -> anyhow::Result<()> {
    // Parse the log level from config
    let filter = match level.to_lowercase().as_str() {
        "trace" => EnvFilter::new("trace"),
        "debug" => EnvFilter::new("debug"), 
        "info" => EnvFilter::new("info"),
        "warn" => EnvFilter::new("warn"),
        "error" => EnvFilter::new("error"),
        _ => {
            panic!("Invalid log level '{}'. Valid levels are: trace, debug, info, warn, error", level);
        }
    };

    tracing_subscriber::registry()
        .with(fmt::layer().with_target(false))
        .with(filter)
        .init();
    
    Ok(())
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Get log level from environment variables and initialize logger
    let log_level = get_log_level_from_env();
    init_logger(&log_level)?;
    
    info!("Starting model registry application");
    debug!("Using log level: {}", log_level);
    
    let config = Config::from_env().expect("Failed to load config");
    
    // Validate that the config log level matches what we're using
    if config.logger.level.to_lowercase() != log_level.to_lowercase() {
        warn!("Config log level '{}' differs from environment log level '{}'. Using environment level.", 
              config.logger.level, log_level);
    }
    
    debug!("Loaded config: {:?}", config);

    // Initialize storage manager and ensure bucket exists
    let storage_manager = StorageManager::new(config.storage.clone());
    
    info!("Checking if storage bucket exists...");
    storage_manager.ensure_bucket_exists().await?;
    
    info!("Model registry is ready!");
    
    Ok(())
}
