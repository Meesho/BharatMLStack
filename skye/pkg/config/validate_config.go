package config

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	HeaderEnvironment    = "environment"
	HeaderDeployableName = "deployable-name"
)

// ConfigValidator validates configuration based on headers and request body.
// To integrate it with your HTTP framework, use the following example:
//
// `httpframework.Instance().POST("/v1/validate-config", configs.ConfigValidator())`
// Mandatory Headers:
//   - `environment`: Specifies the environment (e.g., `prd`, `stg`, `int`).
//   - `deployable-name`: The name of the deployable application.
//
// Request Body (JSON):
//   - Contains key-value pairs representing the configuration to be validated
//
// Response:
//   - HTTP 200: If all keys are valid.
//   - HTTP 400: If any mandatory header is missing, JSON is invalid, or there are mismatched/missing keys.
//
// Example Usage:
//
// Sample `curl` request:
//
//	curl -X POST http://localhost:8080/validate-config \
//	     -H "Content-Type: application/json" \
//	     -H "environment: prod" \
//	     -H "deployable-name: my-app" \
//	     -d '{
//	         "key1": "value1",
//	         "key2": "wrong_value",
//	         "key3": "value3"
//	     }'
//
// Expected Response (if there are missing/mismatched keys):
//
// ```json
//
//	{
//	  "missing_keys": ["key3"],
//	  "mismatched_keys": ["key2"]
//	}
//
// ```
//
// Expected Response (if all keys match):
//
// ```json
//
//	{
//	  "message": "Config is valid"
//	}
//
// ```
func ConfigValidator() gin.HandlerFunc {
	return func(c *gin.Context) {
		environment := c.GetHeader(HeaderEnvironment)
		deployableName := c.GetHeader(HeaderDeployableName)

		if environment == "" || deployableName == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"message": "Headers 'environment' and 'deployable-name' are required",
			})
			return
		}

		var requestBody map[string]interface{}
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"message": "Invalid JSON body",
			})
			return
		}

		configLocation := viper.GetString(envConfigLocation)
		if configLocation == "" {
			configLocation = "/opt/config"
		}

		staticConfigFilePath := fmt.Sprintf("%s/application-%s.yml", configLocation, environment)
		var staticConfig map[string]interface{}
		if err := loadStaticConfig(staticConfigFilePath, &staticConfig, deployableName); err != nil {
			log.Error().Err(err).Msg("Failed to load static config")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to load static configuration file",
			})
			return
		}

		var missingKeys []string
		var mismatchedKeys []string

		for key, value := range requestBody {
			if !viper.IsSet(key) {
				missingKeys = append(missingKeys, key)
				continue
			}
			configValue := viper.GetString(key)
			if configValue != fmt.Sprintf("%v", value) {
				mismatchedKeys = append(mismatchedKeys, key)
			}
		}

		if len(missingKeys) > 0 || len(mismatchedKeys) > 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"missing_keys":    missingKeys,
				"mismatched_keys": mismatchedKeys,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Config is valid",
		})
	}
}
