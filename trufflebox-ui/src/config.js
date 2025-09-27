
const env = window.env || {};

export const REACT_APP_HORIZON_BASE_URL = process.env.REACT_APP_HORIZON_BASE_URL || env.REACT_APP_HORIZON_BASE_URL || "http://localhost:8082";
export const REACT_APP_HORIZON_PROD_BASE_URL = process.env.REACT_APP_HORIZON_PROD_BASE_URL || env.REACT_APP_HORIZON_PROD_BASE_URL || "http://localhost:8083";
