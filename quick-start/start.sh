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
PREDATOR_SERVICES="predator predator-healthcheck"

# Management tools
MANAGEMENT_SERVICES="etcd-workbench kafka-ui"

# Capture version variables from environment (default to latest if not set)
ONFS_VERSION="${ONFS_VERSION:-latest}"
ONFS_CONSUMER_VERSION="${ONFS_CONSUMER_VERSION:-latest}"
HORIZON_VERSION="${HORIZON_VERSION:-latest}"
NUMERIX_VERSION="${NUMERIX_VERSION:-latest}"
TRUFFLEBOX_VERSION="${TRUFFLEBOX_VERSION:-latest}"
INFERFLOW_VERSION="${INFERFLOW_VERSION:-latest}"

# Global variables for user selection
SELECTED_SERVICES="$INFRASTRUCTURE_SERVICES $MANAGEMENT_SERVICES"
START_ONFS=false
START_ONFS_CONSUMER=false
START_HORIZON=false
START_NUMERIX=false
START_TRUFFLEBOX=false
START_INFERFLOW=false
START_PREDATOR=false

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
  
  if [[ "$needs_local_build" == true ]]; then
    echo "   🔧 Modifying docker-compose.yml for local builds..."
    # Get absolute path for compose file to avoid path issues
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local compose_file_abs="$(cd "$script_dir" && cd "$WORKSPACE_DIR" && pwd)/docker-compose.yml"
    # Export variables for Python script
    export COMPOSE_FILE="$compose_file_abs"
    export START_ONFS START_ONFS_CONSUMER START_HORIZON START_NUMERIX START_TRUFFLEBOX START_INFERFLOW
    export ONFS_VERSION ONFS_CONSUMER_VERSION HORIZON_VERSION NUMERIX_VERSION TRUFFLEBOX_VERSION INFERFLOW_VERSION
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
    pattern = r'(  inferflow:\s*\n)\s+(image:.*\n)'
    replacement = r'\1    build:\n      context: ./inferflow\n      dockerfile: cmd/inferflow/Dockerfile\n    # \2'
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
    changes_made = True

if not changes_made and (start_onfs == 'true' or start_onfs_consumer == 'true' or start_horizon == 'true' or 
                         start_numerix == 'true' or start_trufflebox == 'true' or start_inferflow == 'true'):
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
  echo "   • Online Feature Store + Consumer + Horizon + Numerix + TruffleBox UI + Inferflow + Predator"
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
        SELECTED_SERVICES="$SELECTED_SERVICES $ONFS_SERVICES $ONFS_CONSUMER_SERVICES $HORIZON_SERVICES $NUMERIX_SERVICES $TRUFFLEBOX_SERVICES $INFERFLOW_SERVICES $PREDATOR_SERVICES"
        START_ONFS=true
        START_ONFS_CONSUMER=true
        START_HORIZON=true
        START_NUMERIX=true
        START_TRUFFLEBOX=true
        START_INFERFLOW=true
        START_PREDATOR=true
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
    echo "✅ Added: Inferflow"
  fi
  
  read -p "Include Predator (Dummy gRPC Inference Server)? [y/N]: " include_predator
  if [[ $include_predator =~ ^[Yy]$ ]]; then
    SELECTED_SERVICES="$SELECTED_SERVICES $PREDATOR_SERVICES"
    START_PREDATOR=true
    echo "✅ Added: Predator"
  fi
  
  echo ""
  if [[ $START_ONFS == false && $START_ONFS_CONSUMER == false && $START_HORIZON == false && $START_NUMERIX == false && $START_TRUFFLEBOX == false && $START_INFERFLOW == false && $START_PREDATOR == false ]]; then
    echo "🎯 Custom selection complete: Only infrastructure services will be started"
  else
    echo "🎯 Custom selection complete!"
  fi
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
  if [[ $START_PREDATOR == true ]]; then
    echo "   • Predator (Dummy gRPC Inference Server)"
  fi
  
  
  if [[ $START_ONFS == true || $START_ONFS_CONSUMER == true || $START_HORIZON == true || $START_NUMERIX == true || $START_TRUFFLEBOX == true || $START_INFERFLOW == true || $START_PREDATOR == true ]]; then
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
  
  (cd "$WORKSPACE_DIR" && docker-compose up -d --build $SELECTED_SERVICES)
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
  if [[ $START_ONFS == false && $START_ONFS_CONSUMER == false && $START_HORIZON == false && $START_NUMERIX == false && $START_TRUFFLEBOX == false && $START_INFERFLOW == false ]]; then
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
  if [[ $START_ONFS == false && $START_ONFS_CONSUMER == false && $START_HORIZON == false && $START_NUMERIX == false && $START_TRUFFLEBOX == false && $START_INFERFLOW == false && $START_PREDATOR == false ]]; then
    echo "🎉 BharatML Stack infrastructure is now running!"
  else
    echo "🎉 BharatML Stack services are now running!"
  fi
  echo ""
  echo "📋 Access Information:"
  echo "   🔧 etcd Workbench:    http://localhost:8081"
  echo "   📊 Kafka UI:          http://localhost:8084"
  
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
  if [[ $START_PREDATOR == true ]]; then
    echo "   🦁 Predator gRPC:     localhost:8001"
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
  echo "  ./start.sh              # Interactive mode with service selection"
  echo "  ./start.sh --all        # Start all services (non-interactive)"
  echo "  ./start.sh --local      # Start services in local mode (build docker images locally)"
  echo "  ./start.sh --help       # Show this help"
  echo ""
  echo "Infrastructure (ScyllaDB, MySQL, Redis, etcd, Kafka, kafka-init) and Management Tools (etcd-workbench, kafka-ui) are always started."
  echo "You can choose which application services to start:"
  echo "  • Online Feature Store API"
  echo "  • ONFS Consumer (Kafka Ingestion)"
  echo "  • Horizon Backend"
  echo "  • Numerix Matrix Operations"
  echo "  • TruffleBox UI"
  echo "  • Inferflow"
  echo ""
  echo "Version Control:"
  echo "  Set version environment variables to control which images to use:"
  echo "  • ONFS_VERSION, ONFS_CONSUMER_VERSION, HORIZON_VERSION, etc."
  echo "  • Use 'local' as version to build from local Dockerfiles"
  echo "  • Example: ONFS_VERSION=local HORIZON_VERSION=v1.0.0 ./start.sh"
  echo ""
  exit 0
fi

echo "🚀 Starting BharatML Stack Quick Start..."

check_go_version

# Check Python 3 if any version is set to "local"
if [[ "${ONFS_VERSION}" == "local" || "${ONFS_CONSUMER_VERSION}" == "local" || \
      "${HORIZON_VERSION}" == "local" || "${NUMERIX_VERSION}" == "local" || \
      "${TRUFFLEBOX_VERSION}" == "local" || "${INFERFLOW_VERSION}" == "local" ]]; then
  check_python3
fi

setup_workspace

# Handle non-interactive mode
if [ "$1" = "--all" ]; then
  echo "🎯 Non-interactive mode: Starting all services"
  SELECTED_SERVICES="$SELECTED_SERVICES $ONFS_SERVICES $ONFS_CONSUMER_SERVICES $HORIZON_SERVICES $NUMERIX_SERVICES $TRUFFLEBOX_SERVICES $INFERFLOW_SERVICES $PREDATOR_SERVICES"
  START_ONFS=true
  START_ONFS_CONSUMER=true
  START_HORIZON=true
  START_NUMERIX=true
  START_TRUFFLEBOX=true
  START_INFERFLOW=true
  START_PREDATOR=true
else
  # Interactive mode
  get_user_choice
fi

if [ "$1" = "--local" ]; then
  echo "🎯 Starting services in local mode"
  LOCAL_MODE=true
fi

# Setup local builds AFTER service selection (so START_* flags are set)
setup_local_builds

start_selected_services
verify_services
show_access_info

echo "✅ Setup complete! Your workspace is ready at ./$WORKSPACE_DIR"
