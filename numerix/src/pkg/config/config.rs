use crate::logger;
use config::{Config, Environment};
use dotenv;
use once_cell::sync::Lazy;
use serde::Deserialize;
use std::path::Path;

#[derive(Debug, Deserialize)]
pub struct AppConfig {
    pub telegraf_udp_host: String,
    pub telegraf_udp_port: u32,
    pub application_port: u32,
    pub metric_sampling_rate: f64,
    pub log_sampling_rate: f64,
    pub app_env: String,
    pub app_name: String,
    pub etcd_servers: String,
    pub etcd_username: Option<String>,
    pub etcd_password: Option<String>,
    pub app_log_level: String,
    pub channel_buffer_size: u32,
}

static CONFIG: Lazy<AppConfig> = Lazy::new(init_config);

static TELEGRAF_UDP_HOST: &str = "telegraf_udp_host";
static TELEGRAF_UDP_PORT: &str = "telegraf_udp_port";
static LOG_SAMPLING_RATE: &str = "log_sampling_rate";
static TELEGRAF_UDP_HOST_VALUE: &str = "localhost";
static TELEGRAF_UDP_PORT_VALUE: u32 = 8125;
static LOG_SAMPLING_RATE_VALUE: f64 = 1.0;

fn init_config() -> AppConfig {
    dotenv::from_path(Path::new(".env")).ok();

    let mut builder = Config::builder();

    // Add default values - if these fail, it's a critical error
    match builder.set_default(TELEGRAF_UDP_HOST, TELEGRAF_UDP_HOST_VALUE) {
        Ok(b) => builder = b,
        Err(e) => logger::fatal(
            "Failed to set default value for telegraf_udp_host to 'localhost'",
            Some(&e),
        ),
    }

    match builder.set_default(TELEGRAF_UDP_PORT, TELEGRAF_UDP_PORT_VALUE) {
        Ok(b) => builder = b,
        Err(e) => logger::fatal(
            "Failed to set default value for telegraf_udp_port to 8125",
            Some(&e),
        ),
    }

    match builder.set_default(LOG_SAMPLING_RATE, LOG_SAMPLING_RATE_VALUE) {
        Ok(b) => builder = b,
        Err(e) => logger::fatal(
            "Failed to set default value for log_sampling_rate to 1.0",
            Some(&e),
        ),
    }

    #[cfg(test)]
    {
        // Provide sane defaults in tests to avoid deserialization panics
        builder = builder
            .set_default("application_port", 0u32)
            .and_then(|b| b.set_default("telegraf_udp_host", "127.0.0.1"))
            .and_then(|b| b.set_default("telegraf_udp_port", 8125u32))
            .and_then(|b| b.set_default("metric_sampling_rate", 1.0f64))
            .and_then(|b| b.set_default("log_sampling_rate", 1.0f64))
            .and_then(|b| b.set_default("app_env", "test"))
            .and_then(|b| b.set_default("app_name", "numerix-test"))
            .and_then(|b| b.set_default("etcd_servers", "127.0.0.1:2379"))
            .and_then(|b| b.set_default("app_log_level", "ERROR"))
            .and_then(|b| b.set_default("channel_buffer_size", 1024u32))
            .unwrap_or_else(|e| {
                logger::fatal("Failed to set test defaults in config builder", Some(&e))
            });
    }

    let settings = match builder.add_source(Environment::default()).build() {
        Ok(cfg) => cfg,
        Err(e) => logger::fatal(
            "Failed to build configuration from environment variables",
            Some(&e),
        ),
    };

    match settings.try_deserialize() {
        Ok(cfg) => cfg,
        Err(e) => logger::fatal(
            "Failed to deserialize configuration into AppConfig struct",
            Some(&e),
        ),
    }
}

pub fn get_config() -> &'static AppConfig {
    &CONFIG
}
