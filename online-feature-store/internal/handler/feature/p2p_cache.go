package feature

import (
	"context"
	"sync"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/caches"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/provider"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/p2p"
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
	cache, err := h.p2pCacheProvider.GetCache(query.EntityLabel)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting p2p cache for entity %s", query.EntityLabel)
		return nil, err
	}
	p2pCache, ok := cache.(*caches.P2PCache)
	if !ok {
		log.Error().Msgf("Failed to typecast cache to *caches.P2PCache for entity %s", query.EntityLabel)
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
