package main

import (
	"os"

	"github.com/Meesho/interaction-store/internal/config"
	// "github.com/Meesho/interaction-store/internal/consumer"
	conf "github.com/Meesho/go-core/config"
	"github.com/Meesho/go-core/httpframework"
	"github.com/Meesho/go-core/logger"
	"github.com/Meesho/go-core/metric"
	"github.com/Meesho/interaction-store/internal/consumer/click"
	"github.com/Meesho/interaction-store/internal/controller/health"
	"github.com/Meesho/interaction-store/internal/controller/middleware"
	"github.com/Meesho/interaction-store/internal/data/scylla"

	// mqConsumer "github.com/Meesho/go-core/mq/consumer"
	"github.com/Meesho/go-core/profiling"
)

func main() {
	os.Setenv("ENVIRONMENT", "stg")
	os.Setenv("DEPLOYABLE_NAME", "interaction-store-timeseries-consumer")
	os.Setenv("CONFIG_LOCATION", "/Users/shubamkaushik/Desktop/interaction-store/configs/consumers")
	os.Setenv("MQ_API_AUTH_TOKEN", "test-token")
	appConfig := config.GetAppConfig()
	conf.InitGlobalConfig(appConfig)
	scylla.Init()
	click.Init()
	// order.Init()
	logger.Init()
	metric.Init()
	profiling.Init()
	// mqConsumer.Init()
	// var stringType string
	// mqConsumer.ConsumeBatchWithManualAck(appConfig.Configs.ClickConsumerMqId, consumer.ProcessClickEvents, stringType, stringType)
	httpframework.Init()
	health.Init()
	middleware.InitServer(appConfig.Configs.Port)
}
