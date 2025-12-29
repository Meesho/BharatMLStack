#!/bin/bash

set -e

echo "🗂️ Initializing MySQL..."

# Database and table creation
echo "  📋 Creating database and tables..."
mysql -hmysql -uroot -proot --skip-ssl -e "
  DROP DATABASE IF EXISTS testdb;
  CREATE DATABASE testdb;
  USE testdb;
  
  CREATE TABLE entity (
    request_id int NOT NULL AUTO_INCREMENT,
    payload text NOT NULL,
    entity_label varchar(255) NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255) NOT NULL,
    status varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    service varchar(255) NOT NULL,
    reject_reason varchar(255) NOT NULL,
    request_type varchar(255) DEFAULT NULL,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE feature_group (
    request_id int NOT NULL AUTO_INCREMENT,
    payload text NOT NULL,
    entity_label varchar(255) NOT NULL,
    feature_group_label varchar(255) NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255) NOT NULL,
    status varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    service varchar(255) NOT NULL,
    reject_reason varchar(255) NOT NULL,
    request_type varchar(255) DEFAULT NULL,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE features (
    request_id int NOT NULL AUTO_INCREMENT,
    payload text NOT NULL,
    entity_label varchar(255) NOT NULL,
    feature_group_label varchar(255) NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255) NOT NULL,
    status varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    request_type varchar(255) DEFAULT NULL,
    service varchar(255) NOT NULL,
    reject_reason varchar(255) NOT NULL,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE job (
    request_id int NOT NULL AUTO_INCREMENT,
    payload text NOT NULL,
    job_id varchar(255) NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255) NOT NULL,
    status varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    service varchar(255) NOT NULL,
    reject_reason varchar(255) NOT NULL,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE store (
    request_id int NOT NULL AUTO_INCREMENT,
    payload text NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255) NOT NULL,
    status varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    service varchar(255) NOT NULL,
    reject_reason varchar(255) NOT NULL,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE users (
    id bigint unsigned NOT NULL AUTO_INCREMENT,
    first_name varchar(50) NOT NULL,
    last_name varchar(50) NOT NULL,
    email varchar(100) NOT NULL,
    password_hash varchar(255) NOT NULL,
    role varchar(10) DEFAULT 'user',
    is_active boolean DEFAULT false,
    created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY id (id),
    UNIQUE KEY email (email),
    CONSTRAINT users_chk_1 CHECK ((role in ('user','admin')))
  );
  
  CREATE TABLE user_tokens (
    id bigint unsigned NOT NULL AUTO_INCREMENT,
    user_email varchar(255) NOT NULL,
    token varchar(255) NOT NULL,
    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at timestamp NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY id (id),
    UNIQUE KEY token (token)
  );
"

# Create default admin user
echo "  👤 Creating default admin user..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  INSERT INTO users (first_name, last_name, email, password_hash, role, is_active) 
  VALUES ('admin', 'admin', 'admin@admin.com', '\$2a\$10\$kYoMds9IsbvPNhJasKHO7.fTSosfbPhSAf7ElNQJ9pIa0iWBOt97e', 'admin', true);

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
      '{\"cpuLimit\":\"7000\",\"gpu_limit\":\"1\",\"cpuRequest\":\"6500\",\"gpu_request\":\"1\",\"max_replica\":\"300\",\"memoryLimit\":\"15\",\"min_replica\":\"2\",\"cpuLimitUnit\":\"m\",\"machine_type\":\"GPU\",\"cpu_threshold\":\"70\",\"gpu_threshold\":\"60\",\"memoryRequest\":\"13\",\"cpuRequestUnit\":\"m\",\"serviceAccount\":\"sa-dsci-model-inference-servic-datascience-prd-0622.iam.gserviceaccount.com\",\"gcs_bucket_path\":\"gs://gcs-dsci-model-repository-prd/predator/ranking/search-ct-gpu/*\",\"gcs_triton_path\":\"gs://gcs-dsci-model-repository-prd/scripts/launch_triton_server.py\",\"memoryLimitUnit\":\"G\",\"triton_image_tag\":\"gpu-py-ensem-trt-onnx-pytorch-fil-ensem-cache-met-v2.48.0\",\"memoryRequestUnit\":\"G\",\"nodeSelectorValue\":\"mlp-g2-standard-8\",\"deploymentStrategy\":\"rollingUpdate\"}',
      'https://grafana-prd.gcp.in/d/a2605923-52c4-4834-bdae-97570966b765/model-inference-service?orgId=1&var-service=predator-search-ct-gpu&var-query0=',
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
      'inferflow-service-search-organic',
      'inferflow-service-search-organic.prd.example.int',
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
      '{\"inputs\":[{\"dims\":[3],\"name\":\"input__0\",\"features\":[\"ONLINE_FEATURE|user:derived_2_fp32:user__nqp\",\"ONLINE_FEATURE|catalog:derived_fp32:nqp\",\"ONLINE_FEATURE|user:derived_2_string:region\"],\"data_type\":\"BYTES\"}],\"backend\":\"\",\"outputs\":[{\"dims\":[1],\"name\":\"output__0\",\"data_type\":\"FP32\"}],\"platform\":\"ensemble\",\"batch_size\":250,\"instance_type\":\"\",\"instance_count\":0,\"ensemble_scheduling\":{\"step\":[{\"input_map\":{\"INPUT__0\":\"input__0\"},\"model_name\":\"preprocessing_personalised_nqd_lgbm_v2\",\"output_map\":{\"OUTPUT__0\":\"OUTPUT__0\"},\"model_version\":1},{\"input_map\":{\"input__0\":\"OUTPUT__0\"},\"model_name\":\"personalised_nqd_lgbm_v2\",\"output_map\":{\"output__0\":\"output__0\"},\"model_version\":1}]},\"dynamic_batching_enabled\":false}',
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
    '{\"entity-label\":\"catalog\",\"fg-label\":\"derived_fp32\",\"job-id\":\"\",\"store-id\":1,\"ttl-in-seconds\":86400,\"in-memory-cache-enabled\":false,\"distributed-cache-enabled\":false,\"data-type\":\"FP32\",\"features\":{\"nqp\":{\"labels\":\"nqp\",\"feature-meta\":{\"nqp\":{\"sequence\":0,\"default-value\":\"\",\"string-length\":0,\"vector-length\":0}}}},\"layout-version\":1}',
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
      '{\"labels\":\"nqp\",\"default-values\":\"0.0\",\"source-base-path\":\"\",\"source-data-column\":\"\",\"storage-provider\":\"\",\"string-length\":\"0\",\"vector-length\":\"0\"}',
      'admin@admin.com','admin@admin.com','Approved','Onboard','ONLINE_FEATURE_STORE',''
  ), (
      'user',
      'derived_2_string',
      '{\"labels\":\"region\",\"default-values\":\"\",\"source-base-path\":\"\",\"source-data-column\":\"\",\"storage-provider\":\"\",\"string-length\":\"100\",\"vector-length\":\"0\"}',
      'admin@admin.com','admin@admin.com','Approved','Onboard','ONLINE_FEATURE_STORE',''
  );
"

# Verify initialization
echo "  🔍 Verifying MySQL initialization..."
ADMIN_COUNT=$(mysql -hmysql -uroot -proot --skip-ssl testdb -sN -e "SELECT COUNT(*) FROM users WHERE email='admin@admin.com';")
if [ "$ADMIN_COUNT" -eq 1 ]; then
  echo "  ✅ MySQL database and admin user created successfully"
else
  echo "  ❌ Failed to create admin user"
  exit 1
fi 