package infra

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	SQL *SQLConnectors
)

// SQLConnectors holds ConnectionFacade
type SQLConnectors struct {
	SQLConnection ConnectionFacade
}

func (s *SQLConnectors) GetConnection() (ConnectionFacade, error) {
	if s.SQLConnection == nil {
		return nil, errors.New("connection not found")
	}
	return s.SQLConnection, nil
}

// SQLConnection encapsulates MySQL database connections with master/slave setup
type SQLConnection struct {
	Master *gorm.DB
	Slave  *gorm.DB
	Meta   map[string]interface{}
}

// GetConn returns the database connection (master)
func (c *SQLConnection) GetConn() (interface{}, error) {
	if c.Master == nil {
		return nil, errors.New("master connection is nil")
	}
	return c.Master, nil
}

// GetMeta returns metadata about the connection
func (c *SQLConnection) GetMeta() (map[string]interface{}, error) {
	if c.Meta == nil {
		return nil, errors.New("meta is nil")
	}
	return c.Meta, nil
}

// GetMaster returns the master database connection
func (c *SQLConnection) GetMaster() *gorm.DB {
	return c.Master
}

// GetSlave returns the slave database connection
// If no slave is configured, it returns the master connection
func (c *SQLConnection) GetSlave() *gorm.DB {
	if c.Slave != nil {
		return c.Slave
	}
	return c.Master
}

func (c *SQLConnection) IsLive() bool {
	return c.Master != nil && c.Slave != nil
}

// initSQLConns initializes all SQL connections based on environment configuration
func initSQLConns() {

	// Create master and slave connections from config
	masterConfig, slaveConfig, err := BuildSQLConfigFromEnv()
	if err != nil {
		log.Panic().Msg(err.Error())
	}
	// Create master connection
	master, err := CreateMySQLConnection(masterConfig)
	if err != nil {
		log.Panic().Msg(err.Error())
	}
	var slave *gorm.DB
	// Create slave connection if configuration exists
	if slaveConfig.Host != "" {
		slave, err = CreateMySQLConnection(slaveConfig)
		if err != nil {
			log.Error().Err(err).Msg("Failed to connect to slave, will use master only")
			// Continue with master only if slave connection fails
		} else {
			log.Info().Msg("Connected to slave")
		}
	}

	if _, ok := ConfIdDBTypeMap[0]; ok {
		log.Panic().Msg("Duplicate config id")
	}

	ConfIdDBTypeMap[0] = DBTypeMySQL

	conn := &SQLConnection{
		Master: master,
		Slave:  slave,
		Meta: map[string]interface{}{
			"configId": 0,
			"db_name":  masterConfig.DBName,
			"type":     DBTypeMySQL,
		},
	}

	SQL = &SQLConnectors{
		SQLConnection: conn,
	}
}

// CreateMySQLConnection creates a MySQL connection from SQLConfig
func CreateMySQLConnection(config SQLConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Asia%%2FKolkata",
		config.Username, config.Password, config.Host, config.Port, config.DBName)

	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
