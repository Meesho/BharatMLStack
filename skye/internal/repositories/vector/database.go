package vector

type Database interface {
	CreateCollection(entity string, model string, variant string, version int) error
	BulkUpsert(upsertRequest UpsertRequest) error
	BulkDelete(deleteRequest DeleteRequest) error
	BulkUpsertPayload(upsertPayloadRequest UpsertPayloadRequest) error
	// Query - vector search query interface. Supports filtering, fetching metadata/payload, search query params like m, ef etc
	// Query(vectorQueryRequest *QueryRequest, metricTags []string) (*QueryResponse, error)
	// BatchQuery - like query interface but with bulk request support
	BatchQuery(bulkRequest *BatchQueryRequest, metricTags []string) (*BatchQueryResponse, error)
	DeleteCollection(entity, model, variant string, version int) error
	UpdateIndexingThreshold(entity, model, variant string, version int, indexingThreshold string) error
	GetCollectionInfo(entity, model, variant string, version int) (*CollectionInfoResponse, error)
	GetReadCollectionInfo(entity, model, variant string, version int) (*CollectionInfoResponse, error)
	RefreshClients(key, value, eventType string) error
	CreateFieldIndexes(entity, model, variant string, version int) error
}
