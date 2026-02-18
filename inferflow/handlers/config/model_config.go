package config

import "fmt"

var mConfig *ModelConfig

func GetModelConfigMap() *ModelConfig {
	return mConfig
}

// GetModelConfig returns the Config for a specific model config ID.
func GetModelConfig(modelConfigId string) (*Config, error) {
	if mConfig == nil {
		return nil, fmt.Errorf("model config map not initialised")
	}
	conf, ok := mConfig.ConfigMap[modelConfigId]
	if !ok || !validateModelConfig(&conf) {
		return nil, fmt.Errorf("model config not found or invalid for id: %s", modelConfigId)
	}
	return &conf, nil
}

func SetModelConfigMap(config *ModelConfig) {
	mConfig = config
}

func validateModelConfig(c *Config) bool {
	if c == nil ||
		len(c.DAGExecutionConfig.ComponentDependency) == 0 ||
		c.ComponentConfig.FeatureComponentConfig.Size() == 0 &&
			c.ComponentConfig.PredatorComponentConfig.Size() == 0 {
		return false
	}
	return true
}
