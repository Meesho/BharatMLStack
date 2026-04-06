#!/bin/bash

set -e

WORKSPACE_DIR="workspace"

show_help() {
  echo "BharatML Stack Service Restart"
  echo ""
  echo "Usage:"
  echo "  ./restart.sh <service-name>     # Restart a specific service"
  echo "  ./restart.sh --list              # List all available services"
  echo "  ./restart.sh --help              # Show this help"
  echo ""
  echo "Examples:"
  echo "  ./restart.sh onfs-api-server    # Restart ONFS API Server"
  echo "  ./restart.sh horizon            # Restart Horizon Backend"
  echo ""
  echo "Note: This script will stop, remove, and restart the container"
  echo "      using the latest configuration from docker-compose.yml"
}

list_services() {
  echo "📋 Available Services:"
  echo ""
  
  if [ ! -d "$WORKSPACE_DIR" ] || [ ! -f "$WORKSPACE_DIR/docker-compose.yml" ]; then
    echo "⚠️  Workspace not found. Please run ./start.sh first"
    exit 1
  fi
  
  echo "Infrastructure Services:"
  echo "   • scylla"
  echo "   • mysql"
  echo "   • redis"
  echo "   • etcd"
  echo "   • kafka"
  echo ""
  echo "Application Services:"
  echo "   • onfs-api-server"
  echo "   • onfs-consumer"
  echo "   • horizon"
  echo "   • numerix"
  echo "   • trufflebox-ui"
  echo "   • inferflow"
  echo "   • skye-trigger"
  echo "   • skye-admin"
  echo "   • skye-consumers"
  echo "   • skye-serving"
  echo "   • predator"
  echo ""
  echo "Management Tools:"
  echo "   • etcd-workbench"
  echo "   • kafka-ui"
  echo ""
  echo ""
  echo "Health Check Services:"
  echo "   • onfs-healthcheck"
  echo "   • onfs-consumer-healthcheck"
  echo "   • horizon-healthcheck"
  echo "   • numerix-healthcheck"
  echo "   • trufflebox-healthcheck"
  echo "   • inferflow-healthcheck"
  echo "   • skye-admin-healthcheck"
  echo "   • skye-consumers-healthcheck"
  echo "   • skye-serving-healthcheck"
  echo "   • predator-healthcheck"
  echo ""
  echo "Init Services:"
  echo "   • db-init"
  echo "   • kafka-init"
  echo ""
  echo "💡 Tip: Use 'cd $WORKSPACE_DIR && docker-compose ps' to see running services"
}

validate_service() {
  local service_name="$1"
  
  if [ ! -d "$WORKSPACE_DIR" ] || [ ! -f "$WORKSPACE_DIR/docker-compose.yml" ]; then
    echo "❌ Workspace directory not found!"
    echo "👉 Please run ./start.sh first to set up the workspace"
    exit 1
  fi
  
  # Check if service exists in docker-compose.yml
  if ! grep -q "^  ${service_name}:" "$WORKSPACE_DIR/docker-compose.yml"; then
    echo "❌ Service '$service_name' not found in docker-compose.yml"
    echo ""
    echo "💡 Run './restart.sh --list' to see available services"
    exit 1
  fi
  
  # Warn if trying to restart infrastructure services (they should rarely need restarting)
  local infrastructure_services="scylla mysql redis etcd kafka db-init kafka-init"
  if echo "$infrastructure_services" | grep -q "\b${service_name}\b"; then
    echo "⚠️  Warning: You are restarting an infrastructure service: $service_name"
    echo "   Infrastructure services (databases, caches) should rarely be restarted"
    echo "   as this may cause data loss or service disruption."
    echo ""
    read -p "   Are you sure you want to continue? (yes/no): " confirm
    if [ "$confirm" != "yes" ]; then
      echo "   Restart cancelled."
      exit 0
    fi
  fi
}

update_docker_compose() {
  local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  
  echo "   📋 Copying latest docker-compose.yml to workspace..."
  
  # Ensure workspace directory exists
  if [ ! -d "$WORKSPACE_DIR" ]; then
    mkdir -p "$WORKSPACE_DIR"
  fi
  
  # Copy the latest docker-compose.yml from quick-start directory
  if [ -f "$script_dir/docker-compose.yml" ]; then
    cp "$script_dir/docker-compose.yml" "$WORKSPACE_DIR/"
    echo "   ✅ Updated docker-compose.yml in workspace"
  else
    echo "   ⚠️  Warning: docker-compose.yml not found in $script_dir"
    echo "   Using existing docker-compose.yml in workspace"
  fi
  
  # Also copy db-init and predator-dummy directories if they exist
  if [ -d "$script_dir/db-init" ]; then
    if [ -d "$WORKSPACE_DIR/db-init" ]; then
      rm -rf "$WORKSPACE_DIR/db-init"
    fi
    cp -r "$script_dir/db-init" "$WORKSPACE_DIR/"
  fi
  
  if [ -d "$script_dir/predator-dummy" ]; then
    if [ -d "$WORKSPACE_DIR/predator-dummy" ]; then
      rm -rf "$WORKSPACE_DIR/predator-dummy"
    fi
    cp -r "$script_dir/predator-dummy" "$WORKSPACE_DIR/"
  fi

  if [ -d "$script_dir/skye-trigger" ]; then
    if [ -d "$WORKSPACE_DIR/skye-trigger" ]; then
      rm -rf "$WORKSPACE_DIR/skye-trigger"
    fi
    cp -r "$script_dir/skye-trigger" "$WORKSPACE_DIR/"
  fi

  # Copy horizon configs directory for service config loading
  local project_root="$(cd "$script_dir/.." && pwd)"
  if [ -d "$project_root/horizon/configs" ]; then
    if [ -d "$WORKSPACE_DIR/configs" ]; then
      rm -rf "$WORKSPACE_DIR/configs"
    fi
    cp -r "$project_root/horizon/configs" "$WORKSPACE_DIR/"
    echo "   ✅ Updated configs directory in workspace"
  else
    echo "   ⚠️  Warning: horizon/configs directory not found at $project_root/horizon/configs"
  fi
}

restart_service() {
  local service_name="$1"
  
  echo "🔄 Restarting service: $service_name"
  echo ""
  
  # Update docker-compose.yml and related files from quick-start directory
  update_docker_compose
  
  # Check if container is running
  if docker ps --format "{{.Names}}" | grep -q "^${service_name}$"; then
    echo "   🛑 Stopping container..."
    (cd "$WORKSPACE_DIR" && docker-compose stop "$service_name" 2>/dev/null || true)
  fi
  
  # Check if container exists (running or stopped)
  if docker ps -a --format "{{.Names}}" | grep -q "^${service_name}$"; then
    echo "   🗑️  Removing container..."
    (cd "$WORKSPACE_DIR" && docker-compose rm -f "$service_name" 2>/dev/null || true)
  fi
  
  # Pull latest image if using a remote image (not local build)
  echo "   📥 Checking for image updates..."
  (cd "$WORKSPACE_DIR" && docker-compose pull "$service_name" 2>/dev/null || true)
  
  # Start the service with latest configuration
  # Use --no-deps to prevent restarting infrastructure services (redis, etcd, scylla, mysql, db-init)
  echo "   🚀 Starting service with latest configuration..."
  (cd "$WORKSPACE_DIR" && docker-compose up -d --no-deps "$service_name")
  
  echo ""
  echo "✅ Service '$service_name' restarted successfully!"
  echo ""
  echo "💡 Useful commands:"
  echo "   View logs:     cd $WORKSPACE_DIR && docker-compose logs -f $service_name"
  echo "   Check status: cd $WORKSPACE_DIR && docker-compose ps $service_name"
}

# Handle command line arguments
if [ "$1" = "--help" ] || [ "$1" = "-h" ] || [ -z "$1" ]; then
  show_help
  exit 0
fi

if [ "$1" = "--list" ] || [ "$1" = "-l" ]; then
  list_services
  exit 0
fi

# Restart the specified service
SERVICE_NAME="$1"
validate_service "$SERVICE_NAME"
restart_service "$SERVICE_NAME"

