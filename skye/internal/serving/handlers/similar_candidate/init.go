package similar_candidate

import (
	"sync"

	"github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"
	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/distributedcache"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/embedding"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/inmemorycache"
)

var (
	once      sync.Once
	handlerV1 HandlerV1
)

func GetHandler(version int8) grpc.SkyeSimilarCandidateServiceServer {
	switch version {
	case 1:
		return InitV1()
	default:
		return nil
	}
}

// SetMockSimilarCandidateHandler creates the mock handler instance with mock databases
// This would be handy in places where we want to create a handler with specific database instances
func SetMockSimilarCandidateHandler(embeddingStore embedding.Store, configManager config.Manager,
	inMemCache inmemorycache.Database, distributedCache distributedcache.Database) *HandlerV1 {
	once.Do(func() {})
	handlerV1 = HandlerV1{
		configManager:    configManager,
		inMemCache:       inMemCache,
		distributedCache: distributedCache,
		embeddingStore:   embeddingStore,
	}
	return &handlerV1
}
