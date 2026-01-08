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
	strategySelectorOnce    sync.Once
	maxPredatorInactiveAge  int
	defaultModelPath        string
	maxInferflowInactiveAge int
	maxNumerixInactiveAge   int
)

func Init(config configs.Configs) StrategySelectorImpl {
	var strategySelectorImpl StrategySelectorImpl
	strategySelectorOnce.Do(func() {
		maxPredatorInactiveAge = config.MaxPredatorInactiveAge
		defaultModelPath = config.DefaultModelPath
		maxInferflowInactiveAge = config.MaxInferflowInactiveAge
		maxNumerixInactiveAge = config.MaxNumerixInactiveAge

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
			gcsClient:             externalcall.CreateGCSClient(config.GcsEnabled),
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
	case "INFERFLOW":
		return &InferflowService{ss.sqlConn, ss.prometheusClient, ss.slackClient, ss.InferflowEtcdClient}, nil
	case "PREDATOR":
		return &PredatorService{ss.sqlConn, ss.prometheusClient, ss.infrastructureHandler, ss.workingEnv, ss.slackClient, ss.gcsClient}, nil
	case "NUMERIX":
		return &NumerixService{ss.sqlConn, ss.prometheusClient, ss.slackClient, ss.NumerixEtcdClient}, nil
	default:
		log.Warn().Msg("Unknown service type: " + service)
		return nil, errors.New("unknown service type: " + service)
	}
}
