mod config;
mod storage;

use config::Config;
use storage::StorageManager;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let config = Config::from_env().expect("Failed to load config");
    println!("Loaded config: {:?}", config);

    // Initialize storage manager and ensure bucket exists
    let storage_manager = StorageManager::new(config.storage.clone());
    
    println!("Checking if storage bucket exists...");
    storage_manager.ensure_bucket_exists().await?;
    
    println!("Model registry is ready!");
    
    Ok(())
}
