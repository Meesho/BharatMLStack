use crate::logger;
use crate::pkg::config::config::get_config;
use cadence::{BufferedUdpMetricSink, MetricClient, QueuingMetricSink, StatsdClient};
use std::net::UdpSocket;
use std::sync::Arc;
use std::sync::OnceLock;
use std::time::Duration;

static STATSD_CLIENT: OnceLock<Arc<dyn MetricClient + Send + Sync>> = OnceLock::new();

use std::cell::RefCell;

static METRIC_SAMPLING_RATE: OnceLock<f64> = OnceLock::new();
thread_local! {
    static RNG: RefCell<fastrand::Rng> = RefCell::new(fastrand::Rng::new());
}

#[inline]
fn should_send_metric() -> bool {
    let sampling_rate = *METRIC_SAMPLING_RATE.get().unwrap_or(&1.0);
    if sampling_rate >= 1.0 {
        return true;
    }

    if sampling_rate <= 0.0 {
        return false;
    }

    RNG.with(|rng_cell| {
        let mut rng = rng_cell.borrow_mut();
        rng.f64() < sampling_rate
    })
}

pub fn init_config() {
    let config = get_config();
    let res = METRIC_SAMPLING_RATE.set(config.metric_sampling_rate);
    if res.is_err() {
        logger::fatal("Failed to set metric sampling rate", None);
    }
    let host = (
        config.telegraf_udp_host.clone(),
        config.telegraf_udp_port as u16,
    );
    let app_env = &config.app_env;
    let app_name = &config.app_name;
    let socket = UdpSocket::bind("0.0.0.0:0");
    let socket = match socket {
        Ok(socket) => socket,
        Err(e) => logger::fatal(
            format!(
                "Failed to bind UDP socket for metrics client to telegraf {}:{}",
                config.telegraf_udp_host, config.telegraf_udp_port
            ),
            Some(&e),
        ),
    };
    let sink = BufferedUdpMetricSink::from(host, socket);
    let sink = match sink {
        Ok(sink) => sink,
        Err(e) => logger::fatal(
            format!(
                "Failed to create UDP metrics sink for telegraf {}:{}",
                config.telegraf_udp_host, config.telegraf_udp_port
            ),
            Some(&e),
        ),
    };
    let queuing_sink = QueuingMetricSink::from(sink);

    let _ = STATSD_CLIENT.set(Arc::new(
        StatsdClient::builder("numerix", queuing_sink)
            .with_tag("env", app_env)
            .with_tag("service", app_name)
            .build(),
    ));

    logger::info(format!(
        "Metrics client initialized with telegraf address = {}, global tags = {:?}, and sampling rate = {}",
        format!("{}:{}", config.telegraf_udp_host, config.telegraf_udp_port),
        vec![("env", app_env), ("service", app_name)],
        config.metric_sampling_rate));
}

pub fn timing(name: &str, value: Duration, tags: &[(&str, &str)]) {
    let rate = METRIC_SAMPLING_RATE.get().unwrap();
    if !should_send_metric() {
        return;
    }
    let client = STATSD_CLIENT.get().unwrap();

    let mut metric = client.time_with_tags(name, value * 1000);
    for tag in tags {
        metric = metric.with_tag(tag.0, tag.1);
    }
    metric.with_sampling_rate(*rate).send();
}

pub fn count<'a>(name: &'a str, value: u64, tags: &'a [(&'a str, &'a str)]) {
    let rate = METRIC_SAMPLING_RATE.get().unwrap();
    if !should_send_metric() {
        return;
    }
    let client = STATSD_CLIENT.get().unwrap();

    let mut metric = client.count_with_tags(name, value);
    for tag in tags {
        metric = metric.with_tag(tag.0, tag.1);
    }
    metric.with_sampling_rate(*rate).send();
}

pub fn gauge(name: &str, value: f64, tags: &[(&str, &str)]) {
    let rate = METRIC_SAMPLING_RATE.get().unwrap();
    if !should_send_metric() {
        return;
    }
    let client = STATSD_CLIENT.get().unwrap();

    let mut metric = client.gauge_with_tags(name, value);
    for tag in tags {
        metric = metric.with_tag(tag.0, tag.1);
    }
    metric.with_sampling_rate(*rate).send();
}
