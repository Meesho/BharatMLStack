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

  // =============================================================================
  // STORE MANAGEMENT ENDPOINTS
  // =============================================================================

  /**
   * Register Store
   */
  async registerStore(payload) {
    return this.makeRequest('/requests/store/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Approve Store
   */
  async approveStore(payload) {
    return this.makeRequest('/requests/store/approve?request_id=' + payload.request_id, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Get Stores
   */
  async getStores() {
    return this.makeRequest('/data/stores');
  }

  /**
   * Get Store Requests
   * API returns payload as JSON string; normalized to payload object per contract.
   */
  async getStoreRequests() {
    const response = await this.makeRequest('/data/store-requests');
    return normalizeRequestList(response, 'store_requests');
  }

  // =============================================================================
  // ENTITY MANAGEMENT ENDPOINTS
  // =============================================================================

  /**
   * Register Entity
   */
  async registerEntity(payload) {
    return this.makeRequest('/requests/entity/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Approve Entity
   */
  async approveEntity(payload) {
    return this.makeRequest('/requests/entity/approve?request_id=' + payload.request_id, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Get Entities
   */
  async getEntities() {
    return this.makeRequest('/data/entities');
  }

  /**
   * Get Entity Requests
   * API returns payload as JSON string; normalized to payload object per contract.
   */
  async getEntityRequests() {
    const response = await this.makeRequest('/data/entity-requests');
    return normalizeRequestList(response, 'entity_requests');
  }

  // =============================================================================
  // MODEL MANAGEMENT ENDPOINTS
  // =============================================================================

  /**
   * Register Model
   */
  async registerModel(payload) {
    return this.makeRequest('/requests/model/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Edit Model
   */
  async editModel(payload) {
    return this.makeRequest('/requests/model/edit', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Approve Model
   */
  async approveModel(payload) {
    return this.makeRequest('/requests/model/approve?request_id=' + payload.request_id, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Approve Model Edit
   */
  async approveModelEdit(payload) {
    return this.makeRequest('/requests/model/edit/approve?request_id=' + payload.request_id, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Get Models
   */
  async getModels(params = {}) {
    const queryString = new URLSearchParams(params).toString();
    return this.makeRequest(`/data/models${queryString ? `?${queryString}` : ''}`);
  }

  /**
   * Get Model Requests
   * API returns payload as JSON string; normalized to payload object and spread for table (entity, model, model_type).
   */
  async getModelRequests() {
    const response = await this.makeRequest('/data/model-requests');
    return normalizeRequestList(response, 'model_requests', true);
  }

  // =============================================================================
  // VARIANT MANAGEMENT ENDPOINTS
  // =============================================================================

  /**
   * Register Variant
   */
  async registerVariant(payload) {
    return this.makeRequest('/requests/variant/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Edit Variant
   */
  async editVariant(payload) {
    return this.makeRequest('/requests/variant/edit', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Approve Variant
   */
  async approveVariant(payload) {
    return this.makeRequest('/requests/variant/approve?request_id=' + payload.request_id, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Approve Variant Edit
   */
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
   * API returns payload as JSON string; normalized to payload object per contract.
   */
  async getVariantRequests() {
    const response = await this.makeRequest('/data/variant-requests');
    return normalizeRequestList(response, 'variant_requests');
  }

  // =============================================================================
  // FILTER MANAGEMENT ENDPOINTS
  // =============================================================================

  /**
   * Register Filter
   */
  async registerFilter(payload) {
    return this.makeRequest('/requests/filter/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Approve Filter
   */
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

  /**
   * Get All Filters
   * Fetches all filters (e.g. for Filter Discovery).
   */
  async getAllFilters() {
    return this.makeRequest('/data/all-filters');
  }

  /**
   * Get Filter Requests
   * API returns payload as JSON string; normalized to payload object per contract.
   */
  async getFilterRequests() {
    const response = await this.makeRequest('/data/filter-requests');
    return normalizeRequestList(response, 'filter_requests');
  }

  // =============================================================================
  // JOB FREQUENCY MANAGEMENT ENDPOINTS
  // =============================================================================

  /**
   * Register Job Frequency
   */
  async registerJobFrequency(payload) {
    return this.makeRequest('/requests/job-frequency/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Approve Job Frequency
   */
  async approveJobFrequency(payload) {
    return this.makeRequest('/requests/job-frequency/approve?request_id=' + payload.request_id, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Get Job Frequencies
   */
  async getJobFrequencies() {
    return this.makeRequest('/data/job-frequencies');
  }

  /**
   * Get Job Frequency Requests
   * API returns payload as JSON string; normalized to payload object per contract.
   */
  async getJobFrequencyRequests() {
    const response = await this.makeRequest('/data/job-frequency-requests');
    return normalizeRequestList(response, 'job_frequency_requests');
  }

  // =============================================================================
  // VARIANT PROMOTION & ONBOARDING OPERATIONS ENDPOINTS
  // =============================================================================

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

  /**
   * Approve Variant Promotion
   */
  async approveVariantPromotion(payload) {
    return this.makeRequest(`/requests/variant/promote/approve?request_id=${payload.request_id}`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Onboard Variant
   */
  async onboardVariant(payload) {
    return this.makeRequest('/requests/variant/onboard', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Approve Variant Onboarding
   */
  async approveVariantOnboarding(payload) {
    return this.makeRequest(`/requests/variant/onboard/approve?request_id=${payload.request_id}`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Get All Variant Onboarding Requests
   * API returns payload as JSON string; normalized to payload object per contract.
   */
  async getAllVariantOnboardingRequests() {
    const response = await this.makeRequest('/data/variant-onboarding/requests');
    return normalizeRequestList(response, 'variant_onboarding_requests');
  }

  /**
   * Get Variant Onboarding Tasks
   */
  async getVariantOnboardingTasks() {
    return this.makeRequest('/data/variant-onboarding/tasks');
  }

  // =============================================================================
  // MQ ID TO TOPICS MAPPING ENDPOINTS
  // =============================================================================

  /**
   * Get MQ ID to Topics Mapping
   */
  async getMQIdTopics() {
    return this.makeRequest('/data/mq-id-topics');
  }

  // =============================================================================
  // VARIANTS LIST ENDPOINTS
  // =============================================================================

  /**
   * Get Variants List
   */
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

// Export individual methods for convenience
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

