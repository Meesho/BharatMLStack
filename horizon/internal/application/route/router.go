package route

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/application/controller"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
)

var initApplicationRouterOnce sync.Once

func Init() {
	initApplicationRouterOnce.Do(func() {
		api := httpframework.Instance().Group("/api")
		{
			v1 := api.Group("/v1/horizon")
			{

				// Application Config Registry/Discovery routes
				registry := v1.Group("/application-registry/applications")
				{
					registry.POST("", controller.NewConfigController().Onboard)
					registry.PUT("/:token", controller.NewConfigController().Edit)
				}

				discovery := v1.Group("/application-discovery/applications")
				{
					discovery.GET("", controller.NewConfigController().GetAll)
				}

			}
		}
	})
}
