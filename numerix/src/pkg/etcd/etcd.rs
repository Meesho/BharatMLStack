use crate::pkg::config::config::get_config;
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::Mutex;
use std::sync::RwLock;
use std::time::Duration;
use std::sync::OnceLock;
use etcd_client::EventType;
use etcd_client::{Client, ConnectOptions, WatchOptions};
use std::error::Error as StdError;
use serde_json::Value;
use crate::logger;
use regex::Regex;
use std::str::FromStr;
use std::collections::HashSet;

static ETCD_CLIENT: OnceLock<Arc<Mutex<Client>>> = OnceLock::new();
static ETCD_APP_PATH: OnceLock<String> = OnceLock::new();
static ETCD_TIMEOUT: OnceLock<Duration> = OnceLock::new();
static BASE_PATH: &str = "/config/";
static EMPTY_RESPONSE: &str = "";


pub async fn init_etcd_connection() {
    match ETCD_TIMEOUT.set(Duration::from_secs(5)) {
        Ok(_) => (),
        Err(e) => {
            logger::fatal(format!("Failed to initialize ETCD timeout to 5 seconds: {:?}", e), None)
        }
    };

    let config = get_config();
    let etcd_servers_str = config.etcd_servers.clone();
    let etcd_servers = etcd_servers_str.split(",").collect::<Vec<&str>>();
    let etcd_servers = etcd_servers.iter().map(|s| s.to_string()).collect::<Vec<String>>();
    let etcd_path = format!("{}{}", BASE_PATH, config.app_name);

    let timeout = *ETCD_TIMEOUT.get().unwrap();

    let username = match config.etcd_username.clone() {
        Some(username) => username,
        None => EMPTY_RESPONSE.to_string(),
    };
    let password = match config.etcd_password.clone() {
        Some(password) => password,
        None => EMPTY_RESPONSE.to_string(),
    };

    let opts = ConnectOptions::new()
        .with_connect_timeout(timeout)
        .with_keep_alive(timeout, timeout)
        .with_keep_alive_while_idle(true);

    let opts = if !username.is_empty() && !password.is_empty() {
        opts.with_user(username, password)
    } else {
        opts
    };

    let client = Client::connect(&etcd_servers, Some(opts)).await;
    let client = match client {
        Ok(client) => client,
        Err(e) => {
            logger::fatal(format!("Failed to connect to ETCD servers {:?}", etcd_servers), Some(&e));
        }
    };

    if let Err(e) = ETCD_APP_PATH.set(etcd_path.clone()) {
        logger::error(format!("ETCD_APP_PATH already initialized when setting to '{}': {:?}", etcd_path, e), None);
    }

    if let Err(_) = ETCD_CLIENT.set(Arc::new(Mutex::new(client))) {
        logger::error("ETCD_CLIENT already initialized during connection setup", None);
    }
}

pub async fn get_child_nodes(etcd_path: &str) -> Result<(HashMap<String, String>, HashMap<String, Vec<String>>), Box<dyn StdError>> {
    
    let etcd_absolute_path = format!("{}{}/", ETCD_APP_PATH.get().unwrap(), etcd_path);
    let client_arc = ETCD_CLIENT.get().unwrap();

    let mut client =  client_arc.lock().await;

    let resp = client.get(etcd_absolute_path.clone(), Some(etcd_client::GetOptions::new().with_prefix())).await;
    let resp = match resp {
        Ok(resp) => resp,
        Err(e) => {
            logger::error(format!("Failed to get child nodes from ETCD path: {}", etcd_absolute_path), Some(&e));
            return Err(Box::new(e));
        }
    };
    let mut expressions = HashMap::new();
    let mut expression_meta_map = HashMap::new();

    for kv in resp.kvs() {
        let key = kv.key_str()?;
        if let Some(node_name) = key.strip_prefix(&etcd_absolute_path) {
            if let Ok(json) = serde_json::from_slice::<Value>(kv.value()) {
                if let Some(expression) = json.get("expression").and_then(|e| e.as_str()) {
                    expressions.insert(node_name.to_string(), expression.to_string());
                    let meta_data = meta_data_from_expression::<f64>(expression);
                    expression_meta_map.insert(node_name.to_string(), meta_data);
                }
            }
        }
    }
    Ok((expressions, expression_meta_map))
}

fn meta_data_from_expression<T>(expression: &str) -> Vec<String>
where
    T: FromStr + Copy + Default,
    <T as FromStr>::Err: std::fmt::Debug,
{
    let re = match Regex::new(r"-?\d+(\.\d+)?") {
        Ok(re) => re,
        Err(_e) => {
            logger::fatal(format!("Failed to create regex pattern for expression {:?} : {:?}", expression, _e), None);
        }
    };
    let mut seen = HashSet::new();
    let mut unique_numbers = Vec::new();

    for cap in re.find_iter(expression) {
        let number_str = cap.as_str().to_string();
        if seen.insert(number_str.clone()) {
            unique_numbers.push(number_str);
        }
    }

    unique_numbers
}

pub async fn watch_etcd_path(etcd_path: &str, expression_map: Arc<RwLock<HashMap<String, String>>>, expression_meta_map: Arc<RwLock<HashMap<String, Vec<String>>>>) {

    let etcd_absolute_path = format!("{}{}/", ETCD_APP_PATH.get().unwrap(), etcd_path);
    let client_arc = ETCD_CLIENT.get().unwrap().clone();

    tokio::spawn(async move {
        loop {
            let mut client = client_arc.lock().await;
            let options = WatchOptions::new()
            .with_prefix()
            .with_progress_notify();
            let watch_result = client.watch(etcd_absolute_path.clone(), Some(options)).await;

            let mut stream = match watch_result {
                Ok((_, stream)) => stream,
                Err(e) => {
                    logger::error(format!("Failed to start ETCD watch on path: {}", etcd_absolute_path), Some(&e));
                    tokio::time::sleep(Duration::from_secs(2)).await;
                    continue;
                }
            };

            loop {
                match stream.message().await {
                    Ok(Some(event)) => {
                        logger::info(format!("Event: {:?}", event));
                        for ev in event.events() {
                            match ev.event_type() {
                                EventType::Put => {
                                    if let Some(kv) = ev.kv() {
                                        if let Ok(json) = serde_json::from_slice::<Value>(kv.value()) {
                                            if let Some(expression) = json.get("expression").and_then(|e| e.as_str()) {
                                                let key_str = match kv.key_str() {
                                                    Ok(k) => k.to_string(),
                                                    Err(e) => {
                                                        logger::error(format!("Failed to parse ETCD key string during PUT event on path: {}", etcd_absolute_path), Some(&e));
                                                        continue;
                                                    }
                                                };

                                                let key_str = key_str.split("/").last().unwrap().to_string();
                                                if let Ok(mut map) = expression_map.write() {
                                                    map.insert(key_str.clone(), expression.to_string());
                                                }
                                                if let Ok(mut map) = expression_meta_map.write() {
                                                    map.insert(key_str, meta_data_from_expression::<f64>(expression));
                                                }
                                            }
                                        }
                                    }
                                }
                                EventType::Delete => {
                                    if let Some(kv) = ev.kv() {
                                        let key_str = match kv.key_str() {
                                            Ok(k) => k.to_string(),
                                            Err(e) => {
                                                logger::error(format!("Failed to parse ETCD key string during DELETE event on path: {}", etcd_absolute_path), Some(&e));
                                                continue;
                                            }
                                        };

                                        let key_str = key_str.split("/").last().unwrap().to_string();
                                        if let Ok(mut map) = expression_map.write() {
                                            map.remove(&key_str);
                                        }
                                        if let Ok(mut map) = expression_meta_map.write() {
                                            map.remove(&key_str);
                                        }
                                    }
                                }
                                
                            }
                        }
                    }
                    Ok(None) => {
                        logger::error(format!("ETCD watch stream ended unexpectedly for path: {}", etcd_absolute_path), None);
                        break;
                    }
                    Err(e) => {
                        logger::error(format!("ETCD watch stream error on path: {}", etcd_absolute_path), Some(&e));
                        break;
                    }
                }
            }
            logger::info(format!("Re-establishing ETCD watch after error on path: {}", etcd_absolute_path));
            tokio::time::sleep(Duration::from_secs(2)).await;
        }
    });
}
