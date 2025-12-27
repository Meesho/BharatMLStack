package activities

import (
	"context"
	"fmt"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/pkg/argocd"
	"github.com/Meesho/BharatMLStack/horizon/pkg/gcp"
	"github.com/Meesho/BharatMLStack/horizon/pkg/github"
	"github.com/Meesho/BharatMLStack/horizon/pkg/templates"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

// UpdateDeploymentYaml updates the deployment YAML in GitHub helm chart repo
// This corresponds to RingMaster's UpdateDeploymentYamlActivity
func UpdateDeploymentYaml(payload map[string]interface{}, workingEnv string) error {
	appName := getString(payload, "appName")
	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Msg("UpdateDeploymentYaml activity started")

	// Enable backward compatibility
	EnableDeploymentBackwardCompatibility(payload)

	// Set appConfigEnabled only for Java or GoLang app types
	appType := getString(payload, "appType")
	javaGoTypes := []string{"java", "maven", "go", "golang", "Java", "GoLang"}
	if slices.Contains(javaGoTypes, appType) {
		payload["appConfigEnabled"] = true
	}

	// Use configured repository and branch (mandatory from .env file)
	repo := GetRepository()
	if repo == "" {
		return fmt.Errorf("REPOSITORY_NAME is required - set it in .env file")
	}
	branch := GetBranch()
	if branch == "" {
		return fmt.Errorf("BRANCH_NAME is required - set it in .env file")
	}

	// Render and push deployment.yaml
	deploymentFilePath := fmt.Sprintf("deployments/%s.yaml", appName)
	log.Info().
		Str("appName", appName).
		Str("repo", repo).
		Str("branch", branch).
		Str("filePath", deploymentFilePath).
		Str("workingEnv", workingEnv).
		Msg("UpdateDeploymentYaml: Rendering deployment template and pushing to GitHub")

	err := templates.RenderTemplate(payload, "deployment.tmpl", repo, branch, deploymentFilePath)
	if err != nil {
		log.Error().
			Err(err).
			Str("appName", appName).
			Str("repo", repo).
			Str("branch", branch).
			Str("filePath", deploymentFilePath).
			Str("workingEnv", workingEnv).
			Str("templateName", "deployment.tmpl").
			Msg("UpdateDeploymentYaml: Failed to render deployment template or push to GitHub")
		return fmt.Errorf("failed to render deployment template: %w", err)
	}

	log.Info().
		Str("appName", appName).
		Str("repo", repo).
		Str("branch", branch).
		Str("filePath", deploymentFilePath).
		Msg("UpdateDeploymentYaml: Deployment template rendered and pushed to GitHub successfully")

	log.Info().
		Str("appName", appName).
		Str("filePath", deploymentFilePath).
		Msg("UpdateDeploymentYaml activity completed")

	return nil
}

// CreateApplicationYaml creates the ArgoCD application YAML
// Note: This is a simplified implementation. Full implementation would require
// reading deployment.yaml from GitHub and constructing all bindings
func CreateApplicationYaml(payload map[string]interface{}, workingEnv string) error {
	appName := getString(payload, "appName")
	repoName := getString(payload, "repoName")
	bu := getString(payload, "bu")
	team := getString(payload, "team")

	log.Info().
		Str("appName", appName).
		Str("repoName", repoName).
		Str("workingEnv", workingEnv).
		Msg("CreateApplicationYaml activity started")

	// Get environment config (with defaults for unknown environments)
	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Msg("CreateApplicationYaml: Retrieving environment configuration")

	envConfig := github.GetEnvConfig(workingEnv)
	configEnv := envConfig["config_env"]
	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("configEnv", configEnv).
		Msg("CreateApplicationYaml: Environment configuration retrieved successfully")

	// Normalize BU and team (same pattern as RingMaster)
	// RingMaster pattern:
	//   1. bu_norm = bu (save original)
	//   2. team_norm = team (save original)
	//   3. bu = BU_NORMALIZED[bu_norm] (normalize for project field)
	//   4. team = TEAM_NORMALIZED[team_norm] (normalize for project field)
	buNorm := strings.TrimSpace(bu)     // Save original as bu_norm, trim whitespace
	teamNorm := strings.TrimSpace(team) // Save original as team_norm, trim whitespace

	// Validate original values are not empty
	// Use "ml" as fallback when bu/team are absent (maintains backward compatibility)
	if buNorm == "" {
		log.Warn().
			Str("bu", bu).
			Msg("CreateApplicationYaml: BU value is empty, using 'ml' as fallback")
		buNorm = "ml"
	}
	if teamNorm == "" {
		log.Warn().
			Str("team", team).
			Msg("CreateApplicationYaml: Team value is empty, using 'ml' as fallback")
		teamNorm = "ml"
	}

	// Use BU and Team as-is (no normalization for vendor-agnostic support)
	buNormalized := strings.TrimSpace(buNorm)
	if buNormalized == "" {
		log.Warn().
			Str("bu", bu).
			Str("buNorm", buNorm).
			Msg("CreateApplicationYaml: BU is empty, using 'default' as fallback")
		buNormalized = "default" // Vendor-agnostic fallback
	}

	teamNormalized := strings.TrimSpace(teamNorm)
	if teamNormalized == "" {
		log.Warn().
			Str("team", team).
			Str("teamNorm", teamNorm).
			Msg("CreateApplicationYaml: Team is empty, using 'default' as fallback")
		teamNormalized = "default" // Vendor-agnostic fallback
	}
	// Ensure teamNormalized is not empty (critical for YAML rendering)
	teamNormalized = strings.TrimSpace(teamNormalized)
	if teamNormalized == "" {
		log.Warn().
			Str("team", team).
			Str("teamNorm", teamNorm).
			Msg("CreateApplicationYaml: Team normalized value is empty after lookup, using 'ml' as fallback")
		teamNormalized = "ml" // Fallback maintains backward compatibility
	}

	// Final validation: ensure normalized values are never empty (critical for YAML project field)
	if buNormalized == "" {
		log.Error().
			Str("bu", bu).
			Str("buNorm", buNorm).
			Msg("CreateApplicationYaml: CRITICAL - buNormalized is still empty after all checks, forcing 'ml'")
		buNormalized = "ml"
	}
	if teamNormalized == "" {
		log.Error().
			Str("team", team).
			Str("teamNorm", teamNorm).
			Msg("CreateApplicationYaml: CRITICAL - teamNormalized is still empty after all checks, forcing 'ml'")
		teamNormalized = "ml"
	}

	log.Info().
		Str("appName", appName).
		Str("bu", bu).
		Str("buNorm", buNorm).
		Str("buNormalized", buNormalized).
		Str("team", team).
		Str("teamNorm", teamNorm).
		Str("teamNormalized", teamNormalized).
		Msg("CreateApplicationYaml: BU and team normalization completed")

	// Use workingEnv as-is (no normalization)
	// Environment-specific configuration should be provided via .env file

	// Get repository and GitHub owner for template variable substitution
	repo := GetRepository()
	if repo == "" {
		return fmt.Errorf("REPOSITORY_NAME is required - set it in .env file")
	}
	githubOwner := github.GetGitHubOwner()
	if githubOwner == "" {
		return fmt.Errorf("GITHUB_OWNER is required - set it in .env file")
	}

	// Get ArgoCD configuration from environment variables (vendor-agnostic)
	// Uses workingEnv exactly as provided (just uppercased) - no transformation
	argocdNamespace := argocd.GetArgoCDNamespace(workingEnv)
	if argocdNamespace == "" {
		argocdNamespace = "argocd-dev" // Default fallback if not configured
	}
	argocdNamespace = argocd.ProcessArgoCDTemplateVars(argocdNamespace, buNorm, buNormalized, teamNormalized, githubOwner, repo)

	argocdDestinationName := argocd.GetArgoCDDestinationName(workingEnv)
	if argocdDestinationName == "" {
		// Fallback: construct from bu and cluster_env if not configured
		argocdDestinationName = fmt.Sprintf("k8s-%s-%s-ase1", buNorm, envConfig["cluster_env"])
		log.Warn().
			Str("workingEnv", workingEnv).
			Str("fallbackDestinationName", argocdDestinationName).
			Msg("ArgoCD destination name not configured, using constructed fallback")
	} else {
		argocdDestinationName = argocd.ProcessArgoCDTemplateVars(argocdDestinationName, buNorm, buNormalized, teamNormalized, githubOwner, repo)
	}

	// Destination namespace is hardcoded: {env}-{appName}
	// This matches the standard Kubernetes namespace naming convention
	argocdDestinationNamespace := fmt.Sprintf("%s-%s", envConfig["config_env"], appName)

	// Get ArgoCD project as hardcoded value (not a pattern)
	argocdProject := argocd.GetArgoCDProject(workingEnv)
	if argocdProject == "" {
		argocdProject = "default" // Fallback if not configured
		log.Warn().
			Str("workingEnv", workingEnv).
			Str("fallbackProject", argocdProject).
			Msg("ArgoCD project not configured, using default")
	}

	// Helm chart path is configurable via environment variables
	// Pattern: {WORKING_ENV}_ARGOCD_HELMCHART_PATH or ARGOCD_HELMCHART_PATH
	// Defaults to "values_v2" if not configured
	argocdSourcePath := argocd.GetArgoCDHelmChartPath(workingEnv)
	if argocdSourcePath == "" {
		argocdSourcePath = "values_v2" // Standard helm chart path (default fallback)
		log.Info().
			Str("workingEnv", workingEnv).
			Str("helmChartPath", argocdSourcePath).
			Msg("Using default ArgoCD helm chart path: values_v2 (can be configured via {WORKING_ENV}_ARGOCD_HELMCHART_PATH or ARGOCD_HELMCHART_PATH)")
	} else {
		log.Info().
			Str("workingEnv", workingEnv).
			Str("helmChartPath", argocdSourcePath).
			Msg("Using ArgoCD helm chart path from environment variable")
	}

	// Source Repo URL: Automatically constructed from REPOSITORY_NAME and GITHUB_OWNER
	// This is derived from deployable onboarding configuration
	argocdSourceRepoURL := fmt.Sprintf("https://github.com/%s/%s.git", githubOwner, repo)
	log.Debug().
		Str("workingEnv", workingEnv).
		Str("sourceRepoURL", argocdSourceRepoURL).
		Str("githubOwner", githubOwner).
		Str("repo", repo).
		Msg("Using ArgoCD source repo URL derived from REPOSITORY_NAME and GITHUB_OWNER")

	// Source Target Revision: Automatically uses BRANCH_NAME from deployable onboarding
	argocdSourceTargetRevision := GetBranch()
	if argocdSourceTargetRevision == "" {
		// Fallback if BRANCH_NAME not set (should not happen in normal flow)
		argocdSourceTargetRevision = "main"
		log.Warn().
			Str("workingEnv", workingEnv).
			Str("fallbackTargetRevision", argocdSourceTargetRevision).
			Msg("BRANCH_NAME not set, using fallback for ArgoCD source target revision")
	}
	log.Debug().
		Str("workingEnv", workingEnv).
		Str("targetRevision", argocdSourceTargetRevision).
		Msg("Using ArgoCD source target revision from BRANCH_NAME")

	// Parse sync policy options (comma-separated string)
	argocdSyncPolicyOptionsStr := argocd.GetArgoCDSyncPolicyOptions(workingEnv)
	// Split comma-separated options and trim whitespace
	argocdSyncPolicyOptions := []string{}
	for _, opt := range strings.Split(argocdSyncPolicyOptionsStr, ",") {
		opt = strings.TrimSpace(opt)
		if opt != "" {
			argocdSyncPolicyOptions = append(argocdSyncPolicyOptions, opt)
		}
	}
	if len(argocdSyncPolicyOptions) == 0 {
		argocdSyncPolicyOptions = []string{"CreateNamespace=true"} // Ensure at least one option
	}

	// Prepare bindings for argoapp.tmpl
	// All ArgoCD-specific values now come from environment configuration
	bindings := map[string]interface{}{
		"app_name":       appName,
		"env_ns":         envConfig["config_env"],
		"bu_norm":        buNorm,         // Original value for labels
		"team_norm":      teamNorm,       // Original value for labels
		"bu":             buNormalized,   // Normalized value (guaranteed non-empty)
		"team":           teamNormalized, // Normalized value (guaranteed non-empty)
		"primaryowner":   getEmailPrefix(getString(payload, "primaryOwner")),
		"secondaryowner": getEmailPrefix(getString(payload, "secondaryOwner")),
		"priority_v2":    getString(payload, "priorityV2"),
		"zone":           "ase1", // Default zone, can be made configurable
		// ArgoCD configuration from environment config
		"argocd_namespace":             argocdNamespace,
		"destination_namespace":        argocdDestinationNamespace,
		"destination_name":             argocdDestinationName,
		"argocd_project":               argocdProject,
		"argocd_source_path":           argocdSourcePath,
		"argocd_source_repoURL":        argocdSourceRepoURL,
		"argocd_source_targetRevision": argocdSourceTargetRevision,
		"argocd_sync_policy_options":   argocdSyncPolicyOptions,
		// New environment-based path structure: {workingEnv}/deployables/{appName}/values.yaml
		// Relative path from applications directory to values.yaml
		"values_files": []string{fmt.Sprintf("../%s/deployables/%s/values.yaml", workingEnv, appName)},
	}

	// Determine file path using new environment-based structure
	// New structure: {workingEnv}/applications/{appName}.yaml
	// Uses workingEnv as-is (no normalization) - environment-specific config comes from .env file
	argoFilePath := github.GetArgoApplicationPath(workingEnv, appName)
	if argoFilePath == "" {
		log.Error().
			Str("appName", appName).
			Str("workingEnv", workingEnv).
			Msg("CreateApplicationYaml: Failed to construct ArgoCD application path")
		return fmt.Errorf("failed to construct ArgoCD application path for env %s", workingEnv)
	}

	// Get branch for pushing the file
	branch := GetBranch()
	if branch == "" {
		return fmt.Errorf("BRANCH_NAME is required - set it in .env file")
	}

	log.Info().
		Str("appName", appName).
		Str("repo", repo).
		Str("argocdSourceRepoURL", argocdSourceRepoURL).
		Str("githubOwner", githubOwner).
		Str("argocdNamespace", argocdNamespace).
		Str("argocdProject", argocdProject).
		Str("argocdDestinationName", argocdDestinationName).
		Msg("CreateApplicationYaml: Constructed ArgoCD configuration from environment config")

	// Render and push argoapp.yaml
	log.Info().
		Str("appName", appName).
		Str("repo", repo).
		Str("branch", branch).
		Str("filePath", argoFilePath).
		Str("workingEnv", workingEnv).
		Msg("CreateApplicationYaml: Rendering ArgoCD application template and pushing to GitHub")

	err := templates.RenderTemplate(
		bindings,
		"argoapp.tmpl",
		repo,
		branch,
		argoFilePath,
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("appName", appName).
			Str("repo", repo).
			Str("branch", branch).
			Str("filePath", argoFilePath).
			Str("workingEnv", workingEnv).
			Str("templateName", "argoapp.tmpl").
			Msg("CreateApplicationYaml: Failed to render ArgoCD application template or push to GitHub")
		return fmt.Errorf("failed to render ArgoCD application template: %w", err)
	}

	log.Info().
		Str("appName", appName).
		Str("repo", repo).
		Str("branch", branch).
		Str("filePath", argoFilePath).
		Msg("CreateApplicationYaml: ArgoCD application template rendered and pushed to GitHub successfully")

	log.Info().
		Str("appName", appName).
		Str("filePath", argoFilePath).
		Msg("CreateApplicationYaml activity completed")

	return nil
}

// getEmailPrefix extracts the username part from an email address
// Returns empty string if input is empty or invalid
func getEmailPrefix(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return ""
}

// UpdateValuesProperties updates the values.properties.yaml file
// This corresponds to RingMaster's UpdateValuesPropertiesYamlActivity
func UpdateValuesProperties(payload map[string]interface{}, workingEnv string) error {
	appName := getString(payload, "appName")
	bu := getString(payload, "bu")
	team := getString(payload, "team")

	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Msg("UpdateValuesProperties activity started")

	// BU and team are now optional (from labels section in config.yaml)
	// If not provided, use "ml" as fallback (maintains backward compatibility)
	// Normalize BU and team if provided
	buNorm := bu
	if bu != "" {
		// Use BU as-is (no normalization)
		buNorm = bu
	} else {
		buNorm = "ml" // Fallback maintains backward compatibility
	}

	teamNorm := team
	if team != "" {
		// Use Team as-is (no normalization)
		teamNorm = team
	} else {
		teamNorm = "ml" // Fallback maintains backward compatibility
	}

	payload["bu_norm"] = buNorm
	payload["team_norm"] = teamNorm

	// Set host
	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Msg("UpdateValuesProperties: Retrieving environment configuration for host setup")

	envConfig := github.GetEnvConfig(workingEnv)
	configEnv := envConfig["config_env"]
	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Str("configEnv", configEnv).
		Msg("UpdateValuesProperties: Environment configuration retrieved successfully")

	// Generate host using domain from payload (set during onboarding from service config)
	// Format: <appname>.<domain>
	domain := getString(payload, "domain")
	if domain == "" {
		log.Warn().
			Str("appName", appName).
			Str("workingEnv", workingEnv).
			Msg("UpdateValuesProperties: Domain not found in payload, host generation may fail. Ensure domain is set in service config.yaml")
		// Fallback: construct host without domain (will likely fail, but allows error to surface)
		payload["host"] = appName
	} else {
		payload["host"] = fmt.Sprintf("%s.%s", appName, domain)
	}

	// Set ingress class
	ingressClass := getString(payload, "ingress_class")
	priorityV2 := getString(payload, "priorityV2")
	payload["ingress_class"] = GetIngressClass(ingressClass, priorityV2, workingEnv)

	// Check if gRPC service
	if IsServiceBeingOnboardedGrpc(payload) {
		payload["isGrpc"] = "true"
	}

	payload["workingEnv"] = workingEnv

	// Enable backward compatibility
	EnableValuesBackwardCompatibility(payload)

	// Convert as_min/as_max or min_replica/max_replica to minReplica/maxReplica for template
	// This matches RingMaster's utility.go logic
	if _, ok := payload["minReplica"]; !ok {
		if minReplica, ok := payload["min_replica"]; ok {
			payload["minReplica"] = minReplica
		} else if asMin, ok := payload["as_min"]; ok {
			payload["minReplica"] = asMin
		} else {
			payload["minReplica"] = "1" // Default
		}
	}

	if _, ok := payload["maxReplica"]; !ok {
		if maxReplica, ok := payload["max_replica"]; ok {
			payload["maxReplica"] = maxReplica
		} else if asMax, ok := payload["as_max"]; ok {
			payload["maxReplica"] = asMax
		} else {
			payload["maxReplica"] = "5" // Default
		}
	}

	log.Info().
		Str("appName", appName).
		Interface("minReplica", payload["minReplica"]).
		Interface("maxReplica", payload["maxReplica"]).
		Interface("as_min", payload["as_min"]).
		Interface("as_max", payload["as_max"]).
		Msg("UpdateValuesProperties: Converted replica values for template")

	// Get values properties file path
	valuesPropertiesFilePath := GetValuePropertyPath(payload, workingEnv)

	// Use configured repository and branch (mandatory from .env file)
	// All files will be pushed to the configured repository and branch
	repo := GetRepository()
	if repo == "" {
		return fmt.Errorf("REPOSITORY_NAME is required - set it in .env file")
	}
	branch := GetBranch()
	if branch == "" {
		return fmt.Errorf("BRANCH_NAME is required - set it in .env file")
	}

	// Render and push values_properties.yaml
	log.Info().
		Str("appName", appName).
		Str("repo", repo).
		Str("branch", branch).
		Str("filePath", valuesPropertiesFilePath).
		Str("workingEnv", workingEnv).
		Msg("UpdateValuesProperties: Rendering values properties template and pushing to GitHub")

	err := templates.RenderTemplate(
		payload,
		"values_properties.tmpl",
		repo,
		branch,
		valuesPropertiesFilePath,
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("appName", appName).
			Str("repo", repo).
			Str("branch", branch).
			Str("filePath", valuesPropertiesFilePath).
			Str("workingEnv", workingEnv).
			Str("templateName", "values_properties.tmpl").
			Msg("UpdateValuesProperties: Failed to render values properties template or push to GitHub")
		return fmt.Errorf("failed to render values properties template: %w", err)
	}

	log.Info().
		Str("appName", appName).
		Str("repo", repo).
		Str("branch", branch).
		Str("filePath", valuesPropertiesFilePath).
		Msg("UpdateValuesProperties: Values properties template rendered and pushed to GitHub successfully")

	log.Info().
		Str("appName", appName).
		Str("filePath", valuesPropertiesFilePath).
		Msg("UpdateValuesProperties activity completed")

	return nil
}

// CreateValuesYaml creates the values.yaml file by compiling common-values.tmpl with app-type specific template
// This corresponds to RingMaster's UpdateHelmRepo function during deployment
// In RingMaster, values.yaml is created during deployment, but we create it during onboarding for self-reliance
func CreateValuesYaml(payload map[string]interface{}, workingEnv string) error {
	appName := getString(payload, "appName")

	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Msg("CreateValuesYaml activity started")

	// Validate required fields
	if appName == "" {
		return fmt.Errorf("appName is required")
	}

	// Get environment config (with defaults for unknown environments)
	envConfig := github.GetEnvConfig(workingEnv)
	configEnv := envConfig["config_env"]

	// Normalize BU and team (same pattern as CreateApplicationYaml)
	bu := getString(payload, "bu")
	team := getString(payload, "team")
	buNorm := strings.TrimSpace(bu)
	teamNorm := strings.TrimSpace(team)

	// Validate and set defaults if empty
	// Use "ml" as fallback when bu/team are absent (maintains backward compatibility)
	if buNorm == "" {
		log.Warn().Str("bu", bu).Msg("CreateValuesYaml: BU is empty, using 'ml' as fallback")
		buNorm = "ml"
	}
	if teamNorm == "" {
		log.Warn().Str("team", team).Msg("CreateValuesYaml: Team is empty, using 'ml' as fallback")
		teamNorm = "ml"
	}

	// Normalize for template (lookup in normalized maps)
	// Use BU and Team as-is (no normalization for vendor-agnostic support)
	buNormalized := buNorm
	teamNormalized := teamNorm

	// Ensure normalized values are never empty
	// Use "ml" as fallback when normalized values are empty (maintains backward compatibility)
	if buNormalized == "" {
		buNormalized = "ml"
	}
	if teamNormalized == "" {
		teamNormalized = "ml"
	}

	// Horizon only supports triton deployables
	// triton-values.tmpl includes {{ template "common-values.tmpl" . }} at the top
	// So it automatically compiles with common-values.tmpl
	templateName := GetValuesTemplateName("")
	log.Info().
		Str("appName", appName).
		Str("templateName", templateName).
		Msg("CreateValuesYaml: Using triton-values.tmpl (includes common-values.tmpl)")

	// Get values.yaml file path
	valuesYamlPath := GetValuesYamlPath(payload, workingEnv)
	if valuesYamlPath == "" {
		log.Error().
			Str("appName", appName).
			Str("workingEnv", workingEnv).
			Msg("CreateValuesYaml: Failed to construct values.yaml path - base path is empty")
		return fmt.Errorf("failed to construct values.yaml path: base path is empty")
	}

	// Use configured repository and branch (mandatory from .env file)
	// All files will be pushed to the configured repository and branch
	repo := GetRepository()
	if repo == "" {
		return fmt.Errorf("REPOSITORY_NAME is required - set it in .env file")
	}
	branch := GetBranch()
	if branch == "" {
		return fmt.Errorf("BRANCH_NAME is required - set it in .env file")
	}

	// Prepare payload for template rendering
	// Add all required fields that templates expect (matches RingMaster's UpdateHelmRepo pattern)
	// triton_repository and init_container_image must be set in config.yaml
	// These are required fields (vendor-agnostic)
	tritonRepo := getString(payload, "triton_repository")
	if tritonRepo == "" {
		return fmt.Errorf("triton_repository is required in config.yaml for service %s in environment %s", appName, workingEnv)
	}
	log.Info().
		Str("appName", appName).
		Str("triton_repository", tritonRepo).
		Msg("CreateValuesYaml: Using triton_repository from config.yaml")

	// init_container_image is required if gcs_triton_path is enabled
	// initContainerImage := getString(payload, "init_container_image")
	gcsTritonPath := getString(payload, "gcs_triton_path")
	if gcsTritonPath != "" {
		log.Info().
			Str("appName", appName).
			Str("gcs_triton_path", gcsTritonPath).
			Msg("CreateValuesYaml: Using gcs_triton_path from config.yaml (gcs_triton_path enabled)")
	}
	payload["environment_norm"] = envConfig["env_norm"]
	payload["env_ns"] = configEnv
	payload["vault_env"] = configEnv
	payload["app_name"] = appName
	// Set primary_port from appPort (required for targetPort in service)
	// Special handling: If grpc is enabled, targetPort should always be 8001
	appPort := getString(payload, "appPort")
	serviceTypeStr := getString(payload, "service_type")

	// Check if grpc is enabled (either via service_type containing "grpc" or service_type_grpc=true)
	// service_type can be comma-separated like "httpstateless,grpc"
	isGrpc := false
	if serviceTypeStr != "" {
		// Check if "grpc" is present in the service_type string (handles comma-separated values)
		serviceTypes := strings.Split(strings.ReplaceAll(serviceTypeStr, " ", ""), ",")
		for _, st := range serviceTypes {
			if strings.EqualFold(st, "grpc") {
				isGrpc = true
				break
			}
		}
	}
	// Also check service_type_grpc flag (backward compatibility)
	if !isGrpc {
		if grpcFlag, ok := payload["service_type_grpc"]; ok {
			if grpcBool, ok := grpcFlag.(bool); ok && grpcBool {
				isGrpc = true
			} else if grpcStr, ok := grpcFlag.(string); ok && (grpcStr == "true" || grpcStr == "True" || grpcStr == "TRUE") {
				isGrpc = true
			}
		}
	}

	// Set primary_port (and app_port for template compatibility) - this must be set AFTER checking grpc
	// to ensure it's not overridden by earlier code
	if isGrpc {
		// If grpc is enabled, targetPort should always be 8001
		payload["primary_port"] = "8001"
		// Also set app_port to 8001 for template compatibility (some templates check app_port first)
		payload["app_port"] = "8001"
		log.Info().
			Str("appName", appName).
			Str("serviceType", serviceTypeStr).
			Str("targetPort", "8001").
			Msg("CreateValuesYaml: gRPC enabled, setting targetPort to 8001")
	} else {
		// For non-gRPC services, use appPort or default to 8000
		if appPort == "" {
			// Default to 8000 for Triton (matches triton-values.tmpl ports)
			appPort = "8000"
			log.Warn().
				Str("appName", appName).
				Str("defaultPort", appPort).
				Msg("CreateValuesYaml: appPort not set, using default 8000 for Triton")
		}
		payload["primary_port"] = appPort
		payload["app_port"] = appPort // Set app_port for template compatibility
	}
	payload["tag"] = getString(payload, "triton_image_tag")
	if payload["tag"] == "" {
		payload["tag"] = "latest"
	}
	payload["build_team"] = teamNorm
	payload["bu_norm"] = buNorm
	payload["team_norm"] = teamNorm
	payload["bu"] = buNormalized
	payload["team"] = teamNormalized
	payload["priority"] = getString(payload, "priorityV2")
	payload["environment"] = configEnv

	// Set service_type_norm for template (normalized service type)
	// This is used in labels section of the template
	// Reuse serviceTypeStr from above (already declared for grpc check)
	if serviceTypeStr == "" {
		// Default to "httpstateless" if not provided
		serviceTypeStr = "httpstateless"
		payload["service_type"] = serviceTypeStr // Also set in payload for consistency
	}
	payload["service_type_norm"] = serviceTypeStr

	// For triton deployables, dockerBuildVersion should be "triton"
	if _, ok := payload["dockerBuildVersion"]; !ok {
		payload["dockerBuildVersion"] = "triton"
	}

	// nodeSelector should already be in payload from serviceConfig (read from config.yaml)
	// If not present, apply environment-based default (fallback for backward compatibility)
	// nodeSelector is the key (e.g., "dedicated" or "cloud.google.com/compute-class")
	// nodeSelectorValue is the actual value (e.g., "gpu-node")
	if _, ok := payload["nodeSelector"]; !ok {
		if configEnv == "int" {
			payload["nodeSelector"] = "cloud.google.com/compute-class"
		} else {
			payload["nodeSelector"] = "dedicated"
		}
		log.Debug().
			Str("appName", appName).
			Str("workingEnv", workingEnv).
			Str("configEnv", configEnv).
			Str("nodeSelector", payload["nodeSelector"].(string)).
			Msg("CreateValuesYaml: Using default nodeSelector (not found in payload)")
	} else {
		log.Info().
			Str("appName", appName).
			Str("nodeSelector", getString(payload, "nodeSelector")).
			Msg("CreateValuesYaml: Using nodeSelector from config.yaml")
	}

	// Ensure nodeSelectorValue is set (should already be in payload from CreateDeployable)
	if _, ok := payload["nodeSelectorValue"]; !ok {
		payload["nodeSelectorValue"] = "" // Default empty if not provided
		log.Warn().
			Str("appName", appName).
			Msg("CreateValuesYaml: nodeSelectorValue not found in payload, setting to empty string")
	}

	// Enable backward compatibility for deployment
	EnableDeploymentBackwardCompatibility(payload)

	// Probe-related fields should already be in payload from serviceConfig
	// If not present, apply defaults (fallback for backward compatibility)
	if _, ok := payload["liveness_failure_threshold"]; !ok {
		payload["liveness_failure_threshold"] = "5"
		log.Debug().Msg("CreateValuesYaml: Using default liveness_failure_threshold (not found in payload)")
	}
	if _, ok := payload["liveness_period_seconds"]; !ok {
		payload["liveness_period_seconds"] = "10"
		log.Debug().Msg("CreateValuesYaml: Using default liveness_period_seconds (not found in payload)")
	}
	if _, ok := payload["liveness_success_threshold"]; !ok {
		payload["liveness_success_threshold"] = "1"
		log.Debug().Msg("CreateValuesYaml: Using default liveness_success_threshold (not found in payload)")
	}
	if _, ok := payload["liveness_timeout_seconds"]; !ok {
		payload["liveness_timeout_seconds"] = "2"
		log.Debug().Msg("CreateValuesYaml: Using default liveness_timeout_seconds (not found in payload)")
	}
	if _, ok := payload["readiness_failure_threshold"]; !ok {
		payload["readiness_failure_threshold"] = "5"
		log.Debug().Msg("CreateValuesYaml: Using default readiness_failure_threshold (not found in payload)")
	}
	if _, ok := payload["readiness_period_seconds"]; !ok {
		payload["readiness_period_seconds"] = "10"
		log.Debug().Msg("CreateValuesYaml: Using default readiness_period_seconds (not found in payload)")
	}
	if _, ok := payload["readiness_success_threshold"]; !ok {
		payload["readiness_success_threshold"] = "1"
		log.Debug().Msg("CreateValuesYaml: Using default readiness_success_threshold (not found in payload)")
	}
	if _, ok := payload["readiness_timeout_seconds"]; !ok {
		payload["readiness_timeout_seconds"] = "2"
		log.Debug().Msg("CreateValuesYaml: Using default readiness_timeout_seconds (not found in payload)")
	}

	// Triton deployables don't use appConfigEnabled (only Java/Go use it)
	// But we keep the field for template compatibility
	if _, ok := payload["appConfigEnabled"]; !ok {
		payload["appConfigEnabled"] = false
	}

	// Merge podAnnotations: defaults from config.yaml, overrides from payload (higher precedence)
	// Priority: 1. Request payload (pod_annotations), 2. Config.yaml defaults (pod_annotations_from_config)
	// Start with defaults from config.yaml (if present) - these are the base defaults
	mergedPodAnnotations := make(map[string]string)
	if configAnnotations, ok := payload["pod_annotations_from_config"]; ok {
		log.Info().
			Str("appName", appName).
			Interface("configAnnotationsType", fmt.Sprintf("%T", configAnnotations)).
			Interface("configAnnotations", configAnnotations).
			Msg("CreateValuesYaml: Found pod_annotations_from_config in payload, processing...")

		if configMap, ok := configAnnotations.(map[string]string); ok {
			for k, v := range configMap {
				mergedPodAnnotations[k] = v
			}
			log.Info().
				Str("appName", appName).
				Int("configAnnotationCount", len(mergedPodAnnotations)).
				Interface("annotations", mergedPodAnnotations).
				Msg("CreateValuesYaml: Loaded podAnnotations from config.yaml (map[string]string)")
		} else if configMap, ok := configAnnotations.(map[string]interface{}); ok {
			// Handle map[string]interface{} from YAML parsing
			for k, v := range configMap {
				if strVal, ok := v.(string); ok {
					mergedPodAnnotations[k] = strVal
				} else {
					log.Warn().
						Str("appName", appName).
						Str("key", k).
						Interface("value", v).
						Str("valueType", fmt.Sprintf("%T", v)).
						Msg("CreateValuesYaml: Skipping non-string annotation value")
				}
			}
			log.Info().
				Str("appName", appName).
				Int("configAnnotationCount", len(mergedPodAnnotations)).
				Interface("annotations", mergedPodAnnotations).
				Msg("CreateValuesYaml: Loaded podAnnotations from config.yaml (map[string]interface{})")
		} else {
			log.Error().
				Str("appName", appName).
				Interface("configAnnotationsType", fmt.Sprintf("%T", configAnnotations)).
				Interface("configAnnotations", configAnnotations).
				Msg("CreateValuesYaml: pod_annotations_from_config has unexpected type, cannot process")
		}
	} else {
		log.Info().
			Str("appName", appName).
			Msg("CreateValuesYaml: No pod_annotations_from_config found in payload")
	}
	// Override with values from request payload (higher precedence than config.yaml)
	if payloadAnnotations, ok := payload["pod_annotations"]; ok {
		requestAnnotationCount := 0
		if payloadMap, ok := payloadAnnotations.(map[string]string); ok {
			for k, v := range payloadMap {
				mergedPodAnnotations[k] = v
				requestAnnotationCount++
			}
		} else if payloadMap, ok := payloadAnnotations.(map[string]interface{}); ok {
			// Handle map[string]interface{} from JSON parsing
			for k, v := range payloadMap {
				if strVal, ok := v.(string); ok {
					mergedPodAnnotations[k] = strVal
					requestAnnotationCount++
				}
			}
		}
		if requestAnnotationCount > 0 {
			log.Info().
				Str("appName", appName).
				Int("requestAnnotationCount", requestAnnotationCount).
				Msg("CreateValuesYaml: Overriding podAnnotations with request values")
		}
	}
	// Always set merged annotations in payload (even if empty, to ensure template gets the value)
	// This ensures config.yaml defaults are used when request doesn't provide podAnnotations
	payload["pod_annotations"] = mergedPodAnnotations

	// Log final merged annotations for debugging
	if len(mergedPodAnnotations) > 0 {
		log.Info().
			Str("appName", appName).
			Int("finalAnnotationCount", len(mergedPodAnnotations)).
			Interface("annotations", mergedPodAnnotations).
			Msg("CreateValuesYaml: Final merged podAnnotations (will be applied to values.yaml)")

		// Verify the payload is set correctly before template rendering
		if payloadPodAnnotations, ok := payload["pod_annotations"]; ok {
			if payloadMap, ok := payloadPodAnnotations.(map[string]string); ok {
				log.Info().
					Str("appName", appName).
					Int("payloadAnnotationCount", len(payloadMap)).
					Msg("CreateValuesYaml: Verified pod_annotations in payload before template rendering")
			} else {
				log.Error().
					Str("appName", appName).
					Interface("payloadType", fmt.Sprintf("%T", payloadPodAnnotations)).
					Msg("CreateValuesYaml: pod_annotations in payload has wrong type")
			}
		} else {
			log.Error().
				Str("appName", appName).
				Msg("CreateValuesYaml: pod_annotations not found in payload after setting")
		}
	} else {
		log.Info().
			Str("appName", appName).
			Msg("CreateValuesYaml: No podAnnotations to apply (neither config.yaml nor request provided)")
	}

	// Combine CPU/Memory values with units (template expects combined values)
	// CPU values: For "cores" unit, just use the number (Kubernetes format: "2" not "2cores")
	//              For "m" (millicores), combine (e.g., "500m")
	cpuRequest := getString(payload, "cpuRequest")
	cpuRequestUnit := getString(payload, "cpuRequestUnit")
	if cpuRequest != "" && cpuRequestUnit != "" {
		if cpuRequestUnit == "cores" {
			payload["cpu_request"] = cpuRequest // Just the number for cores
		} else {
			payload["cpu_request"] = cpuRequest + cpuRequestUnit // e.g., "500m"
		}
	} else if cpuRequest != "" {
		payload["cpu_request"] = cpuRequest
	}

	cpuLimit := getString(payload, "cpuLimit")
	cpuLimitUnit := getString(payload, "cpuLimitUnit")
	if cpuLimit != "" && cpuLimitUnit != "" {
		if cpuLimitUnit == "cores" {
			payload["cpu_limit"] = cpuLimit // Just the number for cores
		} else {
			payload["cpu_limit"] = cpuLimit + cpuLimitUnit // e.g., "500m"
		}
	} else if cpuLimit != "" {
		payload["cpu_limit"] = cpuLimit
	} else if cpuRequest != "" {
		// Default cpu_limit to cpu_request if not set (matches RingMaster)
		payload["cpu_limit"] = payload["cpu_request"]
	}

	// Memory values: Template adds "i" suffix (e.g., {{ .memory_limit }}i)
	// So if unit is "Gi", pass "4G" not "4Gi" to avoid "4Gii"
	// If unit is "Mi", pass "4M" not "4Mi" to avoid "4Mii"
	memoryRequest := getString(payload, "memoryRequest")
	memoryRequestUnit := getString(payload, "memoryRequestUnit")
	if memoryRequest != "" && memoryRequestUnit != "" {
		// Remove "i" from unit if present (e.g., "Gi" -> "G", "Mi" -> "M")
		// Template will add "i" suffix automatically
		unit := strings.TrimSuffix(memoryRequestUnit, "i")
		payload["memory_request"] = memoryRequest + unit
	} else if memoryRequest != "" {
		payload["memory_request"] = memoryRequest
	}

	memoryLimit := getString(payload, "memoryLimit")
	memoryLimitUnit := getString(payload, "memoryLimitUnit")
	if memoryLimit != "" && memoryLimitUnit != "" {
		// Remove "i" from unit if present (e.g., "Gi" -> "G", "Mi" -> "M")
		// Template will add "i" suffix automatically
		unit := strings.TrimSuffix(memoryLimitUnit, "i")
		payload["memory_limit"] = memoryLimit + unit
	} else if memoryLimit != "" {
		payload["memory_limit"] = memoryLimit
	}

	// GPU values: Only include if not "0" or empty (matches RingMaster pattern)
	// Template uses {{ if .gpu_request }} which treats "0" as truthy, so remove it
	gpuRequest := getString(payload, "gpu_request")
	if gpuRequest != "" && gpuRequest != "0" {
		payload["gpu_request"] = gpuRequest
	} else {
		// Remove from payload so template doesn't include it
		delete(payload, "gpu_request")
	}

	gpuLimit := getString(payload, "gpu_limit")
	if gpuLimit != "" && gpuLimit != "0" {
		payload["gpu_limit"] = gpuLimit
	} else {
		// Remove from payload so template doesn't include it
		delete(payload, "gpu_limit")
	}

	// Autoscaling defaults should already be in payload from serviceConfig (read from config.yaml)
	// If not present, apply defaults (fallback for backward compatibility)
	if _, ok := payload["as_enabled"]; !ok {
		asEnabled := getString(payload, "asEnabled")
		if asEnabled == "" {
			payload["as_enabled"] = "true" // Default enabled
		} else {
			payload["as_enabled"] = asEnabled
		}
	}
	// These should already be in payload from serviceConfig, but ensure they're set
	if _, ok := payload["as_poll"]; !ok {
		payload["as_poll"] = "30"
	}
	if _, ok := payload["as_down_period"]; !ok {
		payload["as_down_period"] = "300"
	}
	if _, ok := payload["as_up_period"]; !ok {
		payload["as_up_period"] = "60"
	}
	if _, ok := payload["as_up_stable_window"]; !ok {
		payload["as_up_stable_window"] = "300"
	}
	if _, ok := payload["as_down_stable_window"]; !ok {
		payload["as_down_stable_window"] = "1800"
	}
	if _, ok := payload["as_trigger_type"]; !ok {
		payload["as_trigger_type"] = "AverageValue"
	}
	if _, ok := payload["as_trigger_metric"]; !ok {
		payload["as_trigger_metric"] = "cpu"
	}
	if _, ok := payload["as_trigger_value"]; !ok {
		asTriggerValue := getString(payload, "cpuThreshold")
		if asTriggerValue == "" {
			payload["as_trigger_value"] = "50"
		} else {
			payload["as_trigger_value"] = asTriggerValue
		}
	}
	if _, ok := payload["as_down_pod_count"]; !ok {
		payload["as_down_pod_count"] = "2"
	}
	if _, ok := payload["as_up_pod_count"]; !ok {
		payload["as_up_pod_count"] = "2"
	}
	if _, ok := payload["as_up_pod_percentage"]; !ok {
		payload["as_up_pod_percentage"] = "10"
	}

	// Deployment strategy defaults should already be in payload from serviceConfig
	// If not present, apply defaults (fallback for backward compatibility)
	if _, ok := payload["maxSurge"]; !ok {
		payload["maxSurge"] = "50"
	}
	if _, ok := payload["terminationGracePeriodSeconds"]; !ok {
		payload["terminationGracePeriodSeconds"] = "300"
	}
	if _, ok := payload["contourResponseTimeout"]; !ok {
		payload["contourResponseTimeout"] = "false"
	}
	if _, ok := payload["podDistributionSkew"]; !ok {
		payload["podDistributionSkew"] = "false"
	}
	if _, ok := payload["enableWebsocket"]; !ok {
		payload["enableWebsocket"] = "false"
	}
	if _, ok := payload["addHeadless"]; !ok {
		payload["addHeadless"] = "false"
	}
	// createContourGateway: Default to "true" (matches values_properties.tmpl which has it hardcoded to true)
	// This can be overridden from config.yaml if needed
	if _, ok := payload["createContourGateway"]; !ok {
		payload["createContourGateway"] = "true"
	}
	if _, ok := payload["pdbMinAvailable"]; !ok {
		payload["pdbMinAvailable"] = ""
	}
	if _, ok := payload["pdbMaxUnavailable"]; !ok {
		payload["pdbMaxUnavailable"] = "10%"
	}

	// Extract email prefixes for primary_owner and secondary_owner (matches RingMaster)
	if primaryOwner, ok := payload["primaryOwner"]; ok && primaryOwner != "" {
		payload["primary_owner"] = getEmailPrefix(getString(payload, "primaryOwner"))
	} else {
		payload["primary_owner"] = ""
	}
	if secondaryOwner, ok := payload["secondaryOwner"]; ok && secondaryOwner != "" {
		payload["secondary_owner"] = getEmailPrefix(getString(payload, "secondaryOwner"))
	} else {
		payload["secondary_owner"] = ""
	}

	// Set priority_v2 (should already be in payload, but ensure it's set)
	if _, ok := payload["priority_v2"]; !ok {
		payload["priority_v2"] = getString(payload, "priorityV2")
		if payload["priority_v2"] == "" {
			payload["priority_v2"] = "cp3" // Default
		}
	}

	// Set repo_name (should already be in payload)
	if _, ok := payload["repo_name"]; !ok {
		payload["repo_name"] = getString(payload, "repoName")
	}

	// Set service_name for template (used to conditionally skip externalSecret for predator)
	// service_name should already be in payload from CreateDeployable
	if _, ok := payload["service_name"]; !ok {
		// Fallback: try to get from serviceName or repo_name
		if serviceName := getString(payload, "serviceName"); serviceName != "" {
			payload["service_name"] = serviceName
		} else if repoName := getString(payload, "repo_name"); repoName != "" {
			payload["service_name"] = repoName
		}
	}

	// commit_id: Removed from template per user request
	// No longer needed in values.yaml

	// Final verification: Ensure pod_annotations is in payload before template rendering
	// This is a safety check to ensure config.yaml defaults are applied
	if finalPodAnnotations, ok := payload["pod_annotations"]; ok {
		if finalMap, ok := finalPodAnnotations.(map[string]string); ok && len(finalMap) > 0 {
			log.Info().
				Str("appName", appName).
				Int("annotationCount", len(finalMap)).
				Interface("annotations", finalMap).
				Msg("CreateValuesYaml: Final verification - pod_annotations ready for template rendering")
		} else if finalMap, ok := finalPodAnnotations.(map[string]interface{}); ok && len(finalMap) > 0 {
			log.Info().
				Str("appName", appName).
				Int("annotationCount", len(finalMap)).
				Interface("annotations", finalMap).
				Msg("CreateValuesYaml: Final verification - pod_annotations ready for template rendering (map[string]interface{})")
		} else {
			log.Warn().
				Str("appName", appName).
				Interface("podAnnotationsType", fmt.Sprintf("%T", finalPodAnnotations)).
				Interface("podAnnotations", finalPodAnnotations).
				Msg("CreateValuesYaml: pod_annotations in payload is empty or has unexpected type")
		}
	} else {
		log.Warn().
			Str("appName", appName).
			Msg("CreateValuesYaml: pod_annotations not found in payload before template rendering")
	}

	// Render and push values.yaml
	// The RenderTemplate function will automatically compile common-values.tmpl with the app-type template
	log.Info().
		Str("appName", appName).
		Str("repo", repo).
		Str("branch", branch).
		Str("filePath", valuesYamlPath).
		Str("templateName", templateName).
		Str("workingEnv", workingEnv).
		Msg("CreateValuesYaml: Rendering values.yaml template and pushing to GitHub")

	err := templates.RenderTemplate(
		payload,
		templateName,
		repo,
		branch,
		valuesYamlPath,
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("appName", appName).
			Str("repo", repo).
			Str("branch", branch).
			Str("filePath", valuesYamlPath).
			Str("templateName", templateName).
			Str("workingEnv", workingEnv).
			Msg("CreateValuesYaml: Failed to render values.yaml template or push to GitHub")
		return fmt.Errorf("failed to render values.yaml template: %w", err)
	}

	log.Info().
		Str("appName", appName).
		Str("repo", repo).
		Str("branch", branch).
		Str("filePath", valuesYamlPath).
		Str("templateName", templateName).
		Msg("CreateValuesYaml: Values.yaml template rendered and pushed to GitHub successfully")

	log.Info().
		Str("appName", appName).
		Str("filePath", valuesYamlPath).
		Msg("CreateValuesYaml activity completed")

	return nil
}

// CreateCloudDNS and CreateCoreDNS are implemented in:
// - dns_activities_stub.go (//go:build !meesho) - stub implementation for open-source builds
// - Organization-specific implementations (//go:build meesho) - real implementation that calls DNS API
// These functions are conditionally compiled based on build tags

// IamBinding binds service account with IAM workload identity user role
// This corresponds to RingMaster's BindServiceAccountActivity
// In RingMaster, this is executed as a separate IamBindingWorkflow after MlpOnboardingWorkflow
// This binds a Kubernetes Service Account (KSA) to a GCP Service Account (GSA) using workload identity
func IamBinding(payload map[string]interface{}, workingEnv string) error {
	appName := getString(payload, "appName")

	log.Info().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Msg("IamBinding activity started")

	// Get service account from payload (it's already provided during onboarding)
	serviceAccount := gcp.GetServiceAccountFromPayload(payload, workingEnv)
	if serviceAccount == "" {
		log.Error().
			Str("appName", appName).
			Str("workingEnv", workingEnv).
			Msg("IamBinding: Service account not found in payload - cannot bind workload identity")
		return fmt.Errorf("service account not found in payload for appName %s and workingEnv %s", appName, workingEnv)
	}

	// Get GCP project ID from environment config
	projectID := gcp.GetGCPProjectID(workingEnv)
	if projectID == "" {
		log.Warn().
			Str("appName", appName).
			Str("workingEnv", workingEnv).
			Str("serviceAccount", serviceAccount).
			Msg("IamBinding: GCP project ID not configured - skipping workload identity binding (not needed for local Kubernetes)")
		// Skip IamBinding gracefully for local Kubernetes setups where GCP workload identity is not required
		// This allows the workflow to continue without GCP configuration
		return nil
	}

	// Get Kubernetes namespace: {env}-{appName}
	namespace := gcp.GetNamespace(appName, workingEnv)
	if namespace == "" {
		log.Error().
			Str("appName", appName).
			Str("workingEnv", workingEnv).
			Msg("IamBinding: Failed to construct namespace")
		return fmt.Errorf("failed to construct namespace for appName %s and workingEnv %s", appName, workingEnv)
	}

	// KSA name is typically the same as namespace (Kubernetes Service Account name)
	ksaName := namespace

	log.Info().
		Str("appName", appName).
		Str("serviceAccount", serviceAccount).
		Str("projectID", projectID).
		Str("namespace", namespace).
		Str("ksaName", ksaName).
		Str("workingEnv", workingEnv).
		Msg("IamBinding: Constructed service account binding configuration")

	// Create service account config
	serviceAccountConfig := gcp.ServiceAccountConfig{
		ServiceAccount: serviceAccount,
		Role:           gcp.IamWorkloadIdentityUser,
		ProjectID:      projectID,
		Namespace:      namespace,
		KSAName:        ksaName,
	}

	// Create service account service and bind
	gcpService := gcp.NewServiceAccountService(serviceAccountConfig)
	ctx := context.Background()
	if err := gcpService.BindServiceAccount(ctx); err != nil {
		log.Error().
			Err(err).
			Str("appName", appName).
			Str("serviceAccount", serviceAccount).
			Str("namespace", namespace).
			Msg("IamBinding: Failed to bind service account")
		return fmt.Errorf("failed to bind service account: %w", err)
	}

	log.Info().
		Str("appName", appName).
		Str("serviceAccount", serviceAccount).
		Str("namespace", namespace).
		Str("ksaName", ksaName).
		Msg("IamBinding activity completed successfully")

	return nil
}

// Helper function
func getString(payload map[string]interface{}, key string) string {
	if val, ok := payload[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
