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
}

restart_service() {
  local service_name="$1"
  
  echo "🔄 Restarting service: $service_name"
  echo ""
  
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
  echo "   🚀 Starting service with latest configuration..."
  (cd "$WORKSPACE_DIR" && docker-compose up -d "$service_name")
  
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

