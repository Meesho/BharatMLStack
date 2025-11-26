package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"strconv"

	"github.com/Meesho/BharatMLStack/cli-tools/pkg/setup"
	"github.com/gocql/gocql"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	setupOnFs := flag.Bool("setup-on-fs", false, "setup on fs")
	flag.Parse()
	mysqlHost := flag.String("mysql-host", "", "mysql host")
	mysqlPort := flag.String("mysql-port", "", "mysql port")
	mysqlUser := flag.String("mysql-user", "", "mysql user")
	mysqlPassword := flag.String("mysql-password", "", "mysql password")
	mysqlDatabase := flag.String("mysql-database", "", "mysql database")

	etcdHost := flag.String("etcd-host", "", "etcd host")
	etcdPort := flag.String("etcd-port", "", "etcd port")
	etcdUser := flag.String("etcd-user", "", "etcd user")
	etcdPassword := flag.String("etcd-password", "", "etcd password")

	setupStorage := flag.Bool("setup-storage", false, "setup storage")

	storageType := flag.String("storage-type", "", "storage type")
	storageHosts := flag.String("storage-hosts", "", "storage hosts")
	storagePort := flag.String("storage-port", "", "storage port")
	storageUser := flag.String("storage-user", "", "storage user")
	storagePassword := flag.String("storage-password", "", "storage password")
	storageKeyspace := flag.String("storage-keyspace", "", "storage keyspace")

	help := flag.Bool("help", false, "help")

	if *help {
		fmt.Println("Usage:")
		fmt.Println("  cli-tools --setup-on-fs --mysql-host <mysql-host> --mysql-port <mysql-port> --mysql-user <mysql-user> --mysql-password <mysql-password> --mysql-database <mysql-database> --etcd-host <etcd-host> --etcd-port <etcd-port> --etcd-user <etcd-user> --etcd-password <etcd-password>")
		fmt.Println("  cli-tools --setup-on-fs --setup-storage --storage-type <storage-type> --storage-hosts <storage-hosts> --storage-port <storage-port> --storage-user <storage-user> --storage-password <storage-password> --storage-keyspace <storage-keyspace>")
		return
	}

	if *setupOnFs {
		if !setup.ValidateOnFsFlags(*mysqlHost, *mysqlPort, *mysqlUser, *mysqlPassword, *mysqlDatabase, *etcdHost, *etcdPort, *etcdUser, *etcdPassword) {
			return
		}

		etcdClient, err := clientv3.New(clientv3.Config{
			Endpoints: []string{*etcdHost + ":" + *etcdPort},
			Username:  *etcdUser,
			Password:  *etcdPassword,
		})
		if err != nil {
			fmt.Println("Error: unable to create etcd client", err)
			return
		}

		if !setup.SetupEtcdPaths(etcdClient) {
			return
		}

		mysqlDns := "%s:%s@tcp(%s:%s)/"
		mysqlDns = fmt.Sprintf(mysqlDns, *mysqlUser, *mysqlPassword, *mysqlHost, *mysqlPort)

		sqlDB, err := sql.Open("mysql", mysqlDns)
		if err != nil {
			fmt.Println("Error: unable to open mysql connection", err)
			return
		} else {
			fmt.Println("MySQL: Database created successfully")
		}

		if !setup.SetupMysqlDb(sqlDB, *mysqlDatabase) {
			return
		}

		mysqlDBDsn := "%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local"
		mysqlDBDsn = fmt.Sprintf(mysqlDBDsn, *mysqlUser, *mysqlPassword, *mysqlHost, *mysqlPort, *mysqlDatabase)

		gormDB, err := gorm.Open(mysql.Open(mysqlDBDsn), &gorm.Config{})
		if err != nil {
			log.Fatal("GORM: Open error: ", err)
		} else {
			fmt.Println("GORM: Connection established successfully")
		}

		if !setup.SetupMysqlTables(gormDB, *mysqlDatabase) {
			return
		}

		if *setupStorage {
			if !setup.ValidateStorageFlags(*storageType, *storageHosts, *storagePort, *storageUser, *storagePassword, *storageKeyspace) {
				return
			}

			if *storageType == "scylla" || *storageType == "cassandra" {
				port, err := strconv.Atoi(*storagePort)
				if err != nil {
					fmt.Println("Error: unable to convert storage port to int", err)
					return
				}

				cluster := gocql.NewCluster(*storageHosts)
				cluster.Port = port
				cluster.Authenticator = gocql.PasswordAuthenticator{
					Username: *storageUser,
					Password: *storagePassword,
				}

				if *storageType == "cassandra" {
					cassandraSession, err := cluster.CreateSession()
					if err != nil {
						fmt.Println("Error: unable to create cassandra session", err)
						return
					}
					if !setup.SetupCassandraKeyspaceAndTables(cassandraSession, *storageKeyspace) {
						return
					}
				} else {
					scyllaSession, err := cluster.CreateSession()
					if err != nil {
						fmt.Println("Error: unable to create scylla session", err)
						return
					}
					if !setup.SetupScyllaKeyspaceAndTables(scyllaSession, *storageKeyspace) {
						return
					}
				}
			}
		} else {
			fmt.Println("Storage setup is disabled")
		}
	}
}
