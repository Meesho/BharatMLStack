export const SERVICES = {
  PREDATOR: 'predator',
  InferFlow: 'inferflow',
  NUMERIX: 'numerix',
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
]; 