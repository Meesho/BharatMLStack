#!/bin/bash

set -e

echo "🔧 Initializing etcd..."

# Create configuration key
echo "  📋 Creating /config/onfs configuration key..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs "{}"

echo "  📋 Creating /reader keys..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/security/reader/test "{\"token\":\"test\"}"

echo "  📋 Creating /entity keys..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/label "user"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/distributed-cache/enabled "true"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/distributed-cache/ttl-in-seconds "3600"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/distributed-cache/jitter-percentage "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/distributed-cache/conf-id "2"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/in-memory-cache/enabled "true"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/in-memory-cache/ttl-in-seconds "3600"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/in-memory-cache/jitter-percentage "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/in-memory-cache/conf-id "3"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/p2p-cache/enabled "true"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/p2p-cache/ttl-in-seconds "3600"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/p2p-cache/jitter-percentage "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/p2p-cache/conf-id "5"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/keys/user_id/sequence "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/keys/user_id/entity-label "user_id"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/keys/user_id/column-label "id"

# Verify etcd initialization
echo "  🔍 Verifying etcd configuration..."
if etcdctl --endpoints=http://etcd:2379 get /config/onfs > /dev/null 2>&1; then
  echo "  ✅ etcd configuration key '/config/onfs' created successfully"
else
  echo "  ❌ Failed to create etcd configuration key"
  exit 1
fi 