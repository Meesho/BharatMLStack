package feature

import (
	"context"
	"sync"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/caches"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/provider"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/p2p"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
	"github.com/rs/zerolog/log"
)

var (
	p2pCacheOnce    sync.Once
	p2pCacheHandler *P2PCacheHandler
)

type P2PCacheHandler struct {
	p2p.P2PCacheServiceServer
	p2pCacheProvider provider.CacheProvider
}

func InitP2PCacheHandler() *P2PCacheHandler {
	p2pCacheOnce.Do(func() {
		p2pCacheHandler = &P2PCacheHandler{
			p2pCacheProvider: provider.P2PCacheProviderImpl,
		}
	})
	return p2pCacheHandler
}

func (h *P2PCacheHandler) GetClusterConfigs(ctx context.Context, query *p2p.Query) (*p2p.ClusterTopology, error) {
	p2pCache, err := h.getP2PCacheForEntity(query.EntityLabel)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting p2p cache for entity %s", query.EntityLabel)
		return nil, err
	}
	clusterTopology := p2pCache.GetClusterTopology()
	clusterMembers := make(map[string]*p2p.PodData)
	for k, v := range clusterTopology.ClusterMembers {
		clusterMembers[k] = &p2p.PodData{
			PodIp:  v.PodIP,
			NodeIp: v.NodeIP,
		}
	}
	return &p2p.ClusterTopology{
		RingTopology:   clusterTopology.RingTopology,
		ClusterMembers: clusterMembers,
	}, nil
}

func (h *P2PCacheHandler) GetP2PCacheValues(ctx context.Context, query *p2p.CacheQuery) (*p2p.CacheKeyValue, error) {
	p2pCache, err := h.getP2PCacheForEntity(query.EntityLabel)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting p2p cache for entity %s", query.EntityLabel)
		return nil, err
	}
	keys := make([]*retrieve.Keys, 0)
	for _, key := range query.Keys {
		keys = append(keys, &retrieve.Keys{
			Cols: []string{key},
		})
	}
	response, err := p2pCache.MultiGetV2(query.EntityLabel, keys)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting p2p cache values for entity %s", query.EntityLabel)
		return nil, err
	}
	responseMap := make(map[string]string)
	for i, key := range query.Keys {
		responseMap[key] = string(response[i])
	}
	return &p2p.CacheKeyValue{
		EntityLabel: query.EntityLabel,
		KeyValue:    responseMap,
	}, nil
}

func (h *P2PCacheHandler) SetP2PCacheValues(ctx context.Context, query *p2p.CacheKeyValue) (*p2p.CacheKeyValue, error) {
	p2pCache, err := h.getP2PCacheForEntity(query.EntityLabel)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting p2p cache for entity %s", query.EntityLabel)
		return nil, err
	}
	keys := make([]*retrieve.Keys, 0)
	for key := range query.KeyValue {
		keys = append(keys, &retrieve.Keys{
			Cols: []string{key},
		})
	}
	values := make([][]byte, 0)
	for _, key := range keys {
		values = append(values, []byte(query.KeyValue[key.Cols[0]]))
	}
	p2pCache.MultiSetV2(query.EntityLabel, keys, values)
	return &p2p.CacheKeyValue{
		EntityLabel: query.EntityLabel,
		KeyValue:    query.KeyValue,
	}, nil
}

func (h *P2PCacheHandler) getP2PCacheForEntity(entityLabel string) (*caches.P2PCache, error) {
	cache, err := h.p2pCacheProvider.GetCache(entityLabel)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting p2p cache for entity %s", entityLabel)
		return nil, err
	}
	p2pCache, ok := cache.(*caches.P2PCache)
	if !ok {
		log.Error().Msgf("Failed to typecast cache to *caches.P2PCache for entity %s", entityLabel)
		return nil, err
	}
	return p2pCache, nil
}
