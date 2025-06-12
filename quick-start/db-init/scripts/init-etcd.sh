#!/bin/bash

set -e

echo "🔧 Initializing etcd..."

# Create configuration key
echo "  📋 Creating /config/onfs configuration key..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs "{}"

echo "  📋 Creating /reader keys..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/security/reader/test "{\"token\":\"test\"}"

# Verify etcd initialization
echo "  🔍 Verifying etcd configuration..."
if etcdctl --endpoints=http://etcd:2379 get /config/onfs > /dev/null 2>&1; then
  echo "  ✅ etcd configuration key '/config/onfs' created successfully"
else
  echo "  ❌ Failed to create etcd configuration key"
  exit 1
fi 