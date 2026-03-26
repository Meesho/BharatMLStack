package resolver

const (
	// Online Feature Store screen types
	screenTypeFeatureDiscovery    = "feature-discovery"
	screenTypeStoreDiscovery      = "store-discovery"
	screenTypeJobDiscovery        = "job-discovery"
	screenTypeClientDiscovery     = "client-discovery"
	screenTypeStoreRegistry       = "store-registry"
	screenTypeEntityRegistry      = "entity-registry"
	screenTypeFeatureGroupRegistry = "feature-group-registry"
	screenTypeFeatureRegistry     = "feature-registry"
	screenTypeJobRegistry         = "job-registry"
	screenTypeFeatureApproval     = "feature-approval"

	serviceOnlineFeatureStore = "online_feature_store"

	// Resolver function names
	resolverRegisterStore              = "OnlineFeatureStoreRegisterStoreResolver"
	resolverRegisterEntity             = "OnlineFeatureStoreRegisterEntityResolver"
	resolverEditEntity                 = "OnlineFeatureStoreEditEntityResolver"
	resolverRegisterFeatureGroup      = "OnlineFeatureStoreRegisterFeatureGroupResolver"
	resolverEditFeatureGroup           = "OnlineFeatureStoreEditFeatureGroupResolver"
	resolverAddFeatures                = "OnlineFeatureStoreAddFeaturesResolver"
	resolverEditFeatures               = "OnlineFeatureStoreEditFeaturesResolver"
	resolverDeleteFeatures             = "OnlineFeatureStoreDeleteFeaturesResolver"
	resolverRegisterJob                = "OnlineFeatureStoreRegisterJobResolver"
	resolverGetEntities                = "OnlineFeatureStoreGetEntitiesResolver"
	resolverGetConfig                  = "OnlineFeatureStoreGetConfigResolver"
	resolverGetCacheConfig             = "OnlineFeatureStoreGetCacheConfigResolver"
	resolverGetStores                  = "OnlineFeatureStoreGetStoresResolver"
	resolverGetJobs                    = "OnlineFeatureStoreGetJobsResolver"
	resolverGetFeatureGroups           = "OnlineFeatureStoreGetFeatureGroupsResolver"
	resolverRetrieveEntities           = "OnlineFeatureStoreRetrieveEntitiesResolver"
	resolverRetrieveFeatureGroups       = "OnlineFeatureStoreRetrieveFeatureGroupsResolver"
	resolverProcessStore               = "OnlineFeatureStoreProcessStoreResolver"
	resolverProcessEntity              = "OnlineFeatureStoreProcessEntityResolver"
	resolverProcessFeatureGroup        = "OnlineFeatureStoreProcessFeatureGroupResolver"
	resolverProcessJob                 = "OnlineFeatureStoreProcessJobResolver"
	resolverProcessAddFeatures         = "OnlineFeatureStoreProcessAddFeaturesResolver"
	resolverProcessDeleteFeatures      = "OnlineFeatureStoreProcessDeleteFeaturesResolver"
	resolverGetStoreRequests           = "OnlineFeatureStoreGetStoreRequestsResolver"
	resolverGetEntityRequests          = "OnlineFeatureStoreGetEntityRequestsResolver"
	resolverGetFeatureGroupRequests    = "OnlineFeatureStoreGetFeatureGroupRequestsResolver"
	resolverGetJobRequests             = "OnlineFeatureStoreGetJobRequestsResolver"
	resolverGetAddFeaturesRequests     = "OnlineFeatureStoreGetAddFeaturesRequestsResolver"
	resolverGetSourceMapping           = "OnlineFeatureStoreGetSourceMappingResolver"
	resolverGetOnlineFeaturesMapping   = "OnlineFeatureStoreGetOnlineFeaturesMappingResolver"
)

type onlineFeatureStoreResolver struct {
}

func NewOnlineFeatureStoreResolver() (ServiceResolver, error) {
	return &onlineFeatureStoreResolver{}, nil
}

func (r *onlineFeatureStoreResolver) GetResolvers() map[string]Func {
	return map[string]Func{
		// Store Registry - Register/Edit
		resolverRegisterStore: StaticResolver(screenTypeStoreRegistry, moduleOnboard, serviceOnlineFeatureStore),
		
		// Entity Registry - Register/Edit
		resolverRegisterEntity: StaticResolver(screenTypeEntityRegistry, moduleOnboard, serviceOnlineFeatureStore),
		resolverEditEntity: StaticResolver(screenTypeEntityRegistry, moduleEdit, serviceOnlineFeatureStore),
		
		// Feature Group Registry - Register/Edit
		resolverRegisterFeatureGroup: StaticResolver(screenTypeFeatureGroupRegistry, moduleOnboard, serviceOnlineFeatureStore),
		resolverEditFeatureGroup: StaticResolver(screenTypeFeatureGroupRegistry, moduleEdit, serviceOnlineFeatureStore),
		
		// Feature Registry - Add/Edit/Delete
		resolverAddFeatures: StaticResolver(screenTypeFeatureRegistry, moduleOnboard, serviceOnlineFeatureStore),
		resolverEditFeatures: StaticResolver(screenTypeFeatureRegistry, moduleEdit, serviceOnlineFeatureStore),
		resolverDeleteFeatures: StaticResolver(screenTypeFeatureRegistry, moduleDelete, serviceOnlineFeatureStore),
		
		// Job Registry - Register
		resolverRegisterJob: StaticResolver(screenTypeJobRegistry, moduleOnboard, serviceOnlineFeatureStore),
		
		// Discovery - View
		resolverGetEntities: StaticResolver(screenTypeEntityRegistry, moduleView, serviceOnlineFeatureStore),
		resolverGetStores: StaticResolver(screenTypeStoreDiscovery, moduleView, serviceOnlineFeatureStore),
		resolverGetJobs: StaticResolver(screenTypeJobDiscovery, moduleView, serviceOnlineFeatureStore),
		resolverGetFeatureGroups: StaticResolver(screenTypeFeatureGroupRegistry, moduleView, serviceOnlineFeatureStore),
		resolverRetrieveEntities: StaticResolver(screenTypeEntityRegistry, moduleView, serviceOnlineFeatureStore),
		resolverRetrieveFeatureGroups: StaticResolver(screenTypeFeatureGroupRegistry, moduleView, serviceOnlineFeatureStore),
		resolverGetConfig: StaticResolver(screenTypeFeatureDiscovery, moduleView, serviceOnlineFeatureStore),
		resolverGetCacheConfig: StaticResolver(screenTypeFeatureDiscovery, moduleView, serviceOnlineFeatureStore),
		
		// Process/Approve - Approve action
		resolverProcessStore: StaticResolver(screenTypeFeatureApproval, moduleApprove, serviceOnlineFeatureStore),
		resolverProcessEntity: StaticResolver(screenTypeFeatureApproval, moduleApprove, serviceOnlineFeatureStore),
		resolverProcessFeatureGroup: StaticResolver(screenTypeFeatureApproval, moduleApprove, serviceOnlineFeatureStore),
		resolverProcessJob: StaticResolver(screenTypeFeatureApproval, moduleApprove, serviceOnlineFeatureStore),
		resolverProcessAddFeatures: StaticResolver(screenTypeFeatureApproval, moduleApprove, serviceOnlineFeatureStore),
		resolverProcessDeleteFeatures: StaticResolver(screenTypeFeatureApproval, moduleApprove, serviceOnlineFeatureStore),
		
		// Approval Requests - View
		resolverGetStoreRequests: StaticResolver(screenTypeFeatureApproval, moduleView, serviceOnlineFeatureStore),
		resolverGetEntityRequests: StaticResolver(screenTypeFeatureApproval, moduleView, serviceOnlineFeatureStore),
		resolverGetFeatureGroupRequests: StaticResolver(screenTypeFeatureApproval, moduleView, serviceOnlineFeatureStore),
		resolverGetJobRequests: StaticResolver(screenTypeFeatureApproval, moduleView, serviceOnlineFeatureStore),
		resolverGetAddFeaturesRequests: StaticResolver(screenTypeFeatureApproval, moduleView, serviceOnlineFeatureStore),
		
		// Utility - View
		resolverGetSourceMapping: StaticResolver(screenTypeFeatureDiscovery, moduleView, serviceOnlineFeatureStore),
		resolverGetOnlineFeaturesMapping: StaticResolver(screenTypeFeatureDiscovery, moduleView, serviceOnlineFeatureStore),
	}
}



