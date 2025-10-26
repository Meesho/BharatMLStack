package qdrant

import (
	"context"
)

// Manager defines the interface for Qdrant database operations
type Manager interface {
	// Client initialization
	LoadCentroids(filepath string) error
	Search(ctx context.Context, req SearchRequest) ([]SearchResponse, error)
	BatchSearch(ctx context.Context, req BatchSearchRequest) ([][]SearchResponse, error)
}
