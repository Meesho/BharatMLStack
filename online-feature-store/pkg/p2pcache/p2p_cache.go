package p2pcache

import (
	"fmt"
	"hash/fnv"
	"maps"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/p2pcache/clustermanager"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/p2pcache/network"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/p2pcache/storage"
	"github.com/rs/zerolog/log"
)

type P2PCacheConfig struct {
	Name                    string
	OwnPartitionSizeInBytes int
	GlobalSizeInBytes       int
	NumClients              int
	ServerPort              int
}

type P2PCache struct {
	cm         clustermanager.ClusterManager
	cacheStore *storage.CacheStore
	clients    []*network.ClientManager
	server     *network.Server
}

func NewP2PCache(config P2PCacheConfig) (*P2PCache, error) {
	err := validateConfig(config)
	if err != nil {
		log.Error().Err(err).Msgf("Error validating p2p cache config %+v", config)
		return nil, err
	}

	cacheStore := storage.NewCacheStore(config.OwnPartitionSizeInBytes, config.GlobalSizeInBytes)
	clients := make([]*network.ClientManager, config.NumClients)
	for i := 0; i < config.NumClients; i++ {
		clients[i] = network.NewClientManager(config.ServerPort)
	}
	return &P2PCache{
		cm:         clustermanager.NewEtcdBasedClusterManager(config.Name),
		cacheStore: cacheStore,
		clients:    clients,
		server:     network.NewServer(config.ServerPort, cacheStore),
	}, nil
}

func validateConfig(config P2PCacheConfig) error {
	if config.OwnPartitionSizeInBytes <= 0 {
		return fmt.Errorf("p2p cache own partition size in bytes must be greater than 0")
	}
	if config.GlobalSizeInBytes <= 0 {
		return fmt.Errorf("p2p cache global size in bytes must be greater than 0")
	}
	if config.Name == "" {
		return fmt.Errorf("p2p cache name must be set")
	}
	if config.NumClients <= 0 {
		return fmt.Errorf("p2p cache num clients must be greater than 0")
	}
	if config.ServerPort <= 0 {
		return fmt.Errorf("p2p cache server port must be greater than 0")
	}
	return nil
}

func (p *P2PCache) MultiGet(keys []string) (map[string][]byte, error) {
	kvResponse := make(map[string][]byte)
	missingKeys := make([]string, 0)

	metric.Count("p2p.cache.keys.total", int64(len(keys)), []string{})
	for _, key := range keys {
		value, err := p.cacheStore.Get(key)
		if err == nil {
			kvResponse[key] = value
		} else {
			missingKeys = append(missingKeys, key)
		}
	}

	if len(missingKeys) > 0 {
		maps.Copy(kvResponse, p.fetchKeysFromOtherPods(missingKeys))
	}
	metric.Count("p2p.cache.keys.global.miss", int64(len(keys)-len(kvResponse)), []string{})
	return kvResponse, nil
}

func (p *P2PCache) MultiSet(kvMap map[string][]byte, ttlInSeconds int) error {
	if len(kvMap) == 0 {
		return nil
	}
	currentPodId := p.cm.GetCurrentPodId()
	for key, value := range kvMap {
		podId := p.cm.GetPodIdForKey(key)
		if podId == currentPodId {
			p.cacheStore.SetIntoOwnPartitionCache(key, value, ttlInSeconds)
		} else {
			p.cacheStore.SetIntoGlobalCache(key, value, ttlInSeconds)
		}
	}
	return nil
}

func (p *P2PCache) MultiDelete(keys []string) error {
	return p.cacheStore.MultiDelete(keys)
}

func (p *P2PCache) GetClusterTopology() clustermanager.ClusterTopology {
	return p.cm.GetClusterTopology()
}

func (p *P2PCache) fetchKeysFromOtherPods(missingKeys []string) map[string][]byte {
	podIdToKeysMap := p.cm.GetPodIdToKeysMap(missingKeys)

	currentPodKeysCount := len(podIdToKeysMap[p.cm.GetCurrentPodId()])
	// Remove keys from the current pod from the map
	metric.Count("p2p.cache.keys.local.miss", int64(currentPodKeysCount), []string{"partition", "own"})
	delete(podIdToKeysMap, p.cm.GetCurrentPodId())

	totalKeysToFetch := len(missingKeys) - currentPodKeysCount
	metric.Count("p2p.cache.keys.local.miss", int64(totalKeysToFetch), []string{"partition", "global"})

	dataChannel := make(chan *network.ResponseMessage, totalKeysToFetch)
	currentPodData, _ := p.cm.GetPodDataForPodId(p.cm.GetCurrentPodId())

	actualKeysFetched := 0
	for podId, keys := range podIdToKeysMap {
		podData, err := p.cm.GetPodDataForPodId(podId)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting pod data for pod id %s", podId)
			continue
		}

		metric.Count("p2p.cache.network.keys.sent", int64(len(keys)), []string{"fromPod", currentPodData.PodIP, "toPod", podData.PodIP})

		actualKeysFetched += len(keys)
		// TODO: Multiple keys belonging to the same pod can be merged and sent in a single packet
		for _, key := range keys {
			go p.fetchKeyFromPod(key, podData, dataChannel)
		}
	}

	kvResponse := make(map[string][]byte)
	for range actualKeysFetched {
		data := <-dataChannel
		if data != nil && data.Data != nil {
			kvResponse[data.Key] = data.Data
		}
	}

	// TODO: Figure out the ttl for the key
	p.cacheStore.MultiSetIntoGlobalCache(kvResponse, 60)
	return kvResponse
}

func (p *P2PCache) fetchKeyFromPod(key string, podData *clustermanager.PodData, dataChannel chan *network.ResponseMessage) {
	dataChannel <- p.clients[p.getClientIdx(key)].GetData(key, podData.PodIP)
}

func (p *P2PCache) getClientIdx(key string) int {
	hash := fnv.New32()
	hash.Write([]byte(key))
	hashValue := hash.Sum32()
	return int(hashValue % uint32(len(p.clients)))
}
