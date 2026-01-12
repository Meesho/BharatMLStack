package handler

type Config interface {
	// ==================== STORE OPERATIONS ====================
	RegisterStore(StoreRegisterRequest) (RequestStatus, error)
	ApproveStoreRequest(int, ApprovalRequest) (ApprovalResponse, error)
	GetStores() (StoreListResponse, error)
	GetAllStoreRequests() (StoreRequestListResponse, error)

	// ==================== ENTITY OPERATIONS ====================
	RegisterEntity(EntityRegisterRequest) (RequestStatus, error)
	ApproveEntityRequest(int, ApprovalRequest) (ApprovalResponse, error)
	GetEntities() (EntityListResponse, error)
	GetAllEntityRequests() (EntityRequestListResponse, error)

	// ==================== MODEL OPERATIONS ====================
	RegisterModel(ModelRegisterRequest) (RequestStatus, error)
	EditModel(ModelEditRequest) (RequestStatus, error)
	ApproveModelRequest(int, ApprovalRequest) (ApprovalResponse, error)
	ApproveModelEditRequest(int, ApprovalRequest) (ApprovalResponse, error)
	GetModels() (ModelListResponse, error)
	GetAllModelRequests() (ModelRequestListResponse, error)

	// ==================== VARIANT OPERATIONS ====================
	RegisterVariant(VariantRegisterRequest) (RequestStatus, error)
	EditVariant(VariantEditRequest) (RequestStatus, error)
	ApproveVariantRequest(int, ApprovalRequest) (ApprovalResponse, error)
	ApproveVariantEditRequest(int, ApprovalRequest) (ApprovalResponse, error)
	GetVariants() (VariantListResponse, error)
	GetAllVariantRequests() (VariantRequestListResponse, error)

	// ==================== FILTER OPERATIONS ====================
	RegisterFilter(FilterRegisterRequest) (RequestStatus, error)
	ApproveFilterRequest(int, ApprovalRequest) (ApprovalResponse, error)
	GetFilters(FilterQueryOptions) (FilterListResponse, error)
	GetAllFilterRequests() (FilterRequestListResponse, error)

	// ==================== DEPLOYMENT OPERATIONS ====================
	// Qdrant Cluster Operations
	CreateQdrantCluster(QdrantClusterRequest) (RequestStatus, error)
	ApproveQdrantClusterRequest(int, ApprovalRequest) (ApprovalResponse, error)
	GetQdrantClusters() (ClusterListResponse, error)

	// Variant Promotion Operations
	PromoteVariant(VariantPromotionRequest) (RequestStatus, error)
	ApproveVariantPromotionRequest(int, ApprovalRequest) (ApprovalResponse, error)

	// Variant Onboarding Operations
	OnboardVariant(VariantOnboardingRequest) (RequestStatus, error)
	ApproveVariantOnboardingRequest(int, ApprovalRequest) (ApprovalResponse, error)
	GetAllVariantOnboardingRequests() (VariantOnboardingRequestListResponse, error)
	GetOnboardedVariants() (OnboardedVariantListResponse, error)

	// ==================== JOB FREQUENCY OPERATIONS ====================
	RegisterJobFrequency(JobFrequencyRegisterRequest) (RequestStatus, error)
	ApproveJobFrequencyRequest(int, ApprovalRequest) (ApprovalResponse, error)
	GetJobFrequencies() (JobFrequencyListResponse, error)
	GetAllJobFrequencyRequests() (JobFrequencyRequestListResponse, error)
}
