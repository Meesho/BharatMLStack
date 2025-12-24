package main

import (
	"strconv"

	"github.com/gin-contrib/cors"

	horizonConfig "github.com/Meesho/BharatMLStack/horizon/internal"
	"github.com/Meesho/BharatMLStack/horizon/internal/configs"

	// inferflowConfig "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/etcd" // TODO: Uncomment for production

	deployableRouter "github.com/Meesho/BharatMLStack/horizon/internal/deployable/router"
	dnsRouter "github.com/Meesho/BharatMLStack/horizon/internal/dns"
	infrastructureRouter "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/router"
	workflowRouter "github.com/Meesho/BharatMLStack/horizon/internal/workflow/router"

	// "github.com/Meesho/BharatMLStack/horizon/internal/middleware" // TODO: Uncomment for production
	// numerixConfig "github.com/Meesho/BharatMLStack/horizon/internal/numerix/etcd" // TODO: Uncomment for production

	// ofsConfig "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config" // TODO: Uncomment for production
	workflowPkg "github.com/Meesho/BharatMLStack/horizon/internal/workflow"
	workflowEtcd "github.com/Meesho/BharatMLStack/horizon/internal/workflow/etcd"
	workflowHandler "github.com/Meesho/BharatMLStack/horizon/internal/workflow/handler"

	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/Meesho/BharatMLStack/horizon/pkg/logger"
	"github.com/Meesho/BharatMLStack/horizon/pkg/metric"
	"github.com/rs/zerolog/log"
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
	horizonConfig.InitAll(appConfig.Configs)

	// Initialize logger first (needed for logging)
	logger.Init(appConfig.Configs)

	// Database initialization (MySQL credentials in env.example)
	infra.InitDBConnectors(appConfig.Configs)

	metric.Init(appConfig.Configs)

	// TODO: Uncomment for production - Auth middleware required for authentication
	// httpframework.Init(middleware.NewMiddleware().GetMiddleWares()...)

	// Local testing: Only CORS middleware, no authentication
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"*"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	corsConfig.AllowCredentials = true
	httpframework.Init(cors.New(corsConfig))

	// TODO: Uncomment for production - etcd initialization required for config management
	// etcd.InitFromAppName(&ofsConfig.FeatureRegistry{}, appConfig.Configs.OnlineFeatureStoreAppName, appConfig.Configs)
	// etcd.InitFromAppName(&numerixConfig.NumerixConfigRegistery{}, appConfig.Configs.NumerixAppName, appConfig.Configs)
	// etcd.InitFromAppName(&inferflowConfig.ModelConfigRegistery{}, appConfig.Configs.InferflowAppName, appConfig.Configs)
	// etcd.InitFromAppName(&inferflowConfig.HorizonRegistry{}, appConfig.Configs.HorizonAppName, appConfig.Configs)

	// Workflow etcd initialization (same pattern as numerix)
	etcd.InitFromAppName(&workflowEtcd.WorkflowRegistry{}, workflowPkg.WorkflowAppName, appConfig.Configs)

	// Initialize workflow handler (starts worker pool) - same pattern as numerix handler initialization
	workflowHandler.InitV1WorkflowHandler()

	// Deployable router - enabled now that MySQL credentials are available
	deployableRouter.Init(appConfig.Configs)

	// TODO: Uncomment for production - These routers require database connection
	// inferflowRouter.Init()        // Requires DB
	// numerixRouter.Init()          // Requires DB
	// applicationRouter.Init()      // May require DB
	// connectionConfigRouter.Init() // Requires DB
	// predatorRouter.Init()         // Requires DB
	// authRouter.Init()             // Requires DB (for login/register)
	// ofsRouter.Init()              // May require DB

	// Infrastructure router - works without DB (uses ArgoCD/GitHub directly)
	infrastructureRouter.Init()

	// DNS router - only available when built with -tags meesho
	// For open-source builds, this is a no-op
	dnsRouter.Init()

	// Workflow router - works without DB (uses etcd for state management)
	workflowRouter.Init()

	// TODO: Uncomment for production
	// scheduler.Init(appConfig.Configs)

	// Use default port if not set (for local testing)
	port := appConfig.Configs.AppPort
	if port == 0 {
		port = 8082
		log.Warn().Int("port", port).Msg("App port not set, defaulting to 8082")
	}
	httpframework.Instance().Run(":" + strconv.Itoa(port))
}
