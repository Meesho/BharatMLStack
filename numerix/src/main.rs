pub mod handler;
pub mod pkg;
pub mod server;
use handler::config as handler_config;
use pkg::config::app_config;
use pkg::etcd::client as etcd_client;
use pkg::logger::log;
use pkg::metrics::client as metrics_client;

#[tokio::main]
async fn main() {
    log::init_logger();
    app_config::get_config();
    metrics_client::init_config();
    etcd_client::init_etcd_connection().await;
    handler_config::init_config().await;
    server::init_server().await;
}
