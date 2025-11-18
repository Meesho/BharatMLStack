package infra

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
)

// SQLConfig represents the configuration for a SQL database connection
type SQLConfig struct {
	Host     string
	Port     int
	DBName   string
	Username string
	Password string
}

// BuildSQLConfigFromConfig constructs a SQL configuration for master and slave
// from the application config struct.
//
// The function extracts key parameters required for configuring MySQL connections
// from the provided config struct.
//
// Mandatory config fields:
//   - MysqlMasterHost: Master host
//   - MysqlMasterPort: Master port
//   - MysqlDbName: Database name
//   - MysqlMasterUsername: Master username
//   - MysqlMasterPassword: Master password
//
// Optional config fields for slave:
//   - MysqlSlaveHost: Slave host
//   - MysqlSlavePort: Slave port
//   - MysqlSlaveUsername: Slave username
//   - MysqlSlavePassword: Slave password
//
// Returns:
//   - Master and slave SQLConfig instances and an error if mandatory fields are missing
func BuildSQLConfigFromConfig(config configs.Configs) (master SQLConfig, slave SQLConfig, err error) {
	// Check required master configuration
	if config.MysqlMasterHost == "" {
		return master, slave, errors.New("MysqlMasterHost not set")
	}
	if config.MysqlMasterPort == 0 {
		return master, slave, errors.New("MysqlMasterPort not set")
	}
	if config.MysqlDbName == "" {
		return master, slave, errors.New("MysqlDbName not set")
	}
	if config.MysqlMasterUsername == "" {
		return master, slave, errors.New("MysqlMasterUsername not set")
	}

	// Set master configuration
	master = SQLConfig{
		Host:     config.MysqlMasterHost,
		Port:     config.MysqlMasterPort,
		DBName:   config.MysqlDbName,
		Username: config.MysqlMasterUsername,
		Password: config.MysqlMasterPassword,
	}

	// Check if slave configuration is provided
	if config.MysqlSlaveHost != "" &&
		config.MysqlSlaveUsername != "" &&
		config.MysqlSlavePassword != "" {

		// If slave port is not set, use master port
		slavePortValue := config.MysqlMasterPort
		if config.MysqlSlavePort != 0 {
			slavePortValue = config.MysqlSlavePort
		}

		// Set slave configuration
		slave = SQLConfig{
			Host:     config.MysqlSlaveHost,
			Port:     slavePortValue,
			DBName:   config.MysqlDbName, // Use master DB name by default
			Username: config.MysqlSlaveUsername,
			Password: config.MysqlSlavePassword,
		}
	}

	return master, slave, nil
}
