package qdrant

type Db interface {
	CreateCollection(*CreateCollectionRequest) error
	TriggerIndexing(*TriggerIndexingRequest) error
	ProcessModel(*ProcessModelRequest) (*ProcessModelResponse, error)
	ProcessMultiVariant(*ProcessMultiVariantRequest) (*ProcessModelResponse, error)
	ProcessModelsWithFrequency(*ProcessModelsWithFrequencyRequest) (*ProcessModelsWithFrequencyResponse, error)
	PromoteVariant(*PromoteVariantRequest) (*ProcessModelResponse, error)
	ProcessMultiVariantForceReset(*ProcessMultiVariantRequest) (*ProcessModelResponse, error)
	PublishCollectionMetrics() error
}
