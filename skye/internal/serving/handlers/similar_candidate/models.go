package similar_candidate

import (
	pb "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"
)

type SkyeStructRequest struct {
	Entity        string
	CandidateIds  []string
	Limit         int
	ModelName     string
	Variant       string
	Filters       [][]*pb.Filter
	GlobalFilters []*pb.Filter
	Attributes    []string
	Embeddings    [][]float32
}
