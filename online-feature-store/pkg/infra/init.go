package infra

import "sync"

var (
	mut             sync.Mutex
	ConfIdDBTypeMap = make(map[int]DBType)
)

func InitDBConnectors() {
	mut.Lock()
	defer mut.Unlock()
	if RedisCluster == nil {
		initRedisClusterConns()
	}
	if RedisStandalone == nil {
		initRedisStandaloneConns()
	}
	if RedisFailover == nil {
		initRedisFailoverConns()
	}
	if Scylla == nil {
		initScyllaClusterConns()
	}
	if InMemoryCache == nil {
		initInMemoryCacheConns()
	}
	if P2PCache == nil {
		initP2PCacheConns()
	}
}
