package clustermanager

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/p2pcache/hashring"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	envEtcdServer   = "ETCD_SERVER"
	envEtcdUsername = "ETCD_USERNAME"
	envEtcdPassword = "ETCD_PASSWORD"
	appName         = "APP_NAME"

	timeout = 5 * time.Second

	configPathFmt = "/config/%s-cluster-manager/%s"
)

type EtcdBasedClusterManager struct {
	mutex          sync.Mutex
	conn           *clientv3.Client
	etcdBasePath   string
	clusterMembers map[string]PodData
	hashRing       *hashring.ConsistentHashRing
	currentPodId   string
	name           string
}

func NewEtcdBasedClusterManager(clusterName string, name string) *EtcdBasedClusterManager {
	if !viper.IsSet(envPodIP) || !viper.IsSet(envNodeIP) {
		log.Panic().Msgf("%s or %s is not set", envPodIP, envNodeIP)
	}
	if !viper.IsSet(envEtcdServer) {
		log.Panic().Msgf("%s is not set", envEtcdServer)
	}
	log.Info().Msgf("Initializing cluster manager with cluster name %s and pod IP %s and node IP %s", clusterName, viper.GetString(envPodIP), viper.GetString(envNodeIP))

	etcdServers := strings.Split(viper.GetString(envEtcdServer), ",")
	var username, password string
	if viper.IsSet(envEtcdUsername) && viper.IsSet(envEtcdPassword) {
		username = viper.GetString(envEtcdUsername)
		password = viper.GetString(envEtcdPassword)
	}

	conn, err := clientv3.New(clientv3.Config{Endpoints: etcdServers, Username: username, Password: password, DialTimeout: timeout, DialKeepAliveTime: timeout, PermitWithoutStream: true})
	if err != nil {
		log.Panic().Err(err).Msg("Error creating etcd client")
	}

	currentPodData := PodData{
		NodeIP: viper.GetString(envNodeIP),
		PodIP:  viper.GetString(envPodIP),
	}
	clusterManager := &EtcdBasedClusterManager{
		hashRing:     hashring.NewConsistentHashRing(),
		conn:         conn,
		etcdBasePath: fmt.Sprintf(configPathFmt, clusterName, name),
		currentPodId: currentPodData.GetUniqueId(),
		name:         name,
	}

	err = clusterManager.init(currentPodData)
	if err != nil {
		log.Panic().Err(err).Msg("Error initializing etcd based cluster manager")
	}

	return clusterManager
}

func (c *EtcdBasedClusterManager) init(currentPodData PodData) error {
	maxRevision, err := c.loadClusterMembers()
	if err != nil {
		return err
	}
	log.Debug().Msgf("Loaded following cluster members from etcd: %v", c.clusterMembers)

	// Watch for changes after loading initial state
	watchChan := c.conn.Watch(context.Background(), c.etcdBasePath, clientv3.WithPrefix(), clientv3.WithRev(maxRevision+1))
	go c.watchEvents(watchChan)

	return c.joinCluster(currentPodData)
}

func (c *EtcdBasedClusterManager) loadClusterMembers() (int64, error) {
	resp, err := c.conn.Get(context.Background(), c.etcdBasePath, clientv3.WithPrefix())
	if err != nil {
		return 0, fmt.Errorf("failed to get cluster pod data from etcd: %v", err)
	}

	c.clusterMembers = make(map[string]PodData)
	for _, kv := range resp.Kvs {
		var podData PodData
		if err := json.Unmarshal(kv.Value, &podData); err != nil {
			return 0, fmt.Errorf("failed to unmarshal pod data: %v", err)
		}

		c.addMember(c.getUniqueIdFromFullConfigKey(string(kv.Key)), podData)
	}

	return resp.Header.Revision, nil
}

func (c *EtcdBasedClusterManager) addMember(key string, podData PodData) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.clusterMembers[key] = podData
	c.hashRing.AddNode(key)
}

func (c *EtcdBasedClusterManager) removeMember(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.clusterMembers, key)
	c.hashRing.RemoveNode(key)
}

func (c *EtcdBasedClusterManager) getUniqueIdFromFullConfigKey(key string) string {
	parts := strings.Split(key, "/")
	return parts[len(parts)-1]
}

func (c *EtcdBasedClusterManager) watchEvents(watchChan clientv3.WatchChan) {
	for watchResp := range watchChan {
		for _, event := range watchResp.Events {
			key := string(event.Kv.Key)

			switch event.Type {
			case clientv3.EventTypePut:
				var podData PodData
				if err := json.Unmarshal(event.Kv.Value, &podData); err != nil {
					log.Error().Err(err).Msgf("Failed to unmarshal updated pod data for key: %s with value: %s", key, string(event.Kv.Value))
					continue
				}
				c.addMember(c.getUniqueIdFromFullConfigKey(key), podData)

			case clientv3.EventTypeDelete:
				c.removeMember(c.getUniqueIdFromFullConfigKey(key))
			}
			log.Debug().Msgf("Updated after eventType: %s for key: %s , cluster members: %v", event.Type, key, c.clusterMembers)
		}
	}
}

func (c *EtcdBasedClusterManager) joinCluster(podData PodData) error {
	// Grant a lease for this pod
	lease, err := c.conn.Grant(context.Background(), 5) // 5 second TTL
	if err != nil {
		return fmt.Errorf("failed to grant lease: %v", err)
	}

	// Start keepalive for the lease
	keepAliveChan, err := c.conn.KeepAlive(context.Background(), lease.ID)
	if err != nil {
		return fmt.Errorf("failed to keep lease alive: %v", err)
	}

	go func() {
		for {
			_, ok := <-keepAliveChan
			if !ok {
				log.Error().Msg("Lost lease keepalive")
				// TODO: Handle with either retry and re-join or bring down the pod itself
				return
			}
		}
	}()

	podDataBytes, err := json.Marshal(podData)
	if err != nil {
		return fmt.Errorf("failed to marshal pod data: %v", err)
	}

	// Put the pod data with lease
	key := fmt.Sprintf("%s/%s", c.etcdBasePath, podData.GetUniqueId())
	_, err = c.conn.Put(context.Background(), key, string(podDataBytes), clientv3.WithLease(lease.ID))
	if err != nil {
		return fmt.Errorf("failed to put pod data in etcd: %v", err)
	}

	return nil
}

func (c *EtcdBasedClusterManager) GetPodIdToKeysMap(keys []string) map[string][]string {
	podIdToKeysMap := make(map[string][]string)
	for _, key := range keys {
		podId := c.GetPodIdForKey(key)
		podIdToKeysMap[podId] = append(podIdToKeysMap[podId], key)
	}
	return podIdToKeysMap
}

func (c *EtcdBasedClusterManager) GetPodIdForKey(key string) string {
	return c.hashRing.GetNodeForKey(key)
}

func (c *EtcdBasedClusterManager) GetCurrentPodId() string {
	return c.currentPodId
}

func (c *EtcdBasedClusterManager) GetPodDataForPodId(podId string) (*PodData, error) {
	podData, ok := c.clusterMembers[podId]
	if !ok {
		return nil, fmt.Errorf("pod data for pod id %s not found", podId)
	}
	return &podData, nil
}

func (c *EtcdBasedClusterManager) GetClusterTopology() ClusterTopology {
	clusterMembers := make(map[string]PodData)
	for key, podData := range c.clusterMembers {
		clusterMembers[key] = podData
	}
	return ClusterTopology{
		RingTopology:   c.hashRing.GetRingTopology(),
		ClusterMembers: clusterMembers,
	}
}
