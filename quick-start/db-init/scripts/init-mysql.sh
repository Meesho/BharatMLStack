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

# Create default admin user
echo "  👤 Creating default admin user..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  INSERT INTO users (first_name, last_name, email, password_hash, role, is_active) 
  VALUES ('admin', 'admin', 'admin@admin.com', '\$2a\$10\$kYoMds9IsbvPNhJasKHO7.fTSosfbPhSAf7ElNQJ9pIa0iWBOt97e', 'admin', 1);
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