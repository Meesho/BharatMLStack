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

# Verify etcd initialization
echo "  🔍 Verifying etcd configuration..."
if etcdctl --endpoints=http://etcd:2379 get /config/onfs > /dev/null 2>&1; then
  echo "  ✅ etcd configuration key '/config/onfs' created successfully"
else
  echo "  ❌ Failed to create etcd configuration key '/config/onfs'"
  exit 1
fi

echo "  📋 Creating /config/horizon/inferflow/inferflow-components/catalog configuration key..."
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/component-id "catalog_id"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/composite-id "false"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/execution-dependency "feature_initializer"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/fs-flatten-res-keys/0 "catalog_id"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/fs-id-schema-to-value-columns/0/data-type "FP32"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/fs-id-schema-to-value-columns/0/schema "catalog_id"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/fs-id-schema-to-value-columns/0/value-col "catalog_id"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/override-component/reel '{"component-id":"reel:derived_int32:reel__hero_catalog_id"}'

echo "  ✅ etcd initialization completed successfully" 