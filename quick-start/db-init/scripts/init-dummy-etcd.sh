#!/bin/bash

set -e

echo "🔧 Initializing dummy data inetcd..."

etcdctl --endpoints=http://etcd:2379 put /config/onfs/security/reader/derived_fp32_client "{\"token\":\"test-client\"}"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/security/writer/derived_fp32_job "{\"token\":\"test-job\"}"

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

# Create entity: user
echo "    👤 Creating entity: user..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/label "user"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/distributed-cache/enabled "true"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/distributed-cache/ttl-in-seconds "30"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/distributed-cache/jitter-percentage "10"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/distributed-cache/conf-id "2"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/in-memory-cache/enabled "true"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/in-memory-cache/ttl-in-seconds "50"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/in-memory-cache/jitter-percentage "10"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/in-memory-cache/conf-id "3"
# Entity keys for user
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/keys/0/sequence "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/keys/0/entity-label "user_id"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/keys/0/column-label "user_id"

# Create entity: catalog
echo "    📦 Creating entity: catalog..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/label "catalog"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/distributed-cache/enabled "true"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/distributed-cache/ttl-in-seconds "30"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/distributed-cache/jitter-percentage "10"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/distributed-cache/conf-id "2"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/in-memory-cache/enabled "true"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/in-memory-cache/ttl-in-seconds "10"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/in-memory-cache/jitter-percentage "10"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/in-memory-cache/conf-id "3"
# Entity keys for catalog
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/keys/0/sequence "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/keys/0/entity-label "catalog_id"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/keys/0/column-label "catalog_id"

# Create feature group: user/derived_2_fp32
echo "    📊 Creating feature group: user/derived_2_fp32..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/id "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/store-id "2"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/data-type "DataTypeFP32"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/ttl-in-seconds "10"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/job-id "derived_fp32_job"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/in-memory-cache-enabled "true"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/distributed-cache-enabled "true"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/active-version "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/layout-version "1"
# Feature group features
# For FP32, "0.0" serialized as 4 bytes (float32) = AAAA in base64
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/features/1/feature-meta '{"user__nqp":{"sequence":0,"default-value":"AAAAAA==","string-length":0,"vector-length":0}}'
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/features/1/labels "user__nqp"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/features/1/default-values "0.0"
# Feature group columns (segments)
# Size = 1 feature (4 bytes FP32) + metadata (9 bytes for layout version 1) = 13 bytes
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/columns/seg_1/label "seg_1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_fp32/columns/seg_1/current-size-in-bytes "13"

# Create feature group: catalog/derived_fp32
echo "    📊 Creating feature group: catalog/derived_fp32..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/id "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/store-id "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/data-type "DataTypeFP32"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/ttl-in-seconds "300"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/job-id "derived_fp32_job"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/in-memory-cache-enabled "true"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/distributed-cache-enabled "true"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/active-version "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/layout-version "1"
# Feature group features
# For FP32, "0.0" serialized as 4 bytes (float32) = AAAA in base64
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/features/1/feature-meta '{"sbid_value":{"sequence":0,"default-value":"AAAAAA==","string-length":0,"vector-length":0}}'
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/features/1/labels "sbid_value"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/features/1/default-values "0.0"
# Feature group columns (segments)
# Size = 1 feature (4 bytes FP32) + metadata (9 bytes for layout version 1) = 13 bytes
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/columns/seg_0/label "seg_0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/catalog/feature-groups/derived_fp32/columns/seg_0/current-size-in-bytes "13"

# Create feature group: user/derived_2_string
echo "    📊 Creating feature group: user/derived_2_string..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/id "0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/store-id "2"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/data-type "DataTypeFP32"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/ttl-in-seconds "10"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/job-id "derived_fp32_job"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/in-memory-cache-enabled "true"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/distributed-cache-enabled "true"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/active-version "1"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/layout-version "1"
# Feature group features
# For STRING, empty string serialized as empty bytes = "" in base64
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/features/1/feature-meta '{"region":{"sequence":0,"default-value":null,"string-length":0,"vector-length":0}}'
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/features/1/labels "region"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/features/1/default-values "delhi"
# Feature group columns (segments) - STRING type
# Size = 1 feature (0 bytes for empty string, but max 100 bytes) + metadata (9 bytes for layout version 1) = 109 bytes
# However, since string-length is 100, we allocate 100 bytes for the feature + 9 bytes metadata = 109 bytes
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/columns/seg_0/label "seg_0"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/entities/user/feature-groups/derived_2_string/columns/seg_0/current-size-in-bytes "13"

# Create source configurations
echo "  📋 Creating source configurations..."
etcdctl --endpoints=http://etcd:2379 put '/config/onfs/source/catalog|derived_fp32|sbid_value' "TABLE|ds_dbc_ofs.catalog__organic__derived|cod_pref_and_prep_pref__search__orders_by_views_28_days_percentile|0.0"
etcdctl --endpoints=http://etcd:2379 put '/config/onfs/source/user|derived_2_fp32|user__nqp' "TABLE|ds_dbc_ofs.catalog__organic__derived|cod_pref_and_prep_pref__search__orders_by_views_28_days_percentile|0.0"
etcdctl --endpoints=http://etcd:2379 put '/config/onfs/source/user|derived_2_string|region' "TABLE|ds_dbc_ofs.catalog__organic__derived|cod_pref_and_prep_pref__search__orders_by_views_28_days_percentile|delhi"

# Create storage store configurations
echo "  📋 Creating storage store configurations..."
etcdctl --endpoints=http://etcd:2379 put /config/onfs/storage/stores/1 "{\"conf-id\":1,\"db-type\":\"scylla\",\"max-column-size-in-bytes\":1024,\"max-row-size-in-bytes\":102400,\"primary-keys\":[\"catalog_id\"],\"table\":\"feature\",\"table-ttl\":100}"
etcdctl --endpoints=http://etcd:2379 put /config/onfs/storage/stores/2 "{\"conf-id\":1,\"db-type\":\"scylla\",\"max-column-size-in-bytes\":1024,\"max-row-size-in-bytes\":102400,\"primary-keys\":[\"user_id\"],\"table\":\"user_15d\",\"table-ttl\":100}"

# Create inferflow model config
echo "  📋 Creating inferflow model config..."
etcdctl --endpoints=http://etcd:2379 put /config/inferflow/services/inferflow/model-config/config-map/fy-ad-exp '{"dag_execution_config":{"component_dependency":{"catalog":["feature_initializer"],"i1":["p1"],"p1":["catalog","user"],"user":["feature_initializer"]}},"component_config":{"cache_enabled":true,"cache_ttl":300,"cache_version":1,"feature_components":[{"component":"catalog","component_id":"catalog_id","comp_cache_enabled":true,"fs_keys":[{"schema":"catalog_id","col":"catalog_id"}],"fs_request":{"label":"catalog","featureGroups":[{"label":"derived_fp32","features":["sbid_value"],"data_type":"DataTypeFP32"}]},"fs_flatten_resp_keys":["catalog_id"]},{"component":"user","component_id":"user_id","comp_cache_enabled":true,"fs_keys":[{"schema":"user_id","col":"user_id"}],"fs_request":{"label":"user","featureGroups":[{"label":"derived_2_fp32","features":["user__nqp"],"data_type":"DataTypeFP32"},{"label":"derived_2_string","features":["region"],"data_type":"DataTypeFP32"}]},"fs_flatten_resp_keys":["user_id"]}],"predator_components":[{"component":"p1","component_id":"catalog_id","model_name":"ensemble_personalised_nqd_lgbm_v2_2","model_end_point":"predator","deadline":1000000,"batch_size":250,"inputs":[{"name":"input__0","features":["user:derived_2_fp32:user__nqp","catalog:derived_fp32:sbid_value","user:derived_2_string:region"],"shape":[3],"data_type":"BYTES"}],"outputs":[{"name":"output__0","model_scores":["pctr_score"],"model_scores_dims":[[1]],"data_type":"FP32"}]}],"numerix_components":[{"component":"i1","component_id":"catalog_id","score_col":"score","compute_id":"1","score_mapping":{"pctr@DataTypeFP32":"pctr_score"},"data_type":"DataTypeFP32"}]},"response_config":{"logging_perc":1,"model_schema_features_perc":0,"features":["catalog_id","score","pctr_score"],"log_features":false,"log_batch_size":1000}}'

# Create numerix expression config
echo "  📋 Creating numerix expression config..."
etcdctl --endpoints=http://etcd:2379 put /config/numerix/expression-config/1 '{"expression":"pctr exp"}'

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

# Verify inferflow model config
if etcdctl --endpoints=http://etcd:2379 get /config/inferflow/services/inferflow/model-config/config-map/fy-ad-exp > /dev/null 2>&1; then
  echo "  ✅ Inferflow model config 'fy-ad-exp' created successfully"
else
  echo "  ❌ Failed to create inferflow model config 'fy-ad-exp'"
  exit 1
fi

# Verify numerix expression config
if etcdctl --endpoints=http://etcd:2379 get /config/numerix/expression-config/1 > /dev/null 2>&1; then
  echo "  ✅ Numerix expression config '1' created successfully"
else
  echo "  ❌ Failed to create numerix expression config '1'"
  exit 1
fi

echo "  ✅ etcd dummy data initialization completed successfully" 