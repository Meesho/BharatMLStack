export const SERVICES = {
  PREDATOR: 'predator',
  InferFlow: 'inferflow',
  NUMERIX: 'numerix',
  EMBEDDING_PLATFORM: 'embedding_platform',
};

export const MENU_PERMISSION_MAP = {
  // InferFlow service mappings
  'Deployable': { service: SERVICES.InferFlow, screenType: 'deployable' },
  'MPConfig': { service: SERVICES.InferFlow, screenType: 'inferflow-config' },
  'MPConfigApproval': { service: SERVICES.InferFlow, screenType: 'inferflow-config-approval' },
  
  // Numerix service mappings
  'NumerixConfigDR': { service: SERVICES.NUMERIX, screenType: 'numerix-config' },
  'NumerixConfigApproval': { service: SERVICES.NUMERIX, screenType: 'numerix-config-approval' },
  
  // Predator service mappings
  'Model': { service: SERVICES.PREDATOR, screenType: 'model' },
  'ModelApproval': { service: SERVICES.PREDATOR, screenType: 'model-approval' },
  
  // Embedding Platform service mappings
  // Note: StoreDiscovery, StoreRegistry, and EntityRegistry also exist in Online Feature Store
  // They only require permission check when under EmbeddingPlatform parent
  'StoreDiscovery': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'store-discovery', requiredParentKey: 'EmbeddingPlatform' },
  'EntityDiscovery': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'entity-discovery' },
  'ModelDiscovery': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'model-discovery' },
  'VariantDiscovery': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'variant-discovery' },
  'FilterDiscovery': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'filter-discovery' },
  'JobFrequencyDiscovery': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'job-frequency-discovery' },
  'StoreRegistry': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'store-registry', requiredParentKey: 'EmbeddingPlatform' },
  'EntityRegistry': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'entity-registry', requiredParentKey: 'EmbeddingPlatform' },
  'ModelRegistry': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'model-registry' },
  'VariantRegistry': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'variant-registry' },
  'FilterRegistry': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'filter-registry' },
  'JobFrequencyRegistry': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'job-frequency-registry' },
  'EmbeddingStoreApproval': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'store-approval' },
  'EmbeddingEntityApproval': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'entity-approval' },
  'EmbeddingModelApproval': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'model-approval' },
  'EmbeddingVariantApproval': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'variant-approval' },
  'EmbeddingFilterApproval': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'filter-approval' },
  'EmbeddingJobFrequencyApproval': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'job-frequency-approval' },
  'DeploymentOperations': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'deployment-operations' },
  'OnboardVariantToDB': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'onboard-variant-to-db' },
  'OnboardVariantApproval': { service: SERVICES.EMBEDDING_PLATFORM, screenType: 'onboard-variant-approval' },
};

export const requiresPermissionCheck = (menuKey, parentKey = null) => {
  const permissionInfo = MENU_PERMISSION_MAP[menuKey];
  
  if (!permissionInfo) {
    return false;
  }
  
  // If the permission entry has a requiredParentKey, check if it matches
  if (permissionInfo.requiredParentKey) {
    return parentKey === permissionInfo.requiredParentKey;
  }
  
  // No parent requirement, so permission check is required
  return true;
};

export const getPermissionInfo = (menuKey, parentKey = null) => {
  const permissionInfo = MENU_PERMISSION_MAP[menuKey];
  
  if (!permissionInfo) {
    return null;
  }
  
  // If the permission entry has a requiredParentKey, check if it matches
  if (permissionInfo.requiredParentKey) {
    if (parentKey === permissionInfo.requiredParentKey) {
      // Return permission info without the requiredParentKey field
      const { requiredParentKey, ...info } = permissionInfo;
      return info;
    }
    return null;
  }
  
  // No parent requirement, return the permission info
  const { requiredParentKey, ...info } = permissionInfo;
  return info;
};

export const PERMISSION_CONTROLLED_SERVICES = [
  SERVICES.PREDATOR,
  SERVICES.InferFlow, 
  SERVICES.NUMERIX,
  SERVICES.EMBEDDING_PLATFORM,
]; 