package router

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/controller"
	inframiddleware "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/middleware"
	"github.com/Meesho/BharatMLStack/horizon/pkg/httpframework"
)

var (
	initInfrastructureRouterOnce sync.Once
)

// Init expects http framework to be initialized before calling this function
func Init() {
	initInfrastructureRouterOnce.Do(func() {
		infrastructureAPI := httpframework.Instance().Group("/api/v1/horizon/infrastructure")
		// Apply workingEnv middleware to all infrastructure routes
		// This matches RingMaster's pattern where middleware validates and injects workingEnv
		infrastructureAPI.Use(inframiddleware.WorkingEnvMiddleware())
		{
			// Application resource operations
			infrastructureAPI.GET("/applications/:appName/hpa", controller.NewController().GetHPAConfig)
			infrastructureAPI.GET("/applications/resources", controller.NewController().GetResourceDetail)
			infrastructureAPI.POST("/applications/:appName/restart", controller.NewController().RestartDeployment)
			infrastructureAPI.PUT("/applications/:appName/hpa/cpu", controller.NewController().UpdateCPUThreshold)
			infrastructureAPI.PUT("/applications/:appName/hpa/gpu", controller.NewController().UpdateGPUThreshold)
			infrastructureAPI.PUT("/applications/:appName/shared-memory", controller.NewController().UpdateSharedMemory)
			infrastructureAPI.PUT("/applications/:appName/pod-annotations", controller.NewController().UpdatePodAnnotations)
			infrastructureAPI.PUT("/applications/:appName/autoscaling-triggers", controller.NewController().UpdateAutoscalingTriggers)
		}
	})
}
