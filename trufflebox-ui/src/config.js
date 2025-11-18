
const env = window.env || {};

export const REACT_APP_HORIZON_BASE_URL = process.env.REACT_APP_HORIZON_BASE_URL || env.REACT_APP_HORIZON_BASE_URL || "http://localhost:8082";
export const REACT_APP_HORIZON_PROD_BASE_URL = process.env.REACT_APP_HORIZON_PROD_BASE_URL || env.REACT_APP_HORIZON_PROD_BASE_URL || "http://localhost:8085";
export const REACT_APP_SKYE_BASE_URL = process.env.REACT_APP_SKYE_BASE_URL || env.REACT_APP_SKYE_BASE_URL || "http://localhost:8083";
export const REACT_APP_MODEL_INFERENCE_BASE_URL = process.env.REACT_APP_MODEL_INFERENCE_BASE_URL || env.REACT_APP_MODEL_INFERENCE_BASE_URL || "http://localhost:8084";
export const REACT_APP_ENVIRONMENT = process.env.REACT_APP_ENVIRONMENT || env.REACT_APP_ENVIRONMENT || "production";
export const REACT_APP_COMPANY = process.env.REACT_APP_COMPANY || env.REACT_APP_COMPANY || "";

// Feature flags
export const isEmbeddingPlatformEnabled = () => {
  const company = REACT_APP_COMPANY.toLowerCase();
  return company === 'meesho';
};