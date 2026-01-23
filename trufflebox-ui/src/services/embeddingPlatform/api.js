import { REACT_APP_HORIZON_BASE_URL } from '../../config';

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
   */
  async getStoreRequests() {
    return this.makeRequest('/data/store-requests');
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
   */
  async getEntityRequests() {
    return this.makeRequest('/data/entity-requests');
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
   */
  async getModelRequests() {
    return this.makeRequest('/data/model-requests');
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
   */
  async getVariants() {
    return this.makeRequest('/data/variants');
  }

  /**
   * Get Variant Requests
   */
  async getVariantRequests() {
    return this.makeRequest('/data/variant-requests');
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
   * Get Filter Requests
   */
  async getFilterRequests() {
    return this.makeRequest('/data/filter-requests');
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
   */
  async getJobFrequencyRequests() {
    return this.makeRequest('/data/job-frequency-requests');
  }

  // =============================================================================
  // DEPLOYMENT OPERATIONS ENDPOINTS
  // =============================================================================

  /**
   * Create Qdrant Cluster
   */
  async createQdrantCluster(payload) {
    return this.makeRequest('/requests/qdrant/create-cluster', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Approve Qdrant Cluster
   */
  async approveQdrantCluster(payload) {
    return this.makeRequest('/requests/qdrant/approve', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Get Qdrant Clusters
   */
  async getQdrantClusters() {
    return this.makeRequest('/data/qdrant/clusters');
  }

  /**
   * Get Deployment Requests (All types: cluster, promotion, onboarding)
   */
  async getDeploymentRequests() {
    return this.makeRequest('/data/deployment-requests');
  }

  /**
   * Promote Variant
   */
  async promoteVariant(payload) {
    return this.makeRequest('/requests/variant/promote', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Approve Variant Promotion
   */
  async approveVariantPromotion(payload) {
    return this.makeRequest('/requests/variant/promote/approve', {
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
    return this.makeRequest('/requests/variant/onboard/approve', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  /**
   * Get All Variant Onboarding Requests
   */
  async getAllVariantOnboardingRequests() {
    return this.makeRequest('/data/variant-onboarding/requests');
  }

  /**
   * Get Onboarded Variants
   */
  async getOnboardedVariants() {
    return this.makeRequest('/data/variant-onboarding/variants');
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
  getFilterRequests,
  
  // Job Frequency Management
  registerJobFrequency,
  approveJobFrequency,
  getJobFrequencies,
  getJobFrequencyRequests,
  
  // Deployment Operations
  createQdrantCluster,
  approveQdrantCluster,
  getQdrantClusters,
  getDeploymentRequests,
  promoteVariant,
  approveVariantPromotion,
  onboardVariant,
  approveVariantOnboarding,
  getAllVariantOnboardingRequests,
  getOnboardedVariants,
  
  // MQ ID to Topics Mapping
  getMQIdTopics,
  
  // Variants List
  getVariantsList,
} = embeddingPlatformAPI;

export default embeddingPlatformAPI;

