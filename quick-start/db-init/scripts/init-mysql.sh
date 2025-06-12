#!/bin/bash

set -e

echo "🗂️ Initializing MySQL..."

# Database and table creation
echo "  📋 Creating database and tables..."
mysql -hmysql -uroot -proot -e "
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
    UNIQUE KEY first_name (first_name),
    UNIQUE KEY last_name (last_name),
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
mysql -hmysql -uroot -proot testdb -e "
  INSERT INTO users (first_name, last_name, email, password_hash, role, is_active) 
  VALUES ('admin', 'admin', 'admin@admin.com', '\$2a\$10\$kYoMds9IsbvPNhJasKHO7.fTSosfbPhSAf7ElNQJ9pIa0iWBOt97e', 'admin', true);
"

# Verify initialization
echo "  🔍 Verifying MySQL initialization..."
ADMIN_COUNT=$(mysql -hmysql -uroot -proot testdb -sN -e "SELECT COUNT(*) FROM users WHERE email='admin@admin.com';")
if [ "$ADMIN_COUNT" -eq 1 ]; then
  echo "  ✅ MySQL database and admin user created successfully"
else
  echo "  ❌ Failed to create admin user"
  exit 1
fi 