package similar_candidate

import (
	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/vector"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	pb "github.com/Meesho/helix-clients/pkg/deployableclients/skye/client/grpc"
	"github.com/rs/zerolog/log"
	protosd "google.golang.org/protobuf/proto"
)

// Convert SkyeRequest proto to internal struct format
func adaptSkyeRequestToStruct(r *pb.SkyeRequest) SkyeStructRequest {
	var numCandidates int
	if len(r.Embeddings) > 0 {
		numCandidates = len(r.Embeddings)
	} else {
		numCandidates = len(r.CandidateIds)
	}

	// Parse candidate-level filters
	filters := make([][]*pb.Filter, numCandidates)
	for i, filter := range r.Filters {
		var fList []*pb.Filter
		for _, f := range filter.GetFilter() {
			fList = append(fList, &pb.Filter{
				Field:  f.Field,
				Op:     f.Op,
				Values: f.Values,
			})
		}
		filters[i] = fList
	}

	// Parse global filters
	var globalFilters []*pb.Filter
	if r.GlobalFilters != nil {
		for _, gf := range r.GlobalFilters.Filter {
			globalFilters = append(globalFilters, &pb.Filter{
				Field:  gf.Field,
				Op:     gf.Op,
				Values: gf.Values,
			})
		}
	}

	// Append global filters to missing candidate-level ones
	if len(globalFilters) > 0 {
		for i := 0; i < numCandidates; i++ {
			if filters[i] == nil {
				filters[i] = globalFilters
			}
		}
	}

	// Parse embeddings
	embeddings := make([][]float32, len(r.Embeddings))
	for i, emb := range r.Embeddings {
		floats := make([]float32, len(emb.Embedding))
		for j, val := range emb.Embedding {
			floats[j] = float32(val)
		}
		embeddings[i] = floats
	}

	return SkyeStructRequest{
		Entity:        r.Entity,
		CandidateIds:  r.CandidateIds,
		Limit:         int(r.Limit),
		ModelName:     r.ModelName,
		Variant:       r.Variant,
		Filters:       filters,
		GlobalFilters: globalFilters,
		Attributes:    r.Attribute,
		Embeddings:    embeddings,
	}
}

// Build vector DB batch query from raw embeddings
func buildVectorBatchQueryFromEmbeddings(request SkyeStructRequest, variantConfig *config.Variant, cacheKeys map[string]repositories.CacheStruct, responseMap map[string]repositories.CandidateResponseStruct) *vector.BatchQueryRequest {
	queries := make([]*vector.QueryDetails, 0, len(cacheKeys))
	for k, v := range cacheKeys {
		if len(v.Embedding) == 0 {
			responseMap[k] = repositories.CandidateResponseStruct{
				Index:    v.Index,
				Response: &pb.CandidateResponse{Candidates: []*pb.CandidateResponse_Candidate{}},
			}
			continue
		}
		queries = append(queries, &vector.QueryDetails{
			CacheKey:        k,
			Embedding:       v.Embedding,
			CandidateLimit:  int32(request.Limit),
			Payload:         request.Attributes,
			SearchParams:    variantConfig.VectorDbConfig.Params,
			MetadataFilters: v.Filters,
		})
	}
	return &vector.BatchQueryRequest{
		Entity:      request.Entity,
		Model:       request.ModelName,
		Variant:     request.Variant,
		Version:     variantConfig.VectorDbReadVersion,
		RequestList: queries,
	}
}

func parseVectorDbResponse(keys map[string]repositories.CacheStruct, batchResp *vector.BatchQueryResponse, responseMap map[string]repositories.CandidateResponseStruct, isEmbeddingsRequest bool) {
	for key, candidates := range batchResp.SimilarCandidatesList {
		cands := make([]*pb.CandidateResponse_Candidate, 0, len(candidates))
		for _, c := range candidates {
			if !isEmbeddingsRequest && c.Id == keys[key].CandidateId {
				continue
			}
			cands = append(cands, &pb.CandidateResponse_Candidate{
				Id: c.Id,
				Meta: &pb.CandidateResponse_Candidate_Meta{
					Score:         float64(c.Score),
					AttributesMap: c.Payload,
				},
			})
		}
		responseMap[key] = repositories.CandidateResponseStruct{
			Index:    keys[key].Index,
			Response: &pb.CandidateResponse{Candidates: cands},
		}
	}
}

// Unmarshal and decode cached byte responses to CandidateResponse structs
func ProcessCacheResponse(keys map[string]repositories.CacheStruct, cachedData map[string][]byte, resMap map[string]repositories.CandidateResponseStruct, limit int, commonMetricTags []string, cacheType string, partialHitDisabled bool) map[string]repositories.CacheStruct {
	missingCacheKeys := make(map[string]repositories.CacheStruct, len(keys))
	for k := range keys {
		if raw, ok := cachedData[k]; ok {
			resp := &pb.CandidateResponse{}
			if err := protosd.Unmarshal(raw, resp); err != nil {
				log.Error().Err(err).Msg("failed to unmarshal candidate response bytes")
				continue
			}
			if !partialHitDisabled {
				if len(resp.Candidates) > 0 && len(resp.Candidates) < limit-1 {
					missingCacheKeys[k] = keys[k]
					if cacheType == "in_memory" {
						metric.Incr("in_memory_embedding_cache_partial_miss", commonMetricTags)
					} else {
						metric.Incr("distributed_embedding_cache_partial_miss", commonMetricTags)
					}
					continue
				}
			}
			if len(resp.Candidates) > limit {
				resp.Candidates = resp.Candidates[:limit:limit]
			}
			resMap[k] = repositories.CandidateResponseStruct{
				Index:    keys[k].Index,
				Response: resp,
			}
			delete(keys, k)
			if cacheType == "in_memory" {
				metric.Incr("in_memory_embedding_cache_hit", commonMetricTags)
			} else {
				metric.Incr("distributed_embedding_cache_hit", commonMetricTags)
			}
		} else {
			missingCacheKeys[k] = keys[k]
			if cacheType == "in_memory" {
				metric.Incr("in_memory_embedding_cache_miss", commonMetricTags)
			} else {
				metric.Incr("distributed_embedding_cache_miss", commonMetricTags)
			}
		}
	}
	return missingCacheKeys
}
