package main

import (
	"embed"
	"io"
	_ "net/http/pprof"

	handlerConfig "github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	extOrion "github.com/Meesho/BharatMLStack/inferflow/handlers/external/featurestore"
	extNumerix "github.com/Meesho/BharatMLStack/inferflow/handlers/external/numerix"
	extPredator "github.com/Meesho/BharatMLStack/inferflow/handlers/external/predator"
	extPrism "github.com/Meesho/BharatMLStack/inferflow/handlers/external/prism"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/inferflow"
	"github.com/Meesho/BharatMLStack/inferflow/internal/server"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/config"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/etcd"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/inmemorycache"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
	goCoreConfig "github.com/Meesho/go-core/config"
	"github.com/Meesho/go-core/datatypeconverter/byteorder"
	"github.com/Meesho/go-core/mq/producer"
	"github.com/Meesho/go-core/profiling"
	"github.com/Meesho/price-aggregator-go/pricingfeatureretrieval/client"
	"github.com/spf13/viper"
	_ "go.uber.org/automaxprocs" // by default sets the GOMAXPROCS
)

//go:embed application.yaml
var content embed.FS

var AppConfigs configs.AppConfigs

func main() {
	viper.Set("ENVIRONMENT", "prd")
	viper.Set("DEPLOYABLE_NAME", "inferflow-experiment")
	viper.Set("CONFIG_LOCATION", "/Users/ayushverma/model-proxy-service/configs/inferflow")
	file, err := content.Open("application.yaml")
	if err != nil {
		panic(err)
	}

	goCoreConfig.Init(config.GetInferflowConfigInstance(), io.Reader(file))
	byteorder.Init()
	goCoreConfig.InitGlobalConfig(&AppConfigs)
	producer.Init()
	logger.InitLogger(&AppConfigs)
	profiling.Init()
	etcd.Init(1, &handlerConfig.ModelConfig{}, &AppConfigs)
	err = etcd.Instance().RegisterWatchPathCallback("", inferflow.ReloadModelConfigMapAndRegisterComponents)
	if err != nil {
		logger.Error("Error registering watch path callback for model configs", err)
	}
	metrics.InitMetrics(&AppConfigs)
	extOrion.InitFSHandler(&AppConfigs)
	extPredator.InitPredatorHandler(&AppConfigs)
	extNumerix.InitNumerixHandler(&AppConfigs)
	extPrism.InitPrismHandler(&AppConfigs)
	inmemorycache.InitInMemoryCache()
	client.Init()
	inferflow.InitInferflowHandler(&AppConfigs)
	server.InitServer(&AppConfigs)
}
