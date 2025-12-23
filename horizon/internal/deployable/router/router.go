package router

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/deployable/controller"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
)

var initDeployableRouterOnce sync.Once

// Init expects http framework to be initialized before calling this function
func Init() {
	initDeployableRouterOnce.Do(func() {
		deployableRegistryApi := httpframework.Instance().Group("/api/v1/horizon/deployable-registry/deployables")
		{
			deployableRegistryApi.POST("", controller.NewConfigController().CreateDeployable)
			deployableRegistryApi.PUT("", controller.NewConfigController().UpdateDeployable)
			deployableRegistryApi.POST("/refresh", controller.NewConfigController().RefreshDeployable)
		}

		deployableDiscoveryApi := httpframework.Instance().Group("/api/v1/horizon/deployable-discovery/deployables")
		{
			deployableDiscoveryApi.GET("/metadata", controller.NewConfigController().GetMetaData)
			deployableDiscoveryApi.GET("", controller.NewConfigController().GetDeployablesByService)

		}
		deployableRegistry := httpframework.Instance().Group("/api/v1/horizon/deployable-registry")
		{
			deployableRegistry.PUT("/deployables/tune-thresholds", controller.NewConfigController().TuneThresholds)
		}
	})
}
