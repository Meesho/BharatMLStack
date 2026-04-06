package embedding

const (
	GenericRetrieveQuery = "SELECT %s FROM %s.%s WHERE %s = ? AND %s = ? AND %s = ?"
	GenericPersistQuery  = "INSERT INTO %s.%s (%s) VALUES (%s) using TTL %v"
	Id                   = "id"
	ModelName            = "model_name"
	Version              = "version"
	Embedding            = "embedding"
	SearchEmbedding      = "search_embedding"
	GenericToBeIndexed   = "_to_be_indexed"
)

type Store interface {
	BulkQuery(storeId string, bulkQuery *BulkQuery, queryType string) error
	BulkQueryConsumer(storeId string, bulkQuery *BulkQuery) (map[string]map[string]interface{}, error)
	Persist(storeId string, ttl int, payload Payload) error
}
