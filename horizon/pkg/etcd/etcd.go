package etcd

import (
	"time"
)

const (
	configPath        = "/config/"
	timeout           = 30 * time.Second
	envAppName        = "APP_NAME"
	envEtcdServer     = "ETCD_SERVER"
	envEtcdUsername   = "ETCD_USERNAME"
	envEtcdPassword   = "ETCD_PASSWORD"
	envWatcherEnabled = "ETCD_WATCHER_ENABLED"
)

type Etcd interface {
	GetConfigInstance() interface{}
	updateConfig(config interface{}) error
	handleStruct(dataMap, metaMap *map[string]string, output interface{}, prefix string) error
	handleMap(dataMap, metaMap *map[string]string, output interface{}, prefix string) error
	SetValue(path string, value interface{}) error
	SetValues(paths map[string]interface{}) error
	CreateNode(path string, value interface{}) error
	CreateNodes(paths map[string]interface{}) error
	IsNodeExist(path string) (bool, error)
	IsLeafNodeExist(path string) (bool, error)
	RegisterWatchPathCallback(path string, callback func() error) error
	DeleteValue(path string) error
	DeleteValues(paths []string) error
}
