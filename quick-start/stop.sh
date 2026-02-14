#!/bin/bash

set -e

WORKSPACE_DIR="workspace"
PURGE_FLAG="$1"

stop_services() {
  echo "🛑 Stopping BharatML Stack services..."
  
  if [ -d "$WORKSPACE_DIR" ] && [ -f "$WORKSPACE_DIR/docker-compose.yml" ]; then
    (cd "$WORKSPACE_DIR" && docker-compose stop)
    echo "✅ All services stopped"
  else
    echo "⚠️  Workspace directory not found, stopping individual containers..."
    
    # Fallback: stop containers individually
    CONTAINERS=("onfs-consumer" "onfs-consumer-healthcheck" "trufflebox" "trufflebox-healthcheck" "numerix" "numerix-healthcheck" "horizon" "horizon-healthcheck" "onfs-api-server" "onfs-healthcheck" "inferflow" "inferflow-healthcheck" "skye-trigger" "skye-admin" "skye-admin-healthcheck" "skye-consumers" "skye-consumers-healthcheck" "skye-serving" "skye-serving-healthcheck" "kafka-ui" "etcd-workbench" "db-init" "kafka-init" "kafka" "broker" "zookeeper" "etcd" "redis" "mysql" "scylla")
    
    for container in "${CONTAINERS[@]}"; do
      if docker ps -q -f name="$container" | grep -q .; then
        echo "🛑 Stopping $container..."
        docker stop "$container" 2>/dev/null || true
      fi
    done
    
    echo "✅ Individual containers stopped"
  fi
}

remove_containers() {
  echo "🗑️  Removing containers..."
  
  if [ -d "$WORKSPACE_DIR" ] && [ -f "$WORKSPACE_DIR/docker-compose.yml" ]; then
    (cd "$WORKSPACE_DIR" && docker-compose down --volumes --remove-orphans)
  else
    # Fallback: remove containers individually
    CONTAINERS=("onfs-consumer" "onfs-consumer-healthcheck" "trufflebox" "trufflebox-healthcheck" "numerix" "numerix-healthcheck" "horizon" "horizon-healthcheck" "onfs-api-server" "onfs-healthcheck" "inferflow" "inferflow-healthcheck" "skye-trigger" "skye-admin" "skye-admin-healthcheck" "skye-consumers" "skye-consumers-healthcheck" "skye-serving" "skye-serving-healthcheck" "kafka-ui" "etcd-workbench" "db-init" "kafka-init" "kafka" "broker" "zookeeper" "etcd" "redis" "mysql" "scylla")
    
    for container in "${CONTAINERS[@]}"; do
      if docker ps -aq -f name="$container" | grep -q .; then
        echo "🗑️  Removing $container..."
        docker rm -f "$container" 2>/dev/null || true
      fi
    done
  fi
  
  echo "✅ Containers removed"
}

remove_volumes() {
  echo "💾 Removing persistent volumes..."
  
  # Remove named volumes
  VOLUMES=("scylla-data" "mysql-data" "kafka-data")
  for volume in "${VOLUMES[@]}"; do
    if docker volume ls -q | grep -q "^${volume}$"; then
      echo "🗑️  Removing volume: $volume"
      docker volume rm "$volume" 2>/dev/null || true
    fi
  done
  
  echo "✅ Volumes removed"
}

remove_images() {
  echo "🖼️  Removing Docker images..."
  
  # List of image patterns to remove
  IMAGES=("ghcr.io/meesho/onfs-consumer" "ghcr.io/meesho/trufflebox-ui" "ghcr.io/meesho/numerix" "ghcr.io/meesho/horizon" "ghcr.io/meesho/onfs-api-server" "ghcr.io/meesho/inferflow" "ghcr.io/meesho/skye-admin" "ghcr.io/meesho/skye-consumers" "ghcr.io/meesho/skye-serving" "provectuslabs/kafka-ui" "apache/kafka" "quay.io/coreos/etcd" "tzfun/etcd-workbench" "redis" "mysql" "scylladb/scylla" "workspace-db-init" "alpine")
  
  for image_pattern in "${IMAGES[@]}"; do
    # Find images that match the pattern
    IMAGE_IDS=$(docker images --filter "reference=*${image_pattern}*" -q 2>/dev/null || true)
    if [ -n "$IMAGE_IDS" ]; then
      echo "🗑️  Removing images matching: $image_pattern"
      echo "$IMAGE_IDS" | xargs docker rmi -f 2>/dev/null || true
    fi
  done
  
  # Also remove any dangling images
  DANGLING_IMAGES=$(docker images -f "dangling=true" -q 2>/dev/null || true)
  if [ -n "$DANGLING_IMAGES" ]; then
    echo "🧹 Removing dangling images..."
    echo "$DANGLING_IMAGES" | xargs docker rmi -f 2>/dev/null || true
  fi
  
  echo "✅ Images removed"
}

remove_network() {
  echo "🌐 Removing Docker network..."
  
  if docker network ls | grep -q "onfs-network"; then
    docker network rm onfs-network 2>/dev/null || true
    echo "✅ Network removed"
  fi
}

remove_workspace() {
  echo "📁 Removing workspace directory..."
  
  if [ -d "$WORKSPACE_DIR" ]; then
    rm -rf "$WORKSPACE_DIR"
    echo "✅ Workspace directory removed"
  fi
}

show_status() {
  echo ""
  echo "📊 Current Status:"
  
  # Check for running containers
  RUNNING_CONTAINERS=$(docker ps --filter "name=scylla|mysql|redis|etcd|kafka|kafka-init|broker|zookeeper|horizon|numerix|trufflebox|onfs-api-server|onfs-consumer|inferflow|skye-admin|skye-consumers|skye-serving|kafka-ui|etcd-workbench" --format "{{.Names}}" 2>/dev/null || true)
  if [ -n "$RUNNING_CONTAINERS" ]; then
    echo "🟢 Running containers:"
    echo "$RUNNING_CONTAINERS" | sed 's/^/   • /'
  else
    echo "🔴 No BharatML Stack containers running"
  fi
  
  # Check for volumes
  EXISTING_VOLUMES=$(docker volume ls -q | grep -E "scylla-data|mysql-data|kafka-data" 2>/dev/null || true)
  if [ -n "$EXISTING_VOLUMES" ]; then
    echo "💾 Existing volumes:"
    echo "$EXISTING_VOLUMES" | sed 's/^/   • /'
  else
    echo "💽 No persistent volumes found"
  fi
  
  # Check workspace
  if [ -d "$WORKSPACE_DIR" ]; then
    echo "📁 Workspace directory: exists"
  else
    echo "📂 Workspace directory: not found"
  fi
  
  echo ""
}

# Main execution
if [ "$PURGE_FLAG" = "--purge" ]; then
  echo "🚨 PURGE MODE: This will remove all containers, images, volumes, networks, and workspace"
  echo "⏳ Starting in 3 seconds... (Ctrl+C to cancel)"
  sleep 3
  
  stop_services
  remove_containers
  remove_images
  remove_volumes
  remove_network
  remove_workspace
  
  echo ""
  echo "🧹 Complete purge completed!"
  echo "💡 To start fresh, run: ./start.sh"
  
else
  echo "🛑 Stopping BharatML Stack..."
  echo "💡 Use './stop.sh --purge' to completely remove everything"
  
  stop_services
  
  echo ""
  echo "✅ Services stopped successfully!"
  echo ""
  echo "💡 Useful commands:"
  echo "   Start again:     ./start.sh"
  echo "   Complete purge:  ./stop.sh --purge"
  echo "   View status:     cd $WORKSPACE_DIR && docker-compose ps"
fi

show_status
