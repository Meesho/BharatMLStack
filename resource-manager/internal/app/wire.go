package app

import (
	"net/http"

	rmconfig "github.com/Meesho/BharatMLStack/resource-manager/internal/config"

	etcdadapter "github.com/Meesho/BharatMLStack/resource-manager/internal/adapters/etcd"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/adapters/kubernetes"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/adapters/redisq"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/api"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/application"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/ports"
	"github.com/Meesho/BharatMLStack/resource-manager/pkg/config"
	"github.com/rs/zerolog/log"
)

func BuildHandler() (http.Handler, error) {
	envCfg := config.Instance()

	var shadowStore ports.ShadowStateStore
	var idempotencyStore ports.IdempotencyKeyStore
	var operationStore ports.OperationStore
	var shadowCache ports.ShadowCache

	if envCfg.UseMockAdapters {
		log.Warn().Msg("USE_MOCK_ADAPTERS=true, using in-memory stores")
		shadowStore = etcdadapter.NewMemoryShadowStateStore(map[string][]models.ShadowDeployable{})
		idempotencyStore = etcdadapter.NewMemoryIdempotencyKeyStore()
	} else {
		log.Info().Strs("endpoints", envCfg.EtcdEndpoints).Dur("etcd_timeout", envCfg.EtcdTimeout).Msg("initializing etcd-backed adapters")
		etcdClient, err := etcdadapter.NewClient(etcdadapter.ClientConfig{
			Endpoints: envCfg.EtcdEndpoints,
			Username:  envCfg.EtcdUsername,
			Password:  envCfg.EtcdPassword,
			Timeout:   envCfg.EtcdTimeout,
		})
		if err != nil {
			return nil, err
		}

		shadowStore = etcdadapter.NewEtcdShadowStateStore(etcdClient.Raw())
		idempotencyStore = etcdadapter.NewEtcdIdempotencyKeyStore(etcdClient.Raw())
		operationStore = etcdadapter.NewEtcdOperationStore(etcdClient.Raw())

		configManager := rmconfig.Instance(rmconfig.DefaultVersion)
		if configManager == nil {
			log.Warn().Msg("etcd config manager is not initialized, shadow list will read directly from store")
		} else {
			shadowCache = configManager
		}
	}

	publisher := redisq.NewInMemoryPublisher()
	kubeExecutor := kubernetes.NewMockExecutor()

	shadowService := application.NewShadowService(shadowStore, shadowCache)
	operationService := application.NewOperationService(publisher, kubeExecutor, operationStore)
	handler := api.NewHandler(shadowService, operationService, idempotencyStore)

	mux := http.NewServeMux()
	handler.Register(mux)
	return mux, nil
}
