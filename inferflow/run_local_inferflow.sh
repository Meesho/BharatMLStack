#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
QUICK_START_DIR="${REPO_ROOT}/quick-start"
INFERFLOW_DIR="${REPO_ROOT}/inferflow"
KAFKA_PROVIDER="${KAFKA_PROVIDER:-brew}" # brew | docker | none

ensure_java17() {
  local java_major
  java_major="$(java -version 2>&1 | awk -F[\".] '/version/ {print $2}')"
  if [[ "${java_major}" == "1" ]]; then
    java_major="$(java -version 2>&1 | awk -F[\".] '/version/ {print $3}')"
  fi

  if [[ -z "${java_major}" || "${java_major}" -lt 17 ]]; then
    if command -v brew >/dev/null 2>&1 && brew list --versions openjdk@17 >/dev/null 2>&1; then
      export JAVA_HOME="$(brew --prefix openjdk@17)/libexec/openjdk.jdk/Contents/Home"
      export PATH="$(brew --prefix openjdk@17)/bin:${PATH}"
      java_major="$(java -version 2>&1 | awk -F[\".] '/version/ {print $2}')"
      if [[ "${java_major}" == "1" ]]; then
        java_major="$(java -version 2>&1 | awk -F[\".] '/version/ {print $3}')"
      fi
    fi
  fi

  if [[ -z "${java_major}" || "${java_major}" -lt 17 ]]; then
    echo "Kafka needs Java 17+."
    echo "Current Java: $(java -version 2>&1 | head -n 1)"
    echo "Install/fix with:"
    echo "  brew install openjdk@17"
    echo "  export JAVA_HOME=\$(brew --prefix openjdk@17)/libexec/openjdk.jdk/Contents/Home"
    echo "  export PATH=\$(brew --prefix openjdk@17)/bin:\$PATH"
    exit 1
  fi
}

start_kafka() {
  case "${KAFKA_PROVIDER}" in
    brew)
      ensure_java17
      if ! command -v brew >/dev/null 2>&1; then
        echo "Homebrew not found. Install brew or set KAFKA_PROVIDER=docker/none."
        exit 1
      fi
      if ! brew list --versions kafka >/dev/null 2>&1; then
        echo "Kafka is not installed via brew. Install it first: brew install kafka"
        exit 1
      fi
      echo "Starting Kafka via Homebrew..."
      brew services start kafka
      sleep 2
      if command -v kafka-topics >/dev/null 2>&1; then
        kafka-topics --bootstrap-server localhost:9092 \
          --create --if-not-exists \
          --topic inferflow_inference_logs \
          --partitions 1 \
          --replication-factor 1 || true
      fi
      ;;
    docker)
      if ! command -v docker >/dev/null 2>&1; then
        echo "Docker not found. Install Docker or set KAFKA_PROVIDER=brew/none."
        exit 1
      fi
      echo "Starting Kafka via Docker compose..."
      docker compose -f "${QUICK_START_DIR}/docker-compose.yml" up -d kafka kafka-init
      ;;
    none)
      echo "Skipping Kafka startup (KAFKA_PROVIDER=none)."
      ;;
    *)
      echo "Invalid KAFKA_PROVIDER=${KAFKA_PROVIDER}. Use brew, docker, or none."
      exit 1
      ;;
  esac
}

start_kafka

echo "Running inferflow..."
cd "${INFERFLOW_DIR}"

APP_ENV="${APP_ENV:-prod}" \
APP_NAME="${APP_NAME:-model-proxy-service-bytes}" \
APP_PORT="${APP_PORT:-8080}" \
APP_LOG_LEVEL="${APP_LOG_LEVEL:-DEBUG}" \
APP_GC_PERCENTAGE="${APP_GC_PERCENTAGE:-100}" \
IN_MEMORY_CACHE_SIZE_IN_BYTES="${IN_MEMORY_CACHE_SIZE_IN_BYTES:-6000000000}" \
DAG_TOPOLOGY_CACHE_TTL_SEC="${DAG_TOPOLOGY_CACHE_TTL_SEC:-300}" \
DAG_TOPOLOGY_CACHE_SIZE="${DAG_TOPOLOGY_CACHE_SIZE:-500}" \
METRIC_SAMPLING_RATE="${METRIC_SAMPLING_RATE:-0.1}" \
TELEGRAF_HOST="${TELEGRAF_HOST:-localhost}" \
TELEGRAF_PORT="${TELEGRAF_PORT:-8125}" \
ETCD_WATCHER_ENABLED="${ETCD_WATCHER_ENABLED:-true}" \
ETCD_SERVER="${ETCD_SERVER:-mlp-etcd-1.prd.meesho.int:2379,mlp-etcd-2.prd.meesho.int:2379,mlp-etcd-3.prd.meesho.int:2379}" \
EXTERNAL_SERVICE_ONFS_FS_HOST="${EXTERNAL_SERVICE_ONFS_FS_HOST:-online-feature-store-api.prd.meesho.int}" \
EXTERNAL_SERVICE_ONFS_FS_PORT="${EXTERNAL_SERVICE_ONFS_FS_PORT:-8080}" \
EXTERNAL_SERVICE_ONFS_FS_GRPC_PLAIN_TEXT="${EXTERNAL_SERVICE_ONFS_FS_GRPC_PLAIN_TEXT:-true}" \
EXTERNAL_SERVICE_ONFS_FS_CALLER_ID="${EXTERNAL_SERVICE_ONFS_FS_CALLER_ID:-model-proxy-service-experiment}" \
EXTERNAL_SERVICE_ONFS_FS_CALLER_TOKEN="${EXTERNAL_SERVICE_ONFS_FS_CALLER_TOKEN:-}" \
EXTERNAL_SERVICE_ONFS_FS_DEAD_LINE="${EXTERNAL_SERVICE_ONFS_FS_DEAD_LINE:-200}" \
EXTERNAL_SERVICE_ONFS_FS_BATCH_SIZE="${EXTERNAL_SERVICE_ONFS_FS_BATCH_SIZE:-50}" \
NUMERIX_CLIENT_V1_HOST="${NUMERIX_CLIENT_V1_HOST:-iris.prd.meesho.int}" \
NUMERIX_CLIENT_V1_PORT="${NUMERIX_CLIENT_V1_PORT:-8080}" \
NUMERIX_CLIENT_V1_DEADLINE_MS="${NUMERIX_CLIENT_V1_DEADLINE_MS:-50}" \
NUMERIX_CLIENT_V1_PLAINTEXT="${NUMERIX_CLIENT_V1_PLAINTEXT:-true}" \
NUMERIX_CLIENT_V1_AUTHTOKEN="${NUMERIX_CLIENT_V1_AUTHTOKEN:-${IRIS_CLIENT_V1_AUTH_TOKEN:-iris}}" \
NUMERIX_CLIENT_V1_BATCHSIZE="${NUMERIX_CLIENT_V1_BATCHSIZE:-800}" \
EXTERNAL_SERVICE_PREDATOR_PORT="${EXTERNAL_SERVICE_PREDATOR_PORT:-8080}" \
EXTERNAL_SERVICE_PREDATOR_GRPC_PLAIN_TEXT="${EXTERNAL_SERVICE_PREDATOR_GRPC_PLAIN_TEXT:-true}" \
EXTERNAL_SERVICE_PREDATOR_CALLER_ID="${EXTERNAL_SERVICE_PREDATOR_CALLER_ID:-model-proxy-service-bytes}" \
EXTERNAL_SERVICE_PREDATOR_CALLER_TOKEN="${EXTERNAL_SERVICE_PREDATOR_CALLER_TOKEN:-model-proxy-service-bytes}" \
EXTERNAL_SERVICE_PREDATOR_DEADLINE="${EXTERNAL_SERVICE_PREDATOR_DEADLINE:-100}" \
KAFKA_BOOTSTRAP_SERVERS="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}" \
KAFKA_V2_LOG_TOPIC="${KAFKA_V2_LOG_TOPIC:-inferflow_inference_logs}" \
go run ./cmd/inferflow/main.go
