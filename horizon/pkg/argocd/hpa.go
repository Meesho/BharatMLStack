package argocd

import (
	"github.com/Meesho/BharatMLStack/horizon/pkg/kubernetes"
	"github.com/rs/zerolog/log"
)

func GetHPAProperties(applicationName string, workingEnv string) (kubernetes.HPA, error) {
	log.Info().Str("workingEnv", workingEnv).Str("applicationName", applicationName).Msg("In GetHPAProperties")
	argocdResource, err := GetArgoCDResource(
		"HorizontalPodAutoscaler",
		applicationName,
		IsCanary(applicationName, workingEnv),
	)
	if err != nil {
		log.Error().Err(err).Str("workingEnv", workingEnv).Str("applicationName", applicationName).Msg("GetHPAProperties: Error after calling GetArgoCDResource")
		return kubernetes.HPA{}, err
	}

	// Fetch resource json from ArgoCD and parse
	hpaJson, err := argocdResource.FetchResourceFromArgoCD(workingEnv)
	if err != nil {
		log.Error().Err(err).Str("workingEnv", workingEnv).Str("applicationName", applicationName).Msg("GetHPAProperties: Error fetching HPA JSON from ArgoCD")
		return kubernetes.HPA{}, err
	}

	// Parse the json and return
	policy, err := kubernetes.ParseHPAJson(hpaJson)
	if err != nil {
		log.Error().Err(err).Str("applicationName", applicationName).Msg("GetHPAProperties: Error parsing HPA JSON")
		return kubernetes.HPA{}, err
	}
	return policy, nil
}

func GetScaledObjectProperties(applicationName string, workingEnv string) (kubernetes.AutoScalingPolicy, error) {
	log.Info().Str("workingEnv", workingEnv).Str("applicationName", applicationName).Msg("In GetScaledObjectProperties")
	argocdResource, err := GetArgoCDResource(
		"ScaledObject",
		applicationName,
		IsCanary(applicationName, workingEnv),
	)
	if err != nil {
		log.Error().Err(err).Str("workingEnv", workingEnv).Str("applicationName", applicationName).Msg("GetScaledObjectProperties: Error after calling GetArgoCDResource")
		return kubernetes.AutoScalingPolicy{}, err
	}

	// Fetch resource json from ArgoCD and parse
	scaledObjectJson, err := argocdResource.FetchResourceFromArgoCD(workingEnv)
	if err != nil {
		log.Error().Err(err).Str("workingEnv", workingEnv).Str("applicationName", applicationName).Msg("GetScaledObjectProperties: Error fetching ScaledObject JSON from ArgoCD")
		return kubernetes.AutoScalingPolicy{}, err
	}

	// Parse the json and return
	policy, err := kubernetes.ParseScaledObjectJson(scaledObjectJson)
	if err != nil {
		log.Error().Err(err).Str("applicationName", applicationName).Msg("GetScaledObjectProperties: Error parsing ScaledObject JSON")
		return kubernetes.AutoScalingPolicy{}, err
	}
	return policy, nil
}
