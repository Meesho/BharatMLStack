package bulkdeletestrategy

import (
	"errors"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	inferflow_etcd "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/etcd"
	infrastructurehandler "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/handler"
	numerix_etcd "github.com/Meesho/BharatMLStack/horizon/internal/numerix/etcd"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
)

type StrategySelectorImplStrategySelector interface {
	GetBulkDeleteStrategy(serviceDeployableId int) error
}

type StrategySelectorImpl struct {
	sqlConn               *infra.SQLConnection
	prometheusClient      externalcall.PrometheusClient
	infrastructureHandler infrastructurehandler.InfrastructureHandler
	workingEnv            string
	slackClient           externalcall.SlackClient
	gcsClient             externalcall.GCSClientInterface
	InferflowEtcdClient   inferflow_etcd.Manager
	NumerixEtcdClient     numerix_etcd.Manager
}

var (
	strategySelectorOnce                sync.Once
	bulkDeletePredatorEnabled           bool
	bulkDeletePredatorMaxInactiveDays   int
	bulkDeleteInferflowEnabled         bool
	bulkDeleteInferflowMaxInactiveDays int
	bulkDeleteNumerixEnabled            bool
	bulkDeleteNumerixMaxInactiveDays    int
	enablePredatorRequestSubmission     bool
)

const (
	inferflowService 	= "inferflow"
	predatorService   	= "predator"
	numerixService      = "numerix"
)

func Init(config configs.Configs) StrategySelectorImpl {
	var strategySelectorImpl StrategySelectorImpl
	strategySelectorOnce.Do(func() {
		bulkDeletePredatorEnabled = config.BulkDeletePredatorEnabled
		bulkDeletePredatorMaxInactiveDays = config.BulkDeletePredatorMaxInactiveDays

		bulkDeleteInferflowEnabled = config.BulkDeleteInferflowEnabled
		bulkDeleteInferflowMaxInactiveDays = config.BulkDeleteInferflowMaxInactiveDays

		bulkDeleteNumerixEnabled = config.BulkDeleteNumerixEnabled
		bulkDeleteNumerixMaxInactiveDays = config.BulkDeleteNumerixMaxInactiveDays

		enablePredatorRequestSubmission = config.BulkDeletePredatorRequestSubmissionEnabled


		connection, err := infra.SQL.GetConnection()
		if err != nil {
			log.Panic().Err(err).Msg("Failed to get SQL connection")
		}
		sqlConn, ok := connection.(*infra.SQLConnection)
		if !ok {
			log.Panic().Msg("Failed to cast connection to SQLConnection")
		}

		inferflowEtcdClient := inferflow_etcd.NewEtcdInstance()
		numerixEtcdClient := numerix_etcd.NewEtcdInstance()
		infrastructureHandler := infrastructurehandler.InitInfrastructureHandler()
		workingEnv := externalcall.GetWorkingEnvironment()

		strategySelectorImpl = StrategySelectorImpl{
			sqlConn:               sqlConn,
			prometheusClient:      externalcall.GetPrometheusClient(),
			slackClient:           externalcall.GetSlackClient(),
			gcsClient:             externalcall.CreateGCSClient(),
			InferflowEtcdClient:   inferflowEtcdClient,
			NumerixEtcdClient:     numerixEtcdClient,
			infrastructureHandler: infrastructureHandler,
			workingEnv:            workingEnv,
		}
	})
	return strategySelectorImpl
}

func (ss *StrategySelectorImpl) GetBulkDeleteStrategy(service string) (BulkDeleteStrategy, error) {
	switch service {
	case inferflowService:
		if !bulkDeleteInferflowEnabled {
			return nil, errors.New("inferflow bulk delete is disabled for this environment")
		}
		return &InferflowService{ss.sqlConn, ss.prometheusClient, ss.slackClient, ss.InferflowEtcdClient}, nil
	case predatorService:
		if !bulkDeletePredatorEnabled {
			return nil, errors.New("predator bulk delete is disabled for this environment")
		}
		return &PredatorService{ss.sqlConn, ss.prometheusClient, ss.infrastructureHandler, ss.workingEnv, ss.slackClient, ss.gcsClient}, nil
	case numerixService:
		if !bulkDeleteNumerixEnabled {
			return nil, errors.New("numerix bulk delete is disabled for this environment")
		}
		return &NumerixService{ss.sqlConn, ss.prometheusClient, ss.slackClient, ss.NumerixEtcdClient}, nil
	default:
		log.Warn().Msg("Unknown service type: " + service)
		return nil, errors.New("unknown service type: " + service)
	}
}

