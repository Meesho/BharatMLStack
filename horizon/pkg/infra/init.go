package infra

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
)

var (
	mut             sync.Mutex
	ConfIdDBTypeMap = make(map[int]DBType)
)

func InitDBConnectors(config configs.Configs) {
	mut.Lock()
	defer mut.Unlock()
	if Scylla == nil {
		initScyllaClusterConns(config)
	}
	if SQL == nil {
		initSQLConns(config)
	}
}
