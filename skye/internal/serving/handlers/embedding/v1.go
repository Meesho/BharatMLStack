package embedding

import (
	"context"
	"sync"
	"time"

	pb "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"
	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/distributedcache"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/embedding"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/inmemorycache"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	v1        *HandlerV1
	once      sync.Once
	appConfig structs.Configs
)

type HandlerV1 struct {
	pb.SkyeEmbeddingServiceServer
	configManager   config.Manager
	embeddingStore  embedding.Store
	inMemoryCache   inmemorycache.Database
	distributeCache distributedcache.Database
}

const (
	RequestTypeEmbeddingRetrieval = "embedding_retrieval"
	RequestTypeDotProduct         = "dot_product"
)

func InitV1Handler() pb.SkyeEmbeddingServiceServer {
	if v1 == nil {
		once.Do(func() {
			appConfig = structs.GetAppConfig().Configs
			v1 = &HandlerV1{
				configManager:   config.NewManager(config.DefaultVersion),
				embeddingStore:  embedding.NewRepository(embedding.DefaultVersion),
				inMemoryCache:   inmemorycache.NewRepository(inmemorycache.DefaultVersion),
				distributeCache: distributedcache.NewRepository(distributedcache.DefaultVersion),
			}
		})
	}
	return v1
}

func (h HandlerV1) GetEmbeddingsForCandidates(ctx context.Context, request *pb.SkyeBulkEmbeddingRequest) (*pb.SkyeBulkEmbeddingResponse, error) {
	start := time.Now()

	if appConfig.AppEnv == "staging" {
		modifyStagingRetrievalRequest(request)
	}

	requestType := RequestTypeEmbeddingRetrieval
	commonMetricTags := getTags(request.Entity, request.ModelName, request.Variant, requestType)

	metric.Incr("bulk_embedding_request", commonMetricTags)
	log.Debug().Msgf("GetEmbeddingsRequest: %+v", request)

	entityConfig, err := h.configManager.GetEntityConfig(request.Entity)
	if err != nil {
		metric.Incr("bulk_embedding_request_5xx", commonMetricTags)
		log.Error().Msgf("BulkEmbedding Request Failed: Error getting store ID for entity %s: %v", request.Entity, err)
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	variantConfig, err := h.configManager.GetVariantConfig(request.Entity, request.ModelName, request.Variant)
	if err != nil {
		metric.Incr("bulk_embedding_request_5xx", commonMetricTags)
		log.Error().Msgf("BulkEmbedding Request Failed: Error getting variant config for entity %s, model %s, variant %s: %v", request.Entity, request.ModelName, request.Variant, err)
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	if !variantConfig.Enabled {
		return nil, status.Errorf(codes.NotFound, "variant %s is not enabled for entity %s, model %s", request.Variant, request.Entity, request.ModelName)
	}
	if ok, err := isValidateEmbeddingsRequest(request); !ok {
		metric.Incr("bulk_embedding_request_4xx", commonMetricTags)
		log.Debug().Msgf("BulkEmbedding Request Failed: Invalid request body, validation failed at %s", err)
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	expectedLength := len(request.CandidateIds)

	// Stores embeddings by cache key
	responseMap := make(map[string]repositories.CandidateResponseStruct)

	// Derive cache keys
	cacheKeys := GetCacheKeysForBulkEmbeddingRequest(request)

	numberOfCacheKeys := len(cacheKeys)
	if numberOfCacheKeys != expectedLength {
		log.Error().Msgf("BulkEmbedding Request Failed: cacheKeys length mismatch for request: %d != %d", numberOfCacheKeys, expectedLength)
		return nil, status.Errorf(codes.Internal, "cacheKeys length mismatch")
	}
	// 1. In-Memory Cache Lookup
	missingInMemoryCacheKeys := make(map[string]repositories.CacheStruct, len(cacheKeys))

	// Trigger InMemory Mget
	if variantConfig.EmbeddingRetrievalInMemoryConfig.Enabled {
		inMemResponseMap := h.inMemoryCache.MGet(cacheKeys, commonMetricTags)
		missingInMemoryCacheKeys = processCacheResponse(cacheKeys, inMemResponseMap, responseMap, commonMetricTags, "in_memory")
		log.Info().Msgf("BulkEmbedding Request Failed: missingInMemoryCacheKeys: %+v", missingInMemoryCacheKeys)
	}

	missingDistributedCacheKeys := make(map[string]repositories.CacheStruct, len(cacheKeys))
	// Trigger Distributed Mget
	if len(cacheKeys) > 0 && variantConfig.EmbeddingRetrievalDistributedConfig.Enabled {
		distResponseMap, err := h.distributeCache.MGet(cacheKeys, commonMetricTags)
		if err != nil {
			log.Error().Msgf("BulkEmbedding Request Failed: Error fetching embeddings from distributed cache, err: %v", err)
			metric.Incr("bulk_embedding_request_5xx", commonMetricTags)
			return nil, status.Errorf(codes.Internal, "error fetching embeddings from distributed cache, err: %v", err)
		}
		missingDistributedCacheKeys = processCacheResponse(cacheKeys, distResponseMap, responseMap, commonMetricTags, "distributed")
		log.Info().Msgf("BulkEmbedding Request Failed: missingDistributedCacheKeys: %+v", missingDistributedCacheKeys)
	}

	if len(cacheKeys) > 0 {
		err := h.embeddingStore.BulkQuery(entityConfig.StoreId, &embedding.BulkQuery{
			CacheKeys: cacheKeys,
			Model:     request.ModelName,
			Variant:   request.Variant,
			Version:   variantConfig.EmbeddingStoreReadVersion,
		}, "embedding")
		if err != nil {
			metric.Incr("bulk_embedding_request_5xx", commonMetricTags)
			log.Error().Msgf("BulkEmbedding Request Failed: Error fetching embeddings from store")
			return nil, status.Errorf(codes.Internal, "error fetching embeddings from store")
		}
		parseEmbeddingResponse(responseMap, cacheKeys)
	}

	// Cache fresh results
	go func() {
		byteResponseMap := make(map[string][]byte, len(missingDistributedCacheKeys))
		if variantConfig.EmbeddingRetrievalDistributedConfig.Enabled && len(missingDistributedCacheKeys) > 0 {
			h.distributeCache.MSet(responseMap, missingDistributedCacheKeys, variantConfig.DistributedCacheTTLSeconds, byteResponseMap, commonMetricTags)
		}
		if variantConfig.EmbeddingRetrievalInMemoryConfig.Enabled && len(missingInMemoryCacheKeys) > 0 {
			h.inMemoryCache.MSet(responseMap, missingInMemoryCacheKeys, variantConfig.InMemoryCacheTTLSeconds, byteResponseMap, commonMetricTags)
		}
	}()
	metric.Timing("bulk_embedding_request_latency", time.Since(start), commonMetricTags)
	return &pb.SkyeBulkEmbeddingResponse{
		CandidatesEmbedding: generateResponseEmbedding(responseMap, numberOfCacheKeys),
	}, nil
}

func (h HandlerV1) GetCandidateEmbeddingScores(
	ctx context.Context,
	request *pb.EmbeddingDotProductRequest,
) (*pb.EmbeddingDotProductResponse, error) {
	startTime := time.Now()

	if appConfig.AppEnv == "staging" {
		modifyStagingDotProductRequest(request)
	}

	requestType := RequestTypeDotProduct
	commonTags := getTags(request.Entity, request.ModelName, request.Variant, requestType)
	log.Debug().Msgf("DotProductRequest: %+v", request)
	metric.Incr("dot_product_request", commonTags)

	entityConfig, err := h.configManager.GetEntityConfig(request.Entity)
	if err != nil {
		metric.Incr("dot_product_request_4xx", commonTags)
		log.Error().Msgf("DotProduct Request Failed: Error getting entity config for %s: %v", request.Entity, err)
		return nil, status.Errorf(codes.InvalidArgument, "%s", err.Error())
	}
	variantConfig, err := h.configManager.GetVariantConfig(request.Entity, request.ModelName, request.Variant)
	if err != nil {
		metric.Incr("dot_product_request_4xx", commonTags)
		log.Error().Msgf("DotProduct Request Failed: Error getting variant config for entity %s, model %s, variant %s: %v", request.Entity, request.ModelName, request.Variant, err)
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	if !variantConfig.Enabled {
		return nil, status.Errorf(codes.NotFound, "variant %s is not enabled for entity %s, model %s", request.Variant, request.Entity, request.ModelName)
	}
	validationErr, errMsg := isValidDotProductRequest(request)
	if !validationErr {
		metric.Incr("dot_product_request_4xx", commonTags)
		log.Debug().Msgf("DotProduct Request Failed: Invalid request body, validation failed at %s", errMsg)
		return nil, status.Errorf(codes.InvalidArgument, "%s", errMsg)
	}
	expectedLength := len(request.CandidateIds)
	// Stores embeddings by cache key
	responseMap := make(map[string]repositories.CandidateResponseStruct)
	foundcacheKeys := make(map[string]repositories.CacheStruct)
	cacheKeys := GetCacheKeysForCandidateDotProductScoreRequest(request, variantConfig.EmbeddingStoreReadVersion)
	missingInMemoryCacheKeys := make(map[string]repositories.CacheStruct, len(cacheKeys))

	if variantConfig.DotProductInMemoryConfig.Enabled {
		inMemResponseMap := h.inMemoryCache.MGet(cacheKeys, commonTags)
		missingInMemoryCacheKeys = processCacheResponseDotProduct(cacheKeys, request.Embedding, inMemResponseMap, responseMap, commonTags, "in_memory", foundcacheKeys)
	}

	missingDistributedCacheKeys := make(map[string]repositories.CacheStruct, len(cacheKeys))
	if len(cacheKeys) > 0 && variantConfig.DotProductDistributedConfig.Enabled {
		distResponseMap, err := h.distributeCache.MGet(cacheKeys, commonTags)
		if err != nil {
			log.Error().Msgf("DotProduct Request Failed: Error fetching embeddings from distributed cache, err: %v", err)
			metric.Incr("dot_product_request_5xx", commonTags)
			return nil, status.Errorf(codes.Internal, "error fetching embeddings from distributed cache, err: %v", err)
		}
		missingDistributedCacheKeys = processCacheResponseDotProduct(cacheKeys, request.Embedding, distResponseMap, responseMap, commonTags, "distributed", foundcacheKeys)
	}

	if len(cacheKeys) > 0 {
		err := h.embeddingStore.BulkQuery(entityConfig.StoreId, &embedding.BulkQuery{
			CacheKeys: cacheKeys,
			Model:     request.ModelName,
			Variant:   request.Variant,
			Version:   variantConfig.EmbeddingStoreReadVersion,
		}, "dot_product")
		if err != nil {
			metric.Incr("dot_product_request_5xx", commonTags)
			log.Error().Msgf("DotProduct Request Failed: Error fetching embeddings from store")
			return nil, status.Errorf(codes.Internal, "error fetching embeddings from store")
		}

		// Step 5: Compute dot products
		for key, cacheStruct := range cacheKeys {
			score := CalculateDotProduct(request.Embedding, cacheStruct.Embedding)
			responseMap[key] = repositories.CandidateResponseStruct{
				Index: cacheKeys[key].Index,
				DotProductResponse: &pb.CandidateEmbeddingScore{
					Score:       score,
					CandidateId: cacheStruct.CandidateId,
				},
			}
		}
	}

	// Step 6: Backfill cache
	go func() {
		if variantConfig.DotProductDistributedConfig.Enabled && len(missingDistributedCacheKeys) > 0 {
			h.distributeCache.MSetDotProduct(cacheKeys, foundcacheKeys, missingDistributedCacheKeys, variantConfig.DistributedCacheTTLSeconds, commonTags)
		}
		if variantConfig.DotProductInMemoryConfig.Enabled && len(missingInMemoryCacheKeys) > 0 {
			h.inMemoryCache.MSetDotProduct(cacheKeys, foundcacheKeys, missingInMemoryCacheKeys, variantConfig.InMemoryCacheTTLSeconds, commonTags)
		}
	}()

	// Step 7: Return result
	metric.Timing("dot_product_request_latency", time.Since(startTime), commonTags)
	return &pb.EmbeddingDotProductResponse{
		CandidateEmbeddingScores: generateResponseDotProduct(responseMap, expectedLength),
	}, nil
}

func generateResponseEmbedding(responseMap map[string]repositories.CandidateResponseStruct, expectedLength int) []*pb.CandidateEmbedding {
	candidateResponse := make([]*pb.CandidateEmbedding, expectedLength)
	for _, v := range responseMap {
		for _, index := range v.Index {
			candidateResponse[index] = v.EmbeddingResponse
		}
	}
	return candidateResponse
}

func generateResponseDotProduct(responseMap map[string]repositories.CandidateResponseStruct, expectedLength int) []*pb.CandidateEmbeddingScore {
	candidateResponse := make([]*pb.CandidateEmbeddingScore, expectedLength)
	for _, v := range responseMap {
		for _, index := range v.Index {
			candidateResponse[index] = v.DotProductResponse
		}
	}
	return candidateResponse
}

func CalculateDotProduct(embedding1 []float64, embedding2 []float32) float64 {
	if len(embedding1) != len(embedding2) {
		return 0
	}
	var dotProduct float32
	f32 := make([]float32, len(embedding1))
	for i, v := range embedding1 {
		f32[i] = float32(v)
	}
	for i := range f32 {
		dotProduct += f32[i] * embedding2[i]
	}
	return float64(dotProduct)
}

func getTags(entity, modelName, variant, requestType string) []string {
	return []string{"entity", entity, "model_name", modelName, "variant", variant, "request_type", requestType}
}

func modifyStagingRetrievalRequest(request *pb.SkyeBulkEmbeddingRequest) {
	//replace requested model name with default model name
	request.ModelName = appConfig.StagingDefaultModelName

	//replace requested variant with staging default variant
	request.Variant = appConfig.StagingDefaultVariant

	//replace requested entity with staging default entity
	request.Entity = appConfig.StagingDefaultEntity
}

func modifyStagingDotProductRequest(request *pb.EmbeddingDotProductRequest) {
	//replace requested model name with default model name
	request.ModelName = appConfig.StagingDefaultModelName

	//replace requested variant with staging default variant
	request.Variant = appConfig.StagingDefaultVariant

	//replace requested entity with staging default entity
	request.Entity = appConfig.StagingDefaultEntity

	//if embedding exists, pad or truncate it to make it equal to default embedding length
	if len(request.Embedding) > 0 {
		if len(request.Embedding) > appConfig.StagingDefaultEmbeddingLength {
			request.Embedding = request.Embedding[:appConfig.StagingDefaultEmbeddingLength]
		} else {
			request.Embedding = append(request.Embedding, make([]float64, appConfig.StagingDefaultEmbeddingLength-len(request.Embedding))...)
		}
	}
}
