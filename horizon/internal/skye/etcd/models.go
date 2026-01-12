package etcd

// SkyeConfigRegistry represents the top-level configuration structure for Skye in ETCD
type SkyeConfigRegistry struct {
	Storage                  StorageConfig                      `json:"storage"`
	EntityConfigs            map[string]EntityConfig            `json:"entity-configs"`
	ModelConfigs             map[string]ModelConfig             `json:"model-configs"`
	VariantConfigs           map[string]VariantConfig           `json:"variant-configs"`
	FilterConfigs            map[string]FilterConfig            `json:"filter-configs"`
	QdrantClusterConfigs     map[string]QdrantClusterConfig     `json:"qdrant-cluster-configs"`
	VariantPromotionConfigs  map[string]VariantPromotionConfig  `json:"variant-promotion-configs"`
	VariantOnboardingConfigs map[string]VariantOnboardingConfig `json:"variant-onboarding-configs"`
	JobFrequencyConfigs      map[string]JobFrequencyConfig      `json:"job-frequency-configs"`
}

// StorageConfig represents the storage configuration structure
type StorageConfig struct {
	Stores map[string]StoreConfig `json:"stores"`
}
