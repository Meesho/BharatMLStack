pub mod server;
pub mod pkg;
use pkg::config::config;
use pkg::metrics::metrics;
use pkg::etcd::etcd;
use handler::config as handler_config;
pub mod handler;
use pkg::logger::logger;

#[tokio::main]
async fn main() {
    logger::init_logger();
    config::get_config();
    metrics::init_config();
    etcd::init_etcd_connection().await;
    handler_config::init_config().await;
    server::init_server().await;
    
}