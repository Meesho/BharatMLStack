#!/bin/bash

set -e

echo "🔧 Initializing etcd..."

# Create configuration key
echo "  📋 Creating /config/onfs configuration key..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs "{}"

echo "  📋 Creating /reader keys..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/security/reader/test "{\"token\":\"test\"}"

echo "  📋 Creating /config/numerix configuration key..."
etcdctl --endpoints=http://etcd:2379 put /config/numerix/expression-config/1 "{\"expression\":\"a b c * *\"}"

echo "  📋 Creating /config/horizon/inferflow configuration key..."
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow "{\"inferflow-components\":{}}"

# Verify etcd initialization
echo "  🔍 Verifying etcd configuration..."
if etcdctl --endpoints=http://etcd:2379 get /config/onfs > /dev/null 2>&1; then
  echo "  ✅ etcd configuration key '/config/onfs' created successfully"
else
  echo "  ❌ Failed to create etcd configuration key '/config/onfs'"
  exit 1
fi

if etcdctl --endpoints=http://etcd:2379 get /config/numerix/expression-config/1 > /dev/null 2>&1; then
  echo "  ✅ etcd configuration key '/config/numerix/expression-config/1' created successfully"
else
  echo "  ❌ Failed to create etcd configuration key '/config/numerix/expression-config/1'"
  exit 1
fi

if etcdctl --endpoints=http://etcd:2379 get /config/horizon/inferflow > /dev/null 2>&1; then
  echo "  ✅ etcd configuration key '/config/horizon/inferflow' created successfully"
else
  echo "  ❌ Failed to create etcd configuration key '/config/horizon/inferflow'"
  exit 1
fi 