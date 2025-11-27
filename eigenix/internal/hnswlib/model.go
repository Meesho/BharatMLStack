package hnswlib

// IndexMetadata stores configuration needed to reload an index
type IndexMetadata struct {
	Name                string `json:"name"`
	Dimension           int32  `json:"dimension"`
	Space               string `json:"space"`
	MaxElements         int32  `json:"max_elements"`
	M                   int32  `json:"m"`
	EfConstruct         int32  `json:"ef_construction"`
	AllowReplaceDeleted bool   `json:"allow_replace_deleted"`
}

// Point represents a vector point
type Point struct {
	ID     uint64    `json:"id"`
	Vector []float32 `json:"vector"`
}

// SearchRequest represents a search request
type SearchRequest struct {
	Vector []float32 `json:"vector"`
	K      int       `json:"k"`
}

// SearchResult represents a search result
type SearchResult struct {
	ID       uint64  `json:"id"`
	Distance float32 `json:"distance"`
}

// SearchResponse represents the response from a search
type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

// CreateIndexRequest represents a request to create an index
type CreateIndexRequest struct {
	Name                string `json:"name"`
	Dimension           int    `json:"dimension"`
	Space               string `json:"space"`
	MaxElements         int    `json:"max_elements"`
	M                   int    `json:"m"`
	EfConstruct         int    `json:"ef_construction"`
	AllowReplaceDeleted bool   `json:"allow_replace_deleted"`
}

// GetIndexRequest represents a request to get an index
type GetIndexRequest struct {
	Name string `json:"name"`
}

// DeleteIndexRequest represents a request to delete an index
type DeleteIndexRequest struct {
	Name string `json:"name"`
}

// SaveIndexRequest represents a request to save an index
type SaveIndexRequest struct {
	Name string `json:"name"`
}

// LoadIndexRequest represents a request to load an index
type LoadIndexRequest struct {
	Name string `json:"name"`
}

// AddPointRequest represents a request to add a single point to an index
type AddPointRequest struct {
	IndexName string `json:"index_name"`
	Point     Point  `json:"point"`
}

// AddPointsRequest represents a request to add points to an index
type AddPointsRequest struct {
	IndexName string  `json:"index_name"`
	Points    []Point `json:"points"`
}

// BatchSearchRequest represents a request to search with multiple query vectors
type BatchSearchRequest struct {
	Queries [][]float32 `json:"vectors"`
	K       int         `json:"k"`
}

// BatchSearchResponse represents the response from a batch search
type BatchSearchResponse struct {
	Results [][]SearchResult `json:"results"`
}

// UpdatePointRequest represents a request to update a single point
type UpdatePointRequest struct {
	IndexName string `json:"index_name"`
	Point     Point  `json:"point"`
}

// UpdatePointsRequest represents a request to update multiple points
type UpdatePointsRequest struct {
	IndexName string  `json:"index_name"`
	Points    []Point `json:"points"`
}

// DeletePointRequest represents a request to delete a single point
type DeletePointRequest struct {
	IndexName string `json:"index_name"`
	ID        uint64 `json:"id"`
}

// DeletePointsRequest represents a request to delete multiple points
type DeletePointsRequest struct {
	IndexName string   `json:"index_name"`
	IDs       []uint64 `json:"ids"`
}
