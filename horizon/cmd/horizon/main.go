package main

import (
	"strconv"

	applicationRouter "github.com/Meesho/BharatMLStack/horizon/internal/application/route"
	authRouter "github.com/Meesho/BharatMLStack/horizon/internal/auth/router"
	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	connectionConfigRouter "github.com/Meesho/BharatMLStack/horizon/internal/connectionconfig/route"
	deployableRouter "github.com/Meesho/BharatMLStack/horizon/internal/deployable/router"
	featureStoreRouter "github.com/Meesho/BharatMLStack/horizon/internal/feature_store/route"
	inferflowConfig "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/etcd"
	inferflowRouter "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/route"
	"github.com/Meesho/BharatMLStack/horizon/internal/middleware"
	numerixConfig "github.com/Meesho/BharatMLStack/horizon/internal/numerix/etcd"
	numerixRouter "github.com/Meesho/BharatMLStack/horizon/internal/numerix/route"
	ofsConfig "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config"
	ofsRouter "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/router"
	predatorRouter "github.com/Meesho/BharatMLStack/horizon/internal/predator/route"
	"github.com/Meesho/BharatMLStack/horizon/pkg/config"
	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/Meesho/BharatMLStack/horizon/pkg/logger"
	"github.com/Meesho/BharatMLStack/horizon/pkg/metric"
	"github.com/Meesho/BharatMLStack/horizon/pkg/scheduler"
	cacConfig "github.com/Meesho/go-core/config"
	pricingclient "github.com/Meesho/price-aggregator-go/pricingfeatureretrieval/client"
	"github.com/spf13/viper"
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
	cacConfig.InitGlobalConfig(&appConfig)
	config.InitEnv()
	infra.InitDBConnectors()
	logger.Init()
	metric.Init()
	httpframework.Init(middleware.NewMiddleware().GetMiddleWares()...)
	etcd.InitFromAppName(&ofsConfig.FeatureRegistry{}, appConfig.Configs.OnlineFeatureStoreAppName, appConfig.Configs)
	etcd.InitFromAppName(&numerixConfig.NumerixConfigRegistery{}, appConfig.Configs.NumerixAppName, appConfig.Configs)
	etcd.InitMPEtcdFromRegistry(&inferflowConfig.ModelConfigRegistery{}, appConfig.Configs)
	etcd.InitFromAppName(&inferflowConfig.HorizonRegistry{}, appConfig.Configs.HorizonAppName, appConfig.Configs)
	deployableRouter.Init()
	inferflowRouter.Init()
	numerixRouter.Init()
	applicationRouter.Init()
	connectionConfigRouter.Init()
	predatorRouter.Init()
	authRouter.Init()
	ofsRouter.Init()
	featureStoreRouter.Init(appConfig.Configs)
	scheduler.Init(appConfig.Configs)
	pricingclient.Init()
	httpframework.Instance().Run(":" + strconv.Itoa(viper.GetInt("APP_PORT")))
}
