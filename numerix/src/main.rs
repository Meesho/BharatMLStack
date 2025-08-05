pub mod pkg;
pub mod server;
use handler::config as handler_config;
use pkg::config::config;
use pkg::etcd::etcd;
use pkg::metrics::metrics;
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
