package etcd

// Manager interface defines all ETCD operations for Skye
type Manager interface {
	// Store operations
	CreateStoreConfig(storeId string, storeConfig StoreConfig) error
	UpdateStoreConfig(storeId string, storeConfig StoreConfig) error
	DeleteStoreConfig(storeId string) error
	StoreExists(storeId string) (bool, error)
	GetStores() (map[string]StoreConfig, error)

	// Entity operations
	CreateEntityConfig(entityId string, entityConfig EntityConfig) error
	UpdateEntityConfig(entityId string, entityConfig EntityConfig) error
	DeleteEntityConfig(entityId string) error
	EntityExists(entityName string) (bool, error)
	GetEntities() (map[string]EntityConfig, error)

	// Model operations
	CreateModelConfig(modelId string, modelConfig ModelConfig) error
	UpdateModelConfig(modelId string, modelConfig ModelConfig) error
	ModelExists(entityName string, modelName string) (bool, error)
	GetModels() (map[string]ModelConfig, error)

	// Variant operations
	CreateVariantConfig(variantId string, variantConfig VariantConfig) error
	UpdateVariantConfig(variantId string, variantConfig VariantConfig) error
	VariantExists(entityName string, modelName string, variantName string) (bool, error)
	GetVariants() (map[string]VariantConfig, error)

	// Filter operations
	CreateFilterConfig(filterId string, filterConfig FilterConfig) error
	UpdateFilterConfig(filterId string, filterConfig FilterConfig) error
	FilterExistsByColumnName(entityName string, columnName string) (bool, error)
	GetFilters() (map[string]FilterConfig, error)

	// Job Frequency operations
	CreateJobFrequencyConfig(frequencyId string, frequencyConfig JobFrequencyConfig) error
	UpdateJobFrequencyConfig(frequencyId string, frequencyConfig JobFrequencyConfig) error
	DeleteJobFrequencyConfig(frequencyId string) error
	GetJobFrequencies() (map[string]JobFrequencyConfig, error)
	GetJobFrequenciesAsString() (string, error)
}
