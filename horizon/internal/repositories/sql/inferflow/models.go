package inferflow

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type DagExecutionConfig struct {
	ComponentDependency map[string][]string `json:"component_dependency"`
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
	Dims     []int    `json:"shape"`
	DataType string   `json:"data_type"`
}

type PredatorOutput struct {
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

type ResponseConfig struct {
	LoggingPerc          int      `json:"logging_perc"`
	ModelSchemaPerc      int      `json:"model_schema_features_perc"`
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
	FSRequest         *FSRequest `json:"fs_request"`
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
	PredatorComponents  []PredatorComponent  `json:"predator_components"`
	NumerixComponents   []NumerixComponent   `json:"numerix_components"`
	SeenScoreComponents []SeenScoreComponent `json:"seen_score_components"`
}

type InferflowConfig struct {
	DagExecutionConfig DagExecutionConfig `json:"dag_execution_config"`
	ComponentConfig    ComponentConfig    `json:"component_config"`
	ResponseConfig     ResponseConfig     `json:"response_config"`
}

type ConfigMapping struct {
	AppToken              string   `json:"app_token"`
	ConnectionConfigID    int      `json:"connection_config_id"`
	DeployableID          int      `json:"deployable_id"`
	ResponseDefaultValues []string `json:"response_default_values"`
	SourceConfigID        string   `json:"source_config_id"`
}

type OnboardPayload struct {
	RealEstate       string            `json:"real_estate"`
	Tenant           string            `json:"tenant"`
	ConfigIdentifier string            `json:"config_identifier"`
	Rankers          []OnboardRanker   `json:"rankers"`
	ReRankers        []OnboardReRanker `json:"re_rankers"`
	Response         ResponseConfig    `json:"response"`
	ConfigMapping    ConfigMapping     `json:"config_mapping"`
}

type OnboardRanker struct {
	ModelName     string           `json:"model_name"`
	BatchSize     int              `json:"batch_size"`
	Deadline      int              `json:"deadline"`
	Calibration   string           `json:"calibration"`
	EndPoint      string           `json:"end_point"`
	Inputs        []PredatorInput  `json:"inputs"`
	Outputs       []PredatorOutput `json:"outputs"`
	EntityID      []string         `json:"entity_id"`
	RoutingConfig []RoutingConfig  `json:"route_config,omitempty"`
}

type OnboardReRanker struct {
	EqVariables map[string]string `json:"eq_variables"`
	Score       string            `json:"score"`
	EqID        int               `json:"eq_id"`
	DataType    string            `json:"data_type"`
	EntityID    []string          `json:"entity_id"`
}

type Payload struct {
	ConfigValue    InferflowConfig `json:"config_value"`
	ConfigMapping  ConfigMapping   `json:"config_mapping"`
	RequestPayload OnboardPayload  `json:"request_payload"`
}

type TestResults struct {
	Tested  bool   `json:"tested"`
	Message string `json:"message"`
}

func (p *Payload) Scan(value interface{}) error {
	if value == nil {
		*p = Payload{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("Scan source is not []byte")
	}
	return json.Unmarshal(bytes, p)
}

func (p Payload) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (m *InferflowConfig) Scan(value interface{}) error {
	if value == nil {
		*m = InferflowConfig{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("Scan source is not []byte")
	}
	return json.Unmarshal(bytes, m)
}

func (m InferflowConfig) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (d *DefaultResponse) Scan(value interface{}) error {
	if value == nil {
		*d = DefaultResponse{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("Scan source is not []byte")
	}
	return json.Unmarshal(bytes, d)
}

func (d DefaultResponse) Value() (driver.Value, error) {
	return json.Marshal(d)
}

type DefaultResponse struct {
	ComponentData []ComponentData `json:"component_data"`
}

type ComponentData struct {
	Data []string `json:"data"`
}

func (t *TestResults) Scan(value interface{}) error {
	if value == nil {
		*t = TestResults{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("Scan source is not []byte")
	}
	return json.Unmarshal(bytes, t)
}

func (t TestResults) Value() (driver.Value, error) {
	return json.Marshal(t)
}

type GetSchemaResponse struct {
	Components []SchemaComponents
}

type SchemaComponents struct {
	FeatureName string `json:"feature_name"`
	FeatureType string `json:"feature_type"`
	FeatureSize any    `json:"feature_size"`
}
