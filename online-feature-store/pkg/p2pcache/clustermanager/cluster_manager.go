package clustermanager

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
)

const (
	envPodIP  = "POD_IP"
	envNodeIP = "NODE_IP"
)

type ClusterTopology struct {
	RingTopology   map[uint32]string
	ClusterMembers map[string]PodData
}

type PodData struct {
	NodeIP string
	PodIP  string
}

func (p *PodData) GetUniqueId() string {
	key := fmt.Sprintf("%s-%s", p.NodeIP, p.PodIP)
	hash := sha1.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

type ClusterManager interface {
	GetPodIdToKeysMap(keys []string) map[string][]string
	GetPodIdForKey(key string) string
	GetCurrentPodId() string
	GetPodDataForPodId(podId string) (*PodData, error)
	GetClusterTopology() ClusterTopology
}
