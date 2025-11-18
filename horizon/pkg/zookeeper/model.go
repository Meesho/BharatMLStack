package zookeeper

import (
	"sync"

	"github.com/go-zookeeper/zk"
)

type ZKConfig struct {
	Server  string `koanf:"server"`
	Watcher string `koanf:"watcher"`
}

type Node struct {
	Path     string           `json:"path"`
	Data     interface{}      `json:"data"`
	Children map[string]*Node `json:"children"`
}

type TreeCache struct {
	conn     *zk.Conn     `json:"-"`
	TreeLock sync.RWMutex `json:"-"`
	RootNode *Node        `json:"rootNode"`
}
