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
    password_hash varchar(255) DEFAULT NULL,
    role varchar(20) DEFAULT 'user',
    is_active boolean DEFAULT true,
    auth_provider enum('password', 'google', 'both') DEFAULT 'password',
    google_id varchar(255) DEFAULT NULL,
    profile_picture_url varchar(500) DEFAULT NULL,
    email_verified boolean DEFAULT false,
    last_login_at timestamp NULL DEFAULT NULL,
    created_by bigint unsigned DEFAULT NULL,
    updated_by bigint unsigned DEFAULT NULL,
    created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY id (id),
    UNIQUE KEY email (email),
    UNIQUE KEY idx_google_id (google_id),
    KEY idx_auth_provider (auth_provider),
    KEY idx_email (email),
    CONSTRAINT users_chk_1 CHECK ((role in ('user','admin','super_admin'))),
    CONSTRAINT fk_users_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_users_updated_by FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
  );
  
  CREATE TABLE user_tokens (
    id bigint unsigned NOT NULL AUTO_INCREMENT,
    user_email varchar(255) NOT NULL,
    token varchar(255) NOT NULL,
    refresh_token varchar(255) DEFAULT NULL,
    token_type enum('access', 'refresh') DEFAULT 'access',
    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at timestamp NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY id (id),
    UNIQUE KEY token (token),
    KEY idx_refresh_token (refresh_token),
    KEY idx_user_email_token_type (user_email, token_type)
  );
  
  -- Metadata tables for permission system
  CREATE TABLE services (
    id int unsigned NOT NULL AUTO_INCREMENT,
    name varchar(255) NOT NULL,
    display_name varchar(255) NOT NULL,
    description text,
    is_active boolean DEFAULT true,
    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by bigint unsigned,
    updated_by bigint unsigned,
    PRIMARY KEY (id),
    UNIQUE KEY unique_name (name),
    KEY idx_is_active (is_active),
    CONSTRAINT fk_services_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_services_updated_by FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
  );
  
  CREATE TABLE screen_types (
    id int unsigned NOT NULL AUTO_INCREMENT,
    service_id int unsigned NOT NULL,
    name varchar(255) NOT NULL,
    display_name varchar(255) NOT NULL,
    description text,
    is_active boolean DEFAULT true,
    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by bigint unsigned,
    updated_by bigint unsigned,
    PRIMARY KEY (id),
    UNIQUE KEY unique_service_screen (service_id, name),
    KEY idx_service_active (service_id, is_active),
    CONSTRAINT fk_screen_types_service FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE,
    CONSTRAINT fk_screen_types_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_screen_types_updated_by FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
  );
  
  CREATE TABLE actions (
    id int unsigned NOT NULL AUTO_INCREMENT,
    name varchar(255) NOT NULL,
    display_name varchar(255) NOT NULL,
    category varchar(50),
    description text,
    is_active boolean DEFAULT true,
    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by bigint unsigned,
    updated_by bigint unsigned,
    PRIMARY KEY (id),
    UNIQUE KEY unique_name (name),
    KEY idx_category (category),
    KEY idx_is_active (is_active),
    CONSTRAINT fk_actions_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_actions_updated_by FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
  );
  
  CREATE TABLE permissions (
    id bigint unsigned NOT NULL AUTO_INCREMENT,
    role enum('super_admin', 'admin', 'user') NOT NULL,
    service_id int unsigned NOT NULL,
    screen_type_id int unsigned NOT NULL,
    allowed_actions json NOT NULL,
    created_by bigint unsigned NOT NULL,
    updated_by bigint unsigned NOT NULL,
    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY unique_role_service_screen (role, service_id, screen_type_id),
    KEY idx_role (role),
    KEY idx_service_screen (service_id, screen_type_id),
    CONSTRAINT fk_permissions_service FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE,
    CONSTRAINT fk_permissions_screen_type FOREIGN KEY (screen_type_id) REFERENCES screen_types(id) ON DELETE CASCADE,
    CONSTRAINT fk_permissions_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_permissions_updated_by FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE RESTRICT
  );
"

# Create default admin user
echo "  👤 Creating default admin user..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  INSERT INTO users (first_name, last_name, email, password_hash, role, is_active, auth_provider) 
  VALUES ('admin', 'admin', 'admin@admin.com', '\$2a\$10\$kYoMds9IsbvPNhJasKHO7.fTSosfbPhSAf7ElNQJ9pIa0iWBOt97e', 'super_admin', true, 'password');
"

# Get the admin user ID for created_by/updated_by fields
ADMIN_ID=$(mysql -hmysql -uroot -proot --skip-ssl testdb -sN -e "SELECT id FROM users WHERE email='admin@admin.com';")

# Step 1: Insert services
echo "  📦 Creating services metadata..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  INSERT INTO services (name, display_name, description, created_by, updated_by) VALUES
  ('predator', 'Predator', 'Model registry and deployment management service', $ADMIN_ID, $ADMIN_ID),
  ('inferflow', 'InferFlow', 'Model proxy and inference flow management service', $ADMIN_ID, $ADMIN_ID),
  ('numerix', 'Numerix', 'Matrix operations and numerical computing service', $ADMIN_ID, $ADMIN_ID),
  ('embedding_platform', 'Embedding Platform', 'Vector database and embedding management platform', $ADMIN_ID, $ADMIN_ID),
  ('online_feature_store', 'Online Feature Store', 'High-performance feature serving platform', $ADMIN_ID, $ADMIN_ID);
"

# Step 2: Insert screen types for each service
echo "  📺 Creating screen types metadata..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  -- Predator screen types
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'deployable', 'Deployable', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'predator';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'model', 'Model Management', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'predator';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'model-approval', 'Model Approval', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'predator';
  
  -- InferFlow screen types
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'deployable', 'Deployable', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'inferflow';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'connection-config', 'Connection Configuration', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'inferflow';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'mp-config', 'InferFlow Configuration', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'inferflow';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'mp-config-approval', 'Configuration Approval', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'inferflow';
  
  -- Numerix screen types
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'numerix-config', 'Numerix Configuration', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'numerix';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'numerix-config-approval', 'Configuration Approval', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'numerix';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'numerix-config-testing', 'Numerix Configuration Testing', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'numerix';
  
  -- Embedding Platform screen types
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'store-discovery', 'Store Discovery', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'entity-discovery', 'Entity Discovery', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'model-discovery', 'Model Discovery', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'variant-discovery', 'Variant Discovery', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'filter-discovery', 'Filter Discovery', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'job-frequency-discovery', 'Job Frequency Discovery', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'store-registry', 'Store Registry', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'entity-registry', 'Entity Registry', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'model-registry', 'Model Registry', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'variant-registry', 'Variant Registry', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'filter-registry', 'Filter Registry', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'job-frequency-registry', 'Job Frequency Registry', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'store-approval', 'Store Approval', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'entity-approval', 'Entity Approval', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'model-approval', 'Model Approval', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'variant-approval', 'Variant Approval', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'filter-approval', 'Filter Approval', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'job-frequency-approval', 'Job Frequency Approval', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'deployment-operations', 'Deployment Operations', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'onboard-variant-to-db', 'Onboard Variant to DB', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'onboard-variant-approval', 'Onboard Variant Approval', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'embedding_platform';
  
  -- Online Feature Store screen types
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'feature-discovery', 'Feature Discovery', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'online_feature_store';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'store-discovery', 'Store Discovery', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'online_feature_store';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'job-discovery', 'Job Discovery', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'online_feature_store';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'client-discovery', 'Client Discovery', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'online_feature_store';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'store-registry', 'Store Registry', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'online_feature_store';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'entity-registry', 'Entity Registry', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'online_feature_store';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'feature-group-registry', 'Feature Group Registry', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'online_feature_store';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'feature-registry', 'Feature Registry', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'online_feature_store';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'job-registry', 'Job Registry', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'online_feature_store';
  INSERT INTO screen_types (service_id, name, display_name, created_by, updated_by) 
  SELECT id, 'feature-approval', 'Feature Approval', $ADMIN_ID, $ADMIN_ID FROM services WHERE name = 'online_feature_store';
"

# Step 3: Insert actions
echo "  ⚡ Creating actions metadata..."
mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  INSERT INTO actions (name, display_name, category, created_by, updated_by) VALUES
  ('view', 'View', 'crud', $ADMIN_ID, $ADMIN_ID),
  ('edit', 'Edit', 'crud', $ADMIN_ID, $ADMIN_ID),
  ('onboard', 'Create/Onboard', 'crud', $ADMIN_ID, $ADMIN_ID),
  ('delete', 'Delete', 'crud', $ADMIN_ID, $ADMIN_ID),
  ('clone', 'Clone', 'management', $ADMIN_ID, $ADMIN_ID),
  ('upload', 'Upload', 'management', $ADMIN_ID, $ADMIN_ID),
  ('upload_edit', 'Upload Edit', 'management', $ADMIN_ID, $ADMIN_ID),
  ('upload_partial', 'Upload Partial', 'management', $ADMIN_ID, $ADMIN_ID),
  ('promote', 'Promote', 'management', $ADMIN_ID, $ADMIN_ID),
  ('scale_up', 'Scale Up', 'management', $ADMIN_ID, $ADMIN_ID),
  ('validate', 'Validate', 'approval', $ADMIN_ID, $ADMIN_ID),
  ('approve', 'Approve', 'approval', $ADMIN_ID, $ADMIN_ID),
  ('reject', 'Reject', 'approval', $ADMIN_ID, $ADMIN_ID),
  ('cancel', 'Cancel', 'approval', $ADMIN_ID, $ADMIN_ID),
  ('test', 'Test', 'testing', $ADMIN_ID, $ADMIN_ID),
  ('load_test', 'Load Test', 'testing', $ADMIN_ID, $ADMIN_ID),
  ('deactivate', 'Deactivate', 'management', $ADMIN_ID, $ADMIN_ID);
"

# Step 4: Create default permissions for super_admin role using IDs
echo "  🔐 Creating default permissions for super_admin role..."
# Get all action IDs as JSON array
ALL_ACTION_IDS=$(mysql -hmysql -uroot -proot --skip-ssl testdb -sN -e "SELECT JSON_ARRAYAGG(id) FROM actions;")

mysql -hmysql -uroot -proot --skip-ssl testdb -e "
  -- Predator permissions
  INSERT INTO permissions (role, service_id, screen_type_id, allowed_actions, created_by, updated_by)
  SELECT 'super_admin', s.id, st.id, '$ALL_ACTION_IDS', $ADMIN_ID, $ADMIN_ID
  FROM services s
  CROSS JOIN screen_types st
  WHERE s.name = 'predator' AND st.service_id = s.id;
  
  -- InferFlow permissions
  INSERT INTO permissions (role, service_id, screen_type_id, allowed_actions, created_by, updated_by)
  SELECT 'super_admin', s.id, st.id, '$ALL_ACTION_IDS', $ADMIN_ID, $ADMIN_ID
  FROM services s
  CROSS JOIN screen_types st
  WHERE s.name = 'inferflow' AND st.service_id = s.id;
  
  -- Numerix permissions
  INSERT INTO permissions (role, service_id, screen_type_id, allowed_actions, created_by, updated_by)
  SELECT 'super_admin', s.id, st.id, '$ALL_ACTION_IDS', $ADMIN_ID, $ADMIN_ID
  FROM services s
  CROSS JOIN screen_types st
  WHERE s.name = 'numerix' AND st.service_id = s.id;
  
  -- Embedding Platform permissions
  INSERT INTO permissions (role, service_id, screen_type_id, allowed_actions, created_by, updated_by)
  SELECT 'super_admin', s.id, st.id, '$ALL_ACTION_IDS', $ADMIN_ID, $ADMIN_ID
  FROM services s
  CROSS JOIN screen_types st
  WHERE s.name = 'embedding_platform' AND st.service_id = s.id;
  
  -- Online Feature Store permissions (if screen types exist)
  INSERT INTO permissions (role, service_id, screen_type_id, allowed_actions, created_by, updated_by)
  SELECT 'super_admin', s.id, st.id, '$ALL_ACTION_IDS', $ADMIN_ID, $ADMIN_ID
  FROM services s
  CROSS JOIN screen_types st
  WHERE s.name = 'online_feature_store' AND st.service_id = s.id;
"

# Verify initialization
echo "  🔍 Verifying MySQL initialization..."
ADMIN_COUNT=$(mysql -hmysql -uroot -proot --skip-ssl testdb -sN -e "SELECT COUNT(*) FROM users WHERE email='admin@admin.com';")
SERVICES_COUNT=$(mysql -hmysql -uroot -proot --skip-ssl testdb -sN -e "SELECT COUNT(*) FROM services;")
SCREEN_TYPES_COUNT=$(mysql -hmysql -uroot -proot --skip-ssl testdb -sN -e "SELECT COUNT(*) FROM screen_types;")
ACTIONS_COUNT=$(mysql -hmysql -uroot -proot --skip-ssl testdb -sN -e "SELECT COUNT(*) FROM actions;")
PERMISSIONS_COUNT=$(mysql -hmysql -uroot -proot --skip-ssl testdb -sN -e "SELECT COUNT(*) FROM permissions WHERE role='super_admin';")
if [ "$ADMIN_COUNT" -eq 1 ]; then
  echo "  ✅ MySQL database and admin user created successfully"
  echo "  ✅ Created $SERVICES_COUNT services"
  echo "  ✅ Created $SCREEN_TYPES_COUNT screen types"
  echo "  ✅ Created $ACTIONS_COUNT actions"
  if [ "$PERMISSIONS_COUNT" -gt 0 ]; then
    echo "  ✅ Created $PERMISSIONS_COUNT default permissions for super_admin role"
  else
    echo "  ⚠️  Warning: No permissions created for super_admin role"
  fi
else
  echo "  ❌ Failed to create admin user"
  exit 1
fi 