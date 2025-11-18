package etcd

type ModelConfigRegistery struct {
	InferflowConfig map[string]InferflowConfigs `json:"inferflow-config"`
}

type HorizonRegistry struct {
	Inferflow InferflowComponents `json:"inferflow"`
}

type InferflowComponents struct {
	InferflowComponents map[string]ComponentData `json:"inferflow-components"`
}

type InferflowConfigs struct {
	ModelConfig ModelConfigData `json:"model-config"`
}

type ModelConfigData struct {
	ConfigMap map[string]InferflowConfig `json:"config-map"`
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

type PredatorComponent struct {
	Component     string           `json:"component"`
	ComponentID   string           `json:"component_id"`
	ModelName     string           `json:"model_name"`
	Calibration   string           `json:"calibration,omitempty"`
	ModelEndPoint string           `json:"model_end_point"`
	Deadline      int              `json:"deadline"`
	BatchSize     int              `json:"batch_size"`
	Inputs        []PredatorInput  `json:"inputs"`
	Outputs       []PredatorOutput `json:"outputs"`
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

type FinalResponseConfig struct {
	LoggingPerc          int      `json:"logging_perc"`
	ModelSchemaPerc      int      `json:"model_schema_features_perc"`
	Features             []string `json:"features"`
	LogSelectiveFeatures bool     `json:"log_features"`
	LogBatchSize         int      `json:"log_batch_size"`
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

type ComponentConfig struct {
	CacheEnabled       bool                `json:"cache_enabled"`
	CacheTTL           int                 `json:"cache_ttl"`
	CacheVersion       int                 `json:"cache_version"`
	FeatureComponents  []FeatureComponent  `json:"feature_components"`
	RTPComponents      []RTPComponent      `json:"real_time_pricing_feature_components"`
	PredatorComponents []PredatorComponent `json:"predator_components"`
	NumerixComponents  []NumerixComponent  `json:"numerix_components"`
}

type DagExecutionConfig struct {
	ComponentDependency map[string][]string `json:"component_dependency"`
}

type InferflowConfig struct {
	DagExecutionConfig DagExecutionConfig  `json:"dag_execution_config"`
	ComponentConfig    ComponentConfig     `json:"component_config"`
	ResponseConfig     FinalResponseConfig `json:"response_config"`
}

type ComponentData struct {
	ComponentID              string                                 `json:"component-id"`
	CompositeID              bool                                   `json:"composite-id"`
	ExecutionDependency      string                                 `json:"execution-dependency"`
	FSFlattenResKeys         map[string]string                      `json:"fs-flatten-res-keys"`
	FSIdSchemaToValueColumns map[string]FSIdSchemaToValueColumnPair `json:"fs-id-schema-to-value-columns"`
	Overridecomponent        map[string]OverrideComponent           `json:"override-component"`
}

type OverrideComponent struct {
	ComponentId string `json:"component-id"`
}

type FSIdSchemaToValueColumnPair struct {
	Schema   string `json:"schema"`
	ValueCol string `json:"value-col"`
	DataType string `json:"data-type"`
}
