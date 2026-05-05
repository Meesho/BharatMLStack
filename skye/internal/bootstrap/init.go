package bootstrap

import (
	"github.com/Meesho/BharatMLStack/skye/internal/admin/handler/qdrant"
	"github.com/Meesho/BharatMLStack/skye/internal/admin/handler/workflow"
	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/Meesho/BharatMLStack/skye/internal/consumers/listener/realtime"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/aggregator"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/distributedcache"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/embedding"
	"github.com/Meesho/BharatMLStack/skye/internal/server/middlewares"
)

func Init() {
	config.InitConfig(structs.GetAppConfig())
	config.Init()
	embedding.Init()
	aggregator.Init()
	qdrant.Init()
}
func InitAdmin() {
	Init()
	workflow.Init()
}
func InitConsumers() {
	Init()
	realtime.Init()
}
func InitServing() {
	Init()
	middlewares.Init()
	distributedcache.Init()
}
