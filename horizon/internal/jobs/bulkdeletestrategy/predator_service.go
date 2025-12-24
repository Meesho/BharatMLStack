package bulkdeletestrategy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	infrastructurehandler "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/predator/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/discoveryconfig"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/predatorconfig"
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
)

func (p *PredatorService) ProcessBulkDelete(serviceDeployable servicedeployableconfig.ServiceDeployableConfig) error {
	discoveryConfigRepo, predatorConfigRepo, err := p.initializeRepositories()
	if err != nil {
		log.Error().Err(err).Msg("Error initializing repositories")
		return err
	}

	discoveryConfigList, err := discoveryConfigRepo.GetByServiceDeployableID(serviceDeployable.ID)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching discovery config list")
		return err
	}

	activeModelNameList, parentModelNameList, parentToChildMapping, discoveryConfigId, err := p.fetchModelNames(serviceDeployable, discoveryConfigList, predatorConfigRepo)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching model names")
		return err
	}

	deployableConfig, err := p.deserializeDeployableConfig(serviceDeployable)
	if err != nil {
		log.Error().Err(err).Msg("Error unmarshaling deployable config")
		return err
	}

	inactiveModelNameList := difference(parentModelNameList, activeModelNameList)
	childModelNameList := make([]string, 0)
	for _, inactiveModel := range inactiveModelNameList {
		if _, found := parentToChildMapping[inactiveModel]; !found {
			continue
		}
		childModelNameList = append(childModelNameList, parentToChildMapping[inactiveModel]...)
	}

	deleteModelNameList := p.processGCSAndDeleteModels(strings.TrimSuffix(deployableConfig.GCSBucketPath, "/*"), append(inactiveModelNameList, childModelNameList...))

	err = p.deactivateModelsAndRestartDeployable(deleteModelNameList, serviceDeployable, predatorConfigRepo, discoveryConfigId)
	if err != nil {
		log.Error().Err(err).Msg("Error deactivating models and restarting deployable")
		return err
	}

	err = p.sendSlackNotification(serviceDeployable.Name, deleteModelNameList)
	if err != nil {
		log.Error().Err(err).Msg("Error sending Slack notification")
		return err
	}

	return nil
}

func (p *PredatorService) initializeRepositories() (discoveryconfig.DiscoveryConfigRepository, predatorconfig.PredatorConfigRepository, error) {
	discoveryConfigRepo, err := discoveryconfig.NewRepository(p.sqlConn)
	if err != nil {
		log.Err(err).Msg("Error initializing discovery config repository")
		return nil, nil, err
	}

	predatorConfigRepo, err := predatorconfig.NewRepository(p.sqlConn)
	if err != nil {
		log.Err(err).Msg("Error initializing predator config repository")
		return nil, nil, err
	}

	return discoveryConfigRepo, predatorConfigRepo, nil
}

func (p *PredatorService) fetchModelNames(serviceDeployable servicedeployableconfig.ServiceDeployableConfig, discoveryConfigList []discoveryconfig.DiscoveryConfig, predatorConfigRepo predatorconfig.PredatorConfigRepository) ([]string, []string, map[string][]string, []int, error) {
	activeModelNameList, err := p.prometheusClient.GetModelNames(serviceDeployable.Name)
	if err != nil {
		log.Err(err).Msg("Error fetching active model names from Prometheus")
		return nil, nil, nil, nil, err
	}

	var allModelNameList []string
	parentToChildMapping := make(map[string][]string)
	childModelNameList := make([]string, 0)
	var discoveryConfigId []int
	for _, discoveryConfigEntity := range discoveryConfigList {
		discoveryConfigId = append(discoveryConfigId, discoveryConfigEntity.ID)
	}

	predatorConfigList, err := predatorConfigRepo.FindByDiscoveryIDsAndCreatedBefore(discoveryConfigId, maxPredatorInactiveAge)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	for _, predatorConfigEntity := range predatorConfigList {
		var metaData handler.MetaData
		if err := json.Unmarshal(predatorConfigEntity.MetaData, &metaData); err != nil {
			continue
		}

		if metaData.Ensembling.Step != nil {
			for _, step := range metaData.Ensembling.Step {
				if step.ModelName != "" {
					parentToChildMapping[predatorConfigEntity.ModelName] = append(parentToChildMapping[predatorConfigEntity.ModelName], step.ModelName)
					childModelNameList = append(childModelNameList, step.ModelName)
				}
			}
		}

		allModelNameList = append(allModelNameList, predatorConfigEntity.ModelName)
	}

	parentModelNameList := difference(allModelNameList, childModelNameList)
	return activeModelNameList, parentModelNameList, parentToChildMapping, discoveryConfigId, nil
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
