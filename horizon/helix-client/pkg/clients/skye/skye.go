package skye

import (
	"github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"
)

type SkyeClient interface {
	GetSimilarCandidates(request *grpc.SkyeRequest) (*grpc.SkyeResponse, error)
	GetEmbeddingsForCandidateIds(request *grpc.SkyeBulkEmbeddingRequest) (*grpc.SkyeBulkEmbeddingResponse, error)
	GetDotProductOfCandidatesForEmbedding(request *grpc.EmbeddingDotProductRequest) (*grpc.EmbeddingDotProductResponse, error)
}
