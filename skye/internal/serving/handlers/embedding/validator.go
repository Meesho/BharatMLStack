package embedding

import "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"

func isValidateEmbeddingsRequest(req *grpc.SkyeBulkEmbeddingRequest) (bool, string) {
	if req.Entity == "" {
		return false, "Entity is Required"
	}
	if req.ModelName == "" {
		return false, "ModelName is required"
	}
	if req.Variant == "" {
		return false, "Variant is required"
	}
	if len(req.CandidateIds) <= 0 {
		return false, "candidateIds are required"
	}
	return true, ""
}

func isValidDotProductRequest(req *grpc.EmbeddingDotProductRequest) (bool, string) {
	if req.GetEntity() == "" {
		return false, "Entity is required"
	}
	if req.GetModelName() == "" {
		return false, "ModelName is required"
	}
	if req.GetVariant() == "" {
		return false, "Variant is required"
	}
	if len(req.GetEmbedding()) == 0 {
		return false, "embedding is required"
	}
	if len(req.GetCandidateIds()) == 0 {
		return false, "at least one candidateId is required"
	}
	return true, ""
}
