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
  
  CREATE TABLE feature_group (
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
  
  CREATE TABLE features (
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
  
  CREATE TABLE job (
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
  
  CREATE TABLE store (
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
  
  CREATE TABLE api_resolvers (
    id int unsigned NOT NULL AUTO_INCREMENT,
    method varchar(255) NOT NULL,
    api_path varchar(255) NOT NULL,
    resolver_fn varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
  );
  
  CREATE TABLE application_config (
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
  
  CREATE TABLE circuit_breakers (
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
  
  CREATE TABLE service_connection_config (
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
  
  CREATE TABLE group_id_counter (
    id bigint NOT NULL AUTO_INCREMENT,
    counter bigint NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
  );
  
  CREATE TABLE deployable_metadata (
    id int NOT NULL,
    \`key\` varchar(255) NOT NULL,
    value varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    active boolean DEFAULT true,
    PRIMARY KEY (id)
  );
  
  CREATE TABLE discovery_config (
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
  
  CREATE TABLE inferflow_config (
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
    PRIMARY KEY (id),
    UNIQUE KEY config_id (config_id)
  );
  
  CREATE TABLE inferflow_request (
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
  
  CREATE TABLE numerix_binary_ops (
    id int unsigned NOT NULL AUTO_INCREMENT,
    operator varchar(255) NOT NULL,
    precedence int unsigned NOT NULL,
    created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
  );
  
  CREATE TABLE numerix_config (
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
  
  CREATE TABLE numerix_request (
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
  
  CREATE TABLE numerix_unary_ops (
    id int unsigned NOT NULL AUTO_INCREMENT,
    operator varchar(255) NOT NULL,
    parameters int unsigned NOT NULL,
    created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
  );
  
  CREATE TABLE predator_config (
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
    PRIMARY KEY (id)
  );
  
  CREATE TABLE predator_requests (
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
  
  CREATE TABLE role_permission (
    id int unsigned NOT NULL AUTO_INCREMENT,
    role varchar(255) NOT NULL,
    service varchar(255) NOT NULL,
    screen_type varchar(255) NOT NULL,
    module varchar(255) NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
  );
  
  CREATE TABLE schedule_job (
    entry_date date NOT NULL,
    created_at datetime DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (entry_date)
  );
  
  CREATE TABLE service_config (
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
  
  CREATE TABLE service_deployable_config (
    id int NOT NULL,
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
    PRIMARY KEY (id),
    UNIQUE KEY host (host)
  );
  
  CREATE TABLE validation_jobs (
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
  
  CREATE TABLE validation_locks (
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

# Create default admin user
echo "  👤 Creating default admin user..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  INSERT INTO users (first_name, last_name, email, password_hash, role, is_active) 
  VALUES ('admin', 'admin', 'admin@admin.com', '\$2a\$10\$kYoMds9IsbvPNhJasKHO7.fTSosfbPhSAf7ElNQJ9pIa0iWBOt97e', 'admin', 1);

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