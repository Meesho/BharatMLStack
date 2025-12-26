package templates

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"

	"github.com/Meesho/BharatMLStack/horizon/pkg/github"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// RenderTemplate renders a template with the given bindings and pushes to GitHub
// This is similar to RingMaster's utils.RenderTemplate function
func RenderTemplate(bindings map[string]interface{}, templateName string, repo string, branch string, path string) error {
	log.Info().
		Str("templateName", templateName).
		Str("repo", repo).
		Str("branch", branch).
		Str("path", path).
		Msg("Entered RenderTemplate - template rendering")

	commonTemplate := "common-values.tmpl"

	// Template function map (same as RingMaster)
	funcMap := template.FuncMap{
		"toYAML":   ToYAML,
		"toLower":  strings.ToLower,
		"contains": strings.Contains,
		"indent":   indent,
		"nindent":  nindent,
	}

	var tmpl *template.Template
	var err error

	// Templates that include common-values.tmpl (app-type templates)
	// Standalone templates don't include common-values.tmpl
	// Note: Only templates actually used in Horizon are listed here
	standaloneTemplates := []string{
		"argoapp.tmpl",
		"deployment.tmpl",
		"values_properties.tmpl",
	}

	isStandalone := false
	for _, standalone := range standaloneTemplates {
		if templateName == standalone {
			isStandalone = true
			break
		}
	}

	if !isStandalone {
		// App-type template: Parse common-values.tmpl first, then the specific template
		tmpl, err = template.New(commonTemplate).Funcs(funcMap).ParseFS(Templates, commonTemplate)
		if err != nil {
			log.Error().Err(err).Str("templateName", templateName).Msg("Failed to parse common template")
			return fmt.Errorf("failed to parse common template: %w", err)
		}
		tmpl, err = tmpl.New(templateName).Funcs(funcMap).ParseFS(Templates, templateName)
		if err != nil {
			log.Error().Err(err).Str("templateName", templateName).Msg("Failed to parse app-type template")
			return fmt.Errorf("failed to parse app-type template: %w", err)
		}
	} else {
		// Standalone template: Parse only the specific template
		tmpl, err = template.New(templateName).Funcs(funcMap).ParseFS(Templates, templateName)
		if err != nil {
			log.Error().Err(err).Str("templateName", templateName).Msg("Failed to parse standalone template")
			return fmt.Errorf("failed to parse template: %w", err)
		}
	}

	// Execute template
	log.Info().
		Str("templateName", templateName).
		Str("repo", repo).
		Str("branch", branch).
		Str("path", path).
		Int("bindingsCount", len(bindings)).
		Msg("RenderTemplate: Executing template with bindings")

	var resultBuffer bytes.Buffer
	err = tmpl.Execute(&resultBuffer, bindings)
	if err != nil {
		log.Error().
			Err(err).
			Str("templateName", templateName).
			Str("repo", repo).
			Str("branch", branch).
			Str("path", path).
			Msg("RenderTemplate: Template execution failed - check template syntax and bindings")
		return fmt.Errorf("error in template rendering: %w", err)
	}

	log.Info().
		Str("templateName", templateName).
		Str("path", path).
		Int("outputSize", resultBuffer.Len()).
		Msg("RenderTemplate: Template executed successfully")

	// Validate YAML by unmarshaling and marshaling (same as RingMaster)
	log.Info().
		Str("templateName", templateName).
		Str("path", path).
		Msg("RenderTemplate: Validating rendered YAML content")

	var formatYaml map[string]interface{}
	err = yaml.Unmarshal(resultBuffer.Bytes(), &formatYaml)
	if err != nil {
		log.Error().
			Err(err).
			Str("templateName", templateName).
			Str("repo", repo).
			Str("branch", branch).
			Str("path", path).
			Str("yamlContent", resultBuffer.String()).
			Msg("RenderTemplate: YAML validation failed - rendered content is not valid YAML")
		return fmt.Errorf("error in yaml validation: %w", err)
	}

	log.Info().
		Str("templateName", templateName).
		Str("path", path).
		Msg("RenderTemplate: YAML validation successful")

	resultContent, err := yaml.Marshal(formatYaml)
	if err != nil {
		log.Error().
			Err(err).
			Str("templateName", templateName).
			Str("repo", repo).
			Str("branch", branch).
			Str("path", path).
			Msg("RenderTemplate: YAML marshalling failed - unable to convert validated YAML back to bytes")
		return fmt.Errorf("yaml marshalling failed: %w", err)
	}

	log.Info().
		Str("templateName", templateName).
		Str("path", path).
		Int("finalContentSize", len(resultContent)).
		Msg("RenderTemplate: YAML marshalling successful")

	// Check if file content has changed (same as RingMaster)
	log.Info().
		Str("templateName", templateName).
		Str("repo", repo).
		Str("branch", branch).
		Str("path", path).
		Msg("RenderTemplate: Checking if file exists and content has changed")

	existingFileContent, err := GetFileContent(path, repo, branch)
	if err == nil {
		// File exists, compare content
		if bytes.Equal(existingFileContent, resultContent) {
			log.Info().
				Str("templateName", templateName).
				Str("path", path).
				Str("repo", repo).
				Str("branch", branch).
				Msg("RenderTemplate: File exists with identical content - skipping push to GitHub")
			return nil
		}
		log.Info().
			Str("templateName", templateName).
			Str("path", path).
			Str("repo", repo).
			Str("branch", branch).
			Int("existingSize", len(existingFileContent)).
			Int("newSize", len(resultContent)).
			Msg("RenderTemplate: File content has changed - will push to GitHub")
	} else {
		log.Info().
			Err(err).
			Str("templateName", templateName).
			Str("path", path).
			Str("repo", repo).
			Str("branch", branch).
			Msg("RenderTemplate: File does not exist or cannot be retrieved - will create new file in GitHub")
	}

	// Push to GitHub
	log.Info().
		Str("templateName", templateName).
		Str("repo", repo).
		Str("branch", branch).
		Str("path", path).
		Int("contentSize", len(resultContent)).
		Msg("RenderTemplate: Pushing content to GitHub")

	err = PushContentToGithub(resultContent, repo, branch, path)
	if err != nil {
		log.Error().
			Err(err).
			Str("templateName", templateName).
			Str("repo", repo).
			Str("branch", branch).
			Str("path", path).
			Int("contentSize", len(resultContent)).
			Msg("RenderTemplate: Failed to push content to GitHub - check GitHub authentication, permissions, and API rate limits")
		return fmt.Errorf("failed to push content to GitHub: %w", err)
	}

	log.Info().
		Str("templateName", templateName).
		Str("repo", repo).
		Str("branch", branch).
		Str("path", path).
		Msg("RenderTemplate: Content pushed to GitHub successfully")

	log.Info().
		Str("templateName", templateName).
		Str("path", path).
		Msg("Template rendering and GitHub push completed successfully")

	return nil
}

// ToYAML converts a value to YAML string
func ToYAML(v interface{}) (string, error) {
	log.Info().Msg("Entered ToYAML func")
	yamlBytes, err := yaml.Marshal(v)
	if err != nil {
		log.Error().Err(err).Msg("ToYAML: Failed to marshal value to YAML")
		return "", err
	}
	log.Info().Msg("Successfully marshalled value to YAML")
	return string(yamlBytes), nil
}

// indent adds padding spaces to each line
func indent(spaces int, v string) string {
	log.Info().Msg("Entered indent func to add padding spaces")
	pad := strings.Repeat(" ", spaces)
	return pad + strings.ReplaceAll(v, "\n", "\n"+pad)
}

// nindent adds a newline followed by padding spaces
func nindent(spaces int, v string) string {
	log.Info().Msg("Entered nindent func to adds a newline followed by padding spaces")
	return "\n" + indent(spaces, v)
}

// GetFileContent retrieves file content from GitHub (helper for RenderTemplate)
func GetFileContent(path string, repo string, branch string) ([]byte, error) {
	log.Info().
		Str("repo", repo).
		Str("branch", branch).
		Str("path", path).
		Msg("GetFileContent: Retrieving file content from GitHub")

	client, err := github.GetGitHubClient()
	if err != nil {
		log.Error().
			Err(err).
			Str("repo", repo).
			Str("branch", branch).
			Str("path", path).
			Msg("GetFileContent: Failed to get GitHub client - GitHub client may not be initialized")
		return nil, fmt.Errorf("failed to get GitHub client: %w", err)
	}
	ctx := context.TODO()

	fileContent, err := github.GetFile(ctx, client, repo, path, branch)
	if err != nil {
		log.Warn().
			Err(err).
			Str("repo", repo).
			Str("branch", branch).
			Str("path", path).
			Msg("GetFileContent: Failed to get file from GitHub - file may not exist (this is OK for new files)")
		return nil, fmt.Errorf("failed to get file from GitHub: %w", err)
	}

	// Decode base64 content
	decoded, err := base64.StdEncoding.DecodeString(*fileContent.Content)
	if err != nil {
		log.Error().
			Err(err).
			Str("repo", repo).
			Str("branch", branch).
			Str("path", path).
			Msg("GetFileContent: Failed to decode base64 file content from GitHub")
		return nil, fmt.Errorf("failed to decode file content: %w", err)
	}

	log.Info().
		Str("repo", repo).
		Str("branch", branch).
		Str("path", path).
		Int("contentSize", len(decoded)).
		Msg("GetFileContent: File content retrieved and decoded successfully")

	return decoded, nil
}

// PushContentToGithub pushes content to GitHub (creates or updates file)
// This is a wrapper around github.PushContentToGithub for consistency
func PushContentToGithub(content []byte, repo string, branch string, path string) error {
	log.Info().
		Str("repo", repo).
		Str("branch", branch).
		Str("path", path).
		Int("contentSize", len(content)).
		Msg("PushContentToGithub: Attempting to push content to GitHub")

	err := github.PushContentToGithub(content, repo, branch, path)
	if err != nil {
		log.Error().
			Err(err).
			Str("repo", repo).
			Str("branch", branch).
			Str("path", path).
			Int("contentSize", len(content)).
			Msg("PushContentToGithub: Failed to push content to GitHub - check GitHub API response, authentication, permissions, and branch existence")
		return fmt.Errorf("failed to push content to GitHub: %w", err)
	}

	log.Info().
		Str("repo", repo).
		Str("branch", branch).
		Str("path", path).
		Msg("PushContentToGithub: Content pushed to GitHub successfully")

	return nil
}
