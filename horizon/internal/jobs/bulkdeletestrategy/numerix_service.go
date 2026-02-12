package bulkdeletestrategy

import (
	"strconv"

	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	etcd "github.com/Meesho/BharatMLStack/horizon/internal/numerix/etcd"
	numerix_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/numerix/config"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
)

type NumerixService struct {
	sqlConn          *infra.SQLConnection
	prometheusClient externalcall.PrometheusClient
	slackClient      externalcall.SlackClient
	etcdClient       etcd.Manager
}

func (i *NumerixService) ProcessBulkDelete(serviceDeployable servicedeployableconfig.ServiceDeployableConfig) error {
	numerixConfigRepo, err := i.initializeRepositories()
	if err != nil {
		log.Error().Err(err).Msg("Error initializing repositories")
		return err
	}

	nonActiveNumerixConfigList, err := i.fetchNonActiveNumerixConfigList(numerixConfigRepo)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching non active numerix config list")
		return err
	}

	err = i.DeleteNumerixConfig(numerixConfigRepo, nonActiveNumerixConfigList)
	if err != nil {
		log.Error().Err(err).Msg("Error deleting numerix config and making backup")
		return err
	}

	err = i.sendSlackNotification(serviceDeployable.Name, nonActiveNumerixConfigList)
	if err != nil {
		log.Error().Err(err).Msg("Error sending Slack notification")
		return err
	}

	return nil
}

func (i *NumerixService) initializeRepositories() (numerix_config.Repository, error) {

	numerixConfigRepo, err := numerix_config.NewRepository(i.sqlConn)
	if err != nil {
		log.Err(err).Msg("Error initializing numerix config repository")
		return nil, err
	}

	return numerixConfigRepo, nil
}

func (i *NumerixService) fetchNonActiveNumerixConfigList(numerixConfigRepo numerix_config.Repository) ([]string, error) {
	activeNumerixConfigList, err := i.prometheusClient.GetNumerixConfigNames()
	if err != nil {
		log.Error().Err(err).Msg("Error fetching active numerix config list")
		return nil, err
	}

	var allNumerixConfigList []string

	numerixConfigList, err := numerixConfigRepo.FindByCreatedBefore(bulkDeleteNumerixMaxInactiveDays)
	if err != nil {
		return nil, err
	}

	for _, numerixConfigEntity := range numerixConfigList {
		allNumerixConfigList = append(allNumerixConfigList, strconv.Itoa(int(numerixConfigEntity.ConfigID)))
	}

	nonActiveNumerixConfigList := difference(allNumerixConfigList, activeNumerixConfigList)
	return nonActiveNumerixConfigList, nil
}

func (i *NumerixService) DeleteNumerixConfig(numerixConfigRepo numerix_config.Repository, nonActiveNumerixConfigList []string) error {
	for _, numerixConfigId := range nonActiveNumerixConfigList {
		err := i.etcdClient.DeleteConfig(numerixConfigId)
		if err != nil {
			log.Error().Err(err).Msg("Error deleting numerix config")
			return err
		}

		err = numerixConfigRepo.Deactivate(numerixConfigId)
		if err != nil {
			log.Error().Err(err).Msg("Error deactivating numerix config")
			return err
		}

	}
	return nil
}

func (i *NumerixService) sendSlackNotification(serviceDeployableName string, nonActiveNumerixConfigList []string) error {
	err := i.slackClient.SendNumerixConfigCleanupNotification(serviceDeployableName, nonActiveNumerixConfigList)
	if err != nil {
		log.Error().Err(err).Msg("Error sending Slack notification")
		return err
	}
	return nil
}
