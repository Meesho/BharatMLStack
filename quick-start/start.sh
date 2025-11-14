#!/bin/bash

set -e

GO_MIN_VERSION="1.22"
INSTALL_LINK="https://go.dev/doc/install"
WORKSPACE_DIR="workspace"

# Infrastructure services (always started)
INFRASTRUCTURE_SERVICES="scylla mysql redis etcd db-init"

# Application services (user selectable)
ONFS_SERVICES="onfs-api-server onfs-healthcheck"
HORIZON_SERVICES="horizon horizon-healthcheck"
NUMERIX_SERVICES="numerix numerix-healthcheck"
TRUFFLEBOX_SERVICES="trufflebox-ui trufflebox-healthcheck"

# Management tools
MANAGEMENT_SERVICES="etcd-workbench"

# Global variables for user selection
SELECTED_SERVICES="$INFRASTRUCTURE_SERVICES $MANAGEMENT_SERVICES"
START_ONFS=false
START_HORIZON=false
START_NUMERIX=false
START_TRUFFLEBOX=false

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
  
  echo "✅ Workspace setup complete"
}

show_service_menu() {
  echo ""
  echo "🎯 BharatML Stack Service Selector"
  echo "=================================="
  echo ""
  echo "Infrastructure (ScyllaDB, MySQL, Redis, etcd) will always be started."
  echo "Choose which application services to start:"
  echo ""
  echo "1) 🚀 All Services"
  echo "   • Online Feature Store + Horizon + Numerix + TruffleBox UI"
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
        SELECTED_SERVICES="$SELECTED_SERVICES $ONFS_SERVICES $HORIZON_SERVICES $NUMERIX_SERVICES $TRUFFLEBOX_SERVICES"
        START_ONFS=true
        START_HORIZON=true
        START_NUMERIX=true
        START_TRUFFLEBOX=true
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
  echo "✅ Infrastructure services (always included): ScyllaDB, MySQL, Redis, etcd"
  echo ""
  
  # Ask about each service
  read -p "Include Online Feature Store API? [y/N]: " include_onfs
  if [[ $include_onfs =~ ^[Yy]$ ]]; then
    SELECTED_SERVICES="$SELECTED_SERVICES $ONFS_SERVICES"
    START_ONFS=true
    echo "✅ Added: Online Feature Store API"
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
  
  echo ""
  if [[ $START_ONFS == false && $START_HORIZON == false && $START_NUMERIX == false && $START_TRUFFLEBOX == false ]]; then
    echo "🎯 Custom selection complete: Only infrastructure services will be started"
  else
    echo "🎯 Custom selection complete!"
  fi
}

start_selected_services() {
  echo ""
  echo "🐳 Starting services with docker-compose..."
  echo ""
  echo "📋 Services to start:"
  echo "   Infrastructure:"
  echo "   • ScyllaDB, MySQL, Redis, etcd, db-init, etcd-workbench"
  
  if [[ $START_ONFS == true ]]; then
    echo "   • Online Feature Store API Server"
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
  
  
  if [[ $START_ONFS == true || $START_HORIZON == true || $START_NUMERIX == true || $START_TRUFFLEBOX == true ]]; then
    echo ""
    echo "🏷️  Application versions:"
    if [[ $START_ONFS == true ]]; then
      echo "   • ONFS API Server: ${ONFS_VERSION:-latest}"
    fi
    if [[ $START_HORIZON == true ]]; then
      echo "   • Horizon Backend: ${HORIZON_VERSION:-latest}"
    fi
    if [[ $START_NUMERIX == true ]]; then
      echo "   • Numerix Matrix: ${NUMERIX_VERSION:-latest}"
    fi
    if [[ $START_TRUFFLEBOX == true ]]; then
      echo "   • Trufflebox UI: ${TRUFFLEBOX_VERSION:-latest}"
    fi
  else
    echo ""
    echo "🏷️  Infrastructure-only setup (no application services selected)"
  fi
  echo ""
  
  (cd "$WORKSPACE_DIR" && docker-compose up -d --build $SELECTED_SERVICES)
  
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
  if [[ $START_ONFS == false && $START_HORIZON == false && $START_NUMERIX == false && $START_TRUFFLEBOX == false ]]; then
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
  if [[ $START_ONFS == false && $START_HORIZON == false && $START_NUMERIX == false && $START_TRUFFLEBOX == false ]]; then
    echo "🎉 BharatML Stack infrastructure is now running!"
  else
    echo "🎉 BharatML Stack services are now running!"
  fi
  echo ""
  echo "📋 Access Information:"
  echo "   🔧 etcd Workbench:    http://localhost:8081"
  
  if [[ $START_ONFS == true ]]; then
    echo "   🚀 ONFS gRPC API:     http://localhost:8089"
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
  echo "Infrastructure (ScyllaDB, MySQL, Redis, etcd) is always started."
  echo "You can choose which application services to start:"
  echo "  • Online Feature Store API"
  echo "  • Horizon Backend"
  echo "  • Numerix Matrix Operations"
  echo "  • TruffleBox UI"
  echo ""
  exit 0
fi

echo "🚀 Starting BharatML Stack Quick Start..."

check_go_version
setup_workspace

# Handle non-interactive mode
if [ "$1" = "--all" ]; then
  echo "🎯 Non-interactive mode: Starting all services"
  SELECTED_SERVICES="$SELECTED_SERVICES $ONFS_SERVICES $HORIZON_SERVICES $NUMERIX_SERVICES $TRUFFLEBOX_SERVICES"
  START_ONFS=true
  START_HORIZON=true
  START_NUMERIX=true
  START_TRUFFLEBOX=true
else
  # Interactive mode
  get_user_choice
fi

if [ "$1" = "--local" ]; then
  echo "🎯 Starting services in local mode"
  LOCAL_MODE=true
fi

start_selected_services
verify_services
show_access_info

echo "✅ Setup complete! Your workspace is ready at ./$WORKSPACE_DIR"
