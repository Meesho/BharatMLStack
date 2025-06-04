package infra

import "sync"

var (
	mut             sync.Mutex
	ConfIdDBTypeMap = make(map[int]DBType)
)

func InitDBConnectors() {
	mut.Lock()
	defer mut.Unlock()
	if Scylla == nil {
		initScyllaClusterConns()
	}
	if SQL == nil {
		initSQLConns()
	}
}
