package infra

type DBType string

const (
	DBTypeScylla        DBType = "scylla"
	DBTypeMySQL         DBType = "mysql"
	activeConfIds              = "ACTIVE_CONFIG_IDS"
	DefaultSqlConfId           = 2
	DefaultScyllaConfId        = 1
)

// ConnectionFacade is a common interface for all database connections
type ConnectionFacade interface {
	// GetConn returns the database connection
	GetConn() (interface{}, error)

	// GetMeta returns metadata about the connection
	GetMeta() (map[string]interface{}, error)
	IsLive() bool
}

type Connector interface {
	GetConnection(configId int) (ConnectionFacade, error)
}
