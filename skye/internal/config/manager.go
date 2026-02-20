package config

import (
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
)

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
	RegisterModel(string, string, bool, int, map[string]interface{}, string, int, string, Metadata, string, int, int, string) error
	RegisterVariant(string, string, string, VectorDbConfig, string, []Criteria, enums.Type, bool, int, bool, int, int, RateLimiter) error
	UpdateEmbeddingVersion(entity, model string, version int) error
	UpdateVariantEmbeddingStoreReadVersion(entity string, model string, variant string, version int) error
	UpdateVariantEmbeddingStoreWriteVersion(entity string, model string, variant string, version int) error
	UpdateVariantStates(entity string, model string, variant map[string]string, state int) error
	UpdatePartitionState(entity string, model string, partition string, state int) error
	GetRateLimiters() map[int]RateLimiter
	UpdateRateLimiter(entity string, model string, variant string, burstLimit int, rateLimit int) error
	RegisterWatchPathCallbackWithEvent(path string, callback func(key, value, eventType string) error) error
	UpdateVectorDbConfig(entity string, model string, variant string, vectorDbConfig VectorDbConfig) error
}
