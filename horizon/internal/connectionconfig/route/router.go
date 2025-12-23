package route

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/connectionconfig/controller"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
)

var initConnectionConfigRouterOnce sync.Once

func Init() {
	initConnectionConfigRouterOnce.Do(func() {
		api := httpframework.Instance().Group("/api")
		{
			v1 := api.Group("/v1/horizon")
			{

				// Application Config Registry/Discovery routes
				registry := v1.Group("/service-connection-registry/connections")
				{
					registry.POST("", controller.NewConfigController().Onboard)
					registry.PUT("/:id", controller.NewConfigController().Edit)
				}

				discovery := v1.Group("/service-connection-discovery/connections")
				{
					discovery.GET("", controller.NewConfigController().GetAll)
				}

			}
		}
	})
}
