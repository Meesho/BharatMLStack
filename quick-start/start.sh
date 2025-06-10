#!/bin/bash

set -e

GO_MIN_VERSION="1.22"
INSTALL_LINK="https://go.dev/doc/install"
WORKSPACE_DIR="workspace"

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

start_all_services() {
  echo "🐳 Starting all services with docker-compose..."
  echo "   This will:"
  echo "   • Start infrastructure: ScyllaDB, MySQL, Redis, etcd"
  echo "   • Build and run database initialization"
  echo "   • Start applications: Horizon, Trufflebox UI, Online Feature Store API"
  echo ""
  echo "🏷️  Application versions:"
  echo "   • ONFS API Server: ${ONFS_VERSION:-latest}"
  echo "   • Horizon Backend: ${HORIZON_VERSION:-latest}"
  echo "   • Trufflebox UI: ${TRUFFLEBOX_VERSION:-latest}"
  
  (cd "$WORKSPACE_DIR" && docker-compose up -d)
  
  echo ""
  echo "⏳ Waiting for services to start up..."
  echo "   📋 You can monitor progress with: cd $WORKSPACE_DIR && docker-compose logs -f"
  echo ""
  
  # Show brief status check
  for i in {1..30}; do
    echo -n "🔄 Checking service status (attempt $i/30)... "
    
    if (cd "$WORKSPACE_DIR" && docker-compose ps --filter status=running | grep -q "onfs-api-server\|horizon\|trufflebox"); then
      echo "✅ Services are starting up!"
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
  echo "🏥 Final health check..."
  
  # Wait a bit more for health checks to pass
  for i in {1..20}; do
    echo -n "⚕️  Health check (attempt $i/20)... "
    
    # Check if key services are responding
    if curl -s http://localhost:8089/health/self > /dev/null 2>&1 && \
       curl -s http://localhost:8082/health > /dev/null 2>&1 && \
       curl -s http://localhost:3000 > /dev/null 2>&1; then
      echo "✅ All services healthy!"
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
  echo "🎉 BharatML Stack is now running!"
  echo ""
  echo "📋 Access Information:"
  echo "   🌐 Trufflebox UI:     http://localhost:3000"
  echo "   📡 Horizon API:       http://localhost:8082"
  echo "   🚀 ONFS gRPC API:     http://localhost:8089"
  echo "   🔧 etcd Workbench:    http://localhost:8081"
  echo ""
  echo "🔑 Default Admin Credentials:"
  echo "   Email:    admin@admin.com"
  echo "   Password: admin"
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

echo "🚀 Starting BharatML Stack Quick Start..."

check_go_version
setup_workspace
start_all_services
verify_services
show_access_info

echo "✅ Setup complete! Your workspace is ready at ./$WORKSPACE_DIR"
