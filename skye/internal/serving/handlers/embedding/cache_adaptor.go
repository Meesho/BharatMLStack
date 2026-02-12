package embedding

import (
	"strconv"
	"strings"

	pb "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories"
)

const (
	Embedding         = "e"
	DotProduct        = "dp"
	CacheVersion      = "V2"
	CacheKeySeparator = ":"
)

func GetCacheKeysForBulkEmbeddingRequest(request *pb.SkyeBulkEmbeddingRequest) map[string]repositories.CacheStruct {
	cacheKeys := make(map[string]repositories.CacheStruct, len(request.CandidateIds))
	for index, id := range request.CandidateIds {
		key := buildBulkEmbeddingCacheKey(Embedding, request.Entity, request.ModelName, id)
		if _, ok := cacheKeys[key]; ok {
			cacheStruct := cacheKeys[key]
			cacheStruct.Index = append(cacheStruct.Index, index)
			cacheKeys[key] = cacheStruct
		} else {
			cacheKeys[key] = repositories.CacheStruct{
				Index:       []int{index},
				CandidateId: id,
			}
		}
	}
	return cacheKeys
}

func GetCacheKeysForCandidateDotProductScoreRequest(request *pb.EmbeddingDotProductRequest, version int) map[string]repositories.CacheStruct {
	cacheKeys := make(map[string]repositories.CacheStruct, len(request.CandidateIds))
	for index, id := range request.CandidateIds {
		key := buildSimpleCacheKey(DotProduct, request.Entity, request.ModelName, strconv.Itoa(version), id)
		if _, ok := cacheKeys[key]; ok {
			cacheStruct := cacheKeys[key]
			cacheStruct.Index = append(cacheStruct.Index, index)
			cacheKeys[key] = cacheStruct
		} else {
			cacheKeys[key] = repositories.CacheStruct{
				Index:       []int{index},
				CandidateId: id,
			}
		}
	}
	return cacheKeys
}

func buildBulkEmbeddingCacheKey(prefix, entity, model, id string) string {
	var b strings.Builder
	b.Grow(len(prefix) + len(entity) + len(model) + len(id) + len(CacheVersion) + 20)
	b.WriteString(prefix)
	b.WriteString(CacheKeySeparator)
	b.WriteString(entity)
	b.WriteString(CacheKeySeparator)
	b.WriteString(model)
	b.WriteString(CacheKeySeparator)
	b.WriteString(id)
	b.WriteString(CacheKeySeparator)
	b.WriteString(CacheVersion)
	return b.String()
}

func buildSimpleCacheKey(prefix, entity, model, version, id string) string {
	var b strings.Builder
	// Pre-allocate capacity to avoid reallocations
	b.Grow(len(prefix) + len(entity) + len(model) + len(id) + len(CacheVersion) + len(version) + 20)
	b.WriteString(prefix)
	b.WriteString(CacheKeySeparator)
	b.WriteString(entity)
	b.WriteString(CacheKeySeparator)
	b.WriteString(model)
	b.WriteString(CacheKeySeparator)
	b.WriteString(version)
	b.WriteString(CacheKeySeparator)
	b.WriteString(id)
	b.WriteString(CacheKeySeparator)
	b.WriteString(CacheVersion)
	return b.String()
}
