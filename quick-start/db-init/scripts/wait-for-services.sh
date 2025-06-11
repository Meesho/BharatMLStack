#!/bin/bash

set -e

# Wait for ports to be ready
wait_for_port() {
  local name=$1
  local host=$2
  local port=$3

  echo -n "  Waiting for $name on $host:$port... "
  while ! nc -z "$host" "$port"; do
    sleep 1
  done
  echo "✅"
}

echo "⏳ Waiting for infrastructure services to be ready..."

wait_for_port "ScyllaDB" "scylla" 9042
wait_for_port "MySQL" "mysql" 3306
wait_for_port "Redis" "redis" 6379
wait_for_port "etcd" "etcd" 2379

# Wait for ScyllaDB CQL to be fully ready
echo -n "  Waiting for ScyllaDB CQL service to be ready... "
until cqlsh scylla 9042 -e "SELECT now() FROM system.local" > /dev/null 2>&1; do
    sleep 2
done
echo "✅"

echo "✅ All infrastructure services are ready!" 