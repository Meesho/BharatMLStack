package main

import (
	"strconv"

	authRouter "github.com/Meesho/BharatMLStack/horizon/internal/auth/router"
	"github.com/Meesho/BharatMLStack/horizon/internal/middlewares"
	ofsConfig "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config"
	ofsRouter "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/router"
	"github.com/Meesho/BharatMLStack/horizon/pkg/config"
	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/Meesho/BharatMLStack/horizon/pkg/logger"
	"github.com/Meesho/BharatMLStack/horizon/pkg/metric"
	"github.com/spf13/viper"
)

func main() {
	config.InitEnv()
	infra.InitDBConnectors()
	logger.Init()
	metric.Init()
	httpframework.Init(middlewares.NewMiddleware().GetMiddleWares()...)
	etcd.InitFromAppName(&ofsConfig.FeatureRegistry{}, viper.GetString("ONLINE_FEATURE_STORE_APP_NAME"))
	authRouter.Init()
	ofsRouter.Init()
	httpframework.Instance().Run(":" + strconv.Itoa(viper.GetInt("APP_PORT")))
}
