package distributedcache

import (
	pb "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"
	"github.com/stretchr/testify/mock"
)

type MockDistributedCacheClient struct {
	mock.Mock
}

func (m *MockDistributedCacheClient) GetCacheKeysForEmbeddings(entity string, model string, variant string, embeddings [][]float32, filters [][]*pb.Filter, attributes []string) ([]string, map[string][]float32) {
	args := m.Called(entity, model, variant, embeddings, filters, attributes)
	if args.Get(0) == nil {
		return make([]string, 0), nil
	}
	return args.Get(0).([]string), nil
}

func (m *MockDistributedCacheClient) GetCacheKeysForCandidateIds(entity string, model string, variant string, ids []string, filters [][]*pb.Filter, attributesMap []string) ([]string, map[string]string) {
	args := m.Called(entity, model, variant, ids, filters, attributesMap)
	if args.Get(0) == nil {
		return make([]string, 0), nil
	}
	return args.Get(0).([]string), nil
}

func (m *MockDistributedCacheClient) GetCacheKeysForBulkEmbeddingRequest(entity string, model string, candidateIds []string) ([]string, map[string]string, map[string]string) {
	args := m.Called(entity, model, candidateIds)
	if args.Get(0) == nil {
		return make([]string, 0), nil, nil
	}
	return args.Get(0).([]string), nil, nil
}

func (m *MockDistributedCacheClient) GetCacheKeysForCandidateDotProductScoreRequest(entity string, model string, candidateIds []string) ([]string, map[string]string, map[string]string) {
	args := m.Called(entity, model, candidateIds)
	if args.Get(0) == nil {
		return make([]string, 0), nil, nil
	}
	return args.Get(0).([]string), nil, nil
}

func (m *MockDistributedCacheClient) MGet(keys []string, tags *[]string) map[string][]byte {
	args := m.Called(keys, tags)
	return args.Get(0).(map[string][]byte)
}

func (m *MockDistributedCacheClient) MSet(data *map[string]interface{}, ttl int, tags *[]string) {
	m.Called(data, ttl, tags)
}
