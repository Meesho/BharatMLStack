package configschemaclient

// SchemaComponents represents a feature schema component
type SchemaComponents struct {
	FeatureName string `json:"feature_name"`
	FeatureType string `json:"feature_type"`
	FeatureSize any    `json:"feature_size"`
}

// ComponentConfig contains all component configurations
type ComponentConfig struct {
	CacheEnabled       bool                `json:"cache_enabled"`
	CacheTTL           int                 `json:"cache_ttl"`
	CacheVersion       int                 `json:"cache_version"`
	FeatureComponents  []FeatureComponent  `json:"feature_components"`
	RTPComponents      []RTPComponent      `json:"real_time_pricing_feature_components,omitempty"`
	PredatorComponents []PredatorComponent `json:"predator_components"`
	NumerixComponents  []NumerixComponent  `json:"numerix_components"`
}

// ResponseConfig contains response configuration
type ResponseConfig struct {
	LoggingPerc          int      `json:"logging_perc"`
	ModelSchemaPerc      int      `json:"model_schema_features_perc"`
	Features             []string `json:"features"`
	LogSelectiveFeatures bool     `json:"log_features"`
	LogBatchSize         int      `json:"log_batch_size"`
}

// NumerixComponent represents a Numerix/Numerix component
type NumerixComponent struct {
	Component    string            `json:"component"`
	ComponentID  string            `json:"component_id"`
	ScoreCol     string            `json:"score_col"`
	ComputeID    string            `json:"compute_id"`
	ScoreMapping map[string]string `json:"score_mapping"`
	DataType     string            `json:"data_type"`
}

// FeatureComponent represents a feature store component
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

// RTPComponent represents a real-time pricing component
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

// PredatorComponent represents a Predator model component
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

// PredatorInput represents input configuration for Predator
type PredatorInput struct {
	Name     string   `json:"name"`
	Features []string `json:"features"`
	Dims     []int    `json:"shape"`
	DataType string   `json:"data_type"`
}

// PredatorOutput represents output configuration for Predator
type PredatorOutput struct {
	Name            string   `json:"name"`
	ModelScores     []string `json:"model_scores"`
	ModelScoresDims [][]int  `json:"model_scores_dims"`
	DataType        string   `json:"data_type"`
}

// RoutingConfig represents routing configuration
type RoutingConfig struct {
	ModelName         string  `json:"model_name"`
	ModelEndpoint     string  `json:"model_endpoint"`
	RoutingPercentage float32 `json:"routing_percentage"`
}

// FSKey represents a feature store key
type FSKey struct {
	Schema string `json:"schema"`
	Col    string `json:"col"`
}

// FSRequest represents a feature store request
type FSRequest struct {
	Label         string           `json:"label"`
	FeatureGroups []FSFeatureGroup `json:"featureGroups"`
}

// FSFeatureGroup represents a feature group
type FSFeatureGroup struct {
	Label    string   `json:"label"`
	Features []string `json:"features"`
	DataType string   `json:"data_type"`
}
