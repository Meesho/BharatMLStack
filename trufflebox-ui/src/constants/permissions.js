
export const SERVICES = {
  PREDATOR: 'predator',
  InferFlow: 'inferflow',
  NUMERIX: 'numerix',
};

export const SCREEN_TYPES = {
  PREDATOR: {
    DEPLOYABLE: 'deployable',
    MODEL: 'model',
    MODEL_APPROVAL: 'model-approval'
  },
  
  InferFlow: {
    DEPLOYABLE: 'deployable',
    CONNECTION_CONFIG: 'connection-config',
    MP_CONFIG: 'inferflow-config',
    MP_CONFIG_APPROVAL: 'inferflow-config-approval',

  },
  
  NUMERIX: {
    CONFIG: 'numerix-config',
    CONFIG_APPROVAL: 'numerix-config-approval'
  },
};

export const ACTIONS = {
  VIEW: 'view',
  ONBOARD: 'onboard',
  EDIT: 'edit',
  CLONE: 'clone',
  DELETE: 'delete',
  
  UPLOAD: 'upload',
  UPLOAD_EDIT: 'upload_edit',
  UPLOAD_PARTIAL: 'upload_partial',
  
  PROMOTE: 'promote',
  SCALE_UP: 'scale_up',
  
  VALIDATE: 'validate',
  APPROVE: 'approve',
  REJECT: 'reject',
  CANCEL: 'cancel',
  DEACTIVATE: 'deactivate',
  
  TEST: 'test'
};

export const ROLES = {
  ADMIN: 'admin',
  USER: 'user'
};

export const PERMISSION_LEVELS = {
  NONE: 'none',
  READ: 'read',
  WRITE: 'write',
  ADMIN: 'admin'
};

export const ALL_SCREEN_TYPES = {
  ...SCREEN_TYPES.PREDATOR,
  ...SCREEN_TYPES.InferFlow,
  ...SCREEN_TYPES.NUMERIX,
};

export const SERVICE_SCREEN_MAP = {
  [SERVICES.PREDATOR]: Object.values(SCREEN_TYPES.PREDATOR),
  [SERVICES.InferFlow]: Object.values(SCREEN_TYPES.InferFlow),
  [SERVICES.NUMERIX]: Object.values(SCREEN_TYPES.NUMERIX),
};

export const ACTION_GROUPS = {
  BASIC: [ACTIONS.VIEW],
  CRUD: [ACTIONS.VIEW, ACTIONS.ONBOARD, ACTIONS.EDIT, ACTIONS.REMOVE],
  MANAGEMENT: [ACTIONS.VIEW, ACTIONS.ONBOARD, ACTIONS.EDIT, ACTIONS.REMOVE, ACTIONS.CLONE, ACTIONS.PROMOTE, ACTIONS.SCALE_UP],
  APPROVAL: [ACTIONS.VIEW, ACTIONS.VALIDATE, ACTIONS.APPROVE, ACTIONS.REJECT, ACTIONS.CANCEL],
  TESTING: [ACTIONS.VIEW, ACTIONS.TEST],
  FULL: Object.values(ACTIONS)
};

export const DEFAULT_ROLE_PERMISSIONS = {
  [ROLES.ADMIN]: ACTION_GROUPS.FULL,
  [ROLES.MANAGER]: ACTION_GROUPS.MANAGEMENT,
  [ROLES.DEVELOPER]: ACTION_GROUPS.CRUD,
  [ROLES.VIEWER]: ACTION_GROUPS.BASIC
};

export const API_ENDPOINTS = {
  PERMISSIONS: '/api/v1/horizon/permission-by-role'
};

export const ERROR_MESSAGES = {
  PERMISSIONS_NOT_LOADED: 'Permissions are still loading. Please wait.',
  PERMISSION_DENIED: 'You do not have permission to perform this action.',
  INVALID_SERVICE: 'Invalid service specified.',
  INVALID_SCREEN_TYPE: 'Invalid screen type specified.',
  INVALID_ACTION: 'Invalid action specified.',
  API_ERROR: 'Failed to fetch permissions from server.',
  NETWORK_ERROR: 'Network error occurred while fetching permissions.'
};

export const SUCCESS_MESSAGES = {
  PERMISSIONS_LOADED: 'Permissions loaded successfully.',
  ACTION_COMPLETED: 'Action completed successfully.'
};

export const isValidService = (service) => {
  return Object.values(SERVICES).includes(service);
};

export const isValidScreenType = (service, screenType) => {
  if (!isValidService(service)) return false;
  return SERVICE_SCREEN_MAP[service]?.includes(screenType) || false;
};

export const isValidAction = (action) => {
  return Object.values(ACTIONS).includes(action);
};

export const isValidRole = (role) => {
  return Object.values(ROLES).includes(role);
};

export const getServiceDisplayName = (service) => {
  const names = {
    [SERVICES.PREDATOR]: 'Predator',
    [SERVICES.InferFlow]: 'InferFlow',
    [SERVICES.NUMERIX]: 'Numerix',
  };
  return names[service] || service;
};

export const getScreenTypeDisplayName = (screenType) => {
  const names = {
    [SCREEN_TYPES.PREDATOR.DEPLOYABLE]: 'Deployable',
    [SCREEN_TYPES.PREDATOR.MODEL]: 'Model Management',
    [SCREEN_TYPES.PREDATOR.MODEL_APPROVAL]: 'Model Approval',
    
    [SCREEN_TYPES.InferFlow.DEPLOYABLE]: 'Deployable',
    [SCREEN_TYPES.InferFlow.CONNECTION_CONFIG]: 'Connection Configuration',
    [SCREEN_TYPES.InferFlow.MP_CONFIG]: 'InferFlow Configuration',
    [SCREEN_TYPES.InferFlow.MP_CONFIG_APPROVAL]: 'Configuration Approval',
    [SCREEN_TYPES.InferFlow.MP_CONFIG_TESTING]: 'Configuration Testing',
    
    [SCREEN_TYPES.NUMERIX.CONFIG]: 'Numerix Configuration',
    [SCREEN_TYPES.NUMERIX.CONFIG_APPROVAL]: 'Configuration Approval',
  };
  return names[screenType] || screenType.replace(/-/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
};

export const getActionDisplayName = (action) => {
  const names = {
    [ACTIONS.VIEW]: 'View',
    [ACTIONS.ONBOARD]: 'Create/Onboard',
    [ACTIONS.EDIT]: 'Edit',
    [ACTIONS.REMOVE]: 'Delete',
    [ACTIONS.CLONE]: 'Clone',
    [ACTIONS.UPLOAD]: 'Upload',
    [ACTIONS.UPLOAD_EDIT]: 'Upload Edit',
    [ACTIONS.UPLOAD_PARTIAL]: 'Upload Partial',
    [ACTIONS.PROMOTE]: 'Promote',
    [ACTIONS.SCALE_UP]: 'Scale Up',
    [ACTIONS.VALIDATE]: 'Validate',
    [ACTIONS.APPROVE]: 'Approve',
    [ACTIONS.REJECT]: 'Reject',
    [ACTIONS.CANCEL]: 'Cancel',
    [ACTIONS.TEST]: 'Test'
  };
  return names[action] || action.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
};

export const validatePermissionRequest = (service, screenType, action) => {
  const errors = [];
  
  if (!isValidService(service)) {
    errors.push(`Invalid service: ${service}`);
  }
  
  if (!isValidScreenType(service, screenType)) {
    errors.push(`Invalid screen type: ${screenType} for service: ${service}`);
  }
  
  if (action && !isValidAction(action)) {
    errors.push(`Invalid action: ${action}`);
  }
  
  return {
    isValid: errors.length === 0,
    errors
  };
};

export const getRoutePath = (service, screenType) => {
  return `/${service}/${screenType}`;
};

export const getAllServiceRoutes = () => {
  const routes = [];
  Object.entries(SERVICE_SCREEN_MAP).forEach(([service, screenTypes]) => {
    screenTypes.forEach(screenType => {
      routes.push({
        service,
        screenType,
        path: getRoutePath(service, screenType),
        displayName: `${getServiceDisplayName(service)} - ${getScreenTypeDisplayName(screenType)}`
      });
    });
  });
  return routes;
};

export const comparePermissions = (permissions1, permissions2) => {
  if (!permissions1 || !permissions2) return false;
  return JSON.stringify(permissions1) === JSON.stringify(permissions2);
};

export const mergePermissions = (basePermissions, additionalPermissions) => {
  if (!basePermissions) return additionalPermissions;
  if (!additionalPermissions) return basePermissions;
  
  return {
    ...basePermissions,
    permissions: [
      ...(basePermissions.permissions || []),
      ...(additionalPermissions.permissions || [])
    ]
  };
};

export const PermissionSystem = {
  SERVICES,
  SCREEN_TYPES,
  ACTIONS,
  ROLES,
  PERMISSION_LEVELS,
  ALL_SCREEN_TYPES,
  SERVICE_SCREEN_MAP,
  ACTION_GROUPS,
  DEFAULT_ROLE_PERMISSIONS,
  API_ENDPOINTS,
  ERROR_MESSAGES,
  SUCCESS_MESSAGES,
  isValidService,
  isValidScreenType,
  isValidAction,
  isValidRole,
  getServiceDisplayName,
  getScreenTypeDisplayName,
  getActionDisplayName,
  validatePermissionRequest,
  getRoutePath,
  getAllServiceRoutes
};

export default PermissionSystem; 