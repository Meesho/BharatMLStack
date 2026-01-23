/**
 * Constants for Embedding Platform (Skye)
 * Business rules, validation constants, and UI configurations
 */

// =============================================================================
// BUSINESS RULE CONSTANTS
// =============================================================================

export const BUSINESS_RULES = {
  // Store Management
  STORE: {
    REQUIRED_CONF_ID: 1, // Only conf_id = 1 is allowed
    SUPPORTED_DATABASES: ['scylla'],
  },

  // Model Management
  MODEL: {
    REQUIRED_PARTITIONS: 24, // Always forced to 24
    ALLOWED_MODEL_TYPES: ['RESET', 'DELTA'],
    NON_EDITABLE_FIELDS: ['mq_id', 'topic_name'], // Cannot edit after creation
  },

  // Variant Management
  VARIANT: {
    FORCED_VECTOR_DB_TYPE: 'QDRANT', // Always QDRANT
    FORCED_TYPE: 'EXPERIMENT', // Always EXPERIMENT
    ADMIN_ONLY_FIELDS: ['vector_db_config', 'rate_limiter', 'rt_partition'],
  },
};

// =============================================================================
// STATUS CONSTANTS
// =============================================================================

export const REQUEST_STATUS = {
  PENDING: 'PENDING',
  IN_PROGRESS: 'IN_PROGRESS',
  APPROVED: 'APPROVED',
  REJECTED: 'REJECTED',
  COMPLETED: 'COMPLETED',
  FAILED: 'FAILED',
  CANCELLED: 'CANCELLED',
};

export const APPROVAL_DECISIONS = {
  APPROVED: 'APPROVED',
  REJECTED: 'REJECTED',
  NEEDS_MODIFICATION: 'NEEDS_MODIFICATION',
};

// =============================================================================
// JOB FREQUENCY OPTIONS
// =============================================================================

export const JOB_FREQUENCIES = [
  { id: 'FREQ_1H', label: 'Hourly', description: 'Runs every hour' },
  { id: 'FREQ_4H', label: '4 Hours', description: 'Runs every 4 hours' },
  { id: 'FREQ_6H', label: '6 Hours', description: 'Runs every 6 hours' },
  { id: 'FREQ_12H', label: '12 Hours', description: 'Runs every 12 hours' },
  { id: 'FREQ_1D', label: 'Daily', description: 'Runs every day' },
  { id: 'FREQ_2D', label: 'Bi-Daily', description: 'Runs every 2 days' },
  { id: 'FREQ_3D', label: '3 Days', description: 'Runs every 3 days' },
  { id: 'FREQ_1W', label: 'Weekly', description: 'Runs every week' },
  { id: 'FREQ_2W', label: 'Bi-Weekly', description: 'Runs every 2 weeks' },
  { id: 'FREQ_1M', label: 'Monthly', description: 'Runs every month' },
];

// =============================================================================
// DISTANCE FUNCTIONS
// =============================================================================

export const DISTANCE_FUNCTIONS = [
  { id: 'EUCLIDEAN', label: 'Euclidean Distance' },
  { id: 'COSINE', label: 'Cosine Similarity' },
  { id: 'DOT_PRODUCT', label: 'Dot Product' },
  { id: 'MANHATTAN', label: 'Manhattan Distance' },
];



// =============================================================================
// STATUS COLOR MAPPINGS
// =============================================================================

export const STATUS_COLORS = {
  [REQUEST_STATUS.PENDING]: {
    background: '#FFF8E1',
    text: '#F57C00',
    label: 'Pending',
  },
  [REQUEST_STATUS.IN_PROGRESS]: {
    background: '#E3F2FD',
    text: '#1976D2',
    label: 'In Progress',
  },
  [REQUEST_STATUS.APPROVED]: {
    background: '#E7F6E7',
    text: '#2E7D32',
    label: 'Approved',
  },
  [REQUEST_STATUS.REJECTED]: {
    background: '#FFEBEE',
    text: '#D32F2F',
    label: 'Rejected',
  },
  [REQUEST_STATUS.COMPLETED]: {
    background: '#E8F5E8',
    text: '#2E7D32',
    label: 'Completed',
  },
  [REQUEST_STATUS.FAILED]: {
    background: '#FFEBEE',
    text: '#D32F2F',
    label: 'Failed',
  },
  [REQUEST_STATUS.CANCELLED]: {
    background: '#F5F5F5',
    text: '#757575',
    label: 'Cancelled',
  },
};

// =============================================================================
// FORM VALIDATION PATTERNS
// =============================================================================

export const VALIDATION_PATTERNS = {
  EMAIL: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
  TOPIC_NAME: /^[a-zA-Z0-9_-]+$/, // Alphanumeric, underscore, and hyphen
  TABLE_NAME: /^[a-zA-Z][a-zA-Z0-9_]*$/, // Must start with letter
  ENTITY_NAME: /^[a-zA-Z][a-zA-Z0-9_]*$/, // Must start with letter
  MODEL_NAME: /^[a-zA-Z][a-zA-Z0-9_]*$/, // Must start with letter
  VARIANT_NAME: /^[a-zA-Z][a-zA-Z0-9_]*$/, // Must start with letter
  COLUMN_NAME: /^[a-zA-Z][a-zA-Z0-9_]*$/, // Must start with letter
  DNS_SUBDOMAIN: /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$/, // RFC 1123 subdomain
};

// =============================================================================
// ERROR MESSAGES
// =============================================================================

export const ERROR_MESSAGES = {
  REQUIRED_FIELD: 'This field is required',
  INVALID_EMAIL: 'Please enter a valid email address',
  INVALID_FORMAT: 'Invalid format',
  INVALID_NUMBER: 'Please enter a valid number',
  POSITIVE_NUMBER_REQUIRED: 'Must be a positive number',
  CONF_ID_FIXED: 'Configuration ID must be 1',
  PARTITIONS_FIXED: 'Number of partitions is fixed at 24',
  VECTOR_DB_FIXED: 'Vector DB type is fixed as QDRANT',
  TYPE_FIXED: 'Type is fixed as EXPERIMENT',
  FIELD_NOT_EDITABLE: 'This field cannot be edited after creation',
  MIN_LENGTH: (min) => `Must be at least ${min} characters long`,
  MAX_LENGTH: (max) => `Must be no more than ${max} characters long`,
  MIN_VALUE: (min) => `Must be at least ${min}`,
  MAX_VALUE: (max) => `Must be no more than ${max}`,
};




export default {
  BUSINESS_RULES,
  REQUEST_STATUS,
  APPROVAL_DECISIONS,
  JOB_FREQUENCIES,
  DISTANCE_FUNCTIONS,
  STATUS_COLORS,
  VALIDATION_PATTERNS,
  ERROR_MESSAGES,
};

