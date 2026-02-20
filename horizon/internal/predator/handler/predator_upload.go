package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	pred "github.com/Meesho/BharatMLStack/horizon/internal/predator"
	"github.com/Meesho/BharatMLStack/horizon/internal/predator/proto/modelconfig"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/prototext"
)

// uploadSingleModel processes a single model upload with improved validation and error handling
func (p *Predator) uploadSingleModel(modelItem ModelUploadItem, bucket, basePath string, isPartial bool, authToken string) ModelUploadResult {
	// Step 1: Extract and validate model name
	modelName, err := p.extractModelName(modelItem.Metadata)
	if err != nil {
		return p.createErrorResult("unknown", "Failed to extract model name", err)
	}

	log.Info().Msgf("Processing %s upload for model: %s from %s",
		map[bool]string{true: "partial", false: "full"}[isPartial], modelName, modelItem.GCSPath)

	// Step 2: Setup destination paths
	destPath := path.Join(basePath, modelName)
	fullGCSPath := fmt.Sprintf("gs://%s/%s", bucket, destPath)

	// Step 3: Validate upload prerequisites
	if err := p.validateUploadPrerequisites(bucket, destPath, isPartial, modelName); err != nil {
		return p.createErrorResult(modelName, "Upload prerequisites validation failed", err)
	}

	// Step 4: Validate source model structure and configuration
	if err := p.validateSourceModel(modelItem.GCSPath, isPartial); err != nil {
		return p.createErrorResult(modelName, "Source model validation failed", err)
	}

	// Step 5: Validate metadata features (after model structure validation)
	if err := p.validateMetadataFeatures(modelItem.Metadata, authToken); err != nil {
		return p.createErrorResult(modelName, "Feature validation failed", err)
	}

	// Step 6: Download/sync model files based on upload type
	if err := p.syncModelFiles(modelItem.GCSPath, bucket, destPath, modelName, isPartial); err != nil {
		return p.createErrorResult(modelName, "Model file sync failed", err)
	}

	// Step 7: Copy config.pbtxt to prod config source (only in production)
	if err := p.copyConfigToProdConfigSource(modelItem.GCSPath, modelName); err != nil {
		return p.createErrorResult(modelName, "Failed to copy config to prod config source", err)
	}

	// Upload processed metadata.json (always done regardless of partial/full)
	metadataPath, err := p.uploadModelMetadata(modelItem.Metadata, bucket, destPath)
	if err != nil {
		return p.createErrorResult(modelName, "Metadata upload failed", err)
	}

	log.Info().Msgf("Successfully completed %s upload for model: %s",
		map[bool]string{true: "partial", false: "full"}[isPartial], modelName)
	return ModelUploadResult{
		ModelName:    modelName,
		GCSPath:      fullGCSPath,
		MetadataPath: metadataPath,
		Status:       "success",
	}
}

// copyConfigToProdConfigSource copies config.pbtxt to the prod config source path
func (p *Predator) copyConfigToProdConfigSource(gcsPath, modelName string) error {
	if pred.GcsConfigBucket == "" || pred.GcsConfigBasePath == "" {
		log.Warn().Msg("Config source not configured, skipping config.pbtxt copy to config source")
		return nil
	}

	srcBucket, srcPath := extractGCSPath(gcsPath)
	if srcBucket == "" || srcPath == "" {
		return fmt.Errorf("invalid GCS path format: %s", gcsPath)
	}

	srcConfigPath := path.Join(srcPath, configFile)
	configData, err := p.GcsClient.ReadFile(srcBucket, srcConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read config.pbtxt from source: %w", err)
	}

	updatedConfigData, err := externalcall.ReplaceModelNameInConfig(configData, modelName)
	if err != nil {
		return fmt.Errorf("failed to replace model name in config: %w", err)
	}
	destConfigPath := path.Join(pred.GcsConfigBasePath, modelName, configFile)
	if err := p.GcsClient.UploadFile(pred.GcsConfigBucket, destConfigPath, updatedConfigData); err != nil {
		return fmt.Errorf("failed to upload config.pbtxt to config source: %w", err)
	}

	log.Info().Msgf("Successfully copied config.pbtxt to config source with model name %s: gs://%s/%s",
		modelName, pred.GcsConfigBucket, destConfigPath)
	return nil
}

// createErrorResult creates a standardized error result
func (p *Predator) createErrorResult(modelName, message string, err error) ModelUploadResult {
	return ModelUploadResult{
		ModelName: modelName,
		Status:    "error",
		Error:     fmt.Sprintf("%s: %v", message, err),
	}
}

// generateUploadSummary creates response message and status code based on results
func (p *Predator) generateUploadSummary(successCount, failCount int, results []ModelUploadResult) (string, int) {
	switch {
	case failCount == 0:
		return fmt.Sprintf("%d model uploaded successfully", successCount), http.StatusOK
	case successCount == 0:
		return fmt.Sprintf("%d model failed to upload. Errors: %s", failCount, results[0].Error), http.StatusBadRequest
	default:
		return fmt.Sprintf("Mixed results: %d successful, %d failed. Errors: %s", successCount, failCount, results[0].Error), http.StatusPartialContent
	}
}

// validateUploadPrerequisites validates upload requirements based on type
func (p *Predator) validateUploadPrerequisites(bucket, destPath string, isPartial bool, modelName string) error {
	exists, err := p.GcsClient.CheckFolderExists(bucket, destPath)
	if err != nil {
		return fmt.Errorf("failed to check model existence: %w", err)
	}

	if isPartial {
		if !exists {
			return fmt.Errorf("partial upload requires existing model folder at destination")
		}
		log.Info().Msgf("Partial upload: updating existing model %s", modelName)
	} else {
		if exists {
			log.Info().Msgf("Full upload: replacing existing model %s", modelName)
		} else {
			log.Info().Msgf("Full upload: creating new model %s", modelName)
		}
	}

	return nil
}

// validateSourceModel validates the source model structure and configuration
func (p *Predator) validateSourceModel(gcsPath string, isPartial bool) error {
	srcBucket, srcPath := extractGCSPath(gcsPath)
	if srcBucket == "" || srcPath == "" {
		return fmt.Errorf("invalid GCS path format: %s", gcsPath)
	}

	if err := p.validateModelConfiguration(gcsPath); err != nil {
		return fmt.Errorf("config.pbtxt validation failed: %w", err)
	}

	if !isPartial {
		if err := p.validateCompleteModelStructure(srcBucket, srcPath); err != nil {
			return fmt.Errorf("complete model structure validation failed: %w", err)
		}
	}

	return nil
}

// validateCompleteModelStructure validates that version "1" folder exists with non-empty files
func (p *Predator) validateCompleteModelStructure(srcBucket, srcPath string) error {
	versionPath := path.Join(srcPath, "1")
	exists, err := p.GcsClient.CheckFolderExists(srcBucket, versionPath)
	if err != nil {
		return fmt.Errorf("failed to check version folder 1/: %w", err)
	}

	if !exists {
		return fmt.Errorf("version folder 1/ not found - required for complete model")
	}

	if err := p.validateVersionHasFiles(srcBucket, versionPath); err != nil {
		return fmt.Errorf("version folder 1/ validation failed: %w", err)
	}

	log.Info().Msg("Model structure validation passed - version 1/ folder exists with files")
	return nil
}

// validateVersionHasFiles checks if version folder has at least one non-empty file
func (p *Predator) validateVersionHasFiles(srcBucket, versionPath string) error {
	info, err := p.GcsClient.GetFolderInfo(srcBucket, versionPath)
	if err != nil {
		return fmt.Errorf("failed to get version folder info: %w", err)
	}
	if info.FileCount == 0 {
		return fmt.Errorf("version folder 1/ is empty - must contain model files")
	}
	log.Info().Msg("Version folder 1/ contains files")
	return nil
}

// syncModelFiles handles file synchronization based on upload type
func (p *Predator) syncModelFiles(gcsPath, destBucket, destPath, modelName string, isPartial bool) error {
	if isPartial {
		return p.syncPartialFiles(gcsPath, destBucket, destPath, modelName)
	}
	return p.syncFullModel(gcsPath, destBucket, destPath, modelName)
}

// uploadModelMetadata uploads metadata.json to GCS and returns the full path
func (p *Predator) uploadModelMetadata(metadata interface{}, bucket, destPath string) (string, error) {
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to serialize metadata: %w", err)
	}

	metadataPath := path.Join(destPath, "metadata.json")
	if err := p.GcsClient.UploadFile(bucket, metadataPath, metadataBytes); err != nil {
		return "", fmt.Errorf("failed to upload metadata: %w", err)
	}

	return fmt.Sprintf("gs://%s/%s", bucket, metadataPath), nil
}

// validateMetadataFeatures validates the features in metadata against online/offline validation APIs
func (p *Predator) validateMetadataFeatures(metadata interface{}, authToken string) error {
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var featureMeta FeatureMetadata
	if err := json.Unmarshal(metadataBytes, &featureMeta); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	if authToken == "" {
		return fmt.Errorf("authorization token is required for feature validation")
	}

	onlineFeaturesByEntity := make(map[string][]string)
	pricingFeaturesByEntity := make(map[string][]string)
	var offlineFeatures []string

	for _, input := range featureMeta.Inputs {
		for _, feature := range input.Features {
			featureType, entity, gf, featureName, isValid := externalcall.ParseFeatureString(feature)
			if !isValid {
				log.Error().Msgf("Invalid feature format: %s", feature)
				return fmt.Errorf("invalid feature format: %s", feature)
			}

			switch featureType {
			case "ONLINE_FEATURE", "PARENT_ONLINE_FEATURE":
				onlineFeaturesByEntity[entity] = append(onlineFeaturesByEntity[entity], gf)
				log.Info().Msgf("Added online feature for validation - entity: %s, feature: %s", entity, gf)
			case "OFFLINE_FEATURE", "PARENT_OFFLINE_FEATURE":
				offlineFeatures = append(offlineFeatures, featureName)
				log.Info().Msgf("Added offline feature for validation: %s", featureName)
			case "RTP_FEATURE", "PARENT_RTP_FEATURE":
				fullFeature := entity + ":" + gf
				pricingFeaturesByEntity[entity] = append(pricingFeaturesByEntity[entity], fullFeature)
				log.Info().Msgf("Added pricing feature for validation - entity: %s, full feature: %s", entity, fullFeature)
			case "DEFAULT_FEATURE", "PARENT_DEFAULT_FEATURE", "MODEL_FEATURE", "CALIBRATION":
				log.Info().Msgf("Skipping API validation for feature type %s: %s (no validation required)", featureType, feature)
				continue
			default:
				log.Warn().Msgf("Unknown feature type %s for feature: %s", featureType, feature)
			}
		}
	}

	for entity, features := range onlineFeaturesByEntity {
		if err := p.validateOnlineFeatures(entity, features, authToken); err != nil {
			return fmt.Errorf("online feature validation failed for entity %s: %w", entity, err)
		}
	}

	if len(offlineFeatures) > 0 {
		if err := p.validateOfflineFeatures(offlineFeatures, authToken); err != nil {
			return fmt.Errorf("offline feature validation failed: %w", err)
		}
	}

	for entity, features := range pricingFeaturesByEntity {
		if err := p.validatePricingFeatures(entity, features); err != nil {
			return fmt.Errorf("pricing feature validation failed for entity %s: %w", entity, err)
		}
	}

	return nil
}

// validateOnlineFeatures validates online features for a specific entity
func (p *Predator) validateOnlineFeatures(entity string, features []string, token string) error {
	response, err := p.featureValidationClient.ValidateOnlineFeatures(entity, token)
	if err != nil {
		return fmt.Errorf("failed to call online validation API: %w", err)
	}

	for _, feature := range features {
		if !externalcall.ValidateFeatureExists(feature, response) {
			return fmt.Errorf("online feature '%s' does not exist for entity '%s'", feature, entity)
		}
	}

	log.Info().Msgf("Successfully validated %d online features for entity %s", len(features), entity)
	return nil
}

// validateOfflineFeatures validates offline features by checking online mapping
func (p *Predator) validateOfflineFeatures(features []string, token string) error {
	response, err := p.featureValidationClient.ValidateOfflineFeatures(features, token)
	if err != nil {
		return fmt.Errorf("failed to call offline validation API: %w", err)
	}

	if response.Error != "" {
		return fmt.Errorf("offline validation API returned error: %s", response.Error)
	}

	for _, feature := range features {
		if _, exists := response.Data[feature]; !exists {
			return fmt.Errorf("offline feature '%s' does not have an online mapping", feature)
		}
	}

	log.Info().Msgf("Successfully validated %d offline features", len(features))
	return nil
}

// validatePricingFeatures validates pricing features for a specific entity
func (p *Predator) validatePricingFeatures(entity string, features []string) error {
	if !pred.IsMeeshoEnabled {
		return nil
	}
	response, err := externalcall.PricingClient.GetDataTypes(entity)
	if err != nil {
		return fmt.Errorf("failed to call pricing service API: %w", err)
	}

	for _, feature := range features {
		if !externalcall.ValidatePricingFeatureExists(feature, response) {
			return fmt.Errorf("pricing feature '%s' does not exist for entity '%s'", feature, entity)
		}
	}

	log.Info().Msgf("Successfully validated %d pricing features for entity %s", len(features), entity)
	return nil
}

// extractModelName extracts model name from metadata
func (p *Predator) extractModelName(metadata interface{}) (string, error) {
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var metadataMap map[string]interface{}
	if err := json.Unmarshal(metadataBytes, &metadataMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	modelName, exists := metadataMap["model_name"]
	if !exists {
		return "", fmt.Errorf("model_name not found in metadata")
	}

	modelNameStr, ok := modelName.(string)
	if !ok || modelNameStr == "" {
		return "", fmt.Errorf("model_name must be a non-empty string")
	}

	return modelNameStr, nil
}

// syncFullModel syncs all model files for full upload
func (p *Predator) syncFullModel(gcsPath, destBucket, destPath, modelName string) error {
	log.Info().Msgf("Syncing full model from GCS path: %s", gcsPath)

	srcBucket, srcPath := extractGCSPath(gcsPath)
	if srcBucket == "" || srcPath == "" {
		return fmt.Errorf("invalid GCS path format: %s", gcsPath)
	}

	pathSegments := strings.Split(strings.TrimSuffix(srcPath, "/"), "/")
	srcModelName := pathSegments[len(pathSegments)-1]
	srcBasePath := strings.TrimSuffix(srcPath, "/"+srcModelName)

	log.Info().Msgf("Full upload: transferring all files from %s/%s to %s/%s",
		srcBucket, srcPath, destBucket, destPath)

	return p.GcsClient.TransferFolder(srcBucket, srcBasePath, srcModelName,
		destBucket, strings.TrimSuffix(destPath, "/"+modelName), modelName)
}

// syncPartialFiles syncs only config.pbtxt for partial upload
func (p *Predator) syncPartialFiles(gcsPath, destBucket, destPath, modelName string) error {
	srcBucket, srcPath := extractGCSPath(gcsPath)
	if srcBucket == "" || srcPath == "" {
		return fmt.Errorf("invalid GCS path format: %s", gcsPath)
	}

	filesToSync := []string{"config.pbtxt"}
	log.Info().Msgf("Partial upload: syncing %v for model %s", filesToSync, modelName)

	for _, fileName := range filesToSync {
		srcFilePath := path.Join(srcPath, fileName)
		destFilePath := path.Join(destPath, fileName)

		data, err := p.GcsClient.ReadFile(srcBucket, srcFilePath)
		if err != nil {
			return fmt.Errorf("required file %s not found in source %s/%s: %w",
				fileName, srcBucket, srcFilePath, err)
		}

		if err := p.GcsClient.UploadFile(destBucket, destFilePath, data); err != nil {
			return fmt.Errorf("failed to upload %s: %w", fileName, err)
		}

		log.Info().Msgf("Successfully synced %s for partial upload of model %s", fileName, modelName)
	}

	return nil
}

// validateModelConfiguration validates the model configuration
func (p *Predator) validateModelConfiguration(gcsPath string) error {
	log.Info().Msgf("Validating model configuration for GCS path: %s", gcsPath)

	srcBucket, srcPath := extractGCSPath(gcsPath)
	if srcBucket == "" || srcPath == "" {
		return fmt.Errorf("invalid GCS path format: %s", gcsPath)
	}

	configPath := path.Join(srcPath, configFile)
	configData, err := p.GcsClient.ReadFile(srcBucket, configPath)
	if err != nil {
		return fmt.Errorf("failed to read config.pbtxt from %s/%s: %w", srcBucket, configPath, err)
	}

	var modelConfig modelconfig.ModelConfig
	if err := prototext.Unmarshal(configData, &modelConfig); err != nil {
		return fmt.Errorf("failed to parse config.pbtxt as proto: %w", err)
	}

	log.Info().Msgf("Parsed model config - Name: %s, Backend: %s", modelConfig.Name, modelConfig.Backend)
	return nil
}

// cleanEnsembleScheduling cleans up ensemble scheduling to avoid storing {"step": null}
func (p *Predator) cleanEnsembleScheduling(metadata MetaData) MetaData {
	if len(metadata.Ensembling.Step) == 0 {
		metadata.Ensembling = Ensembling{Step: nil}
	}
	return metadata
}

// copyConfigToNewNameInConfigSource copies config from old model name to new in config source
func (p *Predator) copyConfigToNewNameInConfigSource(oldModelName, newModelName string) error {
	if oldModelName == newModelName {
		return nil
	}

	if pred.GcsConfigBucket == "" || pred.GcsConfigBasePath == "" {
		log.Warn().Msg("Config source not configured, skipping config.pbtxt copy in config source")
		return nil
	}

	destConfigPath := path.Join(pred.GcsConfigBasePath, newModelName, configFile)
	exists, err := p.GcsClient.CheckFileExists(pred.GcsConfigBucket, destConfigPath)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to check if config exists for %s, will attempt copy anyway", newModelName)
	} else if exists {
		log.Info().Msgf("Config already exists for %s in config source, skipping copy", newModelName)
		return nil
	}

	srcConfigPath := path.Join(pred.GcsConfigBasePath, oldModelName, configFile)

	configData, err := p.GcsClient.ReadFile(pred.GcsConfigBucket, srcConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read config.pbtxt from %s: %w", srcConfigPath, err)
	}

	updatedConfigData, err := externalcall.ReplaceModelNameInConfig(configData, newModelName)
	if err != nil {
		return fmt.Errorf("failed to replace model name in config: %w", err)
	}
	if err := p.GcsClient.UploadFile(pred.GcsConfigBucket, destConfigPath, updatedConfigData); err != nil {
		return fmt.Errorf("failed to upload config.pbtxt to %s: %w", destConfigPath, err)
	}

	log.Info().Msgf("Successfully copied config.pbtxt from %s to %s in config source",
		oldModelName, newModelName)
	return nil
}

// replaceModelNameInConfigPreservingFormat updates only the top-level model name while preserving formatting
func (p *Predator) replaceModelNameInConfigPreservingFormat(data []byte, destModelName string) []byte {
	content := string(data)
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "name:") {
			leadingWhitespace := len(line) - len(strings.TrimLeft(line, " \t"))
			if leadingWhitespace >= 2 {
				continue
			}

			namePattern := regexp.MustCompile(`name\s*:\s*"([^"]+)"`)
			matches := namePattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				oldModelName := matches[1]
				loc := namePattern.FindStringIndex(line)
				if loc != nil {
					before := line[:loc[0]]
					matched := line[loc[0]:loc[1]]
					after := line[loc[1]:]
					valuePattern := regexp.MustCompile(`"([^"]+)"`)
					valueReplaced := valuePattern.ReplaceAllString(matched, fmt.Sprintf(`"%s"`, destModelName))
					lines[i] = before + valueReplaced + after
				} else {
					lines[i] = namePattern.ReplaceAllString(line, fmt.Sprintf(`name: "%s"`, destModelName))
				}
				log.Info().Msgf("Replacing top-level model name in config.pbtxt: '%s' -> '%s'", oldModelName, destModelName)
				break
			}
		}
	}

	return []byte(strings.Join(lines, "\n"))
}
