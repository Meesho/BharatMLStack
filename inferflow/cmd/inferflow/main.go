//go:build !meesho

package main

import (
	"fmt"

	handlerConfig "github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	extOrion "github.com/Meesho/BharatMLStack/inferflow/handlers/external/featurestore"
	extNumerix "github.com/Meesho/BharatMLStack/inferflow/handlers/external/numerix"
	extPredator "github.com/Meesho/BharatMLStack/inferflow/handlers/external/predator"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/inferflow"
	"github.com/Meesho/BharatMLStack/inferflow/internal/server"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/datatypeconverter/byteorder"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/etcd"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/inmemorycache"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
	"github.com/spf13/viper"
)

var AppConfigs configs.AppConfigs

func main() {
	viper.AutomaticEnv()
	viper.SetConfigName("application")                                              // file name without .env
	viper.SetConfigType("env")                                                      // file type
	viper.AddConfigPath("/Users/ayushverma/BharatMLStack/inferflow/cmd/inferflow/") // directory

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("Error reading config file")
	}
	configs.InitConfig(&AppConfigs)
	byteorder.Init()
	logger.InitLogger(&AppConfigs)
	etcd.Init(1, &handlerConfig.ModelConfig{}, &AppConfigs)
	err = etcd.Instance().RegisterWatchPathCallback("", inferflow.ReloadModelConfigMapAndRegisterComponents)
	if err != nil {
		logger.Error("Error registering watch path callback for model configs", err)
	}
	metrics.InitMetrics(&AppConfigs)
	extOrion.InitFSHandler(&AppConfigs)
	extPredator.InitPredatorHandler(&AppConfigs)
	extNumerix.InitNumerixHandler(&AppConfigs)
	inmemorycache.InitInMemoryCache()
	inferflow.InitInferflowHandler(&AppConfigs)
	server.InitServer(&AppConfigs)
}
