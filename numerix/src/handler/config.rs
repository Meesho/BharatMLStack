use crate::pkg::logger::log;
use crate::pkg::etcd::client as etcd;
use std::collections::HashMap;
use std::sync::Arc;
use std::sync::OnceLock;
use std::sync::RwLock;
static EXPRESSION_MAP: OnceLock<Arc<RwLock<HashMap<String, String>>>> = OnceLock::new();
static EXPRESSION_CONFIG_ETCD_PATH: OnceLock<String> = OnceLock::new();
type ExpressionMetaMap = OnceLock<Arc<RwLock<HashMap<String, Vec<String>>>>>;
static EXPRESSION_META_MAP: ExpressionMetaMap = OnceLock::new();

pub async fn init_config() {
    if EXPRESSION_CONFIG_ETCD_PATH.set("/expression-config".to_string()).is_err() {
        log::fatal(
            "Failed to set EXPRESSION_CONFIG_ETCD_PATH, it was already initialized",
            None,
        )
    }

    let etcd_path = match EXPRESSION_CONFIG_ETCD_PATH.get() {
        Some(path) => path,
        None => log::fatal(
            "EXPRESSION_CONFIG_ETCD_PATH was not set and could not be initialized",
            None,
        ),
    };

    let (expression_map, expression_meta_map) = match etcd::get_child_nodes(etcd_path).await {
        Ok((expression, expression_meta)) => (expression, expression_meta),
        Err(_) => {
            log::fatal("Application cannot start without expression config", None);
        }
    };

    let expression_map_arc = Arc::new(RwLock::new(expression_map));
    if EXPRESSION_MAP.set(expression_map_arc.clone()).is_err() {
        log::error("EXPRESSION_MAP was already initialized", None);
    }

    let expression_meta_map_arc = Arc::new(RwLock::new(expression_meta_map));
    if EXPRESSION_META_MAP
        .set(expression_meta_map_arc.clone())
        .is_err()
    {
        log::error("EXPRESSION_META_MAP was already initialized", None);
    }

    let expression_map_for_watch = Arc::clone(EXPRESSION_MAP.get().unwrap());
    let expression_meta_map_for_watch = Arc::clone(EXPRESSION_META_MAP.get().unwrap());

    etcd::watch_etcd_path(
        etcd_path,
        expression_map_for_watch,
        expression_meta_map_for_watch,
    )
    .await;
}

pub fn get_exression(key: &str) -> String {
    let expression_map = EXPRESSION_MAP.get();
    let expression_map = match expression_map {
        Some(expression_map) => expression_map,
        None => {
            log::error(
                format!(
                    "Expression map not initialized when looking up compute_id: {}",
                    key
                ),
                None,
            );
            return "".to_string();
        }
    };
    let expression_map = expression_map.read();
    let expression_map = match expression_map {
        Ok(expression_map) => expression_map,
        Err(e) => {
            log::error(
                format!("Failed to read expression map lock for compute_id: {}", key),
                Some(&e),
            );
            return "".to_string();
        }
    };

    match expression_map.get(key) {
        Some(expression) => expression.clone(),
        None => {
            log::error(
                format!("Expression not found for compute_id: {}", key),
                None,
            );
            "".to_string()
        }
    }
}

pub fn get_meta_data(compute_id: &str) -> Vec<String> {
    let expression_meta_map = EXPRESSION_META_MAP.get();
    let expression_meta_map = match expression_meta_map {
        Some(expression_meta_map) => expression_meta_map,
        None => {
            log::error(
                format!(
                    "Expression meta map not initialized when looking up compute_id: {}",
                    compute_id
                ),
                None,
            );
            return Vec::new();
        }
    };
    let expression_meta_map = expression_meta_map.read();
    let expression_meta_map = match expression_meta_map {
        Ok(expression_meta_map) => expression_meta_map,
        Err(e) => {
            log::error(
                format!(
                    "Failed to read expression meta map lock for compute_id: {}",
                    compute_id
                ),
                Some(&e),
            );
            return Vec::new();
        }
    };
    match expression_meta_map.get(compute_id) {
        Some(meta_data) => meta_data.clone(),
        None => {
            log::error(
                format!("Meta data not found for compute_id: {}", compute_id),
                None,
            );
            Vec::new()
        }
    }
}
