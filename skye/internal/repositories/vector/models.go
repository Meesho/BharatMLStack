package vector

import (
	pb "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"
	"github.com/qdrant/go-client/qdrant"
)

type UpsertRequest struct {
	Data map[string][]Data
}

type Data struct {
	Id      string
	Payload map[string]interface{}
	Vectors []float32
}

type DeleteRequest struct {
	Data map[string][]Data
}

type UpsertPayloadRequest struct {
	Data map[string][]Data
}

type QueryDetails struct {
	CacheKey        string
	Embedding       []float32
	Offset          int
	CandidateLimit  int32
	MetadataFilters []*pb.Filter
	Payload         []string
	SearchParams    map[string]string
}

type QueryRequest struct {
	Entity               string
	Model                string
	Variant              string
	Version              int
	SearchRequestDetails *QueryDetails
}

type BatchQueryRequest struct {
	Entity      string
	Model       string
	Variant     string
	Version     int
	RequestList []*QueryDetails
}

type QueryResponse struct {
	Candidates []*SimilarCandidate
}

type BatchQueryResponse struct {
	SimilarCandidatesList map[string][]*SimilarCandidate
}

type SimilarCandidate struct {
	Id      string
	Score   float32
	Payload map[string]string
}

type CollectionInfoResponse struct {
	Status              string
	IndexedVectorsCount float64
	PointsCount         float64
	PayloadPointsCount  []float64
}

type NgtRequest struct {
	Entity               string
	Model                string
	Variant              string
	Version              int
	SearchRequestDetails *NgtEmbeddingRequest
}

// NgtRequest represents the structure of the incoming request containing the embedding
type NgtEmbeddingRequest struct {
	Embedding []float32 `json:"embedding"`
	K         int       `json:"k"`
	Epsilon   float64   `json:"epsilon"`
	Radius    float64   `json:"radius"`
}

type NgtResponse struct {
	Results []NgtResult `json:"results"`
}

type NgtResult struct {
	ID       int     `json:"id"`
	Distance float32 `json:"distance"`
}

type EigenixResponse struct {
	Results [][]EigenixResult `json:"results"`
}

type EigenixResult struct {
	ID       int     `json:"id"`
	Distance float32 `json:"distance"`
}

type FilterCondition struct {
	Condition *qdrant.Condition
	IsNegated bool
}
