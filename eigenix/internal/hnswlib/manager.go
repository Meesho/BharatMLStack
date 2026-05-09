package hnswlib

import (
	pb "github.com/Meesho/skye-eigenix/internal/client"
)

// Manager defines the interface for database operations on HNSW indices
type Manager interface {
	// Index management (pure CRUD operations)
	CreateIndex(name, space string, dimension int32, maxElements int32, m int32, efConstruct int32, allowReplaceDeleted bool) error
	GetIndex(name string) (*HNSWIndex, bool)
	ListIndices() []*HNSWIndex
	DeleteIndex(name string) error

	// Data operations (pure database operations)
	AddPoint(indexName string, id uint64, vector []float32) error
	AddPoints(indexName string, points []*pb.Point) (int, error)
	UpdatePoint(indexName string, id uint64, vector []float32) error
	UpdatePoints(indexName string, points []*pb.Point) (int, error)
	DeletePoint(indexName string, id uint64) error
	DeletePoints(indexName string, ids []uint64) (int, error)
	Search(indexName string, query []float32, k int) ([]SearchResult, error)
	BatchSearch(indexName string, queries []*pb.Vector, k int) ([][]SearchResult, error)
	GetIndexStats(indexName string) (int, error)

	// Persistence (pure file I/O operations)
	SaveIndex(indexName string) error
	LoadIndex(indexName string) error
	SaveAllIndices()
	LoadAllIndices()
}
