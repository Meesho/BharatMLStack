package bulkdeletestrategy

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	inferflow_etcd "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/etcd"
	discoveryconfig "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/discoveryconfig"
	inferflow_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow/config"
	inferflow_request "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow/request"
	servicedeployableconfig "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
)

type InferflowService struct {
	sqlConn          *infra.SQLConnection
	prometheusClient externalcall.PrometheusClient
	slackClient      externalcall.SlackClient
	etcdClient       inferflow_etcd.Manager
}

func (m *InferflowService) ProcessBulkDelete(serviceDeployable servicedeployableconfig.ServiceDeployableConfig) error {
	discoveryConfigRepo, inferflowConfigRepo, inferflowRequestRepo, err := m.initializeRepositories()
	if err != nil {
		log.Error().Err(err).Msg("Error initializing repositories")
		return err
	}

	discoveryConfigList, err := discoveryConfigRepo.GetByServiceDeployableID(serviceDeployable.ID)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching discovery config list")
		return err
	}

	nonActiveInferflowConfigList, err := m.fetchNonActiveInferflowConfigList(serviceDeployable, discoveryConfigList, inferflowConfigRepo)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching non active inferflow config list")
		return err
	}

	err = m.DeleteInferflowConfig(serviceDeployable, nonActiveInferflowConfigList, inferflowConfigRepo, inferflowRequestRepo)
	if err != nil {
		log.Error().Err(err).Msg("Error deleting inferflow config")
		return err
	}

	err = m.sendSlackNotification(serviceDeployable.Name, nonActiveInferflowConfigList)
	if err != nil {
		log.Error().Err(err).Msg("Error sending Slack notification")
		return err
	}

	return nil
}

func (m *InferflowService) initializeRepositories() (discoveryconfig.DiscoveryConfigRepository, inferflow_config.Repository, inferflow_request.Repository, error) {
	discoveryConfigRepo, err := discoveryconfig.NewRepository(m.sqlConn)
	if err != nil {
		log.Err(err).Msg("Error initializing discovery config repository")
		return nil, nil, nil, err
	}

	inferflowConfigRepo, err := inferflow_config.NewRepository(m.sqlConn)
	if err != nil {
		log.Err(err).Msg("Error initializing inferflow config repository")
		return nil, nil, nil, err
	}

	inferflowRequestRepo, err := inferflow_request.NewRepository(m.sqlConn)
	if err != nil {
		log.Err(err).Msg("Error initializing inferflow request repository")
		return nil, nil, nil, err
	}

	return discoveryConfigRepo, inferflowConfigRepo, inferflowRequestRepo, nil
}

func (m *InferflowService) fetchNonActiveInferflowConfigList(serviceDeployable servicedeployableconfig.ServiceDeployableConfig, discoveryConfigList []discoveryconfig.DiscoveryConfig, inferflowConfigRepo inferflow_config.Repository) ([]string, error) {
	activeInferflowConfigList, err := m.prometheusClient.GetInferflowConfigNames(serviceDeployable.Name)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching active inferflow config list")
		return nil, err
	}

	var allInferflowConfigList []string
	var discoveryConfigId []int
	for _, discoveryConfigEntity := range discoveryConfigList {
		discoveryConfigId = append(discoveryConfigId, discoveryConfigEntity.ID)
	}

	inferflowConfigList, err := inferflowConfigRepo.FindByDiscoveryIDsAndCreatedBefore(discoveryConfigId, bulkDeleteInferflowMaxInactiveDays)
	if err != nil {
		return nil, err
	}

	for _, inferflowConfigEntity := range inferflowConfigList {
		allInferflowConfigList = append(allInferflowConfigList, inferflowConfigEntity.ConfigID)
	}

	nonActiveInferflowConfigList := difference(allInferflowConfigList, activeInferflowConfigList)
	return nonActiveInferflowConfigList, nil
}

func (m *InferflowService) DeleteInferflowConfig(serviceDeployable servicedeployableconfig.ServiceDeployableConfig, nonActiveInferflowConfigList []string, inferflowConfigRepo inferflow_config.Repository, inferflowRequestRepo inferflow_request.Repository) error {
	for _, inferflowConfigId := range nonActiveInferflowConfigList {
		err := m.etcdClient.DeleteConfig(serviceDeployable.Name, inferflowConfigId)
		if err != nil {
			log.Error().Err(err).Msg("Error deleting inferflow config")
			return err
		}

		err = inferflowConfigRepo.Deactivate(inferflowConfigId)
		if err != nil {
			log.Error().Err(err).Msg("Error deactivating inferflow config")
			return err
		}

		err = inferflowRequestRepo.Deactivate(inferflowConfigId)
		if err != nil {
			log.Error().Err(err).Msg("Error deactivating inferflow config")
			return err
		}

	}
	return nil
}

func (m *InferflowService) sendSlackNotification(serviceDeployableName string, nonActiveInferflowConfigList []string) error {
	err := m.slackClient.SendInferflowConfigCleanupNotification(serviceDeployableName, nonActiveInferflowConfigList)
	if err != nil {
		log.Error().Err(err).Msg("Error sending Slack notification")
		return err
	}
	return nil
}
