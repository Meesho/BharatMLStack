package caches

import (
	"fmt"

	"github.com/Meesho/online-feature-store/internal/config"
	"github.com/Meesho/online-feature-store/pkg/infra"
	"github.com/Meesho/online-feature-store/pkg/p2pcache"
	"github.com/Meesho/online-feature-store/pkg/proto/retrieve"
)

type P2PCache struct {
	cache     *p2pcache.P2PCache
	cacheName string
	config    config.Manager
}

func NewP2PCache(conn *infra.P2PCacheConnection) (Cache, error) {
	meta, err := conn.GetMeta()
	if err != nil {
		return nil, err
	}
	configManager := config.Instance(config.DefaultVersion)
	return &P2PCache{
		cache:     conn.Client,
		cacheName: meta["name"].(string),
		config:    configManager,
	}, nil
}

func (c *P2PCache) GetV2(entityLabel string, keys *retrieve.Keys) []byte {
	k := buildCacheKeyForRetrieve(keys, entityLabel)
	kvMap, err := c.cache.MultiGet([]string{k})
	if err != nil {
		return nil
	}
	if v, ok := kvMap[k]; ok {
		return []byte(v)
	}
	return nil
}

func (c *P2PCache) MultiGetV2(entityLabel string, bulkKeys []*retrieve.Keys) ([][]byte, error) {
	cacheKeys := make([]string, len(bulkKeys))
	for i, keys := range bulkKeys {
		cacheKeys[i] = buildCacheKeyForRetrieve(keys, entityLabel)
	}
	kvMap, err := c.cache.MultiGet(cacheKeys)
	if err != nil {
		return nil, err
	}
	cacheData := make([][]byte, len(bulkKeys))
	for i := range bulkKeys {
		if v, ok := kvMap[cacheKeys[i]]; ok {
			cacheData[i] = []byte(v)
		}
	}
	return cacheData, nil
}

func (c *P2PCache) Delete(entityLabel string, key []string) error {
	k := buildCacheKeyForPersist(key, entityLabel)
	return c.cache.MultiDelete([]string{k})
}

func (c *P2PCache) SetV2(entityLabel string, keys []string, data []byte) error {
	k := buildCacheKeyForPersist(keys, entityLabel)
	cacheConfig, err := c.config.GetP2PCacheConfForEntity(entityLabel)
	if err != nil {
		return err
	}
	return c.cache.MultiSet(map[string][]byte{k: data}, cacheConfig.TtlInSeconds)
}

func (c *P2PCache) MultiSetV2(entityLabel string, bulkKeys []*retrieve.Keys, bulkData [][]byte) error {
	if len(bulkKeys) == 0 || len(bulkData) == 0 {
		return fmt.Errorf("%w: keys or values length is zero", ErrInvalidInput)
	}
	if len(bulkKeys) != len(bulkData) {
		return fmt.Errorf("%w: keys and values length mismatch", ErrInvalidInput)
	}
	cacheConfig, err := c.config.GetP2PCacheConfForEntity(entityLabel)
	if err != nil {
		return err
	}
	kvMap := make(map[string][]byte)
	for i, key := range bulkKeys {
		kvMap[buildCacheKeyForPersist(key.Cols, entityLabel)] = bulkData[i]
	}
	return c.cache.MultiSet(kvMap, cacheConfig.TtlInSeconds)
}
