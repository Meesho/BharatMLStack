
const env = window.env || {};

export const REACT_APP_HORIZON_BASE_URL = process.env.REACT_APP_HORIZON_BASE_URL || env.REACT_APP_HORIZON_BASE_URL || "http://localhost:8082";
export const REACT_APP_HORIZON_PROD_BASE_URL = process.env.REACT_APP_HORIZON_PROD_BASE_URL || env.REACT_APP_HORIZON_PROD_BASE_URL || "http://localhost:8085";
export const REACT_APP_SKYE_BASE_URL = process.env.REACT_APP_SKYE_BASE_URL || env.REACT_APP_SKYE_BASE_URL || "http://localhost:8083";
export const REACT_APP_MODEL_INFERENCE_BASE_URL = process.env.REACT_APP_MODEL_INFERENCE_BASE_URL || env.REACT_APP_MODEL_INFERENCE_BASE_URL || "http://localhost:8084";
export const REACT_APP_ENVIRONMENT = process.env.REACT_APP_ENVIRONMENT || env.REACT_APP_ENVIRONMENT || "staging";

// Helper function to parse boolean environment variables
const getBooleanEnv = (key, defaultValue = false) => {
  const value = process.env[key] || env[key];
  if (value === undefined || value === null) return defaultValue;
  if (typeof value === 'string') {
    return value.toLowerCase() === 'true' || value === '1';
  }
  return Boolean(value);
};

// Service-wise feature flags
export const REACT_APP_ONLINE_FEATURE_STORE_ENABLED = getBooleanEnv('REACT_APP_ONLINE_FEATURE_STORE_ENABLED', true);
export const REACT_APP_INFERFLOW_ENABLED = getBooleanEnv('REACT_APP_INFERFLOW_ENABLED', false);
export const REACT_APP_NUMERIX_ENABLED = getBooleanEnv('REACT_APP_NUMERIX_ENABLED', true);
export const REACT_APP_PREDATOR_ENABLED = getBooleanEnv('REACT_APP_PREDATOR_ENABLED', false);
export const REACT_APP_EMBEDDING_PLATFORM_ENABLED = getBooleanEnv('REACT_APP_EMBEDDING_PLATFORM_ENABLED', false);

// Feature flag helper functions
export const isOnlineFeatureStoreEnabled = () => REACT_APP_ONLINE_FEATURE_STORE_ENABLED;
export const isInferFlowEnabled = () => REACT_APP_INFERFLOW_ENABLED;
export const isNumerixEnabled = () => REACT_APP_NUMERIX_ENABLED;
export const isPredatorEnabled = () => REACT_APP_PREDATOR_ENABLED;
export const isEmbeddingPlatformEnabled = () => REACT_APP_EMBEDDING_PLATFORM_ENABLED;