-- Migration script to add SSO fields and update users table
-- Run this on existing databases

-- Add new columns to users table
ALTER TABLE users 
  ADD COLUMN auth_provider enum('password', 'google', 'both') DEFAULT 'password' AFTER is_active,
  ADD COLUMN google_id varchar(255) DEFAULT NULL AFTER auth_provider,
  ADD COLUMN profile_picture_url varchar(500) DEFAULT NULL AFTER google_id,
  ADD COLUMN email_verified boolean DEFAULT false AFTER profile_picture_url,
  ADD COLUMN last_login_at timestamp NULL DEFAULT NULL AFTER email_verified,
  ADD COLUMN created_by bigint unsigned DEFAULT NULL AFTER last_login_at,
  ADD COLUMN updated_by bigint unsigned DEFAULT NULL AFTER created_by;

-- Make password_hash nullable for SSO-only users
ALTER TABLE users MODIFY password_hash varchar(255) DEFAULT NULL;

-- Update role enum to include super_admin
ALTER TABLE users DROP CHECK users_chk_1;
ALTER TABLE users ADD CONSTRAINT users_chk_1 CHECK ((role in ('user','admin','super_admin')));

-- Change is_active default to true
ALTER TABLE users ALTER COLUMN is_active SET DEFAULT true;

-- Add indexes
ALTER TABLE users ADD UNIQUE INDEX idx_google_id (google_id);
ALTER TABLE users ADD INDEX idx_auth_provider (auth_provider);
ALTER TABLE users ADD INDEX idx_email (email);

-- Add foreign key constraints
ALTER TABLE users 
  ADD CONSTRAINT fk_users_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_users_updated_by FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;

-- Update existing users to have auth_provider = 'password'
UPDATE users SET auth_provider = 'password' WHERE auth_provider IS NULL;

-- Update user_tokens table for refresh tokens
ALTER TABLE user_tokens 
  ADD COLUMN refresh_token varchar(255) DEFAULT NULL AFTER token,
  ADD COLUMN token_type enum('access', 'refresh') DEFAULT 'access' AFTER refresh_token;

-- Add indexes for refresh tokens
ALTER TABLE user_tokens 
  ADD INDEX idx_refresh_token (refresh_token),
  ADD INDEX idx_user_email_token_type (user_email, token_type);

-- Create permissions table
CREATE TABLE IF NOT EXISTS permissions (
  id bigint unsigned NOT NULL AUTO_INCREMENT,
  role enum('super_admin', 'admin', 'user') NOT NULL,
  service varchar(255) NOT NULL,
  screen_type varchar(255) NOT NULL,
  allowed_actions json NOT NULL,
  created_by bigint unsigned NOT NULL,
  updated_by bigint unsigned NOT NULL,
  created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_role_service_screen (role, service, screen_type),
  KEY idx_role (role),
  CONSTRAINT fk_permissions_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT,
  CONSTRAINT fk_permissions_updated_by FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


