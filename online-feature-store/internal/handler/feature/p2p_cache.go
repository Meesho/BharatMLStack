package feature

import (
	"context"
	"sync"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/p2pcache"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/p2p"
)

var (
	p2pCacheOnce    sync.Once
	p2pCacheHandler *P2PCacheHandler
)

type P2PCacheHandler struct {
	p2p.P2PCacheServiceServer
	p2PCache *p2pcache.P2PCache
}

func InitP2PCacheHandler() *P2PCacheHandler {
	p2pCacheOnce.Do(func() {
		p2pCacheHandler = &P2PCacheHandler{
			p2PCache: p2pcache.NewP2PCache("test-p2p", 100, 10),
		}
	})
	return p2pCacheHandler
}

func (h *P2PCacheHandler) GetClusterConfigs(ctx context.Context, query *p2p.Empty) (*p2p.ClusterTopology, error) {
	clusterTopology := h.p2PCache.GetClusterTopology()
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
