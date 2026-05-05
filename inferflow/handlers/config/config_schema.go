package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Meesho/BharatMLStack/inferflow/internal/errors"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
	"github.com/emirpasic/gods/maps/linkedhashmap"
)

var (
	featureSchemaCache = make(map[string]*linkedhashmap.Map)
	schemaCacheMu      sync.RWMutex
)

// InitAllFeatureSchema builds and caches feature schemas for all model configs.
// Should be called during config initialization and on config reload.
func InitAllFeatureSchema(modelConfig *ModelConfig) error {
	if modelConfig == nil {
		return &errors.ParsingError{ErrorMsg: "modelConfig is nil"}
	}
	if len(modelConfig.ConfigMap) == 0 {
		return &errors.ParsingError{ErrorMsg: "modelConfig.ConfigMap is empty"}
	}

	schemaCacheMu.Lock()
	defer schemaCacheMu.Unlock()

	newCache := make(map[string]*linkedhashmap.Map)
	for modelConfigID, config := range modelConfig.ConfigMap {
		logger.Info(fmt.Sprintf("building feature schema for modelConfigID: %s", modelConfigID))
		schema, err := buildFeatureSchema(&config)
		if err != nil {
			metrics.Count("inferflow_config_schema_parsing_error", 1, []string{"model-id", modelConfigID})
			return &errors.ParsingError{ErrorMsg: "failed to build feature schema for modelConfigID " + modelConfigID + ": " + err.Error()}
		}
		newCache[modelConfigID] = schema
	}
	featureSchemaCache = newCache
	return nil
}

// GetFeatureSchema returns the cached feature schema for a given model config ID.
func GetFeatureSchema(modelConfigID string) (*linkedhashmap.Map, error) {
	schemaCacheMu.RLock()
	defer schemaCacheMu.RUnlock()

	schema, exists := featureSchemaCache[modelConfigID]
	if !exists {
		return nil, &errors.RequestError{ErrorMsg: "feature schema not found for modelConfigID: " + modelConfigID}
	}
	return schema, nil
}

func buildFeatureSchema(config *Config) (*linkedhashmap.Map, error) {
	if config == nil {
		return nil, &errors.ParsingError{ErrorMsg: "config is nil"}
	}

	existingFeatures := make(map[string]bool)
	var response []SchemaComponents

	addUniqueComponents := func(components []SchemaComponents) {
		for _, component := range components {
			if !existingFeatures[component.FeatureName] {
				response = append(response, component)
				existingFeatures[component.FeatureName] = true
			}
		}
	}

	addOrUpdateComponents := func(components []SchemaComponents) {
		for _, component := range components {
			if !existingFeatures[component.FeatureName] {
				component.FeatureType = "String"
				response = append(response, component)
				existingFeatures[component.FeatureName] = true
			}
		}
	}

	fsComponents := processLinkedHashMap[FeatureComponentConfig](config.ComponentConfig.FeatureComponentConfig)
	predatorComponents := processLinkedHashMap[PredatorComponentConfig](config.ComponentConfig.PredatorComponentConfig)
	numerixComponents := processLinkedHashMap[NumerixComponentConfig](config.ComponentConfig.NumerixComponentConfig)

	// 1. Feature Store components
	addUniqueComponents(processFS(fsComponents))

	// 2. Numerix Output
	addUniqueComponents(processNumerixOutput(numerixComponents))

	// 3. Predator Output
	addUniqueComponents(processPredatorOutput(predatorComponents))

	// 4. Numerix Input (only add if not already present)
	addOrUpdateComponents(processNumerixInput(numerixComponents))

	// 5. Predator Input (only add if not already present)
	addOrUpdateComponents(processPredatorInput(predatorComponents))

	responseSchemaComponents := processResponseConfig(config.ResponseConfig, response)

	result := linkedhashmap.New()
	if config.ResponseConfig.LogFeatures {
		for _, component := range responseSchemaComponents {
			result.Put(component.FeatureName, component)
		}
		return result, nil
	}
	for _, component := range response {
		result.Put(component.FeatureName, component)
	}
	return result, nil
}

func processLinkedHashMap[T any](linkedHashMap linkedhashmap.Map) []T {
	var response []T
	for _, item := range linkedHashMap.Values() {
		typedItem, ok := item.(T)
		if ok {
			response = append(response, typedItem)
		}
	}
	return response
}

func processNumerixOutput(numerixComponents []NumerixComponentConfig) []SchemaComponents {
	if len(numerixComponents) == 0 {
		return nil
	}

	var response []SchemaComponents
	for _, comp := range numerixComponents {
		response = append(response, SchemaComponents{
			FeatureName: comp.ScoreColumn,
			FeatureType: comp.DataType,
			FeatureSize: 1,
		})
	}
	return response
}

func processNumerixInput(numerixComponents []NumerixComponentConfig) []SchemaComponents {
	if len(numerixComponents) == 0 {
		return nil
	}

	var response []SchemaComponents
	for _, comp := range numerixComponents {
		for input, featureName := range comp.ScoreMapping {
			inputParts := strings.Split(input, "@")
			if len(inputParts) < 2 {
				continue
			}
			schemaComponent := SchemaComponents{
				FeatureName: featureName,
				FeatureType: inputParts[1],
				FeatureSize: 1,
			}
			response = append(response, schemaComponent)
		}
	}
	return response
}

func getFeatureName(prefix, entityLabel, fgLabel, feature string) string {
	featureName := ""
	if prefix != "" {
		featureName = prefix
	}
	if entityLabel != "" {
		featureName = featureName + entityLabel + ":"
	}
	if fgLabel != "" {
		featureName = featureName + fgLabel + ":"
	}
	return featureName + feature
}

func processFS(featureComponents []FeatureComponentConfig) []SchemaComponents {
	if len(featureComponents) == 0 {
		return nil
	}

	var response []SchemaComponents
	for _, featureComponent := range featureComponents {
		for _, featureGroup := range featureComponent.FSRequest.FeatureGroups {
			for _, feature := range featureGroup.Features {
				response = append(response, SchemaComponents{
					FeatureName: getFeatureName(featureComponent.ColNamePrefix, featureComponent.FSRequest.Label, featureGroup.Label, feature),
					FeatureType: featureGroup.DataType,
					FeatureSize: 1,
				})
			}
		}
	}
	return response
}

func processPredatorOutput(predatorComponents []PredatorComponentConfig) []SchemaComponents {
	if len(predatorComponents) == 0 {
		return nil
	}

	var response []SchemaComponents
	for _, predatorComponent := range predatorComponents {
		for _, output := range predatorComponent.Outputs {
			for index, modelScore := range output.ModelScores {
				var featureSize any = 1
				dataType := output.DataType
				if index < len(output.ModelScoresDims) {
					featureSize, dataType = getPredatorFeatureTypeAndSize(output.DataType, output.ModelScoresDims[index])
				}
				response = append(response, SchemaComponents{
					FeatureName: modelScore,
					FeatureType: dataType,
					FeatureSize: featureSize,
				})
			}
		}
	}
	return response
}

func processPredatorInput(predatorComponents []PredatorComponentConfig) []SchemaComponents {
	if len(predatorComponents) == 0 {
		return nil
	}

	var response []SchemaComponents
	for _, predatorComponent := range predatorComponents {
		for _, input := range predatorComponent.Inputs {
			for _, feature := range input.Features {
				size, dataType := getPredatorFeatureTypeAndSize(input.DataType, input.Shape)
				response = append(response, SchemaComponents{
					FeatureName: feature,
					FeatureType: dataType,
					FeatureSize: size,
				})
			}
		}
	}
	return response
}

func getPredatorFeatureTypeAndSize(dataType string, shape []int) (int, string) {
	if len(shape) == 1 && shape[0] == 1 {
		return 1, dataType
	}
	if len(shape) == 2 && shape[0] == -1 {
		return shape[1], dataType + "Vector"
	}
	if len(shape) > 0 {
		return shape[0], dataType + "Vector"
	}
	return 1, dataType
}

func processResponseConfig(responseConfig ResponseConfig, schemaComponents []SchemaComponents) []SchemaComponents {
	if len(responseConfig.Features) == 0 {
		return nil
	}

	var response []SchemaComponents
	schemaMap := make(map[string]SchemaComponents)
	for _, component := range schemaComponents {
		schemaMap[component.FeatureName] = component
	}

	for _, feature := range responseConfig.Features {
		if existingComponent, exists := schemaMap[feature]; exists {
			response = append(response, SchemaComponents{
				FeatureName: feature,
				FeatureType: existingComponent.FeatureType,
				FeatureSize: existingComponent.FeatureSize,
			})
		} else {
			response = append(response, SchemaComponents{
				FeatureName: feature,
				FeatureType: "String",
				FeatureSize: 1,
			})
		}
	}
	return response
}
