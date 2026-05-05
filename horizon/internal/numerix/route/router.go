package route

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/numerix/controller"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
)

var initNumerixRouterOnce sync.Once

func Init() {
	initNumerixRouterOnce.Do(func() {
		api := httpframework.Instance().Group("/api")
		{
			v1 := api.Group("/v1/horizon")
			{
				// Numerix Expression routes
				expression := v1.Group("/numerix-expression")
				{
					expression.POST("/generate", controller.NewConfigController().GenerateExpression)
					expression.GET("/:config_id/variables", controller.NewConfigController().GetExpressionVariables)
				}

				// Numerix Config Registry/Discovery routes
				registry := v1.Group("/numerix-config-registry")
				{
					registry.POST("/onboard", controller.NewConfigController().Onboard)
					registry.POST("/promote", controller.NewConfigController().Promote)
					registry.POST("/edit", controller.NewConfigController().Edit)
				}

				discovery := v1.Group("/numerix-config-discovery")
				{
					discovery.GET("/configs", controller.NewConfigController().GetAll)
				}

				// Numerix Config Approval routes
				approval := v1.Group("/numerix-config-approval")
				{
					approval.POST("/review", controller.NewConfigController().ReviewRequest)
					approval.POST("/cancel", controller.NewConfigController().CancelRequest)
					approval.GET("/configs", controller.NewConfigController().GetAllRequests)
				}
				// Numerix Testing
				testing := v1.Group("/numerix-testing")
				{
					functional_testing := testing.Group("/functional-testing")
					{
						functional_testing.POST("/generate-request", controller.NewConfigController().GenerateFuncitonalTestRequest)
						functional_testing.POST("/execute-request", controller.NewConfigController().ExecuteFuncitonalTestRequest)
					}
				}
			}
		}
	})
}
