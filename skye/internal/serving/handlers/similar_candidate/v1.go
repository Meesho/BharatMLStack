package similar_candidate

import (
	"context"
	"math/rand"
	"time"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/distributedcache"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/embedding"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/inmemorycache"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/vector"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	pb "github.com/Meesho/helix-clients/pkg/deployableclients/skye/client/grpc"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HandlerV1 struct {
	pb.SkyeSimilarCandidateServiceServer
	embeddingStore   embedding.Store
	configManager    config.Manager
	inMemCache       inmemorycache.Database
	distributedCache distributedcache.Database
}

type RequestType string

const (
	RequestTypeEmbeddings       RequestType = "embeddings"
	RequestTypeCandidateIDs     RequestType = "candidateIds"
	RequestTypeSimilarCandidate             = "similar_candidate"
)

var appConfig structs.Configs

func InitV1() pb.SkyeSimilarCandidateServiceServer {
	if (HandlerV1{}) == handlerV1 {
		once.Do(func() {
			appConfig = structs.GetAppConfig().Configs
			handlerV1 = HandlerV1{
				embeddingStore:   embedding.NewRepository(embedding.DefaultVersion),
				configManager:    config.NewManager(config.DefaultVersion),
				inMemCache:       inmemorycache.NewRepository(inmemorycache.DefaultVersion),
				distributedCache: distributedcache.NewRepository(distributedcache.DefaultVersion),
			}
		})
	}
	return &handlerV1
}

func (h *HandlerV1) GetSimilarCandidates(ctx context.Context, request *pb.SkyeRequest) (*pb.SkyeResponse, error) {
	startTime := time.Now()

	if appConfig.AppEnv == "staging" {
		modifyStagingSimilarCandidatesRequest(request)
	}

	commonMetricTags := getTags(request, RequestTypeSimilarCandidate)
	metric.Incr("similar_candidate_request", commonMetricTags)
	metric.Gauge("similar_candidate_request_limit", float64(request.Limit), commonMetricTags)
	// Fetch Entity Config and Vector DB Version
	log.Debug().Msgf("SimilarCandidateRequest: %+v", request)
	entityConfig, err := h.configManager.GetEntityConfig(request.Entity)
	if err != nil {
		metric.Incr("similar_candidate_request_5xx", commonMetricTags)
		log.Error().Msgf("SimilarCandidate Request Failed: Error getting store ID for entity %s: %v", request.Entity, err)
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	variantConfig, err := h.configManager.GetVariantConfig(request.Entity, request.ModelName, request.Variant)
	if variantConfig.ReqEmbLoggingPercentage > 0 && rand.Intn(101) < variantConfig.ReqEmbLoggingPercentage {
		log.Error().Msgf("SimilarCandidate Request Logging: %+s", request.Embeddings)
	}
	if err != nil {
		metric.Incr("similar_candidate_request_5xx", commonMetricTags)
		log.Error().Msgf("SimilarCandidate Request Failed: Error getting variant config for entity %s, model %s, variant %s: %v", request.Entity, request.ModelName, request.Variant, err)
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	if !variantConfig.Enabled {
		return nil, status.Errorf(codes.NotFound, "variant %s is not enabled for entity %s, model %s", request.Variant, request.Entity, request.ModelName)
	}

	adaptedRequest := adaptSkyeRequestToStruct(request)

	isValid, msg := validateSkyeRequest(request)
	if !isValid {
		metric.Incr("similar_candidate_request_4xx", commonMetricTags)
		log.Debug().Msgf("SimilarCandidate Request Failed: Invalid request body, validation failed at %s", msg)
		return nil, status.Errorf(codes.InvalidArgument, "%s", msg)
	}
	// Return default response if configured
	if variantConfig.DefaultResponsePercentage > 0 && rand.Intn(101) < variantConfig.DefaultResponsePercentage {
		metric.Incr("similar_candidate_default_response", commonMetricTags)
		return &pb.SkyeResponse{Responses: make([]*pb.CandidateResponse, 0)}, nil
	}

	// Route to appropriate handler based on request
	response, err := h.RetrieveSimilarCandidates(adaptedRequest, entityConfig, variantConfig, commonMetricTags)
	if err != nil {
		metric.Incr("similar_candidate_request_5xx", commonMetricTags)
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	metric.Timing("similar_candidate_latency", time.Since(startTime), commonMetricTags)
	return response, nil
}

func (h *HandlerV1) RetrieveSimilarCandidates(request SkyeStructRequest, entityConfig *config.Models, variantConfig *config.Variant, commonMetricTags []string) (*pb.SkyeResponse, error) {
	isEmbeddingsRequest := len(request.Embeddings) > 0
	// Create Cache Keys
	var cacheKeys map[string]repositories.CacheStruct
	var expectedLength int
	if isEmbeddingsRequest {
		cacheKeys = GetCacheKeysForEmbeddings(request)
		expectedLength = len(request.Embeddings)
	} else {
		cacheKeys = GetCacheKeysForCandidateIds(request)
		expectedLength = len(request.CandidateIds)
	}
	responseMap := make(map[string]repositories.CandidateResponseStruct)
	missingInMemoryCacheKeys := make(map[string]repositories.CacheStruct, len(cacheKeys))

	// Trigger InMemory Mget
	if variantConfig.InMemoryCachingEnabled {
		inMemResponseMap := h.inMemCache.MGet(cacheKeys, commonMetricTags)
		missingInMemoryCacheKeys = ProcessCacheResponse(cacheKeys, inMemResponseMap, responseMap, request.Limit, commonMetricTags, "in_memory", variantConfig.PartialHitDisabled)
		log.Info().Msgf("missingInMemoryCacheKeys: %+v", missingInMemoryCacheKeys)
	}

	missingDistributedCacheKeys := make(map[string]repositories.CacheStruct, len(cacheKeys))
	// Trigger Distributed Mget
	if len(cacheKeys) > 0 && variantConfig.DistributedCachingEnabled {
		distResponseMap, err := h.distributedCache.MGet(cacheKeys, commonMetricTags)
		if err != nil {
			log.Error().Msgf("SimilarCandidate Request Failed: Error fetching similar candidates from distributed cache, err: %v", err)
			return nil, status.Errorf(codes.Internal, "error fetching similar candidates from distributed cache, err: %v", err)
		}
		missingDistributedCacheKeys = ProcessCacheResponse(cacheKeys, distResponseMap, responseMap, request.Limit, commonMetricTags, "distributed", variantConfig.PartialHitDisabled)
		log.Info().Msgf("missingDistributedCacheKeys: %+v", missingDistributedCacheKeys)
	}

	if len(cacheKeys) > 0 {
		if !isEmbeddingsRequest {
			err := h.embeddingStore.BulkQuery(entityConfig.StoreId, &embedding.BulkQuery{
				CacheKeys: cacheKeys,
				Model:     request.ModelName,
				Variant:   request.Variant,
				Version:   variantConfig.EmbeddingStoreReadVersion,
			}, "similar_candidate")
			if err != nil {
				log.Error().Msgf("SimilarCandidate Request Failed: Error fetching embeddings from store, err: %v", err)
				return nil, status.Errorf(codes.Internal, "error fetching embeddings from store, err: %v", err)
			}
		}
		batchQueryRequest := buildVectorBatchQueryFromEmbeddings(request, variantConfig, cacheKeys, responseMap)
		batchQueryResponse, err := vector.GetRepository(variantConfig.VectorDbType).BatchQuery(batchQueryRequest, commonMetricTags)
		if err != nil {
			log.Error().Err(err).Msgf("SimilarCandidate Request Failed: Error fetching similar candidates from vectorDB: %s", variantConfig.VectorDbType)
			return nil, status.Errorf(codes.Internal, "Unexpected Error fetching similar candidates from vectorDB %v %v", variantConfig.VectorDbType, err)
		}

		// Fire test query asynchronously if configured
		if variantConfig.TestConfig.Percentage > 0 && rand.Intn(101) < variantConfig.TestConfig.Percentage {
			batchQueryRequest.Entity = variantConfig.TestConfig.Entity
			batchQueryRequest.Model = variantConfig.TestConfig.Model
			batchQueryRequest.Variant = variantConfig.TestConfig.Variant
			batchQueryRequest.Version = variantConfig.TestConfig.Version
			go vector.GetRepository(variantConfig.TestConfig.VectorDbType).BatchQuery(batchQueryRequest, commonMetricTags)
		}
		parseVectorDbResponse(cacheKeys, batchQueryResponse, responseMap, isEmbeddingsRequest)
	}

	// // Cache fresh results
	go func() {
		byteResponseMap := make(map[string][]byte, len(missingDistributedCacheKeys))
		if variantConfig.DistributedCachingEnabled && len(missingDistributedCacheKeys) > 0 {
			h.distributedCache.MSet(responseMap, missingDistributedCacheKeys, variantConfig.DistributedCacheTTLSeconds, byteResponseMap, commonMetricTags)
		}
		if variantConfig.InMemoryCachingEnabled && len(missingInMemoryCacheKeys) > 0 {
			h.inMemCache.MSet(responseMap, missingInMemoryCacheKeys, variantConfig.InMemoryCacheTTLSeconds, byteResponseMap, commonMetricTags)
		}
	}()

	return &pb.SkyeResponse{Responses: generateResponse(responseMap, expectedLength)}, nil
}

func generateResponse(responseMap map[string]repositories.CandidateResponseStruct, expectedLength int) []*pb.CandidateResponse {
	candidateResponse := make([]*pb.CandidateResponse, expectedLength)
	for _, v := range responseMap {
		for _, index := range v.Index {
			candidateResponse[index] = v.Response
		}
	}
	return candidateResponse
}

func getTags(r *pb.SkyeRequest, requestType string) []string {
	return []string{"entity", r.Entity, "model_name", r.ModelName, "variant", r.Variant, "request_type", requestType}
}

func modifyStagingSimilarCandidatesRequest(request *pb.SkyeRequest) {
	//replace requested model name with default model name
	request.ModelName = appConfig.StagingDefaultModelName

	//replace requested variant with staging default variant
	request.Variant = appConfig.StagingDefaultVariant

	//replace requested entity with staging default entity
	request.Entity = appConfig.StagingDefaultEntity

	//if embeddings exist, pad or truncate them to make them equal to default embedding length
	if len(request.Embeddings) > 0 {
		for _, embedding := range request.Embeddings {
			if len(embedding.Embedding) > appConfig.StagingDefaultEmbeddingLength {
				embedding.Embedding = embedding.Embedding[:appConfig.StagingDefaultEmbeddingLength]
			} else {
				embedding.Embedding = append(embedding.Embedding, make([]float64, appConfig.StagingDefaultEmbeddingLength-len(embedding.Embedding))...)
			}
		}
	}

}
