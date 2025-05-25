package p2pcache

import (
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/p2pcache/clustermanager"
	"github.com/coocood/freecache"
	"github.com/rs/zerolog/log"
)

func NewP2PCache(name string, ownPartitionSizeInBytes int, globalSizeInBytes int) *P2PCache {
	cm := clustermanager.NewEtcdBasedClusterManager(name)
	return &P2PCache{
		cm:                cm,
		ownPartitionCache: freecache.NewCache(ownPartitionSizeInBytes),
		globalCache:       freecache.NewCache(globalSizeInBytes),
	}
}

type P2PCache struct {
	cm                clustermanager.ClusterManager
	ownPartitionCache *freecache.Cache
	globalCache       *freecache.Cache
}

func (c *P2PCache) MultiGet(keys []string) (map[string][]byte, error) {
	podIdToKeysMap, err := c.cm.GetPodIdToKeysMap(keys)
	if err != nil {
		return nil, err
	}

	kvMap := make(map[string][]byte)
	currentPodId := c.cm.GetCurrentPodId()

	if _, ok := podIdToKeysMap[currentPodId]; ok {
		c.fillFromOwnPartition(podIdToKeysMap[currentPodId], kvMap)
	}
	delete(podIdToKeysMap, currentPodId)

	c.fillFromOtherPartitions(podIdToKeysMap, kvMap)
	return kvMap, nil
}

func (c *P2PCache) MultiSet(kvMap map[string][]byte, ttlInSeconds int) error {
	// Get the node mapping for all keys
	keyToPodIdMap, err := c.cm.GetKeyToPodIdMap(c.getKeys(kvMap))
	if err != nil {
		return err
	}

	currentPodId := c.cm.GetCurrentPodId()
	for k, v := range kvMap {
		if keyToPodIdMap[k] == currentPodId {
			_ = c.ownPartitionCache.Set([]byte(k), v, ttlInSeconds)
		} else {
			_ = c.globalCache.Set([]byte(k), v, ttlInSeconds)
		}
	}
	return nil
}

func (c *P2PCache) MultiDelete(keys []string) error {
	for _, k := range keys {
		c.ownPartitionCache.Del([]byte(k))
		c.globalCache.Del([]byte(k))
	}
	return nil
}

func (c *P2PCache) GetClusterTopology() clustermanager.ClusterTopology {
	return c.cm.GetClusterTopology()
}

func (c *P2PCache) getKeys(kvMap map[string][]byte) []string {
	keys := make([]string, 0, len(kvMap))
	for k := range kvMap {
		keys = append(keys, k)
	}
	return keys
}

func (c *P2PCache) fillFromOwnPartition(keys []string, kvMap map[string][]byte) {
	for _, key := range keys {
		value, err := c.ownPartitionCache.Get([]byte(key))
		if err == nil {
			kvMap[key] = value
		}
	}
}

func (c *P2PCache) fillFromOtherPartitions(podIdToKeysMap map[string][]string, kvMap map[string][]byte) {
	missingKeysInGlobalCache := make(map[string][]string)
	for podId, keys := range podIdToKeysMap {
		for _, key := range keys {
			value, err := c.globalCache.Get([]byte(key))
			if err == nil {
				kvMap[key] = value
			} else {
				missingKeysInGlobalCache[podId] = append(missingKeysInGlobalCache[podId], key)
			}
		}
	}

	for podId, keys := range missingKeysInGlobalCache {
		podData, err := c.cm.GetPodDataForPodId(podId)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting pod data for pod id %s", podId)
			continue
		}
		valuesFromOtherPartitions := c.fetchValuesFromOtherPartitions(podData, keys)
		for key, value := range valuesFromOtherPartitions {
			kvMap[key] = value
		}
		c.loadIntoGlobalCache(valuesFromOtherPartitions)
	}
}

func (c *P2PCache) fetchValuesFromOtherPartitions(podData *clustermanager.PodData, keys []string) map[string][]byte {
	// TODO: Implement calling other pods to fetch values
	kvMap := make(map[string][]byte)
	return kvMap
}

func (c *P2PCache) loadIntoGlobalCache(kvMap map[string][]byte) {
	for key, value := range kvMap {
		// TODO: Figure out the ttl for the key
		_ = c.globalCache.Set([]byte(key), value, 0)
	}
}
