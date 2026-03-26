package resolver

const (
	// Embedding Platform screen types
	screenTypeEmbeddingStoreDiscovery         = "store-discovery"
	screenTypeEmbeddingEntityDiscovery        = "entity-discovery"
	screenTypeEmbeddingModelDiscovery         = "model-discovery"
	screenTypeEmbeddingVariantDiscovery       = "variant-discovery"
	screenTypeEmbeddingFilterDiscovery        = "filter-discovery"
	screenTypeEmbeddingJobFrequencyDiscovery  = "job-frequency-discovery"
	screenTypeEmbeddingStoreRegistry          = "store-registry"
	screenTypeEmbeddingEntityRegistry         = "entity-registry"
	screenTypeEmbeddingModelRegistry          = "model-registry"
	screenTypeEmbeddingVariantRegistry        = "variant-registry"
	screenTypeEmbeddingFilterRegistry         = "filter-registry"
	screenTypeEmbeddingJobFrequencyRegistry   = "job-frequency-registry"
	screenTypeEmbeddingStoreApproval          = "store-approval"
	screenTypeEmbeddingEntityApproval         = "entity-approval"
	screenTypeEmbeddingModelApproval          = "model-approval"
	screenTypeEmbeddingVariantApproval        = "variant-approval"
	screenTypeEmbeddingFilterApproval         = "filter-approval"
	screenTypeEmbeddingJobFrequencyApproval   = "job-frequency-approval"
	screenTypeEmbeddingDeploymentOperations  = "deployment-operations"
	screenTypeEmbeddingOnboardVariantToDB    = "onboard-variant-to-db"
	screenTypeEmbeddingOnboardVariantApproval = "onboard-variant-approval"

	serviceEmbeddingPlatform = "embedding_platform"

	// Resolver function names (based on common patterns)
	// Discovery resolvers
	resolverEmbeddingStoreDiscovery        = "EmbeddingPlatformStoreDiscoveryResolver"
	resolverEmbeddingEntityDiscovery       = "EmbeddingPlatformEntityDiscoveryResolver"
	resolverEmbeddingModelDiscovery        = "EmbeddingPlatformModelDiscoveryResolver"
	resolverEmbeddingVariantDiscovery      = "EmbeddingPlatformVariantDiscoveryResolver"
	resolverEmbeddingFilterDiscovery        = "EmbeddingPlatformFilterDiscoveryResolver"
	resolverEmbeddingJobFrequencyDiscovery = "EmbeddingPlatformJobFrequencyDiscoveryResolver"
	
	// Registry resolvers
	resolverEmbeddingStoreRegistry        = "EmbeddingPlatformStoreRegistryResolver"
	resolverEmbeddingEntityRegistry       = "EmbeddingPlatformEntityRegistryResolver"
	resolverEmbeddingModelRegistry        = "EmbeddingPlatformModelRegistryResolver"
	resolverEmbeddingVariantRegistry      = "EmbeddingPlatformVariantRegistryResolver"
	resolverEmbeddingFilterRegistry        = "EmbeddingPlatformFilterRegistryResolver"
	resolverEmbeddingJobFrequencyRegistry  = "EmbeddingPlatformJobFrequencyRegistryResolver"
	
	// Approval resolvers
	resolverEmbeddingStoreApproval        = "EmbeddingPlatformStoreApprovalResolver"
	resolverEmbeddingEntityApproval       = "EmbeddingPlatformEntityApprovalResolver"
	resolverEmbeddingModelApproval        = "EmbeddingPlatformModelApprovalResolver"
	resolverEmbeddingVariantApproval      = "EmbeddingPlatformVariantApprovalResolver"
	resolverEmbeddingFilterApproval       = "EmbeddingPlatformFilterApprovalResolver"
	resolverEmbeddingJobFrequencyApproval = "EmbeddingPlatformJobFrequencyApprovalResolver"
	
	// Operations resolvers
	resolverEmbeddingDeploymentOperations  = "EmbeddingPlatformDeploymentOperationsResolver"
	resolverEmbeddingOnboardVariantToDB    = "EmbeddingPlatformOnboardVariantToDBResolver"
	resolverEmbeddingOnboardVariantApproval = "EmbeddingPlatformOnboardVariantApprovalResolver"
)

type embeddingPlatformResolver struct {
}

func NewEmbeddingPlatformServiceResolver() (ServiceResolver, error) {
	return &embeddingPlatformResolver{}, nil
}

func (r *embeddingPlatformResolver) GetResolvers() map[string]Func {
	return map[string]Func{
		// Discovery - View
		resolverEmbeddingStoreDiscovery: StaticResolver(screenTypeEmbeddingStoreDiscovery, moduleView, serviceEmbeddingPlatform),
		resolverEmbeddingEntityDiscovery: StaticResolver(screenTypeEmbeddingEntityDiscovery, moduleView, serviceEmbeddingPlatform),
		resolverEmbeddingModelDiscovery: StaticResolver(screenTypeEmbeddingModelDiscovery, moduleView, serviceEmbeddingPlatform),
		resolverEmbeddingVariantDiscovery: StaticResolver(screenTypeEmbeddingVariantDiscovery, moduleView, serviceEmbeddingPlatform),
		resolverEmbeddingFilterDiscovery: StaticResolver(screenTypeEmbeddingFilterDiscovery, moduleView, serviceEmbeddingPlatform),
		resolverEmbeddingJobFrequencyDiscovery: StaticResolver(screenTypeEmbeddingJobFrequencyDiscovery, moduleView, serviceEmbeddingPlatform),
		
		// Registry - Onboard/Edit
		resolverEmbeddingStoreRegistry: StaticResolver(screenTypeEmbeddingStoreRegistry, moduleOnboard, serviceEmbeddingPlatform),
		resolverEmbeddingEntityRegistry: StaticResolver(screenTypeEmbeddingEntityRegistry, moduleOnboard, serviceEmbeddingPlatform),
		resolverEmbeddingModelRegistry: StaticResolver(screenTypeEmbeddingModelRegistry, moduleOnboard, serviceEmbeddingPlatform),
		resolverEmbeddingVariantRegistry: StaticResolver(screenTypeEmbeddingVariantRegistry, moduleOnboard, serviceEmbeddingPlatform),
		resolverEmbeddingFilterRegistry: StaticResolver(screenTypeEmbeddingFilterRegistry, moduleOnboard, serviceEmbeddingPlatform),
		resolverEmbeddingJobFrequencyRegistry: StaticResolver(screenTypeEmbeddingJobFrequencyRegistry, moduleOnboard, serviceEmbeddingPlatform),
		
		// Approval - Approve/View
		resolverEmbeddingStoreApproval: StaticResolver(screenTypeEmbeddingStoreApproval, moduleApprove, serviceEmbeddingPlatform),
		resolverEmbeddingEntityApproval: StaticResolver(screenTypeEmbeddingEntityApproval, moduleApprove, serviceEmbeddingPlatform),
		resolverEmbeddingModelApproval: StaticResolver(screenTypeEmbeddingModelApproval, moduleApprove, serviceEmbeddingPlatform),
		resolverEmbeddingVariantApproval: StaticResolver(screenTypeEmbeddingVariantApproval, moduleApprove, serviceEmbeddingPlatform),
		resolverEmbeddingFilterApproval: StaticResolver(screenTypeEmbeddingFilterApproval, moduleApprove, serviceEmbeddingPlatform),
		resolverEmbeddingJobFrequencyApproval: StaticResolver(screenTypeEmbeddingJobFrequencyApproval, moduleApprove, serviceEmbeddingPlatform),
		
		// Operations - Promote/Onboard
		resolverEmbeddingDeploymentOperations: StaticResolver(screenTypeEmbeddingDeploymentOperations, modulePromote, serviceEmbeddingPlatform),
		resolverEmbeddingOnboardVariantToDB: StaticResolver(screenTypeEmbeddingOnboardVariantToDB, moduleOnboard, serviceEmbeddingPlatform),
		resolverEmbeddingOnboardVariantApproval: StaticResolver(screenTypeEmbeddingOnboardVariantApproval, moduleApprove, serviceEmbeddingPlatform),
	}
}



