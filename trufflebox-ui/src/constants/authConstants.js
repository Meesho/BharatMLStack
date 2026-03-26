/**
 * Authentication and Authorization Constants
 * Centralized constants for auth-related functionality
 */

// Token storage keys
export const STORAGE_KEYS = {
  USER: 'user',
  AUTH_TOKEN: 'authToken',
  SESSION_ID: 'sessionId',
  OAUTH_STATE: 'oauth_state',
};

// Token types
export const TOKEN_TYPES = {
  ACCESS: 'access',
  REFRESH: 'refresh',
};

// Auth providers
export const AUTH_PROVIDERS = {
  PASSWORD: 'password',
  GOOGLE: 'google',
  BOTH: 'both',
};

// User roles
export const USER_ROLES = {
  USER: 'user',
  ADMIN: 'admin',
  SUPER_ADMIN: 'super_admin',
};

// Permission actions (standard set)
export const PERMISSION_ACTIONS = {
  VIEW: 'view',
  EDIT: 'edit',
  ONBOARD: 'onboard',
  APPROVE: 'approve',
  REJECT: 'reject',
  DELETE: 'delete',
  TEST: 'test',
  LOAD_TEST: 'load_test',
  PROMOTE: 'promote',
  CLONE: 'clone',
  UPLOAD: 'upload',
  UPLOAD_EDIT: 'upload_edit',
  UPLOAD_PARTIAL: 'upload_partial',
  SCALE_UP: 'scale_up',
  VALIDATE: 'validate',
  CANCEL: 'cancel',
  DEACTIVATE: 'deactivate',
};

// All possible actions for super_admin
export const ALL_ACTIONS = Object.values(PERMISSION_ACTIONS);

// Token refresh timing (in milliseconds)
export const TOKEN_REFRESH = {
  // Refresh token 10 minutes before expiry
  REFRESH_BEFORE_EXPIRY: 10 * 60 * 1000,
  // Minimum time between refresh attempts
  MIN_REFRESH_INTERVAL: 5 * 60 * 1000,
};

// Error messages
export const ERROR_MESSAGES = {
  INVALID_CREDENTIALS: 'Invalid email or password',
  SESSION_EXPIRED: 'Your session has expired. Please log in again.',
  UNAUTHORIZED: 'You do not have permission to access this resource.',
  NETWORK_ERROR: 'Network error. Please check your connection and try again.',
  TOKEN_REFRESH_FAILED: 'Failed to refresh session. Please log in again.',
  OAUTH_STATE_MISMATCH: 'Invalid OAuth state. Please try again.',
  OAUTH_FAILED: 'OAuth authentication failed. Please try again.',
  PERMISSION_DENIED: 'You do not have the required permissions for this action.',
  GENERIC_ERROR: 'An unexpected error occurred. Please try again.',
};

// Success messages
export const SUCCESS_MESSAGES = {
  LOGIN_SUCCESS: 'Successfully logged in',
  LOGOUT_SUCCESS: 'Successfully logged out',
  TOKEN_REFRESHED: 'Session refreshed successfully',
  PERMISSION_UPDATED: 'Permission updated successfully',
  USER_UPDATED: 'User updated successfully',
};

// HTTP status codes
export const HTTP_STATUS = {
  OK: 200,
  CREATED: 201,
  UNAUTHORIZED: 401,
  FORBIDDEN: 403,
  NOT_FOUND: 404,
  INTERNAL_SERVER_ERROR: 500,
};

// API endpoints
export const API_ENDPOINTS = {
  LOGIN: '/login',
  REGISTER: '/register',
  LOGOUT: '/logout',
  REFRESH_TOKEN: '/auth/refresh',
  SSO_STATUS: '/auth/sso/status',
  GOOGLE_INITIATE: '/auth/google/initiate',
  GOOGLE_CALLBACK: '/auth/google/callback',
  LINK_GOOGLE: '/auth/link-google',
  UNLINK_GOOGLE: '/auth/unlink-google',
  USERS: '/users',
  PERMISSIONS: '/permissions',
  PERMISSION_BY_ROLE: '/api/v1/horizon/permission-by-role',
  TRACK_SESSION: '/track-session',
};

// Application routes
export const APP_ROUTES = {
  LOGIN: '/login',
  REGISTER: '/register',
  UNAUTHORIZED: '/unauthorized',
  HOME: '/feature-discovery', // Default landing page after login
  ROOT: '/',
};

// JWT token field names (standard JWT claims)
export const JWT_CLAIMS = {
  SUBJECT: 'sub', // Standard JWT subject claim
  USER_ID: 'user_id', // Custom claim (fallback)
  ROLE: 'role', // Custom claim
  EXPIRY: 'exp', // Standard JWT expiry claim
};

// Timing constants (in milliseconds)
export const TIMING = {
  // HTTP interceptor delays
  LOGOUT_REDIRECT_DELAY: 200, // Delay before redirecting to login after logout
  LOGOUT_CLEANUP_DELAY: 1000, // Delay before resetting logout flag
  
  // Notification display
  SESSION_EXPIRED_NOTIFICATION_DURATION: 3000, // How long to show session expired notification
  
  // Token refresh
  TOKEN_EXPIRY_MULTIPLIER: 1000, // Convert JWT exp (seconds) to milliseconds
};

// Environment configuration
export const ENV_CONFIG = {
  // Staging environment bypass (should be false in production)
  // This is a temporary workaround - should be removed in production
  ENABLE_STAGING_BYPASS: process.env.REACT_APP_ENABLE_STAGING_BYPASS === 'true',
  
  // Default SSO fallback values (used when SSO status fetch fails)
  DEFAULT_SSO_STATUS: {
    sso_enabled: false,
    providers: [],
    allow_password: true,
    allow_both: false,
  },
};

// Default values
export const DEFAULTS = {
  AUTH_PROVIDER: AUTH_PROVIDERS.PASSWORD,
  DEFAULT_ROLE: USER_ROLES.USER,
};

