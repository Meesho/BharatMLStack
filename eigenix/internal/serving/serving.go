package serving

import (
	"context"
	"sync"

	pb "github.com/Meesho/skye-eigenix/internal/client"
	"github.com/Meesho/skye-eigenix/internal/hnswlib"
	"github.com/Meesho/skye-eigenix/internal/qdrant"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HandlerV1 struct {
	pb.EigenixServiceServer
	hnswlib hnswlib.Manager
	qdrant  qdrant.Manager
}

var (
	handlerV1 HandlerV1
	once      sync.Once
)

func Init() pb.EigenixServiceServer {
	once.Do(func() {
		handlerV1 = HandlerV1{
			hnswlib: hnswlib.Init(),
			qdrant:  qdrant.Init(),
		}
	})
	return &handlerV1
}

func (h *HandlerV1) CreateIndex(ctx context.Context, req *pb.CreateIndexRequest) (*pb.CreateIndexResponse, error) {
	if err := ValidateCreateIndexRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	if err := h.hnswlib.CreateIndex(req.Name, req.Space, req.Dimension, req.MaxElements, req.M, req.EfConstruction, req.AllowReplaceDeleted); err != nil {
		return nil, err
	}
	return &pb.CreateIndexResponse{Success: true, Message: "Index created successfully"}, nil
}

func (h *HandlerV1) GetIndex(ctx context.Context, req *pb.GetIndexRequest) (*pb.GetIndexResponse, error) {
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "index name is required")
	}

	index, exists := h.hnswlib.GetIndex(req.Name)
	if !exists {
		return nil, status.Errorf(codes.NotFound, "index not found")
	}
	count, err := h.hnswlib.GetIndexStats(index.Name)
	if err != nil {
		return nil, err
	}

	return &pb.GetIndexResponse{Index: &pb.IndexInfo{
		Name:                index.Name,
		Dimension:           index.Dimension,
		Space:               index.Space,
		MaxElements:         index.MaxElements,
		M:                   index.M,
		EfConstruction:      index.EfConstruct,
		AllowReplaceDeleted: index.AllowReplaceDeleted,
		CurrentCount:        int32(count),
	}}, nil
}

func (h *HandlerV1) DeleteIndex(ctx context.Context, req *pb.DeleteIndexRequest) (*pb.DeleteIndexResponse, error) {
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "index name is required")
	}

	if err := h.hnswlib.DeleteIndex(req.Name); err != nil {
		return nil, err
	}
	return &pb.DeleteIndexResponse{Success: true, Message: "Index deleted successfully"}, nil
}

func (h *HandlerV1) ListIndices(ctx context.Context, req *pb.ListIndicesRequest) (*pb.ListIndicesResponse, error) {
	indices := h.hnswlib.ListIndices()
	indicesInfo := make([]*pb.IndexInfo, len(indices))
	for i, index := range indices {
		count, err := h.hnswlib.GetIndexStats(index.Name)
		if err != nil {
			return nil, err
		}
		indicesInfo[i] = &pb.IndexInfo{
			Name:                index.Name,
			Dimension:           index.Dimension,
			Space:               index.Space,
			MaxElements:         index.MaxElements,
			M:                   index.M,
			EfConstruction:      index.EfConstruct,
			AllowReplaceDeleted: index.AllowReplaceDeleted,
			CurrentCount:        int32(count),
		}
	}
	return &pb.ListIndicesResponse{Indices: indicesInfo}, nil
}

func (h *HandlerV1) SaveIndex(ctx context.Context, req *pb.IndexOperationRequest) (*pb.IndexOperationResponse, error) {
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "index name is required")
	}

	if err := h.hnswlib.SaveIndex(req.Name); err != nil {
		return nil, err
	}
	return &pb.IndexOperationResponse{Success: true, Message: "Index saved successfully"}, nil
}

func (h *HandlerV1) LoadIndex(ctx context.Context, req *pb.IndexOperationRequest) (*pb.IndexOperationResponse, error) {
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "index name is required")
	}

	if err := h.hnswlib.LoadIndex(req.Name); err != nil {
		return nil, err
	}
	return &pb.IndexOperationResponse{Success: true, Message: "Index loaded successfully"}, nil
}

func (h *HandlerV1) AddPoints(ctx context.Context, req *pb.PointsRequest) (*pb.PointsResponse, error) {
	if req.IndexName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "index name is required")
	}
	if len(req.Points) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "points are required")
	}
	count, err := h.hnswlib.AddPoints(req.IndexName, req.Points)
	if err != nil {
		return nil, err
	}
	return &pb.PointsResponse{Success: true, AffectedCount: int32(count), TotalCount: int32(len(req.Points))}, nil
}

func (h *HandlerV1) UpdatePoints(ctx context.Context, req *pb.PointsRequest) (*pb.PointsResponse, error) {
	if req.IndexName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "index name is required")
	}
	if len(req.Points) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "points are required")
	}
	count, err := h.hnswlib.UpdatePoints(req.IndexName, req.Points)
	if err != nil {
		return nil, err
	}
	return &pb.PointsResponse{Success: true, AffectedCount: int32(count), TotalCount: int32(len(req.Points))}, nil
}

func (h *HandlerV1) DeletePoints(ctx context.Context, req *pb.DeletePointsRequest) (*pb.DeletePointsResponse, error) {
	if req.IndexName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "index name is required")
	}
	if len(req.Ids) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "ids are required")
	}
	count, err := h.hnswlib.DeletePoints(req.IndexName, req.Ids)
	if err != nil {
		return nil, err
	}
	return &pb.DeletePointsResponse{Success: true, DeletedCount: int32(count)}, nil
}

func (h *HandlerV1) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	if err := ValidateSearchRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	switch req.DatabaseType {
	case pb.DatabaseType_HNSW_LIB:
		results, err := h.hnswlib.Search(req.IndexName, req.Vector, int(req.Limit))
		if err != nil {
			return nil, err
		}
		resultsPb := make([]*pb.SearchResult, len(results))
		for i, result := range results {
			resultsPb[i] = &pb.SearchResult{Id: result.ID, Distance: result.Distance}
		}
		return &pb.SearchResponse{Results: resultsPb}, nil
	case pb.DatabaseType_QDRANT:
		results, err := h.qdrant.Search(ctx, qdrant.SearchRequest{
			IndexName: req.IndexName,
			Vector:    req.Vector,
			Limit:     uint64(req.Limit),
			Params: qdrant.SearchParams{
				UseIVF:            req.SearchParams.UseIvf,
				CentroidCount:     int(req.SearchParams.CentCount),
				SearchIndexedOnly: req.SearchParams.SearchIndexedOnly,
			},
		})
		if err != nil {
			return nil, err
		}
		resultsPb := make([]*pb.SearchResult, len(results))
		for i, result := range results {
			resultsPb[i] = &pb.SearchResult{Id: result.ID, Distance: result.Score}
		}
		return &pb.SearchResponse{Results: resultsPb}, nil
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid database type")
	}
}

func (h *HandlerV1) BatchSearch(ctx context.Context, req *pb.BatchSearchRequest) (*pb.BatchSearchResponse, error) {
	switch req.DatabaseType {
	case pb.DatabaseType_HNSW_LIB:
		results, err := h.hnswlib.BatchSearch(req.IndexName, req.Vectors, int(req.Limit))
		if err != nil {
			return nil, err
		}
		resultSets := make([]*pb.SearchResultSet, len(results))
		for i, set := range results {
			sr := make([]*pb.SearchResult, len(set))
			for j, r := range set {
				sr[j] = &pb.SearchResult{Id: r.ID, Distance: r.Distance}
			}
			resultSets[i] = &pb.SearchResultSet{Results: sr}
		}
		return &pb.BatchSearchResponse{ResultSets: resultSets}, nil
	case pb.DatabaseType_QDRANT:
		results, err := h.qdrant.BatchSearch(ctx, qdrant.BatchSearchRequest{
			IndexName: req.IndexName,
			Vectors:   req.Vectors,
			Limit:     uint64(req.Limit),
			Params: qdrant.SearchParams{
				UseIVF:            req.SearchParams.UseIvf,
				CentroidCount:     int(req.SearchParams.CentCount),
				SearchIndexedOnly: req.SearchParams.SearchIndexedOnly,
			},
		})
		if err != nil {
			return nil, err
		}
		resultSets := make([]*pb.SearchResultSet, len(results))
		for i, set := range results {
			sr := make([]*pb.SearchResult, len(set))
			for j, r := range set {
				sr[j] = &pb.SearchResult{Id: r.ID, Distance: r.Score}
			}
			resultSets[i] = &pb.SearchResultSet{Results: sr}
		}
		return &pb.BatchSearchResponse{ResultSets: resultSets}, nil
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid database type")
	}
}
