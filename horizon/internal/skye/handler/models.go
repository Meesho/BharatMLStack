package handler

import (
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/entity_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/filter_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/job_frequency_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/model_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/store_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_onboarding_requests"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_onboarding_tasks"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_requests"
	skyeEtcd "github.com/Meesho/BharatMLStack/horizon/internal/skye/etcd"
	"github.com/Meesho/BharatMLStack/horizon/internal/skye/etcd/enums"
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
	AdminID             string                  `json:"admin_id"`
	ApprovalDecision    string                  `json:"approval_decision"` // APPROVED, REJECTED, NEEDS_MODIFICATION
	ApprovalComments    string                  `json:"approval_comments"`
	AdminVectorDBConfig skyeEtcd.VectorDbConfig `json:"admin_vector_db_config,omitempty"`
	AdminRateLimiter    skyeEtcd.RateLimiter    `json:"admin_rate_limiter,omitempty"`
	AdminRTPartition    int                     `json:"admin_rt_partition,omitempty"`
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

type StoreRequestListResponse struct {
	StoreRequests []store_requests.StoreRequest `json:"store_requests"`
	TotalCount    int                           `json:"total_count"`
}

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

type EntityRequestListResponse struct {
	EntityRequests []entity_requests.EntityRequest `json:"entity_requests"`
	TotalCount     int                             `json:"total_count"`
}

type ModelRequestPayload struct {
	Entity                string               `json:"entity"`
	Model                 string               `json:"model"`
	EmbeddingStoreEnabled bool                 `json:"embedding_store_enabled"`
	EmbeddingStoreTTL     int                  `json:"embedding_store_ttl"`
	ModelConfig           skyeEtcd.ModelConfig `json:"model_config"`
	ModelType             string               `json:"model_type"` // RESET, DELTA
	MQID                  int                  `json:"mq_id"`
	JobFrequency          string               `json:"job_frequency"`
	TrainingDataPath      string               `json:"training_data_path"`
	NumberOfPartitions    int                  `json:"number_of_partitions"`
	TopicName             string               `json:"topic_name"`
	Metadata              skyeEtcd.Metadata    `json:"metadata"`
	FailureProducerMqId   int                  `json:"failure_producer_mq_id"`
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

type ModelListResponse struct {
	Models map[string]skyeEtcd.Models `json:"models"`
}

type ModelRequestListResponse struct {
	ModelRequests []model_requests.ModelRequest `json:"model_requests"`
	TotalCount    int                           `json:"total_count"`
}

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
	Operator     string `json:"operator,omitempty"`
	DefaultValue string `json:"default_value"`
}

type FilterConfiguration struct {
	Criteria []skyeEtcd.Criteria `json:"criteria"`
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
	Type                 enums.Type           `json:"type"`
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
	Entity                     string                  `json:"entity"`
	Model                      string                  `json:"model"`
	Variant                    string                  `json:"variant"`
	InMemoryCachingEnabled     bool                    `json:"in_memory_caching_enabled"`
	InMemoryCacheTTLSeconds    int                     `json:"in_memory_cache_ttl_seconds"`
	DistributedCachingEnabled  bool                    `json:"distributed_caching_enabled"`
	DistributedCacheTTLSeconds int                     `json:"distributed_cache_ttl_seconds"`
	Filter                     []skyeEtcd.Criteria     `json:"filter"`
	VectorDBConfig             skyeEtcd.VectorDbConfig `json:"vector_db_config"`
	RateLimiter                skyeEtcd.RateLimiter    `json:"rate_limiter"`
	RTPartition                int                     `json:"rt_partition"`
}

type VariantEditRequest struct {
	Requestor string                    `json:"requestor"`
	Reason    string                    `json:"reason"`
	Payload   VariantEditRequestPayload `json:"payload"`
}

type VariantListResponse struct {
	Variants map[string]skyeEtcd.Variant `json:"variants"`
}

type VariantRequestListResponse struct {
	VariantRequests []variant_requests.VariantRequest `json:"variant_requests"`
	TotalCount      int                               `json:"total_count"`
}

type FilterRequestPayload struct {
	Entity string         `json:"entity"`
	Filter FilterCriteria `json:"filter"`
}

type FilterRegisterRequest struct {
	Requestor string               `json:"requestor"`
	Reason    string               `json:"reason"`
	Payload   FilterRequestPayload `json:"payload"`
}

type FilterCriteriaResponse struct {
	ColumnName   string `json:"column_name"`
	FilterValue  string `json:"filter_value"`
	DefaultValue string `json:"default_value"`
}

type FilterListResponse struct {
	Filters map[string]skyeEtcd.Criteria `json:"filters"`
}

type JobFrequencyRequestPayload struct {
	JobFrequency string `json:"job_frequency"`
}

type JobFrequencyRegisterRequest struct {
	Requestor string                     `json:"requestor"`
	Reason    string                     `json:"reason"`
	Payload   JobFrequencyRequestPayload `json:"payload"`
}

type JobFrequencyListResponse struct {
	Frequencies map[string]string `json:"frequencies"`
	TotalCount  int               `json:"total_count"`
}

type JobFrequencyRequestListResponse struct {
	JobFrequencyRequests []job_frequency_requests.JobFrequencyRequest `json:"job_frequency_requests"`
	TotalCount           int                                          `json:"total_count"`
}

type FilterRequestListResponse struct {
	FilterRequests []filter_requests.FilterRequest `json:"filter_requests"`
	TotalCount     int                             `json:"total_count"`
}

type ClusterMetadata struct {
	TotalInstances int      `json:"total_instances"`
	TotalStorage   string   `json:"total_storage"`
	Environments   []string `json:"environments"`
	Projects       []string `json:"projects"`
}

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

type VariantOnboardingRequestListResponse struct {
	VariantOnboardingRequests []variant_onboarding_requests.VariantOnboardingRequest `json:"variant_onboarding_requests"`
	TotalCount                int                                                    `json:"total_count"`
}

type VariantOnboardingTaskListResponse struct {
	VariantOnboardingTasks []variant_onboarding_tasks.VariantOnboardingTask `json:"variant_onboarding_tasks"`
	TotalCount             int                                              `json:"total_count"`
}

type MQIdTopicMapping struct {
	MQID  int    `json:"mq_id"`
	Topic string `json:"topic"`
}

type MQIdTopicsResponse struct {
	Mappings   []MQIdTopicMapping `json:"mappings"`
	TotalCount int                `json:"total_count"`
}

type VariantsListResponse struct {
	Variants   []string `json:"variants"`
	TotalCount int      `json:"total_count"`
}
