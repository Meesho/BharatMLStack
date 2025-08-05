package p2pcache

import (
	"fmt"
	"maps"
	"time"

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
	cm            clustermanager.ClusterManager
	cacheStore    *storage.CacheStore
	requestRouter *network.RequestRouter
	server        *network.Server
}

func NewP2PCache(config P2PCacheConfig) (*P2PCache, error) {
	err := validateConfig(config)
	if err != nil {
		log.Error().Err(err).Msgf("Error validating p2p cache config %+v", config)
		return nil, err
	}
	cacheStore := storage.NewCacheStore(config.OwnPartitionSizeInBytes, config.GlobalSizeInBytes)
	return &P2PCache{
		cm:            clustermanager.NewEtcdBasedClusterManager(config.Name),
		cacheStore:    cacheStore,
		requestRouter: network.NewRequestRouter(config.NumClients, config.ServerPort),
		server:        network.NewServer(config.ServerPort, cacheStore),
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
	return kvResponse, nil
}

func (p *P2PCache) MultiSet(kvMap map[string][]byte, ttlInSeconds int) error {
	// Setting into global cache is applicable only when fetching data from other pods and it is handled in the getter methods
	return p.cacheStore.MultiSetIntoOwnPartitionCache(kvMap, ttlInSeconds)
}

func (p *P2PCache) MultiDelete(keys []string) error {
	return p.cacheStore.MultiDelete(keys)
}

func (p *P2PCache) GetClusterTopology() clustermanager.ClusterTopology {
	return p.cm.GetClusterTopology()
}

func (p *P2PCache) fetchKeysFromOtherPods(missingKeys []string) map[string][]byte {
	kvResponse := make(map[string][]byte)
	keysToPodIdMap := p.cm.GetKeysToPodIdMap(missingKeys)
	if len(keysToPodIdMap) == 0 {
		return kvResponse
	}

	dataChannel := make(chan *network.ResponseMessage, len(keysToPodIdMap))
	for key, podId := range keysToPodIdMap {
		// TODO: Multiple keys belonging to the same pod can be merged and sent in a single packet
		go p.fetchKeyFromPod(key, podId, dataChannel)
	}

	for i := 0; i < len(keysToPodIdMap); i++ {
		data := <-dataChannel
		if data != nil && data.Data != nil {
			kvResponse[data.Key] = data.Data
		}
	}

	// TODO: Figure out the ttl for the key
	p.cacheStore.MultiSetIntoGlobalCache(kvResponse, 60)
	return kvResponse
}

func (p *P2PCache) fetchKeyFromPod(key string, podId string, dataChannel chan *network.ResponseMessage) {
	_, err := p.cm.GetPodDataForPodId(podId)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting pod data for pod id %s", podId)
		dataChannel <- nil
		return
	}

	responseChan := p.requestRouter.GetData(key, podId)
	select {
	case value := <-responseChan:
		dataChannel <- &value
	case <-time.After(network.REQUEST_TIMEOUT):
		// TODO: Add metrics for timeout
		close(responseChan)
		dataChannel <- nil
	}
}
