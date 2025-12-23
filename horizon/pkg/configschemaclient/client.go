package configschemaclient

import (
	"strings"
)

// BuildFeatureSchema builds a feature schema from the component and response configs.
// It processes components in order: Iris Output → Iris Input → FS → RTP → Predator Input → Predator Output → Response Config
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
				component.FeatureType = "String"
				response = append(response, component)
				existingFeatures[component.FeatureName] = true
			}
		}
	}

	// 1. Iris Output
	addUniqueComponents(processIrisOutput(componentConfig.NumerixComponents))

	// 2. Iris Input
	addUniqueComponents(processIrisInput(componentConfig.NumerixComponents))

	// 3. FS (Feature Store)
	addUniqueComponents(processFS(componentConfig.FeatureComponents))

	// 4. RTP (Real Time Pricing)
	addUniqueComponents(processRTP(componentConfig.RTPComponents))

	// 5. Predator Input (only add if not already present)
	addOrUpdateComponents(processPredatorInput(componentConfig.PredatorComponents))

	// 6. Predator Output (only add if not already present)
	addOrUpdateComponents(processPredatorOutput(componentConfig.PredatorComponents))

	// 7. Response Config features
	responseSchemaComponents := ProcessResponseConfig(responseConfig, response)
	addUniqueComponents(responseSchemaComponents)

	return response
}

func processIrisOutput(irisComponents []NumerixComponent) []SchemaComponents {
	if len(irisComponents) == 0 {
		return nil
	}

	var response []SchemaComponents
	for _, irisComponent := range irisComponents {
		response = append(response, SchemaComponents{
			FeatureName: irisComponent.ScoreCol,
			FeatureType: irisComponent.DataType,
			FeatureSize: 1,
		})
	}
	return response
}

func processIrisInput(irisComponents []NumerixComponent) []SchemaComponents {
	if len(irisComponents) == 0 {
		return nil
	}

	var response []SchemaComponents
	for _, irisComponent := range irisComponents {
		for irisInput, featureName := range irisComponent.ScoreMapping {
			inputParts := strings.Split(irisInput, "@")
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

func processRTP(rtpComponents []RTPComponent) []SchemaComponents {
	if len(rtpComponents) == 0 {
		return nil
	}

	var response []SchemaComponents
	for _, rtpComponent := range rtpComponents {
		if rtpComponent.FSRequest == nil {
			continue
		}
		for _, featureGroup := range rtpComponent.FSRequest.FeatureGroups {
			for _, feature := range featureGroup.Features {
				response = append(response, SchemaComponents{
					FeatureName: getFeatureName(rtpComponent.ColNamePrefix, rtpComponent.FSRequest.Label, featureGroup.Label, feature),
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
				FeatureType: "String",
				FeatureSize: 1,
			})
		}
	}
	return response
}
