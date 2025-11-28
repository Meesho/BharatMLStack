package etcd

import (
	"sync"
	"time"
)

const (
	basePath          = "/config/inferflow/"
	configPath        = "/model-config"
	connectionTimeout = 30 * time.Second
	envAppName        = "applicationName"
	envEtcdServer     = "ETCD_SERVER"
	envWatcherEnabled = "ETCD_WATCHER_ENABLED"
)

var (
	once sync.Once
)

type Etcd interface {
	GetConfigInstance() interface{}
	GetBasePath() string
	UpdateConfig(config interface{}) error
	HandleStruct(dataMap, metaMap *map[string]string, output interface{}, prefix string) error
	HandleMap(dataMap, metaMap *map[string]string, output interface{}, prefix string) error
	SetValue(path string, value interface{}) error
	SetValues(paths map[string]interface{}) error
	CreateNode(path string, value interface{}) error
	CreateNodes(paths map[string]interface{}) error
	IsNodeExist(path string) (bool, error)
	RegisterWatchPathCallback(path string, callback func() error) error
}
