package qdrant

import pb "github.com/Meesho/skye-eigenix/internal/client"

// QdrantConfig holds configuration for Qdrant client initialization
type QdrantConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// SearchParams defines search behavior parameters from proto
type SearchParams struct {
	UseIVF            bool `json:"use_ivf"`             // If true, use IVF routing to multiple collections
	CentroidCount     int  `json:"centroid_count"`      // Number of centroids to search (only for IVF mode)
	SearchIndexedOnly bool `json:"search_indexed_only"` // Only search indexed vectors
}

// SearchRequest represents a search request with optional IVF routing
type SearchRequest struct {
	IndexName string       `json:"index_name"` // Base collection/index name
	Vector    []float32    `json:"vector"`     // Query vector
	Limit     uint64       `json:"limit"`      // Number of results to return
	Params    SearchParams `json:"params"`     // Search parameters (IVF, exact search, etc.)
}

type BatchSearchRequest struct {
	IndexName string       `json:"index_name"` // Base collection/index name
	Vectors   []*pb.Vector `json:"vectors"`    // Query vector
	Limit     uint64       `json:"limit"`      // Number of results to return
	Params    SearchParams `json:"params"`     // Search parameters (IVF, exact search, etc.)
}

// SearchResponse represents the response from a search operation
type SearchResponse struct {
	ID    uint64  `json:"id"`
	Score float32 `json:"score"`
}

// CentroidInfo stores information about K-means centroids for IVF
type CentroidInfo struct {
	Centers    [][]float32 `json:"centers"`     // Normalized centroid vectors
	TotalCount int         `json:"total_count"` // Total number of centroids
	Dimension  int         `json:"dimension"`   // Dimension of vectors
}
