package embedding

import "github.com/Meesho/BharatMLStack/skye/internal/repositories"

type BulkQuery struct {
	CacheKeys    map[string]repositories.CacheStruct `json:"cache_keys"`
	CandidateIds []string                            `json:"candidate_ids"`
	Variant      string                              `json:"variant"`
	Model        string                              `json:"model"`
	Version      int                                 `json:"version"`
}

type Payload struct {
	CandidateId      string          `json:"candidate_id"`
	Embedding        []float32       `json:"embedding"`
	SearchEmbedding  []float32       `json:"search_embedding"`
	Model            string          `json:"model"`
	VariantsIndexMap map[string]bool `json:"variants_index_map"`
	Version          int             `json:"version"`
}

type Query struct {
	CandidateIds []string `json:"candidate_ids"`
	Model        string   `json:"model"`
	Variant      string   `json:"variant"`
}
