package setup

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gocql/gocql"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/gorm"
)

func ValidateOnFsFlags(mysqlHost string, mysqlPort string, mysqlUser string, mysqlPassword string, mysqlDatabase string, etcdHost string, etcdPort string, etcdUser string, etcdPassword string) bool {
	if mysqlHost == "" || mysqlPort == "" || mysqlDatabase == "" {
		fmt.Println("Error: mysql host, port and database are required")
		return false
	}

	if etcdHost == "" || etcdPort == "" {
		fmt.Println("Error: etcd host and port are required")
		return false
	}

	return true
}

func ValidateStorageFlags(storageType string, storageHosts string, storagePort string, storageUser string, storagePassword string, storageKeyspace string) bool {
	if storageType == "" || storageHosts == "" || storagePort == "" || storageKeyspace == "" {
		fmt.Println("Error: storage type, hosts, port and keyspace are required")
		return false
	}

	return true
}

func SetupEtcdPaths(etcdClient *clientv3.Client) bool {
	ctx := context.Background()
	etcdClient.Put(ctx, "/config/onfs", "{}")
	etcdClient.Put(ctx, "/config/onfs/security/reader/test", "{\"token\":\"test\"}")

	resp, err := etcdClient.Get(ctx, "/config/onfs/security/reader/test")
	if err != nil {
		fmt.Println("Error: ", err)
		return false
	}

	if string(resp.Kvs[0].Value) != "{\"token\":\"test\"}" {
		fmt.Println("Error: Token set test failed in etcd")
		return false
	}
	return true
}

func SetupMysqlDb(sqlDB *sql.DB, database string) bool {
	defer sqlDB.Close()
	_, err := sqlDB.Exec("CREATE DATABASE IF NOT EXISTS " + database)
	if err != nil {
		fmt.Println("Error: unable to create database", err)
		return false
	} else {
		fmt.Println("MySQL: Database created successfully")
	}
	return true
}

func SetupMysqlTables(gormDB *gorm.DB, database string) bool {

	// Create tables
	tables := []string{
		`CREATE TABLE IF NOT EXISTS entity (
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
		)`,
		`CREATE TABLE IF NOT EXISTS feature_group (
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
		)`,
		`CREATE TABLE IF NOT EXISTS features (
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
		)`,
		`CREATE TABLE IF NOT EXISTS job (
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
		)`,
		`CREATE TABLE IF NOT EXISTS store (
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
		)`,
		`CREATE TABLE IF NOT EXISTS users (
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
		)`,
		`CREATE TABLE IF NOT EXISTS user_tokens (
			id bigint unsigned NOT NULL AUTO_INCREMENT,
			user_email varchar(255) NOT NULL,
			token varchar(255) NOT NULL,
			created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at timestamp NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY id (id),
			UNIQUE KEY token (token)
		)`,
	}

	for _, tableSQL := range tables {
		err := gormDB.Exec(tableSQL).Error
		if err != nil {
			fmt.Println("Error: unable to create table", err)
			return false
		} else {
			fmt.Println("MySQL: Table created successfully")
		}
	}

	return true
}

func SetupScyllaKeyspaceAndTables(scyllaSession *gocql.Session, keyspace string) bool {
	defer scyllaSession.Close()
	err := scyllaSession.Query(fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s", keyspace)).Exec()
	if err != nil {
		fmt.Println("Error: unable to create keyspace", err)
		return false
	}
	fmt.Println("Scylla: Keyspace created successfully")
	return true
}

func SetupCassandraKeyspaceAndTables(cassandraSession *gocql.Session, keyspace string) bool {
	defer cassandraSession.Close()
	err := cassandraSession.Query(fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s", keyspace)).Exec()
	if err != nil {
		fmt.Println("Error: unable to create keyspace", err)
		return false
	}
	fmt.Println("Cassandra: Keyspace created successfully")
	return true
}
