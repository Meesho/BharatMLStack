#!/bin/bash

set -e

echo "== Checking and initializing all DBs =="

# Wait for ports to be ready
wait_for_port() {
  local name=$1
  local host=$2
  local port=$3

  echo -n "Waiting for $name on $host:$port... "
  while ! nc -z "$host" "$port"; do
    sleep 1
  done
  echo "✅"
}

wait_for_port "ScyllaDB" localhost 9042
wait_for_port "MySQL" localhost 3306
wait_for_port "Redis" localhost 6379
wait_for_port "etcd" localhost 2379

echo "Waiting for Scylla CQL to be ready..."
until docker run --rm --network onfs-network meeshotech/cqlsh:0.1 scylla 9042 -e "SELECT now() FROM system.local"; do
    echo "Waiting for CQL service..."
    sleep 5
done
echo "✅ Scylla CQL is ready!"

echo ""
echo "== ScyllaDB: Creating keyspace and table =="

docker run -i --rm --network "onfs-network" meeshotech/cqlsh:0.1 scylla 9042 <<'EOF'
CREATE KEYSPACE IF NOT EXISTS onfs 
  WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

USE onfs;
EOF

echo ""
echo "== MySQL: Creating tables =="


docker run -i --rm --network "onfs-network" mysql:8 sh -c "cat | mysql -hmysql -uroot -proot testdb" <<'EOF'
DROP DATABASE testdb;
CREATE DATABASE testdb;
USE testdb;
CREATE TABLE `entity` (
  `request_id` int NOT NULL AUTO_INCREMENT,
  `payload` text NOT NULL,
  `entity_label` varchar(255) NOT NULL,
  `created_by` varchar(255) NOT NULL,
  `approved_by` varchar(255) NOT NULL,
  `status` varchar(255) NOT NULL,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `service` varchar(255) NOT NULL,
  `reject_reason` varchar(255) NOT NULL,
  `request_type` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`request_id`)
);


CREATE TABLE `feature_group` (
  `request_id` int NOT NULL AUTO_INCREMENT,
  `payload` text NOT NULL,
  `entity_label` varchar(255) NOT NULL,
  `feature_group_label` varchar(255) NOT NULL,
  `created_by` varchar(255) NOT NULL,
  `approved_by` varchar(255) NOT NULL,
  `status` varchar(255) NOT NULL,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `service` varchar(255) NOT NULL,
  `reject_reason` varchar(255) NOT NULL,
  `request_type` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`request_id`)
);


CREATE TABLE `features` (
  `request_id` int NOT NULL AUTO_INCREMENT,
  `payload` text NOT NULL,
  `entity_label` varchar(255) NOT NULL,
  `feature_group_label` varchar(255) NOT NULL,
  `created_by` varchar(255) NOT NULL,
  `approved_by` varchar(255) NOT NULL,
  `status` varchar(255) NOT NULL,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `request_type` varchar(255) DEFAULT NULL,
  `service` varchar(255) NOT NULL,
  `reject_reason` varchar(255) NOT NULL,
  PRIMARY KEY (`request_id`)
);


CREATE TABLE `job` (
  `request_id` int NOT NULL AUTO_INCREMENT,
  `payload` text NOT NULL,
  `job_id` varchar(255) NOT NULL,
  `created_by` varchar(255) NOT NULL,
  `approved_by` varchar(255) NOT NULL,
  `status` varchar(255) NOT NULL,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `service` varchar(255) NOT NULL,
  `reject_reason` varchar(255) NOT NULL,
  PRIMARY KEY (`request_id`)
);

CREATE TABLE `store` (
  `request_id` int NOT NULL AUTO_INCREMENT,
  `payload` text NOT NULL,
  `created_by` varchar(255) NOT NULL,
  `approved_by` varchar(255) NOT NULL,
  `status` varchar(255) NOT NULL,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `service` varchar(255) NOT NULL,
  `reject_reason` varchar(255) NOT NULL,
  PRIMARY KEY (`request_id`)
);

CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `first_name` varchar(50) NOT NULL,
  `last_name` varchar(50) NOT NULL,
  `email` varchar(100) NOT NULL,
  `password_hash` varchar(255) NOT NULL,
  `role` varchar(10) DEFAULT 'user',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`),
  UNIQUE KEY `first_name` (`first_name`),
  UNIQUE KEY `last_name` (`last_name`),
  UNIQUE KEY `email` (`email`),
  CONSTRAINT `users_chk_1` CHECK ((`role` in (_utf8mb4'user',_utf8mb4'admin')))
);

INSERT INTO `users` (`first_name`, `last_name`, `email`, `password_hash`, `role`) VALUES ('admin', 'admin', 'admin@admin.com', '$2a$10$kYoMds9IsbvPNhJasKHO7.fTSosfbPhSAf7ElNQJ9pIa0iWBOt97e', 'admin');

CREATE TABLE `user_tokens` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_email` varchar(255) NOT NULL,
  `token` varchar(255) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `expires_at` timestamp NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`),
  UNIQUE KEY `token` (`token`)
);
EOF



echo ""
echo "== Redis: Testing connectivity =="

docker run --rm --network onfs-network redis:7 redis-cli -h redis ping

echo ""
echo "== etcd: Creating /config/onfs key =="

docker run --rm --network onfs-network quay.io/coreos/etcd:v3.5.12 etcdctl \
  --endpoints=http://etcd:2379 \
  put /config/onfs "{}"

echo ""
echo "🎉 All checks and setups complete!"
