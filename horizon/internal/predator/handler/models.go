package handler

type GCSFoldersResponse struct {
	Folders []GCSFolder `json:"folders"`
}

type GCSFolder struct {
	Name     string           `json:"name"`
	Path     string           `json:"path"`
	Metadata *FeatureMetadata `json:"metadata,omitempty"`
}

type ModelUploadItem struct {
	GCSPath  string      `json:"gcs_path" binding:"required"`
	Metadata interface{} `json:"metadata" binding:"required"`
}

type UploadModelFolderRequest struct {
	RequestType string            `json:"request_type" binding:"required"` // "create" or "update"
	Models      []ModelUploadItem `json:"models" binding:"required"`
}

type ModelUploadResult struct {
	ModelName    string `json:"model_name"`
	GCSPath      string `json:"gcs_path"`
	MetadataPath string `json:"metadata_path"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
}

type UploadModelFolderResponse struct {
	Message string              `json:"message"`
	Results []ModelUploadResult `json:"results"`
}

type FeatureMetadata struct {
	Inputs []InputMeta `json:"inputs"`
}

type InputMeta struct {
	Name     string   `json:"name"`
	Features []string `json:"features"`
}

// OnlineValidationRequest represents the request to validate online features
type OnlineValidationRequest struct {
	Entity string `json:"entity"`
}

// OnlineValidationResponse represents the response from online feature validation
type OnlineValidationResponse []struct {
	EntityLabel       string `json:"entity-label"`
	FeatureGroupLabel string `json:"feature-group-label"`
	ID                int    `json:"id"`
	ActiveVersion     string `json:"active-version"`
	Features          map[string]struct {
		Labels []string `json:"labels"`
	} `json:"features"`
	StoreID                 string `json:"store-id"`
	DataType                string `json:"data-type"`
	TTLInSeconds            int    `json:"ttl-in-seconds"`
	TTLInEpoch              int    `json:"ttl-in-epoch"`
	JobID                   string `json:"job-id"`
	InMemoryCacheEnabled    bool   `json:"in-memory-cache-enabled"`
	DistributedCacheEnabled bool   `json:"distributed-cache-enabled"`
	LayoutVersion           int    `json:"layout-version"`
}

// OfflineValidationRequest represents the request to validate offline features
type OfflineValidationRequest struct {
	OfflineFeatureList []string `json:"offline-feature-list"`
}

// OfflineValidationResponse represents the response from offline feature validation
type OfflineValidationResponse struct {
	Error string            `json:"error"`
	Data  map[string]string `json:"data"`
}
