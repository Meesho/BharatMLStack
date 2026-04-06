package etcd

import (
	"time"
)

const (
	configPath = "/config/"
	timeout    = 5 * time.Second
)

type Etcd interface {
	GetConfigInstance() interface{}
	updateConfig(config interface{}) error
	handleStruct(dataMap, metaMap *map[string]string, output interface{}, prefix string) error
	handleMap(dataMap, metaMap *map[string]string, output interface{}, prefix string) error
	SetValue(path string, value interface{}) error
	SetValues(paths map[string]interface{}) error
	GetValue(path string) (string, error)
	CreateNode(path string, value interface{}) error
	CreateNodes(paths map[string]interface{}) error
	IsNodeExist(path string) (bool, error)
	IsLeafNodeExist(path string) (bool, error)
	RegisterWatchPathCallback(path string, callback func() error) error
	DeleteNode(path string) error
	GetWatcherDataMaps() (dataMap map[string]string, metaMap map[string]string, err error)
}
