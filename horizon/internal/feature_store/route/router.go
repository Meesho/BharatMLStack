package route

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/feature_store/controller"
	feature_registry "github.com/Meesho/BharatMLStack/horizon/internal/feature_store/handler"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
)

var initFeatureStoreRouterOnce sync.Once

func Init(config configs.Configs) {
	initFeatureStoreRouterOnce.Do(func() {
		api := httpframework.Instance().Group("/api")
		{
			v1 := api.Group("/1.0")

			fsConfigHandler := feature_registry.NewFsConfigHandlerGenerator()
			fsConfigHandler.InitFsConfig(config)
			fsConfig := v1.Group("/fs-config")
			{
				fsConfig.GET("/fetch", controller.NewFsConfigControllerGenerator(fsConfigHandler).GetFsConfigs)
			}
		}
	})
}
