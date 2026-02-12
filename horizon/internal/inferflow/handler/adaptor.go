package handler

import (
	etcdModel "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/etcd"
	dbModel "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow"
	"github.com/rs/zerolog/log"
)

type ConfigMapProvider interface {
	GetConfigMapping() dbModel.ConfigMapping
}

func AdaptOnboardRequestToDBPayload(req interface{}, inferflowConfig InferflowConfig, onboardPayload OnboardPayload) (dbModel.Payload, error) {
	var payload dbModel.Payload

	payload.ConfigMapping = AdaptToDBConfig(req)

	dbResponseConfig := AdaptToDBResponseConfig(inferflowConfig)

	dbPredatorComponents := AdaptToDBPredatorComponent(inferflowConfig)

	dbNumerixComponents := AdaptToDBNumerixComponent(inferflowConfig)

	// Use interface for internal component adaptation (RTP, SeenScore)
	dbRTPComponents := InternalComponentBuilderInstance.AdaptToDBRTPComponent(inferflowConfig)
	dbSeenScoreComponents := InternalComponentBuilderInstance.AdaptToDBSeenScoreComponent(inferflowConfig)

	featureComponents := AdaptToDBFeatureComponent(inferflowConfig)

	dbComponentConfig := AdaptToDBComponentConfig(inferflowConfig, featureComponents, dbNumerixComponents, dbPredatorComponents, dbRTPComponents, dbSeenScoreComponents)

	dbDagExecutionConfig := AdaptToDBDagExecutionConfig(inferflowConfig)

	payload.ConfigValue = AdaptToDBConfigValue(dbDagExecutionConfig, dbComponentConfig, dbResponseConfig)
	payload.RequestPayload = AdaptToDBOnboardPayload(onboardPayload)

	return payload, nil
}

func AdaptEditRequestToDBPayload(req interface{}, inferflowConfig InferflowConfig, onboardPayload OnboardPayload) (dbModel.Payload, error) {
	var payload dbModel.Payload

	payload.ConfigMapping = AdaptToDBConfig(req)

	dbResponseConfig := AdaptToDBResponseConfig(inferflowConfig)

	dbPredatorComponents := AdaptToDBPredatorComponent(inferflowConfig)

	dbNumerixComponents := AdaptToDBNumerixComponent(inferflowConfig)

	// Use interface for internal component adaptation (RTP, SeenScore)
	dbRTPComponents := InternalComponentBuilderInstance.AdaptToDBRTPComponent(inferflowConfig)
	dbSeenScoreComponents := InternalComponentBuilderInstance.AdaptToDBSeenScoreComponent(inferflowConfig)

	featureComponents := AdaptToDBFeatureComponent(inferflowConfig)

	dbComponentConfig := AdaptToDBComponentConfig(inferflowConfig, featureComponents, dbNumerixComponents, dbPredatorComponents, dbRTPComponents, dbSeenScoreComponents)

	dbDagExecutionConfig := AdaptToDBDagExecutionConfig(inferflowConfig)

	payload.ConfigValue = AdaptToDBConfigValue(dbDagExecutionConfig, dbComponentConfig, dbResponseConfig)

	payload.RequestPayload = AdaptToDBOnboardPayload(onboardPayload)

	return payload, nil
}

func AdaptCloneConfigRequestToDBPayload(req interface{}, inferflowConfig InferflowConfig, onboardPayload OnboardPayload) (dbModel.Payload, error) {
	var payload dbModel.Payload

	payload.ConfigMapping = AdaptToDBConfig(req)

	dbResponseConfig := AdaptToDBResponseConfig(inferflowConfig)

	dbPredatorComponents := AdaptToDBPredatorComponent(inferflowConfig)

	dbNumerixComponents := AdaptToDBNumerixComponent(inferflowConfig)

	// Use interface for internal component adaptation (RTP, SeenScore)
	dbRTPComponents := InternalComponentBuilderInstance.AdaptToDBRTPComponent(inferflowConfig)
	dbSeenScoreComponents := InternalComponentBuilderInstance.AdaptToDBSeenScoreComponent(inferflowConfig)

	featureComponents := AdaptToDBFeatureComponent(inferflowConfig)

	dbComponentConfig := AdaptToDBComponentConfig(inferflowConfig, featureComponents, dbNumerixComponents, dbPredatorComponents, dbRTPComponents, dbSeenScoreComponents)

	dbDagExecutionConfig := AdaptToDBDagExecutionConfig(inferflowConfig)

	payload.ConfigValue = AdaptToDBConfigValue(dbDagExecutionConfig, dbComponentConfig, dbResponseConfig)
	payload.RequestPayload = AdaptToDBOnboardPayload(onboardPayload)

	return payload, nil
}

func AdaptPromoteRequestToDBPayload(req interface{}, requestPayload RequestConfig) (dbModel.Payload, error) {
	var payload dbModel.Payload

	payload.ConfigMapping = AdaptToDBConfig(req)

	inferflowConfig := req.(PromoteConfigRequest).Payload.ConfigValue

	dbResponseConfig := AdaptToDBResponseConfig(inferflowConfig)

	dbPredatorComponents := AdaptToDBPredatorComponent(inferflowConfig)

	dbNumerixComponents := AdaptToDBNumerixComponent(inferflowConfig)

	// Use interface for internal component adaptation (RTP, SeenScore)
	dbRTPComponents := InternalComponentBuilderInstance.AdaptToDBRTPComponent(inferflowConfig)
	dbSeenScoreComponents := InternalComponentBuilderInstance.AdaptToDBSeenScoreComponent(inferflowConfig)

	featureComponents := AdaptToDBFeatureComponent(inferflowConfig)

	dbComponentConfig := AdaptToDBComponentConfig(inferflowConfig, featureComponents, dbNumerixComponents, dbPredatorComponents, dbRTPComponents, dbSeenScoreComponents)

	dbDagExecutionConfig := AdaptToDBDagExecutionConfig(inferflowConfig)

	payload.ConfigValue = AdaptToDBConfigValue(dbDagExecutionConfig, dbComponentConfig, dbResponseConfig)

	payload.RequestPayload = AdaptToDBOnboardPayload(requestPayload.Payload.RequestPayload)

	return payload, nil
}

func AdaptScaleUpRequestToDBPayload(req interface{}, requestPayload RequestConfig) (dbModel.Payload, error) {
	var payload dbModel.Payload

	payload.ConfigMapping = AdaptToDBConfig(req)

	inferflowConfig := req.(ScaleUpConfigRequest).Payload.ConfigValue

	dbResponseConfig := AdaptToDBResponseConfig(inferflowConfig)

	dbPredatorComponents := AdaptToDBPredatorComponent(inferflowConfig)

	dbNumerixComponents := AdaptToDBNumerixComponent(inferflowConfig)

	// Use interface for internal component adaptation (RTP, SeenScore)
	dbRTPComponents := InternalComponentBuilderInstance.AdaptToDBRTPComponent(inferflowConfig)
	dbSeenScoreComponents := InternalComponentBuilderInstance.AdaptToDBSeenScoreComponent(inferflowConfig)

	featureComponents := AdaptToDBFeatureComponent(inferflowConfig)

	dbComponentConfig := AdaptToDBComponentConfig(inferflowConfig, featureComponents, dbNumerixComponents, dbPredatorComponents, dbRTPComponents, dbSeenScoreComponents)

	dbDagExecutionConfig := AdaptToDBDagExecutionConfig(inferflowConfig)

	payload.ConfigValue = AdaptToDBConfigValue(dbDagExecutionConfig, dbComponentConfig, dbResponseConfig)
	payload.RequestPayload = AdaptToDBOnboardPayload(requestPayload.Payload.RequestPayload)

	return payload, nil
}

func AdaptToDBConfig(req interface{}) dbModel.ConfigMapping {

	if provider, ok := req.(ConfigMapProvider); ok {
		return provider.GetConfigMapping()
	} else {
		log.Warn().Msgf("AdaptToDBConfig received a type that does not implement ConfigMapProvider: %T", req)
		return dbModel.ConfigMapping{}
	}
}

func AdaptToDBResponseConfig(inferflowConfig InferflowConfig) dbModel.ResponseConfig {
	return dbModel.ResponseConfig{
		LoggingPerc:          inferflowConfig.ResponseConfig.LoggingPerc,
		ModelSchemaPerc:      inferflowConfig.ResponseConfig.ModelSchemaPerc,
		Features:             inferflowConfig.ResponseConfig.Features,
		LogSelectiveFeatures: inferflowConfig.ResponseConfig.LogSelectiveFeatures,
		LogBatchSize:         inferflowConfig.ResponseConfig.LogBatchSize,
		LoggingTTL:           inferflowConfig.ResponseConfig.LoggingTTL,
	}
}

func AdaptToDBPredatorComponent(inferflowConfig InferflowConfig) []dbModel.PredatorComponent {
	var predatorComponents []dbModel.PredatorComponent

	for _, ranker := range inferflowConfig.ComponentConfig.PredatorComponents {
		dbInputs := make([]dbModel.PredatorInput, len(ranker.Inputs))
		for i, input := range ranker.Inputs {
			dbInputs[i] = dbModel.PredatorInput{
				Name:     input.Name,
				Features: input.Features,
				Dims:     input.Dims,
				DataType: input.DataType,
			}
		}

		dbOutputs := make([]dbModel.PredatorOutput, len(ranker.Outputs))
		for i, output := range ranker.Outputs {
			dbOutputs[i] = dbModel.PredatorOutput{
				Name:            output.Name,
				ModelScores:     output.ModelScores,
				ModelScoresDims: output.ModelScoresDims,
				DataType:        output.DataType,
			}
		}

		routingConfig := make([]dbModel.RoutingConfig, len(ranker.RoutingConfig))

		for i, config := range ranker.RoutingConfig {

			routingConfig[i] = dbModel.RoutingConfig{

				ModelName: config.ModelName,

				ModelEndpoint: config.ModelEndpoint,

				RoutingPercentage: config.RoutingPercentage,
			}

		}

		predatorComp := dbModel.PredatorComponent{
			Component:     ranker.Component,
			ComponentID:   ranker.ComponentID,
			Calibration:   ranker.Calibration,
			ModelName:     ranker.ModelName,
			ModelEndPoint: ranker.ModelEndPoint,
			Deadline:      ranker.Deadline,
			BatchSize:     ranker.BatchSize,
			Inputs:        dbInputs,
			Outputs:       dbOutputs,
			RoutingConfig: routingConfig,
		}
		predatorComponents = append(predatorComponents, predatorComp)
	}

	return predatorComponents
}

func AdaptToDBNumerixComponent(inferflowConfig InferflowConfig) []dbModel.NumerixComponent {
	var NumerixComponents []dbModel.NumerixComponent

	for _, reRanker := range inferflowConfig.ComponentConfig.NumerixComponents {
		NumerixComp := dbModel.NumerixComponent{
			Component:    reRanker.Component,
			ComponentID:  reRanker.ComponentID,
			ScoreCol:     reRanker.ScoreCol,
			ComputeID:    reRanker.ComputeID,
			ScoreMapping: reRanker.ScoreMapping,
			DataType:     reRanker.DataType,
		}
		NumerixComponents = append(NumerixComponents, NumerixComp)
	}

	return NumerixComponents
}


func AdaptToDBFeatureComponent(inferflowConfig InferflowConfig) []dbModel.FeatureComponent {
	var featureComponents []dbModel.FeatureComponent

	for _, fc := range inferflowConfig.ComponentConfig.FeatureComponents {
		fsKeys := make([]dbModel.FSKey, len(fc.FSKeys))
		for i, key := range fc.FSKeys {
			fsKeys[i] = dbModel.FSKey{
				Schema: key.Schema,
				Col:    key.Col,
			}
		}

		fsFeatureGroups := make([]dbModel.FSFeatureGroup, len(fc.FSRequest.FeatureGroups))
		for i, grp := range fc.FSRequest.FeatureGroups {
			fsFeatureGroups[i] = dbModel.FSFeatureGroup{
				Label:    grp.Label,
				Features: grp.Features,
				DataType: grp.DataType,
			}
		}

		fsRequest := dbModel.FSRequest{
			Label:         fc.FSRequest.Label,
			FeatureGroups: fsFeatureGroups,
		}

		comp := dbModel.FeatureComponent{
			Component:         fc.Component,
			ComponentID:       fc.ComponentID,
			ColNamePrefix:     fc.ColNamePrefix,
			CompCacheEnabled:  fc.CompCacheEnabled,
			CompositeID:       fc.CompositeID,
			FSKeys:            fsKeys,
			FSRequest:         &fsRequest,
			FSFlattenRespKeys: fc.FSFlattenRespKeys,
		}
		featureComponents = append(featureComponents, comp)
	}

	return featureComponents
}

func AdaptToDBComponentConfig(inferflowConfig InferflowConfig, featureComponents []dbModel.FeatureComponent, NumerixComponents []dbModel.NumerixComponent, predatorComponents []dbModel.PredatorComponent, rtpComponents []dbModel.RTPComponent, seenScoreComponents []dbModel.SeenScoreComponent) dbModel.ComponentConfig {
	return dbModel.ComponentConfig{
		CacheEnabled:        inferflowConfig.ComponentConfig.CacheEnabled,
		CacheTTL:            inferflowConfig.ComponentConfig.CacheTTL,
		CacheVersion:        inferflowConfig.ComponentConfig.CacheVersion,
		FeatureComponents:   featureComponents,
		PredatorComponents:  predatorComponents,
		NumerixComponents:   NumerixComponents,
		RTPComponents:       rtpComponents,
		SeenScoreComponents: seenScoreComponents,
	}
}

func AdaptToDBDagExecutionConfig(inferflowConfig InferflowConfig) dbModel.DagExecutionConfig {
	return dbModel.DagExecutionConfig{
		ComponentDependency: inferflowConfig.DagExecutionConfig.ComponentDependency,
	}
}

func AdaptToDBConfigValue(dagExecutionConfig dbModel.DagExecutionConfig, componentConfig dbModel.ComponentConfig, responseConfig dbModel.ResponseConfig) dbModel.InferflowConfig {
	return dbModel.InferflowConfig{
		DagExecutionConfig: dagExecutionConfig,
		ComponentConfig:    componentConfig,
		ResponseConfig:     responseConfig,
	}
}

func AdaptToDBOnboardPayload(onboardPayload OnboardPayload) dbModel.OnboardPayload {
	dbOnboardPayload := dbModel.OnboardPayload{
		RealEstate:       onboardPayload.RealEstate,
		Tenant:           onboardPayload.Tenant,
		ConfigIdentifier: onboardPayload.ConfigIdentifier,
		Rankers:          make([]dbModel.OnboardRanker, len(onboardPayload.Rankers)),
		ReRankers:        make([]dbModel.OnboardReRanker, len(onboardPayload.ReRankers)),
		Response: dbModel.ResponseConfig{
			LoggingPerc:          onboardPayload.Response.PrismLoggingPerc,
			ModelSchemaPerc:      onboardPayload.Response.RankerSchemaFeaturesInResponsePerc,
			Features:             onboardPayload.Response.ResponseFeatures,
			LogSelectiveFeatures: onboardPayload.Response.LogSelectiveFeatures,
			LogBatchSize:         onboardPayload.Response.LogBatchSize,
			LoggingTTL:           onboardPayload.Response.LoggingTTL,
		},
		ConfigMapping: dbModel.ConfigMapping{
			AppToken:              onboardPayload.ConfigMapping.AppToken,
			ConnectionConfigID:    onboardPayload.ConfigMapping.ConnectionConfigID,
			DeployableID:          onboardPayload.ConfigMapping.DeployableID,
			ResponseDefaultValues: onboardPayload.ConfigMapping.ResponseDefaultValues,
		},
	}

	for i, ranker := range onboardPayload.Rankers {
		dbOnboardPayload.Rankers[i] = dbModel.OnboardRanker{
			ModelName:     ranker.ModelName,
			Calibration:   ranker.Calibration,
			EndPoint:      ranker.EndPoint,
			EntityID:      ranker.EntityID,
			Inputs:        make([]dbModel.PredatorInput, len(ranker.Inputs)),
			Outputs:       make([]dbModel.PredatorOutput, len(ranker.Outputs)),
			BatchSize:     ranker.BatchSize,
			Deadline:      ranker.Deadline,
			RoutingConfig: make([]dbModel.RoutingConfig, len(ranker.RoutingConfig)),
		}
		for j, input := range ranker.Inputs {
			dbOnboardPayload.Rankers[i].Inputs[j] = dbModel.PredatorInput{
				Name:     input.Name,
				Features: input.Features,
				Dims:     input.Dims,
				DataType: input.DataType,
			}
		}

		for k, config := range ranker.RoutingConfig {
			dbOnboardPayload.Rankers[i].RoutingConfig[k] = dbModel.RoutingConfig{
				ModelName:         config.ModelName,
				ModelEndpoint:     config.ModelEndpoint,
				RoutingPercentage: config.RoutingPercentage,
			}
		}
		for j, output := range ranker.Outputs {
			dbOnboardPayload.Rankers[i].Outputs[j] = dbModel.PredatorOutput{
				Name:            output.Name,
				ModelScores:     output.ModelScores,
				ModelScoresDims: output.ModelScoresDims,
				DataType:        output.DataType,
			}
		}
	}
	for i, reRanker := range onboardPayload.ReRankers {
		dbOnboardPayload.ReRankers[i] = dbModel.OnboardReRanker{
			Score:       reRanker.Score,
			EqID:        reRanker.EqID,
			EqVariables: reRanker.EqVariables,
			DataType:    reRanker.DataType,
			EntityID:    reRanker.EntityID,
		}
	}
	return dbOnboardPayload
}

func AdaptFromDbToInferFlowConfig(dbConfig dbModel.InferflowConfig) InferflowConfig {
	return InferflowConfig{
		DagExecutionConfig: AdaptFromDbToDagExecutionConfig(dbConfig.DagExecutionConfig),
		ComponentConfig:    AdaptFromDbToComponentConfig(dbConfig.ComponentConfig),
		ResponseConfig:     AdaptFromDbToResponseConfig(dbConfig.ResponseConfig),
	}
}

func AdaptFromDbToOnboardPayload(dbOnboardPayload dbModel.OnboardPayload) OnboardPayload {
	onboardPayload := OnboardPayload{
		RealEstate:       dbOnboardPayload.RealEstate,
		Tenant:           dbOnboardPayload.Tenant,
		ConfigIdentifier: dbOnboardPayload.ConfigIdentifier,
		Rankers:          []Ranker{},
		ReRankers:        []ReRanker{},
		Response: ResponseConfig{
			PrismLoggingPerc:                   dbOnboardPayload.Response.LoggingPerc,
			RankerSchemaFeaturesInResponsePerc: dbOnboardPayload.Response.ModelSchemaPerc,
			ResponseFeatures:                   dbOnboardPayload.Response.Features,
			LogSelectiveFeatures:               dbOnboardPayload.Response.LogSelectiveFeatures,
			LogBatchSize:                       dbOnboardPayload.Response.LogBatchSize,
			LoggingTTL:                         dbOnboardPayload.Response.LoggingTTL,
		},
		ConfigMapping: ConfigMapping{
			AppToken:              dbOnboardPayload.ConfigMapping.AppToken,
			ConnectionConfigID:    dbOnboardPayload.ConfigMapping.ConnectionConfigID,
			DeployableID:          dbOnboardPayload.ConfigMapping.DeployableID,
			ResponseDefaultValues: dbOnboardPayload.ConfigMapping.ResponseDefaultValues,
		},
	}

	for i, predatorComponent := range dbOnboardPayload.Rankers {
		onboardPayload.Rankers = append(onboardPayload.Rankers, Ranker{
			ModelName:     predatorComponent.ModelName,
			Calibration:   predatorComponent.Calibration,
			EndPoint:      predatorComponent.EndPoint,
			Inputs:        make([]Input, len(predatorComponent.Inputs)),
			Outputs:       make([]Output, len(predatorComponent.Outputs)),
			EntityID:      predatorComponent.EntityID,
			BatchSize:     predatorComponent.BatchSize,
			Deadline:      predatorComponent.Deadline,
			RoutingConfig: make([]RoutingConfig, len(predatorComponent.RoutingConfig)),
		})
		for j, input := range predatorComponent.Inputs {
			onboardPayload.Rankers[i].Inputs[j] = Input{
				Name:     input.Name,
				Features: input.Features,
				Dims:     input.Dims,
				DataType: input.DataType,
			}
		}
		for j, output := range predatorComponent.Outputs {
			onboardPayload.Rankers[i].Outputs[j] = Output{
				Name:            output.Name,
				ModelScores:     output.ModelScores,
				ModelScoresDims: output.ModelScoresDims,
				DataType:        output.DataType,
			}
		}

		for k, config := range predatorComponent.RoutingConfig {
			onboardPayload.Rankers[i].RoutingConfig[k] = RoutingConfig{
				ModelName:         config.ModelName,
				ModelEndpoint:     config.ModelEndpoint,
				RoutingPercentage: config.RoutingPercentage,
			}
		}
	}

	for _, reRanker := range dbOnboardPayload.ReRankers {
		onboardPayload.ReRankers = append(onboardPayload.ReRankers, ReRanker{
			Score:       reRanker.Score,
			EqID:        reRanker.EqID,
			EqVariables: reRanker.EqVariables,
			DataType:    reRanker.DataType,
			EntityID:    reRanker.EntityID,
		})
	}

	return onboardPayload
}

func AdaptFromDbToComponentConfig(dbComponentConfig dbModel.ComponentConfig) *ComponentConfig {
	return &ComponentConfig{
		CacheEnabled:        dbComponentConfig.CacheEnabled,
		CacheTTL:            dbComponentConfig.CacheTTL,
		CacheVersion:        dbComponentConfig.CacheVersion,
		FeatureComponents:   AdaptFromDbToFeatureComponent(dbComponentConfig.FeatureComponents),
		PredatorComponents:  AdaptFromDbToPredatorComponent(dbComponentConfig.PredatorComponents),
		NumerixComponents:   AdaptFromDbToNumerixComponent(dbComponentConfig.NumerixComponents),
		RTPComponents:       InternalComponentBuilderInstance.AdaptFromDbToRTPComponent(dbComponentConfig.RTPComponents),
		SeenScoreComponents: InternalComponentBuilderInstance.AdaptFromDbToSeenScoreComponent(dbComponentConfig.SeenScoreComponents),
	}
}

func AdaptFromDbToResponseConfig(dbResponseConfig dbModel.ResponseConfig) *FinalResponseConfig {
	return &FinalResponseConfig{
		LoggingPerc:          dbResponseConfig.LoggingPerc,
		ModelSchemaPerc:      dbResponseConfig.ModelSchemaPerc,
		Features:             dbResponseConfig.Features,
		LogSelectiveFeatures: dbResponseConfig.LogSelectiveFeatures,
		LogBatchSize:         dbResponseConfig.LogBatchSize,
		LoggingTTL:           dbResponseConfig.LoggingTTL,
	}
}

func AdaptFromDbToDagExecutionConfig(dbDagExecutionConfig dbModel.DagExecutionConfig) *DagExecutionConfig {
	return &DagExecutionConfig{
		ComponentDependency: dbDagExecutionConfig.ComponentDependency,
	}
}

func AdaptFromDbToPredatorComponent(dbPredatorComponents []dbModel.PredatorComponent) []PredatorComponent {

	var predatorComponents []PredatorComponent
	for _, predatorComponent := range dbPredatorComponents {
		dbInputs := make([]PredatorInput, len(predatorComponent.Inputs))
		for i, input := range predatorComponent.Inputs {
			dbInputs[i] = PredatorInput{
				Name:     input.Name,
				Features: input.Features,
				Dims:     input.Dims,
				DataType: input.DataType,
			}
		}

		dbOutputs := make([]PredatorOutput, len(predatorComponent.Outputs))
		for i, output := range predatorComponent.Outputs {
			dbOutputs[i] = PredatorOutput{
				Name:            output.Name,
				ModelScores:     output.ModelScores,
				ModelScoresDims: output.ModelScoresDims,
				DataType:        output.DataType,
			}
		}

		routingConfig := make([]RoutingConfig, len(predatorComponent.RoutingConfig))

		for i, config := range predatorComponent.RoutingConfig {
			routingConfig[i] = RoutingConfig{
				ModelName:         config.ModelName,
				ModelEndpoint:     config.ModelEndpoint,
				RoutingPercentage: config.RoutingPercentage,
			}
		}

		predatorComponent := PredatorComponent{
			Component:     predatorComponent.Component,
			ComponentID:   predatorComponent.ComponentID,
			ModelName:     predatorComponent.ModelName,
			ModelEndPoint: predatorComponent.ModelEndPoint,
			Calibration:   predatorComponent.Calibration,
			Deadline:      predatorComponent.Deadline,
			BatchSize:     predatorComponent.BatchSize,
			Inputs:        dbInputs,
			Outputs:       dbOutputs,
			RoutingConfig: routingConfig,
		}
		predatorComponents = append(predatorComponents, predatorComponent)
	}
	return predatorComponents
}

func AdaptFromDbToNumerixComponent(dbNumerixComponents []dbModel.NumerixComponent) []NumerixComponent {

	var NumerixComponents []NumerixComponent
	for _, numerixComponent := range dbNumerixComponents {
		numerixComponent := NumerixComponent{
			Component:    numerixComponent.Component,
			ComponentID:  numerixComponent.ComponentID,
			ScoreCol:     numerixComponent.ScoreCol,
			ComputeID:    numerixComponent.ComputeID,
			ScoreMapping: numerixComponent.ScoreMapping,
			DataType:     numerixComponent.DataType,
		}
		NumerixComponents = append(NumerixComponents, numerixComponent)
	}
	return NumerixComponents
}


func AdaptFromDbToFeatureComponent(dbFeatureComponents []dbModel.FeatureComponent) []FeatureComponent {
	var featureComponents []FeatureComponent
	for _, fc := range dbFeatureComponents {
		fsKeys := make([]FSKey, len(fc.FSKeys))
		for i, key := range fc.FSKeys {
			fsKeys[i] = FSKey{
				Schema: key.Schema,
				Col:    key.Col,
			}
		}

		fsFeatureGroups := make([]FSFeatureGroup, len(fc.FSRequest.FeatureGroups))
		for i, grp := range fc.FSRequest.FeatureGroups {
			fsFeatureGroups[i] = FSFeatureGroup{
				Label:    grp.Label,
				Features: grp.Features,
				DataType: grp.DataType,
			}
		}

		fsRequest := FSRequest{
			Label:         fc.FSRequest.Label,
			FeatureGroups: fsFeatureGroups,
		}

		comp := FeatureComponent{
			Component:         fc.Component,
			ComponentID:       fc.ComponentID,
			ColNamePrefix:     fc.ColNamePrefix,
			CompCacheEnabled:  fc.CompCacheEnabled,
			CompositeID:       fc.CompositeID,
			FSKeys:            fsKeys,
			FSRequest:         &fsRequest,
			FSFlattenRespKeys: fc.FSFlattenRespKeys,
		}
		featureComponents = append(featureComponents, comp)
	}
	return featureComponents
}

func AdaptFromDbToConfigMapping(dbMapping dbModel.ConfigMapping) ConfigMapping {
	return ConfigMapping{
		AppToken:              dbMapping.AppToken,
		ConnectionConfigID:    dbMapping.ConnectionConfigID,
		DeployableID:          dbMapping.DeployableID,
		ResponseDefaultValues: dbMapping.ResponseDefaultValues,
	}
}

func AdaptToEtcdInferFlowConfig(dpConfig dbModel.InferflowConfig) etcdModel.InferflowConfig {
	return etcdModel.InferflowConfig{
		DagExecutionConfig: AdaptToEtcdDagExecutionConfig(dpConfig.DagExecutionConfig),
		ComponentConfig:    AdaptToEtcdComponentConfig(dpConfig.ComponentConfig),
		ResponseConfig:     AdaptToEtcdResponseConfig(dpConfig.ResponseConfig),
	}
}

func AdaptToEtcdComponentConfig(dbComponentConfig dbModel.ComponentConfig) etcdModel.ComponentConfig {
	return etcdModel.ComponentConfig{
		CacheEnabled:        dbComponentConfig.CacheEnabled,
		CacheTTL:            dbComponentConfig.CacheTTL,
		CacheVersion:        dbComponentConfig.CacheVersion,
		FeatureComponents:   AdaptToEtcdFeatureComponent(dbComponentConfig.FeatureComponents),
		PredatorComponents:  AdaptToEtcdPredatorComponent(dbComponentConfig.PredatorComponents),
		NumerixComponents:   AdaptToEtcdNumerixComponent(dbComponentConfig.NumerixComponents),
		RTPComponents:       InternalComponentBuilderInstance.AdaptToEtcdRTPComponent(dbComponentConfig.RTPComponents),
		SeenScoreComponents: InternalComponentBuilderInstance.AdaptToEtcdSeenScoreComponent(dbComponentConfig.SeenScoreComponents),
	}
}

func AdaptToEtcdResponseConfig(dbResponseConfig dbModel.ResponseConfig) etcdModel.FinalResponseConfig {
	return etcdModel.FinalResponseConfig{
		LoggingPerc:          dbResponseConfig.LoggingPerc,
		ModelSchemaPerc:      dbResponseConfig.ModelSchemaPerc,
		Features:             dbResponseConfig.Features,
		LogSelectiveFeatures: dbResponseConfig.LogSelectiveFeatures,
		LogBatchSize:         dbResponseConfig.LogBatchSize,
	}
}

func AdaptToEtcdDagExecutionConfig(dbDagExecutionConfig dbModel.DagExecutionConfig) etcdModel.DagExecutionConfig {
	return etcdModel.DagExecutionConfig{
		ComponentDependency: dbDagExecutionConfig.ComponentDependency,
	}
}

func AdaptToEtcdPredatorComponent(dbPredatorComponents []dbModel.PredatorComponent) []etcdModel.PredatorComponent {

	var predatorComponents []etcdModel.PredatorComponent
	for _, predatorComponent := range dbPredatorComponents {
		dbInputs := make([]etcdModel.PredatorInput, len(predatorComponent.Inputs))
		for i, input := range predatorComponent.Inputs {
			dbInputs[i] = etcdModel.PredatorInput{
				Name:     input.Name,
				Features: input.Features,
				Dims:     input.Dims,
				DataType: input.DataType,
			}
		}

		dbOutputs := make([]etcdModel.PredatorOutput, len(predatorComponent.Outputs))
		for i, output := range predatorComponent.Outputs {
			dbOutputs[i] = etcdModel.PredatorOutput{
				Name:            output.Name,
				ModelScores:     output.ModelScores,
				ModelScoresDims: output.ModelScoresDims,
				DataType:        output.DataType,
			}
		}
		routingConfig := make([]etcdModel.RoutingConfig, len(predatorComponent.RoutingConfig))

		for i, config := range predatorComponent.RoutingConfig {
			routingConfig[i] = etcdModel.RoutingConfig{
				ModelName:         config.ModelName,
				ModelEndpoint:     config.ModelEndpoint,
				RoutingPercentage: config.RoutingPercentage,
			}
		}

		predatorComponent := etcdModel.PredatorComponent{
			Component:     predatorComponent.Component,
			ComponentID:   predatorComponent.ComponentID,
			Calibration:   predatorComponent.Calibration,
			ModelName:     predatorComponent.ModelName,
			ModelEndPoint: predatorComponent.ModelEndPoint,
			Deadline:      predatorComponent.Deadline,
			BatchSize:     predatorComponent.BatchSize,
			Inputs:        dbInputs,
			Outputs:       dbOutputs,
			RoutingConfig: routingConfig,
		}
		predatorComponents = append(predatorComponents, predatorComponent)
	}
	return predatorComponents
}

func AdaptToEtcdNumerixComponent(dbNumerixComponents []dbModel.NumerixComponent) []etcdModel.NumerixComponent {

	var NumerixComponents []etcdModel.NumerixComponent
	for _, NumerixComponent := range dbNumerixComponents {
		NumerixComponent := etcdModel.NumerixComponent{
			Component:    NumerixComponent.Component,
			ComponentID:  NumerixComponent.ComponentID,
			ScoreCol:     NumerixComponent.ScoreCol,
			ComputeID:    NumerixComponent.ComputeID,
			ScoreMapping: NumerixComponent.ScoreMapping,
			DataType:     NumerixComponent.DataType,
		}
		NumerixComponents = append(NumerixComponents, NumerixComponent)
	}
	return NumerixComponents
}


func AdaptToEtcdFeatureComponent(dbFeatureComponents []dbModel.FeatureComponent) []etcdModel.FeatureComponent {
	var featureComponents []etcdModel.FeatureComponent
	for _, fc := range dbFeatureComponents {
		fsKeys := make([]etcdModel.FSKey, len(fc.FSKeys))
		for i, key := range fc.FSKeys {
			fsKeys[i] = etcdModel.FSKey{
				Schema: key.Schema,
				Col:    key.Col,
			}
		}

		fsFeatureGroups := make([]etcdModel.FSFeatureGroup, len(fc.FSRequest.FeatureGroups))
		for i, grp := range fc.FSRequest.FeatureGroups {
			fsFeatureGroups[i] = etcdModel.FSFeatureGroup{
				Label:    grp.Label,
				Features: grp.Features,
				DataType: grp.DataType,
			}
		}

		fsRequest := etcdModel.FSRequest{
			Label:         fc.FSRequest.Label,
			FeatureGroups: fsFeatureGroups,
		}

		comp := etcdModel.FeatureComponent{
			Component:         fc.Component,
			ComponentID:       fc.ComponentID,
			ColNamePrefix:     fc.ColNamePrefix,
			CompCacheEnabled:  fc.CompCacheEnabled,
			CompositeID:       fc.CompositeID,
			FSKeys:            fsKeys,
			FSRequest:         &fsRequest,
			FSFlattenRespKeys: fc.FSFlattenRespKeys,
		}
		featureComponents = append(featureComponents, comp)
	}
	return featureComponents
}
