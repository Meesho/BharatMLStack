#!/bin/bash

set -e

GO_MIN_VERSION="1.22"
INSTALL_LINK="https://go.dev/doc/install"
WORKSPACE_DIR="workspace"

# Infrastructure services (always started)
INFRASTRUCTURE_SERVICES="scylla mysql redis etcd kafka"
INFRASTRUCTURE_INIT_SERVICES="kafka-init db-init"

# Application services (user selectable)
ONFS_SERVICES="onfs-api-server onfs-healthcheck"
ONFS_CONSUMER_SERVICES="onfs-consumer onfs-consumer-healthcheck"
HORIZON_SERVICES="horizon horizon-healthcheck"
NUMERIX_SERVICES="numerix numerix-healthcheck"
TRUFFLEBOX_SERVICES="trufflebox-ui trufflebox-healthcheck"
INFERFLOW_SERVICES="inferflow inferflow-healthcheck"
SKYE_SERVICES="skye-trigger skye-admin skye-admin-healthcheck skye-consumers skye-consumers-healthcheck skye-serving skye-serving-healthcheck"
PREDATOR_SERVICES="predator predator-healthcheck"

# Management tools
MANAGEMENT_SERVICES="etcd-workbench kafka-ui"

# Single gRPC UI stack: nginx proxy (routes by service path) + one grpcui browser client
GRPCUI_SERVICES="grpc-proxy grpcui"

# Capture version variables from environment (default to latest if not set)
ONFS_VERSION="${ONFS_VERSION:-latest}"
ONFS_CONSUMER_VERSION="${ONFS_CONSUMER_VERSION:-latest}"
HORIZON_VERSION="${HORIZON_VERSION:-latest}"
NUMERIX_VERSION="${NUMERIX_VERSION:-latest}"
TRUFFLEBOX_VERSION="${TRUFFLEBOX_VERSION:-latest}"
INFERFLOW_VERSION="${INFERFLOW_VERSION:-latest}"
SKYE_VERSION="${SKYE_VERSION:-latest}"

# Global variables for user selection
SELECTED_SERVICES="$INFRASTRUCTURE_SERVICES $MANAGEMENT_SERVICES"
START_ONFS=false
START_ONFS_CONSUMER=false
START_HORIZON=false
START_NUMERIX=false
START_TRUFFLEBOX=false
START_INFERFLOW=false
START_SKYE=false
START_PREDATOR=false
INIT_DUMMY_DATA=false
ENABLE_LOCAL_BUILD=false
START_GRPCUI=false

check_go_version() {
  if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed."
    echo "👉 Please install Go $GO_MIN_VERSION+ from: $INSTALL_LINK"
    exit 1
  fi

  GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
  if [ "$(printf '%s\n' "$GO_MIN_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$GO_MIN_VERSION" ]; then
    echo "❌ Go version $GO_VERSION is less than required $GO_MIN_VERSION"
    echo "👉 Please install Go $GO_MIN_VERSION+ from: $INSTALL_LINK"
    exit 1
  fi

  echo "✅ Go version $GO_VERSION detected"
}

check_python3() {
  if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 is not installed."
    echo "👉 Python 3 is required for local build support"
    echo "👉 Please install Python 3 from: https://www.python.org/downloads/"
    exit 1
  fi
}

check_docker() {
  echo ""
  echo "🐳 Checking Docker..."

  local os_type
  os_type="$(uname -s)"

  # ── 1. Install Docker if the binary is not present ──────────────────────────
  if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed."
    case "$os_type" in
      Darwin)
        echo "🍎 macOS detected."
        if command -v brew &> /dev/null; then
          echo "   Installing Docker Desktop via Homebrew..."
          brew install --cask docker
        else
          echo "❌ Homebrew is not installed."
          echo "👉 Option 1: Install Homebrew first:"
          echo "   /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
          echo "👉 Option 2: Download Docker Desktop manually:"
          echo "   https://docs.docker.com/desktop/install/mac-install/"
          exit 1
        fi
        ;;
      Linux)
        echo "🐧 Linux detected."
        if command -v curl &> /dev/null; then
          echo "   Installing Docker Engine via official install script..."
          curl -fsSL https://get.docker.com | sh
          # Add current user to docker group so future commands don't need sudo
          sudo usermod -aG docker "$USER" 2>/dev/null || true
          echo "✅ Docker installed."
          echo "⚠️  NOTE: Log out and back in for the docker group to take effect."
        elif command -v apt-get &> /dev/null; then
          sudo apt-get update -y && sudo apt-get install -y docker.io
        elif command -v dnf &> /dev/null; then
          sudo dnf install -y docker
        elif command -v yum &> /dev/null; then
          sudo yum install -y docker
        else
          echo "❌ Could not detect a supported package manager."
          echo "👉 Please install Docker manually: https://docs.docker.com/engine/install/"
          exit 1
        fi
        ;;
      *)
        echo "❌ Unsupported OS: $os_type"
        echo "👉 Please install Docker from: https://docs.docker.com/engine/install/"
        exit 1
        ;;
    esac
  else
    echo "✅ Docker installed: $(docker --version 2>/dev/null | head -1)"
  fi

  # ── 2. Start the Docker daemon if it is not running ─────────────────────────
  if ! docker info &> /dev/null 2>&1; then
    echo "⚠️  Docker daemon is not running. Starting it now..."
    case "$os_type" in
      Darwin)
        if [ -d "/Applications/Docker.app" ]; then
          echo "   Launching Docker Desktop from /Applications..."
          open -a Docker
        elif [ -d "$HOME/Applications/Docker.app" ]; then
          echo "   Launching Docker Desktop from ~/Applications..."
          open "$HOME/Applications/Docker.app"
        else
          echo "❌ Docker Desktop application not found."
          echo "👉 Please install Docker Desktop: https://docs.docker.com/desktop/install/mac-install/"
          exit 1
        fi
        echo "⏳ Waiting for Docker daemon to be ready (up to 90 seconds)..."
        for i in {1..45}; do
          if docker info &> /dev/null 2>&1; then
            echo "✅ Docker daemon is running!"
            break
          fi
          if [ "$i" -eq 45 ]; then
            echo "❌ Docker daemon did not start within 90 seconds."
            echo "👉 Please open Docker Desktop manually, wait for it to fully start, then re-run this script."
            exit 1
          fi
          sleep 2
        done
        ;;
      Linux)
        echo "   Starting Docker daemon via systemctl / service..."
        if command -v systemctl &> /dev/null; then
          sudo systemctl start docker
        elif command -v service &> /dev/null; then
          sudo service docker start
        else
          echo "❌ Could not start Docker daemon automatically."
          echo "👉 Try: sudo systemctl start docker"
          exit 1
        fi
        echo "⏳ Waiting for Docker daemon to be ready..."
        for i in {1..15}; do
          if docker info &> /dev/null 2>&1; then
            echo "✅ Docker daemon is running!"
            break
          fi
          if [ "$i" -eq 15 ]; then
            echo "❌ Docker daemon did not start. Check logs: sudo journalctl -u docker"
            exit 1
          fi
          sleep 2
        done
        ;;
      *)
        echo "❌ Cannot automatically start the Docker daemon on $os_type."
        echo "👉 Please start Docker manually and re-run this script."
        exit 1
        ;;
    esac
  else
    echo "✅ Docker daemon is running"
  fi

  # ── 3. Ensure docker-compose is available (handles V1 binary vs V2 plugin) ──
  if ! command -v docker-compose &> /dev/null; then
    if docker compose version &> /dev/null 2>&1; then
      echo "   ℹ️  'docker-compose' (V1) not found; using 'docker compose' (V2 plugin) as a transparent wrapper"
      # Define a function so the rest of this script can call 'docker-compose' transparently
      docker-compose() { docker compose "$@"; }
      export -f docker-compose
    else
      echo "❌ Neither 'docker-compose' nor the 'docker compose' plugin is available."
      echo "👉 Install Docker Compose: https://docs.docker.com/compose/install/"
      exit 1
    fi
  else
    echo "✅ docker-compose available: $(docker-compose --version 2>/dev/null | head -1)"
  fi
}

setup_workspace() {
  echo "📁 Setting up workspace in ./$WORKSPACE_DIR"
  rm -rf "$WORKSPACE_DIR"
  mkdir -p "$WORKSPACE_DIR"
  
  # Copy docker-compose.yml
  cp ./docker-compose.yml "$WORKSPACE_DIR"/ 
  
  # Copy db-init directory (remove existing first to ensure fresh copy)
  if [ -d "$WORKSPACE_DIR/db-init" ]; then
    rm -rf "$WORKSPACE_DIR/db-init"
  fi
  cp -r ./db-init "$WORKSPACE_DIR"/
  
  # Copy predator-dummy directory for Docker build
  if [ -d "$WORKSPACE_DIR/predator-dummy" ]; then
    rm -rf "$WORKSPACE_DIR/predator-dummy"
  fi
  cp -r ./predator-dummy "$WORKSPACE_DIR"/

  # Copy skye-trigger directory for OSS Airflow replacement (Docker build)
  if [ -d "$WORKSPACE_DIR/skye-trigger" ]; then
    rm -rf "$WORKSPACE_DIR/skye-trigger"
  fi
  cp -r ./skye-trigger "$WORKSPACE_DIR"/

  # Copy horizon configs directory for service config loading
  local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  local project_root="$(cd "$script_dir/.." && pwd)"
  if [ -d "$project_root/horizon/configs" ]; then
    if [ -d "$WORKSPACE_DIR/configs" ]; then
      rm -rf "$WORKSPACE_DIR/configs"
    fi
    cp -r "$project_root/horizon/configs" "$WORKSPACE_DIR"/
    echo "   ✅ Copied horizon configs to workspace"
  else
    echo "   ⚠️  Warning: horizon/configs directory not found at $project_root/horizon/configs"
  fi
  # Copy skye-config for Skye services
  if [ -d "$script_dir/skye-config" ]; then
    if [ -d "$WORKSPACE_DIR/skye-config" ]; then
      rm -rf "$WORKSPACE_DIR/skye-config"
    fi
    cp -r "$script_dir/skye-config" "$WORKSPACE_DIR"/
    echo "   ✅ Copied skye-config to workspace"
  fi

  # Copy proto files for gRPC UI containers
  local protos_dir="$WORKSPACE_DIR/protos"
  mkdir -p "$protos_dir"

  if [ -d "$project_root/online-feature-store/pkg/proto" ]; then
    mkdir -p "$protos_dir/onfs"
    cp "$project_root/online-feature-store/pkg/proto"/*.proto "$protos_dir/onfs/"
    echo "   ✅ Copied ONFS proto files"
  fi

  if [ -d "$project_root/numerix/src/protos/proto" ]; then
    mkdir -p "$protos_dir/numerix"
    cp "$project_root/numerix/src/protos/proto"/*.proto "$protos_dir/numerix/"
    echo "   ✅ Copied Numerix proto files"
  fi

  if [ -d "$project_root/inferflow/server/proto" ]; then
    mkdir -p "$protos_dir/inferflow"
    cp "$project_root/inferflow/server/proto"/*.proto "$protos_dir/inferflow/"
    echo "   ✅ Copied Inferflow proto files"
  fi

  if [ -d "$project_root/go-sdk/pkg/clients/skye/client/proto" ]; then
    mkdir -p "$protos_dir/skye"
    cp "$project_root/go-sdk/pkg/clients/skye/client/proto"/*.proto "$protos_dir/skye/"
    echo "   ✅ Copied Skye proto files"
  fi

  if [ -d "$project_root/go-sdk/pkg/clients/predator/client/proto" ]; then
    mkdir -p "$protos_dir/predator"
    cp "$project_root/go-sdk/pkg/clients/predator/client/proto"/*.proto "$protos_dir/predator/"
    echo "   ✅ Copied Predator proto files"
  fi

  # Create nginx gRPC proxy config
  # Uses Docker's embedded DNS resolver (127.0.0.11) with set $var trick so nginx
  # starts even when individual gRPC backends are not running yet.
  mkdir -p "$WORKSPACE_DIR/nginx"
  cat > "$WORKSPACE_DIR/nginx/grpc-proxy.conf" << 'NGINX_CONF'
user  nginx;
worker_processes  auto;
error_log  /var/log/nginx/error.log notice;
pid        /var/run/nginx.pid;

events {
    worker_connections 1024;
}

http {
    log_format grpc_log
        '$remote_addr [$time_local] "$uri" $status '
        'upstream=$upstream_addr rt=$request_time';
    access_log /var/log/nginx/access.log grpc_log;

    # Docker embedded DNS — resolves container names at request time,
    # so nginx does not fail to start when a backend is not yet running.
    resolver 127.0.0.11 valid=30s ipv6=off;

    server {
        # http2 directive (nginx 1.25.1+); compatible with older nginx too
        listen 9000;
        http2  on;
        server_name _;

        grpc_read_timeout  300s;
        grpc_send_timeout  300s;

        # ---------- ONFS Feature Store (port 8089) ----------
        location /retrieve.FeatureService/ {
            set $backend onfs-api-server:8089;
            grpc_pass grpc://$backend;
        }
        location /persist.FeatureService/ {
            set $backend onfs-api-server:8089;
            grpc_pass grpc://$backend;
        }

        # ---------- Numerix Matrix Operations (port 8083) ----------
        location /numerix.Numerix/ {
            set $backend numerix:8083;
            grpc_pass grpc://$backend;
        }

        # ---------- Inferflow Inference Gateway (port 8085) ----------
        location /Inferflow/ {
            set $backend inferflow:8085;
            grpc_pass grpc://$backend;
        }
        location /Predict/ {
            set $backend inferflow:8085;
            grpc_pass grpc://$backend;
        }

        # ---------- Skye Vector Search (port 8094) ----------
        location /SkyeSimilarCandidateService/ {
            set $backend skye-serving:8094;
            grpc_pass grpc://$backend;
        }
        location /SkyeEmbeddingService/ {
            set $backend skye-serving:8094;
            grpc_pass grpc://$backend;
        }

        # ---------- Predator Inference Server (port 8001) ----------
        location /inference.GRPCInferenceService/ {
            set $backend predator:8001;
            grpc_pass grpc://$backend;
        }
        location /grpc.health.v1.Health/ {
            set $backend predator:8001;
            grpc_pass grpc://$backend;
        }

        # Catch-all: return a gRPC UNIMPLEMENTED status
        location / {
            return 12 "No gRPC service matched this path";
        }
    }

    # ── Helper landing page (HTTP, port 80) ─────────────────────────────────
    # HTML is served from a mounted file (index.html) to avoid nginx
    # inline-string quoting issues with single quotes in CSS/JS.
    server {
        listen 80;
        server_name _;
        root /usr/share/nginx/grpcui-helper;
        index index.html;
        location / {
            try_files $uri /index.html;
        }
    }
}
NGINX_CONF

  # Copy the helper landing page alongside the nginx config
  if [ -f "$script_dir/nginx/index.html" ]; then
    cp "$script_dir/nginx/index.html" "$WORKSPACE_DIR/nginx/index.html"
    echo "   ✅ Created gRPC UI helper landing page"
  fi
  echo "   ✅ Created nginx gRPC proxy config"

  # Copy gRPC sample request bodies
  if [ -d "$script_dir/grpc-samples" ]; then
    if [ -d "$WORKSPACE_DIR/grpc-samples" ]; then
      rm -rf "$WORKSPACE_DIR/grpc-samples"
    fi
    cp -r "$script_dir/grpc-samples" "$WORKSPACE_DIR/"
    echo "   ✅ Copied gRPC sample requests"
  fi

  echo "✅ Workspace setup complete"
}

setup_local_builds() {
  echo "🔨 Setting up local builds..."
  
  local needs_local_build=false
  local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  local project_root="$(cd "$script_dir/.." && pwd)"
  
  # Check which services need local builds and copy their source directories
  if [[ "$START_ONFS" == true && "$ONFS_VERSION" == "local" ]]; then
    echo "   📦 Preparing ONFS API Server for local build..."
    if [ -d "$project_root/online-feature-store" ]; then
      if [ ! -d "$WORKSPACE_DIR/online-feature-store" ]; then
        cp -r "$project_root/online-feature-store" "$WORKSPACE_DIR"/
      fi
      needs_local_build=true
    else
      echo "   ⚠️  Warning: $project_root/online-feature-store not found, skipping local build for ONFS API Server"
    fi
  fi
  
  if [[ "$START_ONFS_CONSUMER" == true && "$ONFS_CONSUMER_VERSION" == "local" ]]; then
    echo "   📦 Preparing ONFS Consumer for local build..."
    if [ -d "$project_root/online-feature-store" ]; then
      # ONFS Consumer is in the same repo as API Server
      if [ ! -d "$WORKSPACE_DIR/online-feature-store" ]; then
        cp -r "$project_root/online-feature-store" "$WORKSPACE_DIR"/
      fi
      needs_local_build=true
    else
      echo "   ⚠️  Warning: $project_root/online-feature-store not found, skipping local build for ONFS Consumer"
    fi
  fi
  
  if [[ "$START_HORIZON" == true && "$HORIZON_VERSION" == "local" ]]; then
    echo "   📦 Preparing Horizon for local build..."
    if [ -d "$project_root/horizon" ]; then
      cp -r "$project_root/horizon" "$WORKSPACE_DIR"/
      needs_local_build=true
    else
      echo "   ⚠️  Warning: $project_root/horizon not found, skipping local build for Horizon"
    fi
  fi
  
  if [[ "$START_NUMERIX" == true && "$NUMERIX_VERSION" == "local" ]]; then
    echo "   📦 Preparing Numerix for local build..."
    if [ -d "$project_root/numerix" ]; then
      cp -r "$project_root/numerix" "$WORKSPACE_DIR"/
      needs_local_build=true
    else
      echo "   ⚠️  Warning: $project_root/numerix not found, skipping local build for Numerix"
    fi
  fi
  
  if [[ "$START_TRUFFLEBOX" == true && "$TRUFFLEBOX_VERSION" == "local" ]]; then
    echo "   📦 Preparing TruffleBox UI for local build..."
    if [ -d "$project_root/trufflebox-ui" ]; then
      cp -r "$project_root/trufflebox-ui" "$WORKSPACE_DIR"/
      needs_local_build=true
    else
      echo "   ⚠️  Warning: $project_root/trufflebox-ui not found, skipping local build for TruffleBox UI"
    fi
  fi
  
  if [[ "$START_INFERFLOW" == true && "$INFERFLOW_VERSION" == "local" ]]; then
    echo "   📦 Preparing Inferflow for local build..."
    if [ -d "$project_root/inferflow" ]; then
      cp -r "$project_root/inferflow" "$WORKSPACE_DIR"/
      needs_local_build=true
    else
      echo "   ⚠️  Warning: $project_root/inferflow not found, skipping local build for Inferflow"
    fi
  fi
  
  if [[ "$START_SKYE" == true && "$SKYE_VERSION" == "local" ]]; then
    echo "   📦 Preparing Skye for local build..."
    if [ -d "$project_root/skye" ]; then
      cp -r "$project_root/skye" "$WORKSPACE_DIR"/
      needs_local_build=true
    else
      echo "   ⚠️  Warning: $project_root/skye not found, skipping local build for Skye"
    fi
  fi
  
  if [[ "$needs_local_build" == true ]]; then
    echo "   🔧 Modifying docker-compose.yml for local builds..."
    # Get absolute path for compose file to avoid path issues
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local compose_file_abs="$(cd "$script_dir" && cd "$WORKSPACE_DIR" && pwd)/docker-compose.yml"
    # Export variables for Python script
    export COMPOSE_FILE="$compose_file_abs"
    export START_ONFS START_ONFS_CONSUMER START_HORIZON START_NUMERIX START_TRUFFLEBOX START_INFERFLOW START_SKYE
    export ONFS_VERSION ONFS_CONSUMER_VERSION HORIZON_VERSION NUMERIX_VERSION TRUFFLEBOX_VERSION INFERFLOW_VERSION SKYE_VERSION
    modify_docker_compose_for_local_builds
    echo "✅ Local build setup complete"
  else
    echo "✅ No local builds needed"
  fi
}

modify_docker_compose_for_local_builds() {
  # Check Python 3 is available
  if ! command -v python3 &> /dev/null; then
    echo "   ❌ Python 3 is required for local builds but not found"
    return 1
  fi
  
  # Use Python to modify YAML more reliably
  python3 << 'PYTHON_SCRIPT'
import sys
import re
import os

compose_file = os.environ.get('COMPOSE_FILE', '')
if not compose_file:
    sys.stderr.write("Error: COMPOSE_FILE environment variable not set\n")
    sys.exit(1)
start_onfs = os.environ.get('START_ONFS', 'false')
onfs_version = os.environ.get('ONFS_VERSION', '')
start_onfs_consumer = os.environ.get('START_ONFS_CONSUMER', 'false')
onfs_consumer_version = os.environ.get('ONFS_CONSUMER_VERSION', '')
start_horizon = os.environ.get('START_HORIZON', 'false')
horizon_version = os.environ.get('HORIZON_VERSION', '')
start_numerix = os.environ.get('START_NUMERIX', 'false')
numerix_version = os.environ.get('NUMERIX_VERSION', '')
start_trufflebox = os.environ.get('START_TRUFFLEBOX', 'false')
trufflebox_version = os.environ.get('TRUFFLEBOX_VERSION', '')
start_inferflow = os.environ.get('START_INFERFLOW', 'false')
inferflow_version = os.environ.get('INFERFLOW_VERSION', '')
start_skye = os.environ.get('START_SKYE', 'false')
skye_version = os.environ.get('SKYE_VERSION', '')

with open(compose_file, 'r') as f:
    content = f.read()

# ONFS API Server
if start_onfs == 'true' and onfs_version == 'local':
    pattern = r'(  onfs-api-server:\s*\n)\s+(image:.*\n)'
    replacement = r'\1    build:\n      context: ./online-feature-store\n      dockerfile: cmd/api-server/DockerFile\n    # \2'
    content = re.sub(pattern, replacement, content)

# ONFS Consumer
if start_onfs_consumer == 'true' and onfs_consumer_version == 'local':
    pattern = r'(  onfs-consumer:\s*\n)\s+(image:.*\n)'
    replacement = r'\1    build:\n      context: ./online-feature-store\n      dockerfile: cmd/consumer/DockerFile\n    # \2'
    content = re.sub(pattern, replacement, content)

# Horizon
if start_horizon == 'true' and horizon_version == 'local':
    pattern = r'(  horizon:\s*\n)\s+(image:.*\n)'
    replacement = r'\1    build:\n      context: ./horizon\n      dockerfile: cmd/horizon/Dockerfile\n    # \2'
    content = re.sub(pattern, replacement, content)

# Numerix
if start_numerix == 'true' and numerix_version == 'local':
    pattern = r'(  numerix:\s*\n)\s+(image:.*\n)'
    replacement = r'\1    build:\n      context: ./numerix\n      dockerfile: Dockerfile\n    # \2'
    content = re.sub(pattern, replacement, content)

# TruffleBox UI
if start_trufflebox == 'true' and trufflebox_version == 'local':
    pattern = r'(  trufflebox-ui:\s*\n)\s+(image:.*\n)'
    replacement = r'\1    build:\n      context: ./trufflebox-ui\n      dockerfile: DockerFile\n    # \2'
    content = re.sub(pattern, replacement, content)

# Inferflow
if start_inferflow == 'true' and inferflow_version == 'local':
    # Match inferflow service definition with image line (same pattern as other services)
    pattern = r'(  inferflow:\s*\n)\s+(image:.*\n)'
    replacement = r'\1    build:\n      context: ./inferflow\n      dockerfile: cmd/inferflow/Dockerfile\n    # \2'
    content = re.sub(pattern, replacement, content)

# Skye (admin, consumers, serving)
if start_skye == 'true' and skye_version == 'local':
    for svc, dockerfile in [('skye-admin', 'cmd/admin/Dockerfile'), ('skye-consumers', 'cmd/consumers/Dockerfile'), ('skye-serving', 'cmd/serving/Dockerfile')]:
        pattern = r'(  ' + re.escape(svc) + r':\s*\n)\s+(image:.*\n)'
        replacement = r'\1    build:\n      context: ./skye\n      dockerfile: ' + dockerfile + r'\n    # \2'
        content = re.sub(pattern, replacement, content)

with open(compose_file, 'w') as f:
    f.write(content)

# Verify changes were made
changes_made = False
if start_onfs == 'true' and onfs_version == 'local' and 'build:' in content and 'onfs-api-server' in content:
    changes_made = True
if start_onfs_consumer == 'true' and onfs_consumer_version == 'local' and 'build:' in content and 'onfs-consumer' in content:
    changes_made = True
if start_horizon == 'true' and horizon_version == 'local' and 'build:' in content and 'horizon:' in content:
    changes_made = True
if start_numerix == 'true' and numerix_version == 'local' and 'build:' in content and 'numerix:' in content:
    changes_made = True
if start_trufflebox == 'true' and trufflebox_version == 'local' and 'build:' in content and 'trufflebox-ui:' in content:
    changes_made = True
if start_inferflow == 'true' and inferflow_version == 'local' and 'build:' in content and 'inferflow:' in content:
    # Check if the replacement actually happened by looking for the build context
    if './inferflow' in content:
        changes_made = True
    else:
        sys.stderr.write("Warning: inferflow build context not found after replacement\n")
if start_skye == 'true' and skye_version == 'local' and './skye' in content:
    changes_made = True

if not changes_made and (start_onfs == 'true' or start_onfs_consumer == 'true' or start_horizon == 'true' or 
                         start_numerix == 'true' or start_trufflebox == 'true' or start_inferflow == 'true' or start_skye == 'true'):
    sys.stderr.write("Warning: Failed to modify docker-compose.yml for local builds\n")
    sys.exit(1)
PYTHON_SCRIPT
}

show_service_menu() {
  echo ""
  echo "🎯 BharatML Stack Service Selector"
  echo "=================================="
  echo ""
  echo "Infrastructure (ScyllaDB, MySQL, Redis, etcd, Kafka) and Management Tools (etcd-workbench, kafka-ui) will always be started."
  echo "Choose which application services to start:"
  echo ""
  echo "1) 🚀 All Services"
  echo "   • Online Feature Store + Consumer + Horizon + Numerix + TruffleBox UI + Inferflow + Skye + Predator"
  echo ""
  echo "2) 🎛️  Custom Selection"
  echo "   • Choose individual services"
  echo ""
  echo "0) ❌ Exit"
  echo ""
}

get_user_choice() {
  while true; do
    show_service_menu
    read -p "Enter your choice (0-2): " choice
    
    case $choice in
      1)
        echo "✅ Selected: All Services"
        SELECTED_SERVICES="$SELECTED_SERVICES $ONFS_SERVICES $ONFS_CONSUMER_SERVICES $HORIZON_SERVICES $NUMERIX_SERVICES $TRUFFLEBOX_SERVICES $INFERFLOW_SERVICES $SKYE_SERVICES $PREDATOR_SERVICES"
        START_ONFS=true
        START_ONFS_CONSUMER=true
        START_HORIZON=true
        START_NUMERIX=true
        START_TRUFFLEBOX=true
        START_INFERFLOW=true
        START_SKYE=true
        START_PREDATOR=true
        START_GRPCUI=true
        echo ""
        echo "🔧 Optional Infrastructure:"
        ask_dummy_data
        break
        ;;
      2)
        custom_selection
        break
        ;;
      0)
        echo "👋 Exiting..."
        exit 0
        ;;
      *)
        echo "❌ Invalid choice. Please enter 0-2."
        echo ""
        ;;
    esac
  done
}

custom_selection() {
  echo ""
  echo "🎛️  Custom Service Selection"
  echo "============================"
  echo ""
  echo "✅ Infrastructure services (always included): ScyllaDB, MySQL, Redis, etcd, Kafka, kafka-init"
  echo "✅ Management tools (always included): etcd-workbench, kafka-ui"
  echo ""
  
  # Ask about each service
  read -p "Include Online Feature Store API? [y/N]: " include_onfs
  if [[ $include_onfs =~ ^[Yy]$ ]]; then
    SELECTED_SERVICES="$SELECTED_SERVICES $ONFS_SERVICES"
    START_ONFS=true
    START_GRPCUI=true
    echo "✅ Added: Online Feature Store API"
  fi
  
  read -p "Include ONFS Consumer (Kafka ingestion)? [y/N]: " include_onfs_consumer
  if [[ $include_onfs_consumer =~ ^[Yy]$ ]]; then
    SELECTED_SERVICES="$SELECTED_SERVICES $ONFS_CONSUMER_SERVICES"
    START_ONFS_CONSUMER=true
    echo "✅ Added: ONFS Consumer"
  fi
  
  read -p "Include Horizon Backend? [y/N]: " include_horizon
  if [[ $include_horizon =~ ^[Yy]$ ]]; then
    SELECTED_SERVICES="$SELECTED_SERVICES $HORIZON_SERVICES"
    START_HORIZON=true
    echo "✅ Added: Horizon Backend"
  fi
  
  read -p "Include Numerix Matrix Operations? [y/N]: " include_numerix
  if [[ $include_numerix =~ ^[Yy]$ ]]; then
    SELECTED_SERVICES="$SELECTED_SERVICES $NUMERIX_SERVICES"
    START_NUMERIX=true
    START_GRPCUI=true
    echo "✅ Added: Numerix Matrix Operations"
  fi
  
  read -p "Include TruffleBox UI? [y/N]: " include_trufflebox
  if [[ $include_trufflebox =~ ^[Yy]$ ]]; then
    if [[ $START_HORIZON != true ]]; then
      echo "⚠️  TruffleBox UI requires Horizon Backend. Adding Horizon..."
      SELECTED_SERVICES="$SELECTED_SERVICES $HORIZON_SERVICES"
      START_HORIZON=true
    fi
    SELECTED_SERVICES="$SELECTED_SERVICES $TRUFFLEBOX_SERVICES"
    START_TRUFFLEBOX=true
    echo "✅ Added: TruffleBox UI"
  fi
  
  read -p "Include Inferflow? [y/N]: " include_inferflow
  if [[ $include_inferflow =~ ^[Yy]$ ]]; then
    SELECTED_SERVICES="$SELECTED_SERVICES $INFERFLOW_SERVICES"
    START_INFERFLOW=true
    START_GRPCUI=true
    echo "✅ Added: Inferflow"
  fi
  
  read -p "Include Skye (Vector Similarity Search - admin, consumers, serving)? [y/N]: " include_skye
  if [[ $include_skye =~ ^[Yy]$ ]]; then
    SELECTED_SERVICES="$SELECTED_SERVICES $SKYE_SERVICES"
    START_SKYE=true
    START_GRPCUI=true
    echo "✅ Added: Skye"
  fi
  
  read -p "Include Predator (Dummy gRPC Inference Server)? [y/N]: " include_predator
  if [[ $include_predator =~ ^[Yy]$ ]]; then
    SELECTED_SERVICES="$SELECTED_SERVICES $PREDATOR_SERVICES"
    START_PREDATOR=true
    START_GRPCUI=true
    echo "✅ Added: Predator"
  fi
  
  
  echo ""
  if [[ $START_ONFS == false && $START_ONFS_CONSUMER == false && $START_HORIZON == false && $START_NUMERIX == false && $START_TRUFFLEBOX == false && $START_INFERFLOW == false && $START_SKYE == false && $START_PREDATOR == false ]]; then
    echo "🎯 Custom selection complete: Only infrastructure services will be started"
  else
    echo "🎯 Custom selection complete!"
  fi
  
  ask_dummy_data
}

ask_dummy_data() {
  echo ""
  echo "📦 Dummy Data Initialization"
  echo "============================"
  echo ""
  echo "Would you like to initialize databases with dummy data?"
  echo "This will populate MySQL, ScyllaDB, and etcd with example entities,"
  echo "features, and configurations for testing purposes."
  echo ""
  read -p "Initialize dummy data? [y/N]: " init_dummy
  if [[ $init_dummy =~ ^[Yy]$ ]]; then
    INIT_DUMMY_DATA=true
    echo "✅ Dummy data initialization enabled"
  else
    INIT_DUMMY_DATA=false
    echo "⏭️  Skipping dummy data initialization"
  fi
  echo ""
}

start_init_services_if_missing() {
  echo ""
  echo "🔍 Checking init services..."
  
  for service in $INFRASTRUCTURE_INIT_SERVICES; do
    # Check if container exists (running or stopped) by container name
    # Both kafka-init and db-init have explicit container_name in docker-compose.yml
    if docker ps -a --format "{{.Names}}" | grep -q "^${service}$"; then
      echo "   ⏭️  Skipping $service (container already exists)"
    else
      echo "   🚀 Starting $service (container not found)"
      (cd "$WORKSPACE_DIR" && docker-compose up -d "$service")
    fi
  done
}

start_selected_services() {
  echo ""
  echo "🐳 Starting services with docker-compose..."
  echo ""
  echo "📋 Services to start:"
  echo "   Infrastructure:"
  echo "   • ScyllaDB, MySQL, Redis, etcd, Apache Kafka (KRaft), kafka-init, db-init"
  echo "   Management Tools:"
  echo "   • etcd-workbench, kafka-ui"
  
  if [[ $START_ONFS == true ]]; then
    echo "   • Online Feature Store API Server"
  fi
  if [[ $START_ONFS_CONSUMER == true ]]; then
    echo "   • ONFS Consumer (Kafka Ingestion)"
  fi
  if [[ $START_HORIZON == true ]]; then
    echo "   • Horizon Backend API"
  fi
  if [[ $START_NUMERIX == true ]]; then
    echo "   • Numerix Matrix Operations"
  fi
  if [[ $START_TRUFFLEBOX == true ]]; then
    echo "   • TruffleBox UI"
  fi
  if [[ $START_INFERFLOW == true ]]; then
    echo "   • Inferflow"
  fi
  if [[ $START_SKYE == true ]]; then
    echo "   • Skye (trigger, admin, consumers, serving)"
  fi
  if [[ $START_PREDATOR == true ]]; then
    echo "   • Predator (Dummy gRPC Inference Server)"
  fi
  
  
  if [[ $START_ONFS == true || $START_ONFS_CONSUMER == true || $START_HORIZON == true || $START_NUMERIX == true || $START_TRUFFLEBOX == true || $START_INFERFLOW == true || $START_SKYE == true || $START_PREDATOR == true ]]; then
    echo ""
    echo "🏷️  Application versions:"
    if [[ $START_ONFS == true ]]; then
      if [[ "$ONFS_VERSION" == "local" ]]; then
        echo "   • ONFS API Server: ${ONFS_VERSION} (building from local Dockerfile)"
      else
        echo "   • ONFS API Server: ${ONFS_VERSION}"
      fi
    fi
    if [[ $START_ONFS_CONSUMER == true ]]; then
      if [[ "$ONFS_CONSUMER_VERSION" == "local" ]]; then
        echo "   • ONFS Consumer: ${ONFS_CONSUMER_VERSION} (building from local Dockerfile)"
      else
        echo "   • ONFS Consumer: ${ONFS_CONSUMER_VERSION}"
      fi
    fi
    if [[ $START_HORIZON == true ]]; then
      if [[ "$HORIZON_VERSION" == "local" ]]; then
        echo "   • Horizon Backend: ${HORIZON_VERSION} (building from local Dockerfile)"
      else
        echo "   • Horizon Backend: ${HORIZON_VERSION}"
      fi
    fi
    if [[ $START_NUMERIX == true ]]; then
      if [[ "$NUMERIX_VERSION" == "local" ]]; then
        echo "   • Numerix Matrix: ${NUMERIX_VERSION} (building from local Dockerfile)"
      else
        echo "   • Numerix Matrix: ${NUMERIX_VERSION}"
      fi
    fi
    if [[ $START_TRUFFLEBOX == true ]]; then
      if [[ "$TRUFFLEBOX_VERSION" == "local" ]]; then
        echo "   • Trufflebox UI: ${TRUFFLEBOX_VERSION} (building from local Dockerfile)"
      else
        echo "   • Trufflebox UI: ${TRUFFLEBOX_VERSION}"
      fi
    fi
    if [[ $START_INFERFLOW == true ]]; then
      if [[ "$INFERFLOW_VERSION" == "local" ]]; then
        echo "   • Inferflow: ${INFERFLOW_VERSION} (building from local Dockerfile)"
      else
        echo "   • Inferflow: ${INFERFLOW_VERSION}"
      fi
    fi
    if [[ $START_SKYE == true ]]; then
      if [[ "$SKYE_VERSION" == "local" ]]; then
        echo "   • Skye: ${SKYE_VERSION} (building from local Dockerfile)"
      else
        echo "   • Skye: ${SKYE_VERSION}"
      fi
    fi
  else
    echo ""
    echo "🏷️  Infrastructure-only setup (no application services selected)"
  fi
  echo ""
  
  # Export version variables for docker-compose (if set in environment)
  export ONFS_VERSION
  export ONFS_CONSUMER_VERSION
  export HORIZON_VERSION
  export NUMERIX_VERSION
  export TRUFFLEBOX_VERSION
  export INFERFLOW_VERSION
  export SKYE_VERSION
  
  # Export INIT_DUMMY_DATA if set (will be passed to db-init container)
  if [[ "$INIT_DUMMY_DATA" == true ]]; then
    export INIT_DUMMY_DATA=true
    echo "   📦 Dummy data initialization will be enabled for db-init"
  fi
  
  # Rebuild db-init if dummy data is enabled (to ensure main-init.sh has the latest changes)
  if [[ "$INIT_DUMMY_DATA" == true ]]; then
    echo "   🔨 Rebuilding db-init container for dummy data support..."
    (cd "$WORKSPACE_DIR" && INIT_DUMMY_DATA=true docker-compose build db-init)
  fi
  
  # Pass INIT_DUMMY_DATA to docker-compose
  (cd "$WORKSPACE_DIR" && INIT_DUMMY_DATA="${INIT_DUMMY_DATA:-false}" CLUSTER_NAME="${CLUSTER_NAME:-bharatml-stack}" docker-compose up -d --build $SELECTED_SERVICES)
  start_init_services_if_missing
  
  echo ""
  echo "⏳ Waiting for services to start up..."
  echo "   📋 You can monitor progress with: cd $WORKSPACE_DIR && docker-compose logs -f"
  echo ""
  
  # Show brief status check
  for i in {1..30}; do
    echo -n "🔄 Checking service status (attempt $i/30)... "
    
    # Check if at least some key services are running
    running_services=$(cd "$WORKSPACE_DIR" && docker-compose ps --filter status=running --format "table {{.Name}}" | tail -n +2 | wc -l)
    if [ "$running_services" -gt 0 ]; then
      echo "✅ Services are starting up! ($running_services containers running)"
      break
    fi
    
    if [ $i -eq 30 ]; then
      echo "⏰ Services are still starting up. Check logs for details:"
      echo "   cd $WORKSPACE_DIR && docker-compose logs"
      break
    fi
    
    echo "⏳ Still starting..."
    sleep 3
  done
}

verify_services() {
  echo ""
  
  # If no application services selected, skip health checks
  if [[ $START_ONFS == false && $START_ONFS_CONSUMER == false && $START_HORIZON == false && $START_NUMERIX == false && $START_TRUFFLEBOX == false && $START_INFERFLOW == false && $START_SKYE == false ]]; then
    echo "🏥 Infrastructure-only setup - skipping application health checks..."
    echo "✅ Infrastructure services started successfully!"
    return 0
  fi
  
  echo "🏥 Health check for selected application services..."
  
  # Wait a bit more for health checks to pass
  for i in {1..20}; do
    echo -n "⚕️  Health check (attempt $i/20)... "
    
    all_healthy=true
    
    # Check ONFS API if selected
    if [[ $START_ONFS == true ]]; then
      if ! curl -s http://localhost:8089/health/self > /dev/null 2>&1; then
        all_healthy=false
      fi
    fi
    
    # Check ONFS Consumer if selected
    if [[ $START_ONFS_CONSUMER == true ]]; then
      if ! curl -s http://localhost:8090/health/self > /dev/null 2>&1; then
        all_healthy=false
      fi
    fi
    
    # Check Horizon if selected
    if [[ $START_HORIZON == true ]]; then
      if ! curl -s http://localhost:8082/health > /dev/null 2>&1; then
        all_healthy=false
      fi
    fi
    
    # Check Numerix if selected
    if [[ $START_NUMERIX == true ]]; then
      if ! curl -s http://localhost:8083/health > /dev/null 2>&1; then
        all_healthy=false
      fi
    fi
    
    # Check TruffleBox if selected
    if [[ $START_TRUFFLEBOX == true ]]; then
      if ! curl -s http://localhost:3000 > /dev/null 2>&1; then
        all_healthy=false
      fi
    fi
    
    # Check Inferflow if selected
    if [[ $START_INFERFLOW == true ]]; then
      if ! curl -s http://localhost:8085/health/self > /dev/null 2>&1; then
        all_healthy=false
      fi
    fi
    
    # Check Skye if selected
    if [[ $START_SKYE == true ]]; then
      if ! curl -s http://localhost:8092/health > /dev/null 2>&1; then
        all_healthy=false
      fi
      if ! curl -s http://localhost:8093/health > /dev/null 2>&1; then
        all_healthy=false
      fi
      if ! curl -s http://localhost:8094/health/self > /dev/null 2>&1; then
        all_healthy=false
      fi
    fi
    
    # Check Predator if selected (gRPC service, just check if port is open)
    if [[ $START_PREDATOR == true ]]; then
      if ! nc -z localhost 8001 2>/dev/null; then
        all_healthy=false
      fi
    fi
    
    if [[ $all_healthy == true ]]; then
      echo "✅ All selected application services are healthy!"
      return 0
    fi
    
    echo "⏳ Services still initializing..."
    sleep 3
  done
  
  echo "⚠️  Some services may still be starting up. Check individual service logs if needed."
  return 0
}

show_access_info() {
  echo ""
  if [[ $START_ONFS == false && $START_ONFS_CONSUMER == false && $START_HORIZON == false && $START_NUMERIX == false && $START_TRUFFLEBOX == false && $START_INFERFLOW == false && $START_SKYE == false && $START_PREDATOR == false ]]; then
    echo "🎉 BharatML Stack infrastructure is now running!"
  else
    echo "🎉 BharatML Stack services are now running!"
  fi
  echo ""
  echo "📋 Access Information:"
  echo ""
  echo "   Management Tools:"
  echo "   🔧 etcd Workbench:    http://localhost:8081"
  echo "   📊 Kafka UI:          http://localhost:8084"
  
  if [[ $START_ONFS == true || $START_ONFS_CONSUMER == true || $START_HORIZON == true || $START_NUMERIX == true || $START_TRUFFLEBOX == true || $START_INFERFLOW == true || $START_SKYE == true || $START_PREDATOR == true ]]; then
    echo ""
    echo "   Application Services:"
  fi
  
  if [[ $START_ONFS == true ]]; then
    echo "   🚀 ONFS gRPC API:     http://localhost:8089"
  fi
  if [[ $START_ONFS_CONSUMER == true ]]; then
    echo "   📥 ONFS Consumer:     http://localhost:8090"
  fi
  if [[ $START_HORIZON == true ]]; then
    echo "   📡 Horizon API:       http://localhost:8082"
  fi
  if [[ $START_NUMERIX == true ]]; then
    echo "   🔢 Numerix Matrix:    http://localhost:8083"
  fi
  if [[ $START_TRUFFLEBOX == true ]]; then
    echo "   🌐 Trufflebox UI:     http://localhost:3000"
  fi
  if [[ $START_INFERFLOW == true ]]; then
    echo "   🔮 Inferflow:         http://localhost:8085"
  fi
  if [[ $START_SKYE == true ]]; then
    echo "   🔍 Skye Admin:        http://localhost:8092"
    echo "   🔍 Skye Consumers:   http://localhost:8093"
    echo "   🔍 Skye Serving:     http://localhost:8094"
  fi
  if [[ $START_PREDATOR == true ]]; then
    echo "   🦁 Predator gRPC:     localhost:8001"
  fi

  if [[ $START_GRPCUI == true ]]; then
    echo ""
    echo "   🖥️  gRPC UI (single browser client for ALL services):"
    echo "   🖥️  gRPC UI:            http://localhost:8096"
    echo ""
    echo "   All service methods are pre-loaded. Active services:"
    if [[ $START_ONFS == true ]]; then
      echo "     📦 ONFS         → retrieve.FeatureService  (RetrieveFeatures, RetrieveDecodedResult)"
      echo "                       persist.FeatureService   (PersistFeatures)"
    fi
    if [[ $START_NUMERIX == true ]]; then
      echo "     🔢 Numerix      → numerix.Numerix           (Compute)"
    fi
    if [[ $START_INFERFLOW == true ]]; then
      echo "     🔮 Inferflow    → Inferflow                 (RetrieveModelScore)"
      echo "                       Predict                   (InferPointWise, InferPairWise, InferSlateWise)"
    fi
    if [[ $START_SKYE == true ]]; then
      echo "     🔍 Skye         → SkyeSimilarCandidateService (getSimilarCandidates)"
      echo "                       SkyeEmbeddingService        (getCandidateEmbeddingScores, getEmbeddingsForCandidates)"
    fi
    if [[ $START_PREDATOR == true ]]; then
      echo "     🦁 Predator     → inference.GRPCInferenceService (ModelInfer, ServerLive, ServerReady)"
      echo "                       grpc.health.v1.Health           (Check)"
    fi
    echo ""
    echo "   ⚠️  Each service needs its own metadata headers — add them in the"
    echo "   'Request Metadata' panel before invoking. Quick reference:"
    echo ""
    echo "     online-feature-store  → online-feature-store-caller-id: test"
    echo "                             online-feature-store-auth-token: test"
    echo "     skye                  → skye-caller-id: grpcui"
    echo "                             skye-auth-token: test"
    echo "     numerix               → numerix-caller-id: grpcui  (optional, metrics only)"
    echo "     inferflow / predator  → no headers required"
    echo ""
    echo "   📋 Helper page with per-service headers + sample request bodies:"
    echo "      http://localhost:8095"
    echo "   Sample JSON files also at: workspace/grpc-samples/<service>/<Method>.json"
  fi

  if [[ $START_TRUFFLEBOX == true ]]; then
    echo ""
    echo "🔑 Default Admin Credentials:"
    echo "   Email:    admin@admin.com"
    echo "   Password: admin"
  fi
  
  echo ""
  echo "🛠️  Useful Commands:"
  echo "   View logs:     cd $WORKSPACE_DIR && docker-compose logs -f [service-name]"
  echo "   Stop all:      cd $WORKSPACE_DIR && docker-compose down"
  echo "   Restart:       cd $WORKSPACE_DIR && docker-compose restart [service-name]"
  echo "   View status:   cd $WORKSPACE_DIR && docker-compose ps"
  echo ""
  echo "🔍 If any service isn't responding:"
  echo "   cd $WORKSPACE_DIR && docker-compose logs [service-name]"
  echo ""
}

# Handle command line arguments
# --help, -h: Show help
# --all: Start all services (non-interactive)
# --local: Start services in local mode (build docker images locally)
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
  echo "BharatML Stack Quick Start"
  echo ""
  echo "Usage:"
  echo "  ./start.sh                    # Interactive mode with service selection"
  echo "  ./start.sh --all              # Start all services (non-interactive)"
  echo "  ./start.sh --all-local        # Start all services in local mode (build docker images locally)"
  echo "  ./start.sh --local            # Start services in local mode (build docker images locally)"
  echo "  ./start.sh --dummy-data       # Initialize databases with dummy data"
  echo "  ./start.sh --help             # Show this help"
  echo ""
  echo "Flags can be combined:"
  echo "  ./start.sh --all --dummy-data # Start all services with dummy data"
  echo ""
  echo "Infrastructure (ScyllaDB, MySQL, Redis, etcd, Kafka, kafka-init) and Management Tools (etcd-workbench, kafka-ui) are always started."
  echo "You can choose which application services to start:"
  echo "  • Online Feature Store API"
  echo "  • ONFS Consumer (Kafka Ingestion)"
  echo "  • Horizon Backend"
  echo "  • Numerix Matrix Operations"
  echo "  • TruffleBox UI"
  echo "  • Inferflow"
  echo "  • Skye (Vector Similarity Search)"
  echo "  • Predator (Dummy gRPC Inference Server)"
  echo ""
  echo "Dummy Data Initialization:"
  echo "  Use --dummy-data flag to initialize databases with sample data for testing."
  echo "  This will populate MySQL, ScyllaDB, and etcd with example entities, features, and configurations."
  echo ""
  echo "Version Control:"
  echo "  Set version environment variables to control which images to use:"
  echo "  • ONFS_VERSION, ONFS_CONSUMER_VERSION, HORIZON_VERSION, SKYE_VERSION, etc."
  echo "  • Use 'local' as version to build from local Dockerfiles"
  echo "  • Example: ONFS_VERSION=local HORIZON_VERSION=v1.0.0 ./start.sh"
  echo ""
  exit 0
fi

echo "🚀 Starting BharatML Stack Quick Start..."

check_go_version
check_docker

# Check Python 3 if any version is set to "local"
if [[ "${ONFS_VERSION}" == "local" || "${ONFS_CONSUMER_VERSION}" == "local" || \
      "${HORIZON_VERSION}" == "local" || "${NUMERIX_VERSION}" == "local" || \
      "${TRUFFLEBOX_VERSION}" == "local" || "${INFERFLOW_VERSION}" == "local" || "${SKYE_VERSION}" == "local" ]]; then
  check_python3
fi

setup_workspace

# Parse command line arguments
for arg in "$@"; do
  case $arg in
    --all)
      echo "🎯 Non-interactive mode: Starting all services"
      SELECTED_SERVICES="$SELECTED_SERVICES $ONFS_SERVICES $ONFS_CONSUMER_SERVICES $HORIZON_SERVICES $NUMERIX_SERVICES $TRUFFLEBOX_SERVICES $INFERFLOW_SERVICES $SKYE_SERVICES $PREDATOR_SERVICES"
      START_ONFS=true
      START_ONFS_CONSUMER=true
      START_HORIZON=true
      START_NUMERIX=true
      START_TRUFFLEBOX=true
      START_INFERFLOW=true
      START_SKYE=true
      START_PREDATOR=true
      START_GRPCUI=true
      ;;
    --all-local)
      echo "🎯 Non-interactive mode: Starting all services in local mode"
      SELECTED_SERVICES="$SELECTED_SERVICES $ONFS_SERVICES $ONFS_CONSUMER_SERVICES $HORIZON_SERVICES $NUMERIX_SERVICES $TRUFFLEBOX_SERVICES $INFERFLOW_SERVICES $SKYE_SERVICES $PREDATOR_SERVICES"
      START_ONFS=true
      START_ONFS_CONSUMER=true
      START_HORIZON=true
      START_NUMERIX=true
      START_TRUFFLEBOX=true
      START_INFERFLOW=true
      START_SKYE=true
      START_PREDATOR=true
      START_GRPCUI=true
      ENABLE_LOCAL_BUILD=true
      ;;
    --local)
      echo "🎯 Starting services in local mode"
      ENABLE_LOCAL_BUILD=true
      ;;
    --dummy-data)
      echo "🎯 Dummy data initialization enabled"
      INIT_DUMMY_DATA=true
      ;;
  esac
done

# If --all or --all-local was not specified, use interactive mode
if [[ "$START_ONFS" == false && "$START_ONFS_CONSUMER" == false && "$START_HORIZON" == false && "$START_NUMERIX" == false && "$START_TRUFFLEBOX" == false && "$START_INFERFLOW" == false && "$START_SKYE" == false && "$START_PREDATOR" == false ]]; then
  get_user_choice
  if [ "$1" = "--local" ]; then
    ENABLE_LOCAL_BUILD=true
  fi
fi

# Setup local builds AFTER service selection (so START_* flags are set)
# Check if any version is set to "local" or if ENABLE_LOCAL_BUILD is true
if [[ "$ENABLE_LOCAL_BUILD" = true || \
      "$ONFS_VERSION" == "local" || "$ONFS_CONSUMER_VERSION" == "local" || \
      "$HORIZON_VERSION" == "local" || "$NUMERIX_VERSION" == "local" || \
      "$TRUFFLEBOX_VERSION" == "local" || "$INFERFLOW_VERSION" == "local" || "$SKYE_VERSION" == "local" ]]; then
  setup_local_builds
fi

# Add the single gRPC UI stack (nginx proxy + grpcui) whenever any gRPC service is selected
if [[ "$START_GRPCUI" == true ]]; then
  SELECTED_SERVICES="$SELECTED_SERVICES $GRPCUI_SERVICES"
fi

start_selected_services
verify_services
show_access_info

echo "✅ Setup complete! Your workspace is ready at ./$WORKSPACE_DIR"