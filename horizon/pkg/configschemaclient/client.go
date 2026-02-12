package configschemaclient

import (
	"strings"
)

const (
	DataTypeString     = "String"
	DefaultFeatureSize = 1
)

// BuildFeatureSchema builds a feature schema from the component and response configs.
// It processes components in order: FS → RTP → SeenScore → Numerix Output → Predator Output → Numerix Input → Predator Input
func BuildFeatureSchema(componentConfig *ComponentConfig, responseConfig *ResponseConfig) []SchemaComponents {
	if componentConfig == nil {
		return nil
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
				component.FeatureType = DataTypeString
				response = append(response, component)
				existingFeatures[component.FeatureName] = true
			}
		}
	}

	// 1. FS (Feature Store)
	addUniqueComponents(processFS(componentConfig.FeatureComponents))

	// 2. RTP (Real Time Pricing) - Internal component, uses interface
	addUniqueComponents(InternalSchemaProcessorInstance.ProcessRTP(componentConfig.RTPComponents))

	// 3. SeenScore - Internal component, uses interface
	addUniqueComponents(InternalSchemaProcessorInstance.ProcessSeenScore(componentConfig.SeenScoreComponents))

	// 4. Numerix Output
	addUniqueComponents(processNumerixOutput(componentConfig.NumerixComponents))

	// 5. Predator Output
	addUniqueComponents(processPredatorOutput(componentConfig.PredatorComponents))

	// 6. Numerix Input (only add if not already present)
	addOrUpdateComponents(processNumerixInput(componentConfig.NumerixComponents))

	// 7. Predator Input (only add if not already present)
	addOrUpdateComponents(processPredatorInput(componentConfig.PredatorComponents))

	return response
}

func processNumerixOutput(numerixComponents []NumerixComponent) []SchemaComponents {
	if len(numerixComponents) == 0 {
		return nil
	}

	var response []SchemaComponents
	for _, numerixComponent := range numerixComponents {
		response = append(response, SchemaComponents{
			FeatureName: numerixComponent.ScoreCol,
			FeatureType: numerixComponent.DataType,
			FeatureSize: 1,
		})
	}
	return response
}

func processNumerixInput(numerixComponents []NumerixComponent) []SchemaComponents {
	if len(numerixComponents) == 0 {
		return nil
	}

	var response []SchemaComponents
	for _, numerixComponent := range numerixComponents {
		for numerixInput, featureName := range numerixComponent.ScoreMapping {
			inputParts := strings.Split(numerixInput, "@")
			response = append(response, SchemaComponents{
				FeatureName: featureName,
				FeatureType: inputParts[1],
				FeatureSize: 1,
			})
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

func processFS(featureComponents []FeatureComponent) []SchemaComponents {
	if len(featureComponents) == 0 {
		return nil
	}

	var response []SchemaComponents
	for _, featureComponent := range featureComponents {
		if featureComponent.FSRequest == nil {
			continue
		}
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

func processPredatorOutput(predatorComponents []PredatorComponent) []SchemaComponents {
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

func processPredatorInput(predatorComponents []PredatorComponent) []SchemaComponents {
	if len(predatorComponents) == 0 {
		return nil
	}

	var response []SchemaComponents
	for _, predatorComponent := range predatorComponents {
		for _, input := range predatorComponent.Inputs {
			for _, feature := range input.Features {
				size, dataType := getPredatorFeatureTypeAndSize(input.DataType, input.Dims)
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

func getPredatorFeatureTypeAndSize(dataType string, dims []int) (int, string) {
	if len(dims) == 1 && dims[0] == 1 {
		return 1, dataType
	}
	if len(dims) == 2 && dims[0] == -1 {
		return dims[1], dataType + "Vector"
	}
	if len(dims) > 0 {
		return dims[0], dataType + "Vector"
	}
	return 1, dataType
}

// ProcessResponseConfig processes the response config and builds schema components
// based on the features specified in the response config.
func ProcessResponseConfig(responseConfig *ResponseConfig, schemaComponents []SchemaComponents) []SchemaComponents {
	if responseConfig == nil || len(responseConfig.Features) == 0 {
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
				FeatureType: DataTypeString,
				FeatureSize: DefaultFeatureSize,
			})
		}
	}
	return response
}
