package infra

type DBType string

const (
	DBTypeScylla          DBType = "scylla"
	DBTypeRedisStandalone DBType = "standalone_redis"
	DBTypeRedisFailover   DBType = "failover_redis"
	DBTypeRedisCluster    DBType = "cluster_redis"
	DBTypeInMemory        DBType = "in_memory"
	DBTypeP2P             DBType = "p2p"
	activeConfIds                = "ACTIVE_CONFIG_IDS"
)

type ConnectionFacade interface {
	GetConn() (interface{}, error)
	GetMeta() (map[string]interface{}, error)
	IsLive() bool
}

type Connector interface {
	GetConnection(configId int) (ConnectionFacade, error)
}
