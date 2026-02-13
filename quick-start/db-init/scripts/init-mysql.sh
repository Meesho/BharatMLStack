#!/bin/bash

set -e

echo "🗂️ Initializing MySQL..."

# Database and table creation
echo "  📋 Creating database and tables..."
mysql -hmysql -uroot -proot --skip-ssl -e "
  CREATE DATABASE IF NOT EXISTS testdb;
  USE testdb;
  
  CREATE TABLE IF NOT EXISTS entity (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    payload text NOT NULL,
    entity_label varchar(255) NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255) NOT NULL,
    status varchar(255) NOT NULL,
    request_type varchar(255) NOT NULL,
    service varchar(255) NOT NULL,
    reject_reason varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE IF NOT EXISTS feature_group (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    entity_label varchar(255) NOT NULL,
    feature_group_label varchar(255) NOT NULL,
    payload json NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255) NOT NULL,
    status varchar(255) NOT NULL,
    request_type varchar(255) NOT NULL,
    service varchar(255) NOT NULL,
    reject_reason varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE IF NOT EXISTS features (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    entity_label varchar(255) NOT NULL,
    feature_group_label varchar(255) NOT NULL,
    payload json NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255) NOT NULL,
    status varchar(255) NOT NULL,
    request_type varchar(255) NOT NULL,
    service varchar(255) NOT NULL,
    reject_reason varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE IF NOT EXISTS job (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    job_id varchar(255) NOT NULL,
    payload text NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255) NOT NULL,
    status varchar(255) NOT NULL,
    service varchar(255) NOT NULL,
    reject_reason varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE IF NOT EXISTS store (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    payload text NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255) NOT NULL,
    status varchar(255) NOT NULL,
    service varchar(255) NOT NULL,
    reject_reason varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );
  
  -- Skye/Embedding Platform request tables
  CREATE TABLE IF NOT EXISTS store_requests (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    reason text NOT NULL,
    payload text NOT NULL,
    request_type text NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255),
    status varchar(255) NOT NULL DEFAULT 'PENDING',
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE IF NOT EXISTS entity_requests (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    reason text NOT NULL,
    payload text NOT NULL,
    request_type text NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255),
    status varchar(255) NOT NULL DEFAULT 'PENDING',
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE IF NOT EXISTS model_requests (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    reason text NOT NULL,
    payload text NOT NULL,
    request_type text NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255),
    status varchar(255) NOT NULL DEFAULT 'PENDING',
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE IF NOT EXISTS variant_requests (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    reason text NOT NULL,
    payload text NOT NULL,
    request_type text NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255),
    status varchar(255) NOT NULL DEFAULT 'PENDING',
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );

  CREATE TABLE IF NOT EXISTS variant_onboarding_tasks (
    task_id INT AUTO_INCREMENT PRIMARY KEY,
    entity VARCHAR(255) NOT NULL,
    model VARCHAR(255) NOT NULL,
    variant VARCHAR(255) NOT NULL,
    payload TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
  );
  
  CREATE TABLE IF NOT EXISTS filter_requests (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    reason text NOT NULL,
    payload text NOT NULL,
    request_type text NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255),
    status varchar(255) NOT NULL DEFAULT 'PENDING',
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE IF NOT EXISTS job_frequency_requests (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    reason text NOT NULL,
    payload text NOT NULL,
    request_type text NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255),
    status varchar(255) NOT NULL DEFAULT 'PENDING',
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE IF NOT EXISTS qdrant_cluster_requests (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    reason text NOT NULL,
    payload text NOT NULL,
    request_type text NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255),
    status varchar(255) NOT NULL DEFAULT 'PENDING',
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE IF NOT EXISTS variant_promotion_requests (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    reason text NOT NULL,
    payload text NOT NULL,
    request_type text NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255),
    status varchar(255) NOT NULL DEFAULT 'PENDING',
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE IF NOT EXISTS variant_onboarding_requests (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    reason text NOT NULL,
    payload text NOT NULL,
    request_type text NOT NULL,
    created_by varchar(255) NOT NULL,
    approved_by varchar(255),
    status varchar(255) NOT NULL DEFAULT 'PENDING',
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE IF NOT EXISTS users (
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
  
  CREATE TABLE IF NOT EXISTS user_tokens (
    id bigint unsigned NOT NULL AUTO_INCREMENT,
    user_email varchar(255) NOT NULL,
    token varchar(255) NOT NULL,
    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at timestamp NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY id (id),
    UNIQUE KEY token (token)
  );
  
  CREATE TABLE IF NOT EXISTS api_resolvers (
    id int unsigned NOT NULL AUTO_INCREMENT,
    method varchar(255) NOT NULL,
    api_path varchar(255) NOT NULL,
    resolver_fn varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
  );
  
  CREATE TABLE IF NOT EXISTS application_config (
    app_token varchar(255) NOT NULL,
    bu varchar(255) NOT NULL,
    team varchar(255) NOT NULL,
    service varchar(255) NOT NULL,
    active boolean NOT NULL,
    created_by varchar(255) NOT NULL,
    updated_by varchar(255),
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (app_token)
  );
  
  CREATE TABLE IF NOT EXISTS circuit_breakers (
    id int NOT NULL,
    name varchar(255) NOT NULL,
    enabled boolean DEFAULT true,
    slow_call_rate_threshold int,
    failure_rate_threshold int,
    sliding_window_type varchar(255) NOT NULL,
    sliding_window_size int NOT NULL,
    minimum_no_of_calls int,
    auto_transition_open_to_half boolean DEFAULT true,
    max_wait_half_open_state int,
    wait_open_state int,
    slow_call_duration int,
    permitted_calls_half_open int,
    bh_name int,
    bh_core_pool_size int,
    bh_max_pool_size int,
    bh_queue_size int,
    bh_fallback_core_size int,
    bh_keep_alive_ms int,
    created_by varchar(255),
    updated_by varchar(255),
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY name (name),
    UNIQUE KEY bh_name (bh_name)
  );
  
  CREATE TABLE IF NOT EXISTS service_connection_config (
    id int unsigned NOT NULL AUTO_INCREMENT,
    \`default\` boolean NOT NULL,
    service varchar(255) NOT NULL,
    conn_protocol varchar(255) NOT NULL,
    http_config json NOT NULL,
    grpc_config json NOT NULL,
    active boolean NOT NULL,
    created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by varchar(255) NOT NULL,
    updated_by varchar(255),
    PRIMARY KEY (id)
  );
  
  CREATE TABLE IF NOT EXISTS group_id_counter (
    id bigint NOT NULL AUTO_INCREMENT,
    counter bigint NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
  );
  
  CREATE TABLE IF NOT EXISTS deployable_metadata (
    id int NOT NULL,
    \`key\` varchar(255) NOT NULL,
    value varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    active boolean DEFAULT true,
    PRIMARY KEY (id)
  );
  
  CREATE TABLE IF NOT EXISTS discovery_config (
    id int NOT NULL AUTO_INCREMENT,
    app_token varchar(255),
    service_deployable_id int,
    circuit_breaker_id int,
    service_connection_id int,
    route_service_connection_id int,
    route_service_deployable_id int,
    route_percent int,
    active boolean DEFAULT true,
    default_response varchar(5000),
    default_response_percent int,
    default_response_enabled_forCB boolean DEFAULT false,
    created_by varchar(255),
    updated_by varchar(255),
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
  );
  
  CREATE TABLE IF NOT EXISTS inferflow_config (
    id int unsigned NOT NULL AUTO_INCREMENT,
    discovery_id int NOT NULL,
    config_id varchar(255) NOT NULL,
    config_value json NOT NULL,
    default_response json,
    created_by varchar(255) NOT NULL,
    updated_by varchar(255),
    active boolean NOT NULL,
    created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    test_results json,
    source_config_id varchar(255) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY config_id (config_id)
  );
  
  CREATE TABLE IF NOT EXISTS inferflow_request (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    config_id varchar(255) NOT NULL,
    payload json NOT NULL,
    created_by varchar(255) NOT NULL,
    version int NOT NULL DEFAULT 1,
    updated_by varchar(255),
    reviewer varchar(255),
    status varchar(255) NOT NULL,
    request_type varchar(255) NOT NULL,
    reject_reason varchar(255),
    active boolean NOT NULL,
    created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE IF NOT EXISTS numerix_binary_ops (
    id int unsigned NOT NULL AUTO_INCREMENT,
    operator varchar(255) NOT NULL,
    precedence int unsigned NOT NULL,
    created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
  );
  
  CREATE TABLE IF NOT EXISTS numerix_config (
    id int unsigned NOT NULL AUTO_INCREMENT,
    config_id int unsigned NOT NULL,
    config_value json NOT NULL,
    created_by varchar(255) NOT NULL,
    updated_by varchar(255),
    active boolean NOT NULL,
    created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    test_results json,
    PRIMARY KEY (id),
    UNIQUE KEY config_id (config_id)
  );
  
  CREATE TABLE IF NOT EXISTS numerix_request (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    config_id int unsigned NOT NULL,
    created_by varchar(255) NOT NULL,
    updated_by varchar(255),
    reviewer varchar(255) NOT NULL,
    status varchar(255) NOT NULL,
    request_type varchar(255) NOT NULL,
    reject_reason varchar(255),
    payload json NOT NULL,
    created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE IF NOT EXISTS numerix_unary_ops (
    id int unsigned NOT NULL AUTO_INCREMENT,
    operator varchar(255) NOT NULL,
    parameters int unsigned NOT NULL,
    created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
  );
  
  CREATE TABLE IF NOT EXISTS predator_config (
    id int NOT NULL AUTO_INCREMENT,
    discovery_config_id int NOT NULL,
    model_name varchar(255) NOT NULL,
    meta_data json,
    active boolean DEFAULT true,
    created_by varchar(255),
    updated_by varchar(255),
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    test_results json,
    has_nil_data boolean DEFAULT false,
    source_model_name varchar(255) NULL,
    PRIMARY KEY (id)
  );
  
  CREATE TABLE IF NOT EXISTS predator_requests (
    request_id int unsigned NOT NULL AUTO_INCREMENT,
    group_id int unsigned NOT NULL,
    model_name varchar(255) NOT NULL,
    payload json NOT NULL,
    created_by varchar(255) NOT NULL,
    updated_by varchar(255),
    reviewer varchar(255),
    request_stage enum('Clone To Bucket','Restart Deployable','DB Population','Pending','Request Payload Error') DEFAULT 'Pending',
    request_type enum('Onboard','ScaleUp','Promote','Delete','Edit') NOT NULL,
    status enum('Pending Approval','Approved','Rejected','Cancelled','Failed','In Progress','') DEFAULT 'Pending Approval',
    active boolean DEFAULT true,
    reject_reason varchar(255),
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_valid boolean DEFAULT false,
    PRIMARY KEY (request_id)
  );
  
  CREATE TABLE IF NOT EXISTS role_permission (
    id int unsigned NOT NULL AUTO_INCREMENT,
    role varchar(255) NOT NULL,
    service varchar(255) NOT NULL,
    screen_type varchar(255) NOT NULL,
    module varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
  );
  
  CREATE TABLE IF NOT EXISTS schedule_job (
    entry_date date NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (entry_date)
  );
  
  CREATE TABLE IF NOT EXISTS service_config (
    id int NOT NULL,
    service_name varchar(255) NOT NULL,
    primary_owner varchar(255),
    secondary_owner varchar(255),
    repo_name varchar(255),
    branch_name varchar(255),
    health_check varchar(255),
    app_port int,
    team varchar(255),
    bu varchar(255),
    priority_v2 varchar(255),
    module varchar(255),
    app_type varchar(255),
    ingress_class varchar(255),
    build_no varchar(255),
    config json,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
  );
  
  CREATE TABLE IF NOT EXISTS service_deployable_config (
    id int NOT NULL AUTO_INCREMENT,
    name varchar(255),
    host varchar(255) NOT NULL,
    service enum('inferflow', 'predator', 'numerix'),
    active boolean DEFAULT false,
    created_by varchar(255),
    updated_by varchar(255),
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    config json,
    monitoring_url varchar(255) DEFAULT NULL,
    deployable_running_status boolean,
    deployable_work_flow_id varchar(255),
    deployment_run_id varchar(255),
    deployable_health enum('DEPLOYMENT_REASON_ARGO_APP_HEALTH_DEGRADED', 'DEPLOYMENT_REASON_ARGO_APP_HEALTHY'),
    work_flow_status enum('WORKFLOW_COMPLETED','WORKFLOW_NOT_FOUND','WORKFLOW_RUNNING','WORKFLOW_FAILED','WORKFLOW_NOT_STARTED'),
    override_testing TINYINT(1) DEFAULT 0,
    deployable_tag varchar(255) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY host (host)
  );
  
  CREATE TABLE IF NOT EXISTS validation_jobs (
    id int unsigned NOT NULL AUTO_INCREMENT,
    group_id varchar(255) NOT NULL,
    lock_id int unsigned NOT NULL,
    test_deployable_id int NOT NULL,
    service_name varchar(255) NOT NULL,
    status varchar(50) NOT NULL,
    validation_result boolean DEFAULT NULL,
    error_message text,
    restarted_at datetime DEFAULT NULL,
    last_health_check datetime DEFAULT NULL,
    health_check_count int DEFAULT 0,
    max_health_checks int DEFAULT 10,
    health_check_interval int DEFAULT 30,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_group_id (group_id),
    KEY idx_lock_id (lock_id),
    KEY idx_status (status)
  );
  
  CREATE TABLE IF NOT EXISTS validation_locks (
    id int unsigned NOT NULL AUTO_INCREMENT,
    lock_key varchar(255) NOT NULL,
    locked_by varchar(255) NOT NULL,
    locked_at datetime NOT NULL,
    expires_at datetime NOT NULL,
    group_id varchar(255),
    description text,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY lock_key (lock_key),
    KEY idx_expires_at (expires_at)
  );
"

echo "  📦 Inserting deployable metadata..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  INSERT IGNORE INTO deployable_metadata (id,\`key\`, value, active, created_at, updated_at) VALUES
    (1, 'node_selectors', 'bharatml-stack-control-plane', 1, NOW(), NOW()),
    (3, 'gcs_base_path', 'NA', 1, NOW(), NOW()),
    (4, 'gcs_base_path', 'gs://gcs-dsci-model-repository-int/dummy-tmp/*', 1, NOW(), NOW()),
    (6, 'triton_image_tags', '25.06-py3', 1, NOW(), NOW()),
    (7, 'gcs_triton_path', 'NA', 1, NOW(), NOW()),
    (8, 'gcs_triton_path', 'some_path', 1, NOW(), NOW()),
    (9, 'service_account', 'NA', 1, NOW(), NOW()),
    (10, 'service_account', 'sa-dsci-model-inference-svc@meesho-shared-int-0525.iam.gserviceaccount.com', 1, NOW(), NOW())
  ON DUPLICATE KEY UPDATE 
    value = VALUES(value),
    updated_at = NOW();
"

# Create default admin user
echo "  👤 Creating default admin user..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  INSERT IGNORE INTO users (first_name, last_name, email, password_hash, role, is_active) 
  VALUES ('admin', 'admin', 'admin@admin.com', '\$2a\$10\$kYoMds9IsbvPNhJasKHO7.fTSosfbPhSAf7ElNQJ9pIa0iWBOt97e', 'admin', true);

  # INSERT IGNORE INTO group_id_counter (id, counter, created_at, updated_at) 
  # VALUES (1, 1, NOW(), NOW());

  INSERT IGNORE INTO service_deployable_config (
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

  INSERT IGNORE INTO service_deployable_config (
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

  INSERT IGNORE INTO discovery_config (
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

  INSERT IGNORE INTO discovery_config (
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

  INSERT IGNORE INTO service_connection_config (
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

  INSERT IGNORE INTO group_id_counter (id, counter, created_at, updated_at) 
  VALUES (1, 1, NOW(), NOW());
"

# Insert API resolvers
echo "  🔌 Inserting API resolvers..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  -- Predator APIs
  INSERT IGNORE INTO api_resolvers (method, api_path, resolver_fn) VALUES
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
  ('PUT', '/api/v1/horizon/deployable-registry/deployables/tune-thresholds', 'DeployableTuneResolver'),

  -- Skye/Embedding Platform APIs
  -- Store APIs
  ('POST', '/api/v1/horizon/skye/requests/store/register', 'SkyeStoreRegisterResolver'),
  ('POST', '/api/v1/horizon/skye/requests/store/approve', 'SkyeStoreApproveResolver'),
  ('GET', '/api/v1/horizon/skye/data/stores', 'SkyeStoreDiscoveryResolver'),
  ('GET', '/api/v1/horizon/skye/data/store-requests', 'SkyeStoreRequestDiscoveryResolver'),

  -- Entity APIs
  ('POST', '/api/v1/horizon/skye/requests/entity/register', 'SkyeEntityRegisterResolver'),
  ('POST', '/api/v1/horizon/skye/requests/entity/approve', 'SkyeEntityApproveResolver'),
  ('GET', '/api/v1/horizon/skye/data/entities', 'SkyeEntityDiscoveryResolver'),
  ('GET', '/api/v1/horizon/skye/data/entity-requests', 'SkyeEntityRequestDiscoveryResolver'),

  -- Model APIs
  ('POST', '/api/v1/horizon/skye/requests/model/register', 'SkyeModelRegisterResolver'),
  ('POST', '/api/v1/horizon/skye/requests/model/edit', 'SkyeModelEditResolver'),
  ('POST', '/api/v1/horizon/skye/requests/model/approve', 'SkyeModelApproveResolver'),
  ('POST', '/api/v1/horizon/skye/requests/model/edit/approve', 'SkyeModelEditApproveResolver'),
  ('GET', '/api/v1/horizon/skye/data/models', 'SkyeModelDiscoveryResolver'),
  ('GET', '/api/v1/horizon/skye/data/model-requests', 'SkyeModelRequestDiscoveryResolver'),

  -- Variant APIs
  ('POST', '/api/v1/horizon/skye/requests/variant/register', 'SkyeVariantRegisterResolver'),
  ('POST', '/api/v1/horizon/skye/requests/variant/edit', 'SkyeVariantEditResolver'),
  ('POST', '/api/v1/horizon/skye/requests/variant/approve', 'SkyeVariantApproveResolver'),
  ('POST', '/api/v1/horizon/skye/requests/variant/edit/approve', 'SkyeVariantEditApproveResolver'),
  ('GET', '/api/v1/horizon/skye/data/variants', 'SkyeVariantDiscoveryResolver'),
  ('GET', '/api/v1/horizon/skye/data/variant-requests', 'SkyeVariantRequestDiscoveryResolver'),

  -- Filter APIs
  ('POST', '/api/v1/horizon/skye/requests/filter/register', 'SkyeFilterRegisterResolver'),
  ('POST', '/api/v1/horizon/skye/requests/filter/approve', 'SkyeFilterApproveResolver'),
  ('GET', '/api/v1/horizon/skye/data/filters', 'SkyeFilterDiscoveryResolver'),
  ('GET', '/api/v1/horizon/skye/data/filter-requests', 'SkyeFilterRequestDiscoveryResolver'),

  -- Job Frequency APIs
  ('POST', '/api/v1/horizon/skye/requests/job-frequency/register', 'SkyeJobFrequencyRegisterResolver'),
  ('POST', '/api/v1/horizon/skye/requests/job-frequency/approve', 'SkyeJobFrequencyApproveResolver'),
  ('GET', '/api/v1/horizon/skye/data/job-frequencies', 'SkyeJobFrequencyDiscoveryResolver'),
  ('GET', '/api/v1/horizon/skye/data/job-frequency-requests', 'SkyeJobFrequencyRequestDiscoveryResolver'),

  -- Qdrant/Deployment Operations APIs
  ('POST', '/api/v1/horizon/skye/requests/qdrant/create-cluster', 'SkyeQdrantCreateResolver'),
  ('POST', '/api/v1/horizon/skye/requests/qdrant/approve', 'SkyeQdrantApproveResolver'),
  ('GET', '/api/v1/horizon/skye/data/qdrant/clusters', 'SkyeQdrantDiscoveryResolver'),

  -- Variant Promotion APIs
  ('POST', '/api/v1/horizon/skye/requests/variant/promote', 'SkyeVariantPromoteResolver'),
  ('POST', '/api/v1/horizon/skye/requests/variant/promote/approve', 'SkyeVariantPromoteApproveResolver'),

  -- Variant Onboarding APIs
  ('POST', '/api/v1/horizon/skye/requests/variant/onboard', 'SkyeVariantOnboardResolver'),
  ('POST', '/api/v1/horizon/skye/requests/variant/onboard/approve', 'SkyeVariantOnboardApproveResolver'),
  ('GET', '/api/v1/horizon/skye/data/variant-onboarding/requests', 'SkyeVariantOnboardRequestDiscoveryResolver'),
  ('GET', '/api/v1/horizon/skye/data/variant-onboarding/tasks', 'SkyeVariantOnboardTaskDiscoveryResolver'),

  -- Variant Scaleup APIs
  ('POST', '/api/v1/horizon/skye/requests/variant/scale-up', 'SkyeVariantScaleupResolver'),
  ('POST', '/api/v1/horizon/skye/requests/variant/scale-up/approve', 'SkyeVariantScaleupApproveResolver'),
  ('GET', '/api/v1/horizon/skye/data/variant-scaleup/requests', 'SkyeVariantScaleupRequestDiscoveryResolver'),
  ('GET', '/api/v1/horizon/skye/data/variant-scaleup/tasks', 'SkyeVariantScaleupTaskDiscoveryResolver'),

  -- MQ ID to Topics and Variants List APIs
  ('GET', '/api/v1/horizon/skye/data/mq-id-topics', 'SkyeMQIdTopicsDiscoveryResolver'),
  ('GET', '/api/v1/horizon/skye/data/variants-list', 'SkyeVariantsListDiscoveryResolver');
"

# Insert role permissions for admin role
echo "  🔐 Inserting role permissions for admin role..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  -- Deployable permissions (service: all)
  INSERT IGNORE INTO role_permission (role, service, screen_type, module) VALUES
  ('admin', 'all', 'deployable', 'onboard'),
  ('admin', 'all', 'deployable', 'edit'),
  ('admin', 'all', 'deployable', 'view'),

  -- Predator permissions
  ('admin', 'predator', 'model-approval', 'approve'),
  ('admin', 'predator', 'model-approval', 'view'),
  ('admin', 'predator', 'model-approval', 'reject'),
  ('admin', 'predator', 'model-approval', 'cancel'),
  ('admin', 'predator', 'model-approval', 'review'),
  ('admin', 'predator', 'model-approval', 'validate'),
  ('admin', 'predator', 'model', 'delete'),
  ('admin', 'predator', 'model', 'onboard'),
  ('admin', 'predator', 'model', 'view'),
  ('admin', 'predator', 'model', 'test'),
  ('admin', 'predator', 'model', 'edit'),
  ('admin', 'predator', 'model', 'upload'),
  ('admin', 'predator', 'model', 'upload_edit'),
  ('admin', 'predator', 'deployable', 'onboard'),
  ('admin', 'predator', 'deployable', 'edit'),
  ('admin', 'predator', 'deployable', 'view'),

  -- Inferflow permissions 
  ('admin', 'inferflow', 'inferflow-config-testing', 'view'),
  ('admin', 'inferflow', 'inferflow-config-testing', 'test'),
  ('admin', 'inferflow', 'deployable', 'view'),
  ('admin', 'inferflow', 'deployable', 'register'),
  ('admin', 'inferflow', 'deployable', 'edit'),
  ('admin', 'inferflow', 'deployable', 'onboard'),
  ('admin', 'inferflow', 'inferflow-config', 'view'),
  ('admin', 'inferflow', 'inferflow-config', 'approve'),
  ('admin', 'inferflow', 'inferflow-config', 'reject'),
  ('admin', 'inferflow', 'inferflow-config', 'cancel'),
  ('admin', 'inferflow', 'inferflow-config', 'clone'),
  ('admin', 'inferflow', 'inferflow-config', 'deactivate'),
  ('admin', 'inferflow', 'inferflow-config', 'fetch'),
  ('admin', 'inferflow', 'inferflow-config', 'edit'),
  ('admin', 'inferflow', 'inferflow-config', 'onboard'),
  ('admin', 'inferflow', 'inferflow-config', 'test'),
  ('admin', 'inferflow', 'inferflow-config-approval', 'validate'),
  ('admin', 'inferflow', 'inferflow-config-approval', 'approve'),
  ('admin', 'inferflow', 'inferflow-config-approval', 'reject'),
  ('admin', 'inferflow', 'inferflow-config-approval', 'cancel'),
  ('admin', 'inferflow', 'inferflow-config-approval', 'view'),
  ('admin', 'inferflow', 'inferflow-config-approval', 'review'),

  -- Numerix permissions 
  ('admin', 'numerix', 'numerix-config-approval', 'view'),
  ('admin', 'numerix', 'numerix-config-approval', 'approve'),
  ('admin', 'numerix', 'numerix-config-approval', 'reject'),
  ('admin', 'numerix', 'numerix-config-approval', 'cancel'),
  ('admin', 'numerix', 'numerix-config-approval', 'review'),
  ('admin', 'numerix', 'numerix-config-testing', 'view'),
  ('admin', 'numerix', 'numerix-config-testing', 'test'),
  ('admin', 'numerix', 'numerix-config', 'test'),
  ('admin', 'numerix', 'numerix-config', 'view'),
  ('admin', 'numerix', 'numerix-config', 'edit'),
  ('admin', 'numerix', 'numerix-config', 'onboard'),


  -- Application permissions
  ('admin', 'application', 'application-discovery-registry', 'view'),
  ('admin', 'application', 'application-discovery-registry', 'edit'),
  ('admin', 'application', 'application-discovery-registry', 'onboard'),
  ('admin', 'application', 'connection-config', 'view'),
  ('admin', 'application', 'connection-config', 'edit'),
  ('admin', 'application', 'connection-config', 'onboard'),

  -- Embedding Platform (Skye) permissions
  -- Store permissions
  ('admin', 'embedding_platform', 'store-discovery', 'view'),
  ('admin', 'embedding_platform', 'store-registry', 'onboard'),
  ('admin', 'embedding_platform', 'store-registry', 'view'),
  ('admin', 'embedding_platform', 'store-approval', 'review'),
  ('admin', 'embedding_platform', 'store-approval', 'view'),

  -- Entity permissions
  ('admin', 'embedding_platform', 'entity-discovery', 'view'),
  ('admin', 'embedding_platform', 'entity-registry', 'onboard'),
  ('admin', 'embedding_platform', 'entity-registry', 'view'),
  ('admin', 'embedding_platform', 'entity-approval', 'review'),
  ('admin', 'embedding_platform', 'entity-approval', 'view'),

  -- Model permissions
  ('admin', 'embedding_platform', 'model-discovery', 'view'),
  ('admin', 'embedding_platform', 'model-registry', 'onboard'),
  ('admin', 'embedding_platform', 'model-registry', 'edit'),
  ('admin', 'embedding_platform', 'model-registry', 'view'),
  ('admin', 'embedding_platform', 'model-approval', 'review'),
  ('admin', 'embedding_platform', 'model-approval', 'view'),

  -- Variant permissions
  ('admin', 'embedding_platform', 'variant-discovery', 'view'),
  ('admin', 'embedding_platform', 'variant-registry', 'onboard'),
  ('admin', 'embedding_platform', 'variant-registry', 'edit'),
  ('admin', 'embedding_platform', 'variant-registry', 'view'),
  ('admin', 'embedding_platform', 'variant-registry', 'promote'),
  ('admin', 'embedding_platform', 'variant-approval', 'review'),
  ('admin', 'embedding_platform', 'variant-approval', 'view'),

  -- Filter permissions
  ('admin', 'embedding_platform', 'filter-discovery', 'view'),
  ('admin', 'embedding_platform', 'filter-registry', 'onboard'),
  ('admin', 'embedding_platform', 'filter-registry', 'view'),
  ('admin', 'embedding_platform', 'filter-approval', 'review'),
  ('admin', 'embedding_platform', 'filter-approval', 'view'),

  -- Job Frequency permissions
  ('admin', 'embedding_platform', 'job-frequency-discovery', 'view'),
  ('admin', 'embedding_platform', 'job-frequency-registry', 'onboard'),
  ('admin', 'embedding_platform', 'job-frequency-registry', 'view'),
  ('admin', 'embedding_platform', 'job-frequency-approval', 'review'),
  ('admin', 'embedding_platform', 'job-frequency-approval', 'view'),

  -- Deployment Operations (Qdrant) permissions
  ('admin', 'embedding_platform', 'deployment-operations', 'onboard'),
  ('admin', 'embedding_platform', 'deployment-operations', 'view'),
  ('admin', 'embedding_platform', 'deployment-operations', 'review'),

  -- Variant Onboarding permissions
  ('admin', 'embedding_platform', 'onboard-variant-to-db', 'onboard'),
  ('admin', 'embedding_platform', 'onboard-variant-to-db', 'view'),
  ('admin', 'embedding_platform', 'onboard-variant-approval', 'review'),
  ('admin', 'embedding_platform', 'onboard-variant-approval', 'view'),

  -- MQ ID to Topics and Variants List permissions
  ('admin', 'embedding_platform', 'model-discovery', 'view'),
  ('admin', 'embedding_platform', 'variant-discovery', 'view');
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