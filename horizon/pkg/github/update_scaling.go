package github

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/pkg/argocd"
	"github.com/maolinc/copier"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// GetFileYaml retrieves a YAML file from GitHub and returns it as a map along with its SHA
// Uses new environment-based directory structure: {workingEnv}/deployables/{appName}/{fileName}
// The environment is determined from the ArgoCD application metadata to match the directory structure
func GetFileYaml(argocdApplication argocd.ArgoCDApplicationMetadata, fileName string, workingEnv string) (map[string]interface{}, *string, error) {
	log.Info().Str("appName", argocdApplication.Name).Str("fileName", fileName).Str("workingEnv", workingEnv).Msg("Entered GetFileYaml")

	// Determine the environment directory from ArgoCD application metadata
	// This ensures we update files in the correct environment-specific directory
	envDir := determineEnvironmentDirectory(argocdApplication, workingEnv)

	// Use new environment-based path structure (vendor-agnostic)
	// BU and Team are ignored in new structure - only environment and appName matter
	appName := argocdApplication.Labels.AppName
	if appName == "" {
		return nil, nil, fmt.Errorf("appName not found in ArgoCD application labels")
	}
	basePath := GetBasePath("", "", envDir, appName)
	if basePath == "" {
		return nil, nil, fmt.Errorf("failed to construct base path for workingEnv: %s", workingEnv)
	}
	filePath := basePath + "/" + fileName
	client, err := GetGitHubClient()
	if err != nil {
		return nil, nil, err
	}
	ctx := context.TODO()

	// Use configured repository and branch (mandatory from .env file)
	// All files will be pushed to the configured repository and branch
	// This ensures consistency: deployables created in REPOSITORY_NAME/BRANCH_NAME
	// will have their threshold updates in the same repository and branch
	repo := GetRepositoryForCommits()
	if repo == "" {
		return nil, nil, fmt.Errorf("REPOSITORY_NAME is required - set it in .env file and call InitRepositoryAndBranch during startup")
	}
	branch := GetBranchForCommits()
	if branch == "" {
		return nil, nil, fmt.Errorf("BRANCH_NAME is required - set it in .env file and call InitRepositoryAndBranch during startup")
	}

	log.Info().
		Str("repo", repo).
		Str("branch", branch).
		Str("filePath", filePath).
		Str("workingEnv", workingEnv).
		Str("envDir", envDir).
		Msg("GetFileYaml: Using configured repository and branch for threshold updates")

	fileContent, err := GetFile(ctx, client, repo, filePath, branch)
	if err != nil {
		log.Error().Err(err).Msg("In GetFileYaml: Failed to fetch file")
		return nil, nil, err
	}
	fileValue, err := base64.StdEncoding.DecodeString(*fileContent.Content)
	if err != nil {
		log.Error().Err(err).Msg("In GetFileYaml: Failed to decode file content")
		return nil, nil, err
	}

	var data map[string]interface{}
	err = yaml.Unmarshal(fileValue, &data)
	if err != nil {
		log.Error().Err(err).Str("appName", argocdApplication.Name).Msg("In GetFileYaml: First YAML unmarshal failed")
		// Try with duplicate handling
		errIn := yaml.Unmarshal(handleDuplicates(fileValue), &data)
		if errIn != nil {
			log.Error().Err(errIn).Str("appName", argocdApplication.Name).Msg("In GetFileYaml: Second YAML unmarshal failed")
			return data, nil, errors.New("we could not parse the values yaml file because of duplicate keys, please use the job for now. This will be fixed soon")
		}
	}

	return data, fileContent.SHA, nil
}

// UpdateFileYaml updates a YAML file in GitHub repository
// Uses new environment-based directory structure: {workingEnv}/deployables/{appName}/{fileName}
// The environment is determined from the ArgoCD application metadata to match the directory structure
func UpdateFileYaml(argocdApplication argocd.ArgoCDApplicationMetadata, data map[string]interface{}, fileName string, commitMessage string, sha *string, workingEnv string) error {
	log.Info().Str("application", argocdApplication.Name).Str("fileName", fileName).Str("workingEnv", workingEnv).Msg("Entered UpdateFileYaml")

	// Determine the environment directory from ArgoCD application metadata
	// This ensures we update files in the correct environment-specific directory
	envDir := determineEnvironmentDirectory(argocdApplication, workingEnv)

	// Use new environment-based path structure (vendor-agnostic)
	// BU and Team are ignored in new structure - only environment and appName matter
	appName := argocdApplication.Labels.AppName
	if appName == "" {
		return fmt.Errorf("appName not found in ArgoCD application labels")
	}
	basePath := GetBasePath("", "", envDir, appName)
	if basePath == "" {
		return fmt.Errorf("failed to construct base path for workingEnv: %s", workingEnv)
	}
	filePath := basePath + "/" + fileName
	client, err := GetGitHubClient()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	// Use configured repository and branch (mandatory from .env file)
	// All files will be pushed to the configured repository and branch
	// This ensures consistency: deployables created in REPOSITORY_NAME/BRANCH_NAME
	// will have their threshold updates in the same repository and branch
	repo := GetRepositoryForCommits()
	if repo == "" {
		return fmt.Errorf("REPOSITORY_NAME is required - set it in .env file and call InitRepositoryAndBranch during startup")
	}
	branch := GetBranchForCommits()
	if branch == "" {
		return fmt.Errorf("BRANCH_NAME is required - set it in .env file and call InitRepositoryAndBranch during startup")
	}

	log.Info().
		Str("repo", repo).
		Str("branch", branch).
		Str("filePath", filePath).
		Str("workingEnv", workingEnv).
		Str("envDir", envDir).
		Str("commitMessage", commitMessage).
		Msg("UpdateFileYaml: Using configured repository and branch for threshold updates")

	content, err := yaml.Marshal(data)
	if err != nil {
		log.Error().Err(err).Msg("In UpdateFileYaml: Failed to marshal data into YAML")
		return err
	}

	keysToQuote := []string{"start", "end", "timezone"}

	// Loop through the keys and use regular expressions to enclose values with double quotes
	for _, key := range keysToQuote {
		// Compile the regex
		pattern := regexp.MustCompile(fmt.Sprintf(`(?m)(^\s*- metadata:\s*\n(?:\s+.*\n)*?)(\s*%s:\s*)([^"\n]+)(\n|$)`, key))

		// Perform the replacement
		content = pattern.ReplaceAll(content, []byte("${1}${2}\"${3}\"${4}"))
	}

	_, err = UpdateFile(ctx, client, repo, filePath, sha, &commitMessage, content, &branch)
	if err != nil {
		log.Error().Err(err).Str("filePath", filePath).Msg("In UpdateFileYaml: Failed to update file")
		return err
	}

	log.Info().Str("fileName", fileName).Str("application", argocdApplication.Name).Msg("Successfully updated file")
	return nil
}

func handleDuplicates(data []byte) []byte {
	log.Info().Msg("Entered handleDuplicates func")
	str := string(data)

	// Continue adding any more
	str = strings.Replace(str, "replicaCount:", "replicaCountz:", 1)

	return []byte(str)
}

// determineEnvironmentDirectory determines the environment directory name from ArgoCD application metadata
// This ensures threshold updates target the correct environment-specific directory that matches the repository structure
// Priority:
// 1. Use workingEnv from API (if it matches a valid directory pattern)
// 2. Derive from ArgoCD application name (e.g., "prd-appname" -> "gcp_prd" or "prd")
// 3. Use ArgoCD labels.Env if available (may need normalization)
// 4. Fall back to workingEnv as-is
func determineEnvironmentDirectory(argocdApplication argocd.ArgoCDApplicationMetadata, workingEnv string) string {
	// First, try to derive from ArgoCD application name
	// ArgoCD apps are typically named like "prd-appname", "int-appname", "stg-appname"
	appName := argocdApplication.Name
	if appName != "" {
		// Extract environment prefix from application name
		// Examples: "prd-appname" -> "prd", "int-appname" -> "int", "gcp_prd-appname" -> "gcp_prd"
		parts := strings.Split(appName, "-")
		if len(parts) > 0 {
			envPrefix := parts[0]

			// Check if it's a GCP environment (gcp_prd, gcp_int, gcp_stg, gcp_dev)
			if strings.HasPrefix(envPrefix, "gcp_") {
				log.Info().
					Str("appName", appName).
					Str("envDir", envPrefix).
					Str("workingEnv", workingEnv).
					Msg("Determined environment directory from ArgoCD application name (GCP)")
				return envPrefix
			}

			// For non-GCP environments, check if workingEnv has gcp_ prefix
			// If workingEnv is "gcp_prd" but app name is "prd-appname", use workingEnv
			if strings.HasPrefix(workingEnv, "gcp_") {
				// workingEnv is GCP, but app name doesn't have gcp_ prefix
				// Use workingEnv as it likely matches the directory structure
				log.Info().
					Str("appName", appName).
					Str("envDir", workingEnv).
					Str("workingEnv", workingEnv).
					Msg("Using workingEnv as environment directory (GCP environment)")
				return workingEnv
			}

			// For non-GCP, use the prefix from app name if it matches common patterns
			if envPrefix == "prd" || envPrefix == "int" || envPrefix == "stg" || envPrefix == "dev" || envPrefix == "ftr" {
				log.Info().
					Str("appName", appName).
					Str("envDir", envPrefix).
					Str("workingEnv", workingEnv).
					Msg("Determined environment directory from ArgoCD application name")
				return envPrefix
			}
		}
	}

	// Fall back to workingEnv - it should match the directory structure
	log.Info().
		Str("appName", appName).
		Str("envDir", workingEnv).
		Str("workingEnv", workingEnv).
		Msg("Using workingEnv as environment directory (fallback)")
	return workingEnv
}

// UpdateThresholdHelmRepo is the unified function to update CPU or GPU thresholds in GitHub
func UpdateThresholdHelmRepo(argocdApplication argocd.ArgoCDApplicationMetadata, thresholdProps ThresholdProperties, email string, workingEnv string) error {
	triggerType := thresholdProps.GetTriggerType()
	threshold := thresholdProps.GetThreshold()

	log.Info().
		Str("ApplicationName", argocdApplication.Name).
		Str("triggerType", triggerType).
		Str("threshold", threshold).
		Str("UpdatedBy", email).
		Str("WorkingEnv", workingEnv).
		Msg("Entered unified UpdateThresholdHelmRepo")

	// Update values_properties.yaml
	err := UpdateTriggersInValuesProperties(argocdApplication, triggerType, threshold, email, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("triggerType", triggerType).Msg("UpdateThresholdHelmRepo: Failed to update values_properties.yaml")
		return err
	}

	// Update values.yaml
	err = UpdateTriggersInValues(argocdApplication, triggerType, threshold, email, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("triggerType", triggerType).Msg("UpdateThresholdHelmRepo: Failed to update values.yaml")
		return err
	}

	return nil
}

// UpdateTriggersInValues updates triggers in values.yaml
func UpdateTriggersInValues(argocdApplication argocd.ArgoCDApplicationMetadata, triggerType string, threshold string, email string, workingEnv string) error {
	updater := GetTriggerUpdater(triggerType, argocdApplication.Name)
	if updater == nil {
		return fmt.Errorf("unsupported trigger type: %s", triggerType)
	}

	log.Info().
		Str("ApplicationName", argocdApplication.Name).
		Str("triggerType", triggerType).
		Str("threshold", threshold).
		Str("UpdatedBy", email).
		Str("WorkingEnv", workingEnv).
		Msg("Updating triggers in values.yaml")

	data, sha, err := GetFileYaml(argocdApplication, "values.yaml", workingEnv)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve values.yaml")
		return err
	}

	var autoscaling map[string]interface{}
	copier.CopyWithOption(&autoscaling, data["autoscaling"], copier.Option{IgnoreEmpty: true, DeepCopy: true})

	var triggers []interface{}
	copier.CopyWithOption(&triggers, autoscaling["triggers"], copier.Option{IgnoreEmpty: true, DeepCopy: true})

	autoscaling["triggers"] = updater.UpdateTriggers(triggers, threshold)
	data["autoscaling"] = autoscaling

	commitMsg := fmt.Sprintf("%s: %s Threshold updated by %s", argocdApplication.Name, strings.ToUpper(triggerType), email)
	err = UpdateFileYaml(argocdApplication, data, "values.yaml", commitMsg, sha, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("triggerType", triggerType).Msg("Failed to update values.yaml")
		return err
	}

	return nil
}

// UpdateTriggersInValuesProperties updates triggers in values_properties.yaml
func UpdateTriggersInValuesProperties(argocdApplication argocd.ArgoCDApplicationMetadata, triggerType string, threshold string, email string, workingEnv string) error {
	updater := GetTriggerUpdater(triggerType, argocdApplication.Name)
	if updater == nil {
		return fmt.Errorf("unsupported trigger type: %s", triggerType)
	}

	log.Info().
		Str("ApplicationName", argocdApplication.Name).
		Str("triggerType", triggerType).
		Str("threshold", threshold).
		Str("UpdatedBy", email).
		Str("WorkingEnv", workingEnv).
		Msg("Updating triggers in values_properties.yaml")

	data, sha, err := GetFileYaml(argocdApplication, "values_properties.yaml", workingEnv)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve values_properties.yaml")
		return err
	}

	var triggers []interface{}
	copier.CopyWithOption(&triggers, data["triggers"], copier.Option{IgnoreEmpty: true, DeepCopy: true})

	data["triggers"] = updater.UpdateTriggers(triggers, threshold)

	commitMsg := fmt.Sprintf("%s: %s threshold updated by %s", argocdApplication.Name, strings.ToUpper(triggerType), email)
	err = UpdateFileYaml(argocdApplication, data, "values_properties.yaml", commitMsg, sha, workingEnv)
	if err != nil {
		log.Error().Err(err).Str("triggerType", triggerType).Msg("Failed to update values_properties.yaml")
		return err
	}

	return nil
}
