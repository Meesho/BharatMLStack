package config

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/inferflow/internal/errors"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
)

var mConfig *ModelConfig

func GetModelConfigMap() *ModelConfig {
	return mConfig
}

func SetModelConfigMap(config *ModelConfig) {
	mConfig = config
}

func GetModelConfig(modelId string) (*Config, error) {

	configMap := GetModelConfigMap()
	if len(configMap.ConfigMap) == 0 {
		logger.Error("Error while fetching Inferflow config ", nil)
		return nil, &errors.RequestError{ErrorMsg: "Error while fetching Inferflow config"}
	}
	config := configMap.ConfigMap[modelId]
	isValid := validateModelConfig(&config)
	if !isValid {
		logger.Error(fmt.Sprintf("Invalid model config for modelId %s ", modelId), nil)
		return nil, &errors.RequestError{ErrorMsg: fmt.Sprintf("Invalid model config for modelId %s ", modelId)}
	}
	return &config, nil
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
