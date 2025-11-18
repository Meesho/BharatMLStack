export const SERVICES = {
  NUMERIX: 'numerix',
};

export const MENU_PERMISSION_MAP = {
  // Numerix service mappings
  'NumerixConfigDR': { service: SERVICES.NUMERIX, screenType: 'numerix-config' },
  'NumerixConfigApproval': { service: SERVICES.NUMERIX, screenType: 'numerix-config-approval' },
};

export const requiresPermissionCheck = (menuKey) => {
  return MENU_PERMISSION_MAP.hasOwnProperty(menuKey);
};

export const getPermissionInfo = (menuKey) => {
  return MENU_PERMISSION_MAP[menuKey] || null;
};

export const PERMISSION_CONTROLLED_SERVICES = [
  SERVICES.NUMERIX,
]; 