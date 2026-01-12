package main

import (
	"strconv"

	horizonConfig "github.com/Meesho/BharatMLStack/horizon/internal"
	applicationRouter "github.com/Meesho/BharatMLStack/horizon/internal/application/route"
	authRouter "github.com/Meesho/BharatMLStack/horizon/internal/auth/router"
	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	connectionConfigRouter "github.com/Meesho/BharatMLStack/horizon/internal/connectionconfig/route"
	deployableRouter "github.com/Meesho/BharatMLStack/horizon/internal/deployable/router"
	dnsRouter "github.com/Meesho/BharatMLStack/horizon/internal/dns"
	inferflowConfig "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/etcd"
	inferflowRouter "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/route"
	infrastructureRouter "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/router"
	"github.com/Meesho/BharatMLStack/horizon/internal/middleware"
	numerixConfig "github.com/Meesho/BharatMLStack/horizon/internal/numerix/etcd"
	numerixRouter "github.com/Meesho/BharatMLStack/horizon/internal/numerix/route"
	ofsConfig "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config"
	ofsRouter "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/router"
	predatorRouter "github.com/Meesho/BharatMLStack/horizon/internal/predator/route"
	skyeConfig "github.com/Meesho/BharatMLStack/horizon/internal/skye/etcd"
	skyeRouter "github.com/Meesho/BharatMLStack/horizon/internal/skye/route"
	workflowPkg "github.com/Meesho/BharatMLStack/horizon/internal/workflow"
	workflowEtcd "github.com/Meesho/BharatMLStack/horizon/internal/workflow/etcd"
	workflowHandler "github.com/Meesho/BharatMLStack/horizon/internal/workflow/handler"
	workflowRouter "github.com/Meesho/BharatMLStack/horizon/internal/workflow/router"
	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/Meesho/BharatMLStack/horizon/pkg/logger"
	"github.com/Meesho/BharatMLStack/horizon/pkg/metric"
	"github.com/Meesho/BharatMLStack/horizon/pkg/scheduler"
)

type AppConfig struct {
	Configs        configs.Configs
	DynamicConfigs configs.DynamicConfigs
}

func (cfg *AppConfig) GetStaticConfig() interface{} {
	return &cfg.Configs
}

func (cfg *AppConfig) GetDynamicConfig() interface{} {
	return &cfg.DynamicConfigs
}

var (
	appConfig AppConfig
)

func main() {
	configs.InitConfig(&appConfig)
	infra.InitDBConnectors(appConfig.Configs)
	etcd.InitFromAppName(&ofsConfig.FeatureRegistry{}, appConfig.Configs.OnlineFeatureStoreAppName, appConfig.Configs)
	etcd.InitFromAppName(&numerixConfig.NumerixConfigRegistery{}, appConfig.Configs.NumerixAppName, appConfig.Configs)
	etcd.InitFromAppName(&inferflowConfig.ModelConfigRegistery{}, appConfig.Configs.InferflowAppName, appConfig.Configs)
	etcd.InitFromAppName(&inferflowConfig.HorizonRegistry{}, appConfig.Configs.HorizonAppName, appConfig.Configs)
	etcd.InitFromAppName(&workflowEtcd.WorkflowRegistry{}, workflowPkg.WorkflowAppName, appConfig.Configs)
	etcd.InitFromAppName(&skyeConfig.SkyeConfigRegistry{}, appConfig.Configs.SkyeAppName, appConfig.Configs)
	horizonConfig.InitAll(appConfig.Configs)
	logger.Init(appConfig.Configs)
	metric.Init(appConfig.Configs)
	httpframework.Init(middleware.NewMiddleware().GetMiddleWares()...)
	workflowHandler.InitV1WorkflowHandler()
	deployableRouter.Init(appConfig.Configs)
	inferflowRouter.Init()
	numerixRouter.Init()
	applicationRouter.Init()
	connectionConfigRouter.Init()
	predatorRouter.Init()
	authRouter.Init()
	ofsRouter.Init()
	infrastructureRouter.Init()
	dnsRouter.Init()
	workflowRouter.Init()
	skyeRouter.Init()
	scheduler.Init(appConfig.Configs)
	httpframework.Instance().Run(":" + strconv.Itoa(appConfig.Configs.AppPort))
}
