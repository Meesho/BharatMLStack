package argocd

import (
	"fmt"

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

// SetDeployableReplicas sets min and max replicas for a deployable by patching HPA or KEDA ScaledObject.
// applicationName is the ArgoCD application name (e.g. from GetArgocdApplicationNameFromEnv).
func SetDeployableReplicas(applicationName, workingEnv string, minReplica, maxReplica int) error {
	if minReplica < 0 || maxReplica < 0 {
		return fmt.Errorf("replicas must be non-negative: minReplica=%d, maxReplica=%d", minReplica, maxReplica)
	}
	if minReplica > maxReplica {
		return fmt.Errorf("minReplica (%d) must not exceed maxReplica (%d)", minReplica, maxReplica)
	}

	log.Info().
		Str("applicationName", applicationName).
		Str("workingEnv", workingEnv).
		Int("minReplica", minReplica).
		Int("maxReplica", maxReplica).
		Msg("SetDeployableReplicas: patching scaling resource")

	isCanary := IsCanary(applicationName, workingEnv)

	// Try HPA first
	var hpaErr error
	hpaResource, err := GetArgoCDResource("HorizontalPodAutoscaler", applicationName, isCanary)
	if err == nil {
		_, patchErr := hpaResource.PatchArgoCDResource(map[string]interface{}{
			"spec": map[string]interface{}{
				"minReplicas": minReplica,
				"maxReplicas": maxReplica,
			},
		}, workingEnv)
		if patchErr == nil {
			log.Info().Str("applicationName", applicationName).Msg("SetDeployableReplicas: successfully patched HPA")
			return nil
		}
		log.Error().Err(patchErr).Str("applicationName", applicationName).Msg("SetDeployableReplicas: failed to patch HPA, trying ScaledObject")
		hpaErr = patchErr
	} else {
		hpaErr = err
	}

	// HPA not found or patch failed; try KEDA ScaledObject
	scaledObjResource, scaledErr := GetArgoCDResource("ScaledObject", applicationName, isCanary)
	if scaledErr != nil {
		return fmt.Errorf("neither HPA nor ScaledObject found for application %s (HPA: %v; ScaledObject: %v)", applicationName, hpaErr, scaledErr)
	}
	_, patchErr := scaledObjResource.PatchArgoCDResource(map[string]interface{}{
		"spec": map[string]interface{}{
			"minReplicaCount": minReplica,
			"maxReplicaCount": maxReplica,
		},
	}, workingEnv)
	if patchErr != nil {
		log.Error().Err(patchErr).Str("applicationName", applicationName).Msg("SetDeployableReplicas: failed to patch ScaledObject")
		return fmt.Errorf("neither HPA nor ScaledObject succeeded for application %s (HPA: %v; ScaledObject patch: %w)", applicationName, hpaErr, patchErr)
	}
	log.Info().Str("applicationName", applicationName).Msg("SetDeployableReplicas: successfully patched ScaledObject")
	return nil
}
