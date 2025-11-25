use crate::pkg::config::config;
use std::cell::RefCell;
use std::error::Error;
use std::sync::OnceLock;
use std::time::{SystemTime, UNIX_EPOCH};
use tracing::{Level, Subscriber};
use tracing_subscriber::{
    fmt::{format::Writer, FormatEvent, FormatFields},
    registry::LookupSpan,
    EnvFilter,
};

static APP_NAME: OnceLock<String> = OnceLock::new();
static SAMPLING_RATE: OnceLock<f64> = OnceLock::new();
thread_local! {
    static RNG: RefCell<fastrand::Rng> = RefCell::new(fastrand::Rng::new());
}

#[inline]
fn should_log() -> bool {
    let sampling_rate = *SAMPLING_RATE.get().unwrap_or(&1.0);
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

struct CustomFormat;

impl<S, N> FormatEvent<S, N> for CustomFormat
where
    S: Subscriber + for<'a> LookupSpan<'a>,
    N: for<'a> FormatFields<'a> + 'static,
{
    fn format_event(
        &self,
        ctx: &tracing_subscriber::fmt::FmtContext<'_, S, N>,
        mut writer: Writer<'_>,
        event: &tracing::Event<'_>,
    ) -> std::fmt::Result {
        let timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default();
        let secs = timestamp.as_secs();
        let millis = timestamp.subsec_millis();
        let time_str = format!("{}.{:03}", secs, millis);

        let app_name = match APP_NAME.get() {
            Some(name) => name.as_str(),
            None => "UNKNOWN_APP",
        };

        let level = event.metadata().level();
        let level_str = match *level {
            Level::TRACE => "TRACE",
            Level::DEBUG => "DEBUG",
            Level::INFO => "INFO",
            Level::WARN => "WARN",
            Level::ERROR => "ERROR",
        };

        write!(
            writer,
            "{} {} [, ] [] {} numerix ",
            app_name, time_str, level_str
        )?;

        ctx.field_format().format_fields(writer.by_ref(), event)?;

        writeln!(writer)
    }
}

pub fn init_logger() {
    let config = config::get_config();

    match APP_NAME.set(config.app_name.clone()) {
        Ok(_) => (),
        Err(_) => error(
            format!(
                "Logger already initialized for app '{}', cannot reinitialize",
                config.app_name
            ),
            None,
        ),
    }
    println!("SAMPLING_RATE: {}", config.log_sampling_rate);
    // Cache the sampling rate for performance
    match SAMPLING_RATE.set(config.log_sampling_rate) {
        Ok(_) => (),
        Err(_) => error(
            "Logger sampling rate already initialized, cannot reinitialize",
            None,
        ),
    }

    let log_level = config.app_log_level.clone().to_ascii_uppercase();

    let filter_directive = match log_level.as_str() {
        "DEBUG" => "debug", 
        "INFO" => "info",
        "WARN" => "warn",
        "ERROR" => "error",
        "FATAL" | "PANIC" => "error",
        "DISABLED" => "off",
        _ => fatal(format!("Invalid log level '{}' for app '{}', expected: DEBUG/INFO/WARN/ERROR/FATAL/DISABLED", log_level, config.app_name), None),
    };

    let env_filter = EnvFilter::new(filter_directive);

    tracing_subscriber::fmt::Subscriber::builder()
        .with_env_filter(env_filter)
        .event_format(CustomFormat)
        .with_writer(std::io::stdout)
        .init();
}

pub fn debug(message: impl AsRef<str>) {
    tracing::debug!("{}", message.as_ref());
}

pub fn info(message: impl AsRef<str>) {
    tracing::info!("{}", message.as_ref());
}

pub fn warn(message: impl AsRef<str>) {
    tracing::warn!("{}", message.as_ref());
}

pub fn error(message: impl AsRef<str>, error: Option<&dyn Error>) {
    if should_log() {
        match error {
            Some(err) => tracing::error!("{}: {}", message.as_ref(), err),
            None => tracing::error!("{}", message.as_ref()),
        }
    }
}

pub fn fatal(message: impl AsRef<str>, err: Option<&dyn Error>) -> ! {
    // Fatal messages bypass sampling and always get logged
    let full_message = match err {
        Some(e) => format!("FATAL: {}: {}", message.as_ref(), e),
        None => format!("FATAL: {}", message.as_ref()),
    };

    // Log directly to stdout with custom format since it's fatal
    let timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default();
    let secs = timestamp.as_secs();
    let millis = timestamp.subsec_millis();
    let time_str = format!("{}.{:03}", secs, millis);

    let app_name = match APP_NAME.get() {
        Some(name) => name.as_str(),
        None => "UNKNOWN_APP",
    };

    println!(
        "{} {} [, ] [] FATAL numerix {}",
        app_name, time_str, full_message
    );
    panic!("{}", full_message);
}
