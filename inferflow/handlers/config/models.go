//go:build !meesho

package config

import (
	"encoding/json"

	"github.com/emirpasic/gods/maps/linkedhashmap"
)

type ModelConfig struct {
	ConfigMap map[string]Config `json:"model_config_map"`
	DrPercent int               `json:"dr_percent"`
}

type Config struct {
	DAGExecutionConfig DAGExecutionConfig `json:"dag_execution_config"`
	ComponentConfig    ComponentConfig    `json:"component_config"`
	ResponseConfig     ResponseConfig     `json:"response_config"`
	DrConfig           DrConfig           `json:"dr_config"`
}

type DAGExecutionConfig struct {
	ComponentDependency map[string][]string `json:"component_dependency"`
}

type ComponentConfig struct {
	CacheEnabled            bool              `json:"cache_enabled"`
	CacheVersion            int               `json:"cache_version"`
	CacheTtl                int               `json:"cache_ttl"`
	ErrorLoggingPercent     int               `json:"error_logging_percent"`
	FeatureComponentConfig  linkedhashmap.Map `json:"feature_components"`
	PredatorComponentConfig linkedhashmap.Map `json:"predator_components"`
	NumerixComponentConfig  linkedhashmap.Map `json:"numerix_components"`
}

type ResponseConfig struct {
	Features        []string `json:"features"`
	ModelSchemaPerc int      `json:"model_schema_features_perc"`
	LoggingPerc     int      `json:"logging_perc"`
	LogFeatures     bool     `json:"log_features"`
	LogBatchSize    int      `json:"log_batch_size"`
}

type DrConfig struct {
	Res               []string          `json:"res"`
	Version           int               `json:"version"`
	Keys              Keys              `json:"keys"`
	Override          map[string]string `json:"override"`
	OverrideConfigKey string            `json:"override_config_key"`
	DrModelConfigId   string            `json:"dr_model_config_id"`
}

type Keys struct {
	HeaderKeys []string `json:"header_keys"`
	Id         string   `json:"id"`
}

type FeatureComponentConfig struct {
	Component         string         `json:"component"`
	ComponentId       string         `json:"component_id"`
	CompositeId       bool           `json:"composite_id"`
	FSKeys            []FSKey        `json:"fs_keys"`
	FSRequest         FeatureRequest `json:"fs_request"`
	FSFlattenRespKeys []string       `json:"fs_flatten_resp_keys"`
	CompCacheEnabled  bool           `json:"comp_cache_enabled"`
	CompCacheTtl      int            `json:"comp_cache_ttl"`
	ColNamePrefix     string         `json:"col_name_prefix"`
}

type FSKey struct {
	Schema string `json:"schema"`
	Column string `json:"col"`
}

type FeatureRequest struct {
	Label         string         `json:"label"`
	FeatureGroups []FeatureGroup `json:"featureGroups"`
}

type FeatureGroup struct {
	Label    string   `json:"label"`
	DataType string   `json:"data_type"`
	Features []string `json:"features"`
}

type PredatorComponentConfig struct {
	Component      string          `json:"component"`
	ComponentId    string          `json:"component_id"`
	ModelName      string          `json:"model_name"`
	ModelEndpoint  string          `json:"model_end_point"`
	ModelEndPoints []ModelEndpoint `json:"model_end_points"`
	Deadline       int             `json:"deadline"`
	BatchSize      int             `json:"batch_size"`
	Calibration    string          `json:"calibration"`
	Inputs         []ModelInput    `json:"inputs"`
	Outputs        []ModelOutput   `json:"outputs"`
}

type ModelEndpoint struct {
	EndPoint          string `json:"endpoint"`
	RoutingPercentage int    `json:"percentage"`
}

type ModelInput struct {
	Name     string   `json:"name"`
	Features []string `json:"features"`
	Shape    []int    `json:"shape"`
	DataType string   `json:"data_type"`
}

type ModelOutput struct {
	Name            string   `json:"name"`
	ModelScores     []string `json:"model_scores"`
	ModelScoresDims [][]int  `json:"model_scores_dims"`
	DataType        string   `json:"data_type"`
}

type NumerixComponentConfig struct {
	Component    string            `json:"component"`
	ComponentId  string            `json:"component_id"`
	ScoreColumn  string            `json:"score_col"`
	DataType     string            `json:"data_type"`
	ScoreMapping map[string]string `json:"score_mapping"`
	ComputeId    string            `json:"compute_id"`
}

func (c *ComponentConfig) UnmarshalJSON(data []byte) error {
	type Alias ComponentConfig
	aux := &struct {
		CacheEnabled            bool                      `json:"cache_enabled"`
		CacheVersion            int                       `json:"cache_version"`
		CacheTtl                int                       `json:"cache_ttl"`
		ErrorLoggingPercent     int                       `json:"error_logging_percent"`
		FeatureComponentConfig  []FeatureComponentConfig  `json:"feature_components"`
		PredatorComponentConfig []PredatorComponentConfig `json:"predator_components"`
		NumerixComponentConfig  []NumerixComponentConfig  `json:"numerix_components"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Initialize the featurestore component linked hashmap.Map fields
	featureComponentMap := linkedhashmap.New()
	for _, featureConfig := range aux.FeatureComponentConfig {
		featureComponentMap.Put(featureConfig.Component, featureConfig)
	}
	c.FeatureComponentConfig = *featureComponentMap
	c.CacheEnabled = aux.CacheEnabled
	c.CacheVersion = aux.CacheVersion
	c.CacheTtl = aux.CacheTtl
	c.ErrorLoggingPercent = aux.ErrorLoggingPercent

	// Initialize the ranker component linked hashmap.Map fields
	predatorComponentMap := linkedhashmap.New()
	for _, predatorConfig := range aux.PredatorComponentConfig {
		predatorComponentMap.Put(predatorConfig.Component, predatorConfig)
	}
	c.PredatorComponentConfig = *predatorComponentMap

	// Initialize the numerix component linked hashmap.Map fields
	numerixComponentMap := linkedhashmap.New()
	for _, numerixConfig := range aux.NumerixComponentConfig {
		numerixComponentMap.Put(numerixConfig.Component, numerixConfig)
	}
	c.NumerixComponentConfig = *numerixComponentMap
	return nil
}
