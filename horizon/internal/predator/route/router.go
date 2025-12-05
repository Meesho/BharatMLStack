package route

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/predator/controller"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
	"github.com/gin-gonic/gin"
)

var initPredatorRouterOnce sync.Once

// Init expects http framework to be initialized before calling this function
func Init() {
	initPredatorRouterOnce.Do(func() {
		api := httpframework.Instance().Group("/api/v1/horizon/predator-config-registry")
		{
			api.GET("/model-params", controller.NewConfigController().FetchModelConfig)
			api.POST("/models/onboard", controller.NewConfigController().OnboardModel)
			api.POST("/models/edit", controller.NewConfigController().EditModel)
			api.POST("/models/promote", controller.NewConfigController().PromoteModel)
			api.POST("/models/scale-up", controller.NewConfigController().ScaleUpModel)
			api.PATCH("/models/delete", controller.NewConfigController().DeleteModel)
			api.POST("/upload-json", controller.NewConfigController().UploadJSONToGCS)
			api.POST("/upload-model-folder", controller.NewConfigController().UploadModelFolder)
		}

		modelDiscoveryGroup := httpframework.Instance().Group("/api/v1/horizon/predator-config-discovery")
		{
			modelDiscoveryGroup.GET("/models", controller.NewConfigController().GetModels)
			modelDiscoveryGroup.GET("/source-models", controller.NewConfigController().GetSourceModels)
		}

		// Feature types API with simpler path
		httpframework.Instance().GET("/api/v1/horizon/predator-config-discovery/feature-types", controller.NewConfigController().GetFeatureTypes)

		modelApprovalGroup := httpframework.Instance().Group("/api/v1/horizon/predator-config-approval")
		{
			modelApprovalGroup.PUT("/process-request", controller.NewConfigController().ProcessRequest)
			modelApprovalGroup.GET("/requests/:group_id", controller.NewConfigController().Validate)
			modelApprovalGroup.GET("/requests", controller.NewConfigController().GetAllPredatorRequests)
		}

		modelTestingGroup := httpframework.Instance().Group("/api/v1/horizon/predator-config-testing")
		{
			modelTestingGroup.POST("/generate-request", controller.NewConfigController().GenerateFunctionalTestRequest)
			modelTestingGroup.POST("/functional-testing/execute-request", controller.NewConfigController().ExecuteFunctionalTestRequest)
			modelTestingGroup.POST("/load-testing/execute-request", controller.NewConfigController().SendLoadTestRequest)
		}
	})
}

func Health(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Application is up!!!"})
}
