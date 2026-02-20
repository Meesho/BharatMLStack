package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	pred "github.com/Meesho/BharatMLStack/horizon/internal/predator"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/discoveryconfig"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/predatorconfig"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/predatorrequest"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/prototext"
	"gorm.io/gorm"
	"github.com/Meesho/BharatMLStack/horizon/internal/predator/proto/protogen"
)

func (p *Predator) processRequest(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) {
	if req.Status == statusApproved {
		switch predatorRequestList[0].RequestType {
		case OnboardRequestType:
			p.processOnboardFlow(requestIdPayloadMap, predatorRequestList, req)
		case ScaleUpRequestType:
			p.processScaleUpFlow(requestIdPayloadMap, predatorRequestList, req)
		case PromoteRequestType:
			p.processPromoteFlow(requestIdPayloadMap, predatorRequestList, req)
		case DeleteRequestType:
			p.processDeleteRequest(requestIdPayloadMap, predatorRequestList, req)
		case EditRequestType:
			p.processEditRequest(requestIdPayloadMap, predatorRequestList, req)
		default:
			log.Error().Err(errors.New(errInvalidRequestType)).Msg(errInvalidRequestType)
		}
	} else {
		p.processRejectRequest(predatorRequestList, req)
	}
}

func (p *Predator) processRejectRequest(predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) {
	for i := range predatorRequestList {
		predatorRequestList[i].Status = statusRejected
		predatorRequestList[i].RejectReason = req.RejectReason
		predatorRequestList[i].Reviewer = req.ApprovedBy
		predatorRequestList[i].UpdatedBy = req.ApprovedBy
		predatorRequestList[i].UpdatedAt = time.Now()
		predatorRequestList[i].Active = false
	}

	if err := p.Repo.UpdateMany(predatorRequestList); err != nil {
		log.Printf(errFailedToUpdateRequestStatusAndStage, err)
	}

	log.Printf("Request %s rejected successfully.\n", req.GroupID)
}

func (p *Predator) processDeleteRequest(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) {
	transferredGcsModelData, err := p.processGCSCloneToDeleteBucket(req.ApprovedBy, predatorRequestList, requestIdPayloadMap)
	if err != nil {
		log.Error().Err(err).Msg(errFailedToOperateGcsCloneStage)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		p.revertForDelete(transferredGcsModelData)
		return
	}

	p.processDBPopulationStageForDelete(predatorRequestList, requestIdPayloadMap, req)

	if err := p.processRestartDeployableStage(req.ApprovedBy, predatorRequestList, requestIdPayloadMap); err != nil {
		log.Error().Err(err).Msg(errFailedToRestartDeployable)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageRestartDeployable)
		return
	}

}

func (p *Predator) processEditRequest(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) {
	log.Info().Msgf("Starting edit request flow for group ID: %s", req.GroupID)

	// Step 1: Get target deployable configuration from the request
	targetDeployableID := int(requestIdPayloadMap[predatorRequestList[0].RequestID].ConfigMapping.ServiceDeployableID)
	targetServiceDeployable, err := p.ServiceDeployableRepo.GetById(targetDeployableID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch target service deployable for edit request")
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		return
	}

	var targetDeployableConfig PredatorDeployableConfig
	if err := json.Unmarshal(targetServiceDeployable.Config, &targetDeployableConfig); err != nil {
		log.Error().Err(err).Msg("Failed to parse target service deployable config")
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		return
	}

	targetBucket, targetPath := extractGCSPath(strings.TrimSuffix(targetDeployableConfig.GCSBucketPath, "/*"))
	log.Info().Msgf("Target deployable path: gs://%s/%s", targetBucket, targetPath)

	// Step 2: GCS Copy Stage - Copy models from source to target deployable path
	transferredGcsModelData, err := p.processEditGCSCopyStage(requestIdPayloadMap, predatorRequestList, targetBucket, targetPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to copy models for edit request")
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		p.revert(transferredGcsModelData)
		return
	}

	// Update stage to DB Population after successful GCS copy
	p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusInProgress, predatorStageDBPopulation)

	// Step 3: DB Update Stage - Update existing predator config with new metadata from request
	err = p.processEditDBUpdateStage(requestIdPayloadMap, predatorRequestList, req.ApprovedBy)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update database for edit request")
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageDBPopulation)
		p.revert(transferredGcsModelData)
		return
	}

	// Update stage to Restart Deployable after successful DB update
	p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusInProgress, predatorStageRestartDeployable)

	// Step 4: Restart Deployable Stage - Restart target deployable
	if err := p.processRestartDeployableStage(req.ApprovedBy, predatorRequestList, requestIdPayloadMap); err != nil {
		log.Error().Err(err).Msg("Failed to restart deployable for edit request")
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageRestartDeployable)
		return
	}

	// Mark request as approved and completed
	p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusApproved, constant.EmptyString)
	log.Info().Msgf("Edit request completed successfully for group ID: %s", req.GroupID)
}

// processEditGCSCopyStage copies models from source to target deployable path for edit approval
func (p *Predator) processEditGCSCopyStage(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, targetBucket, targetPath string) ([]GcsModelData, error) {
	var transferredGcsModelData []GcsModelData

	// Check if we're in the correct stage for GCS copy
	if predatorRequestList[0].RequestStage != predatorStagePending && predatorRequestList[0].RequestStage != predatorStageCloneToBucket && predatorRequestList[0].RequestStage != constant.EmptyString {
		log.Info().Msgf("Skipping GCS copy stage - current stage: %s", predatorRequestList[0].RequestStage)
		return transferredGcsModelData, nil
	}

	isNotProd := p.isNonProductionEnvironment()

	for _, requestModel := range predatorRequestList {
		payload := requestIdPayloadMap[requestModel.RequestID]
		if payload == nil {
			log.Error().Msgf("Payload not found for request ID %d", requestModel.RequestID)
			continue
		}

		modelName := requestModel.ModelName

		// Use the source path from the payload, not the default GCS bucket
		if payload.ModelSource == "" {
			log.Error().Msgf("ModelSource is empty for request ID %d", requestModel.RequestID)
			return transferredGcsModelData, fmt.Errorf("model source path is empty for model %s", modelName)
		}

		// Normalize GCS URL (handle gcs:// prefix)
		normalizedModelSource := payload.ModelSource
		if strings.HasPrefix(normalizedModelSource, "gcs://") {
			normalizedModelSource = strings.Replace(normalizedModelSource, "gcs://", "gs://", 1)
			log.Info().Msgf("Normalized GCS URL from %s to %s", payload.ModelSource, normalizedModelSource)
		}

		// Parse the source GCS path
		sourceBucket, sourcePath := extractGCSPath(normalizedModelSource)
		if sourceBucket == "" || sourcePath == "" {
			log.Error().Msgf("Invalid source GCS path format: %s (normalized: %s)", payload.ModelSource, normalizedModelSource)
			return transferredGcsModelData, fmt.Errorf("invalid source GCS path format: %s", normalizedModelSource)
		}

		log.Info().Msgf("Copying model %s from source gs://%s/%s to target gs://%s/%s for edit approval",
			modelName, sourceBucket, sourcePath, targetBucket, targetPath)

		// Copy model from source to target deployable path
		// Extract model folder name from source path and copy to target with the same model name
		pathSegments := strings.Split(strings.TrimSuffix(sourcePath, "/"), "/")
		sourceModelName := pathSegments[len(pathSegments)-1]
		sourceBasePath := strings.TrimSuffix(sourcePath, "/"+sourceModelName)

		if isNotProd {
			if err := p.GcsClient.TransferFolder(
				sourceBucket, sourceBasePath, sourceModelName,
				targetBucket, targetPath, modelName,
			); err != nil {
				return transferredGcsModelData, err
			}
		} else {
			configBucket := pred.GcsConfigBucket
			configPath := pred.GcsConfigBasePath
			if configBucket != "" && configPath != "" && payload.MetaData.InstanceCount > 0 {
				if err := p.updateInstanceCountInConfigSource(configBucket, configPath, modelName, payload.MetaData.InstanceCount); err != nil {
					log.Error().Err(err).Msgf("Failed to update instance count in config-source for model %s", modelName)
					return transferredGcsModelData, err
				}
			}
			if err := p.GcsClient.TransferFolderWithSplitSources(
				sourceBucket, sourceBasePath, configBucket, configPath,
				sourceModelName, targetBucket, targetPath, modelName,
			); err != nil {
				return transferredGcsModelData, err
			}
		}

		// Track transferred data for potential rollback
		transferredGcsModelData = append(transferredGcsModelData, GcsModelData{
			Bucket: targetBucket,
			Path:   targetPath,
			Name:   modelName,
		})

		log.Info().Msgf("Successfully copied model %s for edit approval", modelName)
	}

	return transferredGcsModelData, nil
}

func (p *Predator) updateInstanceCountInConfigSource(bucket, basePath, modelName string, instanceCount int) error {
	if modelName == "" {
		return fmt.Errorf("model name is empty, required to update instance count in config-source")
	}

	configPath := path.Join(basePath, modelName, configFile)
	configData, err := p.GcsClient.ReadFile(bucket, configPath)
	if err != nil {
		return fmt.Errorf("failed to read config.pbtxt from config-source for model %s: %w", modelName, err)
	}

	var modelConfig protogen.ModelConfig
	if err := prototext.Unmarshal(configData, &modelConfig); err != nil {
		return fmt.Errorf("failed to parse config.pbtxt from config-source for model %s: %w", modelName, err)
	}
	if len(modelConfig.InstanceGroup) == 0 {
		return fmt.Errorf("%s (model %s)", errNoInstanceGroup, modelName)
	}

	modelConfig.InstanceGroup[0].Count = int32(instanceCount)

	opts := prototext.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}

	newConfigData, err := opts.Marshal(&modelConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config.pbtxt for model %s: %w", modelName, err)
	}
	if err := p.GcsClient.UploadFile(bucket, configPath, newConfigData); err != nil {
		return fmt.Errorf("failed to upload config.pbtxt to config-source for model %s: %w", modelName, err)
	}

	log.Info().Msgf("Updated instance_count to %d in config-source for model %s", instanceCount, modelName)
	return nil
}

// processEditDBUpdateStage updates predator config for edit approval
// This updates the existing predator config with new config.pbtxt and metadata.json changes
func (p *Predator) processEditDBUpdateStage(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, approvedBy string) error {
	// Check if we're in the correct stage for DB update
	if predatorRequestList[0].RequestStage != predatorStageDBPopulation {
		log.Info().Msgf("Skipping DB update stage - current stage: %s", predatorRequestList[0].RequestStage)
		return nil
	}

	log.Info().Msg("Starting DB update stage for edit approval")

	for _, requestModel := range predatorRequestList {
		payload := requestIdPayloadMap[requestModel.RequestID]
		if payload == nil {
			log.Error().Msgf("Payload not found for request ID %d", requestModel.RequestID)
			continue
		}

		modelName := requestModel.ModelName
		log.Info().Msgf("Updating predator config for model %s", modelName)

		// Find existing predator config for this model
		existingPredatorConfig, err := p.PredatorConfigRepo.GetActiveModelByModelName(modelName)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to fetch existing predator config for model %s", modelName)
			return fmt.Errorf("failed to fetch existing predator config for model %s: %w", modelName, err)
		}

		if existingPredatorConfig == nil {
			log.Error().Msgf("No existing predator config found for model %s", modelName)
			return fmt.Errorf("no existing predator config found for model %s", modelName)
		}

		// Clean up ensemble scheduling and update the predator config with new metadata from the request
		cleanedMetaData := p.cleanEnsembleScheduling(payload.MetaData)

		metaDataBytes, err := json.Marshal(cleanedMetaData)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal metadata for model %s", modelName)
			return fmt.Errorf("failed to marshal metadata for model %s: %w", modelName, err)
		}

		// Update the existing config
		existingPredatorConfig.MetaData = metaDataBytes
		existingPredatorConfig.UpdatedBy = approvedBy
		existingPredatorConfig.UpdatedAt = time.Now()
		existingPredatorConfig.HasNilData = true
		existingPredatorConfig.TestResults = nil
		// Save the updated config
		if err := p.PredatorConfigRepo.Update(existingPredatorConfig); err != nil {
			log.Error().Err(err).Msgf("Failed to update predator config for model %s", modelName)
			return fmt.Errorf("failed to update predator config for model %s: %w", modelName, err)
		}

		log.Info().Msgf("Successfully updated predator config for model %s", modelName)
	}

	log.Info().Msg("DB update stage completed successfully for edit approval")
	return nil
}

func (p *Predator) copyAllModelsFromActualToStaging(sourceBucket, sourcePath, targetBucket, targetPath string) error {
	// List all models in the actual target path and copy them to staging
	folders, err := p.GcsClient.ListFolders(sourceBucket, sourcePath)
	if err != nil {
		return fmt.Errorf("failed to list models in actual target path: %w", err)
	}

	// Copy each model folder from actual target to staging
	for _, modelName := range folders {
		log.Info().Msgf("Copying existing model %s from actual target to staging", modelName)

		if err := p.GcsClient.TransferFolder(sourceBucket, sourcePath, modelName, targetBucket, targetPath, modelName); err != nil {
			log.Error().Err(err).Msgf("Failed to copy existing model %s to staging", modelName)
			return fmt.Errorf("failed to copy existing model %s to staging: %w", modelName, err)
		}
	}

	return nil
}

func (p *Predator) deleteServiceDiscoveryAndConfig(req ApproveRequest, predatorRequestList []predatorrequest.PredatorRequest, requestIdPayloadMap map[uint]*Payload) error {
	tx := p.Repo.DB().Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // re-throw panic after rollback
		}
	}()

	for i := range predatorRequestList {
		payload := requestIdPayloadMap[predatorRequestList[i].RequestID]
		if payload == nil {
			log.Error().Msgf(errFailedToParsePayload)
			tx.Rollback()
			return fmt.Errorf("failed to parse payload for request ID %d", predatorRequestList[i].RequestID)
		}
		discoveryConfigID := int(payload.DiscoveryConfigID)
		log.Info().Msgf("Processing delete request for discovery config ID: %d", discoveryConfigID)
		serviceDiscovery, err := p.ServiceDiscoveryRepo.WithTx(tx).GetById(discoveryConfigID)
		if err != nil {
			log.Error().Err(err).Msg(errFailedToFindServiceDiscovery)
			tx.Rollback()
			return err
		}
		serviceDiscovery.Active = false
		serviceDiscovery.UpdatedAt = time.Now()
		serviceDiscovery.UpdatedBy = req.ApprovedBy

		if err := p.ServiceDiscoveryRepo.WithTx(tx).Update(serviceDiscovery); err != nil {
			log.Error().Err(err).Msg(errFailedToUpdateServiceDiscovery)
			tx.Rollback()
			return err
		}

		predatorConfigs, err := p.PredatorConfigRepo.WithTx(tx).GetByDiscoveryConfigID(discoveryConfigID)
		if err != nil {
			log.Error().Err(err).Msg(errFailedToFindPredatorConfig)
			tx.Rollback()
			return err
		}

		for j := range predatorConfigs {
			predatorConfigs[j].Active = false
			predatorConfigs[j].UpdatedAt = time.Now()
			predatorConfigs[j].UpdatedBy = req.ApprovedBy
			if err := p.PredatorConfigRepo.WithTx(tx).Update(&predatorConfigs[j]); err != nil {
				log.Error().Err(err).Msg(errFailedToUpdatePredatorConfig)
				tx.Rollback()
				return err
			}
		}

		predatorRequestList[i].Status = statusInProgress
		predatorRequestList[i].Reviewer = req.ApprovedBy
		predatorRequestList[i].UpdatedBy = req.ApprovedBy
		predatorRequestList[i].RequestStage = predatorStageRestartDeployable
		predatorRequestList[i].UpdatedAt = time.Now()

		if err := p.Repo.WithTx(tx).Update(&predatorRequestList[i]); err != nil {
			log.Error().Err(err).Msg(errFailedToUpdateRequestStatusAndStage)
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Error().Err(err).Msg("transaction commit failed")
		return err
	}

	return nil
}

func (p *Predator) processGCSCloneToDeleteBucket(email string, predatorRequestList []predatorrequest.PredatorRequest, requestIdPayloadMap map[uint]*Payload) ([]GcsTransferredData, error) {
	var transferredGcsModelData []GcsTransferredData
	if predatorRequestList[0].RequestStage == constant.EmptyString || predatorRequestList[0].RequestStage == predatorStagePending || predatorRequestList[0].RequestStage == predatorStageCloneToBucket {
		for _, requestModel := range predatorRequestList {
			srcBucket, srcPath, srcModelName := extractGCSDetails(requestIdPayloadMap[requestModel.RequestID].ModelSource)
			destBucket, destPath := extractGCSPath(pred.DefaultModelPathKey)
			log.Info().Msgf("srcBucket: %s, srcPath: %s, srcModelName: %s, destBucket: %s, destPath: %s", srcBucket, srcPath, srcModelName, destBucket, destPath)
			if srcBucket == constant.EmptyString || srcPath == constant.EmptyString || srcModelName == constant.EmptyString || destBucket == constant.EmptyString || destPath == constant.EmptyString || requestIdPayloadMap[requestModel.RequestID].ModelName == constant.EmptyString {
				log.Error().Err(errors.New(errModelPathFormat)).Msg(errInvalidGcsBucketPath)
				return transferredGcsModelData, errors.New(errModelPathFormat)
			}

			if err := p.GcsClient.TransferAndDeleteFolder(srcBucket, srcPath, srcModelName, destBucket, destPath, requestIdPayloadMap[requestModel.RequestID].ModelName); err != nil {
				log.Error().Err(err).Msg(errGCSCopyFailed)
				return transferredGcsModelData, err
			}

			transferredGcsModelData = append(transferredGcsModelData, GcsTransferredData{
				SrcBucket:  destBucket,
				SrcPath:    destPath,
				SrcName:    requestIdPayloadMap[requestModel.RequestID].ModelName,
				DestBucket: srcBucket,
				DestPath:   srcPath,
				DestName:   srcModelName,
			})
		}
		p.updateRequestStatusAndStage(email, predatorRequestList, statusInProgress, predatorStageDBPopulation)
	}
	return transferredGcsModelData, nil
}

func (p *Predator) processRestartDeployableStage(email string, predatorRequestList []predatorrequest.PredatorRequest, requestIdPayloadMap map[uint]*Payload) error {
	if predatorRequestList[0].RequestStage != predatorStageRestartDeployable {
		return nil
	}
	var serviceDeployableIDList []int
	for _, requestModel := range predatorRequestList {
		serviceDeployableIDList = append(serviceDeployableIDList, int(requestIdPayloadMap[requestModel.RequestID].ConfigMapping.ServiceDeployableID))
	}

	for _, serviceDeployableID := range serviceDeployableIDList {
		sd, err := p.ServiceDeployableRepo.GetById(int(serviceDeployableID))
		if err != nil {
			log.Error().Err(err).Msg(errFailedToFindServiceDeployableEntry)
			return err
		}
		// Extract isCanary from deployable config
		var deployableConfig map[string]interface{}
		isCanary := false
		if err := json.Unmarshal(sd.Config, &deployableConfig); err == nil {
			if strategy, ok := deployableConfig["deploymentStrategy"].(string); ok && strategy == "canary" {
				isCanary = true
			}
		}
		if err := p.infrastructureHandler.RestartDeployment(sd.Name, p.workingEnv, isCanary); err != nil {
			log.Error().Err(err).Msg(errFailedToRestartDeployable)
			return err
		}
	}

	p.updateRequestStatusAndStage(email, predatorRequestList, statusApproved, constant.EmptyString)
	return nil
}

func (p *Predator) processPayload(predatorRequest predatorrequest.PredatorRequest) (*Payload, error) {
	var payload Payload
	decoder := json.NewDecoder(strings.NewReader(predatorRequest.Payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		log.Error().Err(err).Msg("Failed to parse payload with strict decoding")
		return nil, err
	}
	return &payload, nil
}

func (p *Predator) processGCSCloneStage(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) ([]GcsModelData, error) {
	var transferredGcsModelData []GcsModelData
	if predatorRequestList[0].RequestStage == predatorStagePending || predatorRequestList[0].RequestStage == predatorStageCloneToBucket {
		isNotProd := p.isNonProductionEnvironment()
		for _, requestModel := range predatorRequestList {

			serviceDeployable, err := p.ServiceDeployableRepo.GetById(int(requestIdPayloadMap[requestModel.RequestID].ConfigMapping.ServiceDeployableID))

			if err != nil {
				log.Error().Err(err).Msg(serviceDeployableNotFound)
				return transferredGcsModelData, err
			}

			var deployableConfig PredatorDeployableConfig
			if err := json.Unmarshal(serviceDeployable.Config, &deployableConfig); err != nil {
				log.Error().Err(err).Msg(failedToParseServiceConfig)
				return transferredGcsModelData, err
			}

			destBucket, destPath := extractGCSPath(strings.TrimSuffix(deployableConfig.GCSBucketPath, "/*"))
			destModelName := requestIdPayloadMap[requestModel.RequestID].ModelName

			var srcBucket, srcPath, srcModelName string

			srcBucket = pred.GcsModelBucket
			srcPath = pred.GcsModelBasePath
			if requestModel.RequestType == ScaleUpRequestType {
				srcModelName = destModelName
				log.Info().Msgf("Scale-up: Source from model-source gs://%s/%s/%s",
					srcBucket, srcPath, srcModelName)
			} else {
				_, _, srcModelName = extractGCSDetails(requestIdPayloadMap[requestModel.RequestID].ModelSource)
				log.Info().Msgf("Onboard/Promote: Source from payload gs://%s/%s/%s",
					srcBucket, srcPath, srcModelName)
			}

			log.Info().Msgf("Copying to target deployable - src: %s/%s/%s, dest: %s/%s/%s",
				srcBucket, srcPath, srcModelName, destBucket, destPath, destModelName)

			if srcBucket == constant.EmptyString || srcPath == constant.EmptyString ||
				srcModelName == constant.EmptyString || destBucket == constant.EmptyString ||
				destPath == constant.EmptyString || destModelName == constant.EmptyString {
				log.Error().Err(errors.New(errModelPathFormat)).Msg(errInvalidGcsBucketPath)
				return transferredGcsModelData, errors.New(errModelPathFormat)
			}

			if isNotProd {
				if err := p.GcsClient.TransferFolder(srcBucket, srcPath, srcModelName,
					destBucket, destPath, destModelName); err != nil {
					log.Error().Err(err).Msg(errGCSCopyFailed)
					return transferredGcsModelData, err
				}
			} else {
				if err := p.GcsClient.TransferFolderWithSplitSources(
					srcBucket, srcPath, pred.GcsConfigBucket, pred.GcsConfigBasePath,
					srcModelName, destBucket, destPath, destModelName,
				); err != nil {
					log.Error().Err(err).Msg(errGCSCopyFailed)
					return transferredGcsModelData, err
				}
			}

			transferredGcsModelData = append(transferredGcsModelData, GcsModelData{
				Bucket: destBucket,
				Path:   destPath,
				Name:   requestIdPayloadMap[requestModel.RequestID].ModelName,
			})

			log.Info().Msgf("Successfully copied model to target deployable: %s", destModelName)
		}
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusInProgress, predatorStageDBPopulation)
	}
	return transferredGcsModelData, nil
}

func (p *Predator) processGCSCloneStageIndefaultFolder(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) ([]GcsModelData, error) {
	var transferredGcsModelData []GcsModelData
	if predatorRequestList[0].RequestStage != predatorStagePending &&
		predatorRequestList[0].RequestStage != predatorStageCloneToBucket {
		return transferredGcsModelData, nil
	}

	isNotProd := p.isNonProductionEnvironment()

	for _, requestModel := range predatorRequestList {
		payload := requestIdPayloadMap[requestModel.RequestID]

		destBucket := pred.GcsModelBucket
		destPath := pred.GcsModelBasePath
		destModelName := payload.ModelName

		_, _, originalModelName := extractGCSDetails(payload.ModelSource)
		srcBucket := pred.GcsModelBucket
		srcPath := pred.GcsModelBasePath
		srcModelName := originalModelName

		log.Info().Msgf("Scale-up: Copying within model-source %s → %s:\nsrcBucket: %s, srcPath: %s, srcModelName: %s, destBucket: %s, destPath: %s",
			srcModelName, destModelName, srcBucket, srcPath, srcModelName, destBucket, destPath)

		if srcBucket == constant.EmptyString || srcPath == constant.EmptyString ||
			srcModelName == constant.EmptyString || destBucket == constant.EmptyString ||
			destPath == constant.EmptyString || destModelName == constant.EmptyString {
			log.Error().Err(errors.New(errModelPathFormat)).Msg(errInvalidGcsBucketPath)
			return transferredGcsModelData, errors.New(errModelPathFormat)
		}

		if err := p.GcsClient.TransferFolder(srcBucket, srcPath, srcModelName,
			destBucket, destPath, destModelName); err != nil {
			log.Error().Err(err).Msg(errGCSCopyFailed)
			return transferredGcsModelData, err
		}

		log.Info().Msgf("Successfully copied model in model-source: %s → %s", srcModelName, destModelName)

		if !isNotProd && srcModelName != destModelName {
			if err := p.copyConfigToNewNameInConfigSource(srcModelName, destModelName); err != nil {
				log.Error().Err(err).Msgf("Failed to copy config to config-source: %s → %s",
					srcModelName, destModelName)
				return transferredGcsModelData, err
			}
		}

		transferredGcsModelData = append(transferredGcsModelData, GcsModelData{
			Bucket: destBucket,
			Path:   destPath,
			Name:   destModelName,
		})
	}

	return transferredGcsModelData, nil
}

func (p *Predator) processDBPopulationStageForDelete(predatorRequestList []predatorrequest.PredatorRequest, requestIdPayloadMap map[uint]*Payload, req ApproveRequest) {
	if predatorRequestList[0].RequestStage != predatorStageDBPopulation {
		return
	}

	if err := p.deleteServiceDiscoveryAndConfig(req, predatorRequestList, requestIdPayloadMap); err != nil {
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageDBPopulation)
		return
	}
}

func (p *Predator) processDBPopulationStage(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, approvedBy string, successMessage string) error {
	if predatorRequestList[0].RequestStage != predatorStageDBPopulation {
		return nil
	}
	tx := p.Repo.DB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Error().Msgf("panic recovered, transaction rolled back: %v", r)
		}
	}()
	for i := range predatorRequestList {
		if err := p.createDiscoveryAndPredatorConfigTx(tx, predatorRequestList[i], *requestIdPayloadMap[predatorRequestList[i].RequestID], approvedBy); err != nil {
			tx.Rollback()
			log.Error().Err(err).Msg(failedToCreateServiceDiscoveryAndConfig)
			return err
		}

		predatorRequestList[i].Status = statusInProgress
		predatorRequestList[i].RequestStage = predatorStageRestartDeployable
		if err := p.Repo.UpdateStatusAndStage(tx, &predatorRequestList[i]); err != nil {
			tx.Rollback()
			log.Printf(errFailedToUpdateRequestStatusAndStage, err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		log.Printf("failed to commit transaction: %v", err)
		return err
	}
	log.Printf("success %s %d\n", successMessage, predatorRequestList[0].GroupId)
	return nil
}

func (p *Predator) checkIfModelsExist(predatorRequestList []predatorrequest.PredatorRequest) bool {
	for _, requestModel := range predatorRequestList {
		modelName := requestModel.ModelName
		if modelName == "" {
			log.Error().Msgf("model name is empty for request ID %d", requestModel.RequestID)
			continue
		}

		predatorConfig, err := p.PredatorConfigRepo.GetActiveModelByModelName(modelName)
		if err != nil {
			log.Error().Err(err).Msgf("failed to fetch predator config for model %s", modelName)
			continue
		}
		if predatorConfig != nil {
			log.Error().Msgf("model %s already exists", modelName)
			return true
		}
	}
	return false
}

func (p *Predator) processOnboardFlow(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) {
	if p.checkIfModelsExist(predatorRequestList) {
		req.RejectReason = "model already exists"
		req.Status = statusRejected
		p.processRejectRequest(predatorRequestList, req)
		return
	}

	transferredGcsModelData, err := p.processGCSCloneStage(requestIdPayloadMap, predatorRequestList, req)
	if err != nil {
		log.Error().Err(err).Msg(errFailedToOperateGcsCloneStage)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		p.revert(transferredGcsModelData)
		return
	}

	err = p.processDBPopulationStage(requestIdPayloadMap, predatorRequestList, req.ApprovedBy, onboardRequestFlow)
	if err != nil {
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageDBPopulation)
		p.revert(transferredGcsModelData)
		return
	}
	if err := p.processRestartDeployableStage(req.ApprovedBy, predatorRequestList, requestIdPayloadMap); err != nil {
		log.Error().Err(err).Msg(errFailedToRestartDeployable)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageRestartDeployable)
		return
	}
}

func (p *Predator) revert(transferredGcsModelData []GcsModelData) error {
	for _, data := range transferredGcsModelData {
		if err := p.GcsClient.DeleteFolder(data.Bucket, data.Path, data.Name); err != nil {
			log.Error().Err(err).Msg(errGCSCopyFailed)
			return err
		}
	}
	return nil
}

func (p *Predator) revertForDelete(transferredGcsModelData []GcsTransferredData) error {
	for _, data := range transferredGcsModelData {
		if err := p.GcsClient.TransferAndDeleteFolder(data.SrcBucket, data.SrcPath, data.SrcName, data.DestBucket, data.DestPath, data.DestName); err != nil {
			log.Error().Err(err).Msg(errGCSCopyFailed)
			return err
		}
	}
	return nil
}

func (p *Predator) processScaleUpFlow(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) {
	if p.checkIfModelsExist(predatorRequestList) {
		req.RejectReason = fmt.Sprintf("model %s already exists", requestIdPayloadMap[predatorRequestList[0].RequestID].ModelName)
		req.Status = statusRejected
		p.processRejectRequest(predatorRequestList, req)
		return
	}

	transferredGcsModelData, err := p.processGCSCloneStageIndefaultFolder(requestIdPayloadMap, predatorRequestList, req)
	if err != nil {
		log.Error().Err(err).Msg(errFailedToOperateGcsCloneStage)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		p.revert(transferredGcsModelData)
		return
	}

	transferredGcsModelData, err = p.processGCSCloneStage(requestIdPayloadMap, predatorRequestList, req)
	if err != nil {
		log.Error().Err(err).Msg(errFailedToOperateGcsCloneStage)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		p.revert(transferredGcsModelData)
		return
	}

	err = p.processDBPopulationStage(requestIdPayloadMap, predatorRequestList, req.ApprovedBy, cloneRequestFlow)
	if err != nil {
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageDBPopulation)
		p.revert(transferredGcsModelData)
		return
	}
	if err := p.processRestartDeployableStage(req.ApprovedBy, predatorRequestList, requestIdPayloadMap); err != nil {
		log.Error().Err(err).Msg(errFailedToRestartDeployable)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageRestartDeployable)
		return
	}
}

func (p *Predator) processPromoteFlow(requestIdPayloadMap map[uint]*Payload, predatorRequestList []predatorrequest.PredatorRequest, req ApproveRequest) {
	if p.checkIfModelsExist(predatorRequestList) {
		req.RejectReason = fmt.Sprintf("model %s already exists", requestIdPayloadMap[predatorRequestList[0].RequestID].ModelName)
		req.Status = statusRejected
		p.processRejectRequest(predatorRequestList, req)
		return
	}

	transferredGcsModelData, err := p.processGCSCloneStage(requestIdPayloadMap, predatorRequestList, req)
	if err != nil {
		log.Error().Err(err).Msg(errFailedToOperateGcsCloneStage)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageCloneToBucket)
		p.revert(transferredGcsModelData)
		return
	}

	err = p.processDBPopulationStage(requestIdPayloadMap, predatorRequestList, req.ApprovedBy, promoteRequestFlow)
	if err != nil {
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageDBPopulation)
		p.revert(transferredGcsModelData)
		return
	}
	if err := p.processRestartDeployableStage(req.ApprovedBy, predatorRequestList, requestIdPayloadMap); err != nil {
		log.Error().Err(err).Msg(errFailedToRestartDeployable)
		p.updateRequestStatusAndStage(req.ApprovedBy, predatorRequestList, statusFailed, predatorStageRestartDeployable)
		return
	}
}

func (p *Predator) updateRequestStatusAndStage(approvedBy string, predatorRequestList []predatorrequest.PredatorRequest, status, stage string) {
	for i := range predatorRequestList {
		predatorRequestList[i].Status = status
		predatorRequestList[i].Reviewer = approvedBy
		predatorRequestList[i].UpdatedBy = approvedBy
		if stage != constant.EmptyString {
			predatorRequestList[i].RequestStage = stage
		}
		if predatorRequestList[i].Status == statusApproved ||
			predatorRequestList[i].Status == statusFailed ||
			predatorRequestList[i].Status == statusRejected {
			predatorRequestList[i].Active = false
		}
		predatorRequestList[i].UpdatedAt = time.Now()
	}

	if err := p.Repo.UpdateMany(predatorRequestList); err != nil {
		log.Printf(errFailedToUpdateRequestStatusAndStage, err)
	}
}

func (p *Predator) createDiscoveryAndPredatorConfigTx(tx *gorm.DB, requestModel predatorrequest.PredatorRequest, payload Payload, approvedBy string) error {
	discoveryConfig, err := p.createDiscoveryConfigTx(tx, &requestModel, payload)
	if err != nil {
		return err
	}
	return p.createPredatorConfigTx(tx, &requestModel, payload, approvedBy, discoveryConfig.ID)
}

func (p *Predator) createDiscoveryConfigTx(tx *gorm.DB, requestModel *predatorrequest.PredatorRequest, payload Payload) (discoveryconfig.DiscoveryConfig, error) {
	discoveryConfig := discoveryconfig.DiscoveryConfig{
		ServiceDeployableID: int(payload.ConfigMapping.ServiceDeployableID),
		CreatedBy:           requestModel.CreatedBy,
		UpdatedBy:           requestModel.UpdatedBy,
		Active:              true,
		CreatedAt:           requestModel.CreatedAt,
		UpdatedAt:           time.Now(),
	}
	if err := tx.Create(&discoveryConfig).Error; err != nil {
		log.Error().Err(err).Msg(errMsgInsertDiscovery)
		return discoveryConfig, err
	}
	return discoveryConfig, nil
}

func (p *Predator) createPredatorConfigTx(tx *gorm.DB, requestModel *predatorrequest.PredatorRequest, payload Payload, approvedBy string, discoveryConfigID int) error {
	// Clean up ensemble scheduling before marshaling
	cleanedMetaData := p.cleanEnsembleScheduling(payload.MetaData)

	metaDataBytes, err := json.Marshal(cleanedMetaData)
	if err != nil {
		log.Error().Err(err).Msg(errMsgMarshalMeta)
		return err
	}

	serviceDeployableID := int(payload.ConfigMapping.ServiceDeployableID)
	serviceDeployable, err := p.ServiceDeployableRepo.GetById(serviceDeployableID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get service deployable config for ID %d", serviceDeployableID)
		return fmt.Errorf("failed to get service deployable config: %w", err)
	}

	config := predatorconfig.PredatorConfig{
		DiscoveryConfigID: discoveryConfigID,
		ModelName:         payload.ModelName,
		MetaData:          metaDataBytes,
		CreatedBy:         requestModel.CreatedBy,
		UpdatedBy:         approvedBy,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Active:            true,
		SourceModelName:   payload.ConfigMapping.SourceModelName,
	}

	if serviceDeployable.OverrideTesting {
		log.Info().Msgf("OverrideTesting is enabled for deployable %s. Setting test_results for model %s",
			serviceDeployable.Name, payload.ModelName)

		config.TestResults = json.RawMessage(`{"is_functionally_tested": true}`)
		config.HasNilData = false
	}

	if err := tx.Create(&config).Error; err != nil {
		log.Error().Err(err).Msg(errMsgInsertConfig)
		return err
	}
	return nil
}
