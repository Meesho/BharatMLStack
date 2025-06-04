package infra

import (
	"errors"

	"github.com/spf13/viper"
)

// Mandatory config keys
// <STORAGE_SQL_1_MASTER_HOST> =
// <STORAGE_SQL_1_MASTER_PORT> =
// <STORAGE_SQL_1_DB_NAME> =
// <STORAGE_SQL_1_MASTER_USERNAME> =
// <STORAGE_SQL_1_MASTER_PASSWORD> =
const (
	masterHost     = "MYSQL_MASTER_HOST"
	masterPort     = "MYSQL_MASTER_PORT"
	masterDBName   = "MYSQL_DB_NAME"
	masterUsername = "MYSQL_MASTER_USERNAME"
	masterPassword = "MYSQL_MASTER_PASSWORD"

	slaveHost     = "MYSQL_SLAVE_HOST"
	slavePort     = "MYSQL_SLAVE_PORT"
	slaveDBName   = "MYSQL_DB_NAME"
	slaveUsername = "MYSQL_SLAVE_USERNAME"
	slavePassword = "MYSQL_SLAVE_PASSWORD"
)

// SQLConfig represents the configuration for a SQL database connection
type SQLConfig struct {
	Host     string
	Port     int
	DBName   string
	Username string
	Password string
}

// BuildSQLConfigFromEnv constructs a SQL configuration for master and slave
// from environment variables using the specified prefix.
//
// The function leverages Viper to read environment variables, ensuring
// a flexible and configurable setup. It extracts key parameters required
// for configuring MySQL connections.
//
// Mandatory environment variables:
//   - MYSQL_MASTER_HOST: Master host
//   - MYSQL_MASTER_PORT: Master port
//   - MYSQL_DB_NAME: Database name
//   - MYSQL_MASTER_USERNAME: Master username
//   - MYSQL_MASTER_PASSWORD: Master password
//
// Optional environment variables for slave:
//   - MYSQL_SLAVE_HOST: Slave host
//   - MYSQL_SLAVE_PORT: Slave port
//   - MYSQL_SLAVE_USERNAME: Slave username
//   - MYSQL_SLAVE_PASSWORD: Slave password
//
// Returns:
//   - Master and slave SQLConfig instances and an error if mandatory variables are missing
func BuildSQLConfigFromEnv() (master SQLConfig, slave SQLConfig, err error) {
	// Check required master configuration
	if !viper.IsSet(masterHost) {
		return master, slave, errors.New(masterHost + " not set")
	}
	if !viper.IsSet(masterPort) {
		return master, slave, errors.New(masterPort + " not set")
	}
	if !viper.IsSet(masterDBName) {
		return master, slave, errors.New(masterDBName + " not set")
	}
	if !viper.IsSet(masterUsername) {
		return master, slave, errors.New(masterUsername + " not set")
	}

	// Set master configuration
	master = SQLConfig{
		Host:     viper.GetString(masterHost),
		Port:     viper.GetInt(masterPort),
		DBName:   viper.GetString(masterDBName),
		Username: viper.GetString(masterUsername),
		Password: viper.GetString(masterPassword),
	}

	// Check if slave configuration is provided
	if viper.IsSet(slaveHost) &&
		viper.IsSet(slaveUsername) &&
		viper.IsSet(slavePassword) {

		// If slave port is not set, use master port
		slavePortValue := viper.GetInt(masterPort)
		if viper.IsSet(slavePort) {
			slavePortValue = viper.GetInt(slavePort)
		}

		// Set slave configuration
		slave = SQLConfig{
			Host:     viper.GetString(slaveHost),
			Port:     slavePortValue,
			DBName:   viper.GetString(masterDBName), // Use master DB name by default
			Username: viper.GetString(slaveUsername),
			Password: viper.GetString(slavePassword),
		}
	}

	return master, slave, nil
}
