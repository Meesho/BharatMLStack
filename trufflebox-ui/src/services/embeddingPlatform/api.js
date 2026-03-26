import { REACT_APP_HORIZON_BASE_URL } from '../../config';
import { normalizeRequestList } from './utils';

const BASE_URL = `${REACT_APP_HORIZON_BASE_URL}/api/v1/horizon/skye`;

/**
 * API Service for Embedding Platform (Skye)
 * Handles all endpoints with proper error handling and authentication
 */
class EmbeddingPlatformAPI {
  constructor() {
    this.baseUrl = BASE_URL;
  }

  /**
   * Get authentication token from user context
   */
  getAuthToken() {
    const userStr = localStorage.getItem('user');
    if (userStr) {
      const user = JSON.parse(userStr);
      return user.token;
    }
    return null;
  }

  /**
   * Base fetch method with error handling
   */
  async makeRequest(endpoint, options = {}) {
    const token = this.getAuthToken();
    const url = `${this.baseUrl}${endpoint}`;

    const config = {
      headers: {
        'Content-Type': 'application/json',
        ...(token && { Authorization: `Bearer ${token}` }),
        ...options.headers,
      },
      ...options,
    };

    try {
      const response = await fetch(url, config);
      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || `HTTP ${response.status}: ${response.statusText}`);
      }

      return data;
    } catch (error) {
      console.error(`API Error for ${endpoint}:`, error);
      throw error;
    }
  }


  async registerStore(payload) {
    return this.makeRequest('/requests/store/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async approveStore(payload) {
    return this.makeRequest('/requests/store/approve?request_id=' + payload.request_id, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async getStores() {
    return this.makeRequest('/data/stores');
  }

  /**
   * Get Store Requests
   * API returns payload as JSON string; normalized to payload object and key fields flattened for UI.
   */
  async getStoreRequests() {
    const response = await this.makeRequest('/data/store-requests');
    return normalizeRequestList(response, {
      listKey: 'store_requests',
      fallbackListKey: 'stores',
      totalCountKey: 'total_count',
      flattenFromPayload: ['conf_id', 'db', 'embeddings_table', 'aggregator_table'],
    });
  }


  async registerEntity(payload) {
    return this.makeRequest('/requests/entity/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async approveEntity(payload) {
    return this.makeRequest('/requests/entity/approve?request_id=' + payload.request_id, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async getEntities() {
    return this.makeRequest('/data/entities');
  }

  /**
   * Get Entity Requests
   * API returns payload as JSON string; normalized to payload object and entity/store_id flattened for UI.
   */
  async getEntityRequests() {
    const response = await this.makeRequest('/data/entity-requests');
    return normalizeRequestList(response, {
      listKey: 'entity_requests',
      fallbackListKey: 'entities',
      totalCountKey: 'total_count',
      flattenFromPayload: ['entity', 'store_id'],
    });
  }


  async registerModel(payload) {
    return this.makeRequest('/requests/model/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async editModel(payload) {
    return this.makeRequest('/requests/model/edit', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async approveModel(payload) {
    return this.makeRequest('/requests/model/approve?request_id=' + payload.request_id, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async approveModelEdit(payload) {
    return this.makeRequest('/requests/model/edit/approve?request_id=' + payload.request_id, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async getModels(params = {}) {
    const queryString = new URLSearchParams(params).toString();
    return this.makeRequest(`/data/models${queryString ? `?${queryString}` : ''}`);
  }

  /**
   * Get Model Requests
   * API returns payload as JSON string; normalized to payload object and entity/model/model_type flattened for UI.
   */
  async getModelRequests() {
    const response = await this.makeRequest('/data/model-requests');
    return normalizeRequestList(response, {
      listKey: 'model_requests',
      fallbackListKey: 'models',
      totalCountKey: 'total_count',
      flattenFromPayload: ['entity', 'model', 'model_type', 'job_frequency', 'training_data_path'],
    });
  }


  async registerVariant(payload) {
    return this.makeRequest('/requests/variant/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async editVariant(payload) {
    return this.makeRequest('/requests/variant/edit', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async approveVariant(payload) {
    return this.makeRequest('/requests/variant/approve?request_id=' + payload.request_id, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async approveVariantEdit(payload) {
    return this.makeRequest('/requests/variant/edit/approve?request_id=' + payload.request_id, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Get Variants
   * @param {Object} params - Query parameters (entity, model)
   */
  async getVariants(params = {}) {
    const queryString = new URLSearchParams(params).toString();
    const endpoint = queryString ? `/data/variants?${queryString}` : '/data/variants';
    return this.makeRequest(endpoint);
  }

  /**
   * Get Variant Requests
   * API returns payload as JSON string; normalized to payload object, entity/model/variant flattened, filter_config normalized for UI.
   */
  async getVariantRequests() {
    const response = await this.makeRequest('/data/variant-requests');
    return normalizeRequestList(response, {
      listKey: 'variant_requests',
      fallbackListKey: 'variants',
      totalCountKey: 'total_count',
      flattenFromPayload: ['entity', 'model', 'variant', 'vector_db_type', 'type', 'caching_configuration', 'filter_configuration', 'vector_db_config'],
      normalizeVariantFilterConfig: true,
    });
  }


  async registerFilter(payload) {
    return this.makeRequest('/requests/filter/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async approveFilter(payload) {
    return this.makeRequest('/requests/filter/approve?request_id=' + payload.request_id, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Get Filters
   * @param {Object} params - Query parameters
   */
  async getFilters(params = {}) {
    const queryString = new URLSearchParams(params).toString();
    const endpoint = queryString ? `/data/filters?${queryString}` : '/data/filters';
    return this.makeRequest(endpoint);
  }

  async getAllFilters() {
    return this.makeRequest('/data/all-filters');
  }

  /**
   * Get Filter Requests
   * API returns payload as JSON string; normalized to payload object and entity/filter flattened for UI.
   */
  async getFilterRequests() {
    const response = await this.makeRequest('/data/filter-requests');
    return normalizeRequestList(response, {
      listKey: 'filter_requests',
      fallbackListKey: 'filters',
      totalCountKey: 'total_count',
      flattenFromPayload: ['entity', 'filter'],
    });
  }


  async registerJobFrequency(payload) {
    return this.makeRequest('/requests/job-frequency/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }


  async approveJobFrequency(payload) {
    return this.makeRequest('/requests/job-frequency/approve?request_id=' + payload.request_id, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async getJobFrequencies() {
    return this.makeRequest('/data/job-frequencies');
  }

  /**
   * Get Job Frequency Requests
   * API returns payload as JSON string; normalized to payload object and job_frequency flattened for UI.
   */
  async getJobFrequencyRequests() {
    const response = await this.makeRequest('/data/job-frequency-requests');
    return normalizeRequestList(response, {
      listKey: 'job_frequency_requests',
      fallbackListKey: 'job_frequencies',
      totalCountKey: 'total_count',
      flattenFromPayload: ['job_frequency'],
    });
  }


  /**
   * Promote Variant
   * Uses the same create/register variant endpoint with request_type set to PROMOTE.
   */
  async promoteVariant(payload) {
    return this.makeRequest('/requests/variant/register', {
      method: 'POST',
      body: JSON.stringify({ ...payload, request_type: 'PROMOTE' }),
    });
  }

  async onboardVariant(payload) {
    return this.makeRequest('/requests/variant/onboard', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async approveVariantOnboarding(payload) {
    return this.makeRequest('/requests/variant/onboard/approve?request_id=' + payload.request_id, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Get All Variant Onboarding Requests
   * GET /data/variant-onboarding/requests. Response normalized via normalizeRequestList.
   */
  async getAllVariantOnboardingRequests() {
    const response = await this.makeRequest('/data/variant-onboarding/requests');
    return normalizeRequestList(response, {
      listKey: 'variant_onboarding_requests',
      fallbackListKey: 'requests',
      totalCountKey: 'total_count',
      flattenFromPayload: ['entity', 'model', 'variant'],
    });
  }

  async getVariantOnboardingTasks() {
    return this.makeRequest('/data/variant-onboarding/tasks');
  }


  async generateVariantTestRequest(payload) {
    return this.makeRequest('/test/variant/generate-request', {
      method: 'POST',
      body: JSON.stringify({
        entity: payload.entity,
        model: payload.model,
        variant: payload.variant,
      }),
    });
  }

  async executeVariantTestRequest(requestPayload) {
    return this.makeRequest('/test/variant/execute-request', {
      method: 'POST',
      body: JSON.stringify(requestPayload),
    });
  }


  async getMQIdTopics() {
    return this.makeRequest('/data/mq-id-topics');
  }


  async getVariantsList() {
    return this.makeRequest('/data/variants-list');
  }

  /**
   * Build Skye API base URL from Horizon base URL (e.g. REACT_APP_HORIZON_PROD_BASE_URL).
   */
  getSkyeBaseUrl(horizonBaseUrl) {
    const base = (horizonBaseUrl || '').replace(/\/$/, '');
    return base ? `${base}/api/v1/horizon/skye` : '';
  }

  /**
   * Make a request to prod Horizon base URL with provided Bearer token.
   */
  async makeRequestWithAuth(horizonBaseUrl, token, endpoint, options = {}) {
    const baseUrl = this.getSkyeBaseUrl(horizonBaseUrl);
    if (!baseUrl) {
      throw new Error('Horizon base URL is required');
    }
    const url = `${baseUrl}${endpoint}`;
    const config = {
      headers: {
        'Content-Type': 'application/json',
        ...(token && { Authorization: `Bearer ${token}` }),
        ...options.headers,
      },
      ...options,
    };
    const response = await fetch(url, config);
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(data.error || data.message || `HTTP ${response.status}: ${response.statusText}`);
    }
    return data;
  }

  /**
   * Get models from prod Horizon.
   */
  async getModelsWithAuth(horizonBaseUrl, token) {
    return this.makeRequestWithAuth(horizonBaseUrl, token, '/data/models', { method: 'GET' });
  }

  /**
   * Get variants for entity+model from prod Horizon.
   */
  async getVariantsWithAuth(horizonBaseUrl, token, entity, model) {
    const queryString = new URLSearchParams({ entity: String(entity), model: String(model) }).toString();
    return this.makeRequestWithAuth(horizonBaseUrl, token, `/data/variants?${queryString}`, { method: 'GET' });
  }

  /**
   * Register model at prod Horizon (e.g. promote model to prod).
   * Payload: { requestor, reason, payload } where payload is model register payload.
   */
  async registerModelWithAuth(horizonBaseUrl, token, payload) {
    return this.makeRequestWithAuth(horizonBaseUrl, token, '/requests/model/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Register variant at prod Horizon (e.g. promote variant to prod).
   * Payload: { requestor, reason, payload, request_type? } where payload has entity, model, variant.
   */
  async registerVariantWithAuth(horizonBaseUrl, token, payload) {
    return this.makeRequestWithAuth(horizonBaseUrl, token, '/requests/variant/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

}

// Create singleton instance
const embeddingPlatformAPI = new EmbeddingPlatformAPI();

export const {
  // Store Management
  registerStore,
  approveStore,
  getStores,
  getStoreRequests,
  
  // Entity Management
  registerEntity,
  approveEntity,
  getEntities,
  getEntityRequests,
  
  // Model Management
  registerModel,
  editModel,
  approveModel,
  approveModelEdit,
  getModels,
  getModelRequests,
  
  // Variant Management
  registerVariant,
  editVariant,
  approveVariant,
  approveVariantEdit,
  getVariants,
  getVariantRequests,
  
  // Filter Management
  registerFilter,
  approveFilter,
  getFilters,
  getAllFilters,
  getFilterRequests,
  
  // Job Frequency Management
  registerJobFrequency,
  approveJobFrequency,
  getJobFrequencies,
  getJobFrequencyRequests,
  
  // Variant Promotion & Onboarding Operations
  promoteVariant,
  approveVariantPromotion,
  onboardVariant,
  approveVariantOnboarding,
  getAllVariantOnboardingRequests,
  getVariantOnboardingTasks,

  // Test Variant (Generate & Execute request)
  generateVariantTestRequest,
  executeVariantTestRequest,
  
  // MQ ID to Topics Mapping
  getMQIdTopics,
  
  // Variants List
  getVariantsList,

  // Prod base URL API 
  getSkyeBaseUrl,
  makeRequestWithAuth,
  getModelsWithAuth,
  getVariantsWithAuth,
  registerModelWithAuth,
  registerVariantWithAuth,
} = embeddingPlatformAPI;

export default embeddingPlatformAPI;
