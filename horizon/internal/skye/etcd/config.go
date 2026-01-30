package etcd

import "github.com/Meesho/BharatMLStack/horizon/internal/skye/etcd/enums"

type Manager interface {
	GetSkyeConfig() (*Skye, error)
	GetEntities() (map[string]Models, error)
	GetEntityConfig(entity string) (*Models, error)
	GetModelConfig(entity, model string) (*Model, error)
	GetVariantConfig(entity, model, variant string) (*Variant, error)
	GetAllFiltersForActiveVariants(entity string) (map[string]map[string][]Criteria, error)
	SetVariantOnboarded(entity string, model string, variant string, onboarded bool) error
	UpdateVariantState(entity string, model string, variant string, variantState enums.VariantState) error
	UpdateVariantReadVersion(entity string, model string, variant string, version int) error
	UpdateVariantWriteVersion(entity string, model string, variant string, version int) error
	RegisterStore(confId int, db string, embeddingTable string, aggregatorTable string) error
	RegisterFrequency(frequency string) error
	RegisterEntity(entity string, storeId string) error
	RegisterModel(string, string, bool, int, ModelConfig, string, int, string, Metadata, string, int, int, string) error
	RegisterVariant(string, string, string, VectorDbConfig, string, []Criteria, enums.Type, bool, int, bool, int, bool, int, bool, int, bool, int, bool, int, int, RateLimiter) error
	UpdateEmbeddingVersion(entity, model string, version int) error
	UpdateVariantEmbeddingStoreReadVersion(entity string, model string, variant string, version int) error
	UpdateVariantEmbeddingStoreWriteVersion(entity string, model string, variant string, version int) error
	UpdateVariantStates(entity string, model string, variant map[string]string, state int) error
	UpdatePartitionState(entity string, model string, partition string, state int) error
	GetRateLimiters() map[int]RateLimiter
	UpdateVectorDbConfig(entity string, model string, variant string, vectorDbConfig VectorDbConfig) error
	GetStores() (map[string]Data, error)
	RegisterFilter(entity string, columnName string, filterValue string, defaultValue string) error
	GetFilters(entity string) (map[string]Criteria, error)
	GetFrequencies() (map[string]string, error)
}
