package handler

import (
	"time"

	dbModel "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow"
)

type Input struct {
	Name     string   `json:"name"`
	Features []string `json:"features"`
	Dims     []int    `json:"dims"`
	DataType string   `json:"data_type"`
}

type Output struct {
	Name            string   `json:"name"`
	ModelScores     []string `json:"model_scores"`
	ModelScoresDims [][]int  `json:"model_scores_dims"`
	DataType        string   `json:"data_type"`
}

type RoutingConfig struct {
	ModelName         string  `json:"model_name"`
	ModelEndpoint     string  `json:"model_endpoint"`
	RoutingPercentage float32 `json:"routing_percentage"`
}

type Ranker struct {
	ModelName     string          `json:"model_name"`
	BatchSize     int             `json:"batch_size"`
	Deadline      int             `json:"deadline"`
	Calibration   string          `json:"calibration"`
	EndPoint      string          `json:"end_point"`
	Inputs        []Input         `json:"inputs"`
	Outputs       []Output        `json:"outputs"`
	EntityID      []string        `json:"entity_id"`
	RoutingConfig []RoutingConfig `json:"route_config"`
}

type ReRanker struct {
	EqVariables map[string]string `json:"eq_variables"`
	Score       string            `json:"score"`
	EqID        int               `json:"eq_id"`
	DataType    string            `json:"data_type"`
	EntityID    []string          `json:"entity_id"`
}

type ResponseConfig struct {
	PrismLoggingPerc                   int      `json:"prism_logging_perc"`
	RankerSchemaFeaturesInResponsePerc int      `json:"ranker_schema_features_in_response_perc"`
	ResponseFeatures                   []string `json:"response_features"`
	LogSelectiveFeatures               bool     `json:"log_features"`
	LogBatchSize                       int      `json:"log_batch_size"`
	LoggingTTL                         int      `json:"logging_ttl"`
}

type ConfigMapping struct {
	AppToken              string   `json:"app_token"`
	ConnectionConfigID    int      `json:"connection_config_id"`
	DeployableID          int      `json:"deployable_id,omitempty"`
	DeployableName        string   `json:"deployable_name,omitempty"`
	ResponseDefaultValues []string `json:"response_default_values"`
	SourceConfigID        string   `json:"source_config_id"`
}

type OnboardPayload struct {
	RealEstate       string `json:"real_estate"`
	Tenant           string `json:"tenant"`
	ConfigIdentifier string `json:"config_identifier"`
	// EntityID         []string       `json:"entity_id"`
	Rankers       []Ranker       `json:"rankers"`
	ReRankers     []ReRanker     `json:"re_rankers"`
	Response      ResponseConfig `json:"response"`
	ConfigMapping ConfigMapping  `json:"config_mapping"`
}

type InferflowOnboardRequest struct {
	Payload   OnboardPayload `json:"payload"`
	CreatedBy string         `json:"created_by"`
}

func (r InferflowOnboardRequest) GetConfigMapping() dbModel.ConfigMapping {
	return dbModel.ConfigMapping{
		AppToken:              r.Payload.ConfigMapping.AppToken,
		ConnectionConfigID:    r.Payload.ConfigMapping.ConnectionConfigID,
		DeployableID:          r.Payload.ConfigMapping.DeployableID,
		ResponseDefaultValues: r.Payload.ConfigMapping.ResponseDefaultValues,
		SourceConfigID:        r.Payload.ConfigMapping.SourceConfigID,
	}
}

type Message struct {
	Message string `json:"message"`
}

type Response struct {
	Error string  `json:"error"`
	Data  Message `json:"data"`
}

type NumerixComponent struct {
	Component    string            `json:"component"`
	ComponentID  string            `json:"component_id"`
	ScoreCol     string            `json:"score_col"`
	ComputeID    string            `json:"compute_id"`
	ScoreMapping map[string]string `json:"score_mapping"`
	DataType     string            `json:"data_type"`
}

type PredatorInput struct {
	Name     string   `json:"name"`
	Features []string `json:"features"`
	Dims     []int    `json:"shape"` // aligns with "shape" field in new config
	DataType string   `json:"data_type"`
}

type PredatorOutput struct {
	Name            string   `json:"name"`
	ModelScores     []string `json:"model_scores"`
	ModelScoresDims [][]int  `json:"model_scores_dims"`
	DataType        string   `json:"data_type"`
}

type PredatorComponent struct {
	Component     string           `json:"component"`
	ComponentID   string           `json:"component_id"`
	ModelName     string           `json:"model_name"`
	ModelEndPoint string           `json:"model_end_point"`
	Calibration   string           `json:"calibration,omitempty"`
	Deadline      int              `json:"deadline"`
	BatchSize     int              `json:"batch_size"`
	Inputs        []PredatorInput  `json:"inputs"`
	Outputs       []PredatorOutput `json:"outputs"`
	RoutingConfig []RoutingConfig  `json:"route_config,omitempty"`
}

type FinalResponseConfig struct {
	LoggingPerc          int      `json:"logging_perc"`
	ModelSchemaPerc      int      `json:"model_schema_features_perc"` // updated tag as per config
	Features             []string `json:"features"`
	LogSelectiveFeatures bool     `json:"log_features"`
	LogBatchSize         int      `json:"log_batch_size"`
	LoggingTTL           int      `json:"logging_ttl"`
}

type FSKey struct {
	Schema string `json:"schema"`
	Col    string `json:"col"`
}

type FSFeatureGroup struct {
	Label    string   `json:"label"`
	Features []string `json:"features"`
	DataType string   `json:"data_type"`
}

type FSRequest struct {
	Label         string           `json:"label"`
	FeatureGroups []FSFeatureGroup `json:"featureGroups"`
}

type FeatureComponent struct {
	Component         string     `json:"component"`
	ComponentID       string     `json:"component_id"`
	ColNamePrefix     string     `json:"col_name_prefix,omitempty"`
	CompCacheEnabled  bool       `json:"comp_cache_enabled"`
	CompCacheTTL      int        `json:"comp_cache_ttl,omitempty"`
	CompositeID       bool       `json:"composite_id,omitempty"`
	FSKeys            []FSKey    `json:"fs_keys"`
	FSRequest         *FSRequest `json:"fs_request"`
	FSFlattenRespKeys []string   `json:"fs_flatten_resp_keys"`
}

type RTPComponent struct {
	Component         string     `json:"component"`
	ComponentID       string     `json:"component_id"`
	CompositeID       bool       `json:"composite_id"`
	FSKeys            []FSKey    `json:"fs_keys"`
	FeatureRequest    *FSRequest `json:"fs_request"`
	FSFlattenRespKey  string     `json:"fs_flatten_resp_key"`
	FSFlattenRespKeys []string   `json:"fs_flatten_resp_keys"`
	ColNamePrefix     string     `json:"col_name_prefix"`
	CompCacheEnabled  bool       `json:"comp_cache_enabled"`
}

type SeenScoreComponent struct {
	Component     string     `json:"component"`
	ComponentID   string     `json:"component_id,omitempty"`
	ColNamePrefix string     `json:"col_name_prefix,omitempty"`
	FSKeys        []FSKey    `json:"fs_keys"`
	FSRequest     *FSRequest `json:"fs_request"`
}

type ComponentConfig struct {
	CacheEnabled        bool                 `json:"cache_enabled"`
	CacheTTL            int                  `json:"cache_ttl"`
	CacheVersion        int                  `json:"cache_version"`
	FeatureComponents   []FeatureComponent   `json:"feature_components"`
	RTPComponents       []RTPComponent       `json:"real_time_pricing_feature_components,omitempty"`
	SeenScoreComponents []SeenScoreComponent `json:"seen_score_components"`
	PredatorComponents  []PredatorComponent  `json:"predator_components"`
	NumerixComponents   []NumerixComponent   `json:"numerix_components"`
}

type DagExecutionConfig struct {
	ComponentDependency map[string][]string `json:"component_dependency"`
}

type InferflowConfig struct {
	DagExecutionConfig *DagExecutionConfig  `json:"dag_execution_config"`
	ComponentConfig    *ComponentConfig     `json:"component_config"`
	ResponseConfig     *FinalResponseConfig `json:"response_config"`
}

type ReviewRequest struct {
	RequestID    uint   `json:"request_id"`
	Status       string `json:"status"`
	Reviewer     string `json:"reviewer"`
	RejectReason string `json:"reject_reason"`
}

type ProposedModelEndpoint struct {
	ModelName  string `json:"model_name"`
	EndPointID string `json:"end_point_id"`
}

type PromoteConfigPayload struct {
	ConfigID               string                  `json:"config_id"`
	ConfigValue            InferflowConfig         `json:"config_value"`
	LatestRequest          RequestConfig           `json:"latest_request"`
	ProposedModelEndPoints []ProposedModelEndpoint `json:"proposed_model_end_point"`
	ConfigMapping          ConfigMapping           `json:"config_mapping"`
}

type PromoteConfigRequest struct {
	Payload   PromoteConfigPayload `json:"payload"`
	CreatedBy string               `json:"created_by"`
}

func (r PromoteConfigRequest) GetConfigMapping() dbModel.ConfigMapping {
	return dbModel.ConfigMapping{
		AppToken:              r.Payload.ConfigMapping.AppToken,
		ConnectionConfigID:    r.Payload.ConfigMapping.ConnectionConfigID,
		DeployableID:          r.Payload.ConfigMapping.DeployableID,
		ResponseDefaultValues: r.Payload.ConfigMapping.ResponseDefaultValues,
		SourceConfigID:        r.Payload.ConfigMapping.SourceConfigID,
	}
}

type EditConfigOrCloneConfigRequest struct {
	Payload   OnboardPayload `json:"payload"`
	CreatedBy string         `json:"created_by"`
}

func (r EditConfigOrCloneConfigRequest) GetConfigMapping() dbModel.ConfigMapping {
	return dbModel.ConfigMapping{
		AppToken:              r.Payload.ConfigMapping.AppToken,
		ConnectionConfigID:    r.Payload.ConfigMapping.ConnectionConfigID,
		DeployableID:          r.Payload.ConfigMapping.DeployableID,
		ResponseDefaultValues: r.Payload.ConfigMapping.ResponseDefaultValues,
		SourceConfigID:        r.Payload.ConfigMapping.SourceConfigID,
	}
}

type ModelNameToEndPointMap struct {
	CurrentModelName string `json:"current_model_name"`
	NewModelName     string `json:"new_model_name"`
	EndPointID       string `json:"end_point_id"`
}

type ScaleUpConfigPayload struct {
	ConfigID               string                   `json:"config_id"`
	ConfigValue            InferflowConfig          `json:"config_value"`
	ConfigMapping          ConfigMapping            `json:"config_mapping"`
	LoggingPerc            int                      `json:"logging_perc"`
	LoggingTTL             int                      `json:"logging_ttl"`
	ModelNameToEndPointMap []ModelNameToEndPointMap `json:"proposed_model_endpoints"`
}

type ScaleUpConfigRequest struct {
	Payload   ScaleUpConfigPayload `json:"payload"`
	CreatedBy string               `json:"created_by"`
}

func (r ScaleUpConfigRequest) GetConfigMapping() dbModel.ConfigMapping {
	return dbModel.ConfigMapping{
		AppToken:              r.Payload.ConfigMapping.AppToken,
		ConnectionConfigID:    r.Payload.ConfigMapping.ConnectionConfigID,
		DeployableID:          r.Payload.ConfigMapping.DeployableID,
		ResponseDefaultValues: r.Payload.ConfigMapping.ResponseDefaultValues,
	}
}

type DeleteConfigRequest struct {
	ConfigID  string `json:"config_id"`
	CreatedBy string `json:"created_by"`
}

type GetAllRequestConfigsRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type Payload struct {
	ConfigValue    InferflowConfig `json:"config_value"`
	ConfigMapping  ConfigMapping   `json:"config_mapping"`
	RequestPayload OnboardPayload  `json:"request_payload"`
}

type RequestConfig struct {
	RequestID    uint      `json:"request_id"`
	Payload      Payload   `json:"payload"`
	ConfigID     string    `json:"config_id"`
	CreatedBy    string    `json:"created_by"`
	RequestType  string    `json:"request_type"`
	UpdatedBy    string    `json:"updated_by"`
	UpdatedAt    time.Time `json:"updated_at"`
	ApprovedBy   string    `json:"approved_by"`
	CreatedAt    time.Time `json:"created_at"`
	RejectReason string    `json:"reject_reason"`
	Status       string    `json:"status"`
	Reviewer     string    `json:"reviewer"`
}

type GetAllRequestConfigsResponse struct {
	Error interface{}     `json:"error"`
	Data  []RequestConfig `json:"data"`
}

type GetLatestRequestResponse struct {
	Error interface{}   `json:"error"`
	Data  RequestConfig `json:"data"`
}

type GetLoggingTTLResponse struct {
	Data []int `json:"data"`
}

type DefaultResponse struct {
	ComponentData []ComponentData `json:"component_data"`
}

type ComponentData struct {
	Data []string `json:"data"`
}

type ConnectionConfig struct {
	ID                   int    `json:"id"`
	Default              bool   `json:"default"`
	ConnProtocol         string `json:"conn_protocol"`
	Timeout              int    `json:"timeout"`
	Deadline             int    `json:"deadline"`
	PlainText            bool   `json:"plain_text"`
	GrpcChannelAlgorithm string `json:"grpc_channel_algorithm"`
}

type ConfigTable struct {
	ConfigID                string           `json:"config_id"`
	ConfigValue             InferflowConfig  `json:"config_value"`
	DefaultResponse         DefaultResponse  `json:"default_response"`
	Host                    string           `json:"host"`
	AppToken                string           `json:"app_token"`
	ConnectionConfig        ConnectionConfig `json:"connection_config"`
	DeployableRunningStatus bool             `json:"deployable_running_status"`
	MonitoringUrl           string           `json:"monitoring_url"`
	CreatedBy               string           `json:"created_by"`
	UpdatedBy               string           `json:"updated_by"`
	CreatedAt               time.Time        `json:"created_at"`
	UpdatedAt               time.Time        `json:"updated_at"`
	TestResults             TestResults      `json:"test_results"`
	SourceConfigID          string           `json:"source_config_id"`
}

type TestResults struct {
	Tested  bool   `json:"tested"`
	Message string `json:"message"`
}

type GetAllResponse struct {
	Error interface{}   `json:"error"`
	Data  []ConfigTable `json:"data"`
}

type CancelConfigRequest struct {
	RequestID uint   `json:"request_id"`
	UpdatedBy string `json:"updated_by"`
}

type GetOnlineFeatureMappingResponse struct {
	Error string            `json:"error"`
	Data  map[string]string `json:"data"`
}

type GetOnlineFeatureMappingRequest struct {
	OfflineFeatureList []string `json:"offline-feature-list"`
}

type ValidateRequest struct {
	ConfigID string `json:"config_id"`
}

type GenerateRequestFunctionalTestingRequest struct {
	BatchSize       string            `json:"batch_size"`
	MetaData        map[string]string `json:"meta_data"`
	DefaultFeatures map[string]string `json:"default_features"`
	ModelConfigID   string            `json:"model_config_id"`
	Entity          string            `json:"entity"`
}

type GenerateRequestFunctionalTestingResponse struct {
	RequestBody RequestBody       `json:"request_body"`
	EndPoint    string            `json:"end_point"`
	MetaData    map[string]string `json:"meta_data"`
	Error       string            `json:"error"`
}

type RequestBody struct {
	Entities      []Entity `json:"entities"`
	ModelConfigID string   `json:"model_config_id"`
}

type Entity struct {
	Entity   string         `json:"entity"`
	Ids      []string       `json:"ids"`
	Features []FeatureValue `json:"features"`
}

type FeatureValue struct {
	Name            string   `json:"name"`
	IdsFeatureValue []string `json:"ids_feature_value"`
}

type ExecuteRequestFunctionalTestingRequest struct {
	RequestBody RequestBody       `json:"request_body"`
	EndPoint    string            `json:"end_point"`
	MetaData    map[string]string `json:"meta_data"`
}

type ExecuteRequestFunctionalTestingResponse struct {
	ComponentData   []ComponentData        `json:"component_data"`
	Error           string                 `json:"error"`
	ApplicationLogs map[string]interface{} `json:"application_logs"`
}

type GetFeatureGroupDataTypeMappingResponse struct {
	Label    string `json:"feature-group-label"`
	DataType string `json:"data-type"`
}

type GetRTPFeatureGroupDataTypeMappingResponse struct {
	Entities []RTPEntity `json:"entities"`
}

type RTPEntity struct {
	Entity        string            `json:"entity"`
	FeatureGroups []RTPFeatureGroup `json:"featureGroups"`
}

type RTPFeatureGroup struct {
	Label    string   `json:"label"`
	Features []string `json:"features"`
	DataType string   `json:"dataType"`
}

type FeatureSchemaRequest struct {
	ModelConfigId string `json:"model_config_id"`
	Version       string `json:"version"`
}

type FeatureSchemaResponse struct {
	Data []dbModel.SchemaComponents `json:"data"`
}
