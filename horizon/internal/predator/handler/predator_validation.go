package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	pred "github.com/Meesho/BharatMLStack/horizon/internal/predator"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/predatorconfig"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/predatorrequest"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/validationjob"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

func (p *Predator) ValidateDeleteRequest(predatorConfigList []predatorconfig.PredatorConfig, ids []int) (bool, error) {
	if len(predatorConfigList) != len(ids) {
		log.Error().Err(errors.New(errModelNotFound)).Msgf("model not found for ids %v", ids)
		return false, errors.New(errModelNotFound)
	}

	// Create maps for quick lookup
	requestedModelMap := make(map[string]predatorconfig.PredatorConfig)          // modelName -> config
	requestedDeployableMap := make(map[int]bool)                                 // serviceDeployableID -> exists
	deployableModelMap := make(map[int]map[string]predatorconfig.PredatorConfig) // deployableID -> modelName -> config

	// Build maps from requested models
	for _, predatorConfig := range predatorConfigList {
		// Get service deployable ID for this model
		discoveryConfig, err := p.ServiceDiscoveryRepo.GetById(predatorConfig.DiscoveryConfigID)
		if err != nil {
			log.Error().Err(err).Msgf("failed to fetch discovery config for model %s", predatorConfig.ModelName)
			return false, fmt.Errorf(errFailedToFetchDiscoveryConfigForModel, predatorConfig.ModelName, err)
		}

		requestedModelMap[predatorConfig.ModelName] = predatorConfig
		requestedDeployableMap[discoveryConfig.ServiceDeployableID] = true

		// Group models by deployable
		if deployableModelMap[discoveryConfig.ServiceDeployableID] == nil {
			deployableModelMap[discoveryConfig.ServiceDeployableID] = make(map[string]predatorconfig.PredatorConfig)
		}
		deployableModelMap[discoveryConfig.ServiceDeployableID][predatorConfig.ModelName] = predatorConfig
	}

	// Check for duplicate model names within same deployable
	for deployableID, models := range deployableModelMap {
		if len(models) > 1 {
			// Check if any model names are duplicated within this deployable
			modelNameCount := make(map[string]int)
			for modelName := range models {
				modelNameCount[modelName]++
			}
			for modelName, count := range modelNameCount {
				if count > 1 {
					return false, fmt.Errorf(errDuplicateModelNameInDeployable, modelName, deployableID)
				}
			}
		}
	}

	// Validate ensemble-child group deletion requirements
	if err := p.validateEnsembleChildGroupDeletion(requestedModelMap, deployableModelMap); err != nil {
		return false, err
	}

	return true, nil
}

func (p *Predator) validateEnsembleChildGroupDeletion(requestedModelMap map[string]predatorconfig.PredatorConfig, deployableModelMap map[int]map[string]predatorconfig.PredatorConfig) error {
	// Get all active models to check for ensemble relationships
	allModels, err := p.PredatorConfigRepo.FindAllActiveConfig()
	if err != nil {
		log.Error().Err(err).Msgf("failed to fetch all active models")
		return fmt.Errorf("failed to fetch all active models: %w", err)
	}

	// Group all models by deployable for easier lookup
	allModelsByDeployable := make(map[int]map[string]predatorconfig.PredatorConfig)
	for _, model := range allModels {
		discoveryConfig, err := p.ServiceDiscoveryRepo.GetById(model.DiscoveryConfigID)
		if err != nil {
			log.Error().Err(err).Msgf("failed to fetch discovery config for model %s", model.ModelName)
			continue
		}

		if allModelsByDeployable[discoveryConfig.ServiceDeployableID] == nil {
			allModelsByDeployable[discoveryConfig.ServiceDeployableID] = make(map[string]predatorconfig.PredatorConfig)
		}
		allModelsByDeployable[discoveryConfig.ServiceDeployableID][model.ModelName] = model
	}

	// Check each deployable for ensemble-child relationships
	for deployableID, modelsInDeployable := range allModelsByDeployable {
		requestedModelsInDeployable := deployableModelMap[deployableID]
		if requestedModelsInDeployable == nil {
			continue // No models from this deployable in the delete request
		}

		// Check each model in this deployable
		for modelName, model := range modelsInDeployable {
			var metadata MetaData
			if err := json.Unmarshal(model.MetaData, &metadata); err != nil {
				log.Error().Err(err).Msgf("failed to unmarshal metadata for model %s", modelName)
				continue
			}

			// Check if this is an ensemble model
			if len(metadata.Ensembling.Step) > 0 {
				// This is an ensemble model
				isEnsembleInRequest := requestedModelsInDeployable[modelName].ID != 0

				// Check each child of this ensemble
				for _, step := range metadata.Ensembling.Step {
					childModelName := step.ModelName
					isChildInRequest := requestedModelsInDeployable[childModelName].ID != 0

					// If ensemble is in request, all children must be in request
					if isEnsembleInRequest && !isChildInRequest {
						return fmt.Errorf(errEnsembleMissingChild, modelName, childModelName)
					}

					// If child is in request, ensemble must be in request
					if isChildInRequest && !isEnsembleInRequest {
						return fmt.Errorf(errChildMissingEnsemble, childModelName, modelName)
					}
				}
			}
		}
	}

	return nil
}

// releaseLockWithError is a helper function to release lock and log error
func (p *Predator) releaseLockWithError(lockID uint, groupID, errorMsg string) {
	if releaseErr := p.validationLockRepo.ReleaseLock(lockID); releaseErr != nil {
		log.Error().Err(releaseErr).Msgf("Failed to release validation lock for group ID %s after error: %s", groupID, errorMsg)
	}
	log.Error().Msgf("Validation failed for group ID %s: %s", groupID, errorMsg)
}

// getTestDeployableID resolves the test deployable ID: tries DB lookup by node pool when available;
// if not found or no node pool, falls back to env-based ID by machine type (CPU: TEST_DEPLOYABLE_ID, GPU: TEST_GPU_DEPLOYABLE_ID).
func (p *Predator) getTestDeployableID(payload *Payload) (int, error) {
	if payload == nil {
		return 0, fmt.Errorf("payload is required")
	}
	if payload.ConfigMapping.ServiceDeployableID == 0 {
		return 0, fmt.Errorf("service_deployable_id is required in config_mapping")
	}

	targetDeployableID := int(payload.ConfigMapping.ServiceDeployableID)
	serviceDeployable, err := p.ServiceDeployableRepo.GetById(targetDeployableID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch service deployable %d: %w", targetDeployableID, err)
	}

	if len(serviceDeployable.Config) == 0 {
		return 0, fmt.Errorf("target deployable %d has no config; cannot determine machine type", targetDeployableID)
	}

	var deployableConfig PredatorDeployableConfig
	if err := json.Unmarshal(serviceDeployable.Config, &deployableConfig); err != nil {
		return 0, fmt.Errorf("failed to parse service deployable config %d: %w", targetDeployableID, err)
	}

	return p.resolveTestDeployableID(deployableConfig)
}

// resolveTestDeployableID resolves test deployable ID from deployableConfig: node-pool lookup first, then machine-type fallback.
func (p *Predator) resolveTestDeployableID(deployableConfig PredatorDeployableConfig) (int, error) {
	var testID int
	nodePool := strings.TrimSpace(deployableConfig.NodeSelectorValue)
	if nodePool != "" {
		id, lookupErr := p.ServiceDeployableRepo.GetTestDeployableIDByNodePool(nodePool)
		if lookupErr == nil {
			testID = id
			log.Info().Msgf("Using test deployable ID %d for node pool %s", testID, nodePool)
		} else if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			log.Info().Str("nodePool", nodePool).Msgf("no test deployable for node pool %q (deployable_type=test, config.nodeSelectorValue=%q), using machine-type fallback", nodePool, nodePool)
		} else {
			log.Info().Err(lookupErr).Str("nodePool", nodePool).Msg("Test deployable lookup by node pool failed, using machine-type fallback")
		}
	}

	if testID == 0 {
		switch strings.ToUpper(deployableConfig.MachineType) {
		case "CPU":
			testID = pred.TestDeployableID
			log.Info().Msgf("Using CPU fallback test deployable ID: %d", testID)
		case "GPU":
			testID = pred.TestGpuDeployableID
			log.Info().Msgf("Using GPU fallback test deployable ID: %d", testID)
		default:
			testID = pred.TestDeployableID
			log.Warn().Msgf("Unknown machine type %q, defaulting to CPU fallback test deployable ID: %d",
				deployableConfig.MachineType, testID)
		}
	}

	if testID <= 0 {
		return 0, fmt.Errorf("invalid test deployable ID (not configured or not found); check TEST_DEPLOYABLE_ID (CPU), TEST_GPU_DEPLOYABLE_ID or deployable_type=test for node pool (GPU)")
	}
	return testID, nil
}

// getServiceNameFromDeployable extracts service name from deployable configuration
func (p *Predator) getServiceNameFromDeployable(deployableID int) (string, error) {
	serviceDeployable, err := p.ServiceDeployableRepo.GetById(deployableID)
	if err != nil {
		return "", fmt.Errorf("failed to get deployable config: %w", err)
	}
	return serviceDeployable.Name, nil
}

// performAsyncValidation performs the actual validation process asynchronously
func (p *Predator) performAsyncValidation(job *validationjob.Table, requests []predatorrequest.PredatorRequest, payload *Payload, testDeployableID int) {
	defer func() {
		// Always release the lock when validation completes
		if releaseErr := p.validationLockRepo.ReleaseLock(job.LockID); releaseErr != nil {
			log.Error().Err(releaseErr).Msgf("Failed to release validation lock for job %d", job.ID)
		}
		log.Info().Msgf("Released validation lock for job %d", job.ID)
	}()

	log.Info().Msgf("Starting async validation for job %d, group %s", job.ID, job.GroupID)

	// Step 1: Clear temporary deployable
	if err := p.clearTemporaryDeployable(testDeployableID); err != nil {
		log.Error().Err(err).Msg("Failed to clear temporary deployable")
		p.failValidationJob(job.ID, "Failed to clear temporary deployable: "+err.Error())
		return
	}

	// Step 2: Copy existing models to temporary deployable
	targetDeployableID := int(payload.ConfigMapping.ServiceDeployableID)
	if err := p.copyExistingModelsToTemporary(targetDeployableID, testDeployableID); err != nil {
		log.Error().Err(err).Msg("Failed to copy existing models to temporary deployable")
		p.failValidationJob(job.ID, "Failed to copy existing models: "+err.Error())
		return
	}

	// Step 3: Copy new models from request to temporary deployable
	if err := p.copyRequestModelsToTemporary(requests, testDeployableID); err != nil {
		log.Error().Err(err).Msg("Failed to copy request models to temporary deployable")
		p.failValidationJob(job.ID, "Failed to copy request models: "+err.Error())
		return
	}

	// Step 4: Restart temporary deployable
	if err := p.restartTemporaryDeployable(testDeployableID); err != nil {
		log.Error().Err(err).Msg("Failed to restart temporary deployable")
		p.failValidationJob(job.ID, "Failed to restart temporary deployable: "+err.Error())
		return
	}

	// Update job status to checking and record restart time
	now := time.Now()
	if err := p.validationJobRepo.UpdateStatus(job.ID, validationjob.StatusChecking, ""); err != nil {
		log.Error().Err(err).Msgf("Failed to update job %d status to checking", job.ID)
	}

	// Update restart time in the job
	job.RestartedAt = &now
	job.Status = validationjob.StatusChecking

	// Step 5: Start health checking process
	p.startHealthCheckingProcess(job)
}

// startHealthCheckingProcess monitors the deployment health and updates validation status
func (p *Predator) startHealthCheckingProcess(job *validationjob.Table) {
	log.Info().Msgf("Starting health check process for job %d, service %s", job.ID, job.ServiceName)

	for job.HealthCheckCount < job.MaxHealthChecks {
		// Wait for the specified interval before checking
		time.Sleep(time.Duration(job.HealthCheckInterval) * time.Second)

		// Increment health check count
		if err := p.validationJobRepo.IncrementHealthCheck(job.ID); err != nil {
			log.Error().Err(err).Msgf("Failed to increment health check count for job %d", job.ID)
		}
		job.HealthCheckCount++

		// Check deployment health using infrastructure handler
		isHealthy, err := p.checkDeploymentHealth(job.ServiceName)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to check deployment health for job %d", job.ID)
			continue // Continue checking, don't fail immediately on health check errors
		}

		if isHealthy {
			log.Info().Msgf("Deployment is healthy for job %d, validation successful", job.ID)
			p.completeValidationJob(job.ID, true, "Deployment is healthy and running successfully")
			p.updateRequestValidationStatus(job.GroupID, true)
			return
		}

		log.Info().Msgf("Deployment not yet healthy for job %d, check %d/%d", job.ID, job.HealthCheckCount, job.MaxHealthChecks)
	}

	// If we reach here, max health checks exceeded
	log.Warn().Msgf("Max health checks exceeded for job %d, marking as failed", job.ID)
	p.completeValidationJob(job.ID, false, fmt.Sprintf("Deployment failed to become healthy after %d checks", job.MaxHealthChecks))
	p.updateRequestValidationStatus(job.GroupID, false)
}

// checkDeploymentHealth checks if the deployment is healthy using infrastructure handler
func (p *Predator) checkDeploymentHealth(serviceName string) (bool, error) {
	resourceDetail, err := p.infrastructureHandler.GetResourceDetail(serviceName, p.workingEnv)
	if err != nil {
		return false, fmt.Errorf("failed to get resource detail: %w", err)
	}

	if resourceDetail == nil || len(resourceDetail.Nodes) == 0 {
		return false, nil
	}

	healthyPodCount := 0
	totalPodCount := 0

	for _, node := range resourceDetail.Nodes {
		if node.Kind == "Deployment" {
			totalPodCount++
			if node.Health.Status == "Healthy" {
				healthyPodCount++
			}
		}
	}

	log.Info().Msgf("Health check for service %s: %d/%d pods healthy and running", serviceName, healthyPodCount, totalPodCount)

	if totalPodCount == healthyPodCount {
		return true, nil
	}
	// Consider deployment healthy if at least one pod is healthy and running
	return false, nil
}

// failValidationJob marks a validation job as failed
func (p *Predator) failValidationJob(jobID uint, errorMessage string) {
	if err := p.validationJobRepo.UpdateValidationResult(jobID, false, errorMessage); err != nil {
		log.Error().Err(err).Msgf("Failed to update validation job %d as failed", jobID)
	}
}

// completeValidationJob marks a validation job as completed
func (p *Predator) completeValidationJob(jobID uint, success bool, message string) {
	if err := p.validationJobRepo.UpdateValidationResult(jobID, success, message); err != nil {
		log.Error().Err(err).Msgf("Failed to update validation job %d as completed", jobID)
	}
}

// updateRequestValidationStatus updates the request table with validation results
func (p *Predator) updateRequestValidationStatus(groupID string, success bool) {
	id, err := strconv.ParseUint(groupID, 10, 32)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to parse group ID %s for status update", groupID)
		return
	}

	requests, err := p.Repo.GetAllByGroupID(uint(id))
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get requests for group ID %s", groupID)
		return
	}

	// Update all requests in the group
	for _, request := range requests {
		request.UpdatedAt = time.Now()
		request.IsValid = success
		if !success {
			request.RejectReason = "Validation Failed"
			request.Status = statusRejected
			request.UpdatedBy = "Validation Job"
			request.UpdatedAt = time.Now()
			request.Active = false
		}
		if err := p.Repo.Update(&request); err != nil {
			log.Error().Err(err).Msgf("Failed to update request %d status", request.RequestID)
		} else {
			log.Info().Msgf("Updated request %d status to %s", request.RequestID, request.Status)
		}
	}
}

// clearTemporaryDeployable clears all models from the temporary deployable GCS path
func (p *Predator) clearTemporaryDeployable(testDeployableID int) error {
	// Get temporary deployable config
	testServiceDeployable, err := p.ServiceDeployableRepo.GetById(testDeployableID)
	if err != nil {
		return fmt.Errorf("failed to fetch temporary service deployable: %w", err)
	}

	var tempDeployableConfig PredatorDeployableConfig
	if err := json.Unmarshal(testServiceDeployable.Config, &tempDeployableConfig); err != nil {
		return fmt.Errorf("failed to parse temporary deployable config: %w", err)
	}

	if tempDeployableConfig.GCSBucketPath != "NA" {
		// Extract bucket and path from temporary deployable config
		tempBucket, tempPath := extractGCSPath(strings.TrimSuffix(tempDeployableConfig.GCSBucketPath, "/*"))

		// Clear all models from temporary deployable
		log.Info().Msgf("Clearing temporary deployable GCS path: gs://%s/%s", tempBucket, tempPath)
		if err := p.GcsClient.DeleteFolder(tempBucket, tempPath, ""); err != nil {
			return fmt.Errorf("failed to clear temporary deployable GCS path: %w", err)
		}
	}

	return nil
}

// copyExistingModelsToTemporary copies all existing models from target deployable to temporary deployable
func (p *Predator) copyExistingModelsToTemporary(targetDeployableID, tempDeployableID int) error {
	// Get target deployable config
	targetServiceDeployable, err := p.ServiceDeployableRepo.GetById(targetDeployableID)
	if err != nil {
		return fmt.Errorf("failed to fetch target service deployable: %w", err)
	}

	var targetDeployableConfig PredatorDeployableConfig
	if err := json.Unmarshal(targetServiceDeployable.Config, &targetDeployableConfig); err != nil {
		return fmt.Errorf("failed to parse target deployable config: %w", err)
	}

	// Get temporary deployable config
	tempServiceDeployable, err := p.ServiceDeployableRepo.GetById(tempDeployableID)
	if err != nil {
		return fmt.Errorf("failed to fetch temporary service deployable: %w", err)
	}

	var tempDeployableConfig PredatorDeployableConfig
	if err := json.Unmarshal(tempServiceDeployable.Config, &tempDeployableConfig); err != nil {
		return fmt.Errorf("failed to parse temporary deployable config: %w", err)
	}

	if targetDeployableConfig.GCSBucketPath != "NA" {
		// Extract GCS paths
		targetBucket, targetPath := extractGCSPath(strings.TrimSuffix(targetDeployableConfig.GCSBucketPath, "/*"))
		tempBucket, tempPath := extractGCSPath(strings.TrimSuffix(tempDeployableConfig.GCSBucketPath, "/*"))

		// Copy all existing models from target to temporary deployable
		return p.copyAllModelsFromActualToStaging(targetBucket, targetPath, tempBucket, tempPath)
	} else {
		return nil
	}
}

// copyRequestModelsToTemporary copies the requested models to temporary deployable
func (p *Predator) copyRequestModelsToTemporary(requests []predatorrequest.PredatorRequest, tempDeployableID int) error {
	// Get temporary deployable config
	tempServiceDeployable, err := p.ServiceDeployableRepo.GetById(tempDeployableID)
	if err != nil {
		return fmt.Errorf("failed to fetch temporary service deployable: %w", err)
	}

	var tempDeployableConfig PredatorDeployableConfig
	if err := json.Unmarshal(tempServiceDeployable.Config, &tempDeployableConfig); err != nil {
		return fmt.Errorf("failed to parse temporary deployable config: %w", err)
	}

	tempBucket, tempPath := extractGCSPath(strings.TrimSuffix(tempDeployableConfig.GCSBucketPath, "/*"))

	isNotProd := p.isNonProductionEnvironment()

	// Copy each requested model from default GCS location to temporary deployable
	for _, request := range requests {
		modelName := request.ModelName
		payload, err := p.processPayload(request)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to parse payload for request %d", request.RequestID)
			return fmt.Errorf("failed to parse payload for request %d: %w", request.RequestID, err)
		}

		var sourceBucket, sourcePath, sourceModelName string
		if payload.ModelSource != "" {
			sourceBucket, sourcePath, sourceModelName = extractGCSDetails(payload.ModelSource)
			log.Info().Msgf("Using ModelSource from payload for validation: gs://%s/%s/%s",
				sourceBucket, sourcePath, sourceModelName)
		} else {
			sourceBucket = pred.GcsModelBucket
			sourcePath = pred.GcsModelBasePath
			sourceModelName = modelName
			log.Info().Msgf("Using default model source for validation: gs://%s/%s/%s",
				sourceBucket, sourcePath, sourceModelName)
		}
		log.Info().Msgf("Copying model %s from gs://%s/%s/%s to temporary deployable gs://%s/%s",
			modelName, sourceBucket, sourcePath, sourceModelName, tempBucket, tempPath)

		if isNotProd {
			if err := p.GcsClient.TransferFolder(sourceBucket, sourcePath, sourceModelName,
				tempBucket, tempPath, modelName); err != nil {
				return fmt.Errorf("failed to copy requested model %s to temporary deployable: %w", modelName, err)
			}
		} else {
			if err := p.GcsClient.TransferFolderWithSplitSources(
				sourceBucket, sourcePath, pred.GcsConfigBucket, pred.GcsConfigBasePath,
				sourceModelName, tempBucket, tempPath, modelName,
			); err != nil {
				return fmt.Errorf("failed to copy requested model %s to temporary deployable: %w", modelName, err)
			}
		}

		log.Info().Msgf("Successfully copied model %s to temporary deployable", modelName)
	}

	return nil
}

// restartTemporaryDeployable restarts the temporary deployable for validation
func (p *Predator) restartTemporaryDeployable(tempDeployableID int) error {
	tempServiceDeployable, err := p.ServiceDeployableRepo.GetById(tempDeployableID)
	if err != nil {
		return fmt.Errorf("failed to fetch temporary service deployable: %w", err)
	}

	// Extract isCanary from deployable config
	var deployableConfig map[string]interface{}
	isCanary := false
	if err := json.Unmarshal(tempServiceDeployable.Config, &deployableConfig); err == nil {
		if strategy, ok := deployableConfig["deploymentStrategy"].(string); ok && strategy == "canary" {
			isCanary = true
		}
	}
	if err := p.infrastructureHandler.RestartDeployment(tempServiceDeployable.Name, p.workingEnv, isCanary); err != nil {
		return fmt.Errorf("failed to restart temporary deployable: %w", err)
	}

	log.Info().Msgf("Successfully restarted temporary deployable: %s for validation", tempServiceDeployable.Name)
	return nil
}
