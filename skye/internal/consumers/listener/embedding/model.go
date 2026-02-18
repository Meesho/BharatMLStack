package embedding

type Operation string

const (
	add           Operation = "ADD"
	delete        Operation = "DELETE"
	upsertPayload Operation = "UPSERT_PAYLOAD"
)

type Event struct {
	Entity                string      `json:"entity"`
	Model                 string      `json:"model_name"`
	IndexSpace            IndexSpace  `json:"index_space"`
	SearchSpace           SearchSpace `json:"search_space"`
	CandidateId           string      `json:"candidate_id"`
	Environment           string      `json:"environment"`
	EmbeddingStoreVersion int         `json:"embedding_store_version"`
	Partition             string      `json:"partition"`
}

type IndexSpace struct {
	VariantsVersionMap map[string]int    `json:"variants_version_map"`
	Embedding          []float32         `json:"embedding"`
	VariantsIndexMap   map[string]bool   `json:"variants_index_map"`
	Operation          Operation         `json:"operation"`
	Payload            map[string]string `json:"payload"`
}

type SearchSpace struct {
	Embedding []float32 `json:"embedding"`
}
