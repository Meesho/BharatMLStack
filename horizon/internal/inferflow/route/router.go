package route

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/inferflow/controller"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
)

var initInferflowRouterOnce sync.Once

func Init() {
	initInferflowRouterOnce.Do(func() {
		api := httpframework.Instance().Group("/api")
		{
			v1 := api.Group("/v1/horizon")
			{
				register := v1.Group("/inferflow-config-registry")
				{
					register.POST("/onboard", controller.NewConfigController().Onboard)
					register.POST("/promote", controller.NewConfigController().Promote)
					register.POST("/edit", controller.NewConfigController().Edit)
					register.POST("/clone", controller.NewConfigController().Clone)
					register.POST("/scale-up", controller.NewConfigController().ScaleUp)
					register.GET("/logging-ttl", controller.NewConfigController().GetLoggingTTL)
					register.PATCH("/delete", controller.NewConfigController().Delete)
					register.GET("/latestRequest/:config_id", controller.NewConfigController().GetLatestRequest)
					register.GET("/get_feature_schema", controller.NewConfigController().GetFeatureSchema)
				}

				discovery := v1.Group("/inferflow-config-discovery")
				{
					discovery.GET("/configs", controller.NewConfigController().GetAll)
				}

				approval := v1.Group("/inferflow-config-approval")
				{
					approval.POST("/review", controller.NewConfigController().Review)
					approval.POST("/cancel", controller.NewConfigController().Cancel)
					approval.GET("/configs", controller.NewConfigController().GetAllRequests)
					approval.GET("/validate/:request_id", controller.NewConfigController().ValidateRequest)
				}
				testing := v1.Group("/inferflow-config-testing")
				{
					functional_testing := testing.Group("/functional-test")
					{
						functional_testing.POST("/generate-request", controller.NewConfigController().GenerateFunctionalTestRequest)
						functional_testing.POST("/execute-request", controller.NewConfigController().ExecuteFuncitonalTestRequest)
					}
				}
			}
		}
	})
}
