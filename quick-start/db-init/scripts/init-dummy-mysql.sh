#!/bin/bash

set -e

echo "🗂️ Initializing dummy data into MySQL..."

# Insert API resolvers
echo "  🔌 Inserting API resolvers..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  -- Predator APIs
  INSERT INTO api_resolvers (method, api_path, resolver_fn) VALUES
  ('GET', '/api/v1/horizon/predator-config-registry/model-params', 'ModelParamsResolver'),
  ('POST', '/api/v1/horizon/predator-config-registry/models/onboard', 'ModelOnboardResolver'),
  ('POST', '/api/v1/horizon/predator-config-registry/models/edit', 'ModelOnboardResolver'),
  ('POST', '/api/v1/horizon/predator-config-registry/models/promote', 'ModelPromoteResolver'),
  ('POST', '/api/v1/horizon/predator-config-registry/models/scale-up', 'ModelScaleUpResolver'),
  ('PATCH', '/api/v1/horizon/predator-config-registry/models/delete', 'ModelDeleteResolver'),
  ('POST', '/api/v1/horizon/predator-config-registry/upload-json', 'ModelUploadMetaDataResolver'),
  ('POST', '/api/v1/horizon/predator-config-registry/upload-model-folder', 'ModelUploadMetaDataResolver'),
  ('GET', '/api/v1/horizon/predator-config-discovery/models', 'ModelDiscoveryResolver'),
  ('GET', '/api/v1/horizon/predator-config-discovery/source-models', 'ModelSourceDiscoveryResolver'),
  ('GET', '/api/v1/horizon/predator-config-discovery/feature-types', 'ModelSourceDiscoveryResolver'),
  ('PUT', '/api/v1/horizon/predator-config-approval/process-request', 'ModelApprovalResolver'),
  ('GET', '/api/v1/horizon/predator-config-approval/requests/:group_id', 'ModelValidatorResolver'),
  ('GET', '/api/v1/horizon/predator-config-approval/requests', 'ModelRequestDiscoveryResolver'),
  ('POST', '/api/v1/horizon/predator-config-testing/generate-request', 'ModelTestGenerateRequestResolver'),
  ('POST', '/api/v1/horizon/predator-config-testing/functional-testing/execute-request', 'ModelFunctionalTestResolver'),
  ('POST', '/api/v1/horizon/predator-config-testing/load-testing/execute-request', 'ModelLoadTestResolver'),

  -- Inferflow APIs
  ('POST', '/api/v1/horizon/inferflow-config-registry/onboard', 'InferflowOnboardResolver'),
  ('POST', '/api/v1/horizon/inferflow-config-registry/promote', 'InferflowPromoteResolver'),
  ('POST', '/api/v1/horizon/inferflow-config-registry/edit', 'InferflowEditResolver'),
  ('POST', '/api/v1/horizon/inferflow-config-registry/clone', 'InferflowCloneResolver'),
  ('POST', '/api/v1/horizon/inferflow-config-registry/scale-up', 'InferflowScaleUpResolver'),
  ('GET', '/api/v1/horizon/inferflow-config-registry/logging-ttl', 'InferflowDiscoveryResolver'),
  ('PATCH', '/api/v1/horizon/inferflow-config-registry/delete', 'InferflowDeleteResolver'),
  ('GET', '/api/v1/horizon/inferflow-config-registry/latestRequest/:config_id', 'InferflowDiscoveryResolver'),
  ('GET', '/api/v1/horizon/inferflow-config-discovery/configs', 'InferflowDiscoveryResolver'),
  ('POST', '/api/v1/horizon/inferflow-config-approval/review', 'InferflowRequestReviewResolver'),
  ('POST', '/api/v1/horizon/inferflow-config-approval/cancel', 'InferflowRequestCancelResolver'),
  ('GET', '/api/v1/horizon/inferflow-config-approval/configs', 'InferflowRequestDiscoveryResolver'),
  ('GET', '/api/v1/horizon/inferflow-config-approval/validate/:request_id', 'InferflowRequestValidateResolver'),
  ('POST', '/api/v1/horizon/inferflow-config-testing/functional-test/generate-request', 'InferflowTestGenerateRequestResolver'),
  ('POST', '/api/v1/horizon/inferflow-config-testing/functional-test/execute-request', 'InferflowTestExecuteRequestResolver'),

  -- Numerix APIs
  ('POST', '/api/v1/horizon/numerix-expression/generate', 'NumerixExpressionGenerateResolver'),
  ('GET', '/api/v1/horizon/numerix-expression/:config_id/variables', 'NumerixExpressionVariablesResolver'),
  ('POST', '/api/v1/horizon/numerix-config-registry/onboard', 'NumerixConfigOnboardResolver'),
  ('POST', '/api/v1/horizon/numerix-config-registry/promote', 'NumerixConfigPromoteResolver'),
  ('POST', '/api/v1/horizon/numerix-config-registry/edit', 'NumerixConfigEditResolver'),
  ('GET', '/api/v1/horizon/numerix-config-discovery/configs', 'NumerixConfigDiscoveryResolver'),
  ('POST', '/api/v1/horizon/numerix-config-approval/review', 'NumerixConfigRequestReviewResolver'),
  ('POST', '/api/v1/horizon/numerix-config-approval/cancel', 'NumerixConfigRequestCancelResolver'),
  ('GET', '/api/v1/horizon/numerix-config-approval/configs', 'NumerixConfigRequestDiscoveryResolver'),
  ('POST', '/api/v1/horizon/numerix-testing/functional-testing/generate-request', 'NumerixTestGenerateRequestResolver'),
  ('POST', '/api/v1/horizon/numerix-testing/functional-testing/execute-request', 'NumerixTestExecuteRequestResolver'),

  -- Application APIs
  ('POST', '/api/v1/horizon/application-registry/applications', 'ApplicationOnboardResolver'),
  ('PUT', '/api/v1/horizon/application-registry/applications/:token', 'ApplicationEditResolver'),
  ('GET', '/api/v1/horizon/application-discovery/applications', 'ApplicationDiscoveryRegistryResolver'),

  -- Connection Config APIs
  ('POST', '/api/v1/horizon/service-connection-registry/connections', 'ConnectionConfigOnboardResolver'),
  ('PUT', '/api/v1/horizon/service-connection-registry/connections/:id', 'ConnectionConfigEditResolver'),
  ('GET', '/api/v1/horizon/service-connection-discovery/connections', 'ConnectionConfigDiscoveryResolver'),

  -- Deployable APIs
  ('POST', '/api/v1/horizon/deployable-registry/deployables', 'DeployableCreateResolver'),
  ('PUT', '/api/v1/horizon/deployable-registry/deployables', 'DeployableUpdateResolver'),
  ('POST', '/api/v1/horizon/deployable-registry/deployables/refresh', 'DeployableRefreshResolver'),
  ('GET', '/api/v1/horizon/deployable-discovery/deployables/metadata', 'DeployableMetadataViewResolver'),
  ('GET', '/api/v1/horizon/deployable-discovery/deployables', 'DeployableViewResolver'),
  ('PUT', '/api/v1/horizon/deployable-registry/deployables/tune-thresholds', 'DeployableTuneResolver');
"

# Insert role permissions for admin role
echo "  🔐 Inserting role permissions for admin role..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  -- Predator permissions
  INSERT INTO role_permission (role, service, screen_type, module) VALUES
  ('admin', 'predator', 'model-approval', 'review'),
  ('admin', 'predator', 'model-approval', 'validate'),
  ('admin', 'predator', 'model-approval', 'view'),
  ('admin', 'predator', 'model', 'view'),
  ('admin', 'predator', 'model', 'delete'),
  ('admin', 'predator', 'model', 'scale_up'),
  ('admin', 'predator', 'model', 'promote'),
  ('admin', 'predator', 'model', 'onboard'),
  ('admin', 'predator', 'model', 'test'),

  -- Inferflow permissions
  ('admin', 'inferflow', 'inferflow-config', 'view'),
  ('admin', 'inferflow', 'inferflow-config', 'scale-up'),
  ('admin', 'inferflow', 'inferflow-config', 'delete'),
  ('admin', 'inferflow', 'inferflow-config', 'clone'),
  ('admin', 'inferflow', 'inferflow-config', 'promote'),
  ('admin', 'inferflow', 'inferflow-config', 'onboard'),
  ('admin', 'inferflow', 'inferflow-config', 'edit'),
  ('admin', 'inferflow', 'inferflow-config-approval', 'review'),
  ('admin', 'inferflow', 'inferflow-config-approval', 'cancel'),
  ('admin', 'inferflow', 'inferflow-config-approval', 'validate'),
  ('admin', 'inferflow', 'inferflow-config-approval', 'view'),
  ('admin', 'inferflow', 'inferflow-config-testing', 'test'),

  -- Numerix permissions
  ('admin', 'numerix', 'numerix-config', 'view'),
  ('admin', 'numerix', 'numerix-config', 'onboard'),
  ('admin', 'numerix', 'numerix-config', 'promote'),
  ('admin', 'numerix', 'numerix-config', 'edit'),
  ('admin', 'numerix', 'numerix-config-approval', 'review'),
  ('admin', 'numerix', 'numerix-config-approval', 'cancel'),
  ('admin', 'numerix', 'numerix-config-approval', 'view'),
  ('admin', 'numerix', 'numerix-config-approval', 'delete'),
  ('admin', 'numerix', 'numerix-config-testing', 'test'),

  -- Application permissions
  ('admin', 'application', 'application-discovery-registry', 'view'),
  ('admin', 'application', 'application-discovery-registry', 'onboard'),
  ('admin', 'application', 'application-discovery-registry', 'edit'),
  ('admin', 'application', 'connection-config', 'view'),
  ('admin', 'application', 'connection-config', 'onboard'),
  ('admin', 'application', 'connection-config', 'edit'),

  -- Deployable permissions
  ('admin', 'all', 'deployable', 'onboard'),
  ('admin', 'all', 'deployable', 'edit'),
  ('admin', 'all', 'deployable', 'view');
"

# Insert dummy data into MySQL
echo "  👤 Inserting dummy data into MySQL..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "

  INSERT INTO group_id_counter (id, counter, created_at, updated_at) 
  VALUES (1, 1, NOW(), NOW());

  INSERT INTO service_deployable_config (
    id, name, host, service, active, created_by, updated_by,
    created_at, updated_at, config, monitoring_url, deployable_running_status,
    deployable_work_flow_id, deployment_run_id, deployable_health, work_flow_status
  ) VALUES (
      59,
      'predator-search-ct-gpu',
      'predator-search-ct-gpu.prd.int',
      'predator',
      1,
      'admin@admin.com',
      '',
      '2025-09-01 15:03:14',
      '2025-09-01 15:31:31',
      '{\"cpuLimit\":\"7000\",\"gpu_limit\":\"1\",\"cpuRequest\":\"6500\",\"gpu_request\":\"1\",\"max_replica\":\"300\",\"memoryLimit\":\"15\",\"min_replica\":\"2\",\"cpuLimitUnit\":\"m\",\"machine_type\":\"GPU\",\"cpu_threshold\":\"70\",\"gpu_threshold\":\"60\",\"memoryRequest\":\"13\",\"cpuRequestUnit\":\"m\",\"serviceAccount\":\"sa-dsci-predator-datascience-prd-0622.iam.gserviceaccount.com\",\"gcs_bucket_path\":\"gs://gcs-dsci-model-repository-prd/predator/ranking/search-ct-gpu/*\",\"gcs_triton_path\":\"gs://gcs-dsci-model-repository-prd/scripts/launch_triton_server.py\",\"memoryLimitUnit\":\"G\",\"triton_image_tag\":\"gpu-py-ensem-trt-onnx-pytorch-fil-ensem-cache-met-v2.48.0\",\"memoryRequestUnit\":\"G\",\"nodeSelectorValue\":\"mlp-g2-standard-8\",\"deploymentStrategy\":\"rollingUpdate\"}',
      'https://grafana-prd.gcp.in/d/a2605923-52c4-4834-bdae-97570966b765/predator?orgId=1&var-service=predator-search-ct-gpu&var-query0=',
      1,
      'gcp_prd-predator-search-ct-gpu-mlp-deployment-workflow-01 Sep 2025 15:05:57.698922063',
      '9829a49e-616d-4d4b-a8c8-c5ac8de9c090',
      'DEPLOYMENT_REASON_ARGO_APP_HEALTHY',
      'WORKFLOW_COMPLETED'
  );

  INSERT INTO service_deployable_config (
    id, name, host, service, active, created_by, updated_by,
    created_at, updated_at, config, monitoring_url, deployable_running_status,
    deployable_work_flow_id, deployment_run_id, deployable_health, work_flow_status
  ) VALUES (
      3019,
      'inferflow',
      'inferflow:8085',
      'inferflow',
      1,
      'admin@admin.com',
      NULL,
      '2025-10-24 16:57:45',
      '2025-11-17 16:04:29',
      '{\"cpuLimit\":\"4000\",\"gpu_limit\":\"\",\"cpuRequest\":\"3500\",\"gpu_request\":\"\",\"max_replica\":\"100\",\"memoryLimit\":\"8\",\"min_replica\":\"2\",\"cpuLimitUnit\":\"m\",\"memoryRequest\":\"6\",\"cpuRequestUnit\":\"m\",\"memoryLimitUnit\":\"G\",\"memoryRequestUnit\":\"G\"}',
      'https://grafana-prd.example.com/d/dQ1Gux-Vk/inferflow-service?orgId=1&refresh=60s&var-service=inferflow-service-search-ct&from=now-6h&to=now',
      1,
      NULL,
      NULL,
      'DEPLOYMENT_REASON_ARGO_APP_HEALTHY',
      'WORKFLOW_COMPLETED'
  );

  INSERT INTO service_deployable_config (
    id, name, host, service, active, created_by, updated_by,
    created_at, updated_at, config, monitoring_url, deployable_running_status,
    deployable_work_flow_id, deployment_run_id, deployable_health, work_flow_status
  ) VALUES (
      1,
      'numerix',
      'numerix:8083',
      'numerix',
      1,
      'admin@admin.com',
      NULL,
      '2025-11-28 10:00:00',
      '2025-11-28 10:00:00',
      '{\"cpuLimit\": \"3500\", \"gpu_limit\": \"\", \"cpuRequest\": \"2500\", \"gpu_request\": \"0\", \"max_replica\": \"20\", \"memoryLimit\": \"500\", \"min_replica\": \"2\", \"cpuLimitUnit\": \"m\", \"machine_type\": \"CPU\", \"cpu_threshold\": \"70\", \"gpu_threshold\": \"\", \"memoryRequest\": \"200\", \"cpuRequestUnit\": \"m\", \"serviceAccount\": \"\", \"gcs_bucket_path\": \"\", \"memoryLimitUnit\": \"M\", \"triton_image_tag\": \"\", \"memoryRequestUnit\": \"M\", \"nodeSelectorValue\": \"rust-onboard\"}',
      NULL,
      NULL,
      NULL,
      NULL,
      'DEPLOYMENT_REASON_ARGO_APP_HEALTHY',
      'WORKFLOW_COMPLETED'
  );


  INSERT INTO predator_requests (
    request_id, group_id, model_name, payload, created_by, updated_by,
    reviewer, request_stage, request_type, status, active, reject_reason,
    created_at, updated_at, is_valid
  ) VALUES (
      128,
      179,
      'preprocessing_rv_ct_scale_up_v2',
      '{\"meta_data\":{\"inputs\":[{\"dims\":[28],\"name\":\"INPUT__0\",\"data_type\":\"BYTES\"}],\"backend\":\"python\",\"outputs\":[{\"dims\":[24],\"name\":\"OUTPUT__0\",\"data_type\":\"FP32\"}],\"batch_size\":500,\"instance_type\":\"KIND_CPU\",\"instance_count\":1,\"dynamic_batching_enabled\":true},\"model_name\":\"preprocessing_rv_ct_scale_up_v2\",\"config_mapping\":{\"service_deployable_id\":59},\"model_source_path\":\"gs://gcs-model-repository-int/predator/cpu/preprocessing_rv_ct_scale_up_v2\"}',
      'admin@admin.com',
      'admin@admin.com',
      'admin@admin.com',
      'Restart Deployable',
      'Promote',
      'Approved',
      0,
      '',
      '2025-11-26 18:18:39',
      '2025-11-26 18:22:50',
      1
  );

  INSERT INTO discovery_config (
    id, app_token, service_deployable_id, circuit_breaker_id, service_connection_id,
    route_service_connection_id, route_service_deployable_id, route_percent,
    active, default_response, default_response_percent, default_response_enabled_forCB,
    created_by, updated_by, created_at, updated_at
  ) VALUES (
      1425,
      '',
      3019,
      NULL,
      3,
      NULL,
      NULL,
      NULL,
      1,
      '',
      NULL,
      0,
      'admin@admin.com',
      'admin@admin.com',
      '2025-11-26 18:22:50',
      '2025-11-26 18:22:50'
  );

  INSERT INTO discovery_config (
    id, app_token, service_deployable_id, circuit_breaker_id, service_connection_id,
    route_service_connection_id, route_service_deployable_id, route_percent,
    active, default_response, default_response_percent, default_response_enabled_forCB,
    created_by, updated_by, created_at, updated_at
  ) VALUES (
    1104,
    '',
    59,
    NULL,
    0,
    NULL,
    NULL,
    NULL,
    1,
    '',
    NULL,
    0,
    'admin@admin.com',
    'admin@admin.com',
    '2025-10-30 17:17:20',
    '2025-10-30 17:17:20'
  );

  INSERT INTO predator_config (
    id, discovery_config_id, model_name, meta_data, active,
    created_by, updated_by, created_at, updated_at, test_results, has_nil_data
  ) VALUES (
      997,
      1104,
      'ensemble_personalised_nqd_lgbm_v2_2',
      '{\"inputs\":[{\"dims\":[3],\"name\":\"input__0\",\"features\":[\"ONLINE_FEATURE|user:derived_2_fp32:user__nqp\",\"ONLINE_FEATURE|catalog:derived_fp32:sbid_value\",\"ONLINE_FEATURE|user:derived_2_string:region\"],\"data_type\":\"BYTES\"}],\"backend\":\"\",\"outputs\":[{\"dims\":[1],\"name\":\"output__0\",\"data_type\":\"FP32\"}],\"platform\":\"ensemble\",\"batch_size\":250,\"instance_type\":\"\",\"instance_count\":0,\"ensemble_scheduling\":{\"step\":[{\"input_map\":{\"INPUT__0\":\"input__0\"},\"model_name\":\"preprocessing_personalised_nqd_lgbm_v2\",\"output_map\":{\"OUTPUT__0\":\"OUTPUT__0\"},\"model_version\":1},{\"input_map\":{\"input__0\":\"OUTPUT__0\"},\"model_name\":\"personalised_nqd_lgbm_v2\",\"output_map\":{\"output__0\":\"output__0\"},\"model_version\":1}]},\"dynamic_batching_enabled\":false}',
      1,
      'admin@admin.com',
      'admin@admin.com',
      '2025-10-30 17:17:20',
      '2025-11-05 18:01:19',
      '{\"is_functionally_tested\": true}',
      0
  );

  INSERT INTO service_connection_config (
    id,
    \`default\`,
    service,
    conn_protocol,
    http_config,
    grpc_config,
    active,
    created_at,
    updated_at,
    created_by,
    updated_by
  ) VALUES (
      3,
      0,
      'INFERFLOW',
      'grpc',
      '0',
      '1500000',
      0,
      '2024-12-07 19:08:19',
      '2024-12-07 22:15:28',
      'admin@admin.com',
      ''
  );


  INSERT INTO entity (
    payload, entity_label, created_by, approved_by,
    status, request_type, service, reject_reason
  ) VALUES (
    '{\"entity-label\":\"user\",\"key-map\":{\"user_id\":{\"data-type\":\"STRING\"}},\"distributed-cache\":{},\"in-memory-cache\":{}}',
    'user','admin@admin.com','admin@admin.com','Approved','Onboard','ONLINE_FEATURE_STORE',''
  ), (
    '{\"entity-label\":\"catalog\",\"key-map\":{\"catalog_id\":{\"data-type\":\"STRING\"}},\"distributed-cache\":{},\"in-memory-cache\":{}}',
    'catalog','admin@admin.com','admin@admin.com','Approved','Onboard','ONLINE_FEATURE_STORE',''
  );

  INSERT INTO feature_group (
    entity_label, feature_group_label, payload,
    created_by, approved_by, status, request_type, service, reject_reason
  ) VALUES (
    'user',
    'derived_2_fp32',
    '{\"entity-label\":\"user\",\"fg-label\":\"derived_2_fp32\",\"job-id\":\"\",\"store-id\":1,\"ttl-in-seconds\":86400,\"in-memory-cache-enabled\":false,\"distributed-cache-enabled\":false,\"data-type\":\"FP32\",\"features\":{\"user__nqp\":{\"labels\":\"user__nqp\",\"feature-meta\":{\"user__nqp\":{\"sequence\":0,\"default-value\":\"\",\"string-length\":0,\"vector-length\":0}}}},\"layout-version\":1}',
    'admin@admin.com','admin@admin.com','Approved','Onboard','ONLINE_FEATURE_STORE',''
  ), (
    'catalog',
    'derived_fp32',
    '{\"entity-label\":\"catalog\",\"fg-label\":\"derived_fp32\",\"job-id\":\"\",\"store-id\":1,\"ttl-in-seconds\":86400,\"in-memory-cache-enabled\":false,\"distributed-cache-enabled\":false,\"data-type\":\"FP32\",\"features\":{\"sbid_value\":{\"labels\":\"sbid_value\",\"feature-meta\":{\"sbid_value\":{\"sequence\":0,\"default-value\":\"\",\"string-length\":0,\"vector-length\":0}}}},\"layout-version\":1}',
    'admin@admin.com','admin@admin.com','Approved','Onboard','ONLINE_FEATURE_STORE',''
  ), (
    'user',
    'derived_2_string',
    '{\"entity-label\":\"user\",\"fg-label\":\"derived_2_string\",\"job-id\":\"\",\"store-id\":1,\"ttl-in-seconds\":86400,\"in-memory-cache-enabled\":false,\"distributed-cache-enabled\":false,\"data-type\":\"STRING\",\"features\":{\"region\":{\"labels\":\"region\",\"feature-meta\":{\"region\":{\"sequence\":0,\"default-value\":\"\",\"string-length\":100,\"vector-length\":0}}}},\"layout-version\":1}',
    'admin@admin.com','admin@admin.com','Approved','Onboard','ONLINE_FEATURE_STORE',''
  );

  INSERT INTO features (
    entity_label, feature_group_label, payload,
    created_by, approved_by, status, request_type, service, reject_reason
  ) VALUES (
      'user',
      'derived_2_fp32',
      '{\"labels\":\"user__nqp\",\"default-values\":\"0.0\",\"source-base-path\":\"\",\"source-data-column\":\"\",\"storage-provider\":\"\",\"string-length\":\"0\",\"vector-length\":\"0\"}',
      'admin@admin.com','admin@admin.com','Approved','Onboard','ONLINE_FEATURE_STORE',''
  ), (
      'catalog',
      'derived_fp32',
      '{\"labels\":\"sbid_value\",\"default-values\":\"0.0\",\"source-base-path\":\"\",\"source-data-column\":\"\",\"storage-provider\":\"\",\"string-length\":\"0\",\"vector-length\":\"0\"}',
      'admin@admin.com','admin@admin.com','Approved','Onboard','ONLINE_FEATURE_STORE',''
  ), (
      'user',
      'derived_2_string',
      '{\"labels\":\"region\",\"default-values\":\"\",\"source-base-path\":\"\",\"source-data-column\":\"\",\"storage-provider\":\"\",\"string-length\":\"100\",\"vector-length\":\"0\"}',
      'admin@admin.com','admin@admin.com','Approved','Onboard','ONLINE_FEATURE_STORE',''
  );
"

# Insert inferflow config and request
echo "  🔄 Inserting inferflow config and request..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  INSERT INTO inferflow_config (
    discovery_id, config_id, config_value, created_by, active, created_at, updated_at
  ) VALUES (
    1425,
    'fy-ad-exp',
    '{\"dag_execution_config\":{\"component_dependency\":{\"catalog\":[\"feature_initializer\"],\"i1\":[\"p1\"],\"p1\":[\"catalog\",\"user\"],\"user\":[\"feature_initializer\"]}},\"component_config\":{\"cache_enabled\":true,\"cache_ttl\":300,\"cache_version\":1,\"feature_components\":[{\"component\":\"catalog\",\"component_id\":\"catalog_id\",\"comp_cache_enabled\":true,\"fs_keys\":[{\"schema\":\"catalog_id\",\"col\":\"catalog_id\"}],\"fs_request\":{\"label\":\"catalog\",\"featureGroups\":[{\"label\":\"derived_fp32\",\"features\":[\"sbid_value\"],\"data_type\":\"DataTypeFP32\"}]},\"fs_flatten_resp_keys\":[\"catalog_id\"]},{\"component\":\"user\",\"component_id\":\"user_id\",\"comp_cache_enabled\":true,\"fs_keys\":[{\"schema\":\"user_id\",\"col\":\"user_id\"}],\"fs_request\":{\"label\":\"user\",\"featureGroups\":[{\"label\":\"derived_2_fp32\",\"features\":[\"user__nqp\"],\"data_type\":\"DataTypeFP32\"},{\"label\":\"derived_2_string\",\"features\":[\"region\"],\"data_type\":\"DataTypeFP32\"}]},\"fs_flatten_resp_keys\":[\"user_id\"]}],\"predator_components\":[{\"component\":\"p1\",\"component_id\":\"catalog_id\",\"model_name\":\"ensemble_personalised_nqd_lgbm_v2_2\",\"model_end_point\":\"predator\",\"deadline\":110,\"batch_size\":250,\"inputs\":[{\"name\":\"input__0\",\"features\":[\"user:derived_2_fp32:user__nqp\",\"catalog:derived_fp32:sbid_value\",\"user:derived_2_string:region\"],\"shape\":[3],\"data_type\":\"BYTES\"}],\"outputs\":[{\"name\":\"output__0\",\"model_scores\":[\"pctr_score\"],\"model_scores_dims\":[[1]],\"data_type\":\"FP32\"}]}],\"numerix_components\":[{\"component\":\"i1\",\"component_id\":\"catalog_id\",\"score_col\":\"score\",\"compute_id\":\"1\",\"score_mapping\":{\"pctr@DataTypeFP32\":\"pctr_score\"},\"data_type\":\"DataTypeFP32\"}]},\"response_config\":{\"logging_perc\":1,\"model_schema_features_perc\":0,\"features\":[\"catalog_id\",\"score\",\"pctr_score\"],\"log_features\":false,\"log_batch_size\":1000}}',
    'admin@admin.com',
    1,
    NOW(),
    NOW()
  );

  INSERT INTO inferflow_request (
    config_id, payload, created_by, version, status, request_type, active, created_at, updated_at
  ) VALUES (
    'fy-ad-exp',
    '{\"dag_execution_config\":{\"component_dependency\":{\"catalog\":[\"feature_initializer\"],\"i1\":[\"p1\"],\"p1\":[\"catalog\",\"user\"],\"user\":[\"feature_initializer\"]}},\"component_config\":{\"cache_enabled\":true,\"cache_ttl\":300,\"cache_version\":1,\"feature_components\":[{\"component\":\"catalog\",\"component_id\":\"catalog_id\",\"comp_cache_enabled\":true,\"fs_keys\":[{\"schema\":\"catalog_id\",\"col\":\"catalog_id\"}],\"fs_request\":{\"label\":\"catalog\",\"featureGroups\":[{\"label\":\"derived_fp32\",\"features\":[\"sbid_value\"],\"data_type\":\"DataTypeFP32\"}]},\"fs_flatten_resp_keys\":[\"catalog_id\"]},{\"component\":\"user\",\"component_id\":\"user_id\",\"comp_cache_enabled\":true,\"fs_keys\":[{\"schema\":\"user_id\",\"col\":\"user_id\"}],\"fs_request\":{\"label\":\"user\",\"featureGroups\":[{\"label\":\"derived_2_fp32\",\"features\":[\"user__nqp\"],\"data_type\":\"DataTypeFP32\"},{\"label\":\"derived_2_string\",\"features\":[\"region\"],\"data_type\":\"DataTypeFP32\"}]},\"fs_flatten_resp_keys\":[\"user_id\"]}],\"predator_components\":[{\"component\":\"p1\",\"component_id\":\"catalog_id\",\"model_name\":\"ensemble_personalised_nqd_lgbm_v2_2\",\"model_end_point\":\"predator\",\"deadline\":110,\"batch_size\":250,\"inputs\":[{\"name\":\"input__0\",\"features\":[\"user:derived_2_fp32:user__nqp\",\"catalog:derived_fp32:sbid_value\",\"user:derived_2_string:region\"],\"shape\":[3],\"data_type\":\"BYTES\"}],\"outputs\":[{\"name\":\"output__0\",\"model_scores\":[\"pctr_score\"],\"model_scores_dims\":[[1]],\"data_type\":\"FP32\"}]}],\"numerix_components\":[{\"component\":\"i1\",\"component_id\":\"catalog_id\",\"score_col\":\"score\",\"compute_id\":\"1\",\"score_mapping\":{\"pctr@DataTypeFP32\":\"pctr_score\"},\"data_type\":\"DataTypeFP32\"}]},\"response_config\":{\"logging_perc\":1,\"model_schema_features_perc\":0,\"features\":[\"catalog_id\",\"score\",\"pctr_score\"],\"log_features\":false,\"log_batch_size\":1000}}',
    'admin@admin.com',
    1,
    'Approved',
    'Onboard',
    1,
    NOW(),
    NOW()
  );
"

# Insert numerix config and request
echo "  🔢 Inserting numerix config and request..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  INSERT INTO numerix_config (
    config_id, config_value, created_by, active, created_at, updated_at
  ) VALUES (
    1,
    '{\"expression\":\"pctr exp\"}',
    'admin@admin.com',
    1,
    NOW(),
    NOW()
  );

  INSERT INTO numerix_request (
    config_id, payload, created_by, reviewer, status, request_type, created_at, updated_at
  ) VALUES (
    1,
    '{\"expression\":\"pctr exp\"}',
    'admin@admin.com',
    'admin@admin.com',
    'Approved',
    'Onboard',
    NOW(),
    NOW()
  );
"

