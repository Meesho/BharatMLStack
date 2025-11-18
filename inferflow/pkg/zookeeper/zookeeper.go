package zookeeper

import (
	"fmt"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/go-zookeeper/zk"
	"github.com/knadh/koanf"
)

const (
	ZK_CONFIG_PREFIX = "zookeeper"
)

var (
	conn      *zk.Conn
	zkConfig  *ZKConfig
	zkAppPath string
)

func InitZKConnection(kConfig *koanf.Koanf) {

	zkAppPath = "/config/" + kConfig.MustString("applicationName")
	err := kConfig.Unmarshal(ZK_CONFIG_PREFIX, &zkConfig)
	if err != nil {
		logger.Panic("Error while zk config unmarshal ", err)
	}
	servers := strings.Split(zkConfig.Server, ",")
	conn, _, err = zk.Connect(servers, time.Second*5)
	if err != nil {
		logger.Panic("Unable to connect to zk server ", err)
	}
}

func Get(nodePath string) ([]byte, *zk.Stat, error) {

	data, stat, err := conn.Get(nodePath)
	if err != nil {
		logger.Error(fmt.Sprintf("Error getting config from zk path %s ", nodePath), nil)
		return nil, nil, err
	}
	return data, stat, err
}

func GetChildren(zkPath string) (map[string][]byte, *zk.Stat, error) {

	zkAbsolutePath := zkAppPath + zkPath
	children, _, err := conn.Children(zkAbsolutePath)
	if err != nil {
		logger.Error(fmt.Sprintf("Error getting config from zk path %s ", zkAbsolutePath), nil)
		return nil, nil, err
	}

	byteMap := make(map[string][]byte)
	// Iterate over the child nodes and fetch data for each one
	for _, child := range children {
		nodePath := strings.Join([]string{zkAbsolutePath, child}, "/")
		data, _, err := Get(nodePath)
		if err != nil {
			return nil, nil, err
		} else {
			byteMap[child] = data
		}
	}
	return byteMap, nil, nil
}
