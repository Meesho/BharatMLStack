use crate::logger;
#[cfg(not(test))]
use crate::pkg::etcd::etcd;
use std::collections::HashMap;
use std::sync::Arc;
use std::sync::OnceLock;
use std::sync::RwLock;

type ExpressionMap = Arc<RwLock<HashMap<String, String>>>;
type ExpressionMetaMap = Arc<RwLock<HashMap<String, Vec<String>>>>;

static EXPRESSION_MAP: OnceLock<ExpressionMap> = OnceLock::new();
static EXPRESSION_CONFIG_ETCD_PATH: OnceLock<String> = OnceLock::new();
static EXPRESSION_META_MAP: OnceLock<ExpressionMetaMap> = OnceLock::new();

pub async fn init_config() {
    if EXPRESSION_CONFIG_ETCD_PATH
        .set("/expression-config".to_string())
        .is_err()
    {
        logger::error("EXPRESSION_CONFIG_ETCD_PATH was already initialized", None);
    }

    #[allow(unused_variables)]
    let etcd_path = match EXPRESSION_CONFIG_ETCD_PATH.get() {
        Some(path) => path,
        None => logger::fatal(
            "EXPRESSION_CONFIG_ETCD_PATH was not set and could not be initialized",
            None,
        ),
    };

    #[cfg(test)]
    let (expression_map, expression_meta_map) = {
        // Seed test expressions and metadata
        let mut expressions = HashMap::new();
        expressions.insert("1".to_string(), "a b +".to_string());
        let mut meta = HashMap::new();
        meta.insert("1".to_string(), Vec::new());
        (expressions, meta)
    };

    #[cfg(not(test))]
    let (expression_map, expression_meta_map) = match etcd::get_child_nodes(etcd_path).await {
        Ok((expression, expression_meta)) => (expression, expression_meta),
        Err(_) => {
            logger::fatal("Application cannot start without expression config", None);
        }
    };

    let expression_map_arc = Arc::new(RwLock::new(expression_map));
    if EXPRESSION_MAP.set(expression_map_arc.clone()).is_err() {
        logger::error("EXPRESSION_MAP was already initialized", None);
    }

    let expression_meta_map_arc = Arc::new(RwLock::new(expression_meta_map));
    if EXPRESSION_META_MAP
        .set(expression_meta_map_arc.clone())
        .is_err()
    {
        logger::error("EXPRESSION_META_MAP was already initialized", None);
    }

    #[cfg(not(test))]
    {
        let expression_map_for_watch = Arc::clone(EXPRESSION_MAP.get().unwrap());
        let expression_meta_map_for_watch = Arc::clone(EXPRESSION_META_MAP.get().unwrap());

        etcd::watch_etcd_path(
            etcd_path,
            expression_map_for_watch,
            expression_meta_map_for_watch,
        )
        .await;
    }
}

pub fn get_exression(key: &str) -> String {
    let expression_map = EXPRESSION_MAP.get();
    let expression_map = match expression_map {
        Some(expression_map) => expression_map,
        None => {
            logger::error(
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
            logger::error(
                format!("Failed to read expression map lock for compute_id: {}", key),
                Some(&e),
            );
            return "".to_string();
        }
    };

    match expression_map.get(key) {
        Some(expression) => expression.clone(),
        None => {
            logger::error(
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
            logger::error(
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
            logger::error(
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
            logger::error(
                format!("Meta data not found for compute_id: {}", compute_id),
                None,
            );
            Vec::new()
        }
    }
}
