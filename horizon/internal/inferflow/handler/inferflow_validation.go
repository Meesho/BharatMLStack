package handler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	mainHandler "github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	mapset "github.com/deckarep/golang-set/v2"
)

func ValidateInferFlowConfig(config InferflowConfig, token string) (Response, error) {
	ComponentConfig := config.ComponentConfig
	if ComponentConfig != nil {
		for _, featureComponent := range ComponentConfig.FeatureComponents {
			entity := featureComponent.FSRequest.Label
			if entity == "dummy" {
				continue
			}
			response, err := mainHandler.Client.ValidateOnlineFeatures(entity, token)
			if err != nil {
				return Response{
					Error: "failed to validate feature exists: " + err.Error(),
					Data:  Message{Message: emptyResponse},
				}, err
			}
			for _, fg := range featureComponent.FSRequest.FeatureGroups {
				featureMap := make(map[string]bool)
				for _, feature := range fg.Features {
					if _, exists := featureMap[feature]; exists {
						return Response{
							Error: "feature " + feature + " is duplicated",
							Data:  Message{Message: emptyResponse},
						}, errors.New("feature " + feature + " is duplicated")
					}
					featureMap[feature] = true
					if !mainHandler.ValidateFeatureExists(fg.Label+COLON_DELIMITER+feature, response) {
						return Response{
							Error: "feature \"" + entity + COLON_DELIMITER + fg.Label + COLON_DELIMITER + feature + "\" does not exist",
							Data:  Message{Message: emptyResponse},
						}, errors.New("feature \"" + entity + COLON_DELIMITER + fg.Label + COLON_DELIMITER + feature + "\" does not exist")
					}
				}
			}
		}

		for _, predatorComponent := range config.ComponentConfig.PredatorComponents {
			outputMap := make(map[string]bool)
			for _, output := range predatorComponent.Outputs {
				for _, modelScore := range output.ModelScores {
					if _, exists := outputMap[modelScore]; exists {
						return Response{
							Error: "model score " + modelScore + " is duplicated for component " + predatorComponent.Component,
							Data:  Message{Message: emptyResponse},
						}, errors.New("model score " + modelScore + " is duplicated for component " + predatorComponent.Component)
					}
					outputMap[modelScore] = true
				}
			}
		}
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{Message: "Request validated successfully"},
	}, nil
}

func (m *InferFlow) ValidateOnboardRequest(request OnboardPayload) (Response, error) {
	outputs := mapset.NewSet[string]()
	deployableConfig, err := m.ServiceDeployableConfigRepo.GetById(request.ConfigMapping.DeployableID)
	if err != nil {
		return Response{
			Error: "Failed to fetch deployable config for the request",
			Data:  Message{Message: emptyResponse},
		}, errors.New("failed to fetch deployable config for the request")
	}
	permissibleEndpoints := m.EtcdConfig.GetConfiguredEndpoints(deployableConfig.Name)
	for _, ranker := range request.Rankers {
		if len(ranker.EntityID) == 0 {
			return Response{
				Error: "Entity ID is not set for model: " + ranker.ModelName,
				Data:  Message{Message: emptyResponse},
			}, errors.New("Entity ID is not set for model: " + ranker.ModelName)
		}
		if !permissibleEndpoints.Contains(ranker.EndPoint) {
			errorMsg := fmt.Sprintf(
				"invalid endpoint: %s chosen for service deployable: %s for model: %s",
				ranker.EndPoint, deployableConfig.Name, ranker.ModelName,
			)
			return Response{
				Error: errorMsg,
				Data:  Message{Message: emptyResponse},
			}, errors.New(errorMsg)
		}
		for _, output := range ranker.Outputs {
			if len(output.ModelScores) != len(output.ModelScoresDims) {
				return Response{
					Error: "model scores and model scores dims are not equal for model: " + ranker.ModelName,
					Data:  Message{Message: emptyResponse},
				}, errors.New("model scores and model scores dims are not equal for model: " + ranker.ModelName)
			}
			for _, modelScore := range output.ModelScores {
				if outputs.Contains(modelScore) {
					return Response{
						Error: "duplicate model scores: " + modelScore + " for model: " + ranker.ModelName,
						Data:  Message{Message: emptyResponse},
					}, errors.New("duplicate model scores: " + modelScore + " for model: " + ranker.ModelName)
				}
				outputs.Add(modelScore)
			}
		}
	}

	for _, reRanker := range request.ReRankers {
		if len(reRanker.EntityID) == 0 {
			return Response{
				Error: "Entity ID is not set for re ranker: " + reRanker.Score,
				Data:  Message{Message: emptyResponse},
			}, errors.New("Entity ID is not set for re ranker: " + reRanker.Score)
		}
		for _, value := range reRanker.EqVariables {
			parts := strings.Split(value, PIPE_DELIMITER)
			if len(parts) != 2 {
				return Response{
					Error: "invalid eq variable: " + value,
					Data:  Message{Message: emptyResponse},
				}, errors.New("invalid eq variable: " + value)
			}
			if parts[1] == "" {
				return Response{
					Error: "invalid eq variable: " + value,
					Data:  Message{Message: emptyResponse},
				}, errors.New("invalid eq variable: " + value)
			}
		}
		if outputs.Contains(reRanker.Score) {
			return Response{
				Error: "duplicate score: " + reRanker.Score + " for reRanker: " + reRanker.Score,
				Data:  Message{Message: emptyResponse},
			}, errors.New("duplicate score: " + reRanker.Score + " for reRanker: " + reRanker.Score)
		}
		outputs.Add(reRanker.Score)
	}

	// Validate MODEL_FEATURE list
	for _, ranker := range request.Rankers {
		for _, input := range ranker.Inputs {
			for _, feature := range input.Features {
				featureParts := strings.Split(feature, PIPE_DELIMITER)
				if len(featureParts) != 2 {
					return Response{
						Error: "invalid feature: " + feature + " in input features of ranker: " + ranker.ModelName,
						Data:  Message{Message: emptyResponse},
					}, errors.New("invalid feature: " + feature + " in input features of ranker: " + ranker.ModelName)
				}
				if strings.Contains(featureParts[0], MODEL_FEATURE) {
					if !outputs.Contains(featureParts[1]) {
						return Response{
							Error: "model score " + featureParts[1] + " is not found in other model scores of ranker: " + ranker.ModelName,
							Data:  Message{Message: emptyResponse},
						}, errors.New("model score " + featureParts[1] + " is not found in other model scores of ranker: " + ranker.ModelName)
					}
				}
			}
		}
	}

	for _, reRanker := range request.ReRankers {
		for _, feature := range reRanker.EqVariables {
			featureParts := strings.Split(feature, PIPE_DELIMITER)
			if len(featureParts) != 2 {
				return Response{
					Error: "invalid feature: " + feature,
					Data:  Message{Message: emptyResponse},
				}, errors.New("invalid feature: " + feature)
			}
			if strings.Contains(featureParts[0], MODEL_FEATURE) {
				if !outputs.Contains(featureParts[1]) {
					return Response{
						Error: "model score " + featureParts[1] + " is not found in other model scores of re ranker: " + strconv.Itoa(reRanker.EqID),
						Data:  Message{Message: emptyResponse},
					}, errors.New("model score " + featureParts[1] + " is not found in other model scores of re ranker: " + strconv.Itoa(reRanker.EqID))
				}
			}
		}
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{Message: "Request validated successfully"},
	}, nil
}
