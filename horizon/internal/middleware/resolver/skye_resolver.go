package resolver

const (
	screenTypeStoreDiscovery         = "store-discovery"
	screenTypeStoreRegistry          = "store-registry"
	screenTypeStoreApproval          = "store-approval"
	screenTypeEntityDiscovery        = "entity-discovery"
	screenTypeEntityRegistry         = "entity-registry"
	screenTypeEntityApproval         = "entity-approval"
	screenTypeModelDiscovery         = "model-discovery"
	screenTypeModelRegistry          = "model-registry"
	screenTypeModelApproval          = "model-approval"
	screenTypeVariantDiscovery       = "variant-discovery"
	screenTypeVariantRegistry        = "variant-registry"
	screenTypeVariantApproval        = "variant-approval"
	screenTypeFilterDiscovery        = "filter-discovery"
	screenTypeFilterRegistry         = "filter-registry"
	screenTypeFilterApproval         = "filter-approval"
	screenTypeJobFrequencyDiscovery  = "job-frequency-discovery"
	screenTypeJobFrequencyRegistry   = "job-frequency-registry"
	screenTypeJobFrequencyApproval   = "job-frequency-approval"
	screenTypeDeploymentOperations   = "deployment-operations"
	screenTypeOnboardVariantToDB     = "onboard-variant-to-db"
	screenTypeOnboardVariantApproval = "onboard-variant-approval"

	serviceEmbeddingPlatform = "embedding_platform"

	// Resolvers - Store
	resolverSkyeStoreRegister         = "SkyeStoreRegisterResolver"
	resolverSkyeStoreApprove          = "SkyeStoreApproveResolver"
	resolverSkyeStoreDiscovery        = "SkyeStoreDiscoveryResolver"
	resolverSkyeStoreRequestDiscovery = "SkyeStoreRequestDiscoveryResolver"

	// Resolvers - Entity
	resolverSkyeEntityRegister         = "SkyeEntityRegisterResolver"
	resolverSkyeEntityApprove          = "SkyeEntityApproveResolver"
	resolverSkyeEntityDiscovery        = "SkyeEntityDiscoveryResolver"
	resolverSkyeEntityRequestDiscovery = "SkyeEntityRequestDiscoveryResolver"

	// Resolvers - Model
	resolverSkyeModelRegister         = "SkyeModelRegisterResolver"
	resolverSkyeModelEdit             = "SkyeModelEditResolver"
	resolverSkyeModelApprove          = "SkyeModelApproveResolver"
	resolverSkyeModelEditApprove      = "SkyeModelEditApproveResolver"
	resolverSkyeModelDiscovery        = "SkyeModelDiscoveryResolver"
	resolverSkyeModelRequestDiscovery = "SkyeModelRequestDiscoveryResolver"

	// Resolvers - Variant
	resolverSkyeVariantRegister         = "SkyeVariantRegisterResolver"
	resolverSkyeVariantEdit             = "SkyeVariantEditResolver"
	resolverSkyeVariantApprove          = "SkyeVariantApproveResolver"
	resolverSkyeVariantEditApprove      = "SkyeVariantEditApproveResolver"
	resolverSkyeVariantDiscovery        = "SkyeVariantDiscoveryResolver"
	resolverSkyeVariantRequestDiscovery = "SkyeVariantRequestDiscoveryResolver"

	// Resolvers - Filter
	resolverSkyeFilterRegister         = "SkyeFilterRegisterResolver"
	resolverSkyeFilterApprove          = "SkyeFilterApproveResolver"
	resolverSkyeFilterDiscovery        = "SkyeFilterDiscoveryResolver"
	resolverSkyeFilterRequestDiscovery = "SkyeFilterRequestDiscoveryResolver"

	// Resolvers - Job Frequency
	resolverSkyeJobFrequencyRegister         = "SkyeJobFrequencyRegisterResolver"
	resolverSkyeJobFrequencyApprove          = "SkyeJobFrequencyApproveResolver"
	resolverSkyeJobFrequencyDiscovery        = "SkyeJobFrequencyDiscoveryResolver"
	resolverSkyeJobFrequencyRequestDiscovery = "SkyeJobFrequencyRequestDiscoveryResolver"

	// Resolvers - Qdrant
	resolverSkyeQdrantCreate    = "SkyeQdrantCreateResolver"
	resolverSkyeQdrantApprove   = "SkyeQdrantApproveResolver"
	resolverSkyeQdrantDiscovery = "SkyeQdrantDiscoveryResolver"

	// Resolvers - Variant Promotion
	resolverSkyeVariantPromote        = "SkyeVariantPromoteResolver"
	resolverSkyeVariantPromoteApprove = "SkyeVariantPromoteApproveResolver"

	// Resolvers - Variant Onboarding
	resolverSkyeVariantOnboard                 = "SkyeVariantOnboardResolver"
	resolverSkyeVariantOnboardApprove          = "SkyeVariantOnboardApproveResolver"
	resolverSkyeVariantOnboardRequestDiscovery = "SkyeVariantOnboardRequestDiscoveryResolver"
	resolverSkyeVariantOnboardDiscovery        = "SkyeVariantOnboardDiscoveryResolver"
)

type SkyeResolver struct {
}

func NewSkyeServiceResolver() (ServiceResolver, error) {
	return &SkyeResolver{}, nil
}

func (r *SkyeResolver) GetResolvers() map[string]Func {
	return map[string]Func{
		// Store resolvers
		resolverSkyeStoreRegister:         StaticResolver(screenTypeStoreRegistry, moduleOnboard, serviceEmbeddingPlatform),
		resolverSkyeStoreApprove:          StaticResolver(screenTypeStoreApproval, moduleReview, serviceEmbeddingPlatform),
		resolverSkyeStoreDiscovery:        StaticResolver(screenTypeStoreDiscovery, moduleView, serviceEmbeddingPlatform),
		resolverSkyeStoreRequestDiscovery: StaticResolver(screenTypeStoreApproval, moduleView, serviceEmbeddingPlatform),

		// Entity resolvers
		resolverSkyeEntityRegister:         StaticResolver(screenTypeEntityRegistry, moduleOnboard, serviceEmbeddingPlatform),
		resolverSkyeEntityApprove:          StaticResolver(screenTypeEntityApproval, moduleReview, serviceEmbeddingPlatform),
		resolverSkyeEntityDiscovery:        StaticResolver(screenTypeEntityDiscovery, moduleView, serviceEmbeddingPlatform),
		resolverSkyeEntityRequestDiscovery: StaticResolver(screenTypeEntityApproval, moduleView, serviceEmbeddingPlatform),

		// Model resolvers
		resolverSkyeModelRegister:         StaticResolver(screenTypeModelRegistry, moduleOnboard, serviceEmbeddingPlatform),
		resolverSkyeModelEdit:             StaticResolver(screenTypeModelRegistry, moduleEdit, serviceEmbeddingPlatform),
		resolverSkyeModelApprove:          StaticResolver(screenTypeModelApproval, moduleReview, serviceEmbeddingPlatform),
		resolverSkyeModelEditApprove:      StaticResolver(screenTypeModelApproval, moduleReview, serviceEmbeddingPlatform),
		resolverSkyeModelDiscovery:        StaticResolver(screenTypeModelDiscovery, moduleView, serviceEmbeddingPlatform),
		resolverSkyeModelRequestDiscovery: StaticResolver(screenTypeModelApproval, moduleView, serviceEmbeddingPlatform),

		// Variant resolvers
		resolverSkyeVariantRegister:         StaticResolver(screenTypeVariantRegistry, moduleOnboard, serviceEmbeddingPlatform),
		resolverSkyeVariantEdit:             StaticResolver(screenTypeVariantRegistry, moduleEdit, serviceEmbeddingPlatform),
		resolverSkyeVariantApprove:          StaticResolver(screenTypeVariantApproval, moduleReview, serviceEmbeddingPlatform),
		resolverSkyeVariantEditApprove:      StaticResolver(screenTypeVariantApproval, moduleReview, serviceEmbeddingPlatform),
		resolverSkyeVariantDiscovery:        StaticResolver(screenTypeVariantDiscovery, moduleView, serviceEmbeddingPlatform),
		resolverSkyeVariantRequestDiscovery: StaticResolver(screenTypeVariantApproval, moduleView, serviceEmbeddingPlatform),

		// Filter resolvers
		resolverSkyeFilterRegister:         StaticResolver(screenTypeFilterRegistry, moduleOnboard, serviceEmbeddingPlatform),
		resolverSkyeFilterApprove:          StaticResolver(screenTypeFilterApproval, moduleReview, serviceEmbeddingPlatform),
		resolverSkyeFilterDiscovery:        StaticResolver(screenTypeFilterDiscovery, moduleView, serviceEmbeddingPlatform),
		resolverSkyeFilterRequestDiscovery: StaticResolver(screenTypeFilterApproval, moduleView, serviceEmbeddingPlatform),

		// Job Frequency resolvers
		resolverSkyeJobFrequencyRegister:         StaticResolver(screenTypeJobFrequencyRegistry, moduleOnboard, serviceEmbeddingPlatform),
		resolverSkyeJobFrequencyApprove:          StaticResolver(screenTypeJobFrequencyApproval, moduleReview, serviceEmbeddingPlatform),
		resolverSkyeJobFrequencyDiscovery:        StaticResolver(screenTypeJobFrequencyDiscovery, moduleView, serviceEmbeddingPlatform),
		resolverSkyeJobFrequencyRequestDiscovery: StaticResolver(screenTypeJobFrequencyApproval, moduleView, serviceEmbeddingPlatform),

		// Qdrant/Deployment Operations resolvers
		resolverSkyeQdrantCreate:    StaticResolver(screenTypeDeploymentOperations, moduleOnboard, serviceEmbeddingPlatform),
		resolverSkyeQdrantApprove:   StaticResolver(screenTypeDeploymentOperations, moduleReview, serviceEmbeddingPlatform),
		resolverSkyeQdrantDiscovery: StaticResolver(screenTypeDeploymentOperations, moduleView, serviceEmbeddingPlatform),

		// Variant Promotion resolvers (revisit this)
		resolverSkyeVariantPromote:        StaticResolver(screenTypeVariantRegistry, modulePromote, serviceEmbeddingPlatform),
		resolverSkyeVariantPromoteApprove: StaticResolver(screenTypeVariantRegistry, moduleReview, serviceEmbeddingPlatform),

		// Variant Onboarding resolvers
		resolverSkyeVariantOnboard:                 StaticResolver(screenTypeOnboardVariantToDB, moduleOnboard, serviceEmbeddingPlatform),
		resolverSkyeVariantOnboardApprove:          StaticResolver(screenTypeOnboardVariantApproval, moduleReview, serviceEmbeddingPlatform),
		resolverSkyeVariantOnboardRequestDiscovery: StaticResolver(screenTypeOnboardVariantApproval, moduleView, serviceEmbeddingPlatform),
		resolverSkyeVariantOnboardDiscovery:        StaticResolver(screenTypeOnboardVariantToDB, moduleView, serviceEmbeddingPlatform),
	}
}
