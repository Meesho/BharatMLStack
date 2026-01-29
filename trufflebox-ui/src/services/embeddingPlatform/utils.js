import {
  VALIDATION_PATTERNS,
  ERROR_MESSAGES,
  BUSINESS_RULES,
  STATUS_COLORS,
  REQUEST_STATUS,
} from './constants';

// VALIDATION UTILITIES

/**
 * Validate store registration payload
 */
export const validateStorePayload = (payload) => {
  const errors = {};

  // Validate conf_id (must be 1)
  // Convert to number for comparison to handle both string and number types
  if (!payload.conf_id) {
    errors.conf_id = ERROR_MESSAGES.REQUIRED_FIELD;
  } else {
    const confIdNum = Number(payload.conf_id);
    if (isNaN(confIdNum) || confIdNum !== BUSINESS_RULES.STORE.REQUIRED_CONF_ID) {
      errors.conf_id = ERROR_MESSAGES.CONF_ID_FIXED;
    }
  }

  // Validate database type
  if (!payload.db) {
    errors.db = 'Database type is required';
  }

  // Validate embeddings table
  if (!payload.embeddings_table) {
    errors.embeddings_table = ERROR_MESSAGES.REQUIRED_FIELD;
  } else if (!VALIDATION_PATTERNS.TABLE_NAME.test(payload.embeddings_table)) {
    errors.embeddings_table = 'Embeddings table name must start with a letter and contain only alphanumeric characters and underscores';
  }

  // Validate aggregator table
  if (!payload.aggregator_table) {
    errors.aggregator_table = ERROR_MESSAGES.REQUIRED_FIELD;
  } else if (!VALIDATION_PATTERNS.TABLE_NAME.test(payload.aggregator_table)) {
    errors.aggregator_table = 'Aggregator table name must start with a letter and contain only alphanumeric characters and underscores';
  }

  return { isValid: Object.keys(errors).length === 0, errors };
};

/**
 * Validate entity registration payload
 */
export const validateEntityPayload = (payload) => {
  const errors = {};

  // Validate entity name
  if (!payload.entity) {
    errors.entity = ERROR_MESSAGES.REQUIRED_FIELD;
  } else if (!VALIDATION_PATTERNS.ENTITY_NAME.test(payload.entity)) {
    errors.entity = 'Entity name must start with a letter and contain only alphanumeric characters and underscores';
  }

  // Validate store_id
  if (!payload.store_id) {
    errors.store_id = 'Store ID is required';
  }

  return { isValid: Object.keys(errors).length === 0, errors };
};

/**
 * Validate model registration payload
 */
export const validateModelPayload = (payload) => {
  const errors = {};

  // Validate entity
  if (!payload.entity) {
    errors.entity = 'Entity is required';
  }

  // Validate model name
  if (!payload.model) {
    errors.model = ERROR_MESSAGES.REQUIRED_FIELD;
  } else if (!VALIDATION_PATTERNS.MODEL_NAME.test(payload.model)) {
    errors.model = 'Model name must start with a letter and contain only alphanumeric characters and underscores';
  }

  // Validate model type
  if (!payload.model_type) {
    errors.model_type = ERROR_MESSAGES.REQUIRED_FIELD;
  } else if (!BUSINESS_RULES.MODEL.ALLOWED_MODEL_TYPES.includes(payload.model_type)) {
    errors.model_type = `Model type must be one of: ${BUSINESS_RULES.MODEL.ALLOWED_MODEL_TYPES.join(', ')}`;
  }

  // Validate number of partitions (must be 24)
  if (!payload.number_of_partitions) {
    errors.number_of_partitions = ERROR_MESSAGES.REQUIRED_FIELD;
  } else if (payload.number_of_partitions !== BUSINESS_RULES.MODEL.REQUIRED_PARTITIONS) {
    errors.number_of_partitions = ERROR_MESSAGES.PARTITIONS_FIXED;
  }

  // Validate MQ ID
  if (!payload.mq_id && payload.mq_id !== 0) {
    errors.mq_id = 'MQ ID is required';
  } else {
    const num = Number(payload.mq_id);
    if (isNaN(num)) {
      errors.mq_id = 'MQ ID must be a valid number';
    } else if (num <= 0) {
      errors.mq_id = 'MQ ID must be greater than 0';
    }
  }

  // Validate job frequency
  if (!payload.job_frequency) {
    errors.job_frequency = 'Job frequency is required';
  }

  // Validate training data path
  if (!payload.training_data_path) {
    errors.training_data_path = 'Training data path is required';
  }

  // Validate model config
  if (!payload.model_config) {
    errors.model_config = 'Model configuration is required';
  } else {
    const { model_config } = payload;

    if (!model_config.distance_function) {
      errors['model_config.distance_function'] = 'Distance function is required';
    }

    if (!model_config.vector_dimension && model_config.vector_dimension !== 0) {
      errors['model_config.vector_dimension'] = 'Vector dimension is required';
    } else {
      const num = Number(model_config.vector_dimension);
      if (isNaN(num)) {
        errors['model_config.vector_dimension'] = 'Vector dimension must be a valid number';
      } else if (num <= 0) {
        errors['model_config.vector_dimension'] = 'Vector dimension must be greater than 0';
      }
    }
  }

  return { isValid: Object.keys(errors).length === 0, errors };
};


// FORMATTING UTILITIES

/**
 * Format date for display
 */
export const formatDate = (dateString) => {
  if (!dateString) return '-';
  try {
    return new Date(dateString).toLocaleString();
  } catch {
    return dateString;
  }
};

// DATA TRANSFORMATION UTILITIES

/**
 * Transform API response to table format
 */
export const transformToTableData = (data, type = 'requests') => {
  if (!Array.isArray(data)) return [];

  return data.map(item => ({
    RequestId: item.request_id || item.id,
    CreatedBy: item.created_by || item.requestor,
    ApprovedBy: item.approved_by || '-',
    Status: item.status || REQUEST_STATUS.PENDING,
    CreatedAt: formatDate(item.created_at),
    UpdatedAt: formatDate(item.updated_at),
    EntityLabel: item.entity || '-',
    FeatureGroupLabel: item.model || '-',
    Payload: JSON.stringify(item.payload || item),
    RejectReason: item.reject_reason || item.approval_comments || '-',
    ...item, // Include all original fields
  }));
};

// BUSINESS LOGIC UTILITIES

/**
 * Get default form values based on type
 */
export const getDefaultFormValues = (type) => {
  const defaults = {
    store: {
      conf_id: BUSINESS_RULES.STORE.REQUIRED_CONF_ID,
      db: '',
      embeddings_table: '',
      aggregator_table: '',
    },
    entity: {
      entity: '',
      store_id: '',
    },
    model: {
      entity: '',
      model: '',
      embedding_store_enabled: true,
      embedding_store_ttl: 3600,
      model_config: {
        distance_function: 'EUCLIDEAN',
        vector_dimension: 128,
      },
      model_type: 'DELTA',
      mq_id: '',
      job_frequency: 'FREQ_1W',
      training_data_path: '',
      number_of_partitions: BUSINESS_RULES.MODEL.REQUIRED_PARTITIONS,
      topic_name: '',
      metadata: '{}',
      failure_producer_mq_id: 0,
    },
    variant: {
      entity: '',
      model: '',
      variant: '',
      vector_db_type: BUSINESS_RULES.VARIANT.FORCED_VECTOR_DB_TYPE,
      type: BUSINESS_RULES.VARIANT.FORCED_TYPE,
      caching_configuration: {
        in_memory_caching_enabled: true,
        in_memory_cache_ttl_seconds: 300,
        distributed_caching_enabled: false,
        distributed_cache_ttl_seconds: 600,
        embedding_retrieval_in_memory_config: { enabled: true, ttl: 60 },
        embedding_retrieval_distributed_config: { enabled: false, ttl: 300 },
        dot_product_in_memory_config: { enabled: true, ttl: 30 },
        dot_product_distributed_config: { enabled: false, ttl: 120 },
      },
      filter_configuration: {
        criteria: [],
      },
      vector_db_config: {},
      rate_limiter: {},
      rt_partition: 0,
    },
    filter: {
      entity: '',
      filter: {
        column_name: '',
        filter_value: '',
        default_value: '',
      },
    },
    jobFrequency: {
      job_frequency: '',
    },
    qdrantCluster: {
      node_conf: {
        count: 1,
        instance_type: 'm5.large',
        storage: '100GB',
      },
      qdrant_version: 'v1.7.0',
      dns_subdomain: '',
      project: 'embedding-platform',
    },
  };

  return defaults[type] || {};
};

// REQUEST LIST NORMALIZATION (API contract: payload is JSON string)

/**
 * Normalize a request list response by parsing each item's payload (JSON string) into an object
 */
export const normalizeRequestList = (response, arrayKey, spreadPayload = false) => {
  const arr = response?.[arrayKey];
  if (!Array.isArray(arr)) return response;

  const normalized = arr.map((item) => {
    let parsed = {};
    try {
      parsed =
        typeof item.payload === 'string'
          ? JSON.parse(item.payload || '{}')
          : item.payload != null
            ? item.payload
            : {};
    } catch (e) {
      parsed = {};
    }
    if (spreadPayload) {
      return { ...item, payload: parsed, ...parsed };
    }
    return { ...item, payload: parsed };
  });

  return { ...response, [arrayKey]: normalized };
};

export default {
  // Business Rules
  validateStorePayload,
  validateEntityPayload,
  validateModelPayload,

  // Formatting
  formatDate,

  // Data Transformation
  transformToTableData,
  normalizeRequestList,

  // Business Logic
  getDefaultFormValues,
};
