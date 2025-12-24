package argocd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

const (
	COMMON_BU      = "common"
	SHARED_BU      = "shared"
	GCP_INT_ENV    = "gcp_int"
	GCP_PRD_ENV    = "gcp_prd"
	GCP_ENV_PREFIX = "gcp_"
	DEV_ENV        = "dev"
	DR_ENV         = "dr"
	PRD_ENV        = "prd"
)

func argocdWorkingBUMagic(api string, workingEnv string, workingBU string) (string, error) {
	// Default case
	log.Info().Str("API", api).Str("WorkingEnv", workingEnv).Str("WorkingBU", workingBU).Msg("Entered argocdWorkingBUMagic")
	if workingBU == COMMON_BU {
		log.Info().Str("WorkingEnv", workingEnv).Str("workingBU", workingBU).Msg("Working BU is common")
		return workingEnv, nil
	}

	// If workingEnv is GCP int, return `workingEnv_shared`
	if workingEnv == GCP_INT_ENV {
		log.Info().Str("WorkingEnv", workingEnv).Msg("workingEnv is GCP int Returning combined workingEnv_sharedBU")
		return fmt.Sprintf("%s_%s", workingEnv, SHARED_BU), nil
	}

	// If workingBU is not empty
	// If workingEnv is not GCP prd/int, return as it is (no bu)
	if workingBU != "" && workingEnv != GCP_PRD_ENV && workingEnv != GCP_INT_ENV {
		log.Info().Str("WorkingEnv", workingEnv).Msg("workingEnv is not GCP prd/int Returning workingEnv as is")
		return workingEnv, nil
	}

	// If workingBU is not empty
	// return `workingEnv_workingBU`
	if workingBU != "" {
		log.Info().Str("WorkingEnvWorkingBU", fmt.Sprintf("%s_%s", workingEnv, workingBU)).Msg("Returning combined workingEnv_workingBU")
		return fmt.Sprintf("%s_%s", workingEnv, workingBU), nil
	}

	// If workingBU is empty and
	// If workingEnv is not GCP prd/int, return as it is (no bu)
	if workingEnv != GCP_PRD_ENV && workingEnv != GCP_INT_ENV {
		log.Info().Str("WorkingEnv", workingEnv).Msg("workingBU is empty and workingEnv is not GCP prd/int Returning workingEnv as is")
		return workingEnv, nil
	}

	// If workingBU is empty and
	// If workingEnv starts with GCP prefix.
	// We have several cases to handle and we try to fetch the workingBU from the database

	// Extract the namespace/name from the application
	applicationName := ExtractApplicationNameFromAPI(api)
	// Failed case,
	if applicationName == "" {
		log.Error().Str("API", api).Msg("argocdWorkingBUMagic: Failed to extract application name from database")
		return "", errors.New("Failed to fetch application name for " + api + " to predict the workingBU.")
	}

	// For now, return workingEnv if we can't determine BU
	// TODO: Implement database lookup if needed
	log.Warn().Str("ApplicationName", applicationName).Msg("argocdWorkingBUMagic: BU lookup not implemented, using workingEnv")
	return workingEnv, nil
}

func ExtractApplicationNameFromAPI(api string) string {
	log.Info().Str("API", api).Msg("Entered ExtractApplicationNameFromAPI")
	// trim any beginnging /
	api = strings.TrimLeft(api, "/")

	// Split the api at "/"
	segments := strings.Split(api, "/")

	// We cannot find the name if the segments are not enough
	if len(segments) <= 1 {
		log.Warn().Str("API", api).Msg("ExtractApplicationNameFromAPI: API does not contain enough segments")
		return ""
	}

	// If first segments is `/applications` next would be the name
	secondSegment := segments[1]

	// Second segment might have query parameters
	applicationName := strings.Split(secondSegment, "?")[0]
	log.Info().Str("ApplicationName", applicationName).Msg("Extracted application name")

	return applicationName
}

// GetArgocdApplicationNameFromEnv constructs ArgoCD application name from appName and workingEnv
// Uses workingEnv exactly as provided - no transformation or suffix extraction
// Examples:
//   - workingEnv="gcp_stg", appName="test-app" -> "gcp_stg-test-app"
//   - workingEnv="gcp_int", appName="test-app" -> "gcp_int-test-app"
//   - workingEnv="stg", appName="test-app" -> "stg-test-app"
func GetArgocdApplicationNameFromEnv(applicationNameWithoutPrefix string, workingEnv string) string {
	log.Info().Str("ApplicationNameWithoutPrefix", applicationNameWithoutPrefix).Str("WorkingEnv", workingEnv).Msg("Entered GetArgocdApplicationNameFromEnv")
	
	// Use workingEnv exactly as provided - no parsing or transformation
	// Construct ArgoCD application name: {workingEnv}-{appName}
	argocdAppName := workingEnv + "-" + applicationNameWithoutPrefix
	
	log.Info().
		Str("applicationNameWithoutPrefix", applicationNameWithoutPrefix).
		Str("workingEnv", workingEnv).
		Str("argocdAppName", argocdAppName).
		Msg("GetArgocdApplicationNameFromEnv: Constructed ArgoCD application name")
	
	return argocdAppName
}

// Returns the namespace for argocd app resources
// Note, this does not return service namespace, instead the namespace
// where argo app yamls are saved on the admin cluster
func GetArgocdNamespace(workingEnv string, workingBU string) string {
	log.Info().Str("workingEnv", workingEnv).Str("workingBU", workingBU).Msg("Entered GetArgocdNamespace")
	// If not GCP, return argocd
	if !strings.HasPrefix(workingEnv, GCP_ENV_PREFIX) {
		log.Info().Msg("Returning 'argocd' for non-GCP environment")
		return "argocd"
	}

	// If it is GCP
	// prd/int share same namespace
	// stg/dev share same namespace
	// For now, use a simple mapping
	var argocdNameSpaceSuffix string
	if workingBU == "" {
		argocdNameSpaceSuffix = "prd"
	} else if strings.Contains(workingEnv, "int") {
		argocdNameSpaceSuffix = SHARED_BU + "-int"
	} else {
		argocdNameSpaceSuffix = workingBU + "-prd"
	}

	return "argocd-" + argocdNameSpaceSuffix
}
