#!/bin/bash

set -e

echo "🔧 Initializing etcd..."

# Create configuration key
echo "  📋 Creating /config/onfs configuration key..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs "{}"

echo "  📋 Creating /reader keys..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/security/reader/test "{\"token\":\"test\"}"

# Catalog component configuration
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/component-id "catalog_id"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/composite-id "false"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/execution-dependency "feature_initializer"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/fs-flatten-res-keys/0 "catalog_id"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/fs-id-schema-to-value-columns/0/data-type "FP32"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/fs-id-schema-to-value-columns/0/schema "catalog_id"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/fs-id-schema-to-value-columns/0/value-col "catalog_id"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/catalog/override-component/reel '{"component-id": "reel:derived_int32:reel__hero_catalog_id"}'

# User component configuration
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/user/component-id "user_id"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/user/composite-id "false"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/user/execution-dependency "feature_initializer"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/user/fs-flatten-res-keys/0 "user_id"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/user/fs-id-schema-to-value-columns/0/data-type "FP32"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/user/fs-id-schema-to-value-columns/0/schema "user_id"
etcdctl --endpoints=http://etcd:2379 put /config/horizon/inferflow/inferflow-components/user/fs-id-schema-to-value-columns/0/value-col "user_id"



# Initialize Online Feature Store entities, feature groups, and features
echo "  📋 Creating Online Feature Store entities and feature groups..."

# Create store (store-id: 1)
echo "    🗄️  Creating store..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/storage/stores/1 '{"db-type":"","conf-id":0,"table":"","max-column-size-in-bytes":1024,"max-row-size-in-bytes":102400,"primary-keys":[],"table-ttl":0}'

# Create entity: user
echo "    👤 Creating entity: user..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/label "user"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/distributed-cache/enabled ""
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/distributed-cache/ttl-in-seconds "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/distributed-cache/jitter-percentage "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/distributed-cache/conf-id "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/in-memory-cache/enabled ""
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/in-memory-cache/ttl-in-seconds "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/in-memory-cache/jitter-percentage "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/in-memory-cache/conf-id "0"
# Entity keys for user
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/keys/user_id/sequence "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/keys/user_id/entity-label "user"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/keys/user_id/column-label "user_id"

# Create entity: catalog
echo "    📦 Creating entity: catalog..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/label "catalog"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/distributed-cache/enabled ""
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/distributed-cache/ttl-in-seconds "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/distributed-cache/jitter-percentage "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/distributed-cache/conf-id "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/in-memory-cache/enabled ""
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/in-memory-cache/ttl-in-seconds "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/in-memory-cache/jitter-percentage "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/in-memory-cache/conf-id "0"
# Entity keys for catalog
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/keys/catalog_id/sequence "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/keys/catalog_id/entity-label "catalog"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/keys/catalog_id/column-label "catalog_id"

# Create feature group: user/derived_2_fp32
echo "    📊 Creating feature group: user/derived_2_fp32..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/id "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/store-id "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/data-type "FP32"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/ttl-in-seconds "86400"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/job-id ""
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/in-memory-cache-enabled "false"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/distributed-cache-enabled "false"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/active-version "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/layout-version "1"
# Feature group features
# For FP32, "0.0" serialized as 4 bytes (float32) = AAAA in base64
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/features/1/feature-meta '{"user__nqp":{"sequence":0,"default-value":"AAAA","string-length":0,"vector-length":0}}'
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/features/1/labels "user__nqp"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/features/1/default-values "0.0"
# Feature group columns (segments)
# Size = 1 feature (4 bytes FP32) + metadata (9 bytes for layout version 1) = 13 bytes
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/columns/seg_0/label "seg_0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/columns/seg_0/current-size-in-bytes "13"

# Create feature group: catalog/derived_fp32
echo "    📊 Creating feature group: catalog/derived_fp32..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/id "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/store-id "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/data-type "FP32"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/ttl-in-seconds "86400"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/job-id ""
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/in-memory-cache-enabled "false"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/distributed-cache-enabled "false"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/active-version "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/layout-version "1"
# Feature group features
# For FP32, "0.0" serialized as 4 bytes (float32) = AAAA in base64
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/features/1/feature-meta '{"nqp":{"sequence":0,"default-value":"AAAA","string-length":0,"vector-length":0}}'
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/features/1/labels "nqp"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/features/1/default-values "0.0"
# Feature group columns (segments)
# Size = 1 feature (4 bytes FP32) + metadata (9 bytes for layout version 1) = 13 bytes
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/columns/seg_0/label "seg_0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/columns/seg_0/current-size-in-bytes "13"

# Create feature group: user/derived_2_string
echo "    📊 Creating feature group: user/derived_2_string..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/id "2"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/store-id "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/data-type "STRING"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/ttl-in-seconds "86400"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/job-id ""
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/in-memory-cache-enabled "false"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/distributed-cache-enabled "false"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/active-version "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/layout-version "1"
# Feature group features
# For STRING, empty string serialized as empty bytes = "" in base64
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/features/1/feature-meta '{"region":{"sequence":0,"default-value":"","string-length":100,"vector-length":0}}'
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/features/1/labels "region"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/features/1/default-values ""
# Feature group columns (segments) - STRING type
# Size = 1 feature (0 bytes for empty string, but max 100 bytes) + metadata (9 bytes for layout version 1) = 109 bytes
# However, since string-length is 100, we allocate 100 bytes for the feature + 9 bytes metadata = 109 bytes
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/columns/seg_0/label "seg_0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/columns/seg_0/current-size-in-bytes "109"

# Verify etcd initialization
echo "  🔍 Verifying etcd configuration..."
if etcdctl --endpoints=http://etcd:2379 get /config/onfs > /dev/null 2>&1; then
  echo "  ✅ etcd configuration key '/config/onfs' created successfully"
else
  echo "  ❌ Failed to create etcd configuration key '/config/onfs'"
  exit 1
fi

if etcdctl --endpoints=http://etcd:2379 get /config/horizon/inferflow > /dev/null 2>&1; then
  echo "  ✅ etcd configuration key '/config/horizon/inferflow' created successfully"
else
  echo "  ❌ Failed to create etcd configuration key '/config/horizon/inferflow'"
  exit 1
fi

# Verify inferflow components
if etcdctl --endpoints=http://etcd:2379 get /config/horizon/inferflow/inferflow-components/user > /dev/null 2>&1; then
  echo "  ✅ Inferflow component 'user' created successfully"
else
  echo "  ❌ Failed to create inferflow component 'user'"
  exit 1
fi

if etcdctl --endpoints=http://etcd:2379 get /config/horizon/inferflow/inferflow-components/catalog > /dev/null 2>&1; then
  echo "  ✅ Inferflow component 'catalog' created successfully"
else
  echo "  ❌ Failed to create inferflow component 'catalog'"
  exit 1
fi

# Verify entities
if etcdctl --endpoints=http://etcd:2379 get /config/onfs/entities/user/label > /dev/null 2>&1; then
  echo "  ✅ Entity 'user' created successfully"
else
  echo "  ❌ Failed to create entity 'user'"
  exit 1
fi

if etcdctl --endpoints=http://etcd:2379 get /config/onfs/entities/catalog/label > /dev/null 2>&1; then
  echo "  ✅ Entity 'catalog' created successfully"
else
  echo "  ❌ Failed to create entity 'catalog'"
  exit 1
fi

# Verify feature groups
if etcdctl --endpoints=http://etcd:2379 get /config/onfs/entities/user/feature-groups/derived_2_fp32/id > /dev/null 2>&1; then
  echo "  ✅ Feature group 'user/derived_2_fp32' created successfully"
else
  echo "  ❌ Failed to create feature group 'user/derived_2_fp32'"
  exit 1
fi

if etcdctl --endpoints=http://etcd:2379 get /config/onfs/entities/catalog/feature-groups/derived_fp32/id > /dev/null 2>&1; then
  echo "  ✅ Feature group 'catalog/derived_fp32' created successfully"
else
  echo "  ❌ Failed to create feature group 'catalog/derived_fp32'"
  exit 1
fi

if etcdctl --endpoints=http://etcd:2379 get /config/onfs/entities/user/feature-groups/derived_2_string/id > /dev/null 2>&1; then
  echo "  ✅ Feature group 'user/derived_2_string' created successfully"
else
  echo "  ❌ Failed to create feature group 'user/derived_2_string'"
  exit 1
fi

echo "  ✅ etcd initialization completed successfully" 