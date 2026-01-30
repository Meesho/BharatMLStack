package handler

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow"
	"github.com/Meesho/BharatMLStack/horizon/pkg/configschemaclient"
)

// BuildFeatureSchemaFromInferflow builds a feature schema from the component and response configs.
// It converts inferflow types to configschemaclient types, calls the client, and converts back.
func BuildFeatureSchemaFromInferflow(componentConfig *inferflow.ComponentConfig, responseConfig *inferflow.ResponseConfig) []inferflow.SchemaComponents {
	if componentConfig == nil {
		return nil
	}

	clientSchema := configschemaclient.BuildFeatureSchema(
		toClientComponentConfig(componentConfig),
		toClientResponseConfig(responseConfig),
	)

	return toInferflowSchemaComponents(clientSchema)
}

// ProcessResponseConfigFromInferflow processes the response config and builds schema components.
func ProcessResponseConfigFromInferflow(responseConfig *inferflow.ResponseConfig, schemaComponents []inferflow.SchemaComponents) []inferflow.SchemaComponents {
	if responseConfig == nil || len(responseConfig.Features) == 0 {
		return nil
	}

	clientSchemaComponents := make([]configschemaclient.SchemaComponents, len(schemaComponents))
	for i, c := range schemaComponents {
		clientSchemaComponents[i] = configschemaclient.SchemaComponents{
			FeatureName: c.FeatureName,
			FeatureType: c.FeatureType,
			FeatureSize: c.FeatureSize,
		}
	}

	result := configschemaclient.ProcessResponseConfig(
		toClientResponseConfig(responseConfig),
		clientSchemaComponents,
	)

	return toInferflowSchemaComponents(result)
}

func toClientComponentConfig(config *inferflow.ComponentConfig) *configschemaclient.ComponentConfig {
	if config == nil {
		return nil
	}

	return &configschemaclient.ComponentConfig{
		CacheEnabled:       config.CacheEnabled,
		CacheTTL:           config.CacheTTL,
		CacheVersion:       config.CacheVersion,
		FeatureComponents:  toClientFeatureComponents(config.FeatureComponents),
		RTPComponents:      toClientRTPComponents(config.RTPComponents),
		PredatorComponents: toClientPredatorComponents(config.PredatorComponents),
		NumerixComponents:  toClientNumerixComponents(config.NumerixComponents),
	}
}

func toClientResponseConfig(config *inferflow.ResponseConfig) *configschemaclient.ResponseConfig {
	if config == nil {
		return nil
	}

	return &configschemaclient.ResponseConfig{
		LoggingPerc:          config.LoggingPerc,
		ModelSchemaPerc:      config.ModelSchemaPerc,
		Features:             config.Features,
		LogSelectiveFeatures: config.LogSelectiveFeatures,
		LogBatchSize:         config.LogBatchSize,
	}
}

func toClientNumerixComponents(components []inferflow.NumerixComponent) []configschemaclient.NumerixComponent {
	if len(components) == 0 {
		return nil
	}

	result := make([]configschemaclient.NumerixComponent, len(components))
	for i, c := range components {
		result[i] = configschemaclient.NumerixComponent{
			Component:    c.Component,
			ComponentID:  c.ComponentID,
			ScoreCol:     c.ScoreCol,
			ComputeID:    c.ComputeID,
			ScoreMapping: c.ScoreMapping,
			DataType:     c.DataType,
		}
	}
	return result
}

func toClientFeatureComponents(components []inferflow.FeatureComponent) []configschemaclient.FeatureComponent {
	if len(components) == 0 {
		return nil
	}

	result := make([]configschemaclient.FeatureComponent, len(components))
	for i, c := range components {
		result[i] = configschemaclient.FeatureComponent{
			Component:         c.Component,
			ComponentID:       c.ComponentID,
			ColNamePrefix:     c.ColNamePrefix,
			CompCacheEnabled:  c.CompCacheEnabled,
			CompCacheTTL:      c.CompCacheTTL,
			CompositeID:       c.CompositeID,
			FSKeys:            toClientFSKeys(c.FSKeys),
			FSRequest:         toClientFSRequest(c.FSRequest),
			FSFlattenRespKeys: c.FSFlattenRespKeys,
		}
	}
	return result
}

func toClientRTPComponents(components []inferflow.RTPComponent) []configschemaclient.RTPComponent {
	if len(components) == 0 {
		return nil
	}

	result := make([]configschemaclient.RTPComponent, len(components))
	for i, c := range components {
		result[i] = configschemaclient.RTPComponent{
			Component:         c.Component,
			ComponentID:       c.ComponentID,
			CompositeID:       c.CompositeID,
			FSKeys:            toClientFSKeys(c.FSKeys),
			FSRequest:         toClientFSRequest(c.FSRequest),
			FSFlattenRespKeys: c.FSFlattenRespKeys,
			ColNamePrefix:     c.ColNamePrefix,
			CompCacheEnabled:  c.CompCacheEnabled,
		}
	}
	return result
}

func toClientPredatorComponents(components []inferflow.PredatorComponent) []configschemaclient.PredatorComponent {
	if len(components) == 0 {
		return nil
	}

	result := make([]configschemaclient.PredatorComponent, len(components))
	for i, c := range components {
		result[i] = configschemaclient.PredatorComponent{
			Component:     c.Component,
			ComponentID:   c.ComponentID,
			ModelName:     c.ModelName,
			ModelEndPoint: c.ModelEndPoint,
			Calibration:   c.Calibration,
			Deadline:      c.Deadline,
			BatchSize:     c.BatchSize,
			Inputs:        toClientPredatorInputs(c.Inputs),
			Outputs:       toClientPredatorOutputs(c.Outputs),
			RoutingConfig: toClientRoutingConfigs(c.RoutingConfig),
		}
	}
	return result
}

func toClientFSKeys(keys []inferflow.FSKey) []configschemaclient.FSKey {
	if len(keys) == 0 {
		return nil
	}

	result := make([]configschemaclient.FSKey, len(keys))
	for i, k := range keys {
		result[i] = configschemaclient.FSKey{
			Schema: k.Schema,
			Col:    k.Col,
		}
	}
	return result
}

func toClientFSRequest(req *inferflow.FSRequest) *configschemaclient.FSRequest {
	if req == nil {
		return nil
	}

	return &configschemaclient.FSRequest{
		Label:         req.Label,
		FeatureGroups: toClientFSFeatureGroups(req.FeatureGroups),
	}
}

func toClientFSFeatureGroups(groups []inferflow.FSFeatureGroup) []configschemaclient.FSFeatureGroup {
	if len(groups) == 0 {
		return nil
	}

	result := make([]configschemaclient.FSFeatureGroup, len(groups))
	for i, g := range groups {
		result[i] = configschemaclient.FSFeatureGroup{
			Label:    g.Label,
			Features: g.Features,
			DataType: g.DataType,
		}
	}
	return result
}

func toClientPredatorInputs(inputs []inferflow.PredatorInput) []configschemaclient.PredatorInput {
	if len(inputs) == 0 {
		return nil
	}

	result := make([]configschemaclient.PredatorInput, len(inputs))
	for i, input := range inputs {
		result[i] = configschemaclient.PredatorInput{
			Name:     input.Name,
			Features: input.Features,
			Dims:     input.Dims,
			DataType: input.DataType,
		}
	}
	return result
}

func toClientPredatorOutputs(outputs []inferflow.PredatorOutput) []configschemaclient.PredatorOutput {
	if len(outputs) == 0 {
		return nil
	}

	result := make([]configschemaclient.PredatorOutput, len(outputs))
	for i, output := range outputs {
		result[i] = configschemaclient.PredatorOutput{
			Name:            output.Name,
			ModelScores:     output.ModelScores,
			ModelScoresDims: output.ModelScoresDims,
			DataType:        output.DataType,
		}
	}
	return result
}

func toClientRoutingConfigs(configs []inferflow.RoutingConfig) []configschemaclient.RoutingConfig {
	if len(configs) == 0 {
		return nil
	}

	result := make([]configschemaclient.RoutingConfig, len(configs))
	for i, c := range configs {
		result[i] = configschemaclient.RoutingConfig{
			ModelName:         c.ModelName,
			ModelEndpoint:     c.ModelEndpoint,
			RoutingPercentage: c.RoutingPercentage,
		}
	}
	return result
}

func toInferflowSchemaComponents(components []configschemaclient.SchemaComponents) []inferflow.SchemaComponents {
	if len(components) == 0 {
		return nil
	}

	result := make([]inferflow.SchemaComponents, len(components))
	for i, c := range components {
		result[i] = inferflow.SchemaComponents{
			FeatureName: c.FeatureName,
			FeatureType: c.FeatureType,
			FeatureSize: c.FeatureSize,
		}
	}
	return result
}
