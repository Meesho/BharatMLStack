package similar_candidate

import (
	"github.com/Meesho/helix-clients/pkg/deployableclients/skye/client/grpc"
)

func validateSkyeRequest(request *grpc.SkyeRequest) (bool, string) {
	// Cache length calculations to avoid repeated calls
	candidateIdsLen := len(request.CandidateIds)
	embeddingsLen := len(request.Embeddings)
	filtersLen := len(request.Filters)

	// entity should not be empty
	if len(request.Entity) == 0 {
		return false, "Entity is required"
	}

	// model name is required
	if len(request.ModelName) == 0 {
		return false, "modelName is required"
	}

	// variant is required
	if len(request.Variant) == 0 {
		return false, "variant is required"
	}

	// limit should be a positive number
	if request.Limit <= 0 {
		return false, "limit is required"
	}

	// candidateIds and embeddings both should not be empty
	if candidateIdsLen == 0 && embeddingsLen == 0 {
		return false, "candidateIds or embeddings is required"
	}

	// both candidateIds and embeddings should not be present
	if candidateIdsLen > 0 && embeddingsLen > 0 {
		return false, "both candidateIds and embeddings are present, only one is required"
	}

	// either globalFilter or filters should be used, not both
	// Replace expensive reflection with simple nil check and field access
	hasGlobalFilters := request.GlobalFilters != nil && len(request.GlobalFilters.Filter) > 0
	if filtersLen > 0 && hasGlobalFilters {
		return false, "either globalFilter or filters should be used, not both"
	}

	// if filters are used then filter should be present for every candidateId/embedding
	if filtersLen > 0 {
		if candidateIdsLen > 0 && candidateIdsLen != filtersLen {
			return false, "filter should be present for each candidateId"
		}
		if embeddingsLen > 0 && embeddingsLen != filtersLen {
			return false, "filter should be present for each embedding"
		}
	}

	return true, ""
}
