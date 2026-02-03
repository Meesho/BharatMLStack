package bulkdeletestrategy

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	infrastructurehandler "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/predator/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/counter"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/discoveryconfig"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/predatorconfig"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/predatorrequest"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
)

type PredatorService struct {
	sqlConn               *infra.SQLConnection
	prometheusClient      externalcall.PrometheusClient
	infrastructureHandler infrastructurehandler.InfrastructureHandler
	workingEnv            string
	slackClient           externalcall.SlackClient
	gcsClient             externalcall.GCSClientInterface
}

const (
	slashConstant = "/"
	gcsPrefix     = "gs://"
	bulkDeleteCreatedBy = "horizon-bulk-delete" 
)

type ModelInfo struct {
	ModelName         string
	DiscoveryConfigID int
}

type PredatorBulkDeleteRepos struct {
	discoveryConfigRepo discoveryconfig.DiscoveryConfigRepository
	predatorConfigRepo  predatorconfig.PredatorConfigRepository
	predatorRequestRepo predatorrequest.PredatorRequestRepository
	groupCounterRepo    counter.GroupIdCounterRepository
}


func (p *PredatorService) ProcessBulkDelete(serviceDeployable servicedeployableconfig.ServiceDeployableConfig) error {
	predatorBulkDeleteRepos, err := p.initializeRepositories()
	if err != nil {
		log.Error().Err(err).Msg("Error initializing repositories")
		return err
	}

	discoveryConfigList, err := predatorBulkDeleteRepos.discoveryConfigRepo.GetByServiceDeployableID(serviceDeployable.ID)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching discovery config list")
		return err
	}

	_, zeroTrafficModelList, parentToChildMapping, trafficData, err := p.fetchModelNames(
		serviceDeployable,
		discoveryConfigList,
		&predatorBulkDeleteRepos,
	)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching model names")
		return err
	}

	deployableConfig, err := p.deserializeDeployableConfig(serviceDeployable)
	if err != nil {
		log.Error().Err(err).Msg("Error unmarshaling deployable config")
		return err
	}

	// Get child models for zero traffic parents
	var modelsToDelete []ModelInfo
	for _, parentModel := range zeroTrafficModelList {
		modelsToDelete = append(modelsToDelete, parentModel)

		// Add child models if any
		if children, found := parentToChildMapping[parentModel.ModelName]; found {
			for _, childName := range children {
				// Find child's discovery config ID from predator config
				childTraffic, existsInPrometheus := trafficData[childName]
				if existsInPrometheus && childTraffic.TotalTraffic > 0 {
					log.Info().Msgf("[SKIP CHILD] Model: %s (child of %s) has traffic (%.2f), skipping deletion",
						childName, parentModel.ModelName, childTraffic.TotalTraffic)
					continue
				}

				childConfig, _ := predatorBulkDeleteRepos.predatorConfigRepo.GetActiveModelByModelName(childName)
				if childConfig != nil {
					modelsToDelete = append(modelsToDelete, ModelInfo{
						ModelName:         childName,
						DiscoveryConfigID: childConfig.DiscoveryConfigID,
					})
					log.Info().Msgf("[DELETE CHILD] Model: %s (child of %s) has zero traffic, adding to delete list",
						childName, parentModel.ModelName)
				}
			}
		}
	}

	if len(modelsToDelete) == 0 {
		log.Info().Msg("No models to delete")
		return nil
	}

	// Process deletion: GCS delete, deactivate predator_config, deactivate discovery_config, create request
	deletedModels := p.processDeleteModels(
		strings.TrimSuffix(deployableConfig.GCSBucketPath, "/*"),
		modelsToDelete,
		serviceDeployable,
		&predatorBulkDeleteRepos,
	)

	if len(deletedModels) > 0 {
		// Restart deployable after deletion
		err = p.infrastructureHandler.RestartDeployment(serviceDeployable.Name, p.workingEnv, false)
		if err != nil {
			log.Error().Err(err).Msg("Error restarting deployable")
		}
	}

	err = p.sendSlackNotification(serviceDeployable.Name, deletedModels)
	if err != nil {
		log.Error().Err(err).Msg("Error sending Slack notification")
		return err
	}

	return nil
}

func (p *PredatorService) initializeRepositories() (PredatorBulkDeleteRepos, error) {
	discoveryConfigRepo, err := discoveryconfig.NewRepository(p.sqlConn)
	if err != nil {
		log.Err(err).Msg("Error initializing discovery config repository")
		return PredatorBulkDeleteRepos{}, err
	}

	predatorConfigRepo, err := predatorconfig.NewRepository(p.sqlConn)
	if err != nil {
		log.Err(err).Msg("Error initializing predator config repository")
		return PredatorBulkDeleteRepos{}, err
	}

	predatorRequestRepo, err := predatorrequest.NewRepository(p.sqlConn)
	if err != nil {
		log.Err(err).Msg("Error initializing predator request repository")
		return PredatorBulkDeleteRepos{}, err
	}

	groupCounterRepo, err := counter.NewCounterRepository(p.sqlConn)
	if err != nil {
		log.Err(err).Msg("Error initializing group counter repository")
		return PredatorBulkDeleteRepos{}, err
	}

	return PredatorBulkDeleteRepos{
		discoveryConfigRepo: discoveryConfigRepo,
		predatorConfigRepo:  predatorConfigRepo,
		predatorRequestRepo: predatorRequestRepo,
		groupCounterRepo:    groupCounterRepo,
	}, nil
}

func (p *PredatorService) fetchModelNames(
	serviceDeployable servicedeployableconfig.ServiceDeployableConfig,
	discoveryConfigList []discoveryconfig.DiscoveryConfig,
	predatorBulkDeleteRepos *PredatorBulkDeleteRepos,
) ([]ModelInfo, []ModelInfo, map[string][]string, map[string]externalcall.PredatorModelTraffic, error) {

	zeroTrafficDays := bulkDeletePredatorMaxInactiveDays

	trafficData, err := p.prometheusClient.GetPredatorModelTraffic(serviceDeployable.Name, zeroTrafficDays)
	if err != nil {
		log.Err(err).Msg("Error fetching predator model traffic from Prometheus")
		return nil, nil, nil, nil, err
	}

	// Get ALL models from DB
	var discoveryConfigIds []int
	for _, discoveryConfigEntity := range discoveryConfigList {
		if discoveryConfigEntity.Active {
			discoveryConfigIds = append(discoveryConfigIds, discoveryConfigEntity.ID)
		}
	}

	predatorConfigList, err := predatorBulkDeleteRepos.predatorConfigRepo.FindByDiscoveryIDsAndAge(discoveryConfigIds, zeroTrafficDays)
	if err != nil {
		log.Err(err).Msg("Error fetching predator configs from DB")
		return nil, nil, nil, nil, err
	}

	modelInfoMap := make(map[string]ModelInfo) // deduplicate
	parentToChildMapping := make(map[string][]string)
	childModelNames := make(map[string]bool)

	for _, pc := range predatorConfigList {
		if !pc.Active {
			continue
		}

		if _, exists := modelInfoMap[pc.ModelName]; exists {
			continue
		}

		modelInfoMap[pc.ModelName] = ModelInfo{
			ModelName:         pc.ModelName,
			DiscoveryConfigID: pc.DiscoveryConfigID,
		}

		var metaData handler.MetaData
		if err := json.Unmarshal(pc.MetaData, &metaData); err != nil {
			log.Err(err).Msg("Could not unmarshall model metadata for: " + pc.ModelName + " for scheduled deletion")
			continue
		}

		if (&metaData.Ensembling) != nil && metaData.Ensembling.Step != nil {
			for _, step := range metaData.Ensembling.Step {
				if step.ModelName != "" {
					parentToChildMapping[pc.ModelName] = append(parentToChildMapping[pc.ModelName], step.ModelName)
					childModelNames[step.ModelName] = true
				}
			}
		}
	}

	var allParentModels []ModelInfo
	for modelName, info := range modelInfoMap {
		if !childModelNames[modelName] {
			allParentModels = append(allParentModels, info)
		}
	}

	// Separate active vs zero-traffic
	var activeModels []ModelInfo
	var zeroTrafficModels []ModelInfo

	log.Info().Msgf("=== Traffic check for %s (past %d days) ===", serviceDeployable.Name, zeroTrafficDays)

	for _, modelInfo := range allParentModels {
		traffic, existsInPrometheus := trafficData[modelInfo.ModelName]

		if existsInPrometheus && traffic.TotalTraffic > 0 {
			activeModels = append(activeModels, modelInfo)
			log.Info().Msgf("[ACTIVE] Model: %s | Traffic: %.2f", modelInfo.ModelName, traffic.TotalTraffic)
		} else {
			zeroTrafficModels = append(zeroTrafficModels, modelInfo)
			log.Warn().Msgf("[ZERO TRAFFIC - DELETE CANDIDATE] Model: %s | 0 traffic for %d days", modelInfo.ModelName, zeroTrafficDays)
		}
	}

	log.Info().Msgf("Summary: Total: %d | Active: %d | Zero traffic (to delete): %d",
		len(allParentModels), len(activeModels), len(zeroTrafficModels))

	return activeModels, zeroTrafficModels, parentToChildMapping, trafficData, nil
}

// processDeleteModels - NEW: Delete from GCS, DB, and create approved delete request
func (p *PredatorService) processDeleteModels(
	basePath string,
	modelInfoList []ModelInfo,
	serviceDeployableConfig servicedeployableconfig.ServiceDeployableConfig,
	predatorBulkDeleteRepos *PredatorBulkDeleteRepos,
) []string {
	srcBucket, srcPath := extractGCSPath(basePath)

	var deletedModels []string
	for _, modelInfo := range modelInfoList {
		modelName := modelInfo.ModelName
		discoveryConfigID := modelInfo.DiscoveryConfigID
		modelGCSPath := srcPath + "/" + modelName
		existsInGCS, err := p.gcsClient.CheckFolderExists(srcBucket, modelGCSPath)
		if err != nil {
			log.Warn().Err(err).Msgf("Failed to check GCS existence for model %s, assuming it exists", modelName)
			existsInGCS = true
		}

		err = p.processModelDeletion(
			srcBucket, srcPath, modelName, discoveryConfigID, serviceDeployableConfig.ID,
			predatorBulkDeleteRepos, existsInGCS,
		)

		if err != nil {
			log.Error().Err(err).Msgf("Failed to process scheduled deletion for model: %s, skipping", modelName)
			continue
		}

		deletedModels = append(deletedModels, modelInfo.ModelName)
	}

	return deletedModels
}

func (p *PredatorService) processModelDeletion(
	srcBucket, srcPath, modelName string,
	discoveryConfigID int, serviceDeployableID int,
	predatorBulkDeleteRepos *PredatorBulkDeleteRepos,
	existsInGCS bool,
) (err error) {
	db := predatorBulkDeleteRepos.predatorConfigRepo.DB()

	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Error().Msgf("Panic recovered, transaction rolled back for model: %s", modelName)
			err = fmt.Errorf("panic during deletion of model %s: %v", modelName, r)
		}
	}()

	predatorConfig, err := predatorBulkDeleteRepos.predatorConfigRepo.WithTx(tx).GetActiveModelByModelName(modelName)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to fetch predator config for model %s: %w", modelName, err)
	}
	if predatorConfig == nil {
		tx.Rollback()
		return fmt.Errorf("no active predator config found for model %s", modelName)
	}

	// 1. Deactivate predator_config (in transaction)
	predatorConfig.Active = false
	predatorConfig.UpdatedAt = time.Now()
	predatorConfig.UpdatedBy = bulkDeleteCreatedBy
	err = predatorBulkDeleteRepos.predatorConfigRepo.WithTx(tx).Update(predatorConfig)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to deactivate predator_config: %w", err)
	}
	log.Info().Msgf("Deactivated predator_config: %s", modelName)

	// 2. Deactivate discovery_config (in transaction)
	err = predatorBulkDeleteRepos.discoveryConfigRepo.WithTx(tx).DeactivateByID(discoveryConfigID, bulkDeleteCreatedBy)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to deactivate discovery_config ID %d: %w", discoveryConfigID, err)
	}
	log.Info().Msgf("Deactivated discovery_config ID: %d", discoveryConfigID)

	// 3. Create APPROVED delete request (in transaction) if flag enabled
	if enablePredatorRequestSubmission {
		// create predator payload for creating deletion request
		payload := map[string]interface{}{
			"model_name":          modelName,
			"model_source_path":   fmt.Sprintf("gs://%s/%s/%s", srcBucket, srcPath, modelName),
			"meta_data":           predatorConfig.MetaData,
			"discovery_config_id": discoveryConfigID,
			"config_mapping": map[string]interface{}{
				"service_deployable_id": serviceDeployableID,
			},
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to marshall payload for model %s: %w", modelName, err)
		}

		groupID, err := predatorBulkDeleteRepos.groupCounterRepo.GetAndIncrementCounter(1)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to get new groupID for model deletion: %w", err)
		}

		deleteRequest := predatorrequest.PredatorRequest{
			ModelName:    modelName,
			GroupId:      groupID,
			Payload:      string(payloadBytes),
			CreatedBy:    bulkDeleteCreatedBy,
			UpdatedBy:    bulkDeleteCreatedBy,
			Reviewer:     bulkDeleteCreatedBy,
			RequestType:  "Delete",
			Status:       "Approved",
			RequestStage: "DB Population",
			Active:       false,
			IsValid:      true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		err = predatorBulkDeleteRepos.predatorRequestRepo.WithTx(tx).Create(&deleteRequest)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create delete request: %w", err)
		}
		log.Info().Msgf("Created APPROVED delete request: %s", modelName)
	}

	// 4. Commit DB transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	log.Info().Msgf("DB transaction committed for model: %s", modelName)

	// Step 5: Delete from GCS (AFTER successful commit)
	if existsInGCS {
		if err := p.gcsClient.DeleteFolder(srcBucket, srcPath, modelName); err != nil {
			log.Error().Err(err).Msgf("GCS deletion failed for model %s after DB commit - orphaned data may need manual cleanup", modelName)
		} else {
			log.Info().Msgf("Deleted model from GCS: %s", modelName)
		}
	} else {
		log.Info().Msgf("Model %s not found in GCS, skipping GCS deletion (DB cleanup only)", modelName)
	}

	return nil
}

func (p *PredatorService) deserializeDeployableConfig(serviceDeployable servicedeployableconfig.ServiceDeployableConfig) (handler.PredatorDeployableConfig, error) {
	var deployableConfig handler.PredatorDeployableConfig
	err := json.Unmarshal(serviceDeployable.Config, &deployableConfig)
	if err != nil {
		log.Error().Err(err).Msg("Error unmarshaling deployable config")
		return deployableConfig, err
	}
	return deployableConfig, nil
}

func (p *PredatorService) processGCSAndDeleteModels(basePath string, inactiveModelNameList []string) []string {
	srcBucket, srcPath := extractGCSPath(basePath)
	destBucket, destPath := extractGCSPath(defaultModelPath)

	var deleteModelNameList []string
	for _, inactiveModelName := range inactiveModelNameList {
		err := p.gcsClient.TransferAndDeleteFolder(srcBucket, srcPath, inactiveModelName, destBucket, destPath, inactiveModelName)
		if err != nil {
			log.Error().Err(err).Msg("Error transferring and deleting folder in GCS")
			continue
		}
		deleteModelNameList = append(deleteModelNameList, inactiveModelName)
	}

	return deleteModelNameList
}

func (p *PredatorService) deactivateModelsAndRestartDeployable(deleteModelNameList []string, serviceDeployable servicedeployableconfig.ServiceDeployableConfig, predatorConfig predatorconfig.PredatorConfigRepository, discoveryConfigId []int) error {
	err := predatorConfig.BulkDeactivateByModelNames(deleteModelNameList, serviceDeployable.UpdatedBy, discoveryConfigId)
	if err != nil {
		log.Error().Err(err).Msg("Error deactivating models in predator config")
		return err
	}

	// Extract isCanary from deployable config
	var deployableConfig map[string]interface{}
	isCanary := false
	if err := json.Unmarshal(serviceDeployable.Config, &deployableConfig); err == nil {
		if strategy, ok := deployableConfig["deploymentStrategy"].(string); ok && strategy == "canary" {
			isCanary = true
		}
	}
	if err := p.infrastructureHandler.RestartDeployment(serviceDeployable.Name, p.workingEnv, isCanary); err != nil {
		log.Error().Err(err).Msg("Error restarting deployable")
		return fmt.Errorf("failed to restart deployable: %w", err)
	}

	return nil
}

func (p *PredatorService) sendSlackNotification(serviceDeployableName string, deleteModelNameList []string) error {
	err := p.slackClient.SendCleanupNotification(serviceDeployableName, deleteModelNameList)
	if err != nil {
		log.Error().Err(err).Msg("Error sending Slack notification")
		return err
	}
	return nil
}

func difference(all, active []string) []string {
	activeSet := make(map[string]struct{}, len(active))
	for _, name := range active {
		activeSet[name] = struct{}{}
	}

	var result []string
	for _, name := range all {
		if _, found := activeSet[name]; !found {
			result = append(result, name)
		}
	}
	return result
}

func extractGCSPath(gcsURL string) (bucket, objectPath string) {
	bucket, objectPath, ok := parseGCSURL(gcsURL)
	if !ok {
		return constant.EmptyString, constant.EmptyString
	}
	return bucket, objectPath
}

func parseGCSURL(gcsURL string) (bucket, objectPath string, ok bool) {
	if !strings.HasPrefix(gcsURL, gcsPrefix) {
		return constant.EmptyString, constant.EmptyString, false
	}

	trimmed := strings.TrimPrefix(gcsURL, gcsPrefix)
	parts := strings.SplitN(trimmed, slashConstant, 2)
	if len(parts) < 1 {
		return constant.EmptyString, constant.EmptyString, false
	}

	bucket = parts[0]
	if len(parts) == 2 {
		objectPath = parts[1]
	}
	return bucket, objectPath, true
}
