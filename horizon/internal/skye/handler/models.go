package handler

import (
	"time"
)

// Common response structures
type Response struct {
	Error string      `json:"error"`
	Data  interface{} `json:"data"`
}

type Message struct {
	Message string `json:"message"`
}

type RequestStatus struct {
	RequestID         int                `json:"request_id"`
	Status            string             `json:"status"`
	Message           string             `json:"message"`
	CreatedAt         time.Time          `json:"created_at"`
	ProcessingDetails *ProcessingDetails `json:"processing_details,omitempty"`
}

type ProcessingDetails struct {
	CurrentStep         string     `json:"current_step"`
	CompletedSteps      []string   `json:"completed_steps"`
	RemainingSteps      []string   `json:"remaining_steps"`
	ExternalJobID       string     `json:"external_job_id,omitempty"`
	EstimatedCompletion *time.Time `json:"estimated_completion,omitempty"`
}

// Common approval request structure
type ApprovalRequest struct {
	AdminID          string `json:"admin_id"`
	ApprovalDecision string `json:"approval_decision"` // APPROVED, REJECTED, NEEDS_MODIFICATION
	ApprovalComments string `json:"approval_comments"`
	// Admin-configurable fields for variant approval
	AdminVectorDBConfig *VectorDBConfig `json:"admin_vector_db_config,omitempty"`
	AdminRateLimiter    *RateLimiter    `json:"admin_rate_limiter,omitempty"`
	AdminRTPartition    *int            `json:"admin_rt_partition,omitempty"`
}

type ApprovalResponse struct {
	RequestID        int       `json:"request_id"`
	Status           string    `json:"status"`
	Message          string    `json:"message"`
	ApprovedBy       string    `json:"approved_by"`
	ApprovedAt       time.Time `json:"approved_at"`
	ProcessingStatus string    `json:"processing_status"`
}

// STORE OPERATIONS

type StoreRequestPayload struct {
	ConfID          int    `json:"conf_id"`
	DB              string `json:"db"`
	EmbeddingsTable string `json:"embeddings_table"`
	AggregatorTable string `json:"aggregator_table"`
}

type StoreRegisterRequest struct {
	Requestor   string              `json:"requestor"`
	Reason      string              `json:"reason"`
	RequestType string              `json:"request_type"`
	Payload     StoreRequestPayload `json:"payload"`
}

type StoreInfo struct {
	ID              string `json:"id"`
	ConfID          int    `json:"conf_id"`
	DB              string `json:"db"`
	EmbeddingsTable string `json:"embeddings_table"`
	AggregatorTable string `json:"aggregator_table"`
}

type StoreListResponse struct {
	Stores []StoreInfo `json:"stores"`
}

// Add structure for individual store request from database
type StoreRequestInfo struct {
	RequestID   int                 `json:"request_id"`
	Reason      string              `json:"reason"`
	Payload     StoreRequestPayload `json:"payload"`
	RequestType string              `json:"request_type"`
	CreatedBy   string              `json:"created_by"`
	ApprovedBy  string              `json:"approved_by,omitempty"`
	Status      string              `json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// Response for listing all store requests
type StoreRequestListResponse struct {
	StoreRequests []StoreRequestInfo `json:"store_requests"`
	TotalCount    int                `json:"total_count"`
}

// ENTITY OPERATIONS

type EntityRequestPayload struct {
	Entity  string `json:"entity"`
	StoreID string `json:"store_id"`
}

type EntityRegisterRequest struct {
	Requestor   string               `json:"requestor"`
	Reason      string               `json:"reason"`
	RequestType string               `json:"request_type"`
	Payload     EntityRequestPayload `json:"payload"`
}

type EntityInfo struct {
	Name    string `json:"name"`
	StoreID string `json:"store_id"`
}

type EntityListResponse struct {
	Entities []EntityInfo `json:"entities"`
}

// Add structure for individual entity request from database
type EntityRequestInfo struct {
	RequestID   int                  `json:"request_id"`
	Reason      string               `json:"reason"`
	Payload     EntityRequestPayload `json:"payload"`
	RequestType string               `json:"request_type"`
	CreatedBy   string               `json:"created_by"`
	ApprovedBy  string               `json:"approved_by,omitempty"`
	Status      string               `json:"status"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// Response for listing all entity requests
type EntityRequestListResponse struct {
	EntityRequests []EntityRequestInfo `json:"entity_requests"`
	TotalCount     int                 `json:"total_count"`
}

// MODEL OPERATIONS

type ModelConfig struct {
	DistanceFunction string `json:"distance_function"`
	VectorDimension  int    `json:"vector_dimension"`
}

type ModelRequestPayload struct {
	Entity                string      `json:"entity"`
	Model                 string      `json:"model"`
	EmbeddingStoreEnabled bool        `json:"embedding_store_enabled"`
	EmbeddingStoreTTL     int         `json:"embedding_store_ttl"`
	ModelConfig           ModelConfig `json:"model_config"`
	ModelType             string      `json:"model_type"` // RESET, DELTA
	MQID                  int         `json:"mq_id"`
	JobFrequency          string      `json:"job_frequency"`
	TrainingDataPath      string      `json:"training_data_path"`
	NumberOfPartitions    int         `json:"number_of_partitions"`
	TopicName             string      `json:"topic_name"`
	Metadata              string      `json:"metadata"`
	FailureProducerMqId   int         `json:"failure_producer_mq_id"`
}

type ModelRegisterRequest struct {
	Requestor string              `json:"requestor"`
	Reason    string              `json:"reason"`
	Payload   ModelRequestPayload `json:"payload"`
}

type ModelEditRequestPayload struct {
	Entity  string                 `json:"entity"`
	Model   string                 `json:"model"`
	Updates map[string]interface{} `json:"updates"`
}

type ModelEditRequest struct {
	Requestor string                  `json:"requestor"`
	Reason    string                  `json:"reason"`
	Payload   ModelEditRequestPayload `json:"payload"`
}

type ModelInfo struct {
	Entity                string `json:"entity"`
	Model                 string `json:"model"`
	ModelType             string `json:"model_type"`
	EmbeddingStoreEnabled bool   `json:"embedding_store_enabled"`
	EmbeddingStoreTTL     int    `json:"embedding_store_ttl"`
	VectorDimension       int    `json:"vector_dimension"`
	DistanceFunction      string `json:"distance_function"`
	Status                string `json:"status"`
	VariantCount          int    `json:"variant_count"`
	MQID                  int    `json:"mq_id"`
	TrainingDataPath      string `json:"training_data_path"`
	TopicName             string `json:"topic_name"`
	Metadata              string `json:"metadata"`
	FailureProducerMqId   int    `json:"failure_producer_mq_id"`
	// PartitionStatus       map[string]int `json:"partition_status"`
}

type ModelListResponse struct {
	Models []ModelInfo `json:"models"`
}

// Add structure for individual model request from database
type ModelRequestInfo struct {
	RequestID   int                 `json:"request_id"`
	Reason      string              `json:"reason"`
	Payload     ModelRequestPayload `json:"payload"`
	RequestType string              `json:"request_type"`
	CreatedBy   string              `json:"created_by"`
	ApprovedBy  string              `json:"approved_by,omitempty"`
	Status      string              `json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// Response for listing all model requests
type ModelRequestListResponse struct {
	ModelRequests []ModelRequestInfo `json:"model_requests"`
	TotalCount    int                `json:"total_count"`
}

// VARIANT OPERATIONS

// HTTP2Config and VectorDBParams are no longer needed since VectorDBConfig is now flexible

// VectorDBConfig can be any JSON structure - different vector DBs have different requirements
type VectorDBConfig map[string]interface{}

type CachingConfiguration struct {
	InMemoryCachingEnabled              bool                     `json:"in_memory_caching_enabled"`
	InMemoryCacheTTLSeconds             int                      `json:"in_memory_cache_ttl_seconds"`
	DistributedCachingEnabled           bool                     `json:"distributed_caching_enabled"`
	DistributedCacheTTLSeconds          int                      `json:"distributed_cache_ttl_seconds"`
	EmbeddingRetrievalInMemoryConfig    EmbeddingRetrievalConfig `json:"embedding_retrieval_in_memory_config"`
	EmbeddingRetrievalDistributedConfig EmbeddingRetrievalConfig `json:"embedding_retrieval_distributed_config"`
	DotProductInMemoryConfig            DotProductConfig         `json:"dot_product_in_memory_config"`
	DotProductDistributedConfig         DotProductConfig         `json:"dot_product_distributed_config"`
}

type FilterCriteria struct {
	ColumnName   string `json:"column_name"`
	FilterValue  string `json:"filter_value"`
	Operator     string `json:"operator,omitempty"` // Optional operator field
	DefaultValue string `json:"default_value"`
}

type FilterConfiguration struct {
	Criteria []FilterCriteria `json:"criteria"`
}

type EmbeddingRetrievalConfig struct {
	Enabled bool `json:"enabled"`
	TTL     int  `json:"ttl"`
}

type DotProductConfig struct {
	Enabled bool `json:"enabled"`
	TTL     int  `json:"ttl"`
}

type RateLimiter struct {
	RateLimit  int `json:"rate_limit"`
	BurstLimit int `json:"burst_limit"`
}

type VariantRequestPayload struct {
	Entity               string               `json:"entity"`
	Model                string               `json:"model"`
	Variant              string               `json:"variant"`
	VectorDBType         string               `json:"vector_db_type"`
	Type                 string               `json:"type"` // SCALE_UP, EXPERIMENT
	CachingConfiguration CachingConfiguration `json:"caching_configuration"`
	FilterConfiguration  FilterConfiguration  `json:"filter_configuration"`
	VectorDBConfig       VectorDBConfig       `json:"vector_db_config"`
	RateLimiter          RateLimiter          `json:"rate_limiter"`
	RTPartition          int                  `json:"rt_partition"`
}

type VariantRegisterRequest struct {
	Requestor string                `json:"requestor"`
	Reason    string                `json:"reason"`
	Payload   VariantRequestPayload `json:"payload"`
}

type VariantEditRequestPayload struct {
	Entity                     string              `json:"entity"`
	Model                      string              `json:"model"`
	Variant                    string              `json:"variant"`
	InMemoryCachingEnabled     bool                `json:"in_memory_caching_enabled"`
	InMemoryCacheTTLSeconds    int                 `json:"in_memory_cache_ttl_seconds"`
	DistributedCachingEnabled  bool                `json:"distributed_caching_enabled"`
	DistributedCacheTTLSeconds int                 `json:"distributed_cache_ttl_seconds"`
	Filter                     FilterConfiguration `json:"filter"`
	VectorDBConfig             VectorDBConfig      `json:"vector_db_config"`
	RateLimiter                RateLimiter         `json:"rate_limiter"`
	RTPartition                int                 `json:"rt_partition"`
}

type VariantEditRequest struct {
	Requestor string                    `json:"requestor"`
	Reason    string                    `json:"reason"`
	Payload   VariantEditRequestPayload `json:"payload"`
}

type VariantInfo struct {
	Entity                    string    `json:"entity"`
	Model                     string    `json:"model"`
	Variant                   string    `json:"variant"`
	Type                      string    `json:"type"`
	VectorDBType              string    `json:"vector_db_type"`
	Status                    string    `json:"status"`
	CreatedAt                 time.Time `json:"created_at"`
	LastUpdated               time.Time `json:"last_updated"`
	ClusterHost               string    `json:"cluster_host"`
	InMemoryCachingEnabled    bool      `json:"in_memory_caching_enabled"`
	DistributedCachingEnabled bool      `json:"distributed_caching_enabled"`
}

type VariantListResponse struct {
	Variants []VariantInfo `json:"variants"`
}

// Add structure for individual variant request from database
type VariantRequestInfo struct {
	RequestID   int                   `json:"request_id"`
	Reason      string                `json:"reason"`
	Payload     VariantRequestPayload `json:"payload"`
	RequestType string                `json:"request_type"`
	CreatedBy   string                `json:"created_by"`
	ApprovedBy  string                `json:"approved_by,omitempty"`
	Status      string                `json:"status"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

// Response for listing all variant requests
type VariantRequestListResponse struct {
	VariantRequests []VariantRequestInfo `json:"variant_requests"`
	TotalCount      int                  `json:"total_count"`
}

// FILTER OPERATIONS

type FilterRequestPayload struct {
	Entity string         `json:"entity"`
	Filter FilterCriteria `json:"filter"`
}

type FilterRegisterRequest struct {
	Requestor string               `json:"requestor"`
	Reason    string               `json:"reason"`
	Payload   FilterRequestPayload `json:"payload"`
}

type FilterInfo struct {
	ID           string    `json:"id"`
	ColumnName   string    `json:"column_name"`
	FilterValue  string    `json:"filter_value"`
	DefaultValue string    `json:"default_value"`
	DisplayName  string    `json:"display_name"`
	Description  string    `json:"description"`
	Entity       string    `json:"entity"`
	DataType     string    `json:"data_type"`
	IsActive     bool      `json:"is_active"`
	UsageCount   int       `json:"usage_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// New simplified filter criteria structure for response
type FilterCriteriaResponse struct {
	ColumnName   string `json:"column_name"`
	FilterValue  string `json:"filter_value"`
	DefaultValue string `json:"default_value"`
}

// Entity-grouped filter response
type EntityFilterGroup struct {
	Entity  string                   `json:"entity"`
	Filters []FilterCriteriaResponse `json:"filters"`
}

type FilterListResponse struct {
	FilterGroups []EntityFilterGroup `json:"filter_groups"`
}

// Query options for filtering
type FilterQueryOptions struct {
	Entity     string `json:"entity"`
	ColumnName string `json:"column_name"`
	DataType   string `json:"data_type"`
	ActiveOnly bool   `json:"active_only"`
}

// ==================== JOB FREQUENCY OPERATIONS ====================

type JobFrequencyRequestPayload struct {
	JobFrequency string `json:"job_frequency"`
}

type JobFrequencyRegisterRequest struct {
	Requestor string                     `json:"requestor"`
	Reason    string                     `json:"reason"`
	Payload   JobFrequencyRequestPayload `json:"payload"`
}

type JobFrequencyInfo struct {
	ID          string    `json:"id"`
	Frequency   string    `json:"frequency"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

type JobFrequencyListResponse struct {
	JobFrequencies []string `json:"job_frequencies"`
	TotalCount     int      `json:"total_count"`
}

// Add structure for individual job frequency request from database
type JobFrequencyRequestInfo struct {
	RequestID   int                        `json:"request_id"`
	Reason      string                     `json:"reason"`
	Payload     JobFrequencyRequestPayload `json:"payload"`
	RequestType string                     `json:"request_type"`
	CreatedBy   string                     `json:"created_by"`
	ApprovedBy  string                     `json:"approved_by,omitempty"`
	Status      string                     `json:"status"`
	CreatedAt   time.Time                  `json:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
}

// Response for listing all job frequency requests
type JobFrequencyRequestListResponse struct {
	JobFrequencyRequests []JobFrequencyRequestInfo `json:"job_frequency_requests"`
	TotalCount           int                       `json:"total_count"`
}

// Add structure for individual filter request from database
type FilterRequestInfo struct {
	RequestID   int                  `json:"request_id"`
	Reason      string               `json:"reason"`
	Payload     FilterRequestPayload `json:"payload"`
	RequestType string               `json:"request_type"`
	CreatedBy   string               `json:"created_by"`
	ApprovedBy  string               `json:"approved_by,omitempty"`
	Status      string               `json:"status"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// Response for listing all filter requests
type FilterRequestListResponse struct {
	FilterRequests []FilterRequestInfo `json:"filter_requests"`
	TotalCount     int                 `json:"total_count"`
}

// DEPLOYMENT OPERATIONS

// Qdrant Cluster Operations
type NodeConfiguration struct {
	Count        int    `json:"count"`
	InstanceType string `json:"instance_type"`
	Storage      string `json:"storage"`
}

type QdrantClusterRequestPayload struct {
	NodeConf      NodeConfiguration `json:"node_conf"`
	QdrantVersion string            `json:"qdrant_version"`
	DNSSubdomain  string            `json:"dns_subdomain"`
	Project       string            `json:"project"`
}

type QdrantClusterRequest struct {
	Requestor string                      `json:"requestor"`
	Reason    string                      `json:"reason"`
	Payload   QdrantClusterRequestPayload `json:"payload"`
}

type InstanceDetails struct {
	InstanceID       string `json:"instance_id"`
	PrivateIP        string `json:"private_ip"`
	PublicIP         string `json:"public_ip"`
	AvailabilityZone string `json:"availability_zone"`
	Status           string `json:"status"`
}

type MonitoringInfo struct {
	GrafanaDashboardURL   string     `json:"grafana_dashboard_url"`
	ClusterHealthEndpoint string     `json:"cluster_health_endpoint"`
	LastHealthCheck       *time.Time `json:"last_health_check,omitempty"`
	HealthStatus          string     `json:"health_status,omitempty"`
}

type ClusterInfo struct {
	ClusterID         string            `json:"cluster_id"`
	DNSSubdomain      string            `json:"dns_subdomain"`
	Project           string            `json:"project"`
	Environment       string            `json:"environment"`
	Status            string            `json:"status"`
	NodeConfiguration NodeConfiguration `json:"node_configuration"`
	QdrantVersion     string            `json:"qdrant_version"`
	InstanceDetails   []InstanceDetails `json:"instance_details"`
	Monitoring        MonitoringInfo    `json:"monitoring"`
	CreatedAt         time.Time         `json:"created_at"`
	LastUpdated       time.Time         `json:"last_updated"`
}

type ClusterListResponse struct {
	Clusters      []ClusterInfo   `json:"clusters"`
	TotalClusters int             `json:"total_clusters"`
	Metadata      ClusterMetadata `json:"metadata"`
}

type ClusterMetadata struct {
	TotalInstances int      `json:"total_instances"`
	TotalStorage   string   `json:"total_storage"`
	Environments   []string `json:"environments"`
	Projects       []string `json:"projects"`
}

// Variant Promotion Operations
type VariantPromotionRequestPayload struct {
	Entity       string `json:"entity"`
	Model        string `json:"model"`
	VectorDBType string `json:"vector_db_type"`
	Variant      string `json:"variant"`
	Host         string `json:"host"`
}

type VariantPromotionRequest struct {
	Requestor string                         `json:"requestor"`
	Reason    string                         `json:"reason"`
	Payload   VariantPromotionRequestPayload `json:"payload"`
}

// Variant Onboarding Operations
type VariantOnboardingRequestPayload struct {
	Entity       string `json:"entity"`
	Model        string `json:"model"`
	Variant      string `json:"variant"`
	VectorDBType string `json:"vector_db_type"`
}

type VariantOnboardingRequest struct {
	Requestor string                          `json:"requestor"`
	Reason    string                          `json:"reason"`
	Payload   VariantOnboardingRequestPayload `json:"payload"`
}

// Add structure for individual variant onboarding request from database
type VariantOnboardingRequestInfo struct {
	RequestID   int                             `json:"request_id"`
	Reason      string                          `json:"reason"`
	Payload     VariantOnboardingRequestPayload `json:"payload"`
	RequestType string                          `json:"request_type"`
	CreatedBy   string                          `json:"created_by"`
	ApprovedBy  string                          `json:"approved_by,omitempty"`
	Status      string                          `json:"status"`
	CreatedAt   time.Time                       `json:"created_at"`
	UpdatedAt   time.Time                       `json:"updated_at"`
}

// Response for listing all variant onboarding requests
type VariantOnboardingRequestListResponse struct {
	VariantOnboardingRequests []VariantOnboardingRequestInfo `json:"variant_onboarding_requests"`
	TotalCount                int                            `json:"total_count"`
}

// Onboarded Variant Information
type OnboardedVariantInfo struct {
	Entity       string    `json:"entity"`
	Model        string    `json:"model"`
	Variant      string    `json:"variant"`
	VectorDBType string    `json:"vector_db_type"`
	Status       string    `json:"status"` // "ONBOARDED", "IN_PROGRESS", "FAILED"
	OnboardedAt  time.Time `json:"onboarded_at"`
	RequestID    int       `json:"request_id"`
	CreatedBy    string    `json:"created_by"`
	ApprovedBy   string    `json:"approved_by"`
}

// Response for listing onboarded variants
type OnboardedVariantListResponse struct {
	OnboardedVariants []OnboardedVariantInfo `json:"onboarded_variants"`
	TotalCount        int                    `json:"total_count"`
}

// MQ ID TO TOPIC OPERATIONS
type MQIdTopicMapping struct {
	MQID  int    `json:"mq_id"`
	Topic string `json:"topic"`
}

type MQIdTopicsResponse struct {
	Mappings   []MQIdTopicMapping `json:"mappings"`
	TotalCount int                `json:"total_count"`
}

// VARIANTS LIST OPERATIONS
type VariantsListResponse struct {
	Variants   []string `json:"variants"`
	TotalCount int      `json:"total_count"`
}
