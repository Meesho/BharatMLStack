package zookeeper

import (
	"sync"
	"time"
)

const (
	configPath                     = "/config/"
	timeout                        = 5 * time.Second
	envAppName                     = "APP_NAME"
	envZookeeperServer             = "ZOOKEEPER_SERVER"
	envPeriodicUpdateEnabled       = "ZOOKEEPER_PERIODIC_UPDATE_ENABLED"
	envPeriodicUpdateInterval      = "ZOOKEEPER_PERIODIC_UPDATE_INTERVAL"
	envSlackNotificationEnabled    = "ZOOKEEPER_SLACK_NOTIFICATION_ENABLED"
	envSlackChannelId              = "ZOOKEEPER_SLACK_CHANNEL_ID"
	envWatcherEnabled              = "ZOOKEEPER_WATCHER_ENABLED"
	configKeyDynamicSourceOptional = "dynamic-config.optional"
)

var (
	once sync.Once
)

type ZK interface {
	GetConfigInstance() interface{}
	enablePeriodicConfigUpdate(duration time.Duration)
	updateConfig(config interface{}) error
	registerAndWatchNodes(path string) error
	updateConfigForWatcherEvent(data, nodePath string, deleteNode bool) error
	handleStruct(dataMap, metaMap *map[string]string, output interface{}, prefix string, deleteNode bool) error
	handleMap(dataMap, metaMap *map[string]string, output interface{}, prefix string, deleteNode bool) error
	SetValue(path string, value interface{}) error
	SetValues(paths map[string]interface{}) error
	CreateNode(path string, value interface{}) error
	CreateNodes(paths map[string]interface{}) error
	IsNodeExist(path string) (bool, error)
}
