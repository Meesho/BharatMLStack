package repositories

import (
	pb "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"
)

type CacheStruct struct {
	Index           []int
	Embedding       []float32
	SearchEmbedding []float32
	CandidateId     string
	Filters         []*pb.Filter
}

type CandidateResponseStruct struct {
	Index              []int
	Response           *pb.CandidateResponse
	EmbeddingResponse  *pb.CandidateEmbedding
	DotProductResponse *pb.CandidateEmbeddingScore
}
